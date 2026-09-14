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
