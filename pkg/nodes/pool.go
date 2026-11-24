package nodes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/sstp"
	"aimili-vpngate-go/pkg/stats"
)

const (
	// MaxHotPoolSize defines the strict upper boundary of active hot candidate nodes.
	MaxHotPoolSize = 250

	// MaxStoreNodes defines the storage upper bound for historical/cold nodes to prevent indefinite bloat.
	MaxStoreNodes = 600
)

// ProbeStrategy defines timeout values for each probe attempt
type ProbeStrategy struct {
	TCPTimeout time.Duration
	UDPTimeout time.Duration
}

// GetProbeStrategy returns timeout strategy for a given attempt number
func GetProbeStrategy(attempt int) ProbeStrategy {
	strategies := []ProbeStrategy{
		{TCPTimeout: 6 * time.Second, UDPTimeout: 4 * time.Second}, // Attempt 0: generous
		{TCPTimeout: 4 * time.Second, UDPTimeout: 3 * time.Second}, // Attempt 1: tighter
		{TCPTimeout: 3 * time.Second, UDPTimeout: 2 * time.Second}, // Attempt 2: strict
	}
	if attempt < len(strategies) {
		return strategies[attempt]
	}
	return strategies[len(strategies)-1]
}

// ProbeConfig holds configuration for node probing
type ProbeConfig struct {
	MaxAttempts int
}

func DefaultProbeConfig() ProbeConfig {
	return ProbeConfig{
		MaxAttempts: 3,
	}
}

type NodePool struct {
	cfg        *config.Config
	fetcher    *Fetcher
	snapshot   *SnapshotManager
	blacklist  *BlacklistManager
	enricher   *IPEnricher
	favorites  *FavoritesManager
	reputation *ReputationManager
	storePath  string

	isRefreshing atomic.Bool
	mu           sync.RWMutex
	nodeStore    map[string]*Node // 全量持久化增量节点库 (主键: node.ID)
	candidates   []*Node          // 当前有效、过滤并已排序的优质候选节点列表
	activeIDs    map[string]bool  // 当前活跃池内节点 ID 集合
	allRawNodes  []*Node          // 当前库中全部已知节点清单
	lastUpdated  time.Time
	lastSource   string
	lastStatus   string
}

func NewNodePool(cfg *config.Config) *NodePool {
	sm := NewSnapshotManager(cfg.DataDir)
	fetcher := NewFetcher(cfg.ApiURL, cfg.MirrorURL, sm)
	bm := NewBlacklistManager(cfg.DataDir)
	enricher := NewIPEnricher(cfg.DataDir)
	favorites := NewFavoritesManager(cfg.DataDir)
	reputation := NewReputationManager(cfg.DataDir)

	np := &NodePool{
		cfg:        cfg,
		fetcher:    fetcher,
		snapshot:   sm,
		blacklist:  bm,
		enricher:   enricher,
		favorites:  favorites,
		reputation: reputation,
		storePath:  filepath.Join(cfg.DataDir, "nodes_store.json"),
		nodeStore:  make(map[string]*Node),
		activeIDs:  make(map[string]bool),
		lastStatus: "初始化中",
	}

	np.loadStore()
	return np
}

func isMajorCountry(code string) bool {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "JP", "KR", "US":
		return true
	default:
		return false
	}
}

func (np *NodePool) hotPoolLimit() int {
	if np.cfg != nil {
		return np.cfg.GetNodeHotPoolSize()
	}
	return MaxHotPoolSize
}

func (np *NodePool) storeMaxLimit() int {
	if np.cfg != nil {
		return np.cfg.GetNodeStoreMaxSize()
	}
	return MaxStoreNodes
}

func (np *NodePool) minPerCountry() int {
	if np.cfg != nil {
		return np.cfg.GetNodeMinPerCountry()
	}
	return 0
}

func (np *NodePool) majorCountryCap() int {
	if np.cfg != nil {
		return np.cfg.GetNodeMajorCountryCap()
	}
	return 0
}

func formatPoolLimitLabel(limit int) string {
	if limit < 0 {
		return "不限"
	}
	return strconv.Itoa(limit)
}

func (np *NodePool) nodeTier(node *Node) int {
	if node.LatencyMs > 0 || node.ProbeStatus == "connected" {
		if node.FailCount == 0 {
			return 0 // Tier 0: 完全健康优质节点 (无失败历史)
		}
		return 1 // Tier 1: 可达但有偶发失败记录的降权节点
	}
	if node.LatencyMs == 0 {
		return 2 // Tier 2: 未验证节点，包括尚未握手的 UDP 候选
	}
	return 3 // Tier 3: 探测失败/死端口节点
}

func (np *NodePool) sortNodesByRank(list []*Node) {
	sort.Slice(list, func(i, j int) bool {
		a := list[i]
		b := list[j]

		aFav := np.favorites != nil && np.favorites.IsFavorite(a.ID)
		bFav := np.favorites != nil && np.favorites.IsFavorite(b.ID)
		if aFav != bFav {
			return aFav
		}

		tierA := np.nodeTier(a)
		tierB := np.nodeTier(b)
		if tierA != tierB {
			return tierA < tierB
		}

		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.Ping != b.Ping {
			return a.Ping < b.Ping
		}
		return a.ID < b.ID
	})
}

