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
	}{
		{
			name:               "First attempt uses generous timeouts",
			attempt:            0,
			expectedTCPTimeout: 6 * time.Second,
			expectedUDPTimeout: 4 * time.Second,
		},
		{
			name:               "Retry uses tighter timeouts",
			attempt:            1,
			expectedTCPTimeout: 4 * time.Second,
			expectedUDPTimeout: 3 * time.Second,
		},
		{
			name:               "Final attempt uses strict timeouts",
			attempt:            2,
			expectedTCPTimeout: 3 * time.Second,
			expectedUDPTimeout: 2 * time.Second,
		},
		{
			name:               "Subsequent retries cap at strict timeouts",
			attempt:            5,
			expectedTCPTimeout: 3 * time.Second,
			expectedUDPTimeout: 2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := GetProbeStrategy(tt.attempt)

			if strategy.TCPTimeout != tt.expectedTCPTimeout {
				t.Errorf("Expected TCP timeout %v, got %v", tt.expectedTCPTimeout, strategy.TCPTimeout)
			}

			if strategy.UDPTimeout != tt.expectedUDPTimeout {
				t.Errorf("Expected UDP timeout %v, got %v", tt.expectedUDPTimeout, strategy.UDPTimeout)
			}
		})
	}
}

func TestDefaultProbeConfig(t *testing.T) {
	cfg := DefaultProbeConfig()

	if cfg.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts 3, got %d", cfg.MaxAttempts)
	}
}

func TestTimeoutProgression(t *testing.T) {
	// Verify that first attempt is always more generous than retry
	first := GetProbeStrategy(0)
	retry := GetProbeStrategy(1)
	final := GetProbeStrategy(2)

	if first.TCPTimeout <= retry.TCPTimeout {
		t.Errorf("First attempt TCP timeout (%v) should be greater than retry (%v)",
			first.TCPTimeout, retry.TCPTimeout)
	}

	if retry.TCPTimeout <= final.TCPTimeout {
		t.Errorf("Retry TCP timeout (%v) should be greater than final (%v)",
			retry.TCPTimeout, final.TCPTimeout)
	}

	if first.UDPTimeout <= retry.UDPTimeout {
		t.Errorf("First attempt UDP timeout (%v) should be greater than retry (%v)",
			first.UDPTimeout, retry.UDPTimeout)
	}

	if retry.UDPTimeout <= final.UDPTimeout {
		t.Errorf("Retry UDP timeout (%v) should be greater than final (%v)",
			retry.UDPTimeout, final.UDPTimeout)
	}
}
