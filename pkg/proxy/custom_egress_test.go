package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aimili-vpngate-go/pkg/config"
)

func TestParseProxyURL(t *testing.T) {
	tests := []struct {
		raw      string
		wantType string
		wantHost string
		wantPort int
		wantUser string
		wantPass string
	}{
		{
			raw:      "socks5://user:pass@1.2.3.4:1080",
			wantType: "socks5",
			wantHost: "1.2.3.4",
			wantPort: 1080,
			wantUser: "user",
			wantPass: "pass",
		},
		{
			raw:      "http://proxy.example.com:8080",
			wantType: "http",
			wantHost: "proxy.example.com",
			wantPort: 8080,
			wantUser: "",
			wantPass: "",
		},
		{
			raw:      "https://admin:secret@secure.net:443",
			wantType: "https",
			wantHost: "secure.net",
			wantPort: 443,
			wantUser: "admin",
			wantPass: "secret",
		},
		{
			raw:      "192.168.1.50:1080", // Default scheme: socks5
			wantType: "socks5",
			wantHost: "192.168.1.50",
			wantPort: 1080,
		},
	}

	for _, tt := range tests {
		p, err := ParseProxyURL(tt.raw)
		if err != nil {
			t.Fatalf("ParseProxyURL(%q) failed: %v", tt.raw, err)
		}
		if p.Type != tt.wantType || p.Host != tt.wantHost || p.Port != tt.wantPort {
			t.Errorf("ParseProxyURL(%q) = (%s, %s, %d), want (%s, %s, %d)",
				tt.raw, p.Type, p.Host, p.Port, tt.wantType, tt.wantHost, tt.wantPort)
		}
		if p.Username != tt.wantUser || p.Password != tt.wantPass {
			t.Errorf("ParseProxyURL(%q) auth = (%s, %s), want (%s, %s)",
				tt.raw, p.Username, p.Password, tt.wantUser, tt.wantPass)
		}
	}
}

func TestCustomProxy_SOCKS5Handshake(t *testing.T) {
	// Start mock SOCKS5 server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on local mock socks5: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port

	// Mock echo target
	targetLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on mock target: %v", err)
	}
	defer targetLn.Close()
	targetPort := targetLn.Addr().(*net.TCPAddr).Port

	go func() {
		for {
			conn, err := targetLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				n, _ := c.Read(buf)
				if n > 0 {
					_, _ = c.Write(append([]byte("ECHO:"), buf[:n]...))
				}
			}(conn)
		}
	}()

	// Handle SOCKS5 negotiation
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				// Read version + methods
				buf := make([]byte, 256)
				n, err := c.Read(buf)
				if err != nil || n < 2 {
					return
				}
				// Accept username/password auth (0x02)
				_, _ = c.Write([]byte{0x05, 0x02})

				// Read auth (0x01, ulen, user, plen, pass)
				n, err = c.Read(buf)
				if err != nil || n < 2 {
					return
				}
				// Reply auth success
				_, _ = c.Write([]byte{0x01, 0x00})

				// Read CONNECT (0x05, 0x01, 0x00, 0x01...)
				n, err = c.Read(buf)
				if err != nil || n < 4 {
					return
				}

				// Reply success
				_, _ = c.Write([]byte{0x05, 0x00, 0x00, 0x01, 127, 0, 0, 1, 0, 0})

				// Connect to target
				targetConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", targetPort))
				if err != nil {
					return
				}
				defer targetConn.Close()

				go func() { _, _ = io.Copy(targetConn, c) }()
				_, _ = io.Copy(c, targetConn)
			}(conn)
		}
	}()

	proxy := &CustomProxy{
		ID:       "test-socks5",
		Type:     "socks5",
		Host:     "127.0.0.1",
		Port:     port,
		Username: "testuser",
		Password: "testpass",
		Enabled:  true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := proxy.DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", targetPort))
	if err != nil {
		t.Fatalf("proxy.DialContext failed: %v", err)
	}
	defer conn.Close()

	testMsg := "Hello, NXGate Custom SOCKS5!"
	if _, err := io.WriteString(conn, testMsg); err != nil {
		t.Fatalf("failed to write testMsg: %v", err)
	}

	recv := make([]byte, 1024)
	n, err := conn.Read(recv)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	got := string(recv[:n])
	expected := "ECHO:" + testMsg
	if got != expected {
		t.Errorf("got %q, expected %q", got, expected)
	}
}

func TestCustomProxy_HTTPHandshake(t *testing.T) {
	// Mock HTTP proxy with CONNECT support
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "CONNECT" {
			http.Error(w, "Expected CONNECT", http.StatusBadRequest)
			return
		}

		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}
		clientConn, _, err := hj.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer clientConn.Close()

		_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

		buf := make([]byte, 1024)
		n, _ := clientConn.Read(buf)
		if n > 0 {
			_, _ = clientConn.Write(append([]byte("HTTP_TUNNEL:"), buf[:n]...))
		}
	}))
	defer server.Close()

	serverURL, _ := ParseProxyURL(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := serverURL.DialContext(ctx, "tcp", "example.com:80")
	if err != nil {
		t.Fatalf("HTTP DialContext failed: %v", err)
	}
	defer conn.Close()

	testData := "PingHTTP"
	_, _ = io.WriteString(conn, testData)

	resp := make([]byte, 128)
	n, err := conn.Read(resp)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	got := string(resp[:n])
	if got != "HTTP_TUNNEL:PingHTTP" {
		t.Errorf("got %q, want HTTP_TUNNEL:PingHTTP", got)
	}
}

