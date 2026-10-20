package nodes

import (
	"aimili-vpngate-go/pkg/config"
	"context"
	"net"
	"testing"
	"time"
)

func localPort(t *testing.T) (net.Listener, int) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	return l, l.Addr().(*net.TCPAddr).Port
}
func TestPortKnockTCP(t *testing.T) {
	l, p := localPort(t)
	ok, _, err := PortKnockTCP("127.0.0.1", p, time.Second)
	if !ok || err != nil {
		t.Fatal(ok, err)
	}
	l.Close()
	ok, _, err = PortKnockTCP("127.0.0.1", p, time.Second)
	if ok || err == nil {
		t.Fatal("closed TCP accepted")
	}
}
func TestOptimizationUDPUnknown(t *testing.T) {
	udp, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := udp.LocalAddr().(*net.UDPAddr).Port
	udp.Close()
	n := &Node{ID: "udp", IP: "127.0.0.1", Port: port, Proto: "udp", Ping: 42, LatencyMs: 60, FailCount: 3}
	results := BatchPortKnock(context.Background(), []*Node{n}, time.Second)
	if results[0].Reachable || !results[0].Unknown {
		t.Fatal(results)
	}
	pool := NewNodePool(&config.Config{DataDir: t.TempDir()})
	pool.MergeFreshNodesLocked([]*Node{n}, "fixture")
	pool.ProbeNodes(context.Background(), []*Node{n})
	if n.LatencyMs != 0 || n.FailCount != 3 || n.ProbeStatus != "unknown" || n.Ping != 42 {
		t.Fatalf("fabricated result: %+v", n)
	}
	owned := *n
	if got := FilterReachableNodes([]*Node{&owned}, time.Second); len(got) != 1 || owned.FailCount != 3 {
		t.Fatal("unknown UDP discarded or failure erased")
	}
	pool.RecordSuccess(n.ID)
	pool.ProbeNodes(context.Background(), []*Node{n})
	if n.ProbeStatus != "connected" || n.LastConnected.IsZero() || n.LatencyMs != 0 {
		t.Fatal("lost connection evidence")
	}
}
func TestBatchPortKnock(t *testing.T) {
	_, p := localPort(t)
	// A retained open listener and a deliberately closed listener provide deterministic results.
	closed, dead := localPort(t)
	closed.Close()
	ns := []*Node{{IP: "127.0.0.1", Port: p, Proto: "tcp"}, {IP: "127.0.0.1", Port: dead, Proto: "tcp"}}
	rs := BatchPortKnock(context.Background(), ns, time.Second)
	if len(rs) != 2 || !rs[0].Reachable || rs[1].Reachable {
		t.Fatal(rs)
	}
	got := FilterReachableNodes(ns, time.Second)
	if len(got) != 1 || got[0] != ns[0] || ns[0].LatencyMs < 1 || ns[1].LatencyMs != -1 {
		t.Fatal(got)
	}
}

func TestOptimizationActualConnectionSupersedesProbeFailure(t *testing.T) {
	pool := NewNodePool(&config.Config{DataDir: t.TempDir()})
	n := &Node{ID: "actual", Proto: "udp", LatencyMs: -1, ProbeStatus: "unreachable", FailCount: 3}
	pool.MergeFreshNodesLocked([]*Node{n}, "fixture")
	pool.RecordSuccess(n.ID)
	if n.LatencyMs != 0 || n.ProbeStatus != "connected" || n.LastConnected.IsZero() || n.FailCount != 0 {
		t.Fatalf("lost successful connection: %+v", n)
	}
	n.LastSeen = time.Now().Add(-48 * time.Hour)
	if pool.evictStaleNodesLocked(time.Now()) != 0 {
		t.Fatal("recent successful UDP candidate evicted")
	}
	pool.saveStoreLocked()
	reloaded := NewNodePool(&config.Config{DataDir: pool.cfg.DataDir}).GetNodeByID(n.ID)
	if reloaded == nil || reloaded.ProbeStatus != "connected" || reloaded.LatencyMs != 0 {
		t.Fatal("stored connection evidence lost")
	}
}
