package tunnel

import (
	"context"
	"strings"
	"testing"
	"time"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/nodes"
)

type mockPrimaryConnector struct {
	connectedNode *nodes.Node
	status        string
}

func (m *mockPrimaryConnector) Connect(target *nodes.Node) error {
	m.connectedNode = target
	m.status = "connected"
	return nil
}

func (m *mockPrimaryConnector) GetActiveNode() *nodes.Node {
	return m.connectedNode
}

func (m *mockPrimaryConnector) GetStatus() string {
	return m.status
}

func TestUnlockFilterMatching(t *testing.T) {
	aiOnly := &nodes.UnlockResult{
		OpenAI:   nodes.StatusUnlocked,
		Claude:   nodes.StatusUnlocked,
		Gemini:   nodes.StatusUnlocked,
		Netflix:  nodes.StatusBlocked,
		Google:   nodes.StatusBlocked,
		IsProbed: true,
	}

	claudeBlocked := &nodes.UnlockResult{
		OpenAI:   nodes.StatusUnlocked,
		Claude:   nodes.StatusBlocked,
		Gemini:   nodes.StatusUnlocked,
		Netflix:  nodes.StatusBlocked,
		Google:   nodes.StatusBlocked,
		IsProbed: true,
	}

	streamOnly := &nodes.UnlockResult{
		OpenAI:   nodes.StatusBlocked,
		Claude:   nodes.StatusBlocked,
		Gemini:   nodes.StatusBlocked,
		Netflix:  nodes.StatusUnlocked,
		Google:   nodes.StatusUnlocked,
		IsProbed: true,
	}

	fullUnlock := &nodes.UnlockResult{
		OpenAI:   nodes.StatusUnlocked,
		Claude:   nodes.StatusUnlocked,
		Gemini:   nodes.StatusUnlocked,
		Netflix:  nodes.StatusUnlocked,
		Google:   nodes.StatusUnlocked,
		IsProbed: true,
	}

	// 1. None filter matches everything
	if !aiOnly.MatchFilter("none") || !streamOnly.MatchFilter("none") {
		t.Fatalf("none filter should match all")
	}

	// 2. AI filter (三大 AI 全部解锁)
	if !aiOnly.MatchFilter("ai") {
		t.Fatalf("aiOnly should match ai filter")
	}
	if claudeBlocked.MatchFilter("ai") {
		t.Fatalf("claudeBlocked should NOT match ai filter")
	}
	if streamOnly.MatchFilter("ai") {
		t.Fatalf("streamOnly should not match ai filter")
	}

	// 3. Streaming filter
	if !streamOnly.MatchFilter("streaming") {
		t.Fatalf("streamOnly should match streaming filter")
	}
	if aiOnly.MatchFilter("streaming") {
		t.Fatalf("aiOnly should not match streaming filter")
	}

	// 4. Full unlock filter
	if !fullUnlock.MatchFilter("full") {
		t.Fatalf("fullUnlock should match full filter")
	}
	if aiOnly.MatchFilter("full") || streamOnly.MatchFilter("full") || claudeBlocked.MatchFilter("full") {
		t.Fatalf("partial unlock should not match full filter")
	}
}