func (np *NodePool) loadStore() {
	np.mu.Lock()
	defer np.mu.Unlock()

	data, err := os.ReadFile(np.storePath)
	if err != nil {
		return
	}

	var list []*Node
	if err := json.Unmarshal(data, &list); err != nil {
		return
	}

	now := time.Now()
	for _, n := range list {
		if n != nil && n.ID != "" {
			if n.FirstSeen.IsZero() {
				n.FirstSeen = now
			}
			if n.LastSeen.IsZero() {
				n.LastSeen = now
			}
			// Migrate pre-optimization UDP RTTs: neither source Ping nor UDP socket
			// creation established a local RTT. Preserve actual connection history.
			if strings.EqualFold(n.Proto, "udp") {
				n.LatencyMs = 0
				if n.ProbeStatus != "connected" {
					n.ProbeStatus = "unknown"
				}
			}
			// If stored unlock is unprobed prediction or older than 12h, clear it for fresh evaluation
			if n.Unlock != nil && (!n.Unlock.IsProbed || now.Sub(n.Unlock.CheckedAt) > 12*time.Hour) {
				n.Unlock = nil
			}
			np.nodeStore[n.ID] = n
		}
	}

	np.evictStaleNodesLocked(now)
	np.rebuildCandidatesLocked()
	if len(np.candidates) > 0 {
		np.lastStatus = fmt.Sprintf("已恢复历史节点库 (优质热池: %d/%s, 历史总库: %d)", len(np.candidates), formatPoolLimitLabel(np.hotPoolLimit()), len(np.nodeStore))
		np.lastUpdated = now
		np.lastSource = "本地持久化节点库"
	}
}

func (np *NodePool) saveStoreLocked() {
	var list []*Node
	for _, n := range np.nodeStore {
		list = append(list, n)
	}

	data, err := json.Marshal(list)
	if err != nil {
		return
	}

	tmp := np.storePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err == nil {
		_ = os.Chmod(tmp, 0600)
		_ = os.Rename(tmp, np.storePath)
	}
}

// evictStaleNodesLocked cleans up nodes that have been offline or unseen for an extended period.
// Favored nodes are permanently protected from automated eviction.
// Reachable nodes (LatencyMs > 0) are actively preserved in the pool.
func (np *NodePool) evictStaleNodesLocked(now time.Time) int {
	evicted := 0
	for id, n := range np.nodeStore {
		// 1. Never evict user favorites
		if np.favorites.IsFavorite(n.ID) {
			continue
		}

		// 2. Never evict currently reachable nodes
		if n.LatencyMs > 0 || (n.ProbeStatus == "connected" && !n.LastConnected.IsZero() && now.Sub(n.LastConnected) < 24*time.Hour) {
			continue
		}

		// 3. Condition A: Unseen in upstream feeds for >= 24h AND unreachable
		staleByTime := !n.LastSeen.IsZero() && now.Sub(n.LastSeen) >= 24*time.Hour && n.LatencyMs <= 0

		// 4. Condition B: Failed probe >= 5 consecutive times AND unseen in upstream for >= 12h
		staleByFails := n.FailCount >= 5 && !n.LastSeen.IsZero() && now.Sub(n.LastSeen) >= 12*time.Hour

		if staleByTime || staleByFails {
			delete(np.nodeStore, id)
			evicted++
		}
	}

	// 5. Soft limit enforcement: if store exceeds configured storeMaxLimit, prune the worst dead/unreachable nodes
	storeLimit := np.storeMaxLimit()
	if storeLimit > 0 && len(np.nodeStore) > storeLimit {
		type evictCandidate struct {
			id       string
			lastSeen time.Time
			score    int64
			fails    int
			dead     bool
		}
		var pruneList []evictCandidate
		for id, n := range np.nodeStore {
			if np.favorites.IsFavorite(id) || n.LatencyMs > 0 || (n.ProbeStatus == "connected" && !n.LastConnected.IsZero() && now.Sub(n.LastConnected) < 24*time.Hour) {
				continue // protect favorites and reachable nodes
			}
			pruneList = append(pruneList, evictCandidate{
				id:       id,
				lastSeen: n.LastSeen,
				score:    n.Score,
				fails:    n.FailCount,
				dead:     n.LatencyMs < 0,
			})
		}
		// Sort worst first: dead first, then highest fails, oldest lastSeen, lowest score
		sort.Slice(pruneList, func(i, j int) bool {
			if pruneList[i].dead != pruneList[j].dead {
				return pruneList[i].dead
			}
			if pruneList[i].fails != pruneList[j].fails {
				return pruneList[i].fails > pruneList[j].fails
			}
			if !pruneList[i].lastSeen.Equal(pruneList[j].lastSeen) {
				return pruneList[i].lastSeen.Before(pruneList[j].lastSeen)
			}
			return pruneList[i].score < pruneList[j].score
		})

		excess := len(np.nodeStore) - storeLimit
		if excess > len(pruneList) {
			excess = len(pruneList)
		}
		for i := 0; i < excess; i++ {
			delete(np.nodeStore, pruneList[i].id)
			evicted++
		}
	}

	return evicted
}

