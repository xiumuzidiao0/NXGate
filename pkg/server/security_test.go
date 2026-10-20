package server

import (
	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/proxy"
	"aimili-vpngate-go/pkg/singbox"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecurityRejectDefaultAuth(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("UI_USERNAME", "")
	t.Setenv("UI_PASSWORD", "")
	cfg := config.LoadConfig()
	mw := NewMiddleware(cfg)
	handler := mw.SecretPathGuard(mw.BasicAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })))
	req := httptest.NewRequest("GET", "/api/settings", nil)
	req.SetBasicAuth("admin", "aimilivpn")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code == 204 {
		t.Fatalf("unexpected %d", w.Code)
	}

}
func TestSecurityAgeCannotBeBypassed(t *testing.T) {
	secret, pub, _, _ := GenerateAgeKeyPair("x25519")
	cfg := &config.Config{UIHost: "198.51.100.1", UIPort: 8787, UIPath: "admin-secret", SubToken: "subscription-token", UIUsername: "admin", UIPassword: "strong-test-pass", AgeEncryptEnabled: true, AgePublicKey: pub}
	cli := filepath.Join(t.TempDir(), "fixture-cli")
	data := `{"ok":true,"nodes":[{"name":"review","protocol":"Hysteria2","raw_protocol":"hysteria2","port":8443,"address":"198.51.100.1","password":"TEST-ONLY-SECRET","url":"hysteria2://TEST-ONLY-SECRET@198.51.100.1:8443#review"}]}`
	if err := os.WriteFile(cli, []byte("#!/bin/sh\nprintf '%s' '"+data+"'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SINGBOX_CLI", cli)
	s := &Server{cfg: cfg, singboxClient: singbox.NewClient()}
	mw := NewMiddleware(cfg)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/singbox/subscription", s.handleSingBoxGetSub)
	mux.HandleFunc("GET /api/singbox/subscription/raw", s.handleSingBoxRawSub)
	mux.HandleFunc("GET /api/singbox/subscription/clash", s.handleSingBoxClashSub)
	mux.HandleFunc("GET /api/singbox/subscription-info", s.handleSubscriptionInfo)
	handler := mw.SecretPathGuard(mw.BasicAuth(mux))
	get := func(q string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/api/singbox/subscription?token=subscription-token&"+q, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("status=%d %s", w.Code, w.Body.String())
		}
		return w
	}

	for _, format := range []string{"raw", "text", "base64", "b64", "json", "clash"} {
		for _, q := range []string{"", "&age_recipient=invalid"} {
			w := get("format=" + format + q)
			if strings.Contains(w.Body.String(), "TEST-ONLY-SECRET") {
				t.Fatal("plaintext credential leak")
			}
			plain, err := DecryptWithAge(w.Body.Bytes(), secret)
			if format == "base64" || format == "b64" {
				decoded, decodeErr := base64.StdEncoding.DecodeString(string(plain))
				if decodeErr != nil {
					t.Fatal(decodeErr)
				}
				plain = decoded
			}
			if err != nil || !strings.Contains(string(plain), "TEST-ONLY-SECRET") {
				t.Fatalf("format %s: expected configured key encryption: %v", format, err)
			}
		}
	}
	// Both dedicated download handlers and secret-path aliases must enforce encryption too.
	for _, path := range []string{"/api/singbox/subscription/raw?token=subscription-token", "/api/singbox/subscription/clash?token=subscription-token", "/sub/subscription-token?format=json", "/subscription-token/api/singbox/subscription?format=json"} {
		r := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(path, w.Code, w.Body.String())
		}
		if _, err := DecryptWithAge(w.Body.Bytes(), secret); err != nil {
			t.Fatal("alias not encrypted", path, err)
		}
	}
	rInfo := httptest.NewRequest("GET", "/api/singbox/subscription-info", nil)
	rInfo.SetBasicAuth("admin", "strong-test-pass")
	wInfo := httptest.NewRecorder()
	handler.ServeHTTP(wInfo, rInfo)
	if wInfo.Code != 200 || strings.Contains(wInfo.Body.String(), "TEST-ONLY-SECRET") || !strings.Contains(wInfo.Body.String(), `"node_count":1`) {
		t.Fatal("metadata endpoint unusable or exposes credentials", wInfo.Code, wInfo.Body.String())
	}
	sec2, pub2, _, _ := GenerateAgeKeyPair("x25519")
	w := get("format=raw&age_recipient=" + url.QueryEscape(pub2))
	if _, err := DecryptWithAge(w.Body.Bytes(), sec2); err == nil {
		t.Fatal("requester replaced enforced key")
	}
	cfg.AgePublicKey = "invalid"
	for _, format := range []string{"raw", "base64", "json", "clash"} {
		r := httptest.NewRequest("GET", "/api/singbox/subscription?token=subscription-token&format="+format, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 500 || strings.Contains(w.Body.String(), "TEST-ONLY-SECRET") {
			t.Fatal("invalid configured key did not fail closed", format)
		}
	}
	cfg.AgeEncryptEnabled = false
	cfg.AgePublicKey = ""
	w = get("format=raw")
	if !strings.Contains(w.Body.String(), "TEST-ONLY-SECRET") {
		t.Fatal("unencrypted mode changed")
	}
	rInvalid := httptest.NewRequest("GET", "/api/singbox/subscription?token=subscription-token&format=raw&age_recipient=invalid", nil)
	wInvalid := httptest.NewRecorder()
	handler.ServeHTTP(wInvalid, rInvalid)
	if wInvalid.Code != 500 || strings.Contains(wInvalid.Body.String(), "TEST-ONLY-SECRET") {
		t.Fatal("explicit encryption failure leaked plaintext")
	}
}

func TestSecuritySubscriptionTokenNotAdmin(t *testing.T) {
	cfg := &config.Config{UIUsername: "admin", UIPassword: "test-password", SubToken: "subscription-token", UIPath: "secret"}
	mw := NewMiddleware(cfg)
	handler := mw.SecretPathGuard(mw.BasicAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })))
	for _, item := range []struct{ method, path string }{{"GET", "/api/singbox/subscription-info"}, {"POST", "/api/singbox/subscription/age/generate"}, {"POST", "/api/singbox/subscription/sync"}} {
		r := httptest.NewRequest(item.method, item.path+"?token=subscription-token", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code == 204 {
			t.Fatal("token grants admin access", item)
		}
		r = httptest.NewRequest(item.method, item.path, nil)
		r.SetBasicAuth("admin", "test-password")
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 204 {
			t.Fatal("admin denied", item, w.Code)
		}
	}
}

func TestSecurityBackupReportsFailure(t *testing.T) {
	cfg := &config.Config{DataDir: t.TempDir(), UIPort: 8787, ProxyPort: 7928, UIUsername: "admin", UIPassword: "test-password"}
	pm := proxy.NewMultiPortManager(cfg, nil, nil)
	s := &Server{cfg: cfg, portMgr: pm}
	req := httptest.NewRequest("POST", "/api/system/backup/import", strings.NewReader(`{"port_rules":[{"port":8787,"enabled":false,"auth_mode":"none"}]}`))
	w := httptest.NewRecorder()
	s.handleImportBackup(w, req)
	if w.Code != 400 || !strings.Contains(w.Body.String(), `"ok":false`) {
		t.Fatal("restore failure reported success", w.Code, w.Body.String())
	}
}
