package sstp

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"
)

func generateTestTLSConfig(t *testing.T) *tls.Config {
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
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
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
	tlsCfg := generateTestTLSConfig(t)
	listener, err := tls.Listen("tcp", "127.0.0.1:0", tlsCfg)
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
