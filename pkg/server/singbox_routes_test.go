package server

import (
	"testing"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/proxy"
)

func TestIsOutboundEquivalent(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "Identical socks5 URLs",
			a:        "socks5://127.0.0.1:7928",
			b:        "socks5://127.0.0.1:7928",
			expected: true,
		},
		{
			name:     "socks vs socks5 scheme synonym",
			a:        "socks://127.0.0.1:7928",
			b:        "socks5://127.0.0.1:7928",
			expected: true,
		},
		{
			name:     "socks vs socks5 with credentials",
			a:        "socks://admin:aimilivpn@127.0.0.1:7928",
			b:        "socks5://admin:aimilivpn@127.0.0.1:7928",
			expected: true,
		},
		{
			name:     "URL encoded password vs raw password",
			a:        "socks5://admin:aimili%40123@127.0.0.1:7928",
			b:        "socks5://admin:aimili@123@127.0.0.1:7928",
			expected: true,
		},
		{
			name:     "Localhost and 127.0.0.1 alias",
			a:        "socks5://user:pass@localhost:7928",
			b:        "socks://user:pass@127.0.0.1:7928",
			expected: true,
		},
		{
			name:     "Direct equivalents",
			a:        "direct",
			b:        "none",
			expected: true,
		},
		{
			name:     "Direct vs empty",
			a:        "",
			b:        "direct",
			expected: true,
		},
		{
			name:     "Different ports should not be equivalent",
			a:        "socks5://127.0.0.1:7928",
			b:        "socks5://127.0.0.1:7929",
			expected: false,
		},
		{
			name:     "Different passwords should not be equivalent",
			a:        "socks5://admin:pass1@127.0.0.1:7928",
			b:        "socks5://admin:pass2@127.0.0.1:7928",
			expected: false,
		},
		{
			name:     "Different users should not be equivalent",
			a:        "socks5://user1:pass@127.0.0.1:7928",
			b:        "socks5://user2:pass@127.0.0.1:7928",
			expected: false,
		},
		{
			name:     "With credentials vs without credentials",
			a:        "socks5://admin:pass@127.0.0.1:7928",
			b:        "socks5://127.0.0.1:7928",
			expected: false,
		},
		{
			name:     "HTTP vs SOCKS should not be equivalent",
			a:        "http://127.0.0.1:7928",
			b:        "socks5://127.0.0.1:7928",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isOutboundEquivalent(tc.a, tc.b)
			if got != tc.expected {
				t.Errorf("isOutboundEquivalent(%q, %q) = %v; want %v", tc.a, tc.b, got, tc.expected)
			}
		})
	}
}

func TestResolveOutboundURLWithAuth(t *testing.T) {
	cfg := &config.Config{
		DataDir:   t.TempDir(),
		ProxyUser: "vpn_rand",
		ProxyPass: "super%secret@123",
		ProxyPort: 7928,
	}

	s := &Server{
		cfg: cfg,
	}

	// 1. Direct
	if got := s.resolveOutboundURL("direct"); got != "direct" {
		t.Errorf("expected direct, got %s", got)
	}

	// 2. Default random proxy auth enabled
	got := s.resolveOutboundURL("7928")
	expected := "socks5://vpn_rand:super%25secret%40123@127.0.0.1:7928"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}

	// 3. Custom rule with user and password
	portMgr := proxy.NewMultiPortManager(cfg, nil, nil)
	_ = portMgr.ApplyRules([]proxy.PortRule{
		{
			Port:     7930,
			Enabled:  true,
			AuthMode: "custom",
			AuthUser: "custom_user",
			AuthPass: "custom_pass",
		},
		{
			Port:     7931,
			Enabled:  true,
			AuthMode: "none",
		},
	})
	s.portMgr = portMgr

	gotCustom := s.resolveOutboundURL("7930")
	if gotCustom != "socks5://custom_user:custom_pass@127.0.0.1:7930" {
		t.Errorf("expected custom auth url, got %s", gotCustom)
	}

	gotNone := s.resolveOutboundURL("7931")
	if gotNone != "socks5://127.0.0.1:7931" {
		t.Errorf("expected unauthenticated url, got %s", gotNone)
	}
}
