package tunnel

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/nodes"
	"aimili-vpngate-go/pkg/stats"
)

const SystemPrimaryGroupID = "system-primary"

type DynamicGroup struct {
	ID              string    `json:"id"`                        // e.g. "dg-1" or "system-primary"
	Name            string    `json:"name"`                      // e.g. "日本Top3住宅隧道组"
	Enabled         bool      `json:"enabled"`                   // 是否启用自动自适应维护
	IsSystem        bool      `json:"is_system"`                 // 是否为系统主出口专用组
	ConnectMode     string    `json:"connect_mode,omitempty"`    // "tun" (内核虚拟网卡模式) or "zero_tun" (用户态免TUN模式)
	Country         string    `json:"country"`                   // "JP", "US", or "" for 全部
	IPType          string    `json:"ip_type"`                   // "residential", "hosting", "all"
	PurityLevel     string    `json:"purity_level,omitempty"`    // "any" (不限), "pure_residential" (仅极度纯净住宅 IP, Risk <= 15), "clean" (纯净/低风险 IP, Risk <= 35), "datacenter" (仅高速机房)
	UnlockFilter    string    `json:"unlock_filter"`             // "none" (不限), "ai" (OpenAI/Claude), "streaming" (Netflix/Google), "all" (全解锁)
	SortBy          string    `json:"sort_by"`                   // "latency" (延迟优先), "speed" (带宽优先), "score" (评分优先)
	TargetCount     int       `json:"target_count"`              // 维持并发隧道数 (主网关固定为 1, 其余组例如 3)
	IntervalMinutes int       `json:"interval_minutes"`          // 重新评估与动态轮换周期 (分钟, 例如 15)
	ActiveTunnelIDs []string  `json:"active_tunnel_ids"`         // 当前此组维护的隧道 ID 列表
	LastEvaluatedAt time.Time `json:"last_evaluated_at"`         // 上次重新评估并轮换的时间
	StatusText      string    `json:"status_text"`               // 状态摘要
	FallbackPolicy  string    `json:"fallback_policy,omitempty"` // "none", "favorites", "auto_low_latency", "auto_speed", 或国家代码如 "JP"
	InFallback      bool      `json:"in_fallback,omitempty"`     // 是否处于兜底级联降级运行状态
	FallbackReason  string    `json:"fallback_reason,omitempty"` // 级联降级原因与说明
}

type PrimaryConnector interface {
	Connect(target *nodes.Node) error
	ConnectWithMode(target *nodes.Node, mode string) error
	GetActiveNode() *nodes.Node
	GetStatus() string
}

type DynamicGroupManager struct {
	cfg              *config.Config
	pool             *Pool
	nodePool         *nodes.NodePool
	filePath         string
	mu               sync.RWMutex
	groups           map[string]*DynamicGroup
	evalMutex        sync.Mutex
	roundMu          sync.Mutex
	health           map[*Tunnel]*tunnelHealth
	healthProbe      func(context.Context, string, time.Duration) (bool, error)
	loopOnce         sync.Once
	queueMu          sync.Mutex
	evalQueue        chan string
	evalPending      map[string]bool
	primaryConnector PrimaryConnector
}

func NewDynamicGroupManager(cfg *config.Config, pool *Pool, np *nodes.NodePool) *DynamicGroupManager {
	m := &DynamicGroupManager{
		cfg:      cfg,
		pool:     pool,
		nodePool: np,
		filePath: filepath.Join(cfg.DataDir, "dynamic_groups.json"),
		groups:   make(map[string]*DynamicGroup),
	}
	m.load()

	// Ensure system primary group exists
	m.mu.Lock()
	if _, ok := m.groups[SystemPrimaryGroupID]; !ok {
		m.groups[SystemPrimaryGroupID] = &DynamicGroup{
			ID:              SystemPrimaryGroupID,
			Name:            "系统主出口网关组 (tun0)",
			Enabled:         true,
			IsSystem:        true,
			Country:         "",
			IPType:          "all",
			UnlockFilter:    "none",
			SortBy:          "score",
			TargetCount:     1,
			IntervalMinutes: 30,
			StatusText:      "维持系统主网关出口 (tun0)",
		}
		m.saveLocked()
	}
	m.mu.Unlock()

	return m
}

func (m *DynamicGroupManager) SetPrimaryConnector(pc PrimaryConnector) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.primaryConnector = pc
}

func (m *DynamicGroupManager) load() {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return
	}

	var list []*DynamicGroup
	if err := json.Unmarshal(data, &list); err == nil {
		for _, g := range list {
			if g.ID != "" {
				// Reset runtime session state on startup: no tunnels exist yet at boot
				g.ActiveTunnelIDs = nil
				g.LastEvaluatedAt = time.Time{}
				m.groups[g.ID] = g
			}
		}
	}
}

