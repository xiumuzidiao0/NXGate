package vpn

import (
	"net"
	"testing"
	"time"

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

// TestPrimaryFailoverDelegateTakesPrecedence verifies that a governing dynamic
// group (country / IP-type / purity constraints) is consulted before the
// unconstrained global best-node pick, and that no global selection happens when
// the delegate claims ownership of primary egress recovery.
func TestPrimaryFailoverDelegateTakesPrecedence(t *testing.T) {
	t.Setenv("DRIVER_MODE", "userland")
	cfg := &config.Config{DataDir: t.TempDir()}
	mgr := NewManager(cfg, nil, nil)

	called := make(chan struct{}, 1)
	mgr.SetPrimaryFailoverDelegate(func() bool {
		called <- struct{}{}
		return true
	})

	mgr.TriggerAutoFailover()

	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("failover delegate was never consulted")
	}

	snap := mgr.Snapshot()
	if snap.Status != StatusReconnecting {
		t.Fatalf("expected StatusReconnecting while the delegate owns recovery, got %s", snap.Status)
	}
	// pool is nil: reaching the built-in pick would panic, so a clean return also
	// proves the global selection path was skipped.
	if snap.ReconnectCount != 0 {
		t.Fatalf("expected delegate-owned failover to skip the built-in reconnect counter, got %d", snap.ReconnectCount)
	}
}

// TestProbeUserlandEgressRejectsNilTunnel ensures a missing tunnel is never
// reported as a working exit, so the connect path cannot publish a dead primary.
func TestProbeUserlandEgressRejectsNilTunnel(t *testing.T) {
	if probeUserlandEgress(nil) {
		t.Fatal("a nil tunnel must never be reported as a working exit")
	}
}

// TestUserlandEgressProbesUseLiteralIPs guards the probe design: the check must
// measure packet forwarding only, never DNS resolution.
func TestUserlandEgressProbesUseLiteralIPs(t *testing.T) {
	if len(userlandEgressProbes) < 2 {
		t.Fatalf("expected several independent egress probes, got %v", userlandEgressProbes)
	}
	for _, target := range userlandEgressProbes {
		host, _, err := net.SplitHostPort(target)
		if err != nil {
			t.Fatalf("probe target %q is not host:port: %v", target, err)
		}
		if net.ParseIP(host) == nil {
			t.Fatalf("probe target %q must use a literal IP so DNS cannot mask a dead exit", target)
		}
	}
}
