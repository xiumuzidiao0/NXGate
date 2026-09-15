package tunnel

import (
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

func TestCheckThroughput(t *testing.T) {
	// This test requires actual network access
	// Skip in CI or when network is unavailable
	if testing.Short() {
		t.Skip("Skipping network-dependent throughput test in short mode")
	}

	config := ThroughputConfig{
		TestURL:        "https://speed.cloudflare.com/__down?bytes=524288", // 512KB
		MinBytesPerSec: 10 * 1024,                                           // Very low threshold for test
		Timeout:        10 * time.Second,
	}

	// Test with no device binding (direct connection)
	result := CheckThroughputWithRetry("", config)

	if !result.Passed || result.Error != nil {
		t.Logf("Throughput check not passed (expected in offline/limited test environment): %s", result.String())
		return
	}

	if result.BytesRead == 0 {
		t.Error("Expected some bytes to be read")
	}

	if result.BytesPerSecond == 0 {
		t.Error("Expected non-zero bytes per second")
	}

	t.Logf("Throughput test result: %s", result.String())
}

func TestCheckThroughputTimeout(t *testing.T) {
	config := ThroughputConfig{
		TestURL:        "https://speed.cloudflare.com/__down?bytes=10485760", // 10MB
		MinBytesPerSec: 1000 * 1024,                                           // 1 MB/s
		Timeout:        1 * time.Second,                                       // Very short timeout
	}

	start := time.Now()
	result := CheckThroughputWithRetry("", config)
	elapsed := time.Since(start)

	// Should respect timeout
	if elapsed > 3*time.Second {
		t.Errorf("CheckThroughputWithRetry took %v, expected < 3s", elapsed)
	}

	// May or may not pass depending on network speed
	t.Logf("Timeout test result: %s (elapsed: %v)", result.String(), elapsed)
}

func TestCheckThroughputWithRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent retry test in short mode")
	}

	config := ThroughputConfig{
		TestURL:        "https://speed.cloudflare.com/__down?bytes=262144", // 256KB
		MinBytesPerSec: 10 * 1024,                                           // 10 KB/s minimum
		Timeout:        8 * time.Second,
	}

	result := CheckThroughputWithRetry("", config)

	if result.Error != nil {
		t.Logf("Throughput retry test failed (may be expected): %v", result.Error)
		return
	}

	t.Logf("Throughput retry result: %s", result.String())
}
