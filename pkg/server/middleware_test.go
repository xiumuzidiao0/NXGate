package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aimili-vpngate-go/pkg/config"
)

func TestLimitRequestBody(t *testing.T) {
	mw := NewMiddleware(&config.Config{})
	handler := mw.LimitRequestBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", maxRequestBodyBytes+1)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got %d", rec.Code)
	}
}

func TestMiddlewareDisabledPathAndAuth(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("UI_PATH", "")
	t.Setenv("UI_USERNAME", "")
	t.Setenv("UI_PASSWORD", "")

	cfg := config.LoadConfig()
	mw := NewMiddleware(cfg)
	handler := mw.SecretPathGuard(mw.BasicAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	for _, path := range []string{"/", "/api/status", "/api/settings"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 on %s when path and auth are disabled, got %d", path, rec.Code)
		}
	}
}

func TestAuthRateLimitBlocksAndResetsOnSuccess(t *testing.T) {
	cfg := &config.Config{UIUsername: "admin", UIPassword: "strong-password"}
	mw := NewMiddleware(cfg)
	handler := mw.BasicAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < authMaxFailures; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
		req.RemoteAddr = "198.51.100.44:12345"
		req.SetBasicAuth("admin", "wrong")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, rec.Code)
		}
	}

	// Next attempt from the same IP should be rate-limited even with valid credentials until window expires.
	blockedReq := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	blockedReq.RemoteAddr = "198.51.100.44:12346"
	blockedReq.SetBasicAuth("admin", "strong-password")
	blockedRec := httptest.NewRecorder()
	handler.ServeHTTP(blockedRec, blockedReq)
	if blockedRec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests when locked out, got %d", blockedRec.Code)
	}

	// A different IP succeeds and resets cleanly.
	otherReq := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	otherReq.RemoteAddr = "198.51.100.45:22222"
	otherReq.SetBasicAuth("admin", "strong-password")
	otherRec := httptest.NewRecorder()
	handler.ServeHTTP(otherRec, otherReq)
	if otherRec.Code != http.StatusOK {
		t.Fatalf("expected other IP to succeed (200), got %d", otherRec.Code)
	}
}
