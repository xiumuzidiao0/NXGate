package tunnel

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ThroughputConfig defines the configuration for throughput testing
type ThroughputConfig struct {
	TestURL        string        // URL to download from for bandwidth testing
	MinBytesPerSec int64         // Minimum acceptable throughput (bytes/sec)
	Timeout        time.Duration // Total timeout for the test
	MaxAttempts    int           // Number of retry attempts
	SampleDuration time.Duration // How long to measure throughput
	MinSampleBytes int64         // Minimum bytes to read for valid test
}

// ThroughputResult contains the result of a throughput test
type ThroughputResult struct {
	Passed         bool
	BytesPerSecond int64
	BytesRead      int64
	Duration       time.Duration
	Attempt        int
	Error          error
}

func (r ThroughputResult) String() string {
	if !r.Passed {
		if r.Error != nil {
			return fmt.Sprintf("FAILED (%.1f KB/s, %s)", float64(r.BytesPerSecond)/1024, r.Error.Error())
		}
		return fmt.Sprintf("FAILED (%.1f KB/s < 70 KB/s)", float64(r.BytesPerSecond)/1024)
	}
	return fmt.Sprintf("PASSED (%.1f KB/s)", float64(r.BytesPerSecond)/1024)
}

// DefaultThroughputConfig returns the default configuration for throughput testing
func DefaultThroughputConfig() ThroughputConfig {
	return ThroughputConfig{
		TestURL:        "https://speed.cloudflare.com/__down?bytes=1000000", // 1MB test file
		MinBytesPerSec: 70 * 1024,                                           // 70 KB/s minimum (freesub standard)
		Timeout:        8 * time.Second,
		MaxAttempts:    2,
		SampleDuration: 5 * time.Second,
		MinSampleBytes: 50 * 1024, // At least 50KB downloaded for valid test
	}
}

// CheckThroughputWithRetry performs throughput testing with retry logic
func CheckThroughputWithRetry(devName string, cfg ThroughputConfig) ThroughputResult {
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		result := checkThroughput(devName, cfg, attempt)
		if result.Passed {
			return result
		}
		// Don't retry if we got a hard error (network unreachable, etc)
		if result.Error != nil && result.BytesRead == 0 {
			return result
		}
	}

	// Return last attempt result
	return checkThroughput(devName, cfg, cfg.MaxAttempts)
}

// checkThroughput performs a single throughput test on a tunnel interface
func checkThroughput(devName string, cfg ThroughputConfig, attempt int) ThroughputResult {
	result := ThroughputResult{
		Attempt: attempt,
		Passed:  false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	client := newTunnelHTTPClient(devName, cfg.Timeout)
	req, err := http.NewRequestWithContext(ctx, "GET", cfg.TestURL, nil)
	if err != nil {
		result.Error = fmt.Errorf("create request failed: %w", err)
		return result
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("request failed: %w", err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		result.Error = fmt.Errorf("HTTP %d", resp.StatusCode)
		return result
	}

	// Read data for sample duration or until context timeout
	sampleCtx, sampleCancel := context.WithTimeout(ctx, cfg.SampleDuration)
	defer sampleCancel()

	buf := make([]byte, 32*1024) // 32KB buffer
	var totalBytes int64

	for {
		select {
		case <-sampleCtx.Done():
			goto measure
		default:
		}

		n, err := resp.Body.Read(buf)
		totalBytes += int64(n)
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Error = fmt.Errorf("read error: %w", err)
			result.BytesRead = totalBytes
			return result
		}
	}

measure:
	elapsed := time.Since(start)
	result.BytesRead = totalBytes
	result.Duration = elapsed

	if elapsed.Seconds() < 0.1 {
		result.Error = fmt.Errorf("test too short (%.2fs)", elapsed.Seconds())
		return result
	}

	bytesPerSec := int64(float64(totalBytes) / elapsed.Seconds())
	result.BytesPerSecond = bytesPerSec

	// Pass criteria:
	// 1. Downloaded minimum sample size
	// 2. Throughput exceeds minimum threshold
	if totalBytes >= cfg.MinSampleBytes && bytesPerSec >= cfg.MinBytesPerSec {
		result.Passed = true
	} else if totalBytes < cfg.MinSampleBytes {
		result.Error = fmt.Errorf("insufficient data (got %d bytes, need %d)", totalBytes, cfg.MinSampleBytes)
	}

	return result
}
