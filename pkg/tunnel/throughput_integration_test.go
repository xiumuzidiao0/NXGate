//go:build integration

package tunnel

import (
	"testing"
	"time"
)

func TestIntegrationCheckThroughput(t *testing.T) {
	// This test requires actual network access
	// Explicit integration job; failures are real failures, not offline skips.
	if testing.Short() {
		t.Skip("Skipping network-dependent throughput test in short mode")
	}

	config := ThroughputConfig{
		TestURL:        "https://speed.cloudflare.com/__down?bytes=524288", // 512KB
		MinBytesPerSec: 10 * 1024,                                          // Very low threshold for test
		Timeout:        10 * time.Second,
		MaxAttempts:    1, SampleDuration: time.Second, MinSampleBytes: 1,
	}

	// Test with no device binding (direct connection)
	result := CheckThroughputWithRetry("", config)

	if !result.Passed || result.Error != nil {
		t.Errorf("public throughput smoke failed: %s", result.String())
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

func TestIntegrationCheckThroughputTimeout(t *testing.T) {
	config := ThroughputConfig{
		TestURL:        "https://speed.cloudflare.com/__down?bytes=10485760", // 10MB
		MinBytesPerSec: 1000 * 1024,                                          // 1 MB/s
		Timeout:        1 * time.Second,
		MaxAttempts:    1, SampleDuration: time.Second, MinSampleBytes: 1, // Very short timeout
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

func TestIntegrationCheckThroughputWithRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent retry test in short mode")
	}

	config := ThroughputConfig{
		TestURL:        "https://speed.cloudflare.com/__down?bytes=262144", // 256KB
		MinBytesPerSec: 10 * 1024,                                          // 10 KB/s minimum
		Timeout:        8 * time.Second,
		MaxAttempts:    2, SampleDuration: time.Second, MinSampleBytes: 1,
	}

	result := CheckThroughputWithRetry("", config)

	if result.Error != nil {
		t.Errorf("public throughput retry smoke failed: %v", result.Error)
		return
	}

	t.Logf("Throughput retry result: %s", result.String())
}
