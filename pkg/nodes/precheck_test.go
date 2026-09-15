package nodes

import (
	"context"
	"testing"
	"time"
)

func TestPortKnockTCP(t *testing.T) {
	tests := []struct {
		name       string
		ip         string
		port       int
		timeout    time.Duration
		wantReach  bool
		wantLatLow time.Duration
	}{
		{
			name:       "Google DNS should be reachable on port 53",
			ip:         "8.8.8.8",
			port:       53,
			timeout:    2 * time.Second,
			wantReach:  true,
			wantLatLow: 2 * time.Second,
		},
		{
			name:       "Cloudflare DNS should be reachable on port 53",
			ip:         "1.1.1.1",
			port:       53,
			timeout:    2 * time.Second,
			wantReach:  true,
			wantLatLow: 2 * time.Second,
		},
		{
			name:       "Invalid IP should fail",
			ip:         "192.0.2.1", // TEST-NET-1 (should timeout)
			port:       9999,
			timeout:    500 * time.Millisecond,
			wantReach:  false,
			wantLatLow: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reachable, latency, err := PortKnockTCP(tt.ip, tt.port, tt.timeout)

			if reachable != tt.wantReach {
				t.Errorf("PortKnockTCP() reachable = %v, want %v (err: %v)", reachable, tt.wantReach, err)
			}

			if latency > tt.wantLatLow {
				t.Errorf("PortKnockTCP() latency = %v, want < %v", latency, tt.wantLatLow)
			}

			if reachable && err != nil {
				t.Errorf("PortKnockTCP() reachable but got error: %v", err)
			}
		})
	}
}

func TestBatchPortKnock(t *testing.T) {
	nodes := []*Node{
		{
			HostName:     "Google DNS",
			IP:           "8.8.8.8",
			Proto:        "tcp",
			TCPPort:      53,
			CountryLong:  "United States",
			CountryShort: "US",
		},
		{
			HostName:     "Cloudflare DNS",
			IP:           "1.1.1.1",
			Proto:        "tcp",
			TCPPort:      53,
			CountryLong:  "United States",
			CountryShort: "US",
		},
		{
			HostName:     "Dead Node",
			IP:           "192.0.2.1", // TEST-NET-1
			Proto:        "tcp",
			TCPPort:      9999,
			CountryLong:  "Unknown",
			CountryShort: "XX",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := BatchPortKnock(ctx, nodes, 2*time.Second)

	if len(results) != len(nodes) {
		t.Fatalf("BatchPortKnock() returned %d results, want %d", len(results), len(nodes))
	}

	// Check that at least the public DNS servers are reachable
	reachableCount := 0
	for _, r := range results {
		if r.Reachable {
			reachableCount++
		}
	}

	if reachableCount < 2 {
		t.Errorf("BatchPortKnock() only %d nodes reachable, expected at least 2", reachableCount)
	}
}

func TestFilterReachableNodes(t *testing.T) {
	nodes := []*Node{
		{
			HostName:     "Google DNS",
			IP:           "8.8.8.8",
			Proto:        "tcp",
			TCPPort:      53,
			CountryLong:  "United States",
			CountryShort: "US",
		},
		{
			HostName:     "Dead Node",
			IP:           "192.0.2.1",
			Proto:        "tcp",
			TCPPort:      9999,
			CountryLong:  "Unknown",
			CountryShort: "XX",
		},
	}

	reachable := FilterReachableNodes(nodes, 2*time.Second)

	if len(reachable) < 1 {
		t.Errorf("FilterReachableNodes() returned %d nodes, expected at least 1", len(reachable))
	}

	// Verify the reachable node is the Google DNS one
	if len(reachable) > 0 && reachable[0].IP != "8.8.8.8" {
		t.Errorf("FilterReachableNodes() expected first reachable node to be 8.8.8.8, got %s", reachable[0].IP)
	}
}

func TestPortKnockTimeout(t *testing.T) {
	// Test that timeout is respected
	start := time.Now()
	timeout := 500 * time.Millisecond

	// Try to connect to a non-routable IP
	reachable, latency, _ := PortKnockTCP("192.0.2.1", 9999, timeout)

	elapsed := time.Since(start)

	if reachable {
		t.Error("PortKnockTCP() should not be reachable for non-routable IP")
	}

	// Allow some margin for timeout accuracy (±200ms)
	if elapsed > timeout+200*time.Millisecond {
		t.Errorf("PortKnockTCP() took %v, expected around %v", elapsed, timeout)
	}

	if latency > timeout+200*time.Millisecond {
		t.Errorf("PortKnockTCP() latency %v exceeds timeout %v", latency, timeout)
	}
}