// rebuildCandidatesLocked filters and sorts the candidates from the internal nodeStore.
func (np *NodePool) rebuildCandidatesLocked() {
	var all []*Node
	var ranked []*Node

	for _, n := range np.nodeStore {
		all = append(all, n)
		if np.blacklist != nil && np.blacklist.IsNodeBlocked(n.ID, n.IP) {
			continue
		}
		ranked = append(ranked, n)
	}

	np.sortNodesByRank(ranked)

	hotLimit := np.hotPoolLimit()
	minPer := np.minPerCountry()
	majorCap := np.majorCountryCap()

	selected := make([]*Node, 0, len(ranked))
	selectedIDs := make(map[string]bool, len(ranked))
	countryCounts := make(map[string]int)

	withinLimit := func() bool {
		return hotLimit < 0 || len(selected) < hotLimit
	}
	countryKey := func(n *Node) string {
		c := strings.ToUpper(strings.TrimSpace(n.CountryShort))
		if c == "" {
			return "ZZ"
		}
		return c
	}

	// Pass 1: Favorites always stay in the active pool and bypass major-country cap.
	for _, n := range ranked {
		if !withinLimit() {
			break
		}
		if np.favorites != nil && np.favorites.IsFavorite(n.ID) {
			selected = append(selected, n)
			selectedIDs[n.ID] = true
			countryCounts[countryKey(n)]++
		}
	}

	// Pass 2: Guarantee at least minPer slots per country (non-major countries first so rare regions aren't crowded out).
	if minPer > 0 && withinLimit() {
		byCountry := make(map[string][]*Node)
		bestRank := make(map[string]int)
		for idx, n := range ranked {
			if selectedIDs[n.ID] {
				continue
			}
			c := countryKey(n)
			if _, ok := bestRank[c]; !ok {
				bestRank[c] = idx
			}
			byCountry[c] = append(byCountry[c], n)
		}

		orderedCountries := make([]string, 0, len(byCountry))
		for c := range byCountry {
			orderedCountries = append(orderedCountries, c)
		}
		sort.Slice(orderedCountries, func(i, j int) bool {
			ci, cj := orderedCountries[i], orderedCountries[j]
			mi, mj := isMajorCountry(ci), isMajorCountry(cj)
			if mi != mj {
				return !mi // non-major countries first
			}
			if bestRank[ci] != bestRank[cj] {
				return bestRank[ci] < bestRank[cj]
			}
			return ci < cj
		})

		cursors := make(map[string]int, len(orderedCountries))
		for round := 0; round < minPer && withinLimit(); round++ {
			addedInRound := false
			for _, c := range orderedCountries {
				if !withinLimit() {
					break
				}
				if countryCounts[c] >= minPer {
					continue
				}
				if majorCap > 0 && isMajorCountry(c) && countryCounts[c] >= majorCap {
					continue
				}
				queue := byCountry[c]
				cur := cursors[c]
				for cur < len(queue) && selectedIDs[queue[cur].ID] {
					cur++
				}
				if cur < len(queue) {
					n := queue[cur]
					cur++
					selected = append(selected, n)
					selectedIDs[n.ID] = true
					countryCounts[c]++
					addedInRound = true
				}
				cursors[c] = cur
			}
			if !addedInRound {
				break
			}
		}
	}

	// Pass 3: Fill remaining hot pool slots in rank order, enforcing JP/KR/US cap when enabled.
	for _, n := range ranked {
		if !withinLimit() {
			break
		}
		if selectedIDs[n.ID] {
			continue
		}
		c := countryKey(n)
		isFav := np.favorites != nil && np.favorites.IsFavorite(n.ID)
		if majorCap > 0 && !isFav && isMajorCountry(c) && countryCounts[c] >= majorCap {
			continue
		}
		selected = append(selected, n)
		selectedIDs[n.ID] = true
		countryCounts[c]++
	}

	np.sortNodesByRank(selected)

	np.activeIDs = make(map[string]bool, len(selected))
	for _, n := range selected {
		np.activeIDs[n.ID] = true
	}
	np.candidates = selected
	np.allRawNodes = all
}

func (np *NodePool) RecordSuccess(nodeID string) {
	np.mu.Lock()
	defer np.mu.Unlock()

	if n, ok := np.nodeStore[nodeID]; ok {
		n.FailCount = 0
		n.LastChecked = time.Now()
		n.LastConnected = n.LastChecked
		if n.LatencyMs < 0 {
			n.LatencyMs = 0
		} // A completed connection supersedes a failed TCP probe.
		n.ProbeStatus = "connected"
	}
	if np.blacklist != nil {
		np.blacklist.Reset(nodeID)
	}
	np.rebuildCandidatesLocked()
}

