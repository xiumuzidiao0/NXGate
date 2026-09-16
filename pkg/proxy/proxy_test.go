package proxy

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"aimili-vpngate-go/pkg/config"
)

func TestAuthenticator(t *testing.T) {
	auth := NewAuthenticator("user1", "pass123")
	if !auth.IsEnabled() {
		t.Fatal("expected auth to be enabled")
	}
	if !auth.Verify("user1", "pass123") {
		t.Fatal("expected credentials to verify successfully")
	}
	if auth.Verify("user1", "wrongpass") {
		t.Fatal("expected wrong password to fail")
	}

	noAuth := NewAuthenticator("", "")
	if noAuth.IsEnabled() {
		t.Fatal("expected auth to be disabled")
	}
	if !noAuth.Verify("any", "any") {
		t.Fatal("expected any credentials to pass when disabled")
	}
}

func TestCleanPrivacyHeaders(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.195")
	req.Header.Set("Via", "1.1 proxy.example.com")
	req.Header.Set("CF-Connecting-IP", "203.0.113.195")
	req.Header.Set("Proxy-Connection", "keep-alive")
	req.Header.Set("User-Agent", "TestBrowser/1.0")

	cleanPrivacyHeaders(req)

	if req.Header.Get("X-Forwarded-For") != "" {
		t.Errorf("expected X-Forwarded-For to be deleted")
	}
	if req.Header.Get("Via") != "" {
		t.Errorf("expected Via to be deleted")
	}
	if req.Header.Get("CF-Connecting-IP") != "" {
		t.Errorf("expected CF-Connecting-IP to be deleted")
	}
	if req.Header.Get("Proxy-Connection") != "" {
		t.Errorf("expected Proxy-Connection to be deleted")
	}
	if req.Header.Get("User-Agent") != "TestBrowser/1.0" {
		t.Errorf("expected User-Agent to be preserved")
	}
}

