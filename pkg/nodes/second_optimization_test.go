package nodes

import (
	"aimili-vpngate-go/pkg/config"
	"context"
	"testing"
	"time"
)

func TestOptimizationProbeCancellationPreservesEvidence(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	np := NewNodePool(&config.Config{DataDir: t.TempDir()})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n := &Node{IP: "127.0.0.1", Port: 1, Proto: "tcp", LatencyMs: 27, FailCount: 3, ProbeStatus: "reachable"}
	start := time.Now()
	np.ProbeNodes(ctx, []*Node{n})
	if time.Since(start) > time.Second {
		t.Fatal("canceled probe blocked")
	}
	if n.FailCount != 3 || n.LatencyMs != 27 || n.ProbeStatus != "reachable" {
		t.Fatalf("cancellation changed evidence: %+v", n)
	}
}
