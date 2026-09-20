package server

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/singbox"
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
