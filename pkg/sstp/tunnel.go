package sstp

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"aimili-vpngate-go/pkg/dns"
	"aimili-vpngate-go/pkg/stats"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

const (
	maxCoalescePackets = 64
	maxCoalesceBytes   = 24 * 1024
)

type Tunnel struct {
	host       string
	port       int
	username   string
	password   string
	tlsConn    *tls.Conn
	assignedIP net.IP

	mu           sync.RWMutex
	netStack     *stack.Stack
	linkEP       *channel.Endpoint
	txNotifyCh   chan struct{}
	ctrlTxCh     chan []byte
	readBr       *bufio.Reader
	lastRecv     time.Time
	closeReason  string
	closed       bool
	closeOnce    sync.Once
	closeCh      chan struct{}
	dnsResolver  *dns.DualEngineResolver
	dnsTransport *http.Transport
}

func ResolveCandidatePorts(targetPort int, proto string) []int {
	return ResolveCandidatePortsWithPreferred(targetPort, 0)
}

func ResolveCandidatePortsWithPreferred(targetPort int, preferredPort int) []int {
	var candidates []int
	seen := make(map[int]bool)

	add := func(p int) {
		if p > 0 && p <= 65535 && !seen[p] {
			candidates = append(candidates, p)
			seen[p] = true
		}
	}

	// 0. Highest priority: pre-warmed / explicitly verified SSTP port
	if preferredPort > 0 {
		add(preferredPort)
	}

	// 1. If targetPort is valid, prioritize it!
	if targetPort > 0 {
		add(targetPort)
	}

	// 2. Standard SoftEther / SSTP listener ports
	add(443)
	add(995)
	add(5555)

	return candidates
}

// sstpTLSConfig returns the TLS configuration required by the MS-SSTP client.
//
// SSTP servers present certificates issued for their own hostname (for VPNGate
// relays this is "opengw.net") while clients dial literal IP addresses, so chain
// and hostname verification can never succeed. Per [MS-SSTP] the TLS layer is a
// transport wrapper only: session authenticity is established by the PPP layer
// (MS-CHAPv2 / EAP) carried inside the tunnel, so the server certificate is not
// the authentication anchor. Certificate verification is therefore intentionally
// disabled, matching every mainstream SSTP client (sstpc, SoftEther, Windows
// RRAS). Regression guard: enabling it silently breaks every Zero-TUN tunnel.
func sstpTLSConfig(serverName string) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: serverName,
		//nolint:gosec // G402: see sstpTLSConfig doc comment above.
		InsecureSkipVerify: true,
	}
}

// FastProbeSSTPPort tests whether a given host:port accepts SSTP_DUPLEX_POST over TLS.
func FastProbeSSTPPort(ctx context.Context, host string, port int, timeout time.Duration) bool {
	if port <= 0 || port > 65535 {
		return false
	}
	if timeout <= 0 {
		timeout = 2500 * time.Millisecond
	}

	dialer := &net.Dialer{Timeout: timeout}
	rawConn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return false
	}
	defer rawConn.Close()

	_ = rawConn.SetDeadline(time.Now().Add(timeout))

	tlsConfig := sstpTLSConfig(host)
	tlsConn := tls.Client(rawConn, tlsConfig)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		return false
	}
	defer tlsConn.Close()

	req := fmt.Sprintf("SSTP_DUPLEX_POST /sbox/sstp HTTP/1.1\r\nContent-Length: 18446744073709551615\r\nHost: %s\r\n\r\n", host)
	if _, err := tlsConn.Write([]byte(req)); err != nil {
		return false
	}

	reader := bufio.NewReader(tlsConn)
	respLine, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	if strings.Contains(respLine, "200") {
		return true
	}
	return false
}

// DetectSSTPPort concurrently probes candidate ports and returns the first verified SSTP port, or 0.
func DetectSSTPPort(ctx context.Context, host string, candidatePorts []int, timeout time.Duration) int {
	if len(candidatePorts) == 0 {
		return 0
	}
	probeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	resultCh := make(chan int, len(candidatePorts))
	var wg sync.WaitGroup

	for _, p := range candidatePorts {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			if FastProbeSSTPPort(probeCtx, host, port, timeout) {
				select {
				case resultCh <- port:
					cancel()
				default:
				}
			}
		}(p)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	select {
	case <-ctx.Done():
		select {
		case port := <-resultCh:
			return port
		default:
			return 0
		}
	case port, ok := <-resultCh:
		if ok {
			return port
		}
		return 0
	}
}

