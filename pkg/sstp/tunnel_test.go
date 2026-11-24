package sstp

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

func generateTestTLSConfig(t *testing.T) *tls.Config {
	return generateTestTLSConfigWithSANs(t, nil, []string{"opengw.net"})
}

// generateTestTLSConfigWithSANs builds a throwaway server certificate. Real
// SoftEther / VPNGate relays present a certificate issued for their own
// hostname while clients dial literal IP addresses, which is why the default
// fixture below carries a hostname SAN only.
func generateTestTLSConfigWithSANs(t *testing.T, ips []net.IP, dnsNames []string) *tls.Config {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key failed: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"NXGate Test"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           ips,
		DNSNames:              dnsNames,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create cert failed: %v", err)
	}

	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
}

func TestResolveCandidatePortsWithPreferred(t *testing.T) {
	// 1. Without preferred port
	ports := ResolveCandidatePortsWithPreferred(1237, 0)
	if len(ports) < 4 || ports[0] != 1237 {
		t.Fatalf("expected first port to be 1237, got %v", ports)
	}

	// 2. With preferred port
	portsPreferred := ResolveCandidatePortsWithPreferred(1237, 995)
	if len(portsPreferred) < 4 || portsPreferred[0] != 995 {
		t.Fatalf("expected first port to be preferred 995, got %v", portsPreferred)
	}

	// 3. Deduplication
	portsDedupe := ResolveCandidatePortsWithPreferred(443, 443)
	count443 := 0
	for _, p := range portsDedupe {
		if p == 443 {
			count443++
		}
	}
	if count443 != 1 {
		t.Fatalf("expected 443 to appear exactly once, got %v", portsDedupe)
	}
}

func TestFastProbeAndDetectSSTPPort(t *testing.T) {
	listener, err := tls.Listen("tcp", "127.0.0.1:0", generateTestTLSConfig(t))
	if err != nil {
		t.Fatalf("listen tls failed: %v", err)
	}
	defer listener.Close()

	serverPort := listener.Addr().(*net.TCPAddr).Port

	// Run dummy SSTP server
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				reader := bufio.NewReader(c)
				reqLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if reqLine != "" {
					resp := "HTTP/1.1 200 OK\r\nContent-Length: 18446744073709551615\r\nServer: Microsoft-HTTPAPI/2.0\r\n\r\n"
					_, _ = c.Write([]byte(resp))
				}
			}(conn)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 1. Test FastProbeSSTPPort directly
	ok := FastProbeSSTPPort(ctx, "127.0.0.1", serverPort, 1*time.Second)
	if !ok {
		t.Fatalf("FastProbeSSTPPort failed to detect valid SSTP server on port %d", serverPort)
	}

	// FastProbe on non-existent port should return false quickly
	bad := FastProbeSSTPPort(ctx, "127.0.0.1", 1, 300*time.Millisecond)
	if bad {
		t.Fatalf("FastProbeSSTPPort unexpectedly succeeded on dead port 1")
	}

	// 2. Test DetectSSTPPort
	detected := DetectSSTPPort(ctx, "127.0.0.1", []int{1, 2, serverPort, 3}, 1*time.Second)
	if detected != serverPort {
		t.Fatalf("DetectSSTPPort expected %d, got %d", serverPort, detected)
	}
}

// TestSSTPTLSAcceptsHostnameOnlyCertificate locks in the real-world SoftEther
// contract: relays such as VPNGate present a certificate issued for their own
// hostname ("opengw.net") while clients dial literal IP addresses, so chain and
// hostname verification can never succeed. MS-SSTP treats TLS as a transport
// wrapper and authenticates the session in the PPP layer (MS-CHAPv2), therefore
// the client must complete the TLS handshake instead of aborting with a
// certificate error. Regression guard for the Zero-TUN "connecting forever" bug.
func TestSSTPTLSAcceptsHostnameOnlyCertificate(t *testing.T) {
	listener, err := tls.Listen("tcp", "127.0.0.1:0", generateTestTLSConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				reader := bufio.NewReader(c)
				if _, err := reader.ReadString('\n'); err != nil {
					return
				}
				_, _ = c.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 18446744073709551615\r\nServer: Microsoft-HTTPAPI/2.0\r\n\r\n"))
			}(conn)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	port := listener.Addr().(*net.TCPAddr).Port

	if !FastProbeSSTPPort(ctx, "127.0.0.1", port, time.Second) {
		t.Fatal("probe rejected a hostname-only certificate presented by an SSTP listener")
	}

	// Connect must get past the TLS layer; it may still fail later because this
	// stub does not implement the PPP negotiation, but never with a cert error.
	tun, err := Connect(ctx, "127.0.0.1", port, "test", "test")
	if tun != nil {
		_ = tun.Close()
	}
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "certificate") {
		t.Fatalf("TLS layer rejected a hostname-only certificate: %v", err)
	}
}

