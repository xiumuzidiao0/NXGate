package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigUsesPublicWebLoopbackProxyAndSecureDataDir(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("UI_HOST", "")
	t.Setenv("LOCAL_PROXY_HOST", "")

	cfg := LoadConfig()
	if cfg.UIHost != "0.0.0.0" {
		t.Fatalf("expected all-IPv4 UI host, got %q", cfg.UIHost)
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

func TestLoadConfigPreservesExplicitUIHost(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "192.0.2.10", "::1", "::"} {
		t.Run(host, func(t *testing.T) {
			t.Setenv("DATA_DIR", t.TempDir())
			t.Setenv("UI_HOST", host)
			if got := LoadConfig().UIHost; got != host {
				t.Fatalf("explicit UI_HOST = %q, want %q", got, host)
			}
		})
	}
}

func TestWebAccessURL(t *testing.T) {
	for _, tc := range []struct{ host, path, want string }{
		{"", "/enter/", "http://<SERVER-IP>:8787/enter/"},
		{"0.0.0.0", "/enter/", "http://<SERVER-IP>:8787/enter/"},
		{"::", "/enter/", "http://<SERVER-IP>:8787/enter/"},
		{"127.0.0.1", "/enter/", "http://127.0.0.1:8787/enter/"},
		{"::1", "/enter/", "http://[::1]:8787/enter/"},
		{"[2001:db8::1]", "/enter/", "http://[2001:db8::1]:8787/enter/"},
		{"console.example.com", "/enter/", "http://console.example.com:8787/enter/"},
		{"0.0.0.0", "", "http://<SERVER-IP>:8787/"},
	} {
		got := WebAccessURL(tc.host, 8787, tc.path)
		if got != tc.want {
			t.Errorf("host %q path %q: got %q, want %q", tc.host, tc.path, got, tc.want)
		}
	}
}

func TestLoadConfigExplicitEmptyDisablesPathAndAuth(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("UI_PATH", "")
	t.Setenv("UI_USERNAME", "")
	t.Setenv("UI_PASSWORD", "")

	cfg := LoadConfig()
	if cfg.UIPath != "" {
		t.Fatalf("expected UIPath to be disabled when UI_PATH='', got %q", cfg.UIPath)
	}
	if cfg.UIUsername != "" || cfg.UIPassword != "" || cfg.IsUIAuthEnabled() {
		t.Fatalf("expected UI auth to be disabled when UI_USERNAME='' and UI_PASSWORD='', got user=%q auth=%v", cfg.UIUsername, cfg.IsUIAuthEnabled())
	}
	if _, err := os.Stat(filepath.Join(dataDir, "admin_path.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected admin_path.txt not to be created when UI_PATH='', err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "ui_password.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected ui_password.txt not to be created when UI_PASSWORD='', err=%v", err)
	}

	// Persisting unrelated settings while path & auth are disabled must survive restart cleanly.
	dto := cfg.GetSettings()
	dto.TelegramChatID = "123456"
	if err := cfg.UpdateSettings(dto); err != nil {
		t.Fatalf("UpdateSettings failed with auth disabled: %v", err)
	}
	cfg2 := LoadConfig()
	if cfg2.UIPath != "" || cfg2.IsUIAuthEnabled() || cfg2.TelegramChatID != "123456" {
		t.Fatalf("unexpected reloaded config: path=%q auth=%v chat=%q", cfg2.UIPath, cfg2.IsUIAuthEnabled(), cfg2.TelegramChatID)
	}

	// Persisted disabled path & auth must also take precedence if env vars are later removed.
	unsetEnvForTest(t, "UI_PATH")
	unsetEnvForTest(t, "ADMIN_SECRET_PATH")
	unsetEnvForTest(t, "UI_USERNAME")
	unsetEnvForTest(t, "UI_PASSWORD")
	cfg3 := LoadConfig()
	if cfg3.UIPath != "" || cfg3.UIUsername != "" || cfg3.UIPassword != "" || cfg3.IsUIAuthEnabled() {
		t.Fatalf("expected persisted disabled path & auth without env vars, got path=%q user=%q auth=%v", cfg3.UIPath, cfg3.UIUsername, cfg3.IsUIAuthEnabled())
	}
}

func TestUpdateSettingsWithPresenceAndPublicExposure(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	unsetEnvForTest(t, "UI_PATH")
	unsetEnvForTest(t, "ADMIN_SECRET_PATH")
	unsetEnvForTest(t, "UI_USERNAME")
	unsetEnvForTest(t, "UI_PASSWORD")
	t.Setenv("UI_HOST", "0.0.0.0")

	cfg := LoadConfig()
	if cfg.UIPath == "" || !cfg.IsUIAuthEnabled() || cfg.IsPublicUnauthenticated() {
		t.Fatalf("unexpected initial state: path=%q auth=%v publicUnauth=%v", cfg.UIPath, cfg.IsUIAuthEnabled(), cfg.IsPublicUnauthenticated())
	}

	// Half-empty credentials via explicit username clearing must fail closed.
	err := cfg.UpdateSettingsWithPresence(SettingsDTO{UIUsername: ""}, map[string]bool{"ui_username": true}, nil)
	if !errors.Is(err, ErrInvalidSettings) {
		t.Fatalf("expected ErrInvalidSettings for half-empty auth, got %v", err)
	}
	if !cfg.IsUIAuthEnabled() || cfg.UIUsername == "" {
		t.Fatalf("failed validation mutated config")
	}

	// Explicitly clearing ui_path while disabling auth must succeed and persist cleanly.
	disable := true
	err = cfg.UpdateSettingsWithPresence(SettingsDTO{UIPath: ""}, map[string]bool{"ui_path": true}, &disable)
	if err != nil {
		t.Fatalf("expected clean disable update, got %v", err)
	}
	if cfg.UIPath != "" || cfg.IsUIAuthEnabled() || !cfg.IsPublicUnauthenticated() {
		t.Fatalf("expected empty path and public unauthenticated state, got path=%q auth=%v publicUnauth=%v",
			cfg.UIPath, cfg.IsUIAuthEnabled(), cfg.IsPublicUnauthenticated())
	}
	if _, err := os.Stat(filepath.Join(dataDir, "admin_path.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected stale admin_path.txt removed, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "ui_password.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected stale ui_password.txt removed, err=%v", err)
	}
}