func ConnectNodeWithPreferred(ctx context.Context, host string, preferredPort int, targetPort int, proto string, user, pass string) (*Tunnel, error) {
	ports := ResolveCandidatePortsWithPreferred(targetPort, preferredPort)
	attempts := make([]string, 0, len(ports))
	var firstErr error

	for i, p := range ports {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// The node's own listener (preferred / target port) is the authoritative
		// SSTP endpoint. The generic SoftEther fallback ports (443/995/5555) are
		// almost never open on VPNGate relays, so give them a tighter budget to
		// keep failover fast instead of stalling the scheduler for ~30s.
		budget := 7 * time.Second
		if i > 0 {
			budget = 4 * time.Second
		}

		connCtx, cancel := context.WithTimeout(ctx, budget)
		tun, err := Connect(connCtx, host, p, user, pass)
		cancel()
		if err == nil {
			return tun, nil
		}
		if firstErr == nil {
			firstErr = err
		}
		attempts = append(attempts, fmt.Sprintf("%d: %v", p, err))
	}

	if firstErr != nil {
		// Surface every port's failure, not just the last one: the authoritative
		// port's error is the meaningful one and must not be masked by a timeout
		// on an unrelated fallback port.
		return nil, fmt.Errorf("connect to %s failed across ports %v (%s): %w",
			host, ports, strings.Join(attempts, "; "), firstErr)
	}
	return nil, ErrHandshakeFailed
}

func Connect(ctx context.Context, host string, port int, user, pass string) (*Tunnel, error) {
	if port <= 0 {
		port = 443
	}
	if user == "" {
		user = "vpn"
	}
	if pass == "" {
		pass = "vpn"
	}

	dialer := &net.Dialer{Timeout: DefaultConnectTimeout, KeepAlive: 15 * time.Second}
	rawConn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return nil, fmt.Errorf("tcp dial failed: %w", err)
	}
	if tcpConn, ok := rawConn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}

	tlsConfig := sstpTLSConfig(host)
	tlsConn := tls.Client(rawConn, tlsConfig)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = rawConn.Close()
		return nil, fmt.Errorf("tls handshake failed: %w", err)
	}

	t := &Tunnel{
		host:     host,
		port:     port,
		username: user,
		password: pass,
		tlsConn:  tlsConn,
		closeCh:  make(chan struct{}),
	}

	if err := t.performHandshake(ctx); err != nil {
		_ = t.Close()
		return nil, err
	}
	if err := t.initNetstack(); err != nil {
		_ = t.Close()
		return nil, err
	}
	// The PPP negotiation that just completed proves the peer was alive, so the
	// liveness clock starts here rather than being left zero (which would disable
	// the staleness bound below).
	t.markActivity()

	go t.writeLoop()
	go t.readLoop()
	go t.keepAliveLoop()

	stats.LogInfo("SSTP", "用户态免 TUN 隧道已成功建立于 %s:%d (gVisor Netstack MTU %d), 分配虚拟 IP: %s", host, port, DefaultTunnelMTU, t.assignedIP)
	return t, nil
}

func (t *Tunnel) initNetstack() error {
	s, ep, notifyCh, err := newNetstack(t.assignedIP)
	if err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		ep.Close()
		s.Close()
		return ErrTunnelClosed
	}
	t.netStack = s
	t.linkEP = ep
	t.txNotifyCh = notifyCh
	t.ctrlTxCh = make(chan []byte, 64)
	return nil
}

