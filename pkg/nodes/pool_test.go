package nodes

import (
	"fmt"
	"testing"
	"time"

	"aimili-vpngate-go/pkg/config"
)

func TestIncrementalNodePoolAndEviction(t *testing.T) {
	dataDir := t.TempDir()
	cfg := &config.Config{
		DataDir: dataDir,
	}

	np := NewNodePool(cfg)

	// 1. Initial State: empty store
	if len(np.GetCandidates()) != 0 {
		t.Fatalf("expected empty candidates initially")
	}

	// 2. Simulate First Pull: Node A and Node B
	now := time.Now()
	nodeA := &Node{
		ID:           "1.1.1.1:443",
		IP:           "1.1.1.1",
		Port:         443,
		CountryShort: "JP",
		Score:        1000,
		Speed:        50000000,
		FirstSeen:    now,
		LastSeen:     now,
	}
	nodeB := &Node{
		ID:           "2.2.2.2:443",
		IP:           "2.2.2.2",
		Port:         443,
		CountryShort: "US",
		Score:        2000,
		Speed:        60000000,
		FirstSeen:    now,
		LastSeen:     now,
	}

	np.MergeFreshNodesLocked([]*Node{nodeA, nodeB}, "source-1")
	if len(np.GetCandidates()) != 2 {
		t.Fatalf("expected 2 candidates after first merge, got %d", len(np.GetCandidates()))
	}

	// 3. Simulate Second Pull: Node B (updated score) and Node C (new node).
	// Node A is NOT in the second pull, but MUST NOT be overwritten or lost!
	nodeBUpdated := &Node{
		ID:           "2.2.2.2:443",
		IP:           "2.2.2.2",
		Port:         443,
		CountryShort: "US",
		Score:        3000, // Score updated from 2000 to 3000
		Speed:        70000000,
	}
	nodeC := &Node{
		ID:           "3.3.3.3:443",
		IP:           "3.3.3.3",
		Port:         443,
		CountryShort: "KR",
		Score:        1500,
		Speed:        40000000,
	}

	np.MergeFreshNodesLocked([]*Node{nodeBUpdated, nodeC}, "source-2")

	// Verify all 3 nodes exist in pool (Incremental preservation)
	candidates := np.GetCandidates()
	if len(candidates) != 3 {
		t.Fatalf("expected 3 candidates (A preserved, B updated, C added), got %d", len(candidates))
	}

	nodeBStored := np.GetNodeByID("2.2.2.2:443")
	if nodeBStored == nil || nodeBStored.Score != 3000 {
		t.Fatalf("expected Node B to have updated score 3000, got %+v", nodeBStored)
	}

	nodeAStored := np.GetNodeByID("1.1.1.1:443")
	if nodeAStored == nil {
		t.Fatalf("expected Node A to still be preserved in pool even though missing from second pull")
	}

	// 4. Test Eviction of Stale / Dead Nodes
	// Mark Node A as dead and unseen for 48 hours
	np.mu.Lock()
	if n, ok := np.nodeStore["1.1.1.1:443"]; ok {
		n.LastSeen = now.Add(-48 * time.Hour)
		n.LatencyMs = -1
	}
	evicted := np.evictStaleNodesLocked(now)
	np.rebuildCandidatesLocked()
	np.mu.Unlock()

	if evicted != 1 {
		t.Fatalf("expected 1 dead stale node (Node A) to be evicted, got %d", evicted)
	}

	if np.GetNodeByID("1.1.1.1:443") != nil {
		t.Fatalf("expected Node A to be evicted from pool")
	}

	// 5. Test Favorite Protection from Eviction
	// Add dead Node D but mark as favorite
	nodeDFav := &Node{
		ID:           "4.4.4.4:443",
		IP:           "4.4.4.4",
		Port:         443,
		CountryShort: "JP",
		LatencyMs:    -1,
		LastSeen:     now.Add(-72 * time.Hour),
		FailCount:    10,
	}
	np.Favorites().Toggle("4.4.4.4:443") // Mark as favorite
	np.MergeFreshNodesLocked([]*Node{nodeDFav}, "test")

	np.mu.Lock()
	evictedFav := np.evictStaleNodesLocked(now)
	np.mu.Unlock()

	if evictedFav != 0 {
		t.Fatalf("expected favorite Node D to be protected from eviction, got %d evicted", evictedFav)
	}
	if np.GetNodeByID("4.4.4.4:443") == nil {
		t.Fatalf("expected favorite Node D to remain in pool")
	}

	// 6. Test Persistence: reload store from disk
	np2 := NewNodePool(cfg)
	if len(np2.GetCandidates()) == 0 {
		t.Fatalf("expected NodePool to reload saved nodes from disk")
	}
}

