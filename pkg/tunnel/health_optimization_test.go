package tunnel

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestOptimizationHealthBoundedParallelAndShared(t *testing.T) {
	var active, peak, calls atomic.Int32
	m := &DynamicGroupManager{healthProbe: func(ctx context.Context, _ string, _ time.Duration) (bool, error) {
		calls.Add(1)
		n := active.Add(1)
		defer active.Add(-1)
		for {
			old := peak.Load()
			if old >= n || peak.CompareAndSwap(old, n) {
				break
			}
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(15 * time.Millisecond):
			return true, nil
		}
	}}
	var ts []*Tunnel
	for i := 0; i < 12; i++ {
		tunnel := &Tunnel{ID: fmt.Sprint(i), Status: StatusConnected, DevName: "fake"}
		ts = append(ts, tunnel, tunnel)
	}
	res := m.healthBatch(context.Background(), ts)
	if len(res) != 12 || calls.Load() != 12 || peak.Load() > healthWorkers || peak.Load() < 2 {
		t.Fatal(len(res), calls.Load(), peak.Load())
	}
	m.healthBatch(context.Background(), ts)
	if calls.Load() != 12 {
		t.Fatal("recent results not reused")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m.healthBatch(ctx, ts)
	if active.Load() != 0 {
		t.Fatal("workers leaked")
	}
}
func TestOptimizationHealthHysteresisUnknownAndRecovery(t *testing.T) {
	tun := &Tunnel{ID: "one", Status: StatusConnected, DevName: "fake"}
	ok := false
	var probeErr error
	m := &DynamicGroupManager{healthProbe: func(context.Context, string, time.Duration) (bool, error) { return ok, probeErr }}
	sample := func() bool {
		if m.health[tun] != nil {
			m.health[tun].checked = time.Time{}
		}
		return m.healthBatch(context.Background(), []*Tunnel{tun})[tun]
	}
	if !sample() {
		t.Fatal("single failure tripped")
	}
	probeErr = context.DeadlineExceeded
	if !sample() || m.health[tun].failures != 1 {
		t.Fatal("timeout counted as failure")
	}
	probeErr = nil
	if sample() || tun.IsAvailable() {
		t.Fatal("consecutive negatives did not trip")
	}
	ok = true
	if sample() {
		t.Fatal("single recovery bypassed hysteresis")
	}
	if !sample() {
		t.Fatal("two successes failed recovery")
	}
	if tun.IsAvailable() {
		t.Fatal("cleared breaker too soon")
	}
	replacement := &Tunnel{ID: "one", Status: StatusConnected, DevName: "fake"}
	m.healthBatch(context.Background(), []*Tunnel{replacement})
	if len(m.health) != 1 || m.health[replacement] == nil {
		t.Fatal("identity cache survived replacement")
	}
}
func TestOptimizationHealthBudgetCancellation(t *testing.T) {
	m := &DynamicGroupManager{healthProbe: func(ctx context.Context, _ string, _ time.Duration) (bool, error) {
		<-ctx.Done()
		return false, ctx.Err()
	}}
	tun := &Tunnel{Status: StatusConnected, DevName: "fake"}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	m.healthBatch(ctx, []*Tunnel{tun})
	if time.Since(start) > time.Second {
		t.Fatal("cancellation blocked")
	}
	if m.health[tun].failures != 0 || !tun.IsAvailable() {
		t.Fatal("budget treated as failure")
	}
}
func TestOptimizationEvaluationQueueBoundedDeduplicated(t *testing.T) {
	m := &DynamicGroupManager{evalQueue: make(chan string, 2), evalPending: make(map[string]bool)}
	m.enqueueEvaluation("a")
	m.enqueueEvaluation("a")
	m.enqueueEvaluation("b")
	m.enqueueEvaluation("c")
	if len(m.evalQueue) != 2 || len(m.evalPending) != 2 {
		t.Fatal("queue not bounded/deduped")
	}
	if <-m.evalQueue != "a" || <-m.evalQueue != "b" {
		t.Fatal("not FIFO")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() { m.evaluationWorker(ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker leaked")
	}
}

func TestOptimizationGroupSnapshotsOwnSlices(t *testing.T) {
	m := &DynamicGroupManager{groups: map[string]*DynamicGroup{"fixture": {ID: "fixture", ActiveTunnelIDs: []string{"original"}}}}
	m.GetGroup("fixture").ActiveTunnelIDs[0] = "changed"
	m.ListGroups()[0].ActiveTunnelIDs[0] = "changed again"
	if got := m.GetGroup("fixture").ActiveTunnelIDs[0]; got != "original" {
		t.Fatal("caller mutated manager storage", got)
	}
}

func TestOptimizationUserlandHealthBatchDetectsBlackhole(t *testing.T) {
	var calls atomic.Int32
	healthyFlag := true
	userlandOK := false
	tun := &Tunnel{
		ID:          "tun-sstp-1",
		DevName:     "sstp1",
		Status:      StatusConnected,
		IsUserland:  true,
		HealthCheck: func() bool { return healthyFlag },
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			return nil, fmt.Errorf("dead dialer")
		},
	}
	m := &DynamicGroupManager{
		userlandHealthProbe: func(_ context.Context, _ func(context.Context, string, string) (net.Conn, error), _ time.Duration) (bool, error) {
			calls.Add(1)
			return userlandOK, nil
		},
	}

	// First failure does not trip immediately (hysteresis).
	res := m.healthBatch(context.Background(), []*Tunnel{tun})
	if !res[tun] || calls.Load() != 1 {
		t.Fatalf("expected first probe failure to tolerate transient error and run once, got res=%v calls=%d", res[tun], calls.Load())
	}

	// Cached within userlandHealthResultTTL (45s).
	res = m.healthBatch(context.Background(), []*Tunnel{tun})
	if calls.Load() != 1 || !res[tun] {
		t.Fatalf("expected second call within TTL to reuse cached probe result, calls=%d", calls.Load())
	}

	// Expire cache: second consecutive failure trips the circuit breaker.
	m.health[tun].checked = time.Time{}
	res = m.healthBatch(context.Background(), []*Tunnel{tun})
	if res[tun] || tun.IsAvailable() {
		t.Fatalf("expected two consecutive userland probe failures to block tunnel and trip circuit breaker")
	}

	// If underlying session dies, healthBatch immediately marks false without even probing.
	healthyFlag = false
	res = m.healthBatch(context.Background(), []*Tunnel{tun})
	if res[tun] || calls.Load() != 2 {
		t.Fatalf("expected dead userland session to immediately return false without probing, calls=%d", calls.Load())
	}
}
