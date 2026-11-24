package server

import (
	"bufio"
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/nodes"
	"aimili-vpngate-go/pkg/stats"
	"aimili-vpngate-go/pkg/tunnel"
	"aimili-vpngate-go/pkg/vpn"
)

func TestSnapshotsStripConfigDataWithoutMutatingSource(t *testing.T) {
	srcNode := &nodes.Node{
		ID:         "node-pem-1",
		IP:         "198.51.100.1",
		ConfigData: "remote 198.51.100.1 443\n<cert>SECRET-PEM</cert>",
	}

	tun := &tunnel.Tunnel{
		ID:      "tun-0",
		DevName: "tun0",
		Node:    srcNode,
		Status:  tunnel.StatusConnected,
	}
	tunSnap := tun.Snapshot()
	if tunSnap.Node == nil || tunSnap.Node.ID != "node-pem-1" {
		t.Fatalf("expected cloned node metadata, got %+v", tunSnap.Node)
	}
	if tunSnap.Node.ConfigData != "" {
		t.Fatalf("expected Tunnel.Snapshot() to strip ConfigData, got %q", tunSnap.Node.ConfigData)
	}
	if srcNode.ConfigData == "" {
		t.Fatalf("Tunnel.Snapshot() mutated source node ConfigData")
	}

	cfg := &config.Config{DataDir: t.TempDir()}
	np := nodes.NewNodePool(cfg)
	tp := tunnel.NewPool(cfg, np)
	vpnMgr := vpn.NewManager(cfg, np, tp)
	vpnMgr.SetActiveNodeForTest(srcNode)

	vpnSnap := vpnMgr.Snapshot()
	if vpnSnap.ActiveNode == nil || vpnSnap.ActiveNode.ID != "node-pem-1" {
		t.Fatalf("expected active node metadata in snapshot, got %+v", vpnSnap.ActiveNode)
	}
	if vpnSnap.ActiveNode.ConfigData != "" {
		t.Fatalf("expected vpn.Manager.Snapshot() to strip ConfigData, got %q", vpnSnap.ActiveNode.ConfigData)
	}
	if srcNode.ConfigData == "" {
		t.Fatalf("vpn.Manager.Snapshot() mutated source node ConfigData")
	}
}

func TestSSEHubSharedBroadcasterLifecycle(t *testing.T) {
	cfg := &config.Config{
		DataDir:   t.TempDir(),
		UIHost:    "127.0.0.1",
		UIPort:    8787,
		ProxyHost: "127.0.0.1",
		ProxyPort: 7928,
	}
	np := nodes.NewNodePool(cfg)
	tp := tunnel.NewPool(cfg, np)
	vpnMgr := vpn.NewManager(cfg, np, tp)
	vpnMgr.SetActiveNodeForTest(&nodes.Node{
		ID:         "node-sse-1",
		IP:         "198.51.100.2",
		ConfigData: "SECRET-CERT-DATA",
	})
	srv := NewServer(cfg, np, vpnMgr, tp, nil, nil, nil)
	hub := newSSEHubWithInterval(srv, 25*time.Millisecond)

	startClient := func() (context.CancelFunc, *httptest.ResponseRecorder, <-chan struct{}) {
		ctx, cancel := context.WithCancel(context.Background())
		req := httptest.NewRequest("GET", "/api/events", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		done := make(chan struct{})
		go func() {
			defer close(done)
			hub.HandleEvents(rec, req)
		}()
		return cancel, rec, done
	}

	cancel1, rec1, done1 := startClient()
	cancel2, rec2, done2 := startClient()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		hub.mu.Lock()
		n := len(hub.clients)
		hub.mu.Unlock()
		if n == 2 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Give runBroadcaster a moment to subscribe to RingLog before emitting the test log.
	time.Sleep(30 * time.Millisecond)
	stats.LogInfo("SSETest", "shared-broadcast-check")
	time.Sleep(80 * time.Millisecond)

	cancel1()
	cancel2()
	<-done1
	<-done2

	body1 := rec1.Body.String()
	body2 := rec2.Body.String()

	for idx, body := range []string{body1, body2} {
		scanner := bufio.NewScanner(strings.NewReader(body))
		var hasConnected, hasStatus, hasLog bool
		for scanner.Scan() {
			line := scanner.Text()
			if line == ": connected" {
				hasConnected = true
			}
			if line == "event: status" {
				hasStatus = true
			}
			if line == "event: log" {
				hasLog = true
			}
		}
		if !hasConnected || !hasStatus || !hasLog {
			t.Fatalf("client %d missing SSE events (connected=%v status=%v log=%v) body:\n%s", idx+1, hasConnected, hasStatus, hasLog, body)
		}
		if strings.Contains(body, "SECRET-CERT-DATA") {
			t.Fatalf("client %d received leaked ConfigData in SSE status payload", idx+1)
		}
	}

	hub.mu.Lock()
	activeStop := hub.stopCh
	activeClients := len(hub.clients)
	hub.mu.Unlock()
	if activeClients != 0 || activeStop != nil {
		t.Fatalf("expected hub broadcaster stopped after disconnect, clients=%d stopCh=%v", activeClients, activeStop)
	}
}