// TestIsHealthyHonoursIdleTimeout guards the black-hole detection bound: a relay
// that stops forwarding leaves the TCP socket open, so liveness must be judged on
// inbound activity. Before this bound the gateway reported "connected" for many
// minutes while every proxied request failed.
func TestIsHealthyHonoursIdleTimeout(t *testing.T) {
	tun := &Tunnel{
		assignedIP: net.ParseIP("10.211.1.7"),
		closeCh:    make(chan struct{}),
		sessions:   make(map[uint32]*TCPConn),
		udpConns:   make(map[uint16]*UDPConn),
		pending:    make(map[uint16]*pendingHandshake),
	}
	defer tun.Close()

	tun.markActivity()
	if !tun.IsHealthy() {
		t.Fatal("a tunnel that just received traffic must be healthy")
	}

	tun.mu.Lock()
	tun.lastRecv = time.Now().Add(-DefaultIdleTimeout - time.Second)
	tun.mu.Unlock()
	if tun.IsHealthy() {
		t.Fatalf("a tunnel silent for longer than %s must be unhealthy", DefaultIdleTimeout)
	}

	tun.mu.Lock()
	tun.lastRecv = time.Now().Add(-DefaultIdleTimeout / 2)
	tun.mu.Unlock()
	if !tun.IsHealthy() {
		t.Fatalf("a tunnel silent for less than %s must still be healthy", DefaultIdleTimeout)
	}
}

