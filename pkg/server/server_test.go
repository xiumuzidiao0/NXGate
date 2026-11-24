package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/nodes"
	"aimili-vpngate-go/pkg/proxy"
	"aimili-vpngate-go/pkg/tunnel"
	"aimili-vpngate-go/pkg/vpn"
)

func TestServerAPI(t *testing.T) {
	cfg := &config.Config{
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

	// Test Status endpoint handler directly
	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()
	srv.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp StatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse status json: %v", err)
	}
	if resp.AdminPath != "testadmin" {
		t.Fatalf("expected admin path testadmin, got %s", resp.AdminPath)
	}

	// Test Metrics endpoint
	reqMetrics := httptest.NewRequest("GET", "/metrics", nil)
	wMetrics := httptest.NewRecorder()
	srv.handleMetrics(wMetrics, reqMetrics)

	if wMetrics.Code != http.StatusOK {
		t.Fatalf("expected metrics status 200, got %d", wMetrics.Code)
	}
	metricsBody := wMetrics.Body.String()
	if !strings.Contains(metricsBody, "aimili_traffic_upload_bytes_total") {
		t.Fatalf("metrics body missing upload counter: %s", metricsBody)
	}

	// Test Get Settings
	reqSettings := httptest.NewRequest("GET", "/api/settings", nil)
	wSettings := httptest.NewRecorder()
	srv.handleGetSettings(wSettings, reqSettings)
	if wSettings.Code != http.StatusOK {
		t.Fatalf("expected settings 200, got %d", wSettings.Code)
	}

	// Test Update Settings
	updateBody := `{"ui_username":"newadmin","ui_path":"newsecret","ui_port":8888,"proxy_port":7999}`
	reqUpdate := httptest.NewRequest("POST", "/api/settings", strings.NewReader(updateBody))
	wUpdate := httptest.NewRecorder()
	srv.handleUpdateSettings(wUpdate, reqUpdate)
	if wUpdate.Code != http.StatusOK {
		t.Fatalf("expected update settings 200, got %d", wUpdate.Code)
	}
	var updateResp struct {
		RestartRequired bool `json:"restart_required"`
	}
	if err := json.Unmarshal(wUpdate.Body.Bytes(), &updateResp); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if !updateResp.RestartRequired {
		t.Fatalf("expected restart_required=true when ports change, body=%s", wUpdate.Body.String())
	}
	if srv.cfg.GetSettings().UIUsername != "newadmin" {
		t.Fatalf("expected newadmin, got %s", srv.cfg.GetSettings().UIUsername)
	}

	// Test Save Dynamic Tunnel Group
	groupJSON := `{"id":"test-dg","name":"日本Top3住宅组","enabled":true,"country":"JP","ip_type":"residential","sort_by":"latency","target_count":3,"interval_minutes":15}`
	reqGroup := httptest.NewRequest("POST", "/api/tunnel-groups", strings.NewReader(groupJSON))
	wGroup := httptest.NewRecorder()
	srv.handleSaveTunnelGroup(wGroup, reqGroup)
	if wGroup.Code != http.StatusOK {
		t.Fatalf("expected save group 200, got %d", wGroup.Code)
	}

	// Test List Dynamic Groups
	reqListGroups := httptest.NewRequest("GET", "/api/tunnel-groups", nil)
	wListGroups := httptest.NewRecorder()
	srv.handleListTunnelGroups(wListGroups, reqListGroups)
	if wListGroups.Code != http.StatusOK {
		t.Fatalf("expected list groups 200, got %d", wListGroups.Code)
	}

	// Test sing-box status endpoint
	reqSBStatus := httptest.NewRequest("GET", "/api/singbox/status", nil)
	wSBStatus := httptest.NewRecorder()
	srv.handleSingBoxStatus(wSBStatus, reqSBStatus)
	if wSBStatus.Code != http.StatusOK {
		t.Fatalf("expected singbox status 200, got %d", wSBStatus.Code)
	}

	// Test sing-box protocols endpoint
	reqSBProtos := httptest.NewRequest("GET", "/api/singbox/protocols", nil)
	wSBProtos := httptest.NewRecorder()
	srv.handleSingBoxProtocols(wSBProtos, reqSBProtos)
	if wSBProtos.Code != http.StatusOK {
		t.Fatalf("expected singbox protocols 200, got %d", wSBProtos.Code)
	}

	// Test sing-box overview endpoint
	reqSBOverview := httptest.NewRequest("GET", "/api/singbox/overview", nil)
	wSBOverview := httptest.NewRecorder()
	srv.handleSingBoxOverview(wSBOverview, reqSBOverview)
	if wSBOverview.Code != http.StatusOK {
		t.Fatalf("expected singbox overview 200, got %d", wSBOverview.Code)
	}
	var overviewResp SingBoxOverviewResponse
	if err := json.Unmarshal(wSBOverview.Body.Bytes(), &overviewResp); err != nil {
		t.Fatalf("failed to parse overview json: %v", err)
	}
	if !overviewResp.OK || len(overviewResp.AvailableOutbounds) == 0 {
		t.Fatalf("expected overview ok and outbounds > 0, got: %+v", overviewResp)
	}

	// Test resolveOutboundURL
	directURL := srv.resolveOutboundURL("direct")
	if directURL != "direct" {
		t.Fatalf("expected direct, got %s", directURL)
	}

	socksURL := srv.resolveOutboundURL("7928")
	if !strings.HasPrefix(socksURL, "socks5://") || !strings.Contains(socksURL, "7928") {
		t.Fatalf("expected socks5 proxy url with 7928, got %s", socksURL)
	}

	httpExplicitURL := srv.resolveOutboundURL("http://127.0.0.1:7928")
	if !strings.HasPrefix(httpExplicitURL, "http://") || !strings.Contains(httpExplicitURL, "7928") {
		t.Fatalf("expected http proxy url with 7928, got %s", httpExplicitURL)
	}

	// Test Blacklist Add, List, Remove, Clear
	addBody := `{"node_id":"1.2.3.4:443","ip":"1.2.3.4","country":"JP","reason":"测试屏蔽","duration_minutes":60}`
	reqBLAdd := httptest.NewRequest("POST", "/api/blacklist/add", strings.NewReader(addBody))
	wBLAdd := httptest.NewRecorder()
	srv.handleBlacklistAdd(wBLAdd, reqBLAdd)
	if wBLAdd.Code != http.StatusOK {
		t.Fatalf("expected blacklist add 200, got %d", wBLAdd.Code)
	}

	reqBLList := httptest.NewRequest("GET", "/api/blacklist", nil)
	wBLList := httptest.NewRecorder()
	srv.handleBlacklist(wBLList, reqBLList)
	if wBLList.Code != http.StatusOK || !strings.Contains(wBLList.Body.String(), "1.2.3.4:443") {
		t.Fatalf("expected blacklist list contains 1.2.3.4:443, got %s", wBLList.Body.String())
	}

	rmBody := `{"node_id":"1.2.3.4:443"}`
	reqBLRm := httptest.NewRequest("POST", "/api/blacklist/remove", strings.NewReader(rmBody))
	wBLRm := httptest.NewRecorder()
	srv.handleBlacklistRemove(wBLRm, reqBLRm)
	if wBLRm.Code != http.StatusOK {
		t.Fatalf("expected blacklist remove 200, got %d", wBLRm.Code)
	}

	// Test handleConnect with Mode sstp and kernel
	srv.pool.SetCandidatesForTest([]*nodes.Node{
		{
			ID:           "test-node-sstp",
			IP:           "1.2.3.4",
			CountryShort: "JP",
		},
	})
	connectJSON := `{"node_id":"test-node-sstp","mode":"sstp"}`
	reqConn := httptest.NewRequest("POST", "/api/connect", strings.NewReader(connectJSON))
	wConn := httptest.NewRecorder()
	srv.handleConnect(wConn, reqConn)
	if wConn.Code != http.StatusOK {
		t.Fatalf("expected 200 for handleConnect sstp, got %d", wConn.Code)
	}
	if !strings.Contains(wConn.Body.String(), `"driver_mode":"userland"`) {
		t.Fatalf("expected userland driver mode in response, got: %s", wConn.Body.String())
	}

	reqBLClear := httptest.NewRequest("POST", "/api/blacklist/clear", nil)
	wBLClear := httptest.NewRecorder()
	srv.handleBlacklistClear(wBLClear, reqBLClear)
	if wBLClear.Code != http.StatusOK {
		t.Fatalf("expected blacklist clear 200, got %d", wBLClear.Code)
	}

	// Wait for the async handleConnect goroutine to register the connecting
	// epoch, then cancel and await background teardown before TempDir cleanup.
	deadline := time.Now().Add(2 * time.Second)
	for srv.vpn.GetStatus() == string(vpn.StatusDisconnected) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	portMgr.StopAll()
	vpnMgr.Disconnect("test cleanup")
	vpnMgr.Wait()
}

