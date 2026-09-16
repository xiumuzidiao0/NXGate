package proxy

import (
	"os"
	"testing"

	"aimili-vpngate-go/pkg/config"
)

func TestPortRulesFilePermissions(t *testing.T) {
	cfg := &config.Config{
		DataDir:   t.TempDir(),
		ProxyPort: 7928,
	}
	manager := NewMultiPortManager(cfg, nil, nil)

	info, err := os.Stat(manager.rulesPath)
	if err != nil {
		t.Fatalf("stat port rules: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("expected port rules mode 0600, got %o", perm)
	}
}

func TestCustomAuthValidation(t *testing.T) {
	cfg := &config.Config{
		DataDir:   t.TempDir(),
		ProxyPort: 7928,
	}
	manager := NewMultiPortManager(cfg, nil, nil)

	// User provided, password missing -> should fail
	err := manager.ApplyRules([]PortRule{
		{
			Port:     7930,
			Enabled:  true,
			AuthMode: "custom",
			AuthUser: "user_only",
			AuthPass: "",
		},
	})
	if err == nil {
		t.Fatal("expected error for custom auth with missing password")
	}

	// Password provided, user missing -> should fail
	err = manager.ApplyRules([]PortRule{
		{
			Port:     7930,
			Enabled:  true,
			AuthMode: "custom",
			AuthUser: "",
			AuthPass: "pass_only",
		},
	})
	if err == nil {
		t.Fatal("expected error for custom auth with missing username")
	}

	// Both provided -> should succeed
	err = manager.ApplyRules([]PortRule{
		{
			Port:     7930,
			Enabled:  true,
			AuthMode: "custom",
			AuthUser: "valid_user",
			AuthPass: "valid_pass",
		},
	})
	if err != nil {
		t.Fatalf("expected success for valid custom auth: %v", err)
	}
}
