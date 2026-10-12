package vpn

import (
	"testing"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/nodes"
)

func TestAutoDetectDriverModeEnvOverride(t *testing.T) {
	cfg := &config.Config{OpenVPNCommand: "openvpn"}

	t.Run("explicit userland override", func(t *testing.T) {
		t.Setenv("DRIVER_MODE", "userland")
		mode := AutoDetectDriverMode(cfg)
		if mode != DriverUserland {
			t.Fatalf("expected DriverUserland, got %s", mode)
		}
	})

	t.Run("explicit zero_tun override", func(t *testing.T) {
		t.Setenv("DRIVER_MODE", "zero_tun")
		mode := AutoDetectDriverMode(cfg)
		if mode != DriverUserland {
			t.Fatalf("expected DriverUserland, got %s", mode)
		}
	})

	t.Run("missing openvpn fallback to userland", func(t *testing.T) {
		t.Setenv("DRIVER_MODE", "")
		cfgMissing := &config.Config{OpenVPNCommand: "nonexistent-openvpn-binary-12345"}
		mode := AutoDetectDriverMode(cfgMissing)
		if mode != DriverUserland {
			t.Fatalf("expected fallback to DriverUserland, got %s", mode)
		}
	})
}

func TestManagerSnapshotAndDriverMode(t *testing.T) {
	t.Setenv("DRIVER_MODE", "userland")
	cfg := &config.Config{DataDir: t.TempDir()}
	mgr := NewManager(cfg, nil, nil)

	if mgr.GetDefaultDriverMode() != DriverUserland {
		t.Fatalf("expected manager driver mode to be DriverUserland, got %s", mgr.GetDefaultDriverMode())
	}

	snap := mgr.Snapshot()
	if snap.Status != StatusDisconnected {
		t.Fatalf("expected StatusDisconnected, got %s", snap.Status)
	}
	if !snap.ZeroTUN {
		t.Fatal("expected snap.ZeroTUN to be true in userland mode")
	}
}

func TestConnectWithModeAutoDowngrade(t *testing.T) {
	t.Setenv("DRIVER_MODE", "")
	cfg := &config.Config{
		DataDir:        t.TempDir(),
		OpenVPNCommand: "nonexistent-openvpn-cmd-xyz",
	}
	mgr := NewManager(cfg, nil, nil)

	node := &nodes.Node{
		ID:           "test-node",
		IP:           "127.0.0.1",
		Port:         1422,
		SSTPPort:     1422,
		Proto:        "tcp",
		CountryShort: "JP",
	}

	// Requesting kernel mode on a system without OpenVPN binary should trigger auto-downgrade to userland
	err := mgr.ConnectWithMode(node, string(DriverKernel))
	// ConnectUserland will attempt sstp connection asynchronously and return nil (or handle in goroutine)
	if err != nil {
		t.Fatalf("expected auto-downgrade to userland without immediate sync error, got %v", err)
	}

	snap := mgr.Snapshot()
	if snap.DriverMode != DriverUserland {
		t.Fatalf("expected driver mode to be downgraded to userland, got %s", snap.DriverMode)
	}
}
