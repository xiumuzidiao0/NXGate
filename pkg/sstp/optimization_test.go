package sstp

import (
	"testing"
)

func TestOptimizationResolverLifecycle(t *testing.T) {
	tunnel := &Tunnel{closeCh: make(chan struct{})}
	r, e := tunnel.tunnelResolver()
	if e != nil {
		t.Fatal(e)
	}
	same, e := tunnel.tunnelResolver()
	if e != nil || r != same {
		t.Fatal("resolver not reused in tunnel")
	}
	other := &Tunnel{closeCh: make(chan struct{})}
	r2, e := other.tunnelResolver()
	if e != nil || r2 == r {
		t.Fatal("resolver shared across egress")
	}
	if e = tunnel.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = tunnel.tunnelResolver(); e != ErrTunnelClosed {
		t.Fatal("closed tunnel accepted resolver", e)
	}
	if stats := r.Stats(); stats.Entries != 0 || stats.Pending != 0 {
		t.Fatal(stats)
	}
	other.Close()
}

// A non-UDPAddr must never trigger the host resolver before tunnel routing.
type namedUDPAddr string

func (a namedUDPAddr) Network() string { return "udp" }
func (a namedUDPAddr) String() string  { return string(a) }
func TestOptimizationUDPNeverResolvesOnHost(t *testing.T) {
	u := &UDPConn{closed: make(chan struct{})}
	if _, err := u.WriteTo([]byte("local-test"), namedUDPAddr("fixture.invalid:53")); err == nil {
		t.Fatal("accepted unresolved destination")
	}
}
