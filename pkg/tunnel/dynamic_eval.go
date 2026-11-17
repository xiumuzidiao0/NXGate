package tunnel

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"aimili-vpngate-go/pkg/nodes"
	"aimili-vpngate-go/pkg/stats"
)

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
			useZeroTun := g.ConnectMode == "zero_tun" || g.ConnectMode == "sstp"
			if !useZeroTun && !m.pool.CanUseKernelTUN() {
				useZeroTun = true
				stats.LogWarn("DynamicGroup", "[%s] 系统不支持内核 TUN/OpenVPN，出口自适应降级为用户态免 TUN 模式", g.Name)
			}
			if useZeroTun {
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

	isUserland := m.isPrimaryUserlandMode(g)
	// 严苛物理实测校验：如果配置了解锁要求（如三大 AI 全通），实测当前主网关
	primaryUnlockValid := true
	var unlockTarget *Tunnel
	if primaryTun != nil {
		unlockTarget = primaryTun
	} else if isUserland && currentActiveNode != nil && primaryHealthy {
		if dialer := m.primaryUserlandDialer(); dialer != nil {
			unlockTarget = &Tunnel{
				ID:         "primary-userland",
				DevName:    fmt.Sprintf("sstp-%d", currentActiveNode.Port),
				Node:       currentActiveNode,
				Status:     StatusConnected,
				IsUserland: true,
				Dialer:     dialer,
			}
		}
	}
	if g.UnlockFilter != "" && g.UnlockFilter != "none" && unlockTarget != nil && currentActiveNode != nil {
		currUnlock := unlockTarget.GetUnlock()
		if currUnlock == nil || !currUnlock.IsProbed {
			if m.pool.UnlockDetector() != nil {
				probeCtx, probeCancel := context.WithTimeout(ctx, 8*time.Second)
				currUnlock = m.pool.UnlockDetector().ProbeTunnelInstance(probeCtx, unlockTarget)
				probeCancel()
				unlockTarget.SetUnlock(currUnlock)
				if primaryTun == nil && currentActiveNode != nil {
					currentActiveNode.Unlock = currUnlock
				}
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

	modeTag := "tun0"
	if isUserland {
		modeTag = "免TUN"
	}

	// A connection attempt to this very node may already be in flight: the health
	// checker's failover and the evaluation loop overlap, and issuing another
	// connect would tear down the tunnel that is still being established, which
	// shows up as a burst of redundant reconnects to the same relay. Let it finish.
	if m.primaryConnector != nil && currentActiveNode != nil &&
		(currentActiveNode.ID == best.ID || currentActiveNode.IP == best.IP) &&
		m.primaryConnector.GetStatus() == primaryStatusConnecting {
		m.mu.Lock()
		g.LastEvaluatedAt = time.Now()
		g.StatusText = fmt.Sprintf("主网关建立中: %s (%s, %s)", best.IP, best.CountryShort, modeTag)
		m.saveLocked()
		m.mu.Unlock()
		return
	}

	// If periodic rotation is disabled (IntervalMinutes <= 0), keep the healthy primary connection without rotating ONLY IF it satisfies unlock requirement
	if g.IntervalMinutes <= 0 && primaryHealthy && currentActiveNode != nil && primaryUnlockValid {
		m.mu.Lock()
		if primaryTun != nil {
			g.ActiveTunnelIDs = []string{primaryTun.ID}
		} else if isUserland {
			g.ActiveTunnelIDs = []string{"primary-userland"}
		}
		g.LastEvaluatedAt = time.Now()
		g.StatusText = fmt.Sprintf("主网关固定连接: %s (%s, %s)", currentActiveNode.IP, currentActiveNode.CountryShort, modeTag)
		m.saveLocked()
		m.mu.Unlock()
		return
	}

	// If primary is already healthy on this best node AND satisfies unlock requirement, keep it
	if primaryHealthy && currentActiveNode != nil && primaryUnlockValid && (currentActiveNode.ID == best.ID || currentActiveNode.IP == best.IP) {
		m.mu.Lock()
		if primaryTun != nil {
			g.ActiveTunnelIDs = []string{primaryTun.ID}
		} else if isUserland {
			g.ActiveTunnelIDs = []string{"primary-userland"}
		}
		g.LastEvaluatedAt = time.Now()
		g.StatusText = fmt.Sprintf("主网关正常运行: %s (%s, %s)", currentActiveNode.IP, currentActiveNode.CountryShort, modeTag)
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
		} else if isUserland && currentActiveNode != nil {
			cAt := m.primaryConnectedAt()
			if !cAt.IsZero() && time.Since(cAt) < 60*time.Second {
				m.mu.Lock()
				g.ActiveTunnelIDs = []string{"primary-userland"}
				g.LastEvaluatedAt = time.Now()
				g.StatusText = fmt.Sprintf("主网关正常运行: %s (%s, 免TUN)", currentActiveNode.IP, currentActiveNode.CountryShort)
				m.saveLocked()
				m.mu.Unlock()
				return
			}
		}
	}

	if ctx.Err() != nil {
		return
	}
	// Trigger primary connection to the best node
	if m.primaryConnector != nil {
		targetMode := "kernel"
		if isUserland {
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
