package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/vpn"
)

func TestAppInfoAndProfileAPI(t *testing.T) {
	cfg := &config.Config{
		UIHost:     "::",
		UIPort:     8787,
		UIPath:     "enter",
		UIUsername: "testuser",
		UIPassword: "testpassword",
		ProxyPort:  7928,
	}

	vpnMgr := vpn.NewManager(cfg, nil, nil)
	s := &Server{
		cfg: cfg,
		vpn: vpnMgr,
	}

	// 1. Test handleAppInfo
	reqInfo := httptest.NewRequest("GET", "/api/app/info", nil)
	wInfo := httptest.NewRecorder()
	s.handleAppInfo(wInfo, reqInfo)

	if wInfo.Code != http.StatusOK {
		t.Fatalf("expected 200 for app info, got %d", wInfo.Code)
	}

	var appInfo AppInfoResponse
	if err := json.NewDecoder(wInfo.Body).Decode(&appInfo); err != nil {
		t.Fatalf("decode app info json: %v", err)
	}
	if !appInfo.OK || appInfo.App != "aimili-vpngate-go" || appInfo.Version != config.Version {
		t.Fatalf("unexpected app info: %+v", appInfo)
	}
	if len(appInfo.Capabilities) == 0 {
		t.Fatalf("expected capabilities list to be populated")
	}

	// 2. Test handleAppProfile
	reqProfile := httptest.NewRequest("GET", "/api/app/profile?name=Tokyo-VPS", nil)
	reqProfile.Host = "47.238.2.197:8787"
	wProfile := httptest.NewRecorder()
	s.handleAppProfile(wProfile, reqProfile)

	if wProfile.Code != http.StatusOK {
		t.Fatalf("expected 200 for app profile, got %d", wProfile.Code)
	}

	var appProfile AppProfileResponse
	if err := json.NewDecoder(wProfile.Body).Decode(&appProfile); err != nil {
		t.Fatalf("decode app profile json: %v", err)
	}
	if !appProfile.OK {
		t.Fatalf("expected ok=true")
	}
	p := appProfile.Profile
	if p.Host != "47.238.2.197" || p.Port != 8787 || p.Path != "enter" || p.Username != "testuser" || p.Password != "testpassword" || p.Name != "Tokyo-VPS" {
		t.Fatalf("unexpected profile values: %+v", p)
	}
	if !strings.HasPrefix(appProfile.ConnectURI, "aimili://server?") {
		t.Fatalf("expected connect URI prefix aimili://server?, got: %s", appProfile.ConnectURI)
	}
	if !strings.Contains(appProfile.ConnectURI, "host=47.238.2.197") || !strings.Contains(appProfile.ConnectURI, "user=testuser") {
		t.Fatalf("expected connect URI to contain host and user, got: %s", appProfile.ConnectURI)
	}
}

func TestAppQueryAuthAndCorsOptions(t *testing.T) {
	cfg := &config.Config{
		UIPath:     "secretpath",
		UIUsername: "admin",
		UIPassword: "secretpassword",
	}
	mw := NewMiddleware(cfg)

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok-api"))
	})

	// Wrap with SecurityHeaders -> SecretPathGuard -> BasicAuth
	handler := mw.BasicAuth(dummyHandler)
	handler = mw.SecretPathGuard(handler)
	handler = mw.SecurityHeaders(handler)

	// 1. OPTIONS preflight to /api/status -> 204 with CORS
	reqOptions := httptest.NewRequest("OPTIONS", "/api/status", nil)
	wOptions := httptest.NewRecorder()
	handler.ServeHTTP(wOptions, reqOptions)
	if wOptions.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content for OPTIONS, got %d", wOptions.Code)
	}
	if wOptions.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected CORS allow origin header, got %s", wOptions.Header().Get("Access-Control-Allow-Origin"))
	}

	// 2. Query parameter authentication ?user=admin&pass=secretpassword on direct /api/app/info -> 200
	reqQueryAuth := httptest.NewRequest("GET", "/api/app/info?user=admin&pass=secretpassword", nil)
	wQueryAuth := httptest.NewRecorder()
	handler.ServeHTTP(wQueryAuth, reqQueryAuth)
	if wQueryAuth.Code != http.StatusOK {
		t.Fatalf("expected 200 with query auth, got %d", wQueryAuth.Code)
	}
	if wQueryAuth.Body.String() != "ok-api" {
		t.Fatalf("expected body ok-api, got %s", wQueryAuth.Body.String())
	}

	// 3. Query parameter authentication with wrong password -> 404 stealth / 401
	reqWrongAuth := httptest.NewRequest("GET", "/api/app/info?user=admin&pass=wrongpass", nil)
	wWrongAuth := httptest.NewRecorder()
	handler.ServeHTTP(wWrongAuth, reqWrongAuth)
	if wWrongAuth.Code != http.StatusNotFound && wWrongAuth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 404 or 401 for wrong credentials, got %d", wWrongAuth.Code)
	}
}