func TestHotPoolSizeCappingAndPruning(t *testing.T) {
	dataDir := t.TempDir()
	cfg := &config.Config{
		DataDir: dataDir,
	}

	np := NewNodePool(cfg)
	now := time.Now()

	// 1. Generate 300 nodes (exceeding DefaultNodeHotPoolSize of 250)
	var batch []*Node
	for i := 1; i <= 300; i++ {
		batch = append(batch, &Node{
			ID:           "10.0.0." + string(rune(i)) + ":443",
			IP:           "10.0.0." + string(rune(i)),
			Port:         443,
			CountryShort: "JP",
			Score:        int64(i * 10),
			Speed:        1000000,
			FirstSeen:    now,
			LastSeen:     now,
		})
	}

	np.mu.Lock()
	np.MergeFreshNodesLocked(batch, "test-source")
	candidates := np.candidates
	totalStored := len(np.nodeStore)
	np.mu.Unlock()

	// Verify that candidates (hot pool) is strictly capped at DefaultNodeHotPoolSize (250)
	if len(candidates) != config.DefaultNodeHotPoolSize {
		t.Fatalf("expected hot candidates to be capped at %d, got %d", config.DefaultNodeHotPoolSize, len(candidates))
	}
	// Verify that total store contains all 300 nodes in cold storage
	if totalStored != 300 {
		t.Fatalf("expected total store to have 300 nodes, got %d", totalStored)
	}

	// 2. Test DefaultNodeStoreMaxSize capacity pruning
	// Add 400 more dead nodes to exceed DefaultNodeStoreMaxSize (600)
	var excessBatch []*Node
	for i := 301; i <= 700; i++ {
		excessBatch = append(excessBatch, &Node{
			ID:           "10.0.1." + string(rune(i)) + ":443",
			IP:           "10.0.1." + string(rune(i)),
			Port:         443,
			CountryShort: "US",
			Score:        1,
			Speed:        1000,
			LatencyMs:    -1, // dead
			FailCount:    6,
			FirstSeen:    now.Add(-10 * time.Hour),
			LastSeen:     now.Add(-5 * time.Hour),
		})
	}

	np.mu.Lock()
	np.MergeFreshNodesLocked(excessBatch, "excess-source")
	evicted := np.evictStaleNodesLocked(now)
	np.rebuildCandidatesLocked()
	newStoreCount := len(np.nodeStore)
	np.mu.Unlock()

	if newStoreCount > config.DefaultNodeStoreMaxSize {
		t.Fatalf("expected store count <= %d after capacity pruning, got %d (evicted: %d)", config.DefaultNodeStoreMaxSize, newStoreCount, evicted)
	}
}

