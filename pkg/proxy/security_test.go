package proxy

import (
	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/tunnel"
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type reviewConn struct {
	in     *bytes.Reader
	out    bytes.Buffer
	closed bool
}

func (c *reviewConn) Read(b []byte) (int, error)  { return c.in.Read(b) }
func (c *reviewConn) Write(b []byte) (int, error) { return c.out.Write(b) }
func (c *reviewConn) Close() error                { c.closed = true; return nil }
func (c *reviewConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 7928}
}
func (c *reviewConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 12345}
}
func (c *reviewConn) SetDeadline(time.Time) error      { return nil }
func (c *reviewConn) SetReadDeadline(time.Time) error  { return nil }
func (c *reviewConn) SetWriteDeadline(time.Time) error { return nil }
func TestSecurityNoTunnelFailsClosed(t *testing.T) {
	// Only local target HTTP fixture; no proxy listener or real VPN is started.
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true; fmt.Fprint(w, "HOST-DIRECT-TEST") }))
	defer srv.Close()
	cfg := &config.Config{DataDir: t.TempDir()}
	pool := tunnel.NewPool(cfg, nil)
	l := NewPortListener(PortRule{Port: 7928, AuthMode: "none", BoundTunnelIDs: []string{"missing-bound-tunnel"}}, cfg, NewScheduler(pool, nil))
	c := &reviewConn{in: bytes.NewReader([]byte("GET " + srv.URL + "/ HTTP/1.1\r\nHost: " + strings.TrimPrefix(srv.URL, "http://") + "\r\n\r\n"))}
	l.dispatch(c)
	if hit || !strings.Contains(c.out.String(), "502 Bad Gateway") {
		t.Fatalf("expected fail closed, got %s", c.out.String())
	}
	if !c.closed {
		t.Fatal("connection not closed")
	}

}
func TestSecurityAuthFailureCloses(t *testing.T) {
	l := NewPortListener(PortRule{Port: 7928, AuthMode: "custom", AuthUser: "u", AuthPass: "p"}, &config.Config{}, nil)
	for _, req := range [][]byte{[]byte("GET http://invalid.test/ HTTP/1.1\r\nHost: invalid.test\r\n\r\n"), {5, 1, 0}} {
		c := &reviewConn{in: bytes.NewReader(req)}
		l.dispatch(c)
		if !c.closed {
			t.Fatal("connection not closed")
		}
	}
}
func TestSecurityCustomEmptyCredentials(t *testing.T) {
	cfg := &config.Config{DataDir: t.TempDir()}
	m := NewMultiPortManager(cfg, nil, nil)
	rule := PortRule{Port: 7930, Enabled: false, AuthMode: "custom"}
	if err := m.ApplyRules([]PortRule{rule}); err == nil {
		t.Fatal("accepted empty custom auth")
	}
}
func TestSecurityCustomProxyRace(t *testing.T) {
	m := NewCustomProxyManager(t.TempDir())
	_, err := m.AddOrUpdate("http://127.0.0.1:1", "")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			m.ProbeAll(ctx)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 5000; i++ {
			m.List()
		}
	}()
	wg.Wait()
}

func TestSecuritySnapshots(t *testing.T) {
	m := NewCustomProxyManager(t.TempDir())
	p, err := m.AddOrUpdate("http://127.0.0.1:1", "")
	if err != nil {
		t.Fatal(err)
	}
	p.Healthy = true
	got := m.Get(p.ID)
	if got.Healthy {
		t.Fatal("Add returned live pointer")
	}
	got.Healthy = true
	if m.Get(p.ID).Healthy {
		t.Fatal("Get returned live pointer")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				m.Probe(ctx, p.ID, time.Millisecond)
				m.Get(p.ID)
				m.List()
				m.GetHealthy()
			}
		}()
	}
	wg.Wait()
}
func TestSecurityUDPNoEgress(t *testing.T) {
	if c, err := createBoundUDPSocket(""); err == nil {
		c.Close()
		t.Fatal("host UDP fallback")
	}
	if c, err := dialUpstream("127.0.0.1:1", "", time.Millisecond); err == nil {
		c.Close()
		t.Fatal("host TCP fallback")
	}
	c := &reviewConn{in: bytes.NewReader([]byte{0, 0, 0, 0, 0, 0})}
	if err := handleSocks5UDPAssociate(c, atypIPv4, "", nil, nil); err == nil {
		t.Fatal("no UDP egress accepted")
	}
}
func TestSecurityUDPPeer(t *testing.T) {
	a := &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 1234}
	for _, b := range []*net.UDPAddr{{IP: net.ParseIP("192.0.2.2"), Port: 1234}, {IP: a.IP, Port: 1235}, nil} {
		if sameUDPPeer(a, b) {
			t.Fatal("accepted wrong peer")
		}
	}
	if !sameUDPPeer(a, &net.UDPAddr{IP: a.IP, Port: a.Port}) {
		t.Fatal("rejected same peer")
	}
}

