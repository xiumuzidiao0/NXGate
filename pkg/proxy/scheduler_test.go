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

func TestSchedulerConnectModeAndPurityFilter(t *testing.T) {
	nodeRes := &nodes.Node{
		ID:            "node-res",
		IP:            "100.1.1.1",
		IsResidential: true,
		RiskScore:     10.0,
		RiskLevel:     nodes.RiskLevelPure,
	}
	nodeCleanDC := &nodes.Node{
		ID:        "node-dc-clean",
		IP:        "100.2.2.2",
		IsHosting: true,
		IPType:    "hosting",
		RiskScore: 25.0,
		RiskLevel: nodes.RiskLevelClean,
	}
	nodeHighRisk := &nodes.Node{
		ID:        "node-dc-risk",
		IP:        "100.3.3.3",
		IsHosting: true,
		IPType:    "hosting",
		RiskScore: 70.0,
		RiskLevel: nodes.RiskLevelHigh,
	}

	tunKernel := &tunnel.Tunnel{
		ID:         "tun-kernel",
		DevName:    "tun0",
		IsUserland: false,
		Node:       nodeRes,
		Status:     tunnel.StatusConnected,
	}
	tunZero := &tunnel.Tunnel{
		ID:         "tun-zero",
		DevName:    "sstp-1234",
		IsUserland: true,
		Node:       nodeCleanDC,
		Status:     tunnel.StatusConnected,
	}
	tunHighRisk := &tunnel.Tunnel{
		ID:         "tun-high",
		DevName:    "tun1",
		IsUserland: false,
		Node:       nodeHighRisk,
		Status:     tunnel.StatusConnected,
	}

	allTunnels := []*tunnel.Tunnel{tunKernel, tunZero, tunHighRisk}

	// 1. ConnectMode Filter
	tunOnly := filterTunnelsByConnectMode(allTunnels, "tun")
	if len(tunOnly) != 2 || tunOnly[0] != tunKernel || tunOnly[1] != tunHighRisk {
		t.Fatalf("expected 2 kernel tunnels, got %d", len(tunOnly))
	}

	zeroOnly := filterTunnelsByConnectMode(allTunnels, "zero_tun")
	if len(zeroOnly) != 1 || zeroOnly[0] != tunZero {
		t.Fatalf("expected 1 zero_tun tunnel, got %d", len(zeroOnly))
	}

	// 2. PurityFilter
	pureRes := filterTunnelsByPurity(allTunnels, "pure_residential")
	if len(pureRes) != 1 || pureRes[0] != tunKernel {
		t.Fatalf("expected 1 pure_residential tunnel, got %d", len(pureRes))
	}

	clean := filterTunnelsByPurity(allTunnels, "clean")
	if len(clean) != 2 {
		t.Fatalf("expected 2 clean/pure tunnels (risk <= 35), got %d", len(clean))
	}

	dcOnly := filterTunnelsByPurity(allTunnels, "datacenter")
	if len(dcOnly) != 2 {
		t.Fatalf("expected 2 datacenter tunnels, got %d", len(dcOnly))
	}

	// 3. StrictMatch Behavior
	// Zero-TUN + pure_residential: None exists among allTunnels
	zeroResFiltered := filterTunnelsByPurity(zeroOnly, "pure_residential")
	if len(zeroResFiltered) != 0 {
		t.Fatalf("expected 0 zero_tun pure_residential tunnels")
	}
}