func TestBasicAuthMiddleware(t *testing.T) {
	cfg := &config.Config{
		UIUsername: "myuser",
		UIPassword: "mypassword",
	}
	mw := NewMiddleware(cfg)

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler := mw.BasicAuth(dummyHandler)

	// 1. Unauthenticated request -> 401
	req1 := httptest.NewRequest("GET", "/", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	if w1.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 unauthorized, got %d", w1.Code)
	}

	// 2. Authenticated request -> 200
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.SetBasicAuth("myuser", "mypassword")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 ok, got %d", w2.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	mw := NewMiddleware(&config.Config{})
	dummy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := mw.SecurityHeaders(dummy)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("missing nosniff header")
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("missing frame options header")
	}
	if w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Errorf("missing referrer policy header")
	}
}

func TestSecretPathStealth404(t *testing.T) {
	cfg := &config.Config{
		UIPath:     "mysecret",
		UIUsername: "admin",
		UIPassword: "secretpassword",
	}
	mw := NewMiddleware(cfg)
	dummy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("inside"))
	})
	handler := mw.SecretPathGuard(dummy)

	// 1. Random scanner accessing root -> 404
	reqRoot := httptest.NewRequest("GET", "/", nil)
	wRoot := httptest.NewRecorder()
	handler.ServeHTTP(wRoot, reqRoot)
	if wRoot.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unauthenticated root scanner, got %d", wRoot.Code)
	}

	// 2. Unauthenticated scanner probing direct /api/status -> stealth 404 (zero leak of AimiliVPN)
	reqAPI := httptest.NewRequest("GET", "/api/status", nil)
	wAPI := httptest.NewRecorder()
	handler.ServeHTTP(wAPI, reqAPI)
	if wAPI.Code != http.StatusNotFound {
		t.Fatalf("expected stealth 404 for unauthenticated API scanner, got %d", wAPI.Code)
	}

	// 3. Exact secret without slash -> 302 redirect to /mysecret/
	reqExact := httptest.NewRequest("GET", "/mysecret", nil)
	wExact := httptest.NewRecorder()
	handler.ServeHTTP(wExact, reqExact)
	if wExact.Code != http.StatusFound || wExact.Header().Get("Location") != "/mysecret/" {
		t.Fatalf("expected 302 redirect to /mysecret/, got %d", wExact.Code)
	}

	// 4. Scoped secret path /mysecret/ -> 200
	reqSecret := httptest.NewRequest("GET", "/mysecret/", nil)
	wSecret := httptest.NewRecorder()
	handler.ServeHTTP(wSecret, reqSecret)
	if wSecret.Code != http.StatusOK || wSecret.Body.String() != "inside" {
		t.Fatalf("expected 200 inside secret path, got %d", wSecret.Code)
	}

	// 5. Scoped secret path /mysecret/api/nodes -> 200
	reqSecretAPI := httptest.NewRequest("GET", "/mysecret/api/nodes", nil)
	wSecretAPI := httptest.NewRecorder()
	handler.ServeHTTP(wSecretAPI, reqSecretAPI)
	if wSecretAPI.Code != http.StatusOK || wSecretAPI.Body.String() != "inside" {
		t.Fatalf("expected 200 inside /mysecret/api/nodes, got %d", wSecretAPI.Code)
	}
}