func (t *Tunnel) performHandshake(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if t.tlsConn == nil {
		return errors.New("sstp tunnel missing tls connection")
	}

	deadline := time.Now().Add(DefaultConnectTimeout)
	if d, ok := ctx.Deadline(); ok {
		deadline = d
	}
	_ = t.tlsConn.SetDeadline(deadline)
	defer func() { _ = t.tlsConn.SetDeadline(time.Time{}) }()

	stopCancel := context.AfterFunc(ctx, func() {
		_ = t.tlsConn.SetDeadline(time.Now().Add(-time.Second))
	})
	defer stopCancel()

	wrapHandshakeErr := func(msg string, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("%s: %w", msg, ctxErr)
		}
		return fmt.Errorf("%s: %w", msg, err)
	}

	// 1. Send HTTP Handshake + Call Connect Request + LCP Configure-Request
	corrID := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		rand.Uint32(), rand.Uint32()&0xffff, rand.Uint32()&0xffff, rand.Uint32()&0xffff, rand.Uint64()&0xffffffffffff)

	httpReq := fmt.Sprintf(
		"SSTP_DUPLEX_POST /sra_{BA195980-CD49-458b-9E23-C84EE0ADCD75}/ HTTP/1.1\r\n"+
			"Host: %s:%d\r\n"+
			"Content-Length: 18446744073709551615\r\n"+
			"SSTPCORRELATIONID: {%s}\r\n\r\n",
		t.host, t.port, corrID)

	sstpCtrl := BuildControlPacket(MsgCallConnectRequest, []Attribute{
		{ID: AttrEncapsulatedProtocolID, Data: []byte{0x00, 0x01}}, // PPP
	})

	lcpReq := BuildLCPPacket(PPPCodeConfigureRequest, 1, []byte{
		LCPOptMRU, 4, 0x05, 0xdc, // MRU 1500
	})
	sstpLCP := BuildDataPacket(lcpReq)

	var initBuf []byte
	initBuf = append(initBuf, []byte(httpReq)...)
	initBuf = append(initBuf, sstpCtrl...)
	initBuf = append(initBuf, sstpLCP...)

	if _, err := t.tlsConn.Write(initBuf); err != nil {
		return wrapHandshakeErr("send sstp init handshake failed", err)
	}

	// 2. Read HTTP Response Header using persistent buffered reader
	br := t.ensureReadBr()
	if br == nil {
		return errors.New("sstp tunnel missing tls connection")
	}
	statusLine, err := br.ReadString('\n')
	if err != nil {
		return wrapHandshakeErr("read http status failed", err)
	}
	if !strings.Contains(statusLine, "200") {
		return fmt.Errorf("sstp http handshake rejected: %s", strings.TrimSpace(statusLine))
	}

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return wrapHandshakeErr("read http header failed", err)
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	// 3. Negotiate PPP LCP -> PAP -> IPCP
	ipcpFinished := false
	for round := 0; round < 30 && !ipcpFinished; round++ {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		isCtrl, body, err := t.readPacketDirect(br)
		if err != nil {
			return wrapHandshakeErr("handshake read failed", err)
		}
		if isCtrl {
			continue
		}

		offset := 0
		if len(body) >= 2 && body[0] == 0xff && body[1] == 0x03 {
			offset = 2
		}
		// Require 2-byte PPP protocol + 4-byte PPP header (code, ident, length).
		if len(body)-offset < 6 {
			continue
		}

		proto := binary.BigEndian.Uint16(body[offset : offset+2])
		payload := body[offset+2:]
		code := payload[0]
		ident := payload[1]
		plen := int(binary.BigEndian.Uint16(payload[2:4]))
		if plen < 4 || plen > len(payload) {
			continue
		}

		switch proto {
		case ProtoLCP:
			if code == PPPCodeConfigureRequest {
				// ACK peer's LCP and send PAP Auth Request
				ack := BuildLCPPacket(PPPCodeConfigureAck, ident, payload[4:plen])
				pap := BuildPAPAuthRequest(2, t.username, t.password)
				var out []byte
				out = append(out, BuildDataPacket(ack)...)
				out = append(out, BuildDataPacket(pap)...)
				if _, err := t.tlsConn.Write(out); err != nil {
					return wrapHandshakeErr("send lcp ack failed", err)
				}
			}
		case ProtoPAP:
			if code == PAPCodeAuthAck {
				// PAP Success -> Send IPCP Configure-Request
				ipcp := BuildIPCPPacket(PPPCodeConfigureRequest, 3, []byte{
					IPCPOptIPAddress, 6, 0, 0, 0, 0,
				})
				if _, err := t.tlsConn.Write(BuildDataPacket(ipcp)); err != nil {
					return wrapHandshakeErr("send ipcp request failed", err)
				}
			} else if code == PAPCodeAuthNak {
				return ErrAuthFailed
			}
		case ProtoIPCP:
			if code == PPPCodeConfigureRequest {
				// ACK server's IPCP
				ack := BuildIPCPPacket(PPPCodeConfigureAck, ident, payload[4:plen])
				if _, err := t.tlsConn.Write(BuildDataPacket(ack)); err != nil {
					return wrapHandshakeErr("send ipcp ack failed", err)
				}
			} else if code == PPPCodeConfigureNak {
				// Server assigned virtual IP
				if len(payload) >= 10 && payload[4] == IPCPOptIPAddress && payload[5] == 6 {
					t.assignedIP = net.IPv4(payload[6], payload[7], payload[8], payload[9])
					req := BuildIPCPPacket(PPPCodeConfigureRequest, 4, []byte{
						IPCPOptIPAddress, 6, payload[6], payload[7], payload[8], payload[9],
					})
					if _, err := t.tlsConn.Write(BuildDataPacket(req)); err != nil {
						return wrapHandshakeErr("send ipcp confirm failed", err)
					}
				}
			} else if code == PPPCodeConfigureAck {
				ipcpFinished = true
			}
		}
	}

	if t.assignedIP == nil {
		return ErrIPCPFailed
	}

	// 4. Send SSTP MsgCallConnected
	callConn := BuildControlPacket(MsgCallConnected, nil)
	if _, err := t.tlsConn.Write(callConn); err != nil {
		return wrapHandshakeErr("send call connected failed", err)
	}

	return nil
}