func (m *DynamicGroupManager) saveLocked() error {
	var list []*DynamicGroup
	for _, g := range m.groups {
		list = append(list, g)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(m.filePath), ".dynamic-groups-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), m.filePath)
}

func (m *DynamicGroupManager) ListGroups() []*DynamicGroup {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*DynamicGroup
	for _, g := range m.groups {
		cp := *g
		// Clone active IDs slice
		cp.ActiveTunnelIDs = make([]string, len(g.ActiveTunnelIDs))
		copy(cp.ActiveTunnelIDs, g.ActiveTunnelIDs)
		if (cp.IsSystem || cp.ID == SystemPrimaryGroupID) && (cp.ConnectMode == "zero_tun" || cp.ConnectMode == "sstp") {
			if m.primaryConnector != nil && m.primaryConnector.GetStatus() == "connected" {
				activeNode := m.primaryConnector.GetActiveNode()
				if activeNode != nil {
					cp.ActiveTunnelIDs = []string{"primary-userland"}
					cp.StatusText = fmt.Sprintf("主网关正常运行: %s (%s, 免TUN)", activeNode.IP, activeNode.CountryShort)
				}
			}
		}
		list = append(list, &cp)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].IsSystem != list[j].IsSystem {
			return list[i].IsSystem // System primary group always first
		}
		return list[i].ID < list[j].ID
	})
	return list
}

func (m *DynamicGroupManager) GetGroup(id string) *DynamicGroup {
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, ok := m.groups[id]
	if !ok {
		return nil
	}
	cp := *g
	cp.ActiveTunnelIDs = append([]string(nil), g.ActiveTunnelIDs...)
	if (cp.IsSystem || cp.ID == SystemPrimaryGroupID) && (cp.ConnectMode == "zero_tun" || cp.ConnectMode == "sstp") {
		if m.primaryConnector != nil && m.primaryConnector.GetStatus() == "connected" {
			activeNode := m.primaryConnector.GetActiveNode()
			if activeNode != nil {
				cp.ActiveTunnelIDs = []string{"primary-userland"}
				cp.StatusText = fmt.Sprintf("主网关正常运行: %s (%s, 免TUN)", activeNode.IP, activeNode.CountryShort)
			}
		}
	}
	return &cp
}

func (m *DynamicGroupManager) SaveGroup(g *DynamicGroup) error {
	if g == nil {
		return fmt.Errorf("nil dynamic group")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if g.ID == SystemPrimaryGroupID || g.IsSystem {
		g.ID = SystemPrimaryGroupID
		g.IsSystem = true
		g.TargetCount = 1
		if g.Name == "" {
			g.Name = "系统主出口网关组 (tun0)"
		}
	} else {
		if g.ID == "" {
			g.ID = fmt.Sprintf("dg-%d", time.Now().UnixNano()%100000)
		}
		if g.TargetCount <= 0 {
			g.TargetCount = 3
		}
		if g.TargetCount > 16 {
			g.TargetCount = 16
		}
	}

	if g.IntervalMinutes <= 0 {
		if g.IsSystem || g.ID == SystemPrimaryGroupID {
			g.IntervalMinutes = 0
		} else {
			g.IntervalMinutes = 15
		}
	}
	if g.SortBy == "" {
		g.SortBy = "latency"
	}
	if g.IPType == "" {
		g.IPType = "all"
	}
	if g.PurityLevel == "" {
		g.PurityLevel = "any"
	}
	if g.UnlockFilter == "" {
		g.UnlockFilter = "none"
	}
	if g.FallbackPolicy == "" {
		g.FallbackPolicy = "none"
	}

	previous, existed := m.groups[g.ID]
	snapshot := *g
	snapshot.ActiveTunnelIDs = append([]string(nil), g.ActiveTunnelIDs...)
	m.groups[g.ID] = &snapshot
	if err := m.saveLocked(); err != nil {
		if existed {
			m.groups[g.ID] = previous
		} else {
			delete(m.groups, g.ID)
		}
		return fmt.Errorf("save dynamic group: %w", err)
	}

	// 注意：禁止将单个自适应组（如 system-primary）的局部国家偏好反向同步回全局的 DiscoveryCountries。
	// 动态自适应组各司其职，拥有自己独立的国家与策略参数，全局节点池必须面向全球所有国家开放，
	// 否则会导致其他自适应组（如韩国组、美国组）以及节点广场无法获取对应国家的候选节点。

	stats.LogInfo("DynamicGroup", "保存动态自适应组 [%s] (%s): 目标数=%d, 策略=%s, 解锁筛选=%s, 周期=%d分钟",
		g.ID, g.Name, g.TargetCount, g.SortBy, g.UnlockFilter, g.IntervalMinutes)
	return nil
}

func (m *DynamicGroupManager) DeleteGroup(id string) {
	if id == SystemPrimaryGroupID {
		stats.LogWarn("DynamicGroup", "系统主出口网关组受系统保护，禁止删除！")
		return
	}

	m.mu.Lock()
	g, ok := m.groups[id]
	if ok {
		delete(m.groups, id)
		m.saveLocked()
	}
	m.mu.Unlock()

	if g != nil {
		// Stop any tunnels specifically owned by this deleted group
		for _, tunID := range g.ActiveTunnelIDs {
			_ = m.pool.StopTunnel(tunID)
		}
		stats.LogInfo("DynamicGroup", "已删除动态自适应组 [%s] 并释放其维护的出口隧道", id)
	}
}

// GetTunnelsForGroups returns all active healthy tunnels belonging to the specified dynamic groups
func (m *DynamicGroupManager) GetTunnelsForGroups(groupIDs []string) []*Tunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tunIDMap := make(map[string]bool)
	for _, gid := range groupIDs {
		if g, ok := m.groups[gid]; ok && g.Enabled {
			if g.IsSystem || gid == SystemPrimaryGroupID {
				// System primary group always reliably maps to the active primary tun0 tunnel
				for _, t := range m.pool.ListTunnels() {
					if t.DevIndex == 0 {
						tunIDMap[t.ID] = true
						break
					}
				}
			}
			for _, tid := range g.ActiveTunnelIDs {
				tunIDMap[tid] = true
			}
		}
	}

	var available []*Tunnel
	var healthy []*Tunnel
	for tid := range tunIDMap {
		if t := m.pool.GetTunnel(tid); t != nil {
			if t.IsAvailable() {
				available = append(available, t)
			}
			if t.IsHealthy() {
				healthy = append(healthy, t)
			}
		}
	}
	if len(available) > 0 {
		return available
	}
	return healthy
}

