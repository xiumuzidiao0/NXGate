package proxy

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"aimili-vpngate-go/pkg/stats"
	"aimili-vpngate-go/pkg/tunnel"
)

const (
	socks5Version = 0x05

	// Auth methods
	authMethodNoAuth   = 0x00
	authMethodUserPass = 0x02
	authMethodNoAccept = 0xff

	// Commands
	cmdConnect      = 0x01
	cmdBind         = 0x02
	cmdUdpAssociate = 0x03

	// Address types
	atypIPv4   = 0x01
	atypDomain = 0x03
	atypIPv6   = 0x04

	// Reply codes
	repSuccess           = 0x00
	repGeneralFailure    = 0x01
	repCommandNotSupport = 0x07
	repAtypNotSupport    = 0x08
)

func handleSocks5(client net.Conn, auth *Authenticator, devName string, tun *tunnel.Tunnel) error {
	// 1. Negotiation
	// Client sends: VER (1 byte) | NMETHODS (1 byte) | METHODS (1-255 bytes)
	verBuf := make([]byte, 1)
	if _, err := io.ReadFull(client, verBuf); err != nil {
		return fmt.Errorf("read socks5 version failed: %w", err)
	}
	if verBuf[0] != socks5Version {
		return fmt.Errorf("invalid socks5 version: %d", verBuf[0])
	}

	header := make([]byte, 1) // NMETHODS
	if _, err := io.ReadFull(client, header); err != nil {
		return fmt.Errorf("read nmethods failed: %w", err)
	}
	nMethods := int(header[0])
	if nMethods <= 0 {
		return errors.New("socks5 nmethods must be > 0")
	}

	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(client, methods); err != nil {
		return fmt.Errorf("read methods failed: %w", err)
	}

	methodMap := make(map[byte]bool)
	for _, m := range methods {
		methodMap[m] = true
	}

	if auth.IsEnabled() {
		if !methodMap[authMethodUserPass] {
			_, _ = client.Write([]byte{socks5Version, authMethodNoAccept})
			return errors.New("client does not support user/password auth")
		}
		if _, err := client.Write([]byte{socks5Version, authMethodUserPass}); err != nil {
			return err
		}

		// RFC 1929: VER (0x01) | ULEN | UNAME | PLEN | PASSWD
		authVer := make([]byte, 1)
		if _, err := io.ReadFull(client, authVer); err != nil || authVer[0] != 0x01 {
			_, _ = client.Write([]byte{0x01, 0x01})
			return errors.New("invalid socks5 subnegotiation auth version")
		}

		uLenBuf := make([]byte, 1)
		if _, err := io.ReadFull(client, uLenBuf); err != nil {
			return err
		}
		uLen := int(uLenBuf[0])
		username := make([]byte, uLen)
		if _, err := io.ReadFull(client, username); err != nil {
			return err
		}

		pLenBuf := make([]byte, 1)
		if _, err := io.ReadFull(client, pLenBuf); err != nil {
			return err
		}
		pLen := int(pLenBuf[0])
		password := make([]byte, pLen)
		if _, err := io.ReadFull(client, password); err != nil {
			return err
		}

		if !auth.Verify(string(username), string(password)) {
			_, _ = client.Write([]byte{0x01, 0x01}) // Status != 0 -> auth failure
			return errors.New("socks5 user/password authentication failed")
		}
		if _, err := client.Write([]byte{0x01, 0x00}); err != nil { // Status 0 -> success
			return err
		}
	} else {
		if !methodMap[authMethodNoAuth] {
			_, _ = client.Write([]byte{socks5Version, authMethodNoAccept})
			return errors.New("client does not support no-auth")
		}
		if _, err := client.Write([]byte{socks5Version, authMethodNoAuth}); err != nil {
			return err
		}
	}

	// 2. Request details: VER | CMD | RSV | ATYP | DST.ADDR | DST.PORT
	reqHeader := make([]byte, 4)
	if _, err := io.ReadFull(client, reqHeader); err != nil {
		return fmt.Errorf("read socks5 request header failed: %w", err)
	}

	if reqHeader[0] != socks5Version {
		return fmt.Errorf("invalid socks5 version in request: %d", reqHeader[0])
	}

	cmd := reqHeader[1]
	switch cmd {
	case cmdConnect:
		targetHost, targetPort, err := readSocks5Address(client, reqHeader[3])
		if err != nil {
			_, _ = client.Write([]byte{socks5Version, repAtypNotSupport, 0x00, atypIPv4, 0, 0, 0, 0, 0, 0})
			return err
		}
		targetAddr := net.JoinHostPort(targetHost, strconv.Itoa(int(targetPort)))

		// 3. Connect to upstream via selected tunnel devName
		upstream, err := dialUpstream(targetAddr, devName, 10*time.Second)
		if err != nil {
			if tun != nil {
				tun.RecordFailure()
			}
			_, _ = client.Write([]byte{socks5Version, repGeneralFailure, 0x00, atypIPv4, 0, 0, 0, 0, 0, 0})
			return fmt.Errorf("dial upstream %s failed: %w", targetAddr, err)
		}
		if tun != nil {
			tun.RecordSuccess()
		}

		// 4. Send success reply
		// Reply: VER | REP | RSV | ATYP | BND.ADDR | BND.PORT
		reply := []byte{socks5Version, repSuccess, 0x00, atypIPv4, 0, 0, 0, 0, 0, 0}
		if _, err := client.Write(reply); err != nil {
			_ = upstream.Close()
			return err
		}

		// 5. Bidirectional forward
		clearDeadline(client)
		relay(client, upstream)
		return nil

	case cmdUdpAssociate:
		return handleSocks5UDPAssociate(client, reqHeader[3], devName, tun)

	default:
		_, _ = client.Write([]byte{socks5Version, repCommandNotSupport, 0x00, atypIPv4, 0, 0, 0, 0, 0, 0})
		return fmt.Errorf("unsupported socks5 command: %d", cmd)
	}
}

