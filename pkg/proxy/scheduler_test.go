package proxy

import (
	"testing"

	"aimili-vpngate-go/pkg/nodes"
	"aimili-vpngate-go/pkg/tunnel"
)

func TestScheduler(t *testing.T) {
	node1 := &nodes.Node{ID: "node-1", IP: "1.1.1.1", Ping: 80}
	node2 := &nodes.Node{ID: "node-2", IP: "2.2.2.2", Ping: 30}

	tun1 := &tunnel.Tunnel{ID: "tun-1", DevName: "tun0", Node: node1, Status: tunnel.StatusConnected, LatencyMs: 80}
	tun2 := &tunnel.Tunnel{ID: "tun-2", DevName: "tun1", Node: node2, Status: tunnel.StatusConnected, LatencyMs: 30}

	scheduler := &DefaultScheduler{
		pool: nil,
	}

	// 1. Round-Robin test
	t1 := scheduler.selectFromHealthy([]*tunnel.Tunnel{tun1, tun2}, PolicyRoundRobin, 0)
	t2 := scheduler.selectFromHealthy([]*tunnel.Tunnel{tun1, tun2}, PolicyRoundRobin, 0)
	if t1.ID == t2.ID {
		t.Fatalf("expected round robin to alternate between tun-1 and tun-2, got %s then %s", t1.ID, t2.ID)
	}

	// 2. Random test
	tr := scheduler.selectFromHealthy([]*tunnel.Tunnel{tun1, tun2}, PolicyRandom, 0)
	if tr != tun1 && tr != tun2 {
		t.Fatalf("expected random to pick tun1 or tun2")
	}

	// 3. Least-RTT test (tun2 has 30ms latency, tun1 has 80ms)
	tlr := scheduler.selectFromHealthy([]*tunnel.Tunnel{tun1, tun2}, PolicyLeastRTT, 0)
	if tlr.ID != "tun-2" {
		t.Fatalf("expected Least-RTT to pick lowest latency tunnel tun-2 (30ms), got %s (%d ms)", tlr.ID, tlr.LatencyMs)
	}
}