// filterCandidatesForGroup evaluates and filters candidates matching specific country, IP type, purity level and unlock filter
func (m *DynamicGroupManager) filterCandidatesForGroup(allCandidates []*nodes.Node, country, ipType, purityLevel, unlockFilter string) []*nodes.Node {
	var matched []*nodes.Node
	targetCountry := strings.ToUpper(strings.TrimSpace(country))
	isFavoritesGroup := targetCountry == "FAVORITES"

	for _, n := range allCandidates {
		if m.nodePool.Blacklist().IsNodeBlocked(n.ID, n.IP) {
			continue
		}
		if isFavoritesGroup {
			if m.nodePool.Favorites() == nil || !m.nodePool.Favorites().IsFavorite(n.ID) {
				continue
			}
		} else if targetCountry != "" && targetCountry != "ALL" && n.CountryShort != targetCountry {
			continue
		}
		if ipType != "" && ipType != "all" {
			if n.IPType != ipType {
				continue
			}
		}
		if purityLevel != "" && purityLevel != "any" {
			switch purityLevel {
			case "pure_residential":
				// Highly pure residential: is_residential && (risk_score <= 15 or pure)
				if !n.IsResidential || (n.RiskScore > 15 && n.RiskLevel != nodes.RiskLevelPure) {
					continue
				}
			case "clean":
				// Clean / low risk: risk_score <= 35 or clean/pure
				if n.RiskScore > 35 && n.RiskLevel != nodes.RiskLevelPure && n.RiskLevel != nodes.RiskLevelClean {
					continue
				}
			case "datacenter":
				// High speed datacenter: is_hosting or ip_type hosting/datacenter
				if !n.IsHosting && n.IPType != "hosting" && n.IPType != "datacenter" {
					continue
				}
			}
		}
		if unlockFilter != "" && unlockFilter != "none" {
			unlock := n.Unlock
			if unlock == nil && m.pool.UnlockDetector() != nil {
				unlock = m.pool.UnlockDetector().EvaluateNodeUnlock(n)
			}
			if !unlock.MatchFilter(unlockFilter) {
				continue
			}
		}
		matched = append(matched, n)
	}
	return matched
}

