package config

import (
	"os"
	"testing"
)

func TestLoadConfigUsesLoopbackAndSecureDataDir(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("UI_HOST", "")
	t.Setenv("LOCAL_PROXY_HOST", "")

	cfg := LoadConfig()
	if cfg.UIHost != "127.0.0.1" {
		t.Fatalf("expected loopback UI host, got %q", cfg.UIHost)
	}
	if cfg.ProxyHost != "127.0.0.1" {
		t.Fatalf("expected loopback proxy host, got %q", cfg.ProxyHost)
	}

	info, err := os.Stat(dataDir)
	if err != nil {
		t.Fatalf("stat data dir: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Fatalf("expected data dir mode 0700, got %o", perm)
	}
}

func TestUpdateSettingsRejectsEnvInjection(t *testing.T) {
	cfg := &Config{DataDir: t.TempDir()}

	badCases := []struct {
		name string
		dto  SettingsDTO
	}{
		{
			name: "UIUsername injection",
			dto:  SettingsDTO{UIUsername: "admin\nEVIL=1"},
		},
		{
			name: "SubToken injection",
			dto:  SettingsDTO{SubToken: "token\r\nEVIL=1"},
		},
		{
			name: "AgePublicKey injection",
			dto:  SettingsDTO{AgePublicKey: "age1xyz\nEVIL=1"},
		},
		{
			name: "AutoRotateIPType injection",
			dto:  SettingsDTO{AutoRotateIPType: "all\nEVIL=1"},
		},
		{
			name: "DiscoveryCountries injection",
			dto:  SettingsDTO{DiscoveryCountries: []string{"JP", "US\nEVIL=1"}},
		},
	}

	for _, tc := range badCases {
		t.Run(tc.name, func(t *testing.T) {
			err := cfg.UpdateSettings(tc.dto)
			if err == nil {
				t.Fatalf("expected newline in %s to be rejected", tc.name)
			}
		})
	}
}

func TestLoadConfigFallbackEnvVars(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("LOCAL_PROXY_HOST", "")
	t.Setenv("PROXY_HOST", "192.168.1.10")
	t.Setenv("LOCAL_PROXY_PORT", "")
	t.Setenv("PROXY_PORT", "8888")
	t.Setenv("LOCAL_PROXY_USER", "")
	t.Setenv("PROXY_USER", "custom_user")
	t.Setenv("LOCAL_PROXY_PASS", "")
	t.Setenv("PROXY_PASS", "custom_pass")
	t.Setenv("LOCAL_PROXY_MAX_CONNECTIONS", "")
	t.Setenv("PROXY_MAX_CONNECTIONS", "256")
	t.Setenv("FETCH_INTERVAL_SECONDS", "")
	t.Setenv("FETCH_INTERVAL_MINUTES", "20")
	t.Setenv("OPENVPN_CMD", "")
	t.Setenv("OPENVPN_COMMAND", "/usr/bin/openvpn-custom")

	cfg := LoadConfig()
	if cfg.ProxyHost != "192.168.1.10" {
		t.Fatalf("expected PROXY_HOST fallback 192.168.1.10, got %q", cfg.ProxyHost)
	}
	if cfg.ProxyPort != 8888 {
		t.Fatalf("expected PROXY_PORT fallback 8888, got %d", cfg.ProxyPort)
	}
	if cfg.ProxyUser != "custom_user" {
		t.Fatalf("expected PROXY_USER fallback custom_user, got %q", cfg.ProxyUser)
	}
	if cfg.ProxyPass != "custom_pass" {
		t.Fatalf("expected PROXY_PASS fallback custom_pass, got %q", cfg.ProxyPass)
	}
	if cfg.ProxyMaxConnections != 256 {
		t.Fatalf("expected PROXY_MAX_CONNECTIONS fallback 256, got %d", cfg.ProxyMaxConnections)
	}
	if cfg.FetchInterval.Seconds() != 1200 {
		t.Fatalf("expected FETCH_INTERVAL_MINUTES fallback 1200s, got %v", cfg.FetchInterval)
	}
	if cfg.OpenVPNCommand != "/usr/bin/openvpn-custom" {
		t.Fatalf("expected OPENVPN_COMMAND fallback /usr/bin/openvpn-custom, got %q", cfg.OpenVPNCommand)
	}
}
