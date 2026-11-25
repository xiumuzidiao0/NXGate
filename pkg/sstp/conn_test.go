package sstp

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"aimili-vpngate-go/pkg/dns"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

type sstpBridgeOpts struct {
	dropThirdClientData bool
	reorderServerData   bool
	onClientIPv4        func(ipPkt []byte)
}

// sstpPeerBridge wires a real Tunnel (client gVisor stack + SSTP readLoop/writeLoop over TLS)
// to a peer gVisor stack acting as the remote Internet host on the relay side.
type sstpPeerBridge struct {
	t         *testing.T
	lb        *pppLoopback
	tun       *Tunnel
	peerStack *stack.Stack
	peerEP    *channel.Endpoint
	peerNotify chan struct{}
	closeCh   chan struct{}
	closeOnce sync.Once
}

func newSSTPPeerBridge(t *testing.T, opts sstpBridgeOpts) *sstpPeerBridge {
	t.Helper()
	lb := newLoopbackPPP(t)
	tun := lb.newTunnel(t)
	if err := tun.initNetstack(); err != nil {
		lb.Close()
		t.Fatalf("initNetstack failed: %v", err)
	}

	peerStack := stack.New(stack.Options{
		NetworkProtocols:   []stack.NetworkProtocolFactory{ipv4.NewProtocol},
		TransportProtocols: []stack.TransportProtocolFactory{tcp.NewProtocol, udp.NewProtocol},
	})
	sack := tcpip.TCPSACKEnabled(true)
	_ = peerStack.SetTransportProtocolOption(tcp.ProtocolNumber, &sack)

	peerEP := channel.New(512, DefaultTunnelMTU, "")
	notifyCh := make(chan struct{}, 1)
	peerEP.AddNotify(&channelNotification{ch: notifyCh})

	if err := peerStack.CreateNIC(sstpNICID, peerEP); err != nil {
		peerEP.Close()
		peerStack.Close()
		_ = tun.Close()
		lb.Close()
		t.Fatalf("peer CreateNIC failed: %v", err)
	}
	// Accept packets destined for any remote IP on the peer side (promiscuous + spoofing)
	_ = peerStack.SetPromiscuousMode(sstpNICID, true)
	_ = peerStack.SetSpoofing(sstpNICID, true)
	peerStack.SetRouteTable([]tcpip.Route{{Destination: header.IPv4EmptySubnet, NIC: sstpNICID}})

	b := &sstpPeerBridge{
		t:          t,
		lb:         lb,
		tun:        tun,
		peerStack:  peerStack,
		peerEP:     peerEP,
		peerNotify: notifyCh,
		closeCh:    make(chan struct{}),
	}

	go tun.writeLoop()
	go tun.readLoop()
	go b.pumpClientToPeer(opts)
	go b.pumpPeerToClient(opts)

	return b
}

func (b *sstpPeerBridge) Close() {
	b.closeOnce.Do(func() {
		close(b.closeCh)
		_ = b.tun.Close()
		b.peerEP.Close()
		b.peerStack.Close()
		b.lb.Close()
	})
}

func (b *sstpPeerBridge) pumpClientToPeer(opts sstpBridgeOpts) {
	hdr := make([]byte, 4)
	var clientDataCount atomic.Int32
	for {
		if _, err := io.ReadFull(b.lb.br, hdr); err != nil {
			return
		}
		pktLen := int(binary.BigEndian.Uint16(hdr[2:4]) & 0x0fff)
		if pktLen < 4 {
			return
		}
		body := make([]byte, pktLen-4)
		if _, err := io.ReadFull(b.lb.br, body); err != nil {
			return
		}
		isCtrl := (hdr[1] & 1) != 0
		if isCtrl {
			continue
		}
		off := 0
		if len(body) >= 2 && body[0] == 0xff && body[1] == 0x03 {
			off = 2
		}
		if len(body)-off < 2 {
			continue
		}
		proto := binary.BigEndian.Uint16(body[off : off+2])
		if proto != ProtoIPv4 {
			continue
		}
		ipPkt := body[off+2:]
		if opts.onClientIPv4 != nil {
			opts.onClientIPv4(append([]byte(nil), ipPkt...))
		}
		if opts.dropThirdClientData && len(ipPkt) >= 40 && ipPkt[9] == 6 {
			ipHdrLen := int(ipPkt[0]&0x0f) * 4
			if len(ipPkt) > ipHdrLen+20 {
				tcpHdrLen := int((ipPkt[ipHdrLen+12]>>4)&0x0f) * 4
				if len(ipPkt) > ipHdrLen+tcpHdrLen {
					if clientDataCount.Add(1) == 3 {
						// Drop 3rd outgoing TCP data packet once to verify RTO/RACK recovery.
						continue
					}
				}
			}
		}

		pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
			Payload: buffer.MakeWithData(ipPkt),
		})
		b.peerEP.InjectInbound(ipv4.ProtocolNumber, pkt)
		pkt.DecRef()
	}
}

