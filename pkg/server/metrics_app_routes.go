package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/stats"
)

func (s *Server) handleTelegramTest(w http.ResponseWriter, r *http.Request) {
	if s.notifier == nil {
		s.writeError(w, http.StatusBadRequest, "Telegram 通知模块未就绪")
		return
	}
	if !s.notifier.IsConfigured() {
		s.writeError(w, http.StatusBadRequest, "请先填写并保存 Telegram Bot Token 与 Chat ID")
		return
	}
	err := s.notifier.SendMessage("🔔 <b>测试通知</b>: 这是一条来自 NXGate Web 控制台的 Telegram 连通性测试消息。配置成功！")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("发送失败: %v", err))
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{
		"message": "测试消息发送成功，请在 Telegram 查收！",
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	tr := stats.GetTrafficTracker().Snapshot()
	state := s.vpn.Snapshot()

	tunnelsOnline := len(s.getActiveTunnels())
	var nodeCount, totalNodeCount, blCount int
	if s.pool != nil {
		_, _, _, nodeCount, totalNodeCount = s.pool.Status()
		if s.pool.Blacklist() != nil {
			blCount = s.pool.Blacklist().Count()
		}
	}

	connectedVal := 0
	if state.TunnelReady {
		connectedVal = 1
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	// Modern NXGate Prometheus metrics
	_, _ = fmt.Fprintf(w, "# HELP nxgate_traffic_upload_bytes_total Total bytes uploaded through proxy\n")
	_, _ = fmt.Fprintf(w, "# TYPE nxgate_traffic_upload_bytes_total counter\n")
	_, _ = fmt.Fprintf(w, "nxgate_traffic_upload_bytes_total %d\n", tr.TotalUploadBytes)

	_, _ = fmt.Fprintf(w, "# HELP nxgate_traffic_download_bytes_total Total bytes downloaded through proxy\n")
	_, _ = fmt.Fprintf(w, "# TYPE nxgate_traffic_download_bytes_total counter\n")
	_, _ = fmt.Fprintf(w, "nxgate_traffic_download_bytes_total %d\n", tr.TotalDownloadBytes)

	_, _ = fmt.Fprintf(w, "# HELP nxgate_traffic_upload_speed_bps Current upload speed in bytes per second\n")
	_, _ = fmt.Fprintf(w, "# TYPE nxgate_traffic_upload_speed_bps gauge\n")
	_, _ = fmt.Fprintf(w, "nxgate_traffic_upload_speed_bps %d\n", tr.UploadSpeedBps)

	_, _ = fmt.Fprintf(w, "# HELP nxgate_traffic_download_speed_bps Current download speed in bytes per second\n")
	_, _ = fmt.Fprintf(w, "# TYPE nxgate_traffic_download_speed_bps gauge\n")
	_, _ = fmt.Fprintf(w, "nxgate_traffic_download_speed_bps %d\n", tr.DownloadSpeedBps)

	_, _ = fmt.Fprintf(w, "# HELP nxgate_active_connections Current active proxy client connections\n")
	_, _ = fmt.Fprintf(w, "# TYPE nxgate_active_connections gauge\n")
	_, _ = fmt.Fprintf(w, "nxgate_active_connections %d\n", tr.ActiveConnections)
	active, limit, rejected := s.cfg.ProxyConnectionStats()
	fmt.Fprintf(w, "# TYPE nxgate_proxy_budget_active gauge\nnxgate_proxy_budget_active %d\n# TYPE nxgate_proxy_budget_limit gauge\nnxgate_proxy_budget_limit %d\n# TYPE nxgate_proxy_budget_rejected_total counter\nnxgate_proxy_budget_rejected_total %d\n", active, limit, rejected)

	_, _ = fmt.Fprintf(w, "# HELP nxgate_vpn_connected Status of Primary VPN tunnel (1 = connected, 0 = disconnected)\n")
	_, _ = fmt.Fprintf(w, "# TYPE nxgate_vpn_connected gauge\n")
	_, _ = fmt.Fprintf(w, "nxgate_vpn_connected %d\n", connectedVal)

	_, _ = fmt.Fprintf(w, "# HELP nxgate_tunnels_online Total active concurrent egress tunnels\n")
	_, _ = fmt.Fprintf(w, "# TYPE nxgate_tunnels_online gauge\n")
	_, _ = fmt.Fprintf(w, "nxgate_tunnels_online %d\n", tunnelsOnline)

	_, _ = fmt.Fprintf(w, "# HELP nxgate_nodes_available Available candidate nodes ready for connection\n")
	_, _ = fmt.Fprintf(w, "# TYPE nxgate_nodes_available gauge\n")
	_, _ = fmt.Fprintf(w, "nxgate_nodes_available %d\n", nodeCount)

	_, _ = fmt.Fprintf(w, "# HELP nxgate_nodes_total Total known nodes in historical pool\n")
	_, _ = fmt.Fprintf(w, "# TYPE nxgate_nodes_total gauge\n")
	_, _ = fmt.Fprintf(w, "nxgate_nodes_total %d\n", totalNodeCount)

	_, _ = fmt.Fprintf(w, "# HELP nxgate_blacklist_count Number of currently blacklisted nodes\n")
	_, _ = fmt.Fprintf(w, "# TYPE nxgate_blacklist_count gauge\n")
	_, _ = fmt.Fprintf(w, "nxgate_blacklist_count %d\n", blCount)

	// Backward-compatible Aimili aliases for existing Prometheus scraping configs
	_, _ = fmt.Fprintf(w, "\n# Backward-compatible aimili_* aliases\n")
	_, _ = fmt.Fprintf(w, "aimili_traffic_upload_bytes_total %d\n", tr.TotalUploadBytes)
	_, _ = fmt.Fprintf(w, "aimili_traffic_download_bytes_total %d\n", tr.TotalDownloadBytes)
	_, _ = fmt.Fprintf(w, "aimili_active_connections %d\n", tr.ActiveConnections)
	_, _ = fmt.Fprintf(w, "aimili_vpn_connected %d\n", connectedVal)
}

func (s *Server) writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, code int, msg string) {
	s.writeJSON(w, code, map[string]string{"error": msg})
}

