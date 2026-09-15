package nodes

import (
	"testing"
	"time"
)

func TestGetProbeStrategy(t *testing.T) {
	tests := []struct {
		name               string
		attempt            int
		expectedTCPTimeout time.Duration
		expectedUDPTimeout time.Duration
		expectedVPNTimeout time.Duration
	}{
		{
			name:               "First attempt uses generous timeouts",
			attempt:            0,
			expectedTCPTimeout: 12 * time.Second,
			expectedUDPTimeout: 8 * time.Second,
			expectedVPNTimeout: 35 * time.Second,
		},
		{
			name:               "Retry uses strict timeouts",
			attempt:            1,
			expectedTCPTimeout: 4 * time.Second,
			expectedUDPTimeout: 2500 * time.Millisecond,
			expectedVPNTimeout: 15 * time.Second,
		},
		{
			name:               "Multiple retries use same strict timeouts",
			attempt:            5,
			expectedTCPTimeout: 4 * time.Second,
			expectedUDPTimeout: 2500 * time.Millisecond,
			expectedVPNTimeout: 15 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := GetProbeStrategy(tt.attempt)

			if strategy.Attempt != tt.attempt {
				t.Errorf("Expected attempt %d, got %d", tt.attempt, strategy.Attempt)
			}

			if strategy.TCPTimeout != tt.expectedTCPTimeout {
				t.Errorf("Expected TCP timeout %v, got %v", tt.expectedTCPTimeout, strategy.TCPTimeout)
			}

			if strategy.UDPTimeout != tt.expectedUDPTimeout {
				t.Errorf("Expected UDP timeout %v, got %v", tt.expectedUDPTimeout, strategy.UDPTimeout)
			}

			if strategy.OpenVPNTimeout != tt.expectedVPNTimeout {
				t.Errorf("Expected OpenVPN timeout %v, got %v", tt.expectedVPNTimeout, strategy.OpenVPNTimeout)
			}
		})
	}
}

func TestDefaultProbeConfig(t *testing.T) {
	cfg := DefaultProbeConfig()

	if cfg.MaxAttempts != 2 {
		t.Errorf("Expected MaxAttempts 2, got %d", cfg.MaxAttempts)
	}

	if !cfg.UseLayered {
		t.Error("Expected UseLayered to be true for default config")
	}
}

func TestLegacyProbeConfig(t *testing.T) {
	cfg := LegacyProbeConfig()

	if cfg.MaxAttempts != 1 {
		t.Errorf("Expected MaxAttempts 1, got %d", cfg.MaxAttempts)
	}

	if cfg.UseLayered {
		t.Error("Expected UseLayered to be false for legacy config")
	}
}

func TestTimeoutProgression(t *testing.T) {
	// Verify that first attempt is always more generous than retry
	first := GetProbeStrategy(0)
	retry := GetProbeStrategy(1)

	if first.TCPTimeout <= retry.TCPTimeout {
		t.Errorf("First attempt TCP timeout (%v) should be greater than retry (%v)",
			first.TCPTimeout, retry.TCPTimeout)
	}

	if first.UDPTimeout <= retry.UDPTimeout {
		t.Errorf("First attempt UDP timeout (%v) should be greater than retry (%v)",
			first.UDPTimeout, retry.UDPTimeout)
	}

	if first.OpenVPNTimeout <= retry.OpenVPNTimeout {
		t.Errorf("First attempt OpenVPN timeout (%v) should be greater than retry (%v)",
			first.OpenVPNTimeout, retry.OpenVPNTimeout)
	}
}
