package server

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/proxy"
	"aimili-vpngate-go/pkg/singbox"
	"aimili-vpngate-go/pkg/tunnel"
)

func TestGenerateRawAndBase64Subscription(t *testing.T) {
	nodes := []singbox.Node{
		{
			Name:        "Test-Reality",
			Protocol:    "VLESS-REALITY",
			RawProtocol: "vless",
			Port:        443,
			Address:     "auto",
			UUID:        "11111111-2222-3333-4444-555555555555",
			PBK:         "mypublickey123",
			SNI:         "www.apple.com",
			Flow:        "xtls-rprx-vision",
			URL:         "vless://11111111-2222-3333-4444-555555555555@auto:443?encryption=none&flow=xtls-rprx-vision&security=reality&sni=www.apple.com&fp=chrome&pbk=mypublickey123&type=tcp#Test-Reality",
		},
		{
			Name:        "Test-Hy2",
			Protocol:    "Hysteria2",
			RawProtocol: "hysteria2",
			Port:        8443,
			Address:     "auto",
			Password:    "secretpass",
			SNI:         "auto",
			URL:         "hysteria2://secretpass@auto:8443?insecure=1&sni=auto#Test-Hy2",
		},
	}

	serverHost := "203.0.113.10"

	// 1. Test Raw URLs
	rawURLs := GenerateRawSubscription(nodes, serverHost)
	if len(rawURLs) != 2 {
		t.Fatalf("expected 2 URLs, got %d", len(rawURLs))
	}
	if strings.Contains(rawURLs[0], "@auto:") || !strings.Contains(rawURLs[0], "@203.0.113.10:443") {
		t.Errorf("expected auto replaced with serverHost in %s", rawURLs[0])
	}
	if strings.Contains(rawURLs[1], "@auto:") || !strings.Contains(rawURLs[1], "@203.0.113.10:8443") {
		t.Errorf("expected auto replaced with serverHost in %s", rawURLs[1])
	}

	// 2. Test Base64 content
	b64 := GenerateBase64Subscription(nodes, serverHost)
	decodedBytes, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("failed to decode generated base64: %v", err)
	}
	decodedStr := string(decodedBytes)
	if !strings.Contains(decodedStr, rawURLs[0]) || !strings.Contains(decodedStr, rawURLs[1]) {
		t.Errorf("decoded base64 string does not contain expected URLs: %s", decodedStr)
	}
}