func TestSubscriptionBypassUnderSecretPath(t *testing.T) {
	cfg := &config.Config{
		UIPath:     "enter",
		UIUsername: "admin",
		UIPassword: "password123",
	}
	mw := NewMiddleware(cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/singbox/subscription/clash", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("clash-yaml-content"))
	})
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("status-content"))
	})

	// Wrap middleware stack exactly as in server.go:
	// BasicAuth wraps mux, SecretPathGuard wraps BasicAuth
	handler := mw.SecretPathGuard(mw.BasicAuth(mux))

	// 1. Clash client fetching /enter/api/singbox/subscription/clash without BasicAuth -> 200 OK!
	reqClash := httptest.NewRequest("GET", "/enter/api/singbox/subscription/clash", nil)
	wClash := httptest.NewRecorder()
	handler.ServeHTTP(wClash, reqClash)
	if wClash.Code != http.StatusOK || wClash.Body.String() != "clash-yaml-content" {
		t.Fatalf("expected 200 for Clash subscription under secret path without BasicAuth, got %d, body: %s", wClash.Code, wClash.Body.String())
	}

	// 2. Direct /api/singbox/subscription/clash with valid token parameter -> 200 OK!
	reqToken := httptest.NewRequest("GET", "/api/singbox/subscription/clash?token=enter", nil)
	wToken := httptest.NewRecorder()
	handler.ServeHTTP(wToken, reqToken)
	if wToken.Code != http.StatusOK || wToken.Body.String() != "clash-yaml-content" {
		t.Fatalf("expected 200 for Clash subscription with valid token, got %d", wToken.Code)
	}

	// 3. /enter/api/status WITHOUT BasicAuth -> MUST still be 401 Unauthorized!
	reqStatus := httptest.NewRequest("GET", "/enter/api/status", nil)
	wStatus := httptest.NewRecorder()
	handler.ServeHTTP(wStatus, reqStatus)
	if wStatus.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for admin status API without BasicAuth, got %d", wStatus.Code)
	}

	// 4. /api/status without secret path or BasicAuth -> MUST still be stealth 404!
	reqScanner := httptest.NewRequest("GET", "/api/status", nil)
	wScanner := httptest.NewRecorder()
	handler.ServeHTTP(wScanner, reqScanner)
	if wScanner.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for scanner probing /api/status, got %d", wScanner.Code)
	}
}

