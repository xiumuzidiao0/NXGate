package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/nodes"
	"aimili-vpngate-go/pkg/proxy"
	"aimili-vpngate-go/pkg/stats"
	"aimili-vpngate-go/pkg/tunnel"
	"aimili-vpngate-go/pkg/vpn"
)

type StatusResponse struct {
	VPN                  any                    `json:"vpn"`
	Traffic              stats.TrafficSnapshot  `json:"traffic"`
	ProxyAddr            string                 `json:"proxy_addr"`
	NodeCount            int                    `json:"node_count"`
	TotalNodeCount       int                    `json:"total_node_count"`
	StoredAvailableCount int                    `json:"stored_available_count"`
	HotPoolLimit         int                    `json:"hot_pool_limit"`
	MinPerCountry        int                    `json:"min_per_country"`
	MajorCountryCap      int                    `json:"major_country_cap"`
	NodeSource           string                 `json:"node_source"`
	BlacklistCount   int                    `json:"blacklist_count"`
	AdminPath        string                 `json:"admin_path"`
	UIAuthEnabled    bool                   `json:"ui_auth_enabled"`
	UIPublicExposure bool                   `json:"ui_public_exposure"`
	Version          string                 `json:"version"`
	Tunnels          []*tunnel.Tunnel       `json:"tunnels"`
	PortRules        []proxy.PortRule       `json:"port_rules"`
	DynamicGroups    []*tunnel.DynamicGroup `json:"dynamic_groups"`
}

func (s *Server) buildStatusResponse() StatusResponse {
	vpnState := s.vpn.Snapshot()
	if vpnState.ActiveNode != nil && s.pool != nil && s.pool.Favorites() != nil {
		vpnState.ActiveNode.IsFavorite = s.pool.Favorites().IsFavorite(vpnState.ActiveNode.ID)
	}
	traffic := stats.GetTrafficTracker().Snapshot()
	_, source, _, count, totalCount := s.pool.Status()
	blCount := s.pool.Blacklist().Count()

	proxyHost := s.cfg.ProxyHost
	if strings.Contains(proxyHost, ":") {
		proxyHost = "[" + proxyHost + "]"
	}

	tunnels := s.getActiveTunnels()
	var portRules []proxy.PortRule
	if s.portMgr != nil {
		portRules = s.portMgr.GetRules()
	}
	var dynamicGroups []*tunnel.DynamicGroup
	if s.dynamicMgr != nil {
		dynamicGroups = s.dynamicMgr.ListGroups()
	}

	return StatusResponse{
		VPN:                  vpnState,
		Traffic:              traffic,
		ProxyAddr:            fmt.Sprintf("%s:%d", proxyHost, s.cfg.ProxyPort),
		NodeCount:            count,
		TotalNodeCount:       totalCount,
		StoredAvailableCount: s.pool.NonBlacklistedCount(),
		HotPoolLimit:         s.cfg.GetNodeHotPoolSize(),
		MinPerCountry:        s.cfg.GetNodeMinPerCountry(),
		MajorCountryCap:      s.cfg.GetNodeMajorCountryCap(),
		NodeSource:           source,
		BlacklistCount:   blCount,
		AdminPath:        s.cfg.UIPath,
		UIAuthEnabled:    s.cfg.IsUIAuthEnabled(),
		UIPublicExposure: s.cfg.IsPublicUnauthenticated(),
		Version:          config.Version,
		Tunnels:          tunnels,
		PortRules:        portRules,
		DynamicGroups:    dynamicGroups,
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	resp := s.buildStatusResponse()
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	scope := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("scope")))
	var candidates []*nodes.Node
	if scope == "all" {
		candidates = s.pool.GetAllNodes()
	} else {
		candidates = s.pool.GetCandidates()
	}
	if s.tunnelPool != nil && s.tunnelPool.UnlockDetector() != nil {
		detector := s.tunnelPool.UnlockDetector()
		for _, n := range candidates {
			if realUnlock := detector.GetUnlock(n.IP); realUnlock != nil {
				n.Unlock = realUnlock
			} else if n.Unlock == nil || !n.Unlock.IsProbed || time.Since(n.Unlock.CheckedAt) > 6*time.Hour {
				n.Unlock = detector.EvaluateNodeUnlock(n)
			}
		}
	}
	respNodes := make([]nodes.Node, len(candidates))
	for i, n := range candidates {
		respNodes[i] = *n
		respNodes[i].ConfigData = ""
	}
	s.writeJSON(w, http.StatusOK, respNodes)
}

