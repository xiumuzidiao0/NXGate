package tunnel

import (
	"context"
	"sync"
	"time"
)

const healthWorkers = 4
const healthRoundBudget = 8 * time.Second
const healthResultTTL = 8 * time.Second
const userlandHealthResultTTL = 45 * time.Second

type tunnelHealth struct {
	checked             time.Time
	failures, successes int
	blocked             bool
}

// healthBatch probes each tunnel identity at most once. A replacement with the same
// device/ID does not inherit evidence. Only the round owns this map (roundMu).
// Nil/expired contexts are unknown, not proof of a broken egress.
func (m *DynamicGroupManager) healthBatch(ctx context.Context, tunnels []*Tunnel) map[*Tunnel]bool {
	if m.health == nil {
		m.health = make(map[*Tunnel]*tunnelHealth)
	}
	results := make(map[*Tunnel]bool)
	unique := make(map[*Tunnel]bool)
	for _, t := range tunnels {
		if t == nil {
			continue
		}
		if t.IsHealthy() {
			unique[t] = true
		} else if t.IsUserland {
			results[t] = false
		}
	}
	for t := range m.health {
		if !unique[t] {
			delete(m.health, t)
		}
	}
	jobs := make(chan *Tunnel)
	var mu sync.Mutex
	var wg sync.WaitGroup
	probe := m.healthProbe
	if probe == nil {
		probe = probeTunnelConnectivity
	}
	userlandProbe := m.userlandHealthProbe
	if userlandProbe == nil {
		userlandProbe = probeDialerConnectivity
	}
	for w := 0; w < healthWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range jobs {

				mu.Lock()
				state := m.health[t]
				if state == nil {
					state = &tunnelHealth{}
					m.health[t] = state
				}
				results[t] = !state.blocked
				ttl := healthResultTTL
				if t.IsUserland {
					ttl = userlandHealthResultTTL
				}
				fresh := time.Since(state.checked) < ttl
				mu.Unlock()
				if !fresh {
					probeTimeout := 6 * time.Second
					if t.IsUserland {
						// 3 targets * 2s per target <= 6s; keep context timeout slightly above 6s
						// so all-timeout black-holes return (false, nil) instead of context.DeadlineExceeded.
						probeTimeout = 7 * time.Second
					}
					probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
					var ok bool
					var err error
					if t.IsUserland {
						ok, err = userlandProbe(probeCtx, t.Dialer, 2*time.Second)
					} else {
						ok, err = probe(probeCtx, t.DevName, 2*time.Second)
					}
					cancel()
					// The batch budget/caller cancellation is not a failed health sample.
					if ctx.Err() != nil || err != nil {
						continue
					}
					state.checked = time.Now()
					if ok {
						state.failures = 0
						state.successes++
						// Two successes to recover a health-trip; never clear another cause's breaker.
						if state.successes >= 2 {
							state.blocked = false
						}
					} else {
						state.successes = 0
						state.failures++
						if state.failures >= 2 && !state.blocked {
							state.blocked = true
							t.TripCircuitBreaker(60 * time.Second)
						}
					}
				}
				mu.Lock()
				results[t] = !state.blocked
				mu.Unlock()
			}
		}()
	}
	for t := range unique {
		select {
		case jobs <- t:
		case <-ctx.Done():
		}
	}
	close(jobs)
	wg.Wait()
	return results
}

func waitContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