// EvaluateGroup evaluates candidates, probes metrics, and dynamically rotates tunnels for a group
func (m *DynamicGroupManager) EvaluateGroup(ctx context.Context, g *DynamicGroup) {
	if g == nil {
		return
	}
	for !m.evalMutex.TryLock() {
		if !waitContext(ctx, 10*time.Millisecond) {
			return
		}
	}
	defer m.evalMutex.Unlock()
	if ctx.Err() != nil {
		return
	}
	// Evaluate a private snapshot, never an unlocked pointer into the manager.
	m.mu.RLock()
	original := m.groups[g.ID]
	if original == nil || !original.Enabled {
		m.mu.RUnlock()
		return
	}
	copyGroup := *original
	copyGroup.ActiveTunnelIDs = append([]string(nil), original.ActiveTunnelIDs...)
	m.mu.RUnlock()
	g = &copyGroup
	if m.nodePool == nil {
		return
	}

	allCandidates := m.nodePool.GetCandidates()
	if len(allCandidates) == 0 {
		return
	}

	// 1. Filter candidates by country & IP type
	targetCountry := strings.ToUpper(strings.TrimSpace(g.Country))
	isFavoritesGroup := targetCountry == "FAVORITES"

	matched := m.filterCandidatesForGroup(allCandidates, g.Country, g.IPType, g.PurityLevel, g.UnlockFilter)

	inFallback := false
	fallbackReason := ""

	// 1.5 Cascading Fallback: if primary match yields 0 candidates and group has a fallback policy
	if len(matched) == 0 && g.FallbackPolicy != "" && g.FallbackPolicy != "none" {
		switch strings.ToLower(g.FallbackPolicy) {
		case "favorites":
			fb := m.filterCandidatesForGroup(allCandidates, "FAVORITES", g.IPType, g.PurityLevel, g.UnlockFilter)
			if len(fb) == 0 {
				fb = m.filterCandidatesForGroup(allCandidates, "FAVORITES", "all", "any", "none")
			}
			if len(fb) > 0 {
				matched = fb
				inFallback = true
				fallbackReason = fmt.Sprintf("主目标 [%s] 候选枯竭，已自动级联降级至 [我的收藏组]", g.Country)
			}
		case "auto_low_latency", "latency":
			fb := m.filterCandidatesForGroup(allCandidates, "ALL", g.IPType, g.PurityLevel, g.UnlockFilter)
			if len(fb) == 0 {
				fb = m.filterCandidatesForGroup(allCandidates, "ALL", "all", "any", "none")
			}
			if len(fb) > 0 {
				matched = fb
				inFallback = true
				fallbackReason = fmt.Sprintf("主目标 [%s] 候选枯竭，已自动级联降级至 [全球低延迟组]", g.Country)
			}
		case "auto_speed", "speed":
			fb := m.filterCandidatesForGroup(allCandidates, "ALL", g.IPType, g.PurityLevel, g.UnlockFilter)
			if len(fb) == 0 {
				fb = m.filterCandidatesForGroup(allCandidates, "ALL", "all", "any", "none")
			}
			if len(fb) > 0 {
				matched = fb
				inFallback = true
				fallbackReason = fmt.Sprintf("主目标 [%s] 候选枯竭，已自动级联降级至 [全球高带宽组]", g.Country)
			}
		default:
			cUpper := strings.ToUpper(g.FallbackPolicy)
			fb := m.filterCandidatesForGroup(allCandidates, cUpper, g.IPType, g.PurityLevel, g.UnlockFilter)
			if len(fb) == 0 {
				fb = m.filterCandidatesForGroup(allCandidates, cUpper, "all", "any", "none")
			}
			if len(fb) > 0 {
				matched = fb
				inFallback = true
				fallbackReason = fmt.Sprintf("主目标 [%s] 候选枯竭，已自动级联降级至 [%s 备用组]", g.Country, cUpper)
			}
		}
	}

	g.InFallback = inFallback
	g.FallbackReason = fallbackReason
	m.mu.Lock()
	if stored, ok := m.groups[g.ID]; ok && stored == original {
		stored.InFallback = inFallback
		stored.FallbackReason = fallbackReason
	}
	m.mu.Unlock()

	if len(matched) == 0 {
		m.mu.Lock()
		if isFavoritesGroup {
			g.StatusText = "无匹配收藏节点 (请在节点列表中添加收藏)"
		} else {
			g.StatusText = "无匹配候选节点"
		}
		if stored, ok := m.groups[g.ID]; ok && stored == original {
			stored.StatusText = g.StatusText
		}
		m.mu.Unlock()
		return
	}

	// 2. Probe top candidates to get fresh latency if sorting by latency or if unprobed
	probeLimit := 20
	if len(matched) < probeLimit {
		probeLimit = len(matched)
	}
	toProbe := matched[:probeLimit]
	m.nodePool.ProbeNodes(ctx, toProbe)

	// 3. Sort candidates according to g.SortBy
	sort.Slice(matched, func(i, j int) bool {
		a := matched[i]
		b := matched[j]

		// Preserve actual connection evidence without treating unknown UDP as a measured RTT.
		aAvail := a.LatencyMs > 0 || a.ProbeStatus == "connected"
		bAvail := b.LatencyMs > 0 || b.ProbeStatus == "connected"
		if aAvail != bAvail {
			return aAvail
		}

		// If group requires unlock filter, prioritize nodes that have been physically PROBED and verified
		if g.UnlockFilter != "" && g.UnlockFilter != "none" {
			aProbedOk := a.Unlock != nil && a.Unlock.IsProbed && a.Unlock.MatchFilter(g.UnlockFilter)
			bProbedOk := b.Unlock != nil && b.Unlock.IsProbed && b.Unlock.MatchFilter(g.UnlockFilter)
			if aProbedOk != bProbedOk {
				return aProbedOk // Proven verified unlocked nodes come first
			}
		}

		switch g.SortBy {
		case "latency":
			// Keep a strict ordering: measured RTTs, then connected-without-RTT.
			// Mixing pairwise RTT and score comparisons creates ordering cycles.
			if (a.LatencyMs > 0) != (b.LatencyMs > 0) {
				return a.LatencyMs > 0
			}
			if a.LatencyMs > 0 && b.LatencyMs > 0 {
				if a.LatencyMs != b.LatencyMs {
					return a.LatencyMs < b.LatencyMs
				}
			}
			return a.Score > b.Score

		case "speed":
			if a.Speed != b.Speed {
				return a.Speed > b.Speed
			}
			return a.Score > b.Score

		case "score":
			fallthrough
		default:
			if a.Score != b.Score {
				return a.Score > b.Score
			}
			if (a.LatencyMs > 0) != (b.LatencyMs > 0) {
				return a.LatencyMs > 0
			}
			if a.LatencyMs > 0 && b.LatencyMs > 0 {
				if a.LatencyMs != b.LatencyMs {
					return a.LatencyMs < b.LatencyMs
				}
			}
			return a.Ping < b.Ping
		}
	})

	// 4. Special handling for System Primary Gateway Group (tun0)
	if g.IsSystem || g.ID == SystemPrimaryGroupID {
		m.evaluatePrimarySystemGroup(ctx, g, matched)
		m.mu.Lock()
		if stored := m.groups[g.ID]; stored == original {
			stored.ActiveTunnelIDs = append([]string(nil), g.ActiveTunnelIDs...)
			stored.LastEvaluatedAt = g.LastEvaluatedAt
			stored.StatusText = g.StatusText
			_ = m.saveLocked()
		}
		m.mu.Unlock()
		return
	}

	// 4. Target count to maintain
	targetN := g.TargetCount
	if targetN <= 0 {
		targetN = 3
	}

	// 5. Clean up any stale/dead tunnels first
	m.pool.ReapStaleTunnels()

	m.mu.RLock()
	currentTunnelIDs := make([]string, len(g.ActiveTunnelIDs))
	copy(currentTunnelIDs, g.ActiveTunnelIDs)
	m.mu.RUnlock()

	activeTunnels := make(map[string]*Tunnel)
	for _, tid := range currentTunnelIDs {
		if t := m.pool.GetTunnel(tid); t != nil && t.IsHealthy() {
			activeTunnels[tid] = t
		} else {
			_ = m.pool.StopTunnel(tid)
		}
	}

	// 6. Identify existing healthy tunnels that match our group's criteria
	chosenTunnelIDs := make([]string, 0, targetN)
	usedNodeIDs := make(map[string]bool)

	// Prevent collision with Primary Connection (tun0) and other running tunnels:
	// Mark all nodes already in use outside this group as used so this group never steals or conflicts with them.
	for _, t := range m.pool.ListTunnels() {
		if t.Node != nil {
			if _, isOwn := activeTunnels[t.ID]; !isOwn {
				usedNodeIDs[t.Node.ID] = true
				usedNodeIDs[t.Node.IP] = true
			}
		}
	}

	// Keep existing healthy tunnels whose nodes are in the matched list (ranked best)
	for _, n := range matched {
		if len(chosenTunnelIDs) >= targetN {
			break
		}
		for tid, t := range activeTunnels {
			if t.Node != nil && (t.Node.ID == n.ID || t.Node.IP == n.IP) && t.IsHealthy() && t.IsAvailable() {
				chosenTunnelIDs = append(chosenTunnelIDs, tid)
				usedNodeIDs[n.ID] = true
				usedNodeIDs[n.IP] = true
				delete(activeTunnels, tid)
				break
			}
		}
	}

	var startedIDs []string
	defer func() {
		m.mu.RLock()
		retained := make(map[string]bool)
		for _, stored := range m.groups {
			if stored.Enabled {
				for _, id := range stored.ActiveTunnelIDs {
					retained[id] = true
				}
			}
		}
		m.mu.RUnlock()
		for _, id := range startedIDs {
			if !retained[id] {
				_ = m.pool.StopTunnel(id)
			}
		}
	}()
	// 7. Replenish / backfill any missing slots from the sorted matched list
	needed := targetN - len(chosenTunnelIDs)
	if needed > 0 {
		attemptCount := 0
		for _, n := range matched {
			if ctx.Err() != nil {
				return
			}
			if needed <= 0 || attemptCount >= 3 {
				break
			}
			if usedNodeIDs[n.ID] || usedNodeIDs[n.IP] {
				continue
			}
			if m.nodePool.Blacklist().IsNodeBlocked(n.ID, n.IP) {
				continue
			}
			// Skip candidates that failed TCP connectivity probe
			if n.LatencyMs < 0 {
				continue
			}

			attemptCount++
			stats.LogInfo("DynamicGroup", "[%s] 递补启动新出口 (目标: %d, 仍缺: %d): 节点 %s (%s, 模式: %s, 延迟: %dms, 带宽: %.1fMbps)",
				g.Name, targetN, needed, n.ID, n.CountryShort, g.ConnectMode, n.LatencyMs, float64(n.Speed)/1000000.0)

			var newTun *Tunnel
			var err error
			if g.ConnectMode == "zero_tun" || g.ConnectMode == "sstp" {
				newTun, err = m.pool.StartUserlandTunnel(n)
			} else {
				newTun, err = m.pool.StartTunnel(n)
			}
			if err != nil || newTun == nil {
				stats.LogWarn("DynamicGroup", "[%s] 启动候选节点 %s 失败: %v，尝试下一个候选", g.Name, n.ID, err)
				if err != nil && !strings.Contains(err.Error(), "并发隧道已达上限") && !strings.Contains(err.Error(), "并发出口已达上限") {
					m.nodePool.Blacklist().Mark(n, fmt.Sprintf("启动失败: %v", err), 600*time.Second)
				}
				continue
			}

			startedIDs = append(startedIDs, newTun.ID)
			// Wait for handshake completion or early exit (up to 14s)
			connected := false
			for w := 0; w < 28; w++ {
				if !waitContext(ctx, 500*time.Millisecond) {
					_ = m.pool.StopTunnel(newTun.ID)
					return
				}
				st := newTun.GetStatus()
				if st == StatusConnected {
					connected = true
					break
				}
				if st == StatusFailed || st == StatusStopped {
					break
				}
			}

			if connected {
				// Verify genuine outbound Internet connectivity through this tunnel (skip device route check for userland SSTP)
				connOK := true
				if !newTun.IsUserland {
					if !waitContext(ctx, 300*time.Millisecond) {
						_ = m.pool.StopTunnel(newTun.ID)
						return
					}
					connOK = CheckTunnelConnectivityContext(ctx, newTun.DevName, 3*time.Second)
				}
				if ctx.Err() != nil {
					_ = m.pool.StopTunnel(newTun.ID)
					return
				}
				if connOK {
					// 严苛物理入网闸门：若配置了解锁过滤（如三大 AI 全通），立即通过网卡物理实测
					if g.UnlockFilter != "" && g.UnlockFilter != "none" && m.pool.UnlockDetector() != nil {
						probeCtx, probeCancel := context.WithTimeout(ctx, 8*time.Second)
						unlockRes := m.pool.UnlockDetector().ProbeTunnelInstance(probeCtx, newTun)
						probeCancel()
						newTun.SetUnlock(unlockRes)
						if !unlockRes.MatchFilter(g.UnlockFilter) {
							stats.LogWarn("DynamicGroup", "[%s] 候选节点 %s (%s) 连通但实测未通过解锁要求 (%s: GPT=%s, Claude=%s, Gemini=%s)，释放换线...",
								g.Name, n.ID, newTun.DevName, g.UnlockFilter, unlockRes.OpenAI, unlockRes.Claude, unlockRes.Gemini)
							_ = m.pool.StopTunnel(newTun.ID)
							continue
						}
					}
					chosenTunnelIDs = append(chosenTunnelIDs, newTun.ID)
					usedNodeIDs[n.ID] = true
					usedNodeIDs[n.IP] = true
					if m.nodePool != nil {
						m.nodePool.RecordSuccess(n.ID)
					}
					needed--
				} else {
					stats.LogWarn("DynamicGroup", "[%s] 候选节点 %s 握手成功但无法出网 (被对端阻断)，释放重试...", g.Name, n.ID)
					_ = m.pool.StopTunnel(newTun.ID)
					m.nodePool.Blacklist().Mark(n, "握手成功但无法出网", 600*time.Second)
				}
			} else {
				st := newTun.GetStatus()
				stats.LogWarn("DynamicGroup", "[%s] 候选节点 %s 握手超时或失败 (状态: %s)，释放该隧道并递补下一个候选...", g.Name, n.ID, st)
				_ = m.pool.StopTunnel(newTun.ID)
				if st == StatusFailed {
					m.nodePool.Blacklist().Mark(n, "动态组探测握手未通过", 900*time.Second)
				}
			}
		}
	}

	if ctx.Err() != nil {
		return
	}
	m.mu.RLock()
	unchanged := m.groups[g.ID] == original
	m.mu.RUnlock()
	if !unchanged {
		return
	}

	// 8. Stop any leftover old tunnels that are no longer chosen
	for tid, oldTun := range activeTunnels {
		stats.LogInfo("DynamicGroup", "[%s] 平滑淘汰退役旧出口: %s (%s)", g.Name, tid, oldTun.DevName)
		_ = m.pool.StopTunnel(tid)
	}

	// 9. Update group status directly in manager map
	m.mu.Lock()
	if stored, ok := m.groups[g.ID]; ok && stored == original {
		stored.ActiveTunnelIDs = chosenTunnelIDs
		stored.LastEvaluatedAt = time.Now()
		stored.InFallback = g.InFallback
		stored.FallbackReason = g.FallbackReason
		if len(chosenTunnelIDs) >= stored.TargetCount {
			if stored.InFallback {
				stored.StatusText = fmt.Sprintf("降级运行中 (在网出口: %d/%d - %s)", len(chosenTunnelIDs), stored.TargetCount, stored.FallbackReason)
			} else {
				stored.StatusText = fmt.Sprintf("正常运行 (在网出口: %d/%d)", len(chosenTunnelIDs), stored.TargetCount)
			}
		} else {
			if stored.InFallback {
				stored.StatusText = fmt.Sprintf("降级部分就绪 (在网出口: %d/%d - %s)", len(chosenTunnelIDs), stored.TargetCount, stored.FallbackReason)
			} else {
				stored.StatusText = fmt.Sprintf("部分就绪 (在网出口: %d/%d)", len(chosenTunnelIDs), stored.TargetCount)
			}
		}
		g.ActiveTunnelIDs = stored.ActiveTunnelIDs
		g.LastEvaluatedAt = stored.LastEvaluatedAt
		g.StatusText = stored.StatusText
	}
	m.saveLocked()
	m.mu.Unlock()

	stats.LogInfo("DynamicGroup", "[%s] 动态评估完成，当前活跃出口数量: %d/%d", g.Name, len(chosenTunnelIDs), g.TargetCount)
}

