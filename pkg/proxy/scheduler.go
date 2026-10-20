package proxy

import (
	"crypto/rand"
	"math/big"
	"sync/atomic"
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
	counter    atomic.Uint64
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

	return s.selectFromHealthy(healthy, policy, intervalSec)
}

func (s *DefaultScheduler) selectFromHealthy(healthy []*tunnel.Tunnel, policy PortPolicy, intervalSec int) *tunnel.Tunnel {
	n := len(healthy)
	if n == 0 {
		return nil
	}
	if n == 1 {
		return healthy[0]
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
		idx := int(s.counter.Add(1) % uint64(len(candidates)))
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
		slot := (time.Now().Unix() / int64(intervalSec)) % int64(n)
		return healthy[int(slot)]

	case PolicyRoundRobin:
		fallthrough
	default:
		// #nosec G115 -- modulo bounds the result to the slice length.
		idx := int(s.counter.Add(1) % uint64(n))
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
