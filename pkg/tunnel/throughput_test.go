package tunnel

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDefaultThroughputConfig(t *testing.T) {
	config := DefaultThroughputConfig()

	if config.MinBytesPerSec != 70*1024 {
		t.Errorf("Expected MinBytesPerSec = 71680, got %d", config.MinBytesPerSec)
	}

	if config.TestURL == "" {
		t.Error("TestURL should not be empty")
	}

	if config.Timeout == 0 {
		t.Error("Timeout should be set")
	}
}

func TestThroughputResultString(t *testing.T) {
	tests := []struct {
		name   string
		result ThroughputResult
		want   string
	}{
		{
			name: "successful result",
			result: ThroughputResult{
				Passed:         true,
				BytesPerSecond: 100 * 1024, // 100 KB/s
				BytesRead:      500 * 1024,
				Duration:       5 * time.Second,
			},
			want: "PASSED (100.0 KB/s)",
		},
		{
			name: "failed result",
			result: ThroughputResult{
				Passed:         false,
				BytesPerSecond: 30 * 1024, // 30 KB/s (below threshold)
				BytesRead:      150 * 1024,
				Duration:       5 * time.Second,
			},
			want: "FAILED (30.0 KB/s < 70 KB/s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.String()
			if got != tt.want {
				t.Errorf("ThroughputResult.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckThroughputLocalFixture(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		io.WriteString(w, strings.Repeat("x", 128*1024))
	}))
	defer srv.Close()
	cfg := ThroughputConfig{TestURL: srv.URL, MinBytesPerSec: 1, Timeout: time.Second, MaxAttempts: 1, SampleDuration: time.Second, MinSampleBytes: 1}
	dial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, srv.Listener.Addr().String())
	}
	got := CheckThroughputWithDialer(dial, cfg)
	if !got.Passed || got.BytesRead != 128*1024 {
		t.Fatalf("%+v", got)
	}
}
func TestCheckThroughputLocalTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer srv.Close()
	cfg := ThroughputConfig{TestURL: srv.URL, Timeout: 30 * time.Millisecond, MaxAttempts: 1, SampleDuration: time.Second}
	dial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, srv.Listener.Addr().String())
	}
	got := CheckThroughputWithDialer(dial, cfg)
	if got.Error == nil || got.Passed {
		t.Fatalf("%+v", got)
	}
}