// StartEvaluationLoop starts periodic health checking & evaluation for all dynamic groups
func (m *DynamicGroupManager) StartEvaluationLoop(ctx context.Context) {
	m.loopOnce.Do(func() {
		m.queueMu.Lock()
		m.evalQueue = make(chan string, 64)
		m.evalPending = make(map[string]bool)
		m.queueMu.Unlock()
		go m.evaluationWorker(ctx)
		go func() {
			if !waitContext(ctx, 3*time.Second) {
				return
			}
			m.evaluateAll(ctx)
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					m.evaluateAll(ctx)
				}
			}
		}()
	})
}

// Slow handshakes are separated from the bounded health round. One refill worker
// preserves device-allocation ordering; a bounded deduplicated FIFO avoids spawning
// a goroutine per group or starving later groups on every monitor tick.
func (m *DynamicGroupManager) enqueueEvaluation(id string) {
	m.queueMu.Lock()
	defer m.queueMu.Unlock()
	if m.evalQueue == nil || m.evalPending[id] {
		return
	}
	select {
	case m.evalQueue <- id:
		m.evalPending[id] = true
	default:
	}
}
func (m *DynamicGroupManager) evaluationWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-m.evalQueue:
			if ctx.Err() != nil {
				return
			}
			g := m.GetGroup(id)
			if g != nil && g.Enabled {
				evalCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
				m.EvaluateGroup(evalCtx, g)
				cancel()
			}
			m.queueMu.Lock()
			delete(m.evalPending, id)
			m.queueMu.Unlock()
		}
	}
}

