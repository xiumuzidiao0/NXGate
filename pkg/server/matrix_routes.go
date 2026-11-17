package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"aimili-vpngate-go/pkg/nodes"
	"aimili-vpngate-go/pkg/proxy"
	"aimili-vpngate-go/pkg/tunnel"
	"aimili-vpngate-go/pkg/vpn"
)

func (s *Server) getActiveTunnels() []*tunnel.Tunnel {
	var tunnels []*tunnel.Tunnel
	if s.tunnelPool != nil {
		tunnels = s.tunnelPool.ListTunnels()
	}
	// Synthesize primary userland Zero-TUN egress so it is properly exposed in active tunnels and metrics
	if s.vpn != nil && s.vpn.DriverMode() == vpn.DriverUserland && s.vpn.GetStatus() == string(vpn.StatusConnected) {
		activeNode := s.vpn.GetActiveNode()
		if activeNode != nil {
			var unlockRes *nodes.UnlockResult
			if s.tunnelPool != nil && s.tunnelPool.UnlockDetector() != nil {
				unlockRes = s.tunnelPool.UnlockDetector().GetUnlock(activeNode.IP)
			}
			if unlockRes == nil {
				unlockRes = activeNode.Unlock
			}

			primaryUserland := &tunnel.Tunnel{
				ID:          "primary-userland",
				DevName:     fmt.Sprintf("sstp-%d", activeNode.Port),
				DevIndex:    0,
				Node:        activeNode,
				Status:      tunnel.StatusConnected,
				Message:     "系统主连网关 (用户态免TUN直连)",
				ConnectedAt: s.vpn.ConnectedAt(),
				LatencyMs:   activeNode.LatencyMs,
				IsUserland:  true,
				Unlock:      unlockRes,
			}
			tunnels = append([]*tunnel.Tunnel{primaryUserland}, tunnels...)
		}
	}
	return tunnels
}

func (s *Server) handleListTunnels(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, s.getActiveTunnels())
}

type StartTunnelRequest struct {
	NodeID string `json:"node_id"`
}

func (s *Server) handleStartTunnel(w http.ResponseWriter, r *http.Request) {
	var req StartTunnelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NodeID == "" {
		s.writeError(w, http.StatusBadRequest, "缺少 node_id 参数")
		return
	}

	node := s.pool.GetNodeByID(req.NodeID)
	if node == nil {
		s.writeError(w, http.StatusBadRequest, "节点不存在或已被过滤")
		return
	}

	if s.tunnelPool == nil {
		s.writeError(w, http.StatusInternalServerError, "隧道池未初始化")
		return
	}

	tun, err := s.tunnelPool.StartTunnel(node)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"message": "隧道正在发起连接...",
		"tunnel":  tun.Snapshot(),
	})
}

type StopTunnelRequest struct {
	TunnelID string `json:"tunnel_id"`
}

func (s *Server) handleStopTunnel(w http.ResponseWriter, r *http.Request) {
	var req StopTunnelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TunnelID == "" {
		s.writeError(w, http.StatusBadRequest, "缺少 tunnel_id 参数")
		return
	}

	if s.tunnelPool == nil {
		s.writeError(w, http.StatusInternalServerError, "隧道池未初始化")
		return
	}

	if err := s.tunnelPool.StopTunnel(req.TunnelID); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"message": "指定隧道已成功断开并释放",
	})
}

func (s *Server) handleGetPortRules(w http.ResponseWriter, r *http.Request) {
	if s.portMgr == nil {
		s.writeJSON(w, http.StatusOK, []proxy.PortRule{})
		return
	}
	s.writeJSON(w, http.StatusOK, s.portMgr.GetRules())
}

type SetPortRulesRequest struct {
	Rules []proxy.PortRule `json:"rules"`
}

func (s *Server) handleSetPortRules(w http.ResponseWriter, r *http.Request) {
	var req SetPortRulesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "请求格式解析失败")
		return
	}

	if s.portMgr == nil {
		s.writeError(w, http.StatusInternalServerError, "代理端口管理器未初始化")
		return
	}

	if err := s.portMgr.ApplyRules(req.Rules); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	go s.syncSingBoxOutboundCredentials(context.Background())

	s.writeJSON(w, http.StatusOK, map[string]any{
		"message": "端口分流规则已更新生效！",
		"rules":   s.portMgr.GetRules(),
	})
}

