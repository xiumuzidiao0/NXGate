package nodes

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestBlacklistRevival(t *testing.T) {
	dataDir := t.TempDir()
	bm := NewBlacklistManager(dataDir)

	// 1. Start a local dummy TCP listener to simulate a revived node
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	_, pStr, _ := net.SplitHostPort(ln.Addr().String())
	livePort, _ := strconv.Atoi(pStr)

	// 2. Mark the live node as blacklisted (failCount=2, so 3rd failure enters quarantine)
	liveNode := &Node{
		ID:           "live-node-1",
		IP:           "127.0.0.1",
		Port:         livePort,
		Proto:        "tcp",
		CountryShort: "JP",
		FailCount:    2,
	}
	bm.Mark(liveNode, "测试暂时故障", 15*time.Minute)

	// 3. Mark an unreachable dead node (failCount=2, so 3rd failure enters quarantine)
	deadNode := &Node{
		ID:           "dead-node-2",
		IP:           "127.0.0.1",
		Port:         59998, // assuming closed
		Proto:        "tcp",
		CountryShort: "US",
		FailCount:    2,
	}
	bm.Mark(deadNode, "彻底断开", 15*time.Minute)

	if bm.Count() != 2 {
		t.Fatalf("expected 2 blacklisted nodes initially, got %d", bm.Count())
	}

	// 4. Run probe and revive
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	revived, err := bm.ProbeAndRevive(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected probe error: %v", err)
	}

	if len(revived) != 1 || revived[0].ID != "live-node-1" {
		t.Fatalf("expected 1 revived node ('live-node-1'), got %d", len(revived))
	}

	// Verify live node is no longer blacklisted
	if bm.IsBlacklisted("live-node-1") {
		t.Fatalf("expected live-node-1 to be un-blacklisted")
	}

	// Verify dead node remains blacklisted
	if !bm.IsBlacklisted("dead-node-2") {
		t.Fatalf("expected dead-node-2 to remain blacklisted")
	}
}

func TestTieredFailureDegradationAndQuarantine(t *testing.T) {
	dataDir := t.TempDir()
	bm := NewBlacklistManager(dataDir)

	testNode := &Node{
		ID:           "test-node:443",
		IP:           "1.2.3.4",
		Port:         443,
		CountryShort: "JP",
	}

	// 1. First Failure -> Level 1 Degraded (NOT blocked!)
	bm.Mark(testNode, "偶发握手超时", 15*time.Minute)
	if bm.IsBlacklisted(testNode.ID) {
		t.Fatalf("expected node NOT to be blacklisted on first failure (Level 1 Degraded)")
	}
	if bm.Count() != 0 {
		t.Fatalf("expected active blacklist count to be 0 for degraded node, got %d", bm.Count())
	}

	// 2. Second Failure -> Level 1 Degraded (Still NOT blocked!)
	bm.Mark(testNode, "再次超时", 15*time.Minute)
	if bm.IsBlacklisted(testNode.ID) {
		t.Fatalf("expected node NOT to be blacklisted on second failure (Level 1 Degraded)")
	}

	// 3. Third Failure -> Level 2 Quarantine (Blocked for 10 minutes!)
	bm.Mark(testNode, "第三次失败", 15*time.Minute)
	if !bm.IsBlacklisted(testNode.ID) {
		t.Fatalf("expected node to be quarantined on 3rd failure (Level 2 Quarantine)")
	}
	if bm.Count() != 1 {
		t.Fatalf("expected active blacklist count to be 1 for quarantined node, got %d", bm.Count())
	}

	// 4. Reset on success
	bm.Reset(testNode.ID)
	if bm.IsBlacklisted(testNode.ID) {
		t.Fatalf("expected node to be unblocked after Reset on success")
	}

	// 5. Severe Auth Failure -> Immediate Level 3 Hard Blacklist!
	authFailNode := &Node{
		ID:           "bad-auth:443",
		IP:           "5.6.7.8",
		Port:         443,
		CountryShort: "US",
	}
	bm.Mark(authFailNode, "身份认证失败 (AUTH_FAILED)", 15*time.Minute)
	if !bm.IsBlacklisted(authFailNode.ID) {
		t.Fatalf("expected immediate blacklist on AUTH_FAILED even on first failure")
	}
}

func TestIPLevelAndPermanentBlacklist(t *testing.T) {
	dataDir := t.TempDir()
	bm := NewBlacklistManager(dataDir)

	// 1. Test IP-level blocking
	bm.MarkManualWithOptions("198.51.100.1", "198.51.100.1", "JP", "恶意机房IP整机屏蔽", 24*time.Hour, "ip", false)

	// Verify all ports on this IP are blocked
	if !bm.IsNodeBlocked("198.51.100.1:443", "198.51.100.1") {
		t.Fatalf("expected 198.51.100.1:443 to be blocked by IP-level block")
	}
	if !bm.IsNodeBlocked("198.51.100.1:1194", "198.51.100.1") {
		t.Fatalf("expected 198.51.100.1:1194 to be blocked by IP-level block")
	}
	if !bm.IsBlacklisted("198.51.100.1:8080") {
		t.Fatalf("expected 198.51.100.1:8080 to be blocked via IsBlacklisted IP extraction")
	}

	// Verify another IP is NOT blocked
	if bm.IsNodeBlocked("198.51.100.2:443", "198.51.100.2") {
		t.Fatalf("expected 198.51.100.2:443 NOT to be blocked")
	}

	// 2. Test Permanent Tombstone blocking
	bm.MarkManualWithOptions("tombstone-node:443", "198.51.100.99", "US", "用户永久屏蔽", 0, "node", true)
	entry := bm.GetEntry("tombstone-node:443")
	if entry == nil || !entry.IsPermanent {
		t.Fatalf("expected entry to be marked permanent, got %+v", entry)
	}
	if !bm.IsBlacklisted("tombstone-node:443") {
		t.Fatalf("expected permanent entry to be blacklisted")
	}

	// Verify ProbeAndRevive will NOT revive permanent entry even if it's in the list
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	revived, err := bm.ProbeAndRevive(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error during revive: %v", err)
	}
	for _, r := range revived {
		if r.ID == "tombstone-node:443" {
			t.Fatalf("permanent node must NEVER be revived")
		}
	}
}