func (m *DynamicGroupManager) evaluateAll(ctx context.Context) {
	if !m.roundMu.TryLock() {
		return
	}
	defer m.roundMu.Unlock()
	roundCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	ctx = roundCtx
	groups := m.ListGroups()
	var checkTunnels []*Tunnel
	for _, g := range groups {
		if g.Enabled {
			for _, id := range g.ActiveTunnelIDs {
				checkTunnels = append(checkTunnels, m.pool.GetTunnel(id))
			}
		}
	}
	healthCtx, healthCancel := context.WithTimeout(ctx, healthRoundBudget)
	health := m.healthBatch(healthCtx, checkTunnels)
	healthCancel()
	var groupIDs []string
	for _, g := range groups {
		if g.Enabled {
			groupIDs = append(groupIDs, g.ID)
		}
	}

	now := time.Now()
	for _, gid := range groupIDs {
		if ctx.Err() != nil {
			return
		}
		g := m.GetGroup(gid)
		if g == nil || !g.Enabled {
			continue
		}

		// If interval is 0 or less, periodic rotation is disabled.
		// Only evaluate if tun0 or group tunnels are down/unreachable or not yet established.
		if g.IntervalMinutes <= 0 {
			needsEval := false
			if g.IsSystem || gid == SystemPrimaryGroupID {
				hasPrimary := false
				if m.primaryConnector != nil {
					hasPrimary = m.primaryConnector.GetStatus() == "connected"
				} else {
					for _, t := range m.pool.ListTunnels() {
						if t.DevIndex == 0 && t.IsHealthy() && t.IsAvailable() {
							hasPrimary = true
							break
						}
					}
				}
				if !hasPrimary {
					needsEval = true
				}
			} else {
				valid := 0
				for _, id := range g.ActiveTunnelIDs {
					t := m.pool.GetTunnel(id)
					if t != nil && t.IsAvailable() {
						if allowed, known := health[t]; !known || allowed {
							valid++
						}
					}
				}
				if valid < g.TargetCount {
					needsEval = true
				}
			}
			if needsEval {
				m.enqueueEvaluation(g.ID)
			}
			continue
		}

		interval := time.Duration(g.IntervalMinutes) * time.Minute

		// Trigger evaluation if interval expired OR if any active tunnel is down/unreachable
		needsEval := false
		if g.LastEvaluatedAt.IsZero() || now.Sub(g.LastEvaluatedAt) >= interval {
			needsEval = true
		} else {
			validCount := 0
			if g.IsSystem || gid == SystemPrimaryGroupID {
				if m.primaryConnector != nil {
					if m.primaryConnector.GetStatus() == "connected" {
						validCount = 1
					}
				} else {
					for _, t := range m.pool.ListTunnels() {
						if t.DevIndex == 0 && t.IsHealthy() && t.IsAvailable() {
							validCount = 1
							break
						}
					}
				}
			} else {
				for _, tid := range g.ActiveTunnelIDs {
					t := m.pool.GetTunnel(tid)
					if t == nil || !t.IsHealthy() || !t.IsAvailable() {
						continue
					}
					if allowed, known := health[t]; known && !allowed {
						continue
					}

					validCount++
				}
			}
			if validCount < g.TargetCount {
				needsEval = true
			}
		}

		if needsEval {
			m.enqueueEvaluation(g.ID)
		}
	}

	// Never reap a tunnel that the separate refill worker is still establishing.
	if !m.evalMutex.TryLock() {
		return
	}
	defer m.evalMutex.Unlock()
	// Automatic garbage collection: clean up any orphan tunnels that are not claimed
	// by any active dynamic group and not the primary connection (tun0)
	m.mu.RLock()
	legitIDs := make(map[string]bool)
	for _, g := range m.groups {
		if g.Enabled {
			for _, tid := range g.ActiveTunnelIDs {
				legitIDs[tid] = true
			}
		}
	}
	m.mu.RUnlock()

	for _, t := range m.pool.ListTunnels() {
		if t.DevIndex > 0 && !legitIDs[t.ID] {
			stats.LogWarn("DynamicGroup", "回收无归属孤儿出口网卡: %s (%s)", t.ID, t.DevName)
			_ = m.pool.StopTunnel(t.ID)
		}
	}
}