func (b *sstpPeerBridge) pumpPeerToClient(opts sstpBridgeOpts) {
	var held []byte
	for {
		select {
		case <-b.closeCh:
			return
		case <-b.peerNotify:
		}
		for {
			pkt := b.peerEP.Read()
			if pkt == nil {
				break
			}
			var raw []byte
			for _, s := range pkt.AsSlices() {
				raw = append(raw, s...)
			}
			pkt.DecRef()

			if opts.reorderServerData && held == nil && len(raw) >= 48 && raw[9] == 6 {
				ipHdrLen := int(raw[0]&0x0f) * 4
				if len(raw) > ipHdrLen+20 {
					tcpHdrLen := int((raw[ipHdrLen+12]>>4)&0x0f) * 4
					if len(raw) > ipHdrLen+tcpHdrLen {
						// Hold first server TCP data segment briefly so next segment arrives out of order.
						held = raw
						continue
					}
				}
			}

			frameLen := 8 + len(raw)
			frame := make([]byte, frameLen)
			frame[0] = SSTPVersion
			frame[1] = SSTPFlagData
			binary.BigEndian.PutUint16(frame[2:4], uint16(frameLen)|0x8000)
			frame[4] = 0xff
			frame[5] = 0x03
			binary.BigEndian.PutUint16(frame[6:8], ProtoIPv4)
			copy(frame[8:], raw)
			if _, err := b.lb.Write(frame); err != nil {
				return
			}

			if held != nil {
				hFrameLen := 8 + len(held)
				hFrame := make([]byte, hFrameLen)
				hFrame[0] = SSTPVersion
				hFrame[1] = SSTPFlagData
				binary.BigEndian.PutUint16(hFrame[2:4], uint16(hFrameLen)|0x8000)
				hFrame[4] = 0xff
				hFrame[5] = 0x03
				binary.BigEndian.PutUint16(hFrame[6:8], ProtoIPv4)
				copy(hFrame[8:], held)
				held = nil
				if _, err := b.lb.Write(hFrame); err != nil {
					return
				}
			}
		}
	}
}

func (b *sstpPeerBridge) addPeerIP(ip net.IP) {
	ip4 := ip.To4()
	if ip4 == nil {
		return
	}
	addr := tcpip.AddrFrom4Slice(ip4)
	_ = b.peerStack.AddProtocolAddress(sstpNICID, tcpip.ProtocolAddress{
		Protocol:          ipv4.ProtocolNumber,
		AddressWithPrefix: addr.WithPrefix(),
	}, stack.AddressProperties{})
}

func (b *sstpPeerBridge) listenPeerTCP(ip net.IP, port uint16) (*gonet.TCPListener, error) {
	b.addPeerIP(ip)
	addr := tcpip.FullAddress{
		NIC:  sstpNICID,
		Addr: tcpip.AddrFrom4Slice(ip.To4()),
		Port: port,
	}
	return gonet.ListenTCP(b.peerStack, addr, ipv4.ProtocolNumber)
}

func (b *sstpPeerBridge) listenPeerUDP(ip net.IP, port uint16) (*gonet.UDPConn, error) {
	b.addPeerIP(ip)
	addr := tcpip.FullAddress{
		NIC:  sstpNICID,
		Addr: tcpip.AddrFrom4Slice(ip.To4()),
		Port: port,
	}
	return gonet.DialUDP(b.peerStack, &addr, nil, ipv4.ProtocolNumber)
}

func hasTCPOptions(ipPkt []byte, wantMSS uint16) (gotMSS, gotWScale, gotSACK bool) {
	if len(ipPkt) < 40 || ipPkt[9] != 6 {
		return false, false, false
	}
	ipHdrLen := int(ipPkt[0]&0x0f) * 4
	tcpHdr := ipPkt[ipHdrLen:]
	if len(tcpHdr) < 20 {
		return false, false, false
	}
	const tcpFlagSYN = 0x02
	flags := tcpHdr[13]
	if flags&tcpFlagSYN == 0 {
		return false, false, false
	}
	dataOff := int((tcpHdr[12]>>4)&0x0f) * 4
	if dataOff <= 20 || len(tcpHdr) < dataOff {
		return false, false, false
	}
	opts := tcpHdr[20:dataOff]
	for i := 0; i < len(opts); {
		kind := opts[i]
		if kind == 0 {
			break
		}
		if kind == 1 {
			i++
			continue
		}
		if i+1 >= len(opts) {
			break
		}
		optLen := int(opts[i+1])
		if optLen < 2 || i+optLen > len(opts) {
			break
		}
		switch kind {
		case 2: // MSS
			if optLen == 4 {
				mss := binary.BigEndian.Uint16(opts[i+2 : i+4])
				if mss == wantMSS {
					gotMSS = true
				}
			}
		case 3: // Window Scale
			if optLen == 3 {
				gotWScale = true
			}
		case 4: // SACK Permitted
			gotSACK = true
		}
		i += optLen
	}
	return gotMSS, gotWScale, gotSACK
}

