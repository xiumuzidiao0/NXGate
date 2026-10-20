package proxy

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"aimili-vpngate-go/pkg/tunnel"
)

type PortPolicy string

const (
	PolicyRoundRobin PortPolicy = "round_robin"
	PolicyRandom     PortPolicy = "random"
	PolicyInterval   PortPolicy = "interval"
	PolicyLeastRTT   PortPolicy = "least_rtt"
)

type TunnelSelector interface {
	SelectTunnel(port int, boundIDs []string, boundGroupIDs []string, policy PortPolicy, intervalSec int) *tunnel.Tunnel
}

type healthyTunnelPool interface {
	GetHealthyTunnels([]string) []*tunnel.Tunnel
}

type DefaultScheduler struct {
	pool       healthyTunnelPool
	dynamicMgr *tunnel.DynamicGroupManager
	mu         sync.Mutex
	states     map[int]*scheduleState
	now        func() time.Time
}

type scheduleState struct {
	signature string
	counter   uint64
	selected  string
	expires   time.Time
}

// ResetPort discards a removed or replaced rule's routing state.
func (s *DefaultScheduler) ResetPort(port int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, port)
}

func NewScheduler(pool *tunnel.Pool, dm *tunnel.DynamicGroupManager) *DefaultScheduler {
	if pool == nil {
		return &DefaultScheduler{dynamicMgr: dm}
	}
	return &DefaultScheduler{
		pool:       pool,
		dynamicMgr: dm,
	}
}

func (s *DefaultScheduler) SelectTunnel(port int, boundIDs []string, boundGroupIDs []string, policy PortPolicy, intervalSec int) *tunnel.Tunnel {
	if s.pool == nil {
		return nil
	}

	seen := make(map[string]bool)
	var healthy []*tunnel.Tunnel

	// 1. Resolve tunnels from bound dynamic groups (which already enforce PurityLevel, ConnectMode, Country, etc.)
	if s.dynamicMgr != nil && len(boundGroupIDs) > 0 {
		groupTunnels := s.dynamicMgr.GetTunnelsForGroups(boundGroupIDs)
		for _, t := range groupTunnels {
			if t.IsHealthy() && !seen[t.ID] {
				healthy = append(healthy, t)
				seen[t.ID] = true
			}
		}
	}

	// 2. Resolve tunnels from explicitly bound static IDs
	if len(boundIDs) > 0 {
		staticTunnels := s.pool.GetHealthyTunnels(boundIDs)
		for _, t := range staticTunnels {
			if t.IsHealthy() && !seen[t.ID] {
				healthy = append(healthy, t)
				seen[t.ID] = true
			}
		}
	}

	// Explicit bindings are constraints, not preferences. Never escape the set.
	if len(boundIDs) == 0 && len(boundGroupIDs) == 0 {
		healthy = s.pool.GetHealthyTunnels(nil)
	}
	var available []*tunnel.Tunnel
	for _, t := range healthy {
		if t.IsAvailable() {
			available = append(available, t)
		}
	}
	healthy = available

	ids := append([]string(nil), boundIDs...)
	groups := append([]string(nil), boundGroupIDs...)
	sort.Strings(ids)
	sort.Strings(groups)
	return s.selectForPort(port, fmt.Sprintf("%q/%q/%s/%d", ids, groups, policy, intervalSec), healthy, policy, intervalSec)
}

func (s *DefaultScheduler) selectFromHealthy(healthy []*tunnel.Tunnel, policy PortPolicy, intervalSec int) *tunnel.Tunnel {
	return s.selectForPort(0, string(policy), healthy, policy, intervalSec)
}

func (s *DefaultScheduler) selectForPort(port int, signature string, healthy []*tunnel.Tunnel, policy PortPolicy, intervalSec int) *tunnel.Tunnel {
	healthy = append([]*tunnel.Tunnel(nil), healthy...)
	sort.Slice(healthy, func(i, j int) bool { return healthy[i].ID < healthy[j].ID })
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.states == nil {
		s.states = make(map[int]*scheduleState)
	}
	state := s.states[port]
	if state == nil || state.signature != signature {
		state = &scheduleState{signature: signature}
		s.states[port] = state
	}
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	n := len(healthy)
	if n == 0 {
		state.selected = ""
		state.expires = time.Time{}
		return nil
	}

	switch policy {
	case PolicyLeastRTT:
		best := healthy[0]
		minRTT := getTunnelRTT(best)
		var candidates []*tunnel.Tunnel
		candidates = append(candidates, best)
		for i := 1; i < n; i++ {
			rtt := getTunnelRTT(healthy[i])
			if rtt < minRTT {
				best = healthy[i]
				minRTT = rtt
				candidates = candidates[:0]
				candidates = append(candidates, best)
			} else if rtt == minRTT {
				candidates = append(candidates, healthy[i])
			}
		}
		if len(candidates) == 1 {
			return candidates[0]
		}
		// #nosec G115 -- modulo bounds the result to slice length
		idx := int(state.next() % uint64(len(candidates)))
		return candidates[idx]

	case PolicyRandom:
		nBig, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
		if err == nil {
			return healthy[int(nBig.Int64())]
		}
		return healthy[0]

	case PolicyInterval:
		if intervalSec <= 0 {
			intervalSec = 300 // default 5 minutes
		}
		if now.Before(state.expires) {
			for _, t := range healthy {
				if t.ID == state.selected {
					return t
				}
			}
		}
		// Advance from the previous identity when the period expires, not a map index.
		idx := 0
		for i, t := range healthy {
			if t.ID == state.selected {
				idx = (i + 1) % n
				break
			}
		}
		state.selected = healthy[idx].ID
		state.expires = now.Add(time.Duration(intervalSec) * time.Second)
		return healthy[idx]

	case PolicyRoundRobin:
		fallthrough
	default:
		// #nosec G115 -- modulo bounds the result to the slice length.
		idx := int(state.next() % uint64(n))
		return healthy[idx]
	}
}

func isOnlySystemPrimary(boundIDs, boundGroupIDs []string) bool {
	if len(boundIDs) > 0 {
		return false
	}
	if len(boundGroupIDs) == 1 && (boundGroupIDs[0] == "system-primary" || boundGroupIDs[0] == tunnel.SystemPrimaryGroupID) {
		return true
	}
	return false
}

func getTunnelRTT(t *tunnel.Tunnel) int {
	if t == nil {
		return 9999
	}
	if t.LatencyMs > 0 {
		return t.LatencyMs
	}
	if t.Node != nil && t.Node.Ping > 0 {
		return t.Node.Ping
	}
	return 9999
}

func (s *scheduleState) next() uint64 { n := s.counter; s.counter++; return n }