func TestHandleSingBoxSubscriptionEndpoints(t *testing.T) {
	cfg := &config.Config{
		UIHost: "198.51.100.5",
		UIPort: 8787,
		UIPath: "enter",
	}
	s := &Server{
		cfg:           cfg,
		singboxClient: singbox.NewClient(),
	}

	// 1. Request raw text format
	reqRaw := httptest.NewRequest("GET", "/enter/api/singbox/subscription?format=raw", nil)
	wRaw := httptest.NewRecorder()
	s.handleSingBoxGetSub(wRaw, reqRaw)

	if wRaw.Code != http.StatusOK {
		t.Fatalf("expected 200 for raw subscription, got %d", wRaw.Code)
	}
	if wRaw.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Errorf("expected text/plain, got %s", wRaw.Header().Get("Content-Type"))
	}

	// 2. Request Clash format
	reqClash := httptest.NewRequest("GET", "/enter/api/singbox/subscription?format=clash", nil)
	wClash := httptest.NewRecorder()
	s.handleSingBoxGetSub(wClash, reqClash)

	if wClash.Code != http.StatusOK {
		t.Fatalf("expected 200 for clash subscription, got %d", wClash.Code)
	}
	if !strings.Contains(wClash.Header().Get("Content-Type"), "yaml") {
		t.Errorf("expected yaml, got %s", wClash.Header().Get("Content-Type"))
	}

	// 3. Request Base64 (Shadowrocket UA)
	reqUA := httptest.NewRequest("GET", "/enter/api/singbox/subscription", nil)
	reqUA.Header.Set("User-Agent", "Shadowrocket/1982")
	wUA := httptest.NewRecorder()
	s.handleSingBoxGetSub(wUA, reqUA)

	if wUA.Code != http.StatusOK {
		t.Fatalf("expected 200 for Shadowrocket subscription, got %d", wUA.Code)
	}

	// 4. Request JSON metadata
	reqJSON := httptest.NewRequest("GET", "/enter/api/singbox/subscription?format=json", nil)
	reqJSON.Header.Set("Accept", "application/json")
	wJSON := httptest.NewRecorder()
	s.handleSingBoxGetSub(wJSON, reqJSON)

	if wJSON.Code != http.StatusOK {
		t.Fatalf("expected 200 for JSON subscription, got %d", wJSON.Code)
	}
	body := wJSON.Body.String()
	if !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"sub_url"`) {
		t.Fatalf("expected valid JSON response with sub_url, got %s", body)
	}
}

func TestUniversalSubscriptionWithSafeRandomToken(t *testing.T) {
	cfg := &config.Config{
		UIHost:     "198.51.100.5",
		UIPort:     9999,
		UIPath:     "mysecret",
		UIUsername: "admin",
		UIPassword: "password123",
		SubToken:   "randtoken7890abc",
	}

	s := &Server{
		cfg:           cfg,
		singboxClient: singbox.NewClient(),
	}

	// 1. Verify URL generation includes WebUI port and random safe token
	req := httptest.NewRequest("GET", "http://myvps.com:9999/mysecret/api/singbox/overview", nil)
	genericURL := s.buildGenericSubURL(req)
	clashURL := s.buildClashSubURL(req)

	if !strings.Contains(genericURL, ":9999/randtoken7890abc/api/singbox/subscription") {
		t.Fatalf("expected generic subscription URL on WebUI port 9999 with safe token, got: %s", genericURL)
	}
	if !strings.Contains(clashURL, ":9999/randtoken7890abc/api/singbox/subscription/clash") {
		t.Fatalf("expected Clash subscription URL on WebUI port 9999 with safe token, got: %s", clashURL)
	}

	// 2. Setup full middleware stack to test external client fetching
	mw := NewMiddleware(cfg)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/singbox/subscription", s.handleSingBoxGetSub)
	mux.HandleFunc("GET /api/singbox/subscription/clash", s.handleSingBoxClashSub)

	handler := mw.SecretPathGuard(mw.BasicAuth(mux))

	// 2a. Universal subscription via safe token path: /randtoken7890abc/api/singbox/subscription
	reqSub := httptest.NewRequest("GET", "/randtoken7890abc/api/singbox/subscription", nil)
	reqSub.Header.Set("User-Agent", "Shadowrocket/2.2.0")
	wSub := httptest.NewRecorder()
	handler.ServeHTTP(wSub, reqSub)
	if wSub.Code != http.StatusOK {
		t.Fatalf("expected 200 for universal subscription via safe token path, got %d, body: %s", wSub.Code, wSub.Body.String())
	}

	// 2b. Shorthand subscription path: /sub/randtoken7890abc
	reqShort := httptest.NewRequest("GET", "/sub/randtoken7890abc", nil)
	reqShort.Header.Set("User-Agent", "v2rayN/6.0")
	wShort := httptest.NewRecorder()
	handler.ServeHTTP(wShort, reqShort)
	if wShort.Code != http.StatusOK {
		t.Fatalf("expected 200 for shorthand subscription /sub/<token>, got %d", wShort.Code)
	}

	// 2c. Shorthand Clash path: /sub/randtoken7890abc/clash
	reqShortClash := httptest.NewRequest("GET", "/sub/randtoken7890abc/clash", nil)
	wShortClash := httptest.NewRecorder()
	handler.ServeHTTP(wShortClash, reqShortClash)
	if wShortClash.Code != http.StatusOK || !strings.Contains(wShortClash.Header().Get("Content-Type"), "yaml") {
		t.Fatalf("expected 200 YAML for /sub/<token>/clash, got %d", wShortClash.Code)
	}

	// 2d. Invalid token -> MUST be rejected (404 stealth)
	reqBad := httptest.NewRequest("GET", "/invalidtoken/api/singbox/subscription", nil)
	wBad := httptest.NewRecorder()
	handler.ServeHTTP(wBad, reqBad)
	if wBad.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for invalid token probe, got %d", wBad.Code)
	}
}

func TestAgeSubscriptionToggleAndEncryption(t *testing.T) {
	secX, pubX, _, err := GenerateAgeKeyPair("x25519")
	if err != nil {
		t.Fatalf("failed to generate x25519 key: %v", err)
	}

	cfg := &config.Config{
		UIHost:            "198.51.100.5",
		UIPort:            8787,
		UIPath:            "enter",
		AgeEncryptEnabled: false,
		AgePublicKey:      pubX,
	}

	s := &Server{
		cfg:           cfg,
		singboxClient: singbox.NewClient(),
	}

	// 1. When AgeEncryptEnabled is FALSE -> returns unencrypted normal subscriptions
	reqClashOff := httptest.NewRequest("GET", "/enter/api/singbox/subscription?format=clash", nil)
	wClashOff := httptest.NewRecorder()
	s.handleSingBoxGetSub(wClashOff, reqClashOff)
	if wClashOff.Code != http.StatusOK {
		t.Fatalf("expected 200 for clash subscription, got %d", wClashOff.Code)
	}
	bodyClashOff := wClashOff.Body.String()
	if strings.Contains(bodyClashOff, "BEGIN AGE ENCRYPTED FILE") {
		t.Fatalf("expected unencrypted content when age is disabled, got encrypted")
	}

	reqRawOff := httptest.NewRequest("GET", "/enter/api/singbox/subscription?format=raw", nil)
	wRawOff := httptest.NewRecorder()
	s.handleSingBoxGetSub(wRawOff, reqRawOff)
	bodyRawOff := wRawOff.Body.String()
	if strings.Contains(bodyRawOff, "BEGIN AGE ENCRYPTED FILE") {
		t.Fatalf("expected unencrypted raw content when age is disabled, got encrypted")
	}

	// 2. When AgeEncryptEnabled is TRUE -> returns armored age encrypted file
	cfg.AgeEncryptEnabled = true

	// 2a. Clash format encrypted
	reqClashOn := httptest.NewRequest("GET", "/enter/api/singbox/subscription?format=clash", nil)
	wClashOn := httptest.NewRecorder()
	s.handleSingBoxGetSub(wClashOn, reqClashOn)
	if wClashOn.Code != http.StatusOK {
		t.Fatalf("expected 200 for encrypted clash, got %d", wClashOn.Code)
	}
	bodyClashOn := wClashOn.Body.Bytes()
	if !bytes.HasPrefix(bodyClashOn, []byte("-----BEGIN AGE ENCRYPTED FILE-----")) {
		t.Fatalf("expected armor header for encrypted clash subscription, got:\n%s", string(bodyClashOn))
	}

	// Decrypt and verify recovered clash content
	decClash, err := DecryptWithAge(bodyClashOn, secX)
	if err != nil {
		t.Fatalf("failed to decrypt clash subscription: %v", err)
	}
	if string(decClash) != bodyClashOff {
		t.Fatalf("decrypted clash content mismatch with original unencrypted content")
	}

	// 2b. Raw format encrypted
	reqRawOn := httptest.NewRequest("GET", "/enter/api/singbox/subscription?format=raw", nil)
	wRawOn := httptest.NewRecorder()
	s.handleSingBoxGetSub(wRawOn, reqRawOn)
	bodyRawOn := wRawOn.Body.Bytes()
	if !bytes.HasPrefix(bodyRawOn, []byte("-----BEGIN AGE ENCRYPTED FILE-----")) {
		t.Fatalf("expected armor header for encrypted raw subscription, got:\n%s", string(bodyRawOn))
	}

	decRaw, err := DecryptWithAge(bodyRawOn, secX)
	if err != nil {
		t.Fatalf("failed to decrypt raw subscription: %v", err)
	}
	if string(decRaw) != bodyRawOff {
		t.Fatalf("decrypted raw content mismatch with original unencrypted content")
	}

	// 3. Dynamic recipient parameter (e.g. MLKEM768-X25519) on-the-fly
	cfg.AgeEncryptEnabled = false // global switch off
	secPQ, pubPQ, _, err := GenerateAgeKeyPair("mlkem768-x25519")
	if err != nil {
		t.Fatalf("failed to generate hybrid key: %v", err)
	}

	reqPQ := httptest.NewRequest("GET", "/enter/api/singbox/subscription?format=clash&age_recipient="+pubPQ, nil)
	wPQ := httptest.NewRecorder()
	s.handleSingBoxGetSub(wPQ, reqPQ)
	bodyPQ := wPQ.Body.Bytes()
	if !bytes.HasPrefix(bodyPQ, []byte("-----BEGIN AGE ENCRYPTED FILE-----")) {
		t.Fatalf("expected hybrid armor header when age_recipient query param is passed")
	}

	decPQ, err := DecryptWithAge(bodyPQ, secPQ)
	if err != nil {
		t.Fatalf("failed to decrypt hybrid encrypted subscription: %v", err)
	}
	if string(decPQ) != bodyClashOff {
		t.Fatalf("decrypted hybrid content mismatch with original clash content")
	}
}

func TestSubscriptionGroupPrefixAndFiltering(t *testing.T) {
	cfg := &config.Config{
		DataDir:   t.TempDir(),
		ProxyPort: 7928,
		UIHost:    "127.0.0.1",
		UIPort:    8787,
	}

	dm := tunnel.NewDynamicGroupManager(cfg, nil, nil)
	pm := proxy.NewMultiPortManager(cfg, nil, dm)

	// Create dynamic groups
	_ = dm.SaveGroup(&tunnel.DynamicGroup{
		ID:      "dg-japan",
		Name:    "日本高速专线",
		Enabled: true,
		Country: "JP",
	})
	_ = dm.SaveGroup(&tunnel.DynamicGroup{
		ID:      "dg-fav",
		Name:    "我的收藏出口组",
		Enabled: true,
		Country: "FAVORITES",
	})

	// Bind ports
	_ = pm.ApplyRules([]proxy.PortRule{
		{Port: 1081, Enabled: true, BoundGroupIDs: []string{"dg-japan"}},
		{Port: 1082, Enabled: true, BoundGroupIDs: []string{"dg-fav"}},
	})

	s := &Server{
		cfg:        cfg,
		portMgr:    pm,
		dynamicMgr: dm,
	}

	testNodes := []singbox.Node{
		{
			Name:         "[旧分组] reality-443.json",
			Protocol:     "VLESS-REALITY",
			RawProtocol:  "vless",
			Port:         443,
			Outbound:     "socks5://127.0.0.1:1081",
			OutboundPort: 1081,
			URL:          "vless://uuid@auto:443?type=tcp#[旧分组] reality-443",
		},
		{
			Name:         "hy2-8443.json",
			Protocol:     "Hysteria2",
			RawProtocol:  "hysteria2",
			Port:         8443,
			Outbound:     "socks5://127.0.0.1:1082",
			OutboundPort: 1082,
			URL:          "hysteria2://pass@auto:8443#hy2-8443",
		},
		{
			Name:         "direct-node.json",
			Protocol:     "Trojan",
			RawProtocol:  "trojan",
			Port:         4443,
			Outbound:     "direct",
			OutboundPort: 0,
		},
	}

	// 1. Test Node Decoration
	decNodes := s.DecorateSubscriptionNodes(testNodes, false)
	if len(decNodes) != 3 {
		t.Fatalf("expected 3 decorated nodes, got %d", len(decNodes))
	}

	// First node: bound to dg-japan -> [日本高速专线] reality-443
	if decNodes[0].Name != "[日本高速专线] reality-443" {
		t.Errorf("expected [日本高速专线] reality-443, got %s", decNodes[0].Name)
	}
	if !strings.Contains(decNodes[0].URL, "%5B%E6%97%A5%E6%9C%AC%E9%AB%98%E9%80%9F%E4%B8%93%E7%BA%BF%5D") {
		t.Errorf("expected decorated URL fragment in node 0, got %s", decNodes[0].URL)
	}

	// Second node: bound to dg-fav -> [我的收藏出口组] hy2-8443
	if decNodes[1].Name != "[我的收藏出口组] hy2-8443" {
		t.Errorf("expected [我的收藏出口组] hy2-8443, got %s", decNodes[1].Name)
	}

	// Third node: direct -> [直连] direct-node
	if decNodes[2].Name != "[直连] direct-node" {
		t.Errorf("expected [直连] direct-node, got %s", decNodes[2].Name)
	}

	// Test English decoration
	decNodesEn := s.DecorateSubscriptionNodes(testNodes, true)
	if decNodesEn[2].Name != "[Direct] direct-node" {
		t.Errorf("expected [Direct] direct-node in English, got %s", decNodesEn[2].Name)
	}

	// 2. Test Clash YAML Generation with decorated nodes
	clashYaml := GenerateClashYAML(decNodes, "198.51.100.1")
	if !strings.Contains(clashYaml, "name: \"[日本高速专线] reality-443\"") {
		t.Errorf("Clash YAML missing decorated reality proxy name: %s", clashYaml)
	}
	if !strings.Contains(clashYaml, "- \"[日本高速专线] reality-443\"") {
		t.Errorf("Clash YAML proxy groups missing decorated reality proxy name: %s", clashYaml)
	}

	// 3. Test Filtering
	// 3a. Filter by favorites
	qFav := url.Values{"filter": []string{"favorites"}}
	filteredFav := s.FilterSubscriptionNodes(testNodes, qFav)
	if len(filteredFav) != 1 || filteredFav[0].Port != 8443 {
		t.Errorf("expected only hy2-8443 for favorites filter, got %d nodes", len(filteredFav))
	}

	// 3b. Filter by country JP
	qJP := url.Values{"countries": []string{"JP"}}
	filteredJP := s.FilterSubscriptionNodes(testNodes, qJP)
	if len(filteredJP) != 1 || filteredJP[0].Port != 443 {
		t.Errorf("expected only reality-443 for JP filter, got %d nodes", len(filteredJP))
	}

	// 3c. Filter by protocol
	qProto := url.Values{"protocol": []string{"trojan"}}
	filteredProto := s.FilterSubscriptionNodes(testNodes, qProto)
	if len(filteredProto) != 1 || filteredProto[0].Port != 4443 {
		t.Errorf("expected only trojan for protocol filter, got %d nodes", len(filteredProto))
	}

	// 4. Test URL filter query propagation in buildGenericSubURL and buildClashSubURL
	reqWithFav := httptest.NewRequest("GET", "/api/singbox/subscription?format=json&filter=favorites", nil)
	genURL := s.buildGenericSubURL(reqWithFav)
	if !strings.Contains(genURL, "filter=favorites") {
		t.Errorf("expected filter=favorites in generated generic URL, got %s", genURL)
	}
	clashSubURL := s.buildClashSubURL(reqWithFav)
	if !strings.Contains(clashSubURL, "filter=favorites") {
		t.Errorf("expected filter=favorites in generated clash URL, got %s", clashSubURL)
	}
}