func TestNetstackSYNIncludesMSSWindowScaleAndSACK(t *testing.T) {
	var sawOptions atomic.Bool
	bridge := newSSTPPeerBridge(t, sstpBridgeOpts{
		onClientIPv4: func(ipPkt []byte) {
			gotMSS, gotWScale, gotSACK := hasTCPOptions(ipPkt, DefaultTunnelMTU-40)
			if gotMSS && gotWScale && gotSACK {
				sawOptions.Store(true)
			}
		},
	})
	defer bridge.Close()

	ln, err := bridge.listenPeerTCP(net.ParseIP("93.184.216.34"), 443)
	if err != nil {
		t.Fatalf("listenPeerTCP failed: %v", err)
	}
	defer ln.Close()
	go func() {
		c, aerr := ln.Accept()
		if aerr == nil {
			_ = c.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := bridge.tun.Dial(ctx, "tcp", "93.184.216.34:443")
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	_ = conn.Close()

	if !sawOptions.Load() {
		t.Fatal("expected outgoing TCP SYN to negotiate MSS=1360, Window Scale, and SACK")
	}
}

func TestNetstackBulkTransferWithLossAndReorder(t *testing.T) {
	bridge := newSSTPPeerBridge(t, sstpBridgeOpts{
		dropThirdClientData: true,
		reorderServerData:   true,
	})
	defer bridge.Close()

	ln, err := bridge.listenPeerTCP(net.ParseIP("93.184.216.34"), 8080)
	if err != nil {
		t.Fatalf("listenPeerTCP failed: %v", err)
	}
	defer ln.Close()

	const payloadSize = 512 * 1024
	payload := make([]byte, payloadSize)
	for i := range payload {
		payload[i] = byte((i*31 + 7) & 0xff)
	}

	serverErr := make(chan error, 1)
	go func() {
		c, aerr := ln.Accept()
		if aerr != nil {
			serverErr <- aerr
			return
		}
		defer c.Close()
		_, werr := io.Copy(c, c)
		serverErr <- werr
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := bridge.tun.Dial(ctx, "tcp", "93.184.216.34:8080")
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	writeDone := make(chan error, 1)
	go func() {
		_, werr := conn.Write(payload)
		if cw, ok := conn.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		}
		writeDone <- werr
	}()

	got := make([]byte, payloadSize)
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("ReadFull after loss & reorder failed: %v", err)
	}
	if err := <-writeDone; err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("512 KiB echo payload mismatch")
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("server copy error: %v", err)
	}
}

func TestNetstackRemoteRSTClosesPromptly(t *testing.T) {
	bridge := newSSTPPeerBridge(t, sstpBridgeOpts{})
	defer bridge.Close()

	ln, err := bridge.listenPeerTCP(net.ParseIP("93.184.216.34"), 9000)
	if err != nil {
		t.Fatalf("listenPeerTCP failed: %v", err)
	}
	defer ln.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		c, aerr := ln.Accept()
		if aerr == nil {
			accepted <- c
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	clientConn, err := bridge.tun.Dial(ctx, "tcp", "93.184.216.34:9000")
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer clientConn.Close()

	srvConn := <-accepted
	// Force TCP abortive reset (RST) via SetLinger(0) if supported, else Close.
	if lc, ok := srvConn.(interface{ SetLinger(int) error }); ok {
		_ = lc.SetLinger(0)
	}
	_ = srvConn.Close()

	_ = clientConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 32)
	start := time.Now()
	if _, err := clientConn.Read(buf); err == nil {
		t.Fatal("expected error after peer abortive close, got nil")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("read after peer reset blocked too long: %s", elapsed)
	}
}