func (s *Server) handleListTunnelGroups(w http.ResponseWriter, r *http.Request) {
	if s.dynamicMgr == nil {
		s.writeJSON(w, http.StatusOK, []*tunnel.DynamicGroup{})
		return
	}
	s.writeJSON(w, http.StatusOK, s.dynamicMgr.ListGroups())
}

func (s *Server) handleSaveTunnelGroup(w http.ResponseWriter, r *http.Request) {
	var g tunnel.DynamicGroup
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		s.writeError(w, http.StatusBadRequest, "请求格式解析失败")
		return
	}

	if s.dynamicMgr == nil {
		s.writeError(w, http.StatusInternalServerError, "动态隧道组管理器未初始化")
		return
	}

	if err := s.dynamicMgr.SaveGroup(&g); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if g.Enabled {
		go s.dynamicMgr.EvaluateGroup(context.Background(), &g)
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"message": "动态自适应隧道组已保存并启动评估！",
		"group":   s.dynamicMgr.GetGroup(g.ID),
	})
}

type DeleteGroupRequest struct {
	ID string `json:"id"`
}

func (s *Server) handleDeleteTunnelGroup(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		var req DeleteGroupRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		id = req.ID
	}

	if id == "" {
		s.writeError(w, http.StatusBadRequest, "缺少 id 参数")
		return
	}

	if s.dynamicMgr == nil {
		s.writeError(w, http.StatusInternalServerError, "动态隧道组管理器未初始化")
		return
	}

	s.dynamicMgr.DeleteGroup(id)
	s.writeJSON(w, http.StatusOK, map[string]string{
		"message": "动态自适应组已删除并释放关联出口",
	})
}

func (s *Server) handleEvaluateTunnelGroups(w http.ResponseWriter, r *http.Request) {
	if s.dynamicMgr == nil {
		s.writeError(w, http.StatusInternalServerError, "动态隧道组管理器未初始化")
		return
	}

	go func() {
		for _, g := range s.dynamicMgr.ListGroups() {
			if g.Enabled {
				s.dynamicMgr.EvaluateGroup(context.Background(), g)
			}
		}
	}()

	s.writeJSON(w, http.StatusOK, map[string]string{
		"message": "已在后台启动全量动态自适应组评估与轮换",
	})
}

func (s *Server) handleGetUnlockStatus(w http.ResponseWriter, r *http.Request) {
	if s.tunnelPool == nil || s.tunnelPool.UnlockDetector() == nil {
		s.writeJSON(w, http.StatusOK, map[string]*tunnel.UnlockResult{})
		return
	}
	s.writeJSON(w, http.StatusOK, s.tunnelPool.UnlockDetector().GetAllCached())
}

type ProbeUnlockRequest struct {
	TunnelID string `json:"tunnel_id"`
}

func (s *Server) handleProbeTunnelUnlock(w http.ResponseWriter, r *http.Request) {
	var req ProbeUnlockRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if s.tunnelPool == nil {
		s.writeError(w, http.StatusInternalServerError, "隧道池未初始化")
		return
	}

	var t *tunnel.Tunnel
	if req.TunnelID == "primary-userland" {
		if s.vpn != nil && s.vpn.DriverMode() == vpn.DriverUserland && s.vpn.GetStatus() == string(vpn.StatusConnected) {
			activeNode := s.vpn.GetActiveNode()
			sTun := s.vpn.GetSSTPTunnel()
			if activeNode != nil && sTun != nil {
				t = &tunnel.Tunnel{
					ID:         "primary-userland",
					DevName:    fmt.Sprintf("sstp-%d", activeNode.Port),
					Node:       activeNode,
					Status:     tunnel.StatusConnected,
					IsUserland: true,
					Dialer:     sTun.Dial,
				}
			}
		}
	} else {
		t = s.tunnelPool.GetTunnel(req.TunnelID)
	}

	if t == nil || t.Node == nil {
		s.writeError(w, http.StatusBadRequest, "指定隧道不存在或未上线")
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		detector := s.tunnelPool.UnlockDetector()
		if detector == nil {
			return
		}
		res := detector.ProbeTunnelInstance(ctx, t)
		if t.ID == "primary-userland" {
			if t.Node != nil {
				t.Node.Unlock = res
			}
		} else {
			t.SetUnlock(res)
		}
	}()

	s.writeJSON(w, http.StatusOK, map[string]string{
		"message": "已在后台启动流媒体与 AI 解锁探测",
	})
}
