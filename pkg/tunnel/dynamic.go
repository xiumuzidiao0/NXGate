package tunnel

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
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

// primaryStatusConnecting mirrors vpn.StatusConnecting. It is duplicated as a
// literal because pkg/vpn imports pkg/tunnel, so the constant cannot be shared
// without an import cycle.
const primaryStatusConnecting = "connecting"

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
	health              map[*Tunnel]*tunnelHealth
	healthProbe         func(context.Context, string, time.Duration) (bool, error)
	userlandHealthProbe func(context.Context, func(context.Context, string, string) (net.Conn, error), time.Duration) (bool, error)
	loopOnce            sync.Once
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

func (m *DynamicGroupManager) isPrimaryUserlandMode(g *DynamicGroup) bool {
	if g != nil && (g.ConnectMode == "zero_tun" || g.ConnectMode == "sstp") {
		return true
	}
	if g == nil || g.ConnectMode == "" {
		if reporter, ok := m.primaryConnector.(interface{ IsZeroTun() bool }); ok {
			return reporter.IsZeroTun()
		}
	}
	return false
}

func (m *DynamicGroupManager) primaryConnectedAt() time.Time {
	if reporter, ok := m.primaryConnector.(interface{ ConnectedAt() time.Time }); ok {
		return reporter.ConnectedAt()
	}
	return time.Time{}
}

func (m *DynamicGroupManager) populateSystemPrimaryView(cp *DynamicGroup) {
	if !(cp.IsSystem || cp.ID == SystemPrimaryGroupID) || len(cp.ActiveTunnelIDs) > 0 {
		return
	}
	if m.pool != nil {
		for _, t := range m.pool.ListTunnels() {
			if t.DevIndex == 0 && t.IsHealthy() {
				cp.ActiveTunnelIDs = []string{t.ID}
				if t.Node != nil {
					modeTag := t.DevName
					if t.IsUserland {
						modeTag = "免TUN"
					}
					cp.StatusText = fmt.Sprintf("主网关正常运行: %s (%s, %s)", t.Node.IP, t.Node.CountryShort, modeTag)
				}
				return
			}
		}
	}
	if m.isPrimaryUserlandMode(cp) && m.primaryConnector != nil && m.primaryConnector.GetStatus() == "connected" {
		if activeNode := m.primaryConnector.GetActiveNode(); activeNode != nil {
			cp.ActiveTunnelIDs = []string{PrimaryUserlandTunnelID}
			cp.StatusText = fmt.Sprintf("主网关正常运行: %s (%s, 免TUN)", activeNode.IP, activeNode.CountryShort)
		}
	}
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
		m.populateSystemPrimaryView(&cp)
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
	m.populateSystemPrimaryView(&cp)
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

// EvictTunnelIDs removes stopped or blacklisted tunnel IDs from every dynamic group's active list
// and returns the enabled group IDs whose membership changed so callers can enqueue re-evaluation.
func (m *DynamicGroupManager) EvictTunnelIDs(tunnelIDs []string) []string {
	if len(tunnelIDs) == 0 {
		return nil
	}
	evictSet := make(map[string]bool, len(tunnelIDs))
	for _, tid := range tunnelIDs {
		if tid != "" {
			evictSet[tid] = true
		}
	}
	if len(evictSet) == 0 {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var affected []string
	changed := false
	for _, g := range m.groups {
		if g == nil || len(g.ActiveTunnelIDs) == 0 {
			continue
		}
		kept := make([]string, 0, len(g.ActiveTunnelIDs))
		groupChanged := false
		for _, tid := range g.ActiveTunnelIDs {
			if evictSet[tid] {
				groupChanged = true
				continue
			}
			kept = append(kept, tid)
		}
		if groupChanged {
			g.ActiveTunnelIDs = kept
			changed = true
			if g.Enabled {
				affected = append(affected, g.ID)
			}
		}
	}
	if changed {
		_ = m.saveLocked()
	}
	return affected
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
