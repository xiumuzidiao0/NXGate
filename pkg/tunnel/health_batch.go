package tunnel

import (
	"context"
	"sync"
	"time"
)

const healthWorkers = 4
const healthRoundBudget = 8 * time.Second
const healthResultTTL = 8 * time.Second

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
	unique := make(map[*Tunnel]bool)
	for _, t := range tunnels {
		if t != nil && !t.IsUserland && t.IsHealthy() {
			unique[t] = true
		}
	}
	for t := range m.health {
		if !unique[t] {
			delete(m.health, t)
		}
	}
	results := make(map[*Tunnel]bool)
	jobs := make(chan *Tunnel)
	var mu sync.Mutex
	var wg sync.WaitGroup
	probe := m.healthProbe
	if probe == nil {
		probe = probeTunnelConnectivity
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
				fresh := time.Since(state.checked) < healthResultTTL
				mu.Unlock()
				if !fresh {
					probeCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
					ok, err := probe(probeCtx, t.DevName, 2*time.Second)
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
