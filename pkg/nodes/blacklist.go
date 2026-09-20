package nodes

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"aimili-vpngate-go/pkg/stats"
)

type BlacklistEntry struct {
	ID          string    `json:"id"`
	IP          string    `json:"ip"`
	Port        int       `json:"port,omitempty"`
	Protocol    string    `json:"protocol,omitempty"`
	Country     string    `json:"country"`
	Reason      string    `json:"reason"`
	MarkedAt    time.Time `json:"marked_at"`
	Until       time.Time `json:"until"`
	FailCount   int       `json:"fail_count"`
	Level       string    `json:"level,omitempty"`        // "degraded" (降权), "quarantine" (隔离观察), "blacklist" (硬拉黑)
	IsPermanent bool      `json:"is_permanent,omitempty"` // 是否为永久屏蔽 (Tombstone)
	Scope       string    `json:"scope,omitempty"`        // "node" (单节点) 或 "ip" (整机IP屏蔽)
}

type BlacklistManager struct {
	mu       sync.RWMutex
	filePath string
	entries  map[string]*BlacklistEntry
}

func NewBlacklistManager(dataDir string) *BlacklistManager {
	bm := &BlacklistManager{
		filePath: filepath.Join(dataDir, "blacklist.json"),
		entries:  make(map[string]*BlacklistEntry),
	}
	bm.load()
	return bm
}

func (bm *BlacklistManager) load() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	data, err := os.ReadFile(bm.filePath)
	if err != nil {
		return
	}

	var raw map[string]*BlacklistEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}

	now := time.Now()
	cleaned := make(map[string]*BlacklistEntry)
	for k, v := range raw {
		if v != nil && (v.IsPermanent || v.Until.After(now)) {
			cleaned[k] = v
		}
	}
	bm.entries = cleaned
}

func (bm *BlacklistManager) saveLocked() {
	data, err := json.MarshalIndent(bm.entries, "", "  ")
	if err != nil {
		return
	}
	tmpFile := bm.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err == nil {
		_ = os.Chmod(tmpFile, 0600)
		_ = os.Rename(tmpFile, bm.filePath)
	}
}

func (bm *BlacklistManager) IsBlacklisted(nodeID string) bool {
	return bm.IsNodeBlocked(nodeID, "")
}

func (bm *BlacklistManager) IsNodeBlocked(nodeID, ip string) bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	now := time.Now()

	// 1. Direct Node ID match
	if entry, ok := bm.entries[nodeID]; ok {
		if entry.IsPermanent || entry.Until.After(now) {
			return true
		}
	}

	// 2. IP-level match (handles entire IP blocking regardless of port)
	targetIP := ip
	if targetIP == "" && strings.Contains(nodeID, ":") {
		if h, _, err := net.SplitHostPort(nodeID); err == nil {
			targetIP = h
		}
	}

	if targetIP != "" {
		if entry, ok := bm.entries[targetIP]; ok {
			if entry.IsPermanent || entry.Until.After(now) {
				return true
			}
		}
		for _, entry := range bm.entries {
			if entry.Scope == "ip" && entry.IP == targetIP {
				if entry.IsPermanent || entry.Until.After(now) {
					return true
				}
			}
		}
	}

	return false
}

func (bm *BlacklistManager) GetEntry(key string) *BlacklistEntry {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	if entry, ok := bm.entries[key]; ok {
		cp := *entry
		return &cp
	}
	return nil
}

func (bm *BlacklistManager) Mark(node *Node, reason string, baseDuration time.Duration) {
	if node == nil {
		return
	}
	bm.mu.Lock()
	defer bm.mu.Unlock()

	now := time.Now()
	failCount := 1
	if existing, ok := bm.entries[node.ID]; ok {
		failCount = existing.FailCount + 1
	} else if node.FailCount > 0 {
		failCount = node.FailCount + 1
	}
	node.FailCount = failCount

	isAuthOrManual := strings.Contains(reason, "身份认证失败") ||
		strings.Contains(reason, "AUTH_FAILED") ||
		strings.Contains(reason, "用户手动") ||
		strings.Contains(reason, "certificate") ||
		strings.Contains(reason, "expired")

	var level string
	var duration time.Duration

	if isAuthOrManual || failCount >= 5 {
		// Level 3: 深度硬拉黑 (Hard Blacklist) - 连续失败 5 次以上或严重证书/认证拒绝
		level = "blacklist"
		if baseDuration <= 0 {
			baseDuration = 30 * time.Minute
		}
		shift := failCount - 5
		if shift < 0 {
			shift = 0
		}
		if shift > 12 {
			shift = 12
		}
		multiplier := 1 << shift
		duration = baseDuration * time.Duration(multiplier)
		if duration > 6*time.Hour {
			duration = 6 * time.Hour
		}
		stats.LogWarn("Blacklist", "节点 [%s] 严重/持久故障 (失败 %d 次: %s)，进入深度硬拉黑 (%v)", node.ID, failCount, reason, duration)
	} else if failCount >= 3 {
		// Level 2: 隔离观察期 (Quarantine) - 连续失败 3~4 次，临时冷却隔离 10 分钟
		level = "quarantine"
		duration = 10 * time.Minute
		stats.LogWarn("Blacklist", "节点 [%s] 连续失败 %d 次 (%s)，进入 10 分钟临时隔离观察期", node.ID, failCount, reason)
	} else {
		// Level 1: 仅降权扣分 (Degraded) - 偶发失败 1~2 次，不阻止访问，排序沉底
		level = "degraded"
		duration = 0
		stats.LogWarn("Blacklist", "节点 [%s] 偶发失败 (%d/3: %s)，触发降权惩罚，暂不隔离", node.ID, failCount, reason)
	}

	var until time.Time
	if duration > 0 {
		until = now.Add(duration)
	} else {
		until = now // immediate expiry, so IsBlacklisted returns false
	}

	bm.entries[node.ID] = &BlacklistEntry{
		ID:        node.ID,
		IP:        node.IP,
		Port:      node.Port,
		Protocol:  node.Proto,
		Country:   node.CountryShort,
		Reason:    reason,
		MarkedAt:  now,
		Until:     until,
		FailCount: failCount,
		Level:     level,
	}

	bm.saveLocked()
}