// MergeFreshNodesLocked merges freshly fetched nodes into the incremental store without wiping historical valid nodes.
func (np *NodePool) MergeFreshNodesLocked(fresh []*Node, source string) (int, int, int) {
	now := time.Now()
	newCount := 0
	updatedCount := 0

	for _, n := range fresh {
		// Permanently blacklisted (Tombstone): completely drop, never add to store or memory!
		if np.blacklist != nil && np.blacklist.IsNodeBlocked(n.ID, n.IP) {
			if entry := np.blacklist.GetEntry(n.ID); entry != nil && entry.IsPermanent {
				continue
			}
			if entry := np.blacklist.GetEntry(n.IP); entry != nil && entry.IsPermanent {
				continue
			}
		}

		if existing, exists := np.nodeStore[n.ID]; exists {
			// Update real-time network metrics from official source
			existing.Score = n.Score
			existing.Ping = n.Ping
			existing.Speed = n.Speed
			existing.NumVpnSessions = n.NumVpnSessions
			existing.TotalUsers = n.TotalUsers
			existing.TotalTraffic = n.TotalTraffic
			existing.Operator = n.Operator
			existing.Message = n.Message
			existing.LastSeen = now
			// A feed appearance is not a successful local connection; preserve FailCount.
			if n.ConfigData != "" {
				existing.ConfigData = n.ConfigData
			}
			updatedCount++
		} else {
			// Brand new node
			n.FirstSeen = now
			n.LastSeen = now
			np.nodeStore[n.ID] = n
			newCount++
		}
	}

	// Perform stale node eviction
	evictedCount := np.evictStaleNodesLocked(now)

	np.rebuildCandidatesLocked()
	np.saveStoreLocked()

	np.lastUpdated = now
	np.lastSource = source
	np.lastStatus = fmt.Sprintf("就绪 (优质热池 %d/%s, 历史总库 %d)", len(np.candidates), formatPoolLimitLabel(np.hotPoolLimit()), len(np.nodeStore))

	return newCount, updatedCount, evictedCount
}

func (np *NodePool) IsRefreshing() bool {
	return np.isRefreshing.Load()
}

func (np *NodePool) Refresh(ctx context.Context) error {
	if !np.isRefreshing.CompareAndSwap(false, true) {
		stats.LogInfo("Nodes", "节点池刷新任务已在后台运行中，跳过重复请求")
		return nil
	}
	defer np.isRefreshing.Store(false)

	np.mu.Lock()
	np.lastStatus = "正在拉取节点列表"
	np.mu.Unlock()

	result, err := np.fetcher.FetchNodes(ctx)
	if err != nil {
		np.mu.Lock()
		np.lastStatus = fmt.Sprintf("拉取失败: %v", err)
		np.mu.Unlock()
		return err
	}

	nodes, err := ParseVPNGateCSV(result.Data, np.cfg.MaxScanRows)
	if err != nil {
		np.mu.Lock()
		np.lastStatus = fmt.Sprintf("解析失败: %v", err)
		np.mu.Unlock()
		return err
	}

	// Cache raw snapshot CSV
	_ = np.snapshot.Save(result.Data, result.Source, len(nodes))

	// Perform incremental merge and stale eviction
	np.mu.Lock()
	newCount, updatedCount, evictedCount := np.MergeFreshNodesLocked(nodes, result.Source)
	currentFiltered := make([]*Node, len(np.candidates))
	for i, n := range np.candidates {
		snapshot := *n
		currentFiltered[i] = &snapshot
	}
	precheckStarted := time.Now()
	np.mu.Unlock()

	stats.LogInfo("Nodes", "节点池增量刷新完成 (新增: %d, 更新: %d, 淘汰失效: %d)，当前全量库: %d 个，优质热池: %d/%s 个 (来自: %s)",
		newCount, updatedCount, evictedCount, len(np.allRawNodes), len(currentFiltered), formatPoolLimitLabel(np.hotPoolLimit()), result.Source)

	// Phase 1: Port Knock Pre-Filter (快速淘汰死端口，节省测速时间)
	stats.LogInfo("Nodes", "🚪 端口预检：快速扫描热池 %d 个候选节点的 TCP 连通性...", len(currentFiltered))
	reachableNodes := FilterReachableNodes(currentFiltered, 2500*time.Millisecond)
	stats.LogInfo("Nodes", "端口预检完成：保留 %d/%d 个候选 (含未验证 UDP)，过滤 %d 个失败端口",
		len(reachableNodes), len(currentFiltered), len(currentFiltered)-len(reachableNodes))

	// The filter owns snapshots; publish under the pool lock. Do not overwrite
	// a real connection result recorded while the precheck was in flight.
	np.mu.Lock()
	for _, result := range currentFiltered {
		if target := np.nodeStore[result.ID]; target != nil && !target.LastChecked.After(precheckStarted) {
			target.LatencyMs = result.LatencyMs
			target.ProbeStatus = result.ProbeStatus
			target.LastChecked = result.LastChecked
			target.FailCount = result.FailCount
		}
	}
	np.rebuildCandidatesLocked()
	currentCandidates := make([]*Node, len(np.candidates))
	copy(currentCandidates, np.candidates)
	np.mu.Unlock()

	// Phase 2: 后台并发测试核心节点的真实延迟
	// 分级按需探测：跳过 Phase 1 刚完成测速的活跃节点，仅补充探测延迟过期或未测速的高优先级节点
	probeLimit := 50
	var highPriorityNodes []*Node
	for _, n := range currentCandidates {
		if !n.LastChecked.IsZero() && time.Since(n.LastChecked) < 2*time.Minute && n.LatencyMs > 0 {
			continue
		}
		if n.LatencyMs > 0 || n.ProbeStatus == "connected" {
			highPriorityNodes = append(highPriorityNodes, n)
			if len(highPriorityNodes) >= probeLimit {
				break
			}
		}
	}
	if len(highPriorityNodes) > 0 {
		go np.ProbeNodes(ctx, highPriorityNodes)
	}

	// Phase 3: Async IP type classification (residential vs hosting) in background
	// 优先对热池中前 60 个活跃候选节点分析住宅/机房属性，已分析过的节点不重复查询
	enrichLimit := 60
	var toEnrich []*Node
	for _, n := range currentCandidates {
		if n.IPType == "" || n.IPType == "unknown" {
			toEnrich = append(toEnrich, n)
			if len(toEnrich) >= enrichLimit {
				break
			}
		}
	}
	if len(toEnrich) > 0 {
		go np.enricher.EnrichNodes(ctx, toEnrich)
	}

	return nil
}