func (t *Tunnel) ensureReadBr() *bufio.Reader {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.readBr == nil && t.tlsConn != nil {
		t.readBr = bufio.NewReaderSize(t.tlsConn, 64*1024)
	}
	return t.readBr
}

func (t *Tunnel) readPacketDirect(r io.Reader) (bool, []byte, error) {
	hdr := make([]byte, 4)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return false, nil, err
	}
	pktLen := int(binary.BigEndian.Uint16(hdr[2:4]) & 0x0fff)
	isCtrl := (hdr[1] & 1) != 0
	if pktLen < 4 {
		return false, nil, errors.New("invalid sstp packet length")
	}

	body := make([]byte, pktLen-4)
	if _, err := io.ReadFull(r, body); err != nil {
		return false, nil, err
	}
	return isCtrl, body, nil
}

func (t *Tunnel) readPacketInto(r io.Reader, hdr *[4]byte, bodyBuf []byte) (bool, []byte, error) {
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return false, nil, err
	}
	pktLen := int(binary.BigEndian.Uint16(hdr[2:4]) & 0x0fff)
	isCtrl := (hdr[1] & 1) != 0
	if pktLen < 4 {
		return false, nil, errors.New("invalid sstp packet length")
	}
	bodyLen := pktLen - 4
	if cap(bodyBuf) < bodyLen {
		bodyBuf = make([]byte, bodyLen)
	}
	body := bodyBuf[:bodyLen]
	if _, err := io.ReadFull(r, body); err != nil {
		return false, nil, err
	}
	return isCtrl, body, nil
}

func (t *Tunnel) readLoop() {
	defer t.Close()

	br := t.ensureReadBr()
	if br == nil {
		return
	}

	var hdr [4]byte
	rxBuf := make([]byte, 4096)

	for {
		isCtrl, body, err := t.readPacketInto(br, &hdr, rxBuf)
		if err != nil {
			t.noteCloseReason(fmt.Sprintf("read from peer failed: %v", err))
			return
		}
		t.markActivity()

		if isCtrl {
			if len(body) >= 2 {
				msgType := binary.BigEndian.Uint16(body[0:2])
				if msgType == MsgEchoRequest {
					// Respond with MsgEchoResponse
					echoResp := BuildControlPacket(MsgEchoResponse, nil)
					_ = t.sendRaw(echoResp)
				} else if msgType == MsgCallDisconnect || msgType == MsgCallAbort {
					t.noteCloseReason(fmt.Sprintf("peer sent control message 0x%04x", msgType))
					return
				}
			}
			continue
		}

		offset := 0
		if len(body) >= 2 && body[0] == 0xff && body[1] == 0x03 {
			offset = 2
		}
		if len(body)-offset < 2 {
			continue
		}

		proto := binary.BigEndian.Uint16(body[offset : offset+2])
		if proto != ProtoIPv4 {
			if t.handlePPPControl(proto, body[offset+2:]) {
				return
			}
			continue
		}

		ipPacket := body[offset+2:]
		if len(ipPacket) < 20 {
			continue
		}

		t.mu.RLock()
		ep := t.linkEP
		closed := t.closed
		t.mu.RUnlock()
		if closed || ep == nil {
			continue
		}

		pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
			Payload: buffer.MakeWithData(ipPacket),
		})
		ep.InjectInbound(ipv4.ProtocolNumber, pkt)
		pkt.DecRef()
	}
}