type AppInfoResponse struct {
	OK               bool     `json:"ok"`
	App              string   `json:"app"`
	Version          string   `json:"version"`
	ServerTime       string   `json:"server_time"`
	Status           string   `json:"status"`
	ActiveNode       string   `json:"active_node,omitempty"`
	TunnelsCount     int      `json:"tunnels_count"`
	DownloadSpeedBps int64    `json:"download_speed_bps"`
	UploadSpeedBps   int64    `json:"upload_speed_bps"`
	TotalDownloadMB  int64    `json:"total_download_mb"`
	TotalUploadMB    int64    `json:"total_upload_mb"`
	Capabilities     []string `json:"capabilities"`
}

func (s *Server) handleAppInfo(w http.ResponseWriter, r *http.Request) {
	vpnState := s.vpn.Snapshot()
	traffic := stats.GetTrafficTracker().Snapshot()
	tunnelsCount := len(s.getActiveTunnels())

	capabilities := []string{
		"status",
		"nodes",
		"tunnels",
		"dynamic_groups",
		"port_rules",
		"singbox",
		"sse_events",
		"blacklist",
	}

	resp := AppInfoResponse{
		OK:               true,
		App:              "nxgate",
		Version:          config.Version,
		ServerTime:       time.Now().UTC().Format(time.RFC3339),
		Status:           string(vpnState.Status),
		ActiveNode:       vpnState.ActiveNodeID,
		TunnelsCount:     tunnelsCount,
		DownloadSpeedBps: int64(traffic.DownloadSpeedBps),
		UploadSpeedBps:   int64(traffic.UploadSpeedBps),
		TotalDownloadMB:  int64(traffic.TotalDownloadBytes / (1024 * 1024)),
		TotalUploadMB:    int64(traffic.TotalUploadBytes / (1024 * 1024)),
		Capabilities:     capabilities,
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) activeUIPort() int {
	s.mu.Lock()
	ln := s.listener
	s.mu.Unlock()
	if ln != nil {
		if tcpAddr, ok := ln.Addr().(*net.TCPAddr); ok && tcpAddr.Port > 0 {
			return tcpAddr.Port
		}
	}
	return s.cfg.UIPort
}

type ServerProfile struct {
	Type      string `json:"type"`       // "nxgate_server"
	Version   int    `json:"version"`    // 1
	Name      string `json:"name"`       // display name
	Host      string `json:"host"`       // host IP or domain
	Port      int    `json:"port"`       // UI port
	Path      string `json:"path"`       // UI path
	Username  string `json:"username"`   // UI Username
	Password  string `json:"password"`   // UI Password
	ProxyPort int    `json:"proxy_port"` // Proxy port
	TLS       bool   `json:"tls"`        // whether https
}

type AppProfileResponse struct {
	OK         bool          `json:"ok"`
	Profile    ServerProfile `json:"profile"`
	ConnectURI string        `json:"connect_uri"`
}

func (s *Server) handleAppProfile(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if h, _, err := net.SplitHostPort(r.Host); err == nil {
		host = h
	}
	if host == "" || host == "::" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}

	port := s.activeUIPort()
	path := strings.Trim(s.cfg.UIPath, "/")
	username, password := s.cfg.GetUICredentials()
	tls := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")

	name := "NXGate (" + host + ")"
	if customName := r.URL.Query().Get("name"); customName != "" {
		name = customName
	}

	profile := ServerProfile{
		Type:      "nxgate_server",
		Version:   1,
		Name:      name,
		Host:      host,
		Port:      port,
		Path:      path,
		Username:  username,
		Password:  password,
		ProxyPort: s.cfg.ProxyPort,
		TLS:       tls,
	}

	q := url.Values{}
	q.Set("host", host)
	q.Set("port", strconv.Itoa(port))
	q.Set("path", path)
	q.Set("user", username)
	q.Set("pass", password)
	q.Set("name", name)
	if tls {
		q.Set("tls", "1")
	} else {
		q.Set("tls", "0")
	}
	connectURI := "nxgate://server?" + q.Encode()

	s.writeJSON(w, http.StatusOK, AppProfileResponse{
		OK:         true,
		Profile:    profile,
		ConnectURI: connectURI,
	})
}