func buildMockDNSAnswer(query []byte, answerIP net.IP) []byte {
	// Echo query + append compressed CNAME and A RR
	resp := make([]byte, len(query))
	copy(resp, query)
	binary.BigEndian.PutUint16(resp[2:4], 0x8180) // standard response, no error
	binary.BigEndian.PutUint16(resp[6:8], 2)      // ANCOUNT = 2

	// Answer 1: CNAME pointing name 0xc00c -> "edge.alias" (0x04 edge 0x05 alias 0x00)
	cnameTarget := []byte{4, 'e', 'd', 'g', 'e', 5, 'a', 'l', 'i', 'a', 's', 0}
	resp = append(resp, 0xc0, 0x0c)
	resp = append(resp, 0x00, 0x05)             // TYPE = CNAME
	resp = append(resp, 0x00, 0x01)             // CLASS = IN
	resp = append(resp, 0x00, 0x00, 0x00, 0x78) // TTL = 120s
	resp = append(resp, 0x00, byte(len(cnameTarget)))
	resp = append(resp, cnameTarget...)

	// Answer 2: A record for alias
	ip4 := answerIP.To4()
	resp = append(resp, 0xc0, 0x0c)
	resp = append(resp, 0x00, 0x01)             // TYPE = A
	resp = append(resp, 0x00, 0x01)             // CLASS = IN
	resp = append(resp, 0x00, 0x00, 0x00, 0x5a) // TTL = 90s
	resp = append(resp, 0x00, 0x04)             // RDLENGTH = 4
	resp = append(resp, ip4...)
	return resp
}

func TestTunnelUDPDNSAndDoHFallback(t *testing.T) {
	// 1. Primary UDP 53 works in 1 RTT (without hitting DoH)
	bridge := newSSTPPeerBridge(t, sstpBridgeOpts{})
	defer bridge.Close()

	dnsSrv, err := bridge.listenPeerUDP(net.ParseIP("1.1.1.1"), 53)
	if err != nil {
		t.Fatalf("listenPeerUDP 1.1.1.1:53 failed: %v", err)
	}
	defer dnsSrv.Close()

	var udpCalls atomic.Int32
	go func() {
		buf := make([]byte, 512)
		for {
			n, from, rerr := dnsSrv.ReadFrom(buf)
			if rerr != nil {
				return
			}
			udpCalls.Add(1)
			resp := buildMockDNSAnswer(buf[:n], net.ParseIP("203.0.113.55"))
			_, _ = dnsSrv.WriteTo(resp, from)
		}
	}()

	resolver, err := bridge.tun.tunnelResolver()
	if err != nil {
		t.Fatalf("tunnelResolver failed: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ips, err := resolver.LookupIPv4(ctx, "fast.example.com")
	if err != nil {
		t.Fatalf("LookupIPv4 via UDP 53 failed: %v", err)
	}
	if len(ips) != 1 || ips[0].String() != "203.0.113.55" {
		t.Fatalf("expected [203.0.113.55], got %v", ips)
	}
	if udpCalls.Load() != 1 {
		t.Fatalf("expected 1 UDP 53 query, got %d", udpCalls.Load())
	}

	// 2. When UDP 53 drops packets, resolver automatically falls back to tunnel-internal DoH.
	fallbackBridge := newSSTPPeerBridge(t, sstpBridgeOpts{})
	defer fallbackBridge.Close()

	dohRawLn, err := fallbackBridge.listenPeerTCP(net.ParseIP("1.1.1.1"), 443)
	if err != nil {
		t.Fatalf("listenPeerTCP 1.1.1.1:443 failed: %v", err)
	}
	defer dohRawLn.Close()

	srvTLS := generateTestTLSConfigWithSANs(t, []net.IP{net.ParseIP("1.1.1.1")}, nil)
	leafCert, err := x509.ParseCertificate(srvTLS.Certificates[0].Certificate[0])
	if err != nil {
		t.Fatalf("parse cert failed: %v", err)
	}
	rootPool := x509.NewCertPool()
	rootPool.AddCert(leafCert)

	dohTLS := tls.NewListener(dohRawLn, srvTLS)
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/dns-json")
			_, _ = w.Write([]byte(`{"Status":0,"Answer":[{"name":"doh.example.com","type":1,"TTL":60,"data":"198.51.100.77"}]}`))
		}),
	}
	defer srv.Close()
	go func() { _ = srv.Serve(dohTLS) }()

	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return fallbackBridge.tun.Dial(ctx, "tcp4", addr)
		},
		TLSClientConfig: &tls.Config{RootCAs: rootPool, MinVersion: tls.VersionTLS12},
	}
	defer tr.CloseIdleConnections()

	rFallback := dns.NewTunnelResolver([]string{dns.CloudflareDoHIP}, &http.Client{Transport: tr, Timeout: 4 * time.Second})
	rFallback.SetPrimaryLookup(fallbackBridge.tun.lookupIPv4ViaUDP)

	ctxFB, cancelFB := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFB()
	fbIPs, err := rFallback.LookupIPv4(ctxFB, "doh.example.com")
	if err != nil {
		t.Fatalf("DoH fallback after UDP 53 drop failed: %v", err)
	}
	if len(fbIPs) != 1 || fbIPs[0].String() != "198.51.100.77" {
		t.Fatalf("expected [198.51.100.77], got %v", fbIPs)
	}
}