type ProbeNodesRequest struct {
	NodeIDs []string `json:"node_ids"`
}

func (s *Server) handleProbeNodes(w http.ResponseWriter, r *http.Request) {
	var req ProbeNodesRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		targets := s.pool.ProbeSpecificNodes(ctx, req.NodeIDs)
		if s.tunnelPool != nil && s.tunnelPool.UnlockDetector() != nil {
			detector := s.tunnelPool.UnlockDetector()
			for _, n := range targets {
				if realUnlock := detector.GetUnlock(n.IP); realUnlock != nil {
					n.Unlock = realUnlock
				} else {
					n.Unlock = detector.EvaluateNodeUnlock(n)
				}
			}
		}
	}()

	s.writeJSON(w, http.StatusOK, map[string]string{
		"message": "已在后台启动节点可用性与流媒体/AI解锁同步检测",
	})
}

type FavoriteRequest struct {
	NodeID string `json:"node_id"`
}

func (s *Server) handleToggleFavorite(w http.ResponseWriter, r *http.Request) {
	var req FavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NodeID == "" {
		s.writeError(w, http.StatusBadRequest, "缺少 node_id 参数")
		return
	}

	isFav := s.pool.Favorites().Toggle(req.NodeID)
	s.pool.RebuildCandidates()
	s.writeJSON(w, http.StatusOK, map[string]any{
		"node_id":     req.NodeID,
		"is_favorite": isFav,
	})
}

type ConnectRequest struct {
	NodeID string `json:"node_id"`
	Mode   string `json:"mode,omitempty"`
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	var req ConnectRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	var target *nodes.Node
	if req.NodeID != "" {
		target = s.pool.GetNodeByID(req.NodeID)
		if target == nil {
			s.writeError(w, http.StatusBadRequest, "指定节点不存在或已被过滤")
			return
		}
	} else {
		target = s.pool.SelectBest()
		if target == nil {
			s.writeError(w, http.StatusBadRequest, "当前无可用候选节点，请先刷新节点源")
			return
		}
	}

	mode := vpn.DriverKernel
	if req.Mode == "sstp" || req.Mode == "userland" || req.Mode == "zero_tun" {
		mode = vpn.DriverUserland
	}

	go func() {
		_ = s.vpn.ConnectWithMode(target, string(mode))
	}()

	s.writeJSON(w, http.StatusOK, map[string]any{
		"message":     "已发起连接请求",
		"node_id":     target.ID,
		"ip":          target.IP,
		"country":     target.CountryShort,
		"driver_mode": string(mode),
	})
}

func (s *Server) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	s.vpn.Disconnect("用户控制台主动断开")
	s.writeJSON(w, http.StatusOK, map[string]string{
		"message": "已成功断开 VPN",
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if s.pool != nil && s.pool.IsRefreshing() {
		s.writeJSON(w, http.StatusOK, map[string]string{
			"message": "节点列表刷新任务已在后台执行中，请勿重复操作",
		})
		return
	}

	go func() {
		_ = s.pool.Refresh(context.Background())
	}()

	s.writeJSON(w, http.StatusOK, map[string]string{
		"message": "已在后台启动节点列表刷新与测速",
	})
}

func (s *Server) handleBlacklist(w http.ResponseWriter, r *http.Request) {
	list := s.pool.Blacklist().List()
	s.writeJSON(w, http.StatusOK, list)
}

type BlacklistNodeReq struct {
	NodeID          string `json:"node_id"`
	IP              string `json:"ip,omitempty"`
	Country         string `json:"country,omitempty"`
	DurationMinutes int    `json:"duration_minutes,omitempty"`
	Reason          string `json:"reason,omitempty"`
	Permanent       bool   `json:"permanent,omitempty"`
	Scope           string `json:"scope,omitempty"`    // "node" (默认) 或 "ip" (整机IP屏蔽)
	GroupID         string `json:"group_id,omitempty"` // 来自哪个动态分流组的触发，便于立即重评
}

func (s *Server) handleBlacklistRemove(w http.ResponseWriter, r *http.Request) {
	var req BlacklistNodeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NodeID == "" {
		s.writeError(w, http.StatusBadRequest, "缺少 node_id 参数")
		return
	}
	s.pool.Blacklist().Remove(req.NodeID)
	if req.IP != "" {
		s.pool.Blacklist().Remove(req.IP)
	}
	s.pool.RebuildCandidates()
	stats.LogInfo("Server", "管理员解除了对节点/IP [%s] 的屏蔽", req.NodeID)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "已解除屏蔽",
		"node_id": req.NodeID,
	})
}