func TestHandleUpdateSettingsClearsPathAndAuthExplicitly(t *testing.T) {
	cfg := &config.Config{
		DataDir:    t.TempDir(),
		UIHost:     "0.0.0.0",
		UIPath:     "secret",
		UIUsername: "admin",
		UIPassword: "oldpassword",
		UIPort:     8787,
		ProxyPort:  7928,
	}
	srv := &Server{cfg: cfg}

	// Half-empty credentials must fail closed with HTTP 400.
	wBad := httptest.NewRecorder()
	srv.handleUpdateSettings(wBad, httptest.NewRequest("POST", "/api/settings", strings.NewReader(`{"ui_username":""}`)))
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for half-empty auth update, got %d (%s)", wBad.Code, wBad.Body.String())
	}

	// Clearing ui_path while keeping existing credentials works without sending ui_password.
	wPath := httptest.NewRecorder()
	srv.handleUpdateSettings(wPath, httptest.NewRequest("POST", "/api/settings", strings.NewReader(`{"ui_path":""}`)))
	if wPath.Code != http.StatusOK || cfg.UIPath != "" || !cfg.IsUIAuthEnabled() {
		t.Fatalf("expected path cleared & auth preserved, code=%d path=%q auth=%v body=%s", wPath.Code, cfg.UIPath, cfg.IsUIAuthEnabled(), wPath.Body.String())
	}

	// Disabling auth explicitly succeeds and removes login gate.
	wDisable := httptest.NewRecorder()
	srv.handleUpdateSettings(wDisable, httptest.NewRequest("POST", "/api/settings", strings.NewReader(`{"disable_ui_auth":true}`)))
	if wDisable.Code != http.StatusOK || cfg.IsUIAuthEnabled() || !cfg.IsPublicUnauthenticated() {
		t.Fatalf("expected auth disabled & public exposure true, code=%d auth=%v public=%v body=%s", wDisable.Code, cfg.IsUIAuthEnabled(), cfg.IsPublicUnauthenticated(), wDisable.Body.String())
	}
}