func (np *NodePool) ProbeNodes(ctx context.Context, nodeList []*Node) {
	if len(nodeList) == 0 {
		return
	}

	stats.LogInfo("Probe", "开始对 %d 个候选节点执行全量连通性与延迟测速...", len(nodeList))

	var wg sync.WaitGroup
	concurrency := 24
	sem := make(chan struct{}, concurrency)

	probeConfig := DefaultProbeConfig()

	for _, n := range nodeList {
		wg.Add(1)
		go func(target *Node) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			np.mu.Lock()
			ip, port, proto := target.IP, target.Port, target.Proto
			if strings.EqualFold(proto, "udp") {
				// Opening a UDP socket proves nothing about the peer. Retain failures
				// and real connection evidence, but discard legacy fabricated RTTs.
				target.LatencyMs = 0
				if target.ProbeStatus != "connected" {
					target.ProbeStatus = "unknown"
				}
				np.mu.Unlock()
				return
			}
			np.mu.Unlock()
			addr := net.JoinHostPort(ip, strconv.Itoa(port))

			// Layered timeout strategy: generous first attempt, strict retry
			for attempt := 0; attempt < probeConfig.MaxAttempts; attempt++ {
				if ctx.Err() != nil {
					return
				}

				strategy := GetProbeStrategy(attempt)
				start := time.Now()
				var dialErr error

				dialer := net.Dialer{Timeout: strategy.TCPTimeout}
				conn, err := dialer.DialContext(ctx, "tcp", addr)
				dialErr = err
				if err == nil {
					_ = conn.Close()
					latency := max(1, int(time.Since(start).Milliseconds()))
					np.mu.Lock()
					target.LatencyMs = latency
					target.ProbeStatus = "reachable"
					target.LastChecked = time.Now()
					target.FailCount = 0
					np.mu.Unlock()
					return
				}

				// Log retry attempts for debugging
				if attempt > 0 && dialErr != nil {
					stats.LogInfo("Probe", "节点 %s:%d 第 %d 次重试失败: %v", target.IP, target.Port, attempt+1, dialErr)
				}
			}

			// Cancellation/budget expiry is not evidence that the node failed.
			if ctx.Err() != nil {
				return
			}
			// All attempts failed
			np.mu.Lock()
			target.LatencyMs = -1
			target.ProbeStatus = "unreachable"
			target.LastChecked = time.Now()
			target.FailCount++
			np.mu.Unlock()
		}(n)
	}
	wg.Wait()

	// Re-sort candidates and persist updated probe latencies
	np.mu.Lock()
	np.rebuildCandidatesLocked()
	candidatesCopy := make([]*Node, len(np.candidates))
	copy(candidatesCopy, np.candidates)
	np.saveStoreLocked()
	np.mu.Unlock()

	stats.LogInfo("Probe", "全量节点测速完成！")

	if len(candidatesCopy) > 0 {
		go np.PreWarmSSTPPorts(ctx, candidatesCopy)
	}
}

