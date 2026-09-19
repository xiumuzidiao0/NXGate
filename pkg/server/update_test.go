package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1, v2 string
		want   int
	}{
		{"2.5.6", "2.5.6", 0},
		{"v2.5.6", "2.5.6", 0},
		{"2.5.7", "2.5.6", 1},
		{"2.5.6", "2.5.7", -1},
		{"v2.6.0", "v2.5.9", 1},
		{"2.5.10", "2.5.9", 1},
		{"2.5.9", "2.5.10", -1},
		{"3.0.0", "2.9.99", 1},
		{"2.5", "2.5.0", 0},
		{"2.5.1", "2.5", 1},
	}

	for _, tt := range tests {
		got := CompareVersions(tt.v1, tt.v2)
		if got != tt.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.want)
		}
	}
}

func TestUpdateEndpointsLifecycle(t *testing.T) {
	s := &Server{}

	// 1. Check status endpoint
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/update/status", nil)
	wStatus := httptest.NewRecorder()
	s.handleGetUpdateStatus(wStatus, reqStatus)

	if wStatus.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", wStatus.Code)
	}

	var statusResp UpdateStatusResponse
	if err := json.Unmarshal(wStatus.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("failed to decode status response: %v", err)
	}
	if statusResp.InProgress {
		t.Fatalf("expected in_progress=false initially")
	}

	// 2. Test Check Update endpoint
	reqCheck := httptest.NewRequest(http.MethodGet, "/api/update/check", nil)
	wCheck := httptest.NewRecorder()
	s.handleCheckUpdate(wCheck, reqCheck)

	if wCheck.Code != http.StatusOK {
		t.Fatalf("expected check 200, got %d", wCheck.Code)
	}
	var checkResp CheckUpdateResponse
	if err := json.Unmarshal(wCheck.Body.Bytes(), &checkResp); err != nil {
		t.Fatalf("failed to decode check response: %v", err)
	}
	if checkResp.CurrentVersion == "" {
		t.Fatalf("current version should not be empty")
	}
}
