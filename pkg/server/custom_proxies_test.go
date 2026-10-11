package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/nodes"
	"aimili-vpngate-go/pkg/proxy"
	"aimili-vpngate-go/pkg/tunnel"
	"aimili-vpngate-go/pkg/vpn"
)

func TestCustomProxiesAPIEndpoints(t *testing.T) {
	cfg := &config.Config{
		UIPort:     8787,
		UIPath:     "testadmin",
		UIUsername: "admin",
		UIPassword: "password123",
		DataDir:    t.TempDir(),
	}

	pool := nodes.NewNodePool(cfg)
	tunnelPool := tunnel.NewPool(cfg, pool)
	vpnMgr := vpn.NewManager(cfg, pool, tunnelPool)
	dynamicMgr := tunnel.NewDynamicGroupManager(cfg, tunnelPool, pool)
	portMgr := proxy.NewMultiPortManager(cfg, tunnelPool, dynamicMgr)
	srv := NewServer(cfg, pool, vpnMgr, tunnelPool, dynamicMgr, portMgr, nil)

	// 1. Initial GET /api/custom-proxies -> empty list
	reqGet := httptest.NewRequest("GET", "/api/custom-proxies", nil)
	wGet := httptest.NewRecorder()
	srv.handleGetCustomProxies(wGet, reqGet)
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wGet.Code)
	}

	var listResp CustomProxyListResponse
	if err := json.Unmarshal(wGet.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if listResp.Count != 0 {
		t.Fatalf("expected 0 proxies initially, got %d", listResp.Count)
	}

	// 2. POST /api/custom-proxies (single proxy)
	addPayload := `{"url": "socks5://testuser:testpass@1.2.3.4:1080", "tag": "Test Proxy"}`
	reqAdd := httptest.NewRequest("POST", "/api/custom-proxies", bytes.NewReader([]byte(addPayload)))
	wAdd := httptest.NewRecorder()
	srv.handleAddCustomProxies(wAdd, reqAdd)
	if wAdd.Code != http.StatusOK {
		t.Fatalf("expected 200 adding proxy, got %d: %s", wAdd.Code, wAdd.Body.String())
	}

	// 3. POST /api/custom-proxies (batch import)
	batchPayload := `{"batch": ["http://10.0.0.1:8080", "socks5://10.0.0.2:1080"], "tag": "Batch Group"}`
	reqBatch := httptest.NewRequest("POST", "/api/custom-proxies", bytes.NewReader([]byte(batchPayload)))
	wBatch := httptest.NewRecorder()
	srv.handleAddCustomProxies(wBatch, reqBatch)
	if wBatch.Code != http.StatusOK {
		t.Fatalf("expected 200 batch adding proxies, got %d: %s", wBatch.Code, wBatch.Body.String())
	}

	// 4. GET /api/custom-proxies -> verify 3 total proxies
	reqGet2 := httptest.NewRequest("GET", "/api/custom-proxies", nil)
	wGet2 := httptest.NewRecorder()
	srv.handleGetCustomProxies(wGet2, reqGet2)
	var listResp2 CustomProxyListResponse
	_ = json.Unmarshal(wGet2.Body.Bytes(), &listResp2)
	if listResp2.Count != 3 {
		t.Fatalf("expected 3 proxies, got %d", listResp2.Count)
	}

	// 5. DELETE /api/custom-proxies
	delPayload := `{"id": "socks5-1.2.3.4-1080"}`
	reqDel := httptest.NewRequest("DELETE", "/api/custom-proxies", bytes.NewReader([]byte(delPayload)))
	wDel := httptest.NewRecorder()
	srv.handleDeleteCustomProxy(wDel, reqDel)
	if wDel.Code != http.StatusOK {
		t.Fatalf("expected 200 deleting proxy, got %d", wDel.Code)
	}

	// Verify remaining count is 2
	reqGet3 := httptest.NewRequest("GET", "/api/custom-proxies", nil)
	wGet3 := httptest.NewRecorder()
	srv.handleGetCustomProxies(wGet3, reqGet3)
	var listResp3 CustomProxyListResponse
	_ = json.Unmarshal(wGet3.Body.Bytes(), &listResp3)
	if listResp3.Count != 2 {
		t.Fatalf("expected 2 proxies after deletion, got %d", listResp3.Count)
	}
}
