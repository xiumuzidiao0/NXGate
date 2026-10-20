package proxy

import (
	"aimili-vpngate-go/pkg/config"
	"path/filepath"
	"sync"
	"testing"
	"time"

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

func TestSchedulerSystemPrimaryFallback(t *testing.T) {
	if !isOnlySystemPrimary(nil, []string{"system-primary"}) {
		t.Fatal("expected isOnlySystemPrimary to be true for [system-primary]")
	}
	if isOnlySystemPrimary([]string{"tun-1"}, []string{"system-primary"}) {
		t.Fatal("expected isOnlySystemPrimary to be false when boundIDs is non-empty")
	}
	if isOnlySystemPrimary(nil, []string{"system-primary", "dg-123"}) {
		t.Fatal("expected isOnlySystemPrimary to be false when multiple groups bound")
	}
	if isOnlySystemPrimary(nil, []string{"dg-123"}) {
		t.Fatal("expected isOnlySystemPrimary to be false for non-primary group")
	}
}

func TestOptimizationIntervalIdentity(t *testing.T) {
	now := time.Unix(100, 0)
	s := &DefaultScheduler{now: func() time.Time { return now }}
	a, b, c := &tunnel.Tunnel{ID: "a"}, &tunnel.Tunnel{ID: "b"}, &tunnel.Tunnel{ID: "c"}
	choose := func(ts []*tunnel.Tunnel) *tunnel.Tunnel { return s.selectForPort(8000, "rule", ts, PolicyInterval, 10) }
	first := choose([]*tunnel.Tunnel{b, a})
	if first.ID != "a" {
		t.Fatal(first)
	}
	for i := 0; i < 100; i++ {
		if choose([]*tunnel.Tunnel{c, b, a}).ID != "a" {
			t.Fatal("changed within period")
		}
	}
	now = now.Add(10 * time.Second)
	if choose([]*tunnel.Tunnel{b, a}).ID != "b" {
		t.Fatal("did not rotate")
	}
	if choose([]*tunnel.Tunnel{a}).ID != "a" {
		t.Fatal("failed tunnel retained")
	}
	if choose(nil) != nil {
		t.Fatal("must fail closed")
	}
	s.ResetPort(8000)
	if len(s.states) != 0 {
		t.Fatal("rule retained")
	}
}
func TestOptimizationRoundRobinPorts(t *testing.T) {
	s := &DefaultScheduler{}
	ts := []*tunnel.Tunnel{{ID: "b"}, {ID: "a"}}
	if s.selectForPort(1, "rule", ts, PolicyRoundRobin, 0).ID != "a" {
		t.Fatal("first")
	}
	if s.selectForPort(2, "rule", ts, PolicyRoundRobin, 0).ID != "a" {
		t.Fatal("ports coupled")
	}
	if s.selectForPort(1, "rule", ts, PolicyRoundRobin, 0).ID != "b" {
		t.Fatal("not advanced")
	}
	if s.selectForPort(1, "changed", ts, PolicyRoundRobin, 0).ID != "a" {
		t.Fatal("rule did not reset")
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				s.selectForPort(2, "rule", ts, PolicyInterval, 5)
			}
		}()
	}
	wg.Wait()
}

func TestOptimizationRuleStateLifecycle(t *testing.T) {
	scheduler := &DefaultScheduler{}
	manager := &MultiPortManager{cfg: &config.Config{}, scheduler: scheduler, rulesPath: filepath.Join(t.TempDir(), "rules.json"), listeners: make(map[int]*PortListener)}
	rule := PortRule{Port: 8000, Enabled: false, Policy: PolicyInterval}
	if err := manager.ApplyRules([]PortRule{rule}); err != nil {
		t.Fatal(err)
	}
	tunnels := []*tunnel.Tunnel{{ID: "a"}, {ID: "b"}}
	scheduler.selectForPort(8000, "fixture", tunnels, PolicyRoundRobin, 0)
	if err := manager.ApplyRules([]PortRule{rule, {Port: 8001, Enabled: false}}); err != nil {
		t.Fatal(err)
	}
	if got := scheduler.selectForPort(8000, "fixture", tunnels, PolicyRoundRobin, 0); got.ID != "b" {
		t.Fatal("unchanged port state reset")
	}
	if err := manager.ApplyRules(nil); err != nil {
		t.Fatal(err)
	}
	if len(scheduler.states) != 0 {
		t.Fatal("removed rule state retained")
	}
}