// PreWarmSSTPPorts checks and caches SSTP listening ports on top responsive candidate nodes in the background.
func (np *NodePool) PreWarmSSTPPorts(ctx context.Context, nodes []*Node) {
	if len(nodes) == 0 {
		return
	}
	limit := 30
	var toCheck []*Node
	for _, n := range nodes {
		np.mu.RLock()
		unprobed := n.SSTPPort == 0 && n.LatencyMs > 0
		np.mu.RUnlock()
		if unprobed {
			toCheck = append(toCheck, n)
			if len(toCheck) >= limit {
				break
			}
		}
	}

	if len(toCheck) == 0 {
		return
	}

	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup

	for _, n := range toCheck {
		select {
		case <-ctx.Done():
			return
		default:
		}

		wg.Add(1)
		go func(target *Node) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			ports := sstp.ResolveCandidatePorts(target.Port, target.Proto)
			foundPort := sstp.DetectSSTPPort(ctx, target.IP, ports, 2*time.Second)
			if foundPort > 0 {
				np.mu.Lock()
				target.SSTPPort = foundPort
				np.mu.Unlock()
			}
		}(n)
	}
	wg.Wait()

	np.mu.Lock()
	np.saveStoreLocked()
	np.mu.Unlock()
}

func (np *NodePool) ProbeSpecificNodes(ctx context.Context, ids []string) []*Node {
	np.mu.RLock()
	var targets []*Node
	if len(ids) == 0 {
		targets = append(targets, np.candidates...)
	} else {
		idMap := make(map[string]bool)
		for _, id := range ids {
			idMap[id] = true
		}
		for _, n := range np.candidates {
			if idMap[n.ID] || idMap[n.IP] {
				targets = append(targets, n)
			}
		}
		// Also search cold storage if a requested node is not currently in the hot 250 candidates
		if len(targets) < len(ids) {
			for _, n := range np.nodeStore {
				if idMap[n.ID] || idMap[n.IP] {
					alreadyAdded := false
					for _, t := range targets {
						if t.ID == n.ID {
							alreadyAdded = true
							break
						}
					}
					if !alreadyAdded {
						targets = append(targets, n)
					}
				}
			}
		}
	}
	np.mu.RUnlock()

	np.ProbeNodes(ctx, targets)
	return targets
}

func (np *NodePool) cloneNodeWithMetadata(n *Node, inActive bool) *Node {
	cp := *n
	cp.IsFavorite = np.favorites != nil && np.favorites.IsFavorite(n.ID)
	cp.InActivePool = inActive
	if np.reputation != nil {
		cp.ReputationScore = np.reputation.GetScore(n.IP)
	} else {
		cp.ReputationScore = 60
	}
	return &cp
}

func (np *NodePool) GetCandidates() []*Node {
	np.mu.RLock()
	defer np.mu.RUnlock()

	res := make([]*Node, len(np.candidates))
	for i, n := range np.candidates {
		res[i] = np.cloneNodeWithMetadata(n, true)
	}
	return res
}

// GetAllNodes returns all non-blacklisted nodes across both the active hot pool and cold storage.
// Active pool nodes appear first (InActivePool=true), followed by ranked cold-storage nodes (InActivePool=false).
func (np *NodePool) GetAllNodes() []*Node {
	np.mu.RLock()
	defer np.mu.RUnlock()

	res := make([]*Node, 0, len(np.nodeStore))
	seen := make(map[string]bool, len(np.candidates))
	for _, n := range np.candidates {
		res = append(res, np.cloneNodeWithMetadata(n, true))
		seen[n.ID] = true
	}

	var cold []*Node
	for _, n := range np.nodeStore {
		if seen[n.ID] {
			continue
		}
		if np.blacklist != nil && np.blacklist.IsNodeBlocked(n.ID, n.IP) {
			continue
		}
		cold = append(cold, n)
	}
	np.sortNodesByRank(cold)
	for _, n := range cold {
		res = append(res, np.cloneNodeWithMetadata(n, false))
	}
	return res
}

// NonBlacklistedCount returns the total number of stored non-blacklisted nodes.
func (np *NodePool) NonBlacklistedCount() int {
	np.mu.RLock()
	defer np.mu.RUnlock()

	count := 0
	for _, n := range np.nodeStore {
		if np.blacklist != nil && np.blacklist.IsNodeBlocked(n.ID, n.IP) {
			continue
		}
		count++
	}
	return count
}

func (np *NodePool) Favorites() *FavoritesManager {
	return np.favorites
}

func (np *NodePool) Reputation() *ReputationManager {
	return np.reputation
}

func (np *NodePool) GetNodeByID(id string) *Node {
	np.mu.RLock()
	defer np.mu.RUnlock()

	if n, ok := np.nodeStore[id]; ok {
		cp := *n
		cp.IsFavorite = np.favorites.IsFavorite(n.ID)
		if np.reputation != nil {
			cp.ReputationScore = np.reputation.GetScore(n.IP)
		} else {
			cp.ReputationScore = 60
		}
		return &cp
	}

	for _, n := range np.candidates {
		if n.ID == id || n.IP == id {
			cp := *n
			cp.IsFavorite = np.favorites.IsFavorite(n.ID)
			if np.reputation != nil {
				cp.ReputationScore = np.reputation.GetScore(n.IP)
			} else {
				cp.ReputationScore = 60
			}
			return &cp
		}
	}
	return nil
}