type BlacklistClearReq struct {
	Type string `json:"type,omitempty"` // "all", "temporary", "permanent"
}

func (s *Server) handleBlacklistClear(w http.ResponseWriter, r *http.Request) {
	clearType := r.URL.Query().Get("type")
	if clearType == "" && r.Body != nil {
		var req BlacklistClearReq
		_ = json.NewDecoder(r.Body).Decode(&req)
		clearType = req.Type
	}
	if clearType == "" {
		clearType = "temporary"
	}

	switch clearType {
	case "all":
		s.pool.Blacklist().ClearWithOptions(true)
		stats.LogInfo("Server", "管理员一键清空了全部屏蔽库 (包括永久与临时)")
	case "permanent":
		s.pool.Blacklist().ClearPermanent()
		stats.LogInfo("Server", "管理员清空了用户永久黑名单库")
	default:
		s.pool.Blacklist().ClearWithOptions(false)
		stats.LogInfo("Server", "管理员清空了临时故障隔离库")
	}
	s.pool.RebuildCandidates()

	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"type":    clearType,
		"message": "已清空对应屏蔽库",
	})
}

func (s *Server) handleBlacklistAdd(w http.ResponseWriter, r *http.Request) {
	var req BlacklistNodeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || (req.NodeID == "" && req.IP == "") {
		s.writeError(w, http.StatusBadRequest, "缺少 node_id 或 ip 参数")
		return
	}
	if req.NodeID == "" {
		req.NodeID = req.IP
	}
	dur := time.Duration(req.DurationMinutes) * time.Minute
	if dur <= 0 && !req.Permanent {
		dur = 24 * time.Hour
	}
	reason := req.Reason
	if reason == "" {
		if req.Permanent {
			reason = "用户手动永久屏蔽 (Tombstone)"
		} else {
			reason = "用户手动屏蔽"
		}
	}
	s.pool.Blacklist().MarkManualWithOptions(req.NodeID, req.IP, req.Country, reason, dur, req.Scope, req.Permanent)
	s.pool.RebuildCandidates()

	evictedTunnels, reEvaluatedGroups := s.evictBlockedTunnelsAndReevaluate(req.GroupID)

	targetDesc := req.NodeID
	if req.Scope == "ip" && req.IP != "" {
		targetDesc = fmt.Sprintf("整机IP [%s]", req.IP)
	} else if req.Scope == "cidr" {
		targetDesc = fmt.Sprintf("网段 [%s]", req.IP)
	}
	stats.LogInfo("Server", "管理员手动将 %s (%s) 屏蔽 (永久=%v, 时长=%v, 驱逐隧道=%d, 触发重评组=%d): %s",
		targetDesc, req.Country, req.Permanent, dur, len(evictedTunnels), len(reEvaluatedGroups), reason)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok":                  true,
		"message":             "已成功加入屏蔽库",
		"node_id":             req.NodeID,
		"permanent":           req.Permanent,
		"scope":               req.Scope,
		"evicted_tunnels":     evictedTunnels,
		"re_evaluated_groups": reEvaluatedGroups,
	})
}