func TestCustomProxyManager_CRUD(t *testing.T) {
	tempDir := t.TempDir()
	mgr := NewCustomProxyManager(tempDir)

	p, err := mgr.AddOrUpdate("socks5://user:pass@1.1.1.1:1080", "Cloudflare SOCKS5")
	if err != nil {
		t.Fatalf("AddOrUpdate failed: %v", err)
	}
	if p.Tag != "Cloudflare SOCKS5" {
		t.Errorf("expected tag Cloudflare SOCKS5, got %s", p.Tag)
	}

	list := mgr.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 proxy, got %d", len(list))
	}

	got := mgr.Get(p.ID)
	if got == nil || got.Host != "1.1.1.1" {
		t.Fatalf("expected proxy with host 1.1.1.1, got %v", got)
	}

	if !mgr.Delete(p.ID) {
		t.Errorf("expected delete to succeed")
	}

	if len(mgr.List()) != 0 {
		t.Errorf("expected 0 proxies after delete")
	}
}

func TestPortListener_RouteViaCustomProxy(t *testing.T) {
	// 1. Mock final target server
	targetLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("targetLn: %v", err)
	}
	defer targetLn.Close()
	targetPort := targetLn.Addr().(*net.TCPAddr).Port

	go func() {
		for {
			conn, err := targetLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 512)
				n, _ := c.Read(buf)
				if n > 0 {
					_, _ = c.Write(append([]byte("TARGET_REPLY:"), buf[:n]...))
				}
			}(conn)
		}
	}()

	// 2. Mock upstream SOCKS5 proxy server
	upstreamLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("upstreamLn: %v", err)
	}
	defer upstreamLn.Close()
	upstreamPort := upstreamLn.Addr().(*net.TCPAddr).Port

	go func() {
		for {
			conn, err := upstreamLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 256)
				n, _ := c.Read(buf)
				if n < 2 {
					return
				}
				// Accept no-auth (0x00)
				_, _ = c.Write([]byte{0x05, 0x00})

				// Read CONNECT
				n, _ = c.Read(buf)
				if n < 4 {
					return
				}
				// Success reply
				_, _ = c.Write([]byte{0x05, 0x00, 0x00, 0x01, 127, 0, 0, 1, 0, 0})

				targetConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", targetPort))
				if err != nil {
					return
				}
				defer targetConn.Close()

				go func() { _, _ = io.Copy(targetConn, c) }()
				_, _ = io.Copy(c, targetConn)
			}(conn)
		}
	}()

	// 3. Setup CustomProxyManager
	tempDir := t.TempDir()
	mgr := NewCustomProxyManager(tempDir)
	rawProxyURL := fmt.Sprintf("socks5://127.0.0.1:%d", upstreamPort)
	cp, err := mgr.AddOrUpdate(rawProxyURL, "Mock Upstream")
	if err != nil {
		t.Fatalf("AddOrUpdate: %v", err)
	}
	cp.Healthy = true // Mark healthy

	// 4. Start PortListener on an ephemeral port, bound to this custom proxy
	rule := PortRule{
		Port:                0, // ephemeral
		Enabled:             true,
		BoundCustomProxyIDs: []string{cp.ID},
		AuthMode:            "none",
	}

	listener := NewPortListener(rule, &config.Config{ProxyHost: "127.0.0.1", ProxyMaxConnections: 100}, nil)
	listener.SetCustomProxyManager(mgr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = listener.Start(ctx)
	}()

	// Wait for listener to bind
	var listenerAddr string
	for i := 0; i < 20; i++ {
		listener.mu.Lock()
		if listener.listener != nil {
			listenerAddr = listener.listener.Addr().String()
			listener.mu.Unlock()
			break
		}
		listener.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	if listenerAddr == "" {
		t.Fatalf("PortListener failed to start")
	}
	defer listener.Close()

	// 5. Connect as a client via this PortListener
	clientConn, err := net.Dial("tcp", listenerAddr)
	if err != nil {
		t.Fatalf("dial listener: %v", err)
	}
	defer clientConn.Close()

	// Client sends SOCKS5 greeting
	_, _ = clientConn.Write([]byte{0x05, 0x01, 0x00})
	greetingResp := make([]byte, 2)
	_, _ = io.ReadFull(clientConn, greetingResp)

	// Client sends CONNECT to targetPort
	connectReq := []byte{0x05, 0x01, 0x00, 0x01, 127, 0, 0, 1, byte(targetPort >> 8), byte(targetPort & 0xff)}
	_, _ = clientConn.Write(connectReq)
	connectResp := make([]byte, 10)
	_, _ = io.ReadFull(clientConn, connectResp)
	if connectResp[1] != 0x00 {
		t.Fatalf("SOCKS5 client connect failed, rep: %d", connectResp[1])
	}

	// Send message and expect target reply
	_, _ = clientConn.Write([]byte("DATA_PING"))
	replyBuf := make([]byte, 256)
	n, err := clientConn.Read(replyBuf)
	if err != nil {
		t.Fatalf("read reply: %v", err)
	}

	got := string(replyBuf[:n])
	if got != "TARGET_REPLY:DATA_PING" {
		t.Errorf("got %q, want TARGET_REPLY:DATA_PING", got)
	}
}