func readSocks5Address(r io.Reader, atyp byte) (string, uint16, error) {
	var targetHost string
	switch atyp {
	case atypIPv4:
		ipv4 := make([]byte, 4)
		if _, err := io.ReadFull(r, ipv4); err != nil {
			return "", 0, err
		}
		targetHost = net.IP(ipv4).String()
	case atypDomain:
		dLenBuf := make([]byte, 1)
		if _, err := io.ReadFull(r, dLenBuf); err != nil {
			return "", 0, err
		}
		domain := make([]byte, int(dLenBuf[0]))
		if _, err := io.ReadFull(r, domain); err != nil {
			return "", 0, err
		}
		targetHost = string(domain)
	case atypIPv6:
		ipv6 := make([]byte, 16)
		if _, err := io.ReadFull(r, ipv6); err != nil {
			return "", 0, err
		}
		targetHost = net.IP(ipv6).String()
	default:
		return "", 0, fmt.Errorf("unsupported socks5 atyp: %d", atyp)
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, portBuf); err != nil {
		return "", 0, err
	}
	targetPort := binary.BigEndian.Uint16(portBuf)
	return targetHost, targetPort, nil
}

func handleSocks5UDPAssociate(client net.Conn, atyp byte, devName string, tun *tunnel.Tunnel) error {
	// 1. Read advertised client source address/port (usually 0.0.0.0:0)
	_, _, err := readSocks5Address(client, atyp)
	if err != nil {
		_, _ = client.Write([]byte{socks5Version, repGeneralFailure, 0x00, atypIPv4, 0, 0, 0, 0, 0, 0})
		return fmt.Errorf("read udp associate address failed: %w", err)
	}

	// 2. Bind local UDP relay listener on the same IP network family as client TCP connection
	var bindIP net.IP
	if tcpAddr, ok := client.LocalAddr().(*net.TCPAddr); ok && tcpAddr != nil {
		bindIP = tcpAddr.IP
	}

	udpNetwork := "udp4"
	isIPv6 := false
	if bindIP != nil && bindIP.To4() == nil && bindIP.To16() != nil {
		udpNetwork = "udp6"
		isIPv6 = true
	}
	if bindIP == nil || bindIP.IsUnspecified() {
		bindIP = net.ParseIP("127.0.0.1")
		udpNetwork = "udp4"
		isIPv6 = false
	}

	relayConn, err := net.ListenUDP(udpNetwork, &net.UDPAddr{IP: bindIP, Port: 0})
	if err != nil {
		// Fallback to IPv4 loopback
		bindIP = net.ParseIP("127.0.0.1")
		isIPv6 = false
		relayConn, err = net.ListenUDP("udp4", &net.UDPAddr{IP: bindIP, Port: 0})
		if err != nil {
			_, _ = client.Write([]byte{socks5Version, repGeneralFailure, 0x00, atypIPv4, 0, 0, 0, 0, 0, 0})
			return fmt.Errorf("listen udp relay failed: %w", err)
		}
	}
	defer relayConn.Close()

	relayPort := relayConn.LocalAddr().(*net.UDPAddr).Port

	// 3. Send SOCKS5 reply with BND.ADDR and BND.PORT over TCP
	var reply []byte
	if isIPv6 {
		bndIP6 := bindIP.To16()
		reply = make([]byte, 22)
		reply[0] = socks5Version
		reply[1] = repSuccess
		reply[2] = 0x00
		reply[3] = atypIPv6
		copy(reply[4:20], bndIP6)
		binary.BigEndian.PutUint16(reply[20:22], uint16(relayPort))
	} else {
		bndIP4 := bindIP.To4()
		if bndIP4 == nil {
			bndIP4 = net.ParseIP("127.0.0.1").To4()
		}
		reply = make([]byte, 10)
		reply[0] = socks5Version
		reply[1] = repSuccess
		reply[2] = 0x00
		reply[3] = atypIPv4
		copy(reply[4:8], bndIP4)
		binary.BigEndian.PutUint16(reply[8:10], uint16(relayPort))
	}
	if _, err := client.Write(reply); err != nil {
		return err
	}

	// 4. Create upstream UDP socket bound to target tunnel devName
	upstreamConn, err := createBoundUDPSocket(devName)
	if err != nil {
		stats.LogWarn("Proxy", "UDP Associate 创建物理网卡套接字失败: %v", err)
		return fmt.Errorf("create bound udp socket failed: %w", err)
	}
	defer upstreamConn.Close()

	clearDeadline(client)

	// 5. Manage TCP lifecycle: per RFC 1928, UDP association terminates when TCP drops
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		defer cancel()
		buf := make([]byte, 1)
		for {
			_, err := client.Read(buf)
			if err != nil {
				break
			}
		}
		_ = relayConn.Close()
		_ = upstreamConn.Close()
	}()

	var clientUDPMu sync.RWMutex
	var clientUDPAddr *net.UDPAddr

	// Relay Upstream -> Client
	go func() {
		respBuf := make([]byte, 65535)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			n, remoteAddr, err := upstreamConn.ReadFrom(respBuf)
			if err != nil {
				return
			}
			if n <= 0 {
				continue
			}

			clientUDPMu.RLock()
			destClient := clientUDPAddr
			clientUDPMu.RUnlock()
			if destClient == nil {
				continue
			}

			if rUDP, ok := remoteAddr.(*net.UDPAddr); ok {
				packed := packSocks5UDPPacket(rUDP, respBuf[:n])
				if len(packed) > 0 {
					_, _ = relayConn.WriteTo(packed, destClient)
					stats.GetTrafficTracker().AddDownload(uint64(n))
					if tun != nil {
						tun.RecordSuccess()
					}
				}
			}
		}
	}()

	// Relay Client -> Upstream
	reqBuf := make([]byte, 65535)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		n, fromAddr, err := relayConn.ReadFrom(reqBuf)
		if err != nil {
			return nil
		}
		if n < 4 {
			continue
		}

		if reqBuf[0] != 0x00 || reqBuf[1] != 0x00 {
			continue // Invalid RSV
		}
		if reqBuf[2] != 0x00 {
			continue // Fragment not supported
		}

		if fromUDP, ok := fromAddr.(*net.UDPAddr); ok {
			clientUDPMu.Lock()
			clientUDPAddr = fromUDP
			clientUDPMu.Unlock()
		}

		targetUDPAddr, payload, err := parseSocks5UDPPacket(ctx, reqBuf[:n], devName)
		if err != nil || targetUDPAddr == nil || len(payload) == 0 {
			continue
		}

		_, err = upstreamConn.WriteTo(payload, targetUDPAddr)
		if err != nil {
			if tun != nil {
				tun.RecordFailure()
			}
			continue
		}

		stats.GetTrafficTracker().AddUpload(uint64(len(payload)))
	}
}