type securityPool struct{ tunnels []*tunnel.Tunnel }

func (p securityPool) GetHealthyTunnels(ids []string) []*tunnel.Tunnel {
	var out []*tunnel.Tunnel
	for _, tun := range p.tunnels {
		allowed := len(ids) == 0
		for _, id := range ids {
			if id == tun.ID {
				allowed = true
			}
		}
		if allowed && tun.IsHealthy() {
			out = append(out, tun)
		}
	}
	return out
}
func TestSecuritySchedulerStrictBindings(t *testing.T) {
	a := &tunnel.Tunnel{ID: "a", Status: tunnel.StatusConnected}
	b := &tunnel.Tunnel{ID: "b", Status: tunnel.StatusConnected}
	s := &DefaultScheduler{pool: securityPool{[]*tunnel.Tunnel{a, b}}}
	if s.SelectTunnel(1, []string{"missing"}, nil, PolicyRandom, 0) != nil {
		t.Fatal("escaped missing static binding")
	}
	if s.SelectTunnel(1, nil, []string{"missing"}, PolicyRandom, 0) != nil {
		t.Fatal("escaped missing group binding")
	}
	if got := s.SelectTunnel(1, []string{"a"}, nil, PolicyRandom, 0); got != a {
		t.Fatal("lost valid binding")
	}
	a.TripCircuitBreaker(time.Minute)
	if s.SelectTunnel(1, []string{"a"}, nil, PolicyRandom, 0) != nil {
		t.Fatal("escaped broken binding or selected circuit broken tunnel")
	}
	if got := s.SelectTunnel(1, nil, nil, PolicyRandom, 0); got != b {
		t.Fatal("unbound did not select available tunnel")
	}
	b.TripCircuitBreaker(time.Minute)
	if s.SelectTunnel(1, nil, nil, PolicyRandom, 0) != nil {
		t.Fatal("selected circuit-broken tunnel")
	}
}
func TestSecurityExplicitDialerAndBinding(t *testing.T) {
	called := false
	d := func(ctx context.Context, network, addr string) (net.Conn, error) {
		called = true
		return nil, fmt.Errorf("test egress")
	}
	cfg := &config.Config{}
	l := NewPortListener(PortRule{Port: 7928, AuthMode: "none"}, cfg, nil)
	l.SetUserlandDialer(d)
	request := []byte("GET http://invalid.test/ HTTP/1.1\r\nHost: invalid.test\r\n\r\n")
	c := &reviewConn{in: bytes.NewReader(request)}
	l.dispatch(c)
	if !called || !c.closed {
		t.Fatal("explicit dialer not used/connection not closed")
	}
	for _, rule := range []PortRule{{BoundTunnelIDs: []string{"missing"}}, {BoundGroupIDs: []string{"missing"}}, {BoundCustomProxyIDs: []string{"missing"}}} {
		rule.AuthMode = "none"
		l = NewPortListener(rule, cfg, nil)
		l.SetUserlandDialer(d)
		called = false
		c = &reviewConn{in: bytes.NewReader(request)}
		l.dispatch(c)
		if called {
			t.Fatal("binding escaped into primary dialer")
		}
	}
}
func TestSecurityInvalidPersistedCustomAuth(t *testing.T) {
	l := NewPortListener(PortRule{AuthMode: "custom"}, &config.Config{}, nil)
	a := l.getAuthenticator()
	if !a.IsEnabled() || a.Verify("", "") {
		t.Fatal("invalid persisted custom credentials permit no-auth")
	}
}

func TestSecurityRulesPersistenceFailure(t *testing.T) {
	cfg := &config.Config{DataDir: t.TempDir()}
	m := NewMultiPortManager(cfg, nil, nil)
	before := m.GetRules()
	os.Remove(m.rulesPath)
	if err := os.Mkdir(m.rulesPath, 0700); err != nil {
		t.Fatal(err)
	}
	if err := m.ApplyRules([]PortRule{{Port: 7931, Enabled: false, AuthMode: "none"}}); err == nil {
		t.Fatal("persistence failure hidden")
	}
	if got := m.GetRules(); len(got) != len(before) || got[0].Port != before[0].Port {
		t.Fatal("failed persistence changed live rules")
	}
}