func TestNodePoolConfigurableLimitsBalanceAndColdExposure(t *testing.T) {
	now := time.Now()
	makeNode := func(ip, country string, score int64) *Node {
		return &Node{
			ID:           ip + ":443",
			IP:           ip,
			Port:         443,
			CountryShort: country,
			Score:        score,
			Speed:        10_000_000,
			LatencyMs:    20,
			ProbeStatus:  "reachable",
			FirstSeen:    now,
			LastSeen:     now,
		}
	}

	// 1. Major country cap (JP/KR/US) + per-country guarantee + favorites exemption + cold exposure
	cfg := &config.Config{
		DataDir:             t.TempDir(),
		NodeHotPoolSize:     50,
		NodeStoreMaxSize:    1000,
		NodeMinPerCountry:   2,
		NodeMajorCountryCap: 5,
	}
	np := NewNodePool(cfg)

	var batch []*Node
	for i := 1; i <= 30; i++ {
		batch = append(batch, makeNode(fmt.Sprintf("10.1.0.%d", i), "JP", int64(50000-i)))
	}
	for i := 1; i <= 15; i++ {
		batch = append(batch, makeNode(fmt.Sprintf("10.2.0.%d", i), "US", int64(40000-i)))
	}
	// Star one low-score JP node to verify favorites bypass the major country cap.
	favJP := makeNode("10.1.0.200", "JP", 10)
	batch = append(batch, favJP)
	np.Favorites().Toggle(favJP.ID)

	// Rare countries with lower scores than JP/US should still get guaranteed slots.
	for i := 1; i <= 3; i++ {
		batch = append(batch, makeNode(fmt.Sprintf("10.3.0.%d", i), "CH", int64(1000-i)))
		batch = append(batch, makeNode(fmt.Sprintf("10.4.0.%d", i), "DE", int64(900-i)))
	}

	np.mu.Lock()
	np.MergeFreshNodesLocked(batch, "balanced-source")
	np.mu.Unlock()

	active := np.GetCandidates()
	all := np.GetAllNodes()

	if len(all) != len(batch) {
		t.Fatalf("expected GetAllNodes() to expose all %d non-blacklisted nodes, got %d", len(batch), len(all))
	}

	jpActive, usActive, chActive, deActive := 0, 0, 0, 0
	favInActive := false
	for _, n := range active {
		if !n.InActivePool {
			t.Fatalf("GetCandidates() returned node %s with InActivePool=false", n.ID)
		}
		if n.ID == favJP.ID {
			favInActive = true
		}
		switch n.CountryShort {
		case "JP":
			jpActive++
		case "US":
			usActive++
		case "CH":
			chActive++
		case "DE":
			deActive++
		}
	}
	// Cap applies only to non-favorites (or favorites counted before non-favs); with 1 fav + max 5 non-fav or capped total <= 6
	if !favInActive {
		t.Fatalf("expected favorited JP node %s to remain in active pool", favJP.ID)
	}
	if jpActive > 6 {
		t.Fatalf("expected JP active count capped at <=6 (cap 5 + optional fav), got %d", jpActive)
	}
	if usActive > 5 {
		t.Fatalf("expected US active count <= 5, got %d", usActive)
	}
	if chActive < 2 || deActive < 2 {
		t.Fatalf("expected guaranteed rare countries CH>=2, DE>=2 in active pool; got CH=%d, DE=%d", chActive, deActive)
	}

	coldCount := 0
	for _, n := range all {
		if !n.InActivePool {
			coldCount++
		}
	}
	if coldCount != len(all)-len(active) || coldCount == 0 {
		t.Fatalf("expected cold nodes=%d (all=%d - active=%d) > 0", coldCount, len(all), len(active))
	}

	// Cold-storage fallback in SelectBestWithFilter should locate a cold-only country (BR).
	coldBR := makeNode("10.5.0.1", "BR", 5)
	np2Cfg := &config.Config{
		DataDir:             t.TempDir(),
		NodeHotPoolSize:     50,
		NodeStoreMaxSize:    600,
		NodeMinPerCountry:   0,
		NodeMajorCountryCap: 0,
	}
	np2 := NewNodePool(np2Cfg)
	var hotOnly []*Node
	for i := 1; i <= 55; i++ {
		hotOnly = append(hotOnly, makeNode(fmt.Sprintf("10.6.0.%d", i), "JP", int64(10000-i)))
	}
	hotOnly = append(hotOnly, coldBR)
	np2.mu.Lock()
	np2.MergeFreshNodesLocked(hotOnly, "cold-fallback")
	np2.mu.Unlock()
	if picked := np2.SelectBestWithFilter("all", []string{"BR"}, false); picked == nil || picked.ID != coldBR.ID {
		t.Fatalf("expected SelectBestWithFilter to recover cold BR node %s, got %+v", coldBR.ID, picked)
	}

	// 2. Unlimited hot pool and unlimited store size (-1)
	cfgUnlim := &config.Config{
		DataDir:             t.TempDir(),
		NodeHotPoolSize:     -1,
		NodeStoreMaxSize:    -1,
		NodeMinPerCountry:   0,
		NodeMajorCountryCap: 0,
	}
	npUnlim := NewNodePool(cfgUnlim)
	var largeBatch []*Node
	for i := 1; i <= 700; i++ {
		n := makeNode(fmt.Sprintf("10.7.%d.%d", i/250, i%250+1), "NL", int64(i))
		n.LatencyMs = -1
		n.FailCount = 6
		n.LastSeen = now.Add(-10 * time.Hour)
		largeBatch = append(largeBatch, n)
	}
	npUnlim.mu.Lock()
	npUnlim.MergeFreshNodesLocked(largeBatch, "unlimited-source")
	npUnlim.mu.Unlock()
	if got := len(npUnlim.GetCandidates()); got != 700 {
		t.Fatalf("expected unlimited hot pool & store to retain all 700 nodes, got %d", got)
	}
}