func (s *Server) evictBlockedTunnelsAndReevaluate(triggerGroupID string) ([]string, []string) {
	evictedTunnels := make([]string, 0)
	affectedGroups := make(map[string]bool)
	disconnectPrimary := false

	if s.tunnelPool != nil && s.pool != nil {
		for _, t := range s.tunnelPool.ListTunnels() {
			if t == nil || t.Node == nil {
				continue
			}
			if !s.pool.Blacklist().IsNodeBlocked(t.Node.ID, t.Node.IP) {
				continue
			}
			evictedTunnels = append(evictedTunnels, t.ID)
			if t.DevIndex == 0 {
				disconnectPrimary = true
			} else {
				_ = s.tunnelPool.StopTunnel(t.ID)
			}
		}
	}

	if s.vpn != nil && s.pool != nil {
		if active := s.vpn.GetActiveNode(); active != nil && s.pool.Blacklist().IsNodeBlocked(active.ID, active.IP) {
			disconnectPrimary = true
		}
	}

	if disconnectPrimary {
		if s.vpn != nil {
			s.vpn.Disconnect("节点已被手动拉黑")
		} else if s.tunnelPool != nil {
			for _, tid := range evictedTunnels {
				_ = s.tunnelPool.StopTunnel(tid)
			}
		}
		if s.dynamicMgr != nil {
			if g := s.dynamicMgr.GetGroup(tunnel.SystemPrimaryGroupID); g != nil && g.Enabled {
				affectedGroups[g.ID] = true
			}
		}
		evictedTunnels = appendIfMissing(evictedTunnels, tunnel.PrimaryUserlandTunnelID)
	}

	if s.dynamicMgr != nil {
		for _, gid := range s.dynamicMgr.EvictTunnelIDs(evictedTunnels) {
			affectedGroups[gid] = true
		}
		if triggerGroupID != "" {
			if g := s.dynamicMgr.GetGroup(triggerGroupID); g != nil && g.Enabled {
				affectedGroups[g.ID] = true
			}
		}
	}

	reEvaluatedGroups := make([]string, 0, len(affectedGroups))
	for gid := range affectedGroups {
		reEvaluatedGroups = append(reEvaluatedGroups, gid)
	}
	sort.Strings(reEvaluatedGroups)
	if s.dynamicMgr != nil {
		for _, gid := range reEvaluatedGroups {
			s.dynamicMgr.EnqueueGroupEvaluation(gid)
		}
	}
	return evictedTunnels, reEvaluatedGroups
}

func appendIfMissing(list []string, target string) []string {
	for _, item := range list {
		if item == target {
			return list
		}
	}
	return append(list, target)
}

func (s *Server) handleBlacklistResurrect(w http.ResponseWriter, r *http.Request) {
	revivedCount := s.pool.ReviveBlacklistedNodes(r.Context())
	remainingCount := s.pool.Blacklist().Count()
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"revived_count":   revivedCount,
		"remaining_count": remainingCount,
		"message":         fmt.Sprintf("探活复活检测完成：成功复活并释放 %d 个节点，当前剩余屏蔽节点 %d 个", revivedCount, remainingCount),
	})
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	recent := stats.GetRingLog().Recent(100)
	s.writeJSON(w, http.StatusOK, recent)
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings := s.cfg.GetSettings()
	s.writeJSON(w, http.StatusOK, settings)
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "请求参数读取失败")
		return
	}
	var req config.SettingsDTO
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, "请求参数解析失败")
		return
	}

	var rawFields map[string]json.RawMessage
	_ = json.Unmarshal(body, &rawFields)
	present := make(map[string]bool, len(rawFields))
	for k := range rawFields {
		present[k] = true
	}
	var meta struct {
		DisableUIAuth *bool `json:"disable_ui_auth"`
	}
	_ = json.Unmarshal(body, &meta)

	if req.UIPort > 0 && (req.UIPort < 1 || req.UIPort > 65535) {
		s.writeError(w, http.StatusBadRequest, "Web 端口必须在 1 至 65535 之间")
		return
	}
	if req.ProxyPort > 0 && (req.ProxyPort < 1 || req.ProxyPort > 65535) {
		s.writeError(w, http.StatusBadRequest, "代理端口必须在 1 至 65535 之间")
		return
	}
	if req.UIPort > 0 && req.ProxyPort > 0 && req.UIPort == req.ProxyPort {
		s.writeError(w, http.StatusBadRequest, "Web 端口不能与代理端口相同")
		return
	}

	prevSettings := s.cfg.GetSettings()

	if err := s.cfg.UpdateSettingsWithPresence(req, present, meta.DisableUIAuth); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, config.ErrInvalidSettings) {
			status = http.StatusBadRequest
		}
		s.writeError(w, status, fmt.Sprintf("更新配置失败: %v", err))
		return
	}
	if s.pool != nil {
		s.pool.RebuildCandidates()
	}

	current := s.cfg.GetSettings()
	restartRequired := current.UIPort != prevSettings.UIPort || current.ProxyPort != prevSettings.ProxyPort
	stats.LogInfo("Server", "管理员通过 Web 控制台更新了系统配置: 账号=%s, 路径=/%s, Web端口=%d, 代理端口=%d (需重启=%v)",
		current.UIUsername, current.UIPath, current.UIPort, current.ProxyPort, restartRequired)

	go s.syncSingBoxOutboundCredentials(context.Background())

	msg := "配置修改成功并已持久化保存！"
	if restartRequired {
		msg = "配置已保存，端口变更将在重启 NXGate 服务后生效。"
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"message":          msg,
		"restart_required": restartRequired,
		"settings":         current,
	})
}