func (m *DynamicGroupManager) evaluatePrimarySystemGroup(ctx context.Context, g *DynamicGroup, matched []*nodes.Node) {
	if len(matched) == 0 {
		m.mu.Lock()
		g.StatusText = "无匹配主节点候选"
		m.saveLocked()
		m.mu.Unlock()
		return
	}

	best := matched[0]

	// Find current primary tunnel (devIndex 0)
	var primaryTun *Tunnel
	for _, t := range m.pool.ListTunnels() {
		if t.DevIndex == 0 {
			primaryTun = t
			break
		}
	}

	primaryHealthy := primaryTun != nil && primaryTun.IsHealthy()
	if m.primaryConnector != nil && m.primaryConnector.GetStatus() == "connected" {
		primaryHealthy = true
	}
	var currentActiveNode *nodes.Node
	if m.primaryConnector != nil {
		currentActiveNode = m.primaryConnector.GetActiveNode()
	} else if primaryTun != nil {
		currentActiveNode = primaryTun.Node
	}

	// 严苛物理实测校验：如果配置了解锁要求（如三大 AI 全通），实测当前主网关
	primaryUnlockValid := true
	if g.UnlockFilter != "" && g.UnlockFilter != "none" && primaryTun != nil && currentActiveNode != nil {
		currUnlock := primaryTun.GetUnlock()
		if currUnlock == nil || !currUnlock.IsProbed {
			if m.pool.UnlockDetector() != nil {
				probeCtx, probeCancel := context.WithTimeout(ctx, 8*time.Second)
				currUnlock = m.pool.UnlockDetector().ProbeTunnelInstance(probeCtx, primaryTun)
				probeCancel()
				primaryTun.SetUnlock(currUnlock)
			}
		}
		if currUnlock != nil && !currUnlock.MatchFilter(g.UnlockFilter) {
			primaryUnlockValid = false
			stats.LogWarn("DynamicGroup", "[%s] 当前主网关 %s 实测不满足解锁要求 (%s: GPT=%s, Claude=%s, Gemini=%s)，立即换线寻找支持三大AI的节点...",
				g.Name, currentActiveNode.IP, g.UnlockFilter, currUnlock.OpenAI, currUnlock.Claude, currUnlock.Gemini)
		}
	}

	// 若当前主网关未通过解锁检测，排除当前节点，向后顺延选取下一个候选节点
	if !primaryUnlockValid && currentActiveNode != nil {
		for _, cand := range matched {
			if cand.ID != currentActiveNode.ID && cand.IP != currentActiveNode.IP {
				best = cand
				break
			}
		}
	}

	// If periodic rotation is disabled (IntervalMinutes <= 0), keep the healthy primary connection without rotating ONLY IF it satisfies unlock requirement
	if g.IntervalMinutes <= 0 && primaryHealthy && currentActiveNode != nil && primaryUnlockValid {
		m.mu.Lock()
		if primaryTun != nil {
			g.ActiveTunnelIDs = []string{primaryTun.ID}
		}
		g.LastEvaluatedAt = time.Now()
		if g.ConnectMode == "zero_tun" || g.ConnectMode == "sstp" {
			g.StatusText = fmt.Sprintf("主网关固定连接: %s (%s, 免TUN)", currentActiveNode.IP, currentActiveNode.CountryShort)
		} else {
			g.StatusText = fmt.Sprintf("主网关固定连接: %s (%s, tun0)", currentActiveNode.IP, currentActiveNode.CountryShort)
		}
		m.saveLocked()
		m.mu.Unlock()
		return
	}

	// If primary is already healthy on this best node AND satisfies unlock requirement, keep it
	if primaryHealthy && currentActiveNode != nil && primaryUnlockValid && (currentActiveNode.ID == best.ID || currentActiveNode.IP == best.IP) {
		m.mu.Lock()
		if primaryTun != nil {
			g.ActiveTunnelIDs = []string{primaryTun.ID}
		}
		g.LastEvaluatedAt = time.Now()
		if g.ConnectMode == "zero_tun" || g.ConnectMode == "sstp" {
			g.StatusText = fmt.Sprintf("主网关正常运行: %s (%s, 免TUN)", currentActiveNode.IP, currentActiveNode.CountryShort)
		} else {
			g.StatusText = fmt.Sprintf("主网关正常运行: %s (%s, tun0)", currentActiveNode.IP, currentActiveNode.CountryShort)
		}
		m.saveLocked()
		m.mu.Unlock()
		return
	}

	// If primary connection was established within last 60 seconds and satisfies unlock requirement, don't interrupt it
	if primaryHealthy && primaryUnlockValid {
		if primaryTun != nil && !primaryTun.ConnectedAt.IsZero() && time.Since(primaryTun.ConnectedAt) < 60*time.Second {
			m.mu.Lock()
			g.ActiveTunnelIDs = []string{primaryTun.ID}
			g.LastEvaluatedAt = time.Now()
			g.StatusText = fmt.Sprintf("主网关刚建立: %s (%s, tun0)", primaryTun.Node.IP, primaryTun.Node.CountryShort)
			m.saveLocked()
			m.mu.Unlock()
			return
		} else if (g.ConnectMode == "zero_tun" || g.ConnectMode == "sstp") && currentActiveNode != nil {
			m.mu.Lock()
			g.ActiveTunnelIDs = []string{"primary-userland"}
			g.LastEvaluatedAt = time.Now()
			g.StatusText = fmt.Sprintf("主网关正常运行: %s (%s, 免TUN)", currentActiveNode.IP, currentActiveNode.CountryShort)
			m.saveLocked()
			m.mu.Unlock()
			return
		}
	}

	if ctx.Err() != nil {
		return
	}
	// Trigger primary connection to the best node
	if m.primaryConnector != nil {
		targetMode := "kernel"
		if g.ConnectMode == "zero_tun" || g.ConnectMode == "sstp" {
			targetMode = "userland"
		}
		stats.LogInfo("DynamicGroup", "[%s] 主出口自适应组触发自适应连接 -> 节点: %s (%s, 模式: %s)...",
			g.Name, best.ID, best.CountryShort, targetMode)
		err := m.primaryConnector.ConnectWithMode(best, targetMode)
		m.mu.Lock()
		g.LastEvaluatedAt = time.Now()
		if err != nil {
			g.StatusText = fmt.Sprintf("主网关连接失败: %v", err)
		} else {
			if targetMode == "userland" {
				g.StatusText = fmt.Sprintf("主网关建立中: %s (%s, 免TUN)", best.IP, best.CountryShort)
			} else {
				g.StatusText = fmt.Sprintf("主网关正在建立: %s (%s, tun0)", best.IP, best.CountryShort)
			}
		}
		m.saveLocked()
		m.mu.Unlock()
	} else {
		newTun, err := m.pool.StartPrimaryTunnel(best)
		m.mu.Lock()
		g.LastEvaluatedAt = time.Now()
		if err == nil && newTun != nil {
			g.ActiveTunnelIDs = []string{newTun.ID}
			g.StatusText = fmt.Sprintf("主网关运行中: %s (%s, tun0)", best.IP, best.CountryShort)
		} else {
			g.StatusText = fmt.Sprintf("主网关连接失败: %v", err)
		}
		m.saveLocked()
		m.mu.Unlock()
	}
}