// TestDialRejectsStaleTunnel verifies that a black-holed peer is rejected
// immediately rather than after the full SYN timeout for every request.
func TestDialRejectsStaleTunnel(t *testing.T) {
	tun := &Tunnel{
		assignedIP: net.ParseIP("10.211.1.7"),
		closeCh:    make(chan struct{}),
		sessions:   make(map[uint32]*TCPConn),
		udpConns:   make(map[uint16]*UDPConn),
		pending:    make(map[uint16]*pendingHandshake),
	}
	defer tun.Close()

	tun.markActivity()
	tun.mu.Lock()
	tun.lastRecv = time.Now().Add(-DefaultIdleTimeout - time.Second)
	tun.mu.Unlock()

	start := time.Now()
	_, err := tun.Dial(context.Background(), "tcp", "1.1.1.1:443")
	if !errors.Is(err, ErrTunnelStale) {
		t.Fatalf("expected ErrTunnelStale for a black-holed peer, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("stale dial must fail fast, took %s", elapsed)
	}
}

// TestHandlePPPControlAnswersLCPEcho is the regression guard for the Zero-TUN
// "connects then dies after ~40s" loop: SoftEther relays police the PPP link
// layer with LCP echo requests, so the client must answer them.
func TestHandlePPPControlAnswersLCPEcho(t *testing.T) {
	server := newLoopbackPPP(t)
	defer server.Close()

	tun := server.newTunnel(t)
	defer tun.Close()
	go tun.readLoop()

	// Server asks for an LCP echo the way a SoftEther relay does.
	req := BuildLCPPacket(PPPCodeEchoRequest, 7, []byte{0xde, 0xad, 0xbe, 0xef})
	if _, err := server.Write(BuildDataPacket(req)); err != nil {
		t.Fatalf("write echo request: %v", err)
	}

	proto, code, ident, data := server.readPPPControl(t, 5*time.Second)
	if proto != ProtoLCP || code != PPPCodeEchoReply {
		t.Fatalf("expected LCP Echo-Reply, got proto=0x%04x code=%d", proto, code)
	}
	if ident != 7 {
		t.Fatalf("echo reply must reuse the request identifier, got %d", ident)
	}
	if string(data) != string([]byte{0xde, 0xad, 0xbe, 0xef}) {
		t.Fatalf("echo reply must carry the request data, got %x", data)
	}
}

// TestHandlePPPControlTerminateIsFatal verifies a peer-initiated PPP teardown
// ends the session instead of being silently ignored.
func TestHandlePPPControlTerminateIsFatal(t *testing.T) {
	tun := &Tunnel{
		assignedIP: net.ParseIP("10.211.1.7"),
		closeCh:    make(chan struct{}),
		sessions:   make(map[uint32]*TCPConn),
		udpConns:   make(map[uint16]*UDPConn),
		pending:    make(map[uint16]*pendingHandshake),
	}
	defer tun.Close()

	term := BuildLCPPacket(PPPCodeTerminateRequest, 1, nil)
	if !tun.handlePPPControl(ProtoLCP, term[2:]) {
		t.Fatal("LCP Terminate-Request must be treated as fatal")
	}
	if tun.closeReason == "" {
		t.Fatal("expected a recorded close reason")
	}

	// IPCP teardown must be fatal too.
	termIPCP := BuildIPCPPacket(PPPCodeTerminateRequest, 2, nil)
	if !tun.handlePPPControl(ProtoIPCP, termIPCP[2:]) {
		t.Fatal("IPCP Terminate-Request must be treated as fatal")
	}
}

// TestHandlePPPControlIgnoresMalformed guards against panics on truncated frames.
func TestHandlePPPControlIgnoresMalformed(t *testing.T) {
	tun := &Tunnel{closeCh: make(chan struct{})}
	defer tun.Close()
	for _, payload := range [][]byte{nil, {0x09}, {0x09, 0x01}, {0x09, 0x01, 0xff, 0xff, 0xaa}} {
		if tun.handlePPPControl(ProtoLCP, payload) {
			t.Fatalf("malformed payload %x must not be fatal", payload)
		}
	}
}

// pppLoopback is a minimal SSTP peer used to exercise the post-handshake PPP
// control plane without depending on a real relay.
type pppLoopback struct {
	t      *testing.T
	lis    net.Listener
	srv    net.Conn
	br     *bufio.Reader
	client *tls.Conn
}

func newLoopbackPPP(t *testing.T) *pppLoopback {
	t.Helper()
	lis, err := tls.Listen("tcp", "127.0.0.1:0", generateTestTLSConfig(t))
	if err != nil {
		t.Fatalf("listen tls failed: %v", err)
	}
	lb := &pppLoopback{t: t, lis: lis}

	accepted := make(chan net.Conn, 1)
	go func() {
		c, err := lis.Accept()
		if err != nil {
			return
		}
		// tls.Listen defers the handshake until first use, so drive it here;
		// otherwise the client side blocks forever waiting for the ServerHello.
		if err := c.(*tls.Conn).Handshake(); err != nil {
			return
		}
		accepted <- c
	}()

	raw, err := net.Dial("tcp", lis.Addr().String())
	if err != nil {
		t.Fatalf("dial loopback failed: %v", err)
	}
	_ = raw.SetDeadline(time.Now().Add(5 * time.Second))
	client := tls.Client(raw, sstpTLSConfig("127.0.0.1"))
	if err := client.Handshake(); err != nil {
		t.Fatalf("loopback tls handshake failed: %v", err)
	}
	_ = raw.SetDeadline(time.Time{})
	lb.client = client

	select {
	case c := <-accepted:
		lb.srv = c
		lb.br = bufio.NewReader(c)
	case <-time.After(5 * time.Second):
		t.Fatal("loopback accept timed out")
	}
	return lb
}

func (lb *pppLoopback) Close() {
	if lb.client != nil {
		_ = lb.client.Close()
	}
	if lb.srv != nil {
		_ = lb.srv.Close()
	}
	_ = lb.lis.Close()
}

// newTunnel binds a Tunnel to the client side of the loopback peer.
func (lb *pppLoopback) newTunnel(t *testing.T) *Tunnel {
	t.Helper()
	return &Tunnel{
		host:       "127.0.0.1",
		port:       443,
		assignedIP: net.ParseIP("10.211.1.7"),
		tlsConn:    lb.client,
		sessions:   make(map[uint32]*TCPConn),
		udpConns:   make(map[uint16]*UDPConn),
		pending:    make(map[uint16]*pendingHandshake),
		closeCh:    make(chan struct{}),
	}
}

func (lb *pppLoopback) Write(b []byte) (int, error) { return lb.srv.Write(b) }

// readPPPControl reads one SSTP data frame and returns its PPP control fields.
func (lb *pppLoopback) readPPPControl(t *testing.T, timeout time.Duration) (uint16, byte, byte, []byte) {
	t.Helper()
	_ = lb.srv.SetReadDeadline(time.Now().Add(timeout))

	hdr := make([]byte, 4)
	if _, err := io.ReadFull(lb.br, hdr); err != nil {
		t.Fatalf("read sstp header: %v", err)
	}
	pktLen := int(binary.BigEndian.Uint16(hdr[2:4]) & 0x0fff)
	if pktLen < 4 {
		t.Fatalf("invalid sstp frame length %d", pktLen)
	}
	body := make([]byte, pktLen-4)
	if _, err := io.ReadFull(lb.br, body); err != nil {
		t.Fatalf("read sstp body: %v", err)
	}

	off := 0
	if len(body) >= 2 && body[0] == 0xff && body[1] == 0x03 {
		off = 2
	}
	if len(body)-off < 4 {
		t.Fatalf("ppp payload too short: %x", body)
	}
	proto := binary.BigEndian.Uint16(body[off : off+2])
	payload := body[off+2:]
	plen := int(binary.BigEndian.Uint16(payload[2:4]))
	if plen < 4 || plen > len(payload) {
		plen = len(payload)
	}
	return proto, payload[0], payload[1], payload[4:plen]
}

func (lb *pppLoopback) readIPv4Frame(t *testing.T, timeout time.Duration) []byte {
	t.Helper()
	_ = lb.srv.SetReadDeadline(time.Now().Add(timeout))
	hdr := make([]byte, 4)
	if _, err := io.ReadFull(lb.br, hdr); err != nil {
		t.Fatalf("read sstp frame header: %v", err)
	}
	pktLen := int(binary.BigEndian.Uint16(hdr[2:4]) & 0x0fff)
	if pktLen < 6 {
		t.Fatalf("invalid sstp frame len %d", pktLen)
	}
	body := make([]byte, pktLen-4)
	if _, err := io.ReadFull(lb.br, body); err != nil {
		t.Fatalf("read sstp frame body: %v", err)
	}
	if len(body) < 4 || body[0] != 0xff || body[1] != 0x03 {
		t.Fatalf("unexpected HDLC header: %x", body)
	}
	return body[4:]
}

func TestTunnelReadLoopAndTxBufReuseNoAliasing(t *testing.T) {
	lb := newLoopbackPPP(t)
	defer lb.Close()

	tun := lb.newTunnel(t)
	defer tun.Close()

	remoteTCP := net.ParseIP("93.184.216.34")
	tcpConn := newTCPConn(tun, tun.assignedIP, remoteTCP, 45100, 80, 1000, 2000)
	sessKey := (uint32(45100) << 16) | uint32(80)
	tun.mu.Lock()
	tun.sessions[sessKey] = tcpConn
	tun.mu.Unlock()

	pktConn, err := tun.OpenPacketConn()
	if err != nil {
		t.Fatalf("OpenPacketConn failed: %v", err)
	}
	uConn := pktConn.(*UDPConn)

	go tun.readLoop()

	// Write two UDP packets and two TCP segments back-to-back into rxLoop before consumer reads.
	dns1 := BuildIPv4UDPPacket(net.ParseIP("8.8.8.8"), tun.assignedIP, 53, uConn.localPort, []byte("UDP-FIRST-LONGER-PAYLOAD"))
	dns2 := BuildIPv4UDPPacket(net.ParseIP("1.1.1.1"), tun.assignedIP, 53, uConn.localPort, []byte("UDP-TWO"))
	tcp1 := BuildIPv4TCPPacket(remoteTCP, tun.assignedIP, 80, 45100, 2000, 1000, TCPFlagACK|TCPFlagPSH, []byte("TCP-ONE-"))
	tcp2 := BuildIPv4TCPPacket(remoteTCP, tun.assignedIP, 80, 45100, 2008, 1000, TCPFlagACK|TCPFlagPSH, []byte("TCP-TWO"))

	var batch []byte
	batch = append(batch, dns1...)
	batch = append(batch, dns2...)
	batch = append(batch, tcp1...)
	batch = append(batch, tcp2...)
	if _, err := lb.Write(batch); err != nil {
		t.Fatalf("loopback write failed: %v", err)
	}

	_ = uConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	uBuf := make([]byte, 128)
	n1, _, err := uConn.ReadFrom(uBuf)
	if err != nil || string(uBuf[:n1]) != "UDP-FIRST-LONGER-PAYLOAD" {
		t.Fatalf("first UDP payload corrupted/aliased: %q err=%v", string(uBuf[:n1]), err)
	}
	n2, _, err := uConn.ReadFrom(uBuf)
	if err != nil || string(uBuf[:n2]) != "UDP-TWO" {
		t.Fatalf("second UDP payload mismatch: %q err=%v", string(uBuf[:n2]), err)
	}

	_ = tcpConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	tBuf := make([]byte, 15)
	if _, err := io.ReadFull(tcpConn, tBuf); err != nil || string(tBuf) != "TCP-ONE-TCP-TWO" {
		t.Fatalf("TCP stream mismatch: %q err=%v", string(tBuf), err)
	}

	// Verify txBuf reuse for multiple outgoing TCP & UDP writes produces valid checksums and distinct headers.
	if _, err := tcpConn.Write([]byte("OUT-TCP-A")); err != nil {
		t.Fatalf("tcpConn write A failed: %v", err)
	}
	if _, err := tcpConn.Write([]byte("OUT-TCP-B")); err != nil {
		t.Fatalf("tcpConn write B failed: %v", err)
	}
	if _, err := uConn.WriteTo([]byte("OUT-UDP-1"), &net.UDPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53}); err != nil {
		t.Fatalf("uConn write 1 failed: %v", err)
	}
	if _, err := uConn.WriteTo([]byte("OUT-UDP-2"), &net.UDPAddr{IP: net.ParseIP("1.1.1.1"), Port: 53}); err != nil {
		t.Fatalf("uConn write 2 failed: %v", err)
	}

	var gotPayloads []string
	deadline := time.Now().Add(2 * time.Second)
	for len(gotPayloads) < 4 && time.Now().Before(deadline) {
		ipPkt := lb.readIPv4Frame(t, 2*time.Second)
		// Validate IPv4 checksum was recomputed cleanly on reused txBuf
		if InternetChecksum(ipPkt, 0, 20) != 0 {
			t.Fatalf("invalid IPv4 header checksum on reused txBuf: %x", ipPkt[:20])
		}
		switch ipPkt[9] {
		case 6:
			p, err := ParseIncomingIPv4TCP(ipPkt)
			if err != nil {
				t.Fatalf("parse outgoing TCP failed: %v", err)
			}
			// Skip pure ACK control frames triggered by delayed ACK
			if len(p.Payload) > 0 {
				if tcpChecksum(p.SrcIP.To4(), p.DstIP.To4(), ipPkt[20:]) != 0 {
					t.Fatalf("invalid TCP checksum on reused txBuf")
				}
				gotPayloads = append(gotPayloads, string(p.Payload))
			}
		case 17:
			p, err := ParseIncomingIPv4UDP(ipPkt)
			if err != nil {
				t.Fatalf("parse outgoing UDP failed: %v", err)
			}
			if udpChecksum(p.SrcIP.To4(), p.DstIP.To4(), ipPkt[20:]) != 0xffff {
				t.Fatalf("invalid UDP checksum on reused txBuf")
			}
			gotPayloads = append(gotPayloads, string(p.Payload))
		}
	}

	joined := strings.Join(gotPayloads, "|")
	if joined != "OUT-TCP-A|OUT-TCP-B|OUT-UDP-1|OUT-UDP-2" {
		t.Fatalf("unexpected outgoing payloads on reused txBuf: %q", joined)
	}
}
