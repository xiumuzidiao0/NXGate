package proxy

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/tunnel"
)

func TestPortListenerHonorsConfiguredProxyHost(t *testing.T) {
	cfg := &config.Config{
		ProxyHost:           "127.0.0.1",
		ProxyMaxConnections: 16,
	}
	listener := NewPortListener(PortRule{Port: 0, Enabled: true}, cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = listener.Start(ctx)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		listener.mu.Lock()
		bound := listener.listener
		listener.mu.Unlock()
		if bound != nil {
			addr, ok := bound.Addr().(*net.TCPAddr)
			if !ok {
				t.Fatalf("unexpected listener address type %T", bound.Addr())
			}
			if !addr.IP.Equal(net.ParseIP("127.0.0.1")) {
				t.Fatalf("listener ignored ProxyHost and bound to %s", bound.Addr())
			}
			listener.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("listener did not start")
}

func TestPortListenerStartAfterCloseDoesNotBind(t *testing.T) {
	cfg := &config.Config{ProxyHost: "127.0.0.1", ProxyMaxConnections: 8}
	l := NewPortListener(PortRule{Port: 0, Enabled: true}, cfg, nil)
	l.Close()
	if err := l.Start(context.Background()); err != nil {
		t.Fatalf("expected Start on closed listener to return nil, got %v", err)
	}
	l.mu.Lock()
	bound := l.listener
	l.mu.Unlock()
	if bound != nil {
		t.Fatal("expected closed listener not to bind socket")
	}
}

type mockHealthyPool struct {
	tunnels map[string]*tunnel.Tunnel
}

func (m *mockHealthyPool) GetHealthyTunnels(targetIDs []string) []*tunnel.Tunnel {
	var out []*tunnel.Tunnel
	if len(targetIDs) > 0 {
		for _, id := range targetIDs {
			if t := m.tunnels[id]; t != nil && t.IsHealthy() {
				out = append(out, t)
			}
		}
		return out
	}
	for _, t := range m.tunnels {
		if t.IsHealthy() {
			out = append(out, t)
		}
	}
	return out
}

func socks5ConnectOnce(t *testing.T, proxyAddr string) (byte, net.Conn) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", proxyAddr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		t.Fatalf("write socks5 greeting: %v", err)
	}
	method := make([]byte, 2)
	if _, err := io.ReadFull(conn, method); err != nil || method[0] != 0x05 || method[1] != 0x00 {
		t.Fatalf("unexpected socks5 auth reply %v err=%v", method, err)
	}
	req := []byte{0x05, 0x01, 0x00, 0x01, 127, 0, 0, 1, 0x00, 0x50}
	if _, err := conn.Write(req); err != nil {
		t.Fatalf("write socks5 connect: %v", err)
	}
	reply := make([]byte, 10)
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatalf("read socks5 connect reply: %v", err)
	}
	return reply[1], conn
}

func TestPortListenerUserlandFailoverStaysWithinBoundGroup(t *testing.T) {
	cfg := &config.Config{ProxyHost: "127.0.0.1", ProxyMaxConnections: 16}

	// Target echo server for healthy dialer.
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echoLn.Close()
	go func() {
		for {
			c, aerr := echoLn.Accept()
			if aerr != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}(c)
		}
	}()

	var otherDialed, failDialed, okDialed, allFail bool

	tunFail := &tunnel.Tunnel{
		ID:         "tun-a-fail",
		DevName:    "sstp1",
		Status:     tunnel.StatusConnected,
		IsUserland: true,
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			failDialed = true
			return nil, errors.New("stale relay")
		},
	}
	tunOK := &tunnel.Tunnel{
		ID:         "tun-b-ok",
		DevName:    "sstp2",
		Status:     tunnel.StatusConnected,
		IsUserland: true,
		Dialer: func(ctx context.Context, _, _ string) (net.Conn, error) {
			if allFail {
				return nil, errors.New("second relay down too")
			}
			okDialed = true
			return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "tcp", echoLn.Addr().String())
		},
	}
	tunOther := &tunnel.Tunnel{
		ID:         "tun-c-other",
		DevName:    "sstp3",
		Status:     tunnel.StatusConnected,
		IsUserland: true,
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			otherDialed = true
			return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(context.Background(), "tcp", echoLn.Addr().String())
		},
	}

	mPool := &mockHealthyPool{
		tunnels: map[string]*tunnel.Tunnel{
			tunFail.ID:  tunFail,
			tunOK.ID:    tunOK,
			tunOther.ID: tunOther,
		},
	}
	sched := &DefaultScheduler{pool: mPool}
	// Verify SelectTunnelExcluding excludes failed tunnel ID and never escapes to tunOther.
	if got := sched.SelectTunnelExcluding(7928, []string{tunFail.ID, tunOK.ID}, nil, PolicyRoundRobin, 0, map[string]bool{tunFail.ID: true}); got != tunOK {
		t.Fatalf("expected SelectTunnelExcluding to return tunOK, got %v", got)
	}
	if got := sched.SelectTunnelExcluding(7928, []string{tunFail.ID, tunOK.ID}, nil, PolicyRoundRobin, 0, map[string]bool{tunFail.ID: true, tunOK.ID: true}); got != nil {
		t.Fatalf("expected SelectTunnelExcluding not to escape bound tunnels, got %v", got)
	}

	l := NewPortListener(PortRule{
		Port:           0,
		Enabled:        true,
		BoundTunnelIDs: []string{tunFail.ID, tunOK.ID},
		Policy:         PolicyRoundRobin,
		AuthMode:       "none",
	}, cfg, sched)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = l.Start(ctx)
	}()

	var proxyAddr string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		l.mu.Lock()
		bound := l.listener
		l.mu.Unlock()
		if bound != nil {
			proxyAddr = bound.Addr().String()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if proxyAddr == "" {
		t.Fatal("listener did not start")
	}
	defer l.Close()

	// 1. First request picks tun-a-fail, fails over automatically to tun-b-ok, echoes payload.
	rep, conn := socks5ConnectOnce(t, proxyAddr)
	if rep != repSuccess {
		_ = conn.Close()
		t.Fatalf("expected SOCKS5 failover to succeed (0x00), got reply 0x%02x", rep)
	}
	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil || string(buf) != "ping" {
		_ = conn.Close()
		t.Fatalf("expected echoed ping, got %q err=%v", buf, err)
	}
	_ = conn.Close()

	if !failDialed || !okDialed || otherDialed {
		t.Fatalf("unexpected dial flags fail=%v ok=%v other=%v", failDialed, okDialed, otherDialed)
	}

	// 2. When all bound tunnels fail, listener must fail closed (repGeneralFailure) and never touch tunOther.
	allFail = true
	rep2, conn2 := socks5ConnectOnce(t, proxyAddr)
	_ = conn2.Close()
	if rep2 != repGeneralFailure {
		t.Fatalf("expected repGeneralFailure (0x01) when all bound tunnels fail, got 0x%02x", rep2)
	}
	if otherDialed {
		t.Fatal("bound port listener escaped binding to unrelated tunnel")
	}
}
