package tunnel

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/nodes"
)

type noopCloser struct {
	closed bool
}

func (c *noopCloser) Close() error {
	c.closed = true
	return nil
}

func TestProbeNodeDeepEphemeralSSTPAndPoolIsolation(t *testing.T) {
	cfg := &config.Config{DataDir: t.TempDir()}
	np := nodes.NewNodePool(cfg)
	pool := NewPool(cfg, np)

	// Start a local listener so TCP probe & mock dialer complete immediately offline.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	tcpAddr := ln.Addr().(*net.TCPAddr)

	okNode := &nodes.Node{
		ID:           "node-deep-ok",
		IP:           "127.0.0.1",
		Port:         tcpAddr.Port,
		Proto:        "tcp",
		CountryShort: "JP",
		FirstSeen:    time.Now(),
		LastSeen:     time.Now(),
	}
	partialNode := &nodes.Node{
		ID:           "node-deep-partial",
		IP:           "127.0.0.1",
		Port:         tcpAddr.Port,
		Proto:        "tcp",
		CountryShort: "US",
		FirstSeen:    time.Now(),
		LastSeen:     time.Now(),
	}
	failNode := &nodes.Node{
		ID:           "node-deep-fail",
		IP:           "127.0.0.1",
		Port:         1,
		Proto:        "udp",
		CountryShort: "KR",
		FirstSeen:    time.Now(),
		LastSeen:     time.Now(),
	}
	np.SetCandidatesForTest([]*nodes.Node{okNode, partialNode, failNode})

	var lastCloser *noopCloser
	pool.sstpProbeDialer = func(ctx context.Context, n *nodes.Node) (func(context.Context, string, string) (net.Conn, error), interface{ Close() error }, int, error) {
		if n.ID == "node-deep-fail" {
			return nil, nil, 0, errors.New("mock sstp handshake refused")
		}
		cl := &noopCloser{}
		lastCloser = cl
		dialer := func(dctx context.Context, network, addr string) (net.Conn, error) {
			if n.ID == "node-deep-partial" {
				return nil, errors.New("mock relay blackhole")
			}
			// Succeed on literal connectivity targets; fail fast on unlock HTTPS hosts so tests run offline quickly.
			for _, target := range UserlandConnectivityTargets {
				if addr == target {
					return net.DialTimeout("tcp", ln.Addr().String(), time.Second)
				}
			}
			return nil, errors.New("offline test dialer rejects external internet")
		}
		return dialer, cl, tcpAddr.Port, nil
	}

	// Register a hidden IsProbe tunnel to confirm ListTunnels/GetHealthyTunnels exclude probe tunnels.
	pool.mu.Lock()
	pool.tunnels["probe-hidden"] = &Tunnel{
		ID:         "probe-hidden",
		DevName:    "tun9",
		DevIndex:   9,
		Status:     StatusConnected,
		IsProbe:    true,
		IsUserland: true,
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			return nil, io.EOF
		},
	}
	pool.mu.Unlock()
	if got := pool.ListTunnels(); len(got) != 0 {
		t.Fatalf("expected ListTunnels to ignore IsProbe tunnels, got %d", len(got))
	}
	if got := pool.GetHealthyTunnels(nil); len(got) != 0 {
		t.Fatalf("expected GetHealthyTunnels to ignore IsProbe tunnels, got %d", len(got))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Healthy SSTP node
	resOK := pool.ProbeNodeDeep(ctx, okNode)
	if resOK.Status != "ok" || resOK.Driver != "sstp" || !resOK.TCPReachable || !resOK.TunnelConnected || !resOK.EgressReachable {
		t.Fatalf("expected ok deep probe result, got %+v", resOK)
	}
	if resOK.TCPLatencyMs <= 0 || resOK.EgressLatencyMs <= 0 {
		t.Fatalf("expected positive TCP & egress latencies, got tcp=%d egress=%d", resOK.TCPLatencyMs, resOK.EgressLatencyMs)
	}
	if lastCloser == nil || !lastCloser.closed {
		t.Fatal("expected ephemeral probe closer to be closed after ProbeNodeDeep")
	}
	persistedOK := np.GetNodeByID("node-deep-ok")
	if persistedOK == nil || persistedOK.ProbeStatus != "connected" || persistedOK.SSTPPort != tcpAddr.Port {
		t.Fatalf("expected persisted connected status and sstp port, got %+v", persistedOK)
	}

	// 2. Partial node (handshake ok, forwarding blocked) - must NOT blacklist
	resPartial := pool.ProbeNodeDeep(ctx, partialNode)
	if resPartial.Status != "partial" || !resPartial.TunnelConnected || resPartial.EgressReachable {
		t.Fatalf("expected partial status when egress blocked, got %+v", resPartial)
	}
	if np.Blacklist().IsNodeBlocked(partialNode.ID, partialNode.IP) {
		t.Fatal("manual deep probe must never blacklist partial nodes")
	}

	// 3. Handshake failure - must NOT blacklist
	resFail := pool.ProbeNodeDeep(ctx, failNode)
	if resFail.Status != "failed" || resFail.Driver != "tcp_only" || resFail.TunnelConnected {
		t.Fatalf("expected failed tcp_only result, got %+v", resFail)
	}
	if np.Blacklist().IsNodeBlocked(failNode.ID, failNode.IP) {
		t.Fatal("manual deep probe must never blacklist failed nodes")
	}
}

func TestProbeNodeDeepReusesExistingHealthyTunnel(t *testing.T) {
	cfg := &config.Config{DataDir: t.TempDir()}
	np := nodes.NewNodePool(cfg)
	pool := NewPool(cfg, np)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()

	existingNode := &nodes.Node{
		ID:           "node-existing",
		IP:           "198.51.100.77",
		Port:         443,
		Proto:        "udp",
		CountryShort: "SG",
		FirstSeen:    time.Now(),
		LastSeen:     time.Now(),
	}
	np.SetCandidatesForTest([]*nodes.Node{existingNode})

	pool.mu.Lock()
	pool.tunnels["sstp-1"] = &Tunnel{
		ID:         "sstp-1",
		DevName:    "sstp1",
		DevIndex:   1,
		Node:       existingNode,
		Status:     StatusConnected,
		IsUserland: true,
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			for _, target := range UserlandConnectivityTargets {
				if addr == target {
					return net.DialTimeout("tcp", ln.Addr().String(), time.Second)
				}
			}
			return nil, errors.New("offline test dialer")
		},
	}
	pool.mu.Unlock()

	pool.sstpProbeDialer = func(context.Context, *nodes.Node) (func(context.Context, string, string) (net.Conn, error), interface{ Close() error }, int, error) {
		t.Fatal("sstpProbeDialer should not be called when healthy tunnel already exists")
		return nil, nil, 0, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res := pool.ProbeNodeDeep(ctx, existingNode)
	if res.Status != "ok" || res.Driver != "existing" || !res.TunnelConnected || !res.EgressReachable || res.EgressLatencyMs <= 0 {
		t.Fatalf("expected existing tunnel reuse success, got %+v", res)
	}
}