func TestGatewaySocks5AndHTTP(t *testing.T) {
	// 1. Setup a dummy target TCP echo server
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen echo server: %v", err)
	}
	defer echoLn.Close()

	go func() {
		for {
			c, err := echoLn.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}(c)
		}
	}()

	echoPort := echoLn.Addr().(*net.TCPAddr).Port

	// 2. Setup Gateway on random port
	cfg := &config.Config{
		ProxyHost:           "127.0.0.1",
		ProxyPort:           0, // will pick dynamic port
		ProxyMaxConnections: 64,
	}

	// Listen dynamically
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen gateway: %v", err)
	}
	gatewayPort := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	cfg.ProxyPort = gatewayPort

	gw := NewGateway(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = gw.Start(ctx)
	}()
	time.Sleep(100 * time.Millisecond)

	// Test 3: SOCKS5 Client request
	t.Run("SOCKS5 Protocol", func(t *testing.T) {
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", gatewayPort))
		if err != nil {
			t.Fatalf("failed to dial gateway: %v", err)
		}
		defer conn.Close()

		// Handshake: VER 0x05, NMETHODS 1, METHOD 0x00 (No auth)
		_, _ = conn.Write([]byte{0x05, 0x01, 0x00})
		resp := make([]byte, 2)
		if _, err := io.ReadFull(conn, resp); err != nil || resp[0] != 0x05 || resp[1] != 0x00 {
			t.Fatalf("invalid socks5 handshake reply: %v", resp)
		}

		// Connect command: VER 0x05, CMD 0x01, RSV 0x00, ATYP 0x01 (IPv4 127.0.0.1), PORT echoPort
		portBytes := []byte{byte(echoPort >> 8), byte(echoPort & 0xff)}
		req := append([]byte{0x05, 0x01, 0x00, 0x01, 127, 0, 0, 1}, portBytes...)
		if _, err := conn.Write(req); err != nil {
			t.Fatalf("failed to send socks5 connect: %v", err)
		}

		connectReply := make([]byte, 10)
		if _, err := io.ReadFull(conn, connectReply); err != nil || connectReply[1] != 0x00 {
			t.Fatalf("failed socks5 connect reply: %v", connectReply)
		}

		// Send payload through proxy to echo server
		testMsg := "hello-socks5-aimili"
		_, _ = conn.Write([]byte(testMsg))
		buf := make([]byte, len(testMsg))
		if _, err := io.ReadFull(conn, buf); err != nil || string(buf) != testMsg {
			t.Fatalf("echo verification failed, got: %s", string(buf))
		}
	})

	// Test 4: HTTP CONNECT Protocol
	t.Run("HTTP CONNECT Protocol", func(t *testing.T) {
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", gatewayPort))
		if err != nil {
			t.Fatalf("failed to dial gateway: %v", err)
		}
		defer conn.Close()

		connectReq := fmt.Sprintf("CONNECT 127.0.0.1:%d HTTP/1.1\r\nHost: 127.0.0.1:%d\r\n\r\n", echoPort, echoPort)
		_, _ = conn.Write([]byte(connectReq))

		br := bufio.NewReader(conn)
		resp, err := http.ReadResponse(br, nil)
		if err != nil || resp.StatusCode != 200 {
			t.Fatalf("invalid http connect response: %v, status: %v", err, resp)
		}

		testMsg := "hello-http-connect-aimili"
		_, _ = conn.Write([]byte(testMsg))
		buf := make([]byte, len(testMsg))
		if _, err := io.ReadFull(conn, buf); err != nil || string(buf) != testMsg {
			t.Fatalf("echo verification failed over http connect, got: %s", string(buf))
		}
	})

	// Test 5: SOCKS5 UDP Associate Protocol
	t.Run("SOCKS5 UDP Associate Protocol", func(t *testing.T) {
		// Setup dummy target UDP echo server
		udpEcho, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
		if err != nil {
			t.Fatalf("failed to listen udp echo server: %v", err)
		}
		defer udpEcho.Close()
		udpEchoPort := udpEcho.LocalAddr().(*net.UDPAddr).Port

		go func() {
			buf := make([]byte, 2048)
			for {
				n, from, err := udpEcho.ReadFrom(buf)
				if err != nil {
					return
				}
				_, _ = udpEcho.WriteTo(buf[:n], from)
			}
		}()

		// Dial SOCKS5 TCP control connection
		tcpConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", gatewayPort))
		if err != nil {
			t.Fatalf("failed to dial gateway for udp associate: %v", err)
		}
		defer tcpConn.Close()

		// 1. Handshake
		_, _ = tcpConn.Write([]byte{0x05, 0x01, 0x00})
		resp := make([]byte, 2)
		if _, err := io.ReadFull(tcpConn, resp); err != nil || resp[0] != 0x05 || resp[1] != 0x00 {
			t.Fatalf("socks5 handshake failed: %v", resp)
		}

		// 2. UDP ASSOCIATE command: VER 0x05, CMD 0x03, RSV 0x00, ATYP 0x01, ADDR 0.0.0.0, PORT 0
		_, _ = tcpConn.Write([]byte{0x05, 0x03, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		assocReply := make([]byte, 10)
		if _, err := io.ReadFull(tcpConn, assocReply); err != nil || assocReply[1] != 0x00 {
			t.Fatalf("socks5 udp associate reply failed: %v", assocReply)
		}

		relayIP := net.IP(assocReply[4:8])
		relayPort := int(binary.BigEndian.Uint16(assocReply[8:10]))
		if relayPort <= 0 {
			t.Fatalf("invalid relay port returned: %d", relayPort)
		}

		// 3. Send encapsulated UDP packet to relay
		clientUDP, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
		if err != nil {
			t.Fatalf("failed to create client udp: %v", err)
		}
		defer clientUDP.Close()

		udpPayload := []byte("ping-udp-socks5-associate")
		// SOCKS5 UDP header: RSV(2) | FRAG(1) | ATYP(1) | IP(4) | PORT(2) | DATA
		reqPacket := make([]byte, 10+len(udpPayload))
		reqPacket[0] = 0x00
		reqPacket[1] = 0x00
		reqPacket[2] = 0x00 // FRAG
		reqPacket[3] = 0x01 // IPv4
		copy(reqPacket[4:8], net.ParseIP("127.0.0.1").To4())
		binary.BigEndian.PutUint16(reqPacket[8:10], uint16(udpEchoPort))
		copy(reqPacket[10:], udpPayload)

		relayAddr := &net.UDPAddr{IP: relayIP, Port: relayPort}
		_, err = clientUDP.WriteTo(reqPacket, relayAddr)
		if err != nil {
			t.Fatalf("failed to write to udp relay: %v", err)
		}

		// 4. Receive echoed response
		_ = clientUDP.SetReadDeadline(time.Now().Add(2 * time.Second))
		recvBuf := make([]byte, 2048)
		n, _, err := clientUDP.ReadFrom(recvBuf)
		if err != nil {
			t.Fatalf("failed to receive echoed udp packet through socks5 relay: %v", err)
		}

		if n < 10 {
			t.Fatalf("response packet too short: %d", n)
		}
		receivedData := recvBuf[10:n]
		if string(receivedData) != string(udpPayload) {
			t.Fatalf("expected payload %q, got %q", string(udpPayload), string(receivedData))
		}
	})
}

func TestPortListenerSocks5WithAuthAndUDP(t *testing.T) {
	// Setup dummy target UDP echo server
	udpEcho, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("failed to listen udp echo server: %v", err)
	}
	defer udpEcho.Close()
	udpEchoPort := udpEcho.LocalAddr().(*net.UDPAddr).Port

	go func() {
		buf := make([]byte, 2048)
		for {
			n, from, err := udpEcho.ReadFrom(buf)
			if err != nil {
				return
			}
			_, _ = udpEcho.WriteTo(buf[:n], from)
		}
	}()

	cfg := &config.Config{
		ProxyHost:           "127.0.0.1",
		ProxyMaxConnections: 64,
	}

	rule := PortRule{
		Port:     0,
		Enabled:  true,
		AuthMode: "custom",
		AuthUser: "secuser",
		AuthPass: "sec%pass@123",
	}

	listener := NewPortListener(rule, cfg, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = listener.Start(ctx)
	}()

	var listenerPort int
	for i := 0; i < 50; i++ {
		listener.mu.Lock()
		if listener.listener != nil {
			listenerPort = listener.listener.Addr().(*net.TCPAddr).Port
			listener.mu.Unlock()
			break
		}
		listener.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	if listenerPort == 0 {
		t.Fatal("listener failed to start")
	}
	defer listener.Close()

	// 1. Connect without auth -> should be rejected
	noAuthConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", listenerPort))
	if err != nil {
		t.Fatalf("dial listener failed: %v", err)
	}
	_, _ = noAuthConn.Write([]byte{0x05, 0x01, 0x00}) // method 0x00
	noAuthResp := make([]byte, 2)
	_, _ = io.ReadFull(noAuthConn, noAuthResp)
	noAuthConn.Close()
	if noAuthResp[1] != 0xff {
		t.Fatalf("expected 0xff (no acceptable methods) for unauthenticated client, got: 0x%02x", noAuthResp[1])
	}

	// 2. Connect with auth and perform UDP Associate
	tcpConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", listenerPort))
	if err != nil {
		t.Fatalf("dial listener failed: %v", err)
	}
	defer tcpConn.Close()

	// Method negotiation: offer 0x00 and 0x02
	_, _ = tcpConn.Write([]byte{0x05, 0x02, 0x00, 0x02})
	methodResp := make([]byte, 2)
	if _, err := io.ReadFull(tcpConn, methodResp); err != nil || methodResp[1] != 0x02 {
		t.Fatalf("expected method 0x02 selected, got: %v", methodResp)
	}

	// Subnegotiation: RFC 1929 username/password
	user := "secuser"
	pass := "sec%pass@123"
	authReq := []byte{0x01, byte(len(user))}
	authReq = append(authReq, []byte(user)...)
	authReq = append(authReq, byte(len(pass)))
	authReq = append(authReq, []byte(pass)...)
	_, _ = tcpConn.Write(authReq)

	authResp := make([]byte, 2)
	if _, err := io.ReadFull(tcpConn, authResp); err != nil || authResp[1] != 0x00 {
		t.Fatalf("expected auth status 0x00 success, got: %v", authResp)
	}

	// UDP ASSOCIATE command: VER 0x05, CMD 0x03, RSV 0x00, ATYP 0x01, ADDR 0.0.0.0, PORT 0
	_, _ = tcpConn.Write([]byte{0x05, 0x03, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	assocReply := make([]byte, 10)
	if _, err := io.ReadFull(tcpConn, assocReply); err != nil || assocReply[1] != 0x00 {
		t.Fatalf("expected udp associate success, got: %v", assocReply)
	}

	relayIP := net.IP(assocReply[4:8])
	relayPort := int(binary.BigEndian.Uint16(assocReply[8:10]))

	// Send UDP packet through relay to echo server
	clientUDP, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("failed to create client udp: %v", err)
	}
	defer clientUDP.Close()

	udpPayload := []byte("hello-socks5-authenticated-udp")
	reqPacket := make([]byte, 10+len(udpPayload))
	reqPacket[0] = 0x00
	reqPacket[1] = 0x00
	reqPacket[2] = 0x00
	reqPacket[3] = 0x01
	copy(reqPacket[4:8], net.ParseIP("127.0.0.1").To4())
	binary.BigEndian.PutUint16(reqPacket[8:10], uint16(udpEchoPort))
	copy(reqPacket[10:], udpPayload)

	_, err = clientUDP.WriteTo(reqPacket, &net.UDPAddr{IP: relayIP, Port: relayPort})
	if err != nil {
		t.Fatalf("failed to write to udp relay: %v", err)
	}

	_ = clientUDP.SetReadDeadline(time.Now().Add(2 * time.Second))
	recvBuf := make([]byte, 2048)
	n, _, err := clientUDP.ReadFrom(recvBuf)
	if err != nil {
		t.Fatalf("failed to receive echoed udp packet through auth relay: %v", err)
	}

	if n < 10 || string(recvBuf[10:n]) != string(udpPayload) {
		t.Fatalf("payload mismatch in auth udp associate test")
	}
}

func TestSocks5NoAuthAcceptsClientUserPassOffer(t *testing.T) {
	// Listener configured in no-auth mode
	cfg := &config.Config{
		ProxyHost:           "127.0.0.1",
		ProxyMaxConnections: 16,
	}
	rule := PortRule{
		Port:     0,
		Enabled:  true,
		AuthMode: "none",
	}

	listener := NewPortListener(rule, cfg, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = listener.Start(ctx)
	}()

	var listenerPort int
	for i := 0; i < 50; i++ {
		listener.mu.Lock()
		if listener.listener != nil {
			listenerPort = listener.listener.Addr().(*net.TCPAddr).Port
			listener.mu.Unlock()
			break
		}
		listener.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	if listenerPort == 0 {
		t.Fatal("listener failed to start")
	}
	defer listener.Close()

	// Client offers ONLY method 0x02 (User/Pass) to an unauthenticated proxy
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", listenerPort))
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	_, _ = conn.Write([]byte{0x05, 0x01, 0x02}) // VER 5, NMETHODS 1, METHOD 2
	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		t.Fatalf("read handshake reply failed: %v", err)
	}
	if resp[1] != 0x02 {
		t.Fatalf("expected server to gracefully select method 0x02, got: 0x%02x", resp[1])
	}

	// Subnegotiation: send dummy user/pass
	authReq := []byte{0x01, 0x04, 'u', 's', 'e', 'r', 0x04, 'p', 'a', 's', 's'}
	_, _ = conn.Write(authReq)
	authResp := make([]byte, 2)
	if _, err := io.ReadFull(conn, authResp); err != nil || authResp[1] != 0x00 {
		t.Fatalf("expected auth success 0x00, got: %v", authResp)
	}
}

func TestPortListenerWithRandomProxyAuth(t *testing.T) {
	cfg := &config.Config{
		ProxyUser: "vpn_random",
		ProxyPass: "pass123456",
		ProxyHost: "127.0.0.1",
	}

	rule := PortRule{
		Port:     0,
		Enabled:  true,
		AuthMode: "random",
	}

	listener := NewPortListener(rule, cfg, nil)
	auth := listener.getAuthenticator()
	if !auth.IsEnabled() {
		t.Fatal("expected authenticator to be enabled under random proxy auth mode")
	}
	if !auth.Verify("vpn_random", "pass123456") {
		t.Fatal("expected proxy credentials to verify successfully")
	}
	if auth.Verify("vpn_random", "wrongpassword") {
		t.Fatal("expected wrong password to fail")
	}
}