func (t *Tunnel) writeLoop() {
	batchBuf := make([]byte, 0, 32*1024)
	hasMore := false

	for {
		var ctrl []byte
		if !hasMore {
			select {
			case <-t.closeCh:
				return
			case ctrl = <-t.ctrlTxCh:
			case <-t.txNotifyCh:
			}
		}
		hasMore = false
		batchBuf = batchBuf[:0]

		if len(ctrl) > 0 {
			batchBuf = append(batchBuf, ctrl...)
		}
	drainCtrl:
		for len(batchBuf) < maxCoalesceBytes {
			select {
			case extra := <-t.ctrlTxCh:
				if len(extra) > 0 {
					batchBuf = append(batchBuf, extra...)
				}
			default:
				break drainCtrl
			}
		}

		t.mu.RLock()
		ep := t.linkEP
		tlsConn := t.tlsConn
		closed := t.closed
		t.mu.RUnlock()
		if closed {
			return
		}

		if ep != nil {
			pktCount := 0
			for pktCount < maxCoalescePackets && len(batchBuf) < maxCoalesceBytes {
				pkt := ep.Read()
				if pkt == nil {
					break
				}
				ipLen := pkt.Size()
				frameLen := 8 + ipLen
				if ipLen > 0 && frameLen <= 0x0fff {
					batchBuf = append(batchBuf,
						SSTPVersion,
						SSTPFlagData,
						byte((frameLen>>8)|0x80),
						byte(frameLen),
						0xff,
						0x03,
						byte(ProtoIPv4>>8),
						byte(ProtoIPv4),
					)
					for _, slice := range pkt.AsSlices() {
						batchBuf = append(batchBuf, slice...)
					}
					pktCount++
				}
				pkt.DecRef()
			}
			if pktCount >= maxCoalescePackets || len(batchBuf) >= maxCoalesceBytes {
				hasMore = true
			}
		}

		if len(batchBuf) == 0 || tlsConn == nil {
			continue
		}

		_ = tlsConn.SetWriteDeadline(time.Now().Add(DefaultWriteTimeout))
		if _, err := tlsConn.Write(batchBuf); err != nil {
			t.noteCloseReason(fmt.Sprintf("write to peer failed: %v", err))
			_ = t.Close()
			return
		}
	}
}

func (t *Tunnel) keepAliveLoop() {
	t.runKeepAlive(DefaultEchoInterval)
}

func (t *Tunnel) runKeepAlive(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-t.closeCh:
			return
		case <-ticker.C:
			if !t.IsHealthy() {
				t.noteCloseReason("idle timeout waiting for peer activity")
				_ = t.Close()
				return
			}
			echoReq := BuildControlPacket(MsgEchoRequest, nil)
			if err := t.sendRaw(echoReq); err != nil {
				return
			}
			// SoftEther relays also police the PPP link layer. An SSTP-level echo
			// alone keeps TLS alive but does not satisfy LCP, so drive both.
			if err := t.sendPPP(BuildLCPPacket(PPPCodeEchoRequest, 0, pppMagic)); err != nil {
				return
			}
		}
	}
}

func (t *Tunnel) sendRaw(b []byte) error {
	t.mu.RLock()
	closed := t.closed
	ctrlCh := t.ctrlTxCh
	t.mu.RUnlock()

	if closed {
		return ErrTunnelClosed
	}

	if ctrlCh == nil {
		return nil
	}

	cp := append([]byte(nil), b...)
	select {
	case <-t.closeCh:
		return ErrTunnelClosed
	case ctrlCh <- cp:
		return nil
	case <-time.After(2 * time.Second):
		return ErrTimeout
	}
}