func TestHandleNodesScopeAndLiveSettingsRebuild(t *testing.T) {
	cfg := &config.Config{
		DataDir:             t.TempDir(),
		UIPort:              8787,
		ProxyPort:           7928,
		NodeHotPoolSize:     50,
		NodeStoreMaxSize:    600,
		NodeMinPerCountry:   0,
		NodeMajorCountryCap: 2,
	}
	pool := nodes.NewNodePool(cfg)
	tunnelPool := tunnel.NewPool(cfg, pool)
	vpnMgr := vpn.NewManager(cfg, pool, tunnelPool)
	srv := &Server{
		cfg:        cfg,
		pool:       pool,
		tunnelPool: tunnelPool,
		vpn:        vpnMgr,
	}

	var batch []*nodes.Node
	for i := 1; i <= 5; i++ {
		batch = append(batch, &nodes.Node{
			ID:           fmt.Sprintf("10.8.0.%d:443", i),
			IP:           fmt.Sprintf("10.8.0.%d", i),
			Port:         443,
			CountryShort: "JP",
			Score:        int64(1000 - i),
			LatencyMs:    20,
		})
	}
	pool.SetCandidatesForTest(batch)
	pool.RebuildCandidates()

	// Default scope=active reflects major_country_cap=2.
	wActive := httptest.NewRecorder()
	srv.handleNodes(wActive, httptest.NewRequest("GET", "/api/nodes", nil))
	var activeNodes []nodes.Node
	if err := json.Unmarshal(wActive.Body.Bytes(), &activeNodes); err != nil {
		t.Fatalf("decode active nodes: %v", err)
	}
	if len(activeNodes) != 2 {
		t.Fatalf("expected 2 active nodes with major_country_cap=2, got %d", len(activeNodes))
	}

	// scope=all exposes all 5 nodes with accurate InActivePool flags.
	wAll := httptest.NewRecorder()
	srv.handleNodes(wAll, httptest.NewRequest("GET", "/api/nodes?scope=all", nil))
	var allNodes []nodes.Node
	if err := json.Unmarshal(wAll.Body.Bytes(), &allNodes); err != nil {
		t.Fatalf("decode all nodes: %v", err)
	}
	if len(allNodes) != 5 {
		t.Fatalf("expected 5 total stored nodes via scope=all, got %d", len(allNodes))
	}

	// Updating settings to disable major cap (0) and expand pool (-1) immediately rebuilds /api/nodes.
	wSet := httptest.NewRecorder()
	srv.handleUpdateSettings(wSet, httptest.NewRequest("POST", "/api/settings", strings.NewReader(`{"node_hot_pool_size":-1,"node_major_country_cap":0}`)))
	if wSet.Code != http.StatusOK {
		t.Fatalf("expected 200 updating pool settings, got %d (%s)", wSet.Code, wSet.Body.String())
	}

	wActiveAfter := httptest.NewRecorder()
	srv.handleNodes(wActiveAfter, httptest.NewRequest("GET", "/api/nodes", nil))
	var rebuilt []nodes.Node
	if err := json.Unmarshal(wActiveAfter.Body.Bytes(), &rebuilt); err != nil {
		t.Fatalf("decode rebuilt active nodes: %v", err)
	}
	if len(rebuilt) != 5 {
		t.Fatalf("expected 5 active nodes immediately after live settings update, got %d", len(rebuilt))
	}

	// /api/status exposes pool limits & counts.
	status := srv.buildStatusResponse()
	if status.NodeCount != 5 || status.StoredAvailableCount != 5 || status.HotPoolLimit != -1 || status.MajorCountryCap != 0 {
		t.Fatalf("unexpected status pool metadata: %+v", status)
	}
}