func TestSystemPrimaryGroupEvaluation(t *testing.T) {
	cfg := &config.Config{
		DataDir: t.TempDir(),
	}
	pool := NewPool(cfg, nil)
	np := nodes.NewNodePool(cfg)
	mgr := NewDynamicGroupManager(cfg, pool, np)

	mockPC := &mockPrimaryConnector{}
	mgr.SetPrimaryConnector(mockPC)

	// Inject candidate nodes
	np.SetCandidatesForTest([]*nodes.Node{
		{
			ID:           "node-us-ai",
			IP:           "1.1.1.1",
			CountryShort: "US",
			Score:        1000,
			Unlock: &nodes.UnlockResult{
				OpenAI: nodes.StatusUnlocked,
			},
		},
		{
			ID:           "node-jp-stream",
			IP:           "2.2.2.2",
			CountryShort: "JP",
			Score:        2000,
			Unlock: &nodes.UnlockResult{
				Netflix: nodes.StatusUnlocked,
			},
		},
	})

	sysGroup := mgr.GetGroup(SystemPrimaryGroupID)
	if sysGroup == nil || !sysGroup.IsSystem {
		t.Fatalf("expected system-primary group to exist and be system")
	}

	// Set criteria: Japan only
	sysGroup.Country = "JP"
	sysGroup.UnlockFilter = "streaming"
	_ = mgr.SaveGroup(sysGroup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mgr.EvaluateGroup(ctx, sysGroup)

	if mockPC.connectedNode == nil {
		t.Fatalf("expected primary connector to be called with JP streaming node")
	}
	if mockPC.connectedNode.ID != "node-jp-stream" {
		t.Fatalf("expected node-jp-stream, got %s", mockPC.connectedNode.ID)
	}
}

func TestFavoritesDynamicGroupEvaluation(t *testing.T) {
	cfg := &config.Config{
		DataDir: t.TempDir(),
	}
	pool := NewPool(cfg, nil)
	np := nodes.NewNodePool(cfg)
	mgr := NewDynamicGroupManager(cfg, pool, np)

	// Inject candidate nodes
	np.SetCandidatesForTest([]*nodes.Node{
		{
			ID:           "node-fast-unfavorite",
			IP:           "1.1.1.1",
			CountryShort: "JP",
			Score:        9999,
			LatencyMs:    10,
		},
		{
			ID:           "node-favorite-1",
			IP:           "2.2.2.2",
			CountryShort: "US",
			Score:        5000,
			LatencyMs:    50,
		},
	})

	// Add node-favorite-1 to favorites
	np.Favorites().Add("node-favorite-1")

	favGroup := &DynamicGroup{
		ID:              "dg-fav",
		Name:            "我的收藏组",
		Country:         "FAVORITES",
		TargetCount:     1,
		IntervalMinutes: 15,
		SortBy:          "latency",
	}
	_ = mgr.SaveGroup(favGroup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mgr.EvaluateGroup(ctx, favGroup)

	// Should only pick favorite node even if unfavorite has higher score / lower latency
	if favGroup.StatusText == "无匹配候选节点" || favGroup.StatusText == "无匹配收藏节点 (请在节点列表中添加收藏)" {
		t.Fatalf("expected to match favorite node, got %s", favGroup.StatusText)
	}
}

func TestCascadingFallbackAndConnectivityFilter(t *testing.T) {
	// 1. Test 204 / Connectivity Unlock filter
	probedGood := &nodes.UnlockResult{
		Google:           nodes.StatusUnlocked,
		Cloudflare:       nodes.StatusUnlocked,
		ThroughputPassed: true,
		IsProbed:         true,
	}
	if !probedGood.MatchFilter("204") || !probedGood.MatchFilter("connectivity") {
		t.Errorf("expected probedGood to pass 204 & connectivity filter")
	}

	probedDeadThroughput := &nodes.UnlockResult{
		Google:           nodes.StatusUnlocked,
		Cloudflare:       nodes.StatusUnlocked,
		ThroughputPassed: false,
		IsProbed:         true,
	}
	if probedDeadThroughput.MatchFilter("204") || probedDeadThroughput.MatchFilter("connectivity") {
		t.Errorf("expected dead throughput node to fail 204 & connectivity filter")
	}

	// 2. Test Dynamic Group Cascading Fallback
	cfg := &config.Config{
		DataDir: t.TempDir(),
	}

	np := nodes.NewNodePool(cfg)
	favNode := &nodes.Node{
		ID:           "node-fav-us",
		IP:           "198.51.100.5",
		CountryShort: "US",
		Ping:         40,
		Score:        800,
	}
	jpNode := &nodes.Node{
		ID:           "node-jp",
		IP:           "203.0.113.1",
		CountryShort: "JP",
		Ping:         60,
		Score:        600,
	}
	np.SetCandidatesForTest([]*nodes.Node{favNode, jpNode})
	np.Favorites().Add("node-fav-us")

	pool := NewPool(cfg, np)
	mgr := NewDynamicGroupManager(cfg, pool, np)

	// Group targeting KR with fallback to favorites
	fallbackGroup := &DynamicGroup{
		ID:              "dg-kr",
		Name:            "韩国出口组",
		Enabled:         true,
		Country:         "KR",
		TargetCount:     1,
		IntervalMinutes: 15,
		FallbackPolicy:  "favorites",
	}
	_ = mgr.SaveGroup(fallbackGroup)

	// Evaluate: KR has 0 nodes, should trigger fallback to favorites
	mgr.EvaluateGroup(context.Background(), fallbackGroup)

	g := mgr.GetGroup("dg-kr")
	if !g.InFallback {
		t.Fatalf("expected group to be in fallback mode when KR has 0 candidates")
	}
	if !strings.Contains(g.FallbackReason, "我的收藏组") {
		t.Errorf("expected fallback reason to mention 我的收藏组, got: %s", g.FallbackReason)
	}

	// Now add a healthy KR candidate -> evaluation should automatically failback to primary!
	krNode := &nodes.Node{
		ID:           "node-kr",
		IP:           "210.100.1.1",
		CountryShort: "KR",
		Ping:         45,
		Score:        900,
	}
	np.SetCandidatesForTest([]*nodes.Node{favNode, jpNode, krNode})

	mgr.EvaluateGroup(context.Background(), fallbackGroup)

	gRecovered := mgr.GetGroup("dg-kr")
	if gRecovered.InFallback {
		t.Fatalf("expected group to recover and NOT be in fallback after KR node became available")
	}
	if gRecovered.FallbackReason != "" {
		t.Errorf("expected empty fallback reason after recovery, got: %s", gRecovered.FallbackReason)
	}
}
