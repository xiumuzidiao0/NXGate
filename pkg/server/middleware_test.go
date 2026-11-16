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
