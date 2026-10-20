package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"aimili-vpngate-go/pkg/proxy"
)

type CustomProxyListResponse struct {
	OK      bool                 `json:"ok"`
	Count   int                  `json:"count"`
	Proxies []*proxy.CustomProxy `json:"proxies"`
}

type AddCustomProxyRequest struct {
	URL   string   `json:"url,omitempty"`
	Tag   string   `json:"tag,omitempty"`
	Batch []string `json:"batch,omitempty"`
}

type DeleteCustomProxyRequest struct {
	ID string `json:"id"`
}

type TestCustomProxyRequest struct {
	ID string `json:"id,omitempty"`
}

// GET /api/custom-proxies
func (s *Server) handleGetCustomProxies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	mgr := s.portMgr.GetCustomProxyManager()
	if mgr == nil {
		_ = json.NewEncoder(w).Encode(CustomProxyListResponse{OK: true, Count: 0, Proxies: []*proxy.CustomProxy{}})
		return
	}

	list := mgr.List()
	if list == nil {
		list = []*proxy.CustomProxy{}
	}
	_ = json.NewEncoder(w).Encode(CustomProxyListResponse{
		OK:      true,
		Count:   len(list),
		Proxies: list,
	})
}

// POST /api/custom-proxies
func (s *Server) handleAddCustomProxies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	mgr := s.portMgr.GetCustomProxyManager()
	if mgr == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "custom proxy manager not available"})
		return
	}

	var req AddCustomProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid json payload"})
		return
	}

	var added []*proxy.CustomProxy
	if len(req.Batch) > 0 {
		var errs []error
		added, errs = mgr.BatchImport(req.Batch, req.Tag)
		if len(added) == 0 && len(errs) > 0 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": errs[0].Error()})
			return
		}
	} else if strings.TrimSpace(req.URL) != "" {
		p, err := mgr.AddOrUpdate(req.URL, req.Tag)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}
		added = append(added, p)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "url or batch proxies required"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"msg":     "已成功导入自定义代理",
		"added":   len(added),
		"proxies": mgr.List(),
	})
}

// POST /api/custom-proxies/test
func (s *Server) handleTestCustomProxies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	mgr := s.portMgr.GetCustomProxyManager()
	if mgr == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "custom proxy manager not available"})
		return
	}

	var req TestCustomProxyRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if req.ID != "" {
		p := mgr.Get(req.ID)
		if p == nil {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "proxy not found"})
			return
		}
		p, err := mgr.Probe(ctx, req.ID, 6*time.Second)
		if p == nil {
			s.writeError(w, http.StatusNotFound, "proxy removed during probe")
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":         err == nil,
			"healthy":    p.Healthy,
			"latency_ms": p.LatencyMs,
			"exit_ip":    p.ExitIP,
			"error":      p.LastError,
			"proxy":      p,
		})
		return
	}

	// Probe all
	tested := mgr.ProbeAll(ctx)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"count":   len(tested),
		"proxies": tested,
	})
}

// DELETE /api/custom-proxies
func (s *Server) handleDeleteCustomProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	mgr := s.portMgr.GetCustomProxyManager()
	if mgr == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "custom proxy manager not available"})
		return
	}

	var req DeleteCustomProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "id is required"})
		return
	}

	deleted := mgr.Delete(req.ID)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      deleted,
		"proxies": mgr.List(),
	})
}
