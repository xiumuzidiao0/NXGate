package sstp

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
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