func (t *Tunnel) OpenPacketConn() (net.PacketConn, error) {
	t.mu.RLock()
	closed := t.closed
	netStack := t.netStack
	assigned := t.assignedIP
	t.mu.RUnlock()
	if closed || netStack == nil || assigned == nil {
		return nil, ErrTunnelClosed
	}

	ip4 := assigned.To4()
	if ip4 == nil {
		return nil, ErrTunnelClosed
	}

	laddr := tcpip.FullAddress{
		NIC:  sstpNICID,
		Addr: tcpip.AddrFrom4Slice(ip4),
		Port: 0,
	}
	rawUDP, err := gonet.DialUDP(netStack, &laddr, nil, ipv4.ProtocolNumber)
	if err != nil {
		return nil, fmt.Errorf("sstp open packet conn: %w", err)
	}

	return newUDPConn(rawUDP), nil
}

func (t *Tunnel) tunnelResolver() (*dns.DualEngineResolver, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil, ErrTunnelClosed
	}
	if t.dnsResolver == nil {
		transport := &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, _, err := net.SplitHostPort(addr)
				if err != nil || net.ParseIP(host) == nil {
					return nil, fmt.Errorf("DoH bootstrap requires literal IP")
				}
				return t.Dial(ctx, "tcp4", addr)
			},
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			MaxIdleConns:    2, MaxIdleConnsPerHost: 1, MaxConnsPerHost: 4,
			IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 3 * time.Second,
		}
		t.dnsTransport = transport
		resolver := dns.NewTunnelResolver([]string{dns.CloudflareDoHIP, dns.GoogleDoHIP}, &http.Client{Transport: transport, Timeout: 5 * time.Second})
		resolver.SetPrimaryLookup(t.lookupIPv4ViaUDP)
		t.dnsResolver = resolver
	}
	return t.dnsResolver, nil
}

func (t *Tunnel) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	if network != "tcp" && network != "tcp4" {
		return nil, fmt.Errorf("unsupported network %q for sstp dialer", network)
	}

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address %q: %w", addr, err)
	}
	portInt, err := strconv.Atoi(portStr)
	if err != nil || portInt <= 0 || portInt > 65535 {
		return nil, fmt.Errorf("invalid port %q", portStr)
	}
	targetPort := uint16(portInt)

	// Check tunnel health BEFORE attempting DNS so stale/black-holed peers fail fast.
	t.mu.RLock()
	closed := t.closed
	lastRecv := t.lastRecv
	netStack := t.netStack
	t.mu.RUnlock()
	if closed {
		return nil, ErrTunnelClosed
	}
	if !lastRecv.IsZero() && time.Since(lastRecv) > DefaultIdleTimeout {
		return nil, ErrTunnelStale
	}

	// Resolve over this tunnel, never via host/system DNS. Per-instance cache
	// and transport are discarded on Close; a replacement tunnel has a fresh namespace.
	targetIP := net.ParseIP(host)
	if targetIP == nil {
		resolver, err := t.tunnelResolver()
		if err != nil {
			return nil, err
		}
		ips, err := resolver.LookupIPv4(ctx, host)
		if err != nil || len(ips) == 0 {
			return nil, fmt.Errorf("tunnel DNS for %q: %w", host, err)
		}
		targetIP = ips[0].To4()
	} else {
		targetIP = targetIP.To4()
		if targetIP == nil {
			return nil, fmt.Errorf("SSTP supports IPv4 targets only")
		}
	}

	if netStack == nil {
		t.mu.RLock()
		netStack = t.netStack
		closed = t.closed
		t.mu.RUnlock()
		if closed || netStack == nil {
			return nil, ErrTunnelClosed
		}
	}

	dialCtx := ctx
	var cancel context.CancelFunc
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		dialCtx, cancel = context.WithTimeout(ctx, DefaultDialTimeout)
		defer cancel()
	}

	dst := tcpip.FullAddress{
		NIC:  sstpNICID,
		Addr: tcpip.AddrFrom4Slice(targetIP),
		Port: targetPort,
	}
	return gonet.DialContextTCP(dialCtx, netStack, dst, ipv4.ProtocolNumber)
}

func (t *Tunnel) AssignedIP() net.IP {
	return t.assignedIP
}

func (t *Tunnel) Port() int {
	return t.port
}

// Done returns a channel that is closed when the SSTP tunnel closes.
func (t *Tunnel) Done() <-chan struct{} {
	return t.closeCh
}

// pppMagic is the LCP Magic-Number advertised by this client. PPP requires an
// Echo-Reply to carry the replying peer's own magic number.
var pppMagic = []byte{0x4e, 0x58, 0x47, 0x54} // "NXGT"