func (np *NodePool) SelectBestWithFilter(ipType string, countries []string, preferFavorites bool) *Node {
	np.mu.RLock()
	defer np.mu.RUnlock()

	countryMap := make(map[string]bool)
	for _, c := range countries {
		countryMap[strings.ToUpper(strings.TrimSpace(c))] = true
	}

	matchesFilter := func(n *Node) bool {
		if np.blacklist.IsBlacklisted(n.ID) {
			return false
		}
		if len(countryMap) > 0 && !countryMap[n.CountryShort] {
			return false
		}
		if ipType != "" && ipType != "all" && n.IPType != ipType {
			return false
		}
		return true
	}

	// 1. If preferFavorites is true, look for reachable favorites first
	if preferFavorites {
		var favs []*Node
		for _, n := range np.candidates {
			if np.favorites.IsFavorite(n.ID) && matchesFilter(n) && (n.LatencyMs > 0 || n.ProbeStatus == "connected") {
				favs = append(favs, n)
			}
		}
		if len(favs) > 0 {
			sort.Slice(favs, func(i, j int) bool {
				return favs[i].LatencyMs < favs[j].LatencyMs
			})
			return favs[0]
		}
	}

	// 2. Reachable nodes matching filter
	var reachable []*Node
	for _, n := range np.candidates {
		if matchesFilter(n) && (n.LatencyMs > 0 || n.ProbeStatus == "connected") {
			reachable = append(reachable, n)
		}
	}
	if len(reachable) > 0 {
		sort.Slice(reachable, func(i, j int) bool {
			scoreA := 60
			scoreB := 60
			if np.reputation != nil {
				scoreA = np.reputation.GetScore(reachable[i].IP)
				scoreB = np.reputation.GetScore(reachable[j].IP)
			}
			if (scoreA < 35) != (scoreB < 35) {
				return scoreA >= 35
			}
			if reachable[i].LatencyMs != reachable[j].LatencyMs {
				return reachable[i].LatencyMs < reachable[j].LatencyMs
			}
			return scoreA > scoreB
		})
		return reachable[0]
	}

	// 3. Fallback to any node matching filter in active pool
	for _, n := range np.candidates {
		if matchesFilter(n) {
			return n
		}
	}

	// 3.5 Cold storage fallback when explicit countries were requested
	if len(countryMap) > 0 {
		var coldMatched []*Node
		for _, n := range np.nodeStore {
			if (np.activeIDs != nil && np.activeIDs[n.ID]) || !matchesFilter(n) {
				continue
			}
			coldMatched = append(coldMatched, n)
		}
		if len(coldMatched) > 0 {
			np.sortNodesByRank(coldMatched)
			return coldMatched[0]
		}
	}

	// 4. Ultimate fallback to standard SelectBest
	return np.selectBestLocked()
}

func (np *NodePool) selectBestLocked() *Node {
	var reachable []*Node
	for _, n := range np.candidates {
		if !np.blacklist.IsBlacklisted(n.ID) && (n.LatencyMs > 0 || n.ProbeStatus == "connected") {
			reachable = append(reachable, n)
		}
	}
	if len(reachable) > 0 {
		sort.Slice(reachable, func(i, j int) bool {
			scoreA := 60
			scoreB := 60
			if np.reputation != nil {
				scoreA = np.reputation.GetScore(reachable[i].IP)
				scoreB = np.reputation.GetScore(reachable[j].IP)
			}
			if (scoreA < 35) != (scoreB < 35) {
				return scoreA >= 35
			}
			if reachable[i].LatencyMs != reachable[j].LatencyMs {
				return reachable[i].LatencyMs < reachable[j].LatencyMs
			}
			return scoreA > scoreB
		})
		return reachable[0]
	}
	for _, n := range np.candidates {
		if !np.blacklist.IsBlacklisted(n.ID) {
			return n
		}
	}
	return nil
}

func (np *NodePool) SelectBest() *Node {
	np.mu.RLock()
	defer np.mu.RUnlock()
	return np.selectBestLocked()
}

func (np *NodePool) RebuildCandidates() {
	np.mu.Lock()
	defer np.mu.Unlock()
	np.rebuildCandidatesLocked()
}

func (np *NodePool) Blacklist() *BlacklistManager {
	return np.blacklist
}

func (np *NodePool) Status() (string, string, time.Time, int, int) {
	np.mu.RLock()
	defer np.mu.RUnlock()

	return np.lastStatus, np.lastSource, np.lastUpdated, len(np.candidates), len(np.nodeStore)
}

func (np *NodePool) SetCandidatesForTest(candidates []*Node) {
	np.mu.Lock()
	defer np.mu.Unlock()
	np.candidates = candidates
	np.allRawNodes = candidates
	np.activeIDs = make(map[string]bool, len(candidates))
	for _, c := range candidates {
		np.activeIDs[c.ID] = true
		np.nodeStore[c.ID] = c
	}
}

