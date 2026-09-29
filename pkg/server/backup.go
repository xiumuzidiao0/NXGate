package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/nodes"
	"aimili-vpngate-go/pkg/proxy"
	"aimili-vpngate-go/pkg/tunnel"
)

type GatewayBackupPackage struct {
	Version       string                  `json:"version"`
	ExportedAt    time.Time               `json:"exported_at"`
	ServerName    string                  `json:"server_name,omitempty"`
	Settings      *config.SettingsDTO     `json:"settings,omitempty"`
	DynamicGroups []*tunnel.DynamicGroup  `json:"dynamic_groups,omitempty"`
	PortRules     []proxy.PortRule        `json:"port_rules,omitempty"`
	Favorites     []string                `json:"favorites,omitempty"`
	Blacklist     []*nodes.BlacklistEntry `json:"blacklist,omitempty"`
}

func (s *Server) handleExportBackup(w http.ResponseWriter, r *http.Request) {
	pkg := GatewayBackupPackage{
		Version:    config.Version,
		ExportedAt: time.Now().UTC(),
		ServerName: s.cfg.UIHost,
	}

	// 1. Settings
	settings := s.cfg.GetSettings()
	pkg.Settings = &settings

	// 2. Dynamic Groups
	if s.dynamicMgr != nil {
		pkg.DynamicGroups = s.dynamicMgr.ListGroups()
	}

	// 3. Port Rules
	if s.portMgr != nil {
		pkg.PortRules = s.portMgr.GetRules()
	}

	// 4. Favorites
	if s.pool != nil && s.pool.Favorites() != nil {
		pkg.Favorites = s.pool.Favorites().List()
	}

	// 5. Blacklist
	if s.pool != nil && s.pool.Blacklist() != nil {
		pkg.Blacklist = s.pool.Blacklist().List()
	}

	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("生成备份包失败: %v", err))
		return
	}

	filename := fmt.Sprintf("nxgate-backup-%s.json", time.Now().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) handleImportBackup(w http.ResponseWriter, r *http.Request) {
	var pkg GatewayBackupPackage
	if err := json.NewDecoder(r.Body).Decode(&pkg); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("备份文件解析失败: %v", err))
		return
	}

	restoredCount := 0

	// 1. Restore Settings if provided
	if pkg.Settings != nil {
		if err := s.cfg.UpdateSettings(*pkg.Settings); err == nil {
			restoredCount++
		}
	}

	// 2. Restore Favorites if provided
	if s.pool != nil && s.pool.Favorites() != nil && len(pkg.Favorites) > 0 {
		for _, id := range pkg.Favorites {
			s.pool.Favorites().Add(id)
		}
		restoredCount++
	}

	// 3. Restore Blacklist if provided
	if s.pool != nil && s.pool.Blacklist() != nil && len(pkg.Blacklist) > 0 {
		for _, b := range pkg.Blacklist {
			if b.IsPermanent {
				s.pool.Blacklist().MarkManualWithOptions(b.ID, b.IP, b.Country, b.Reason, 0, b.Scope, true)
			} else {
				s.pool.Blacklist().MarkManualWithOptions(b.ID, b.IP, b.Country, b.Reason, 24*time.Hour, b.Scope, false)
			}
		}
		restoredCount++
	}

	// 4. Restore Dynamic Groups if provided
	if s.dynamicMgr != nil && len(pkg.DynamicGroups) > 0 {
		for _, g := range pkg.DynamicGroups {
			_ = s.dynamicMgr.SaveGroup(g)
		}
		restoredCount++
	}

	// 5. Restore Port Rules if provided
	if s.portMgr != nil && len(pkg.PortRules) > 0 {
		_ = s.portMgr.ApplyRules(pkg.PortRules)
		restoredCount++
	}

	// Rebuild candidates to reflect restored blacklists and favorites
	if s.pool != nil {
		s.pool.RebuildCandidates()
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok":             true,
		"message":        "全量配置已成功导入并生效！",
		"restored_items": restoredCount,
		"version":        pkg.Version,
	})
}