// sendPPP frames and writes a PPP control payload inside an SSTP data packet.
func (t *Tunnel) sendPPP(pppPayload []byte) error {
	return t.sendRaw(BuildDataPacket(pppPayload))
}

// handlePPPControl answers link-layer control protocols that arrive after the
// handshake completes and reports whether the session must be torn down.
//
// This is not optional: SoftEther relays use PPP LCP echo requests as their
// session keepalive. A client that only speaks SSTP-level echoes never answers
// them, so the relay declares the link dead and disconnects with
// MsgCallDisconnect roughly a minute after the tunnel comes up - which shows up
// as tunnels that connect successfully and then die in a loop.
func (t *Tunnel) handlePPPControl(proto uint16, payload []byte) (fatal bool) {
	if len(payload) < 4 {
		return false
	}
	code := payload[0]
	ident := payload[1]
	plen := int(binary.BigEndian.Uint16(payload[2:4]))
	if plen < 4 || plen > len(payload) {
		plen = len(payload)
	}
	opts := payload[4:plen]

	switch proto {
	case ProtoLCP:
		switch code {
		case PPPCodeEchoRequest:
			// Echo the peer's data back, as every PPP implementation does.
			_ = t.sendPPP(BuildLCPPacket(PPPCodeEchoReply, ident, opts))
		case PPPCodeEchoReply:
			// Nothing to do: inbound activity already refreshed liveness.
		case PPPCodeConfigureRequest:
			_ = t.sendPPP(BuildLCPPacket(PPPCodeConfigureAck, ident, opts))
		case PPPCodeTerminateRequest:
			_ = t.sendPPP(BuildLCPPacket(PPPCodeTerminateAck, ident, opts))
			t.noteCloseReason("peer terminated the PPP link (LCP Terminate-Request)")
			return true
		}
	case ProtoIPCP:
		switch code {
		case PPPCodeConfigureRequest:
			_ = t.sendPPP(BuildIPCPPacket(PPPCodeConfigureAck, ident, opts))
		case PPPCodeTerminateRequest:
			_ = t.sendPPP(BuildIPCPPacket(PPPCodeTerminateAck, ident, opts))
			t.noteCloseReason("peer terminated IPCP (Terminate-Request)")
			return true
		}
	}
	return false
}

// noteCloseReason records why the session ended so operators can tell a relay
// side disconnect apart from a local transport failure.
func (t *Tunnel) noteCloseReason(reason string) {
	t.mu.Lock()
	if t.closeReason == "" {
		t.closeReason = reason
	}
	t.mu.Unlock()
}

// markActivity records inbound liveness for IsHealthy.
func (t *Tunnel) markActivity() {
	t.mu.Lock()
	t.lastRecv = time.Now()
	t.mu.Unlock()
}

func (t *Tunnel) IsHealthy() bool {
	t.mu.RLock()
	closed := t.closed
	assigned := t.assignedIP
	lastRecv := t.lastRecv
	t.mu.RUnlock()

	if closed || assigned == nil {
		return false
	}
	// A relay that stopped forwarding (or was silently black-holed) leaves the
	// TCP socket open, so liveness must be judged on inbound activity instead of
	// on the socket merely being writable.
	if !lastRecv.IsZero() && time.Since(lastRecv) > DefaultIdleTimeout {
		return false
	}
	return true
}

func (t *Tunnel) Close() error {
	t.closeOnce.Do(func() {
		t.mu.Lock()
		t.closed = true
		if t.dnsResolver != nil {
			t.dnsResolver.Clear()
		}
		transport := t.dnsTransport
		ep := t.linkEP
		netStack := t.netStack
		tlsConn := t.tlsConn
		t.linkEP = nil
		t.netStack = nil
		close(t.closeCh)
		reason := t.closeReason
		t.mu.Unlock()

		if transport != nil {
			transport.CloseIdleConnections()
		}
		if ep != nil {
			ep.Close()
		}
		if netStack != nil {
			netStack.Close()
		}
		if tlsConn != nil {
			_ = tlsConn.Close()
		}
		if reason == "" {
			reason = "closed locally"
		}
		stats.LogInfo("SSTP", "用户态免 TUN 隧道已关闭: %s:%d (%s)", t.host, t.port, reason)
	})
	return nil
}