func (bm *BlacklistManager) Reset(nodeID string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	delete(bm.entries, nodeID)
	bm.saveLocked()
}

func (bm *BlacklistManager) Remove(nodeID string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	delete(bm.entries, nodeID)
	bm.saveLocked()
}

func (bm *BlacklistManager) Clear() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.entries = make(map[string]*BlacklistEntry)
	bm.saveLocked()
}

func (bm *BlacklistManager) MarkManual(id, ip, country, reason string, duration time.Duration) {
	bm.MarkManualWithOptions(id, ip, country, reason, duration, "node", false)
}

func (bm *BlacklistManager) MarkManualWithOptions(id, ip, country, reason string, duration time.Duration, scope string, permanent bool) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	now := time.Now()
	if reason == "" {
		if permanent {
			reason = "用户手动永久屏蔽 (Tombstone)"
		} else {
			reason = "用户手动屏蔽"
		}
	}

	cleanIP := ip
	port := 0
	if strings.Contains(id, ":") {
		if h, pStr, err := net.SplitHostPort(id); err == nil {
			if cleanIP == "" {
				cleanIP = h
			}
			port, _ = strconv.Atoi(pStr)
		}
	} else if cleanIP == "" {
		cleanIP = id
	} else if strings.Contains(cleanIP, ":") {
		if h, pStr, err := net.SplitHostPort(cleanIP); err == nil {
			cleanIP = h
			port, _ = strconv.Atoi(pStr)
		}
	}

	if scope == "" {
		if port > 0 && id != cleanIP {
			scope = "node"
		} else {
			scope = "ip"
		}
	}

	var until time.Time
	if permanent {
		until = now.Add(100 * 365 * 24 * time.Hour) // 100 years
	} else {
		if duration <= 0 {
			duration = 24 * time.Hour
		}
		until = now.Add(duration)
	}

	key := id
	if scope == "ip" && cleanIP != "" {
		key = cleanIP
	}

	bm.entries[key] = &BlacklistEntry{
		ID:          key,
		IP:          cleanIP,
		Port:        port,
		Country:     country,
		Reason:      reason,
		MarkedAt:    now,
		Until:       until,
		FailCount:   5,
		Level:       "blacklist",
		IsPermanent: permanent,
		Scope:       scope,
	}
	bm.saveLocked()
}

// ProbeAndRevive tests blacklisted nodes and auto-revives those that respond to TCP dial
func (bm *BlacklistManager) ProbeAndRevive(ctx context.Context, fallbackPortFinder func(nodeID, ip string) int) ([]*BlacklistEntry, error) {
	bm.mu.RLock()
	now := time.Now()
	var candidates []*BlacklistEntry
	for _, entry := range bm.entries {
		if entry != nil && !entry.IsPermanent && entry.Until.After(now) {
			cp := *entry
			candidates = append(candidates, &cp)
		}
	}
	bm.mu.RUnlock()

	if len(candidates) == 0 {
		return nil, nil
	}

	var revived []*BlacklistEntry
	var revMu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 16)

	for _, item := range candidates {
		wg.Add(1)
		go func(entry *BlacklistEntry) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			port := entry.Port
			if port <= 0 && fallbackPortFinder != nil {
				port = fallbackPortFinder(entry.ID, entry.IP)
			}
			if port <= 0 {
				port = 443
			}

			addr := net.JoinHostPort(entry.IP, strconv.Itoa(port))
			conn, err := net.DialTimeout("tcp", addr, 3500*time.Millisecond)
			if err == nil {
				_ = conn.Close()
				revMu.Lock()
				revived = append(revived, entry)
				revMu.Unlock()
			}
		}(item)
	}

	wg.Wait()

	if len(revived) > 0 {
		bm.mu.Lock()
		for _, r := range revived {
			delete(bm.entries, r.ID)
		}
		bm.saveLocked()
		bm.mu.Unlock()
	}

	return revived, nil
}

func (bm *BlacklistManager) Count() int {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	now := time.Now()
	count := 0
	for _, v := range bm.entries {
		if v.IsPermanent || v.Until.After(now) {
			count++
		}
	}
	return count
}

func (bm *BlacklistManager) List() []*BlacklistEntry {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	now := time.Now()
	var list []*BlacklistEntry
	for _, v := range bm.entries {
		if v.IsPermanent || v.Until.After(now) {
			list = append(list, v)
		}
	}
	return list
}