func (np *NodePool) FindPort(nodeID, ip string) int {
	np.mu.RLock()
	defer np.mu.RUnlock()
	if n, ok := np.nodeStore[nodeID]; ok && n.Port > 0 {
		return n.Port
	}
	for _, n := range np.nodeStore {
		if n.IP == ip && n.Port > 0 {
			return n.Port
		}
	}
	return 0
}

// ReviveBlacklistedNodes probes all currently blacklisted nodes and restores those that are alive.
func (np *NodePool) ReviveBlacklistedNodes(ctx context.Context) int {
	revived, err := np.blacklist.ProbeAndRevive(ctx, np.FindPort)
	if err != nil || len(revived) == 0 {
		return 0
	}

	stats.LogInfo("Blacklist", "🎉 探活复活检测完成：成功复活 %d 个已屏蔽节点并重新释放！", len(revived))

	np.mu.Lock()
	np.rebuildCandidatesLocked()
	np.saveStoreLocked()
	var restoredNodes []*Node
	for _, r := range revived {
		if n, ok := np.nodeStore[r.ID]; ok {
			restoredNodes = append(restoredNodes, n)
		}
	}
	np.mu.Unlock()

	if len(restoredNodes) > 0 {
		go np.ProbeNodes(ctx, restoredNodes)
		go np.enricher.EnrichNodes(ctx, restoredNodes)
	}

	return len(revived)
}

// StartRevivalLoop starts periodic background probe testing for blacklisted nodes every 3 hours.
func (np *NodePool) StartRevivalLoop(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(3 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats.LogInfo("Blacklist", "正在执行每 3 小时定期屏蔽库探活与节点复活检测...")
				np.ReviveBlacklistedNodes(ctx)
			}
		}
	}()
}

// PortKnockTCP checks if a specific TCP port is reachable within the given timeout.
func PortKnockTCP(ip string, port int, timeout time.Duration) (bool, time.Duration, error) {
	addr := net.JoinHostPort(ip, strconv.Itoa(port))
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, timeout)
	latency := time.Since(start)
	if err == nil {
		_ = conn.Close()
		return true, latency, nil
	}
	return false, latency, err
}

// PortKnockResult represents the result of knocking a node's port.
type PortKnockResult struct {
	Node      *Node
	Reachable bool
	Unknown   bool // UDP requires a protocol handshake; not a successful port knock.
	Latency   time.Duration
	Error     error
}

// BatchPortKnock concurrently checks TCP reachability for a batch of nodes.
func BatchPortKnock(ctx context.Context, nodes []*Node, timeout time.Duration) []PortKnockResult {
	if len(nodes) == 0 {
		return nil
	}

	results := make([]PortKnockResult, len(nodes))
	concurrency := 48
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, n := range nodes {
		wg.Add(1)
		go func(idx int, target *Node) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				results[idx] = PortKnockResult{Node: target, Reachable: false, Error: ctx.Err()}
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			if strings.ToLower(target.Proto) == "udp" {
				results[idx] = PortKnockResult{Node: target, Unknown: true}
				return
			}

			reachable, latency, err := PortKnockTCP(target.IP, target.Port, timeout)
			results[idx] = PortKnockResult{
				Node:      target,
				Reachable: reachable,
				Latency:   latency,
				Error:     err,
			}
		}(i, n)
	}

	wg.Wait()
	return results
}

// FilterReachableNodes mutates caller-owned snapshots, not shared pool nodes.
// It performs fast TCP port knock to filter out dead TCP ports.
// UDP nodes remain candidates with unknown reachability until an actual connection succeeds.
func FilterReachableNodes(nodes []*Node, timeout time.Duration) []*Node {
	if len(nodes) == 0 {
		return nodes
	}

	batches := (len(nodes) + 47) / 48
	budget := timeout + 2*time.Second
	if scaled := time.Duration(batches) * timeout; scaled > budget {
		budget = scaled
	}
	if budget > 20*time.Second {
		budget = 20 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()

	results := BatchPortKnock(ctx, nodes, timeout)
	var reachable []*Node
	now := time.Now()
	for _, res := range results {
		if res.Unknown {
			reachable = append(reachable, res.Node)
			res.Node.LatencyMs = 0
			if res.Node.ProbeStatus != "connected" {
				res.Node.ProbeStatus = "unknown"
			}
			continue
		}
		if res.Reachable {
			res.Node.ProbeStatus = "reachable"
			reachable = append(reachable, res.Node)
			if res.Latency > 0 {
				res.Node.LatencyMs = max(1, int(res.Latency.Milliseconds()))
				res.Node.LastChecked = now
			}
			res.Node.FailCount = 0
		} else if res.Error != nil && (errors.Is(res.Error, context.Canceled) || errors.Is(res.Error, context.DeadlineExceeded)) {
			// Context cancellation/budget expiry is not proof the remote port is dead; keep node unpenalized.
			reachable = append(reachable, res.Node)
		} else if strings.ToLower(res.Node.Proto) != "udp" {
			res.Node.LatencyMs = -1
			res.Node.ProbeStatus = "unreachable"
			res.Node.LastChecked = now
			res.Node.FailCount++
		}
	}
	return reachable
}