func packSocks5UDPPacket(addr *net.UDPAddr, data []byte) []byte {
	if addr == nil {
		return nil
	}
	ip4 := addr.IP.To4()
	if ip4 != nil {
		// RSV(2) | FRAG(1) | ATYP(1) | IP(4) | PORT(2) | DATA
		header := make([]byte, 10+len(data))
		header[0] = 0x00
		header[1] = 0x00
		header[2] = 0x00 // FRAG
		header[3] = atypIPv4
		copy(header[4:8], ip4)
		binary.BigEndian.PutUint16(header[8:10], uint16(addr.Port))
		copy(header[10:], data)
		return header
	}

	ip6 := addr.IP.To16()
	if ip6 != nil {
		// RSV(2) | FRAG(1) | ATYP(1) | IP(16) | PORT(2) | DATA
		header := make([]byte, 22+len(data))
		header[0] = 0x00
		header[1] = 0x00
		header[2] = 0x00
		header[3] = atypIPv6
		copy(header[4:20], ip6)
		binary.BigEndian.PutUint16(header[20:22], uint16(addr.Port))
		copy(header[22:], data)
		return header
	}

	return nil
}

func parseSocks5UDPPacket(ctx context.Context, packet []byte, devName string) (*net.UDPAddr, []byte, error) {
	if len(packet) < 10 {
		return nil, nil, errors.New("packet too short")
	}

	atyp := packet[3]
	offset := 4
	var host string

	switch atyp {
	case atypIPv4:
		if len(packet) < offset+4+2 {
			return nil, nil, errors.New("invalid ipv4 packet length")
		}
		host = net.IP(packet[offset : offset+4]).String()
		offset += 4
	case atypDomain:
		dLen := int(packet[offset])
		offset++
		if len(packet) < offset+dLen+2 {
			return nil, nil, errors.New("invalid domain packet length")
		}
		host = string(packet[offset : offset+dLen])
		offset += dLen
	case atypIPv6:
		if len(packet) < offset+16+2 {
			return nil, nil, errors.New("invalid ipv6 packet length")
		}
		host = net.IP(packet[offset : offset+16]).String()
		offset += 16
	default:
		return nil, nil, fmt.Errorf("unsupported atyp in udp packet: %d", atyp)
	}

	port := int(binary.BigEndian.Uint16(packet[offset : offset+2]))
	offset += 2

	payload := packet[offset:]

	destAddr, err := resolveUDPAddrThroughTunnel(ctx, host, port, devName)
	if err != nil {
		return nil, nil, err
	}

	return destAddr, payload, nil
}
