package proxy

import (
	"crypto/rand"
	"math/big"
	"sync/atomic"
	"time"

	"aimili-vpngate-go/pkg/nodes"
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
	SelectTunnelForRule(rule PortRule) *tunnel.Tunnel
}

type DefaultScheduler struct {
	pool       *tunnel.Pool
	dynamicMgr *tunnel.DynamicGroupManager
	counter    atomic.Uint64
}

func NewScheduler(pool *tunnel.Pool, dm *tunnel.DynamicGroupManager) *DefaultScheduler {
	return &DefaultScheduler{
		pool:       pool,
		dynamicMgr: dm,
	}
}

func (s *DefaultScheduler) SelectTunnel(port int, boundIDs []string, boundGroupIDs []string, policy PortPolicy, intervalSec int) *tunnel.Tunnel {
	return s.SelectTunnelForRule(PortRule{
		Port:            port,
		BoundTunnelIDs:  boundIDs,
		BoundGroupIDs:   boundGroupIDs,
		Policy:          policy,
		IntervalSeconds: intervalSec,
	})
}

func (s *DefaultScheduler) SelectTunnelForRule(rule PortRule) *tunnel.Tunnel {
	if s.pool == nil {
		return nil
	}

	seen := make(map[string]bool)
	var healthy []*tunnel.Tunnel

	// 1. Resolve tunnels from bound dynamic groups
	if s.dynamicMgr != nil && len(rule.BoundGroupIDs) > 0 {
		groupTunnels := s.dynamicMgr.GetTunnelsForGroups(rule.BoundGroupIDs)
		for _, t := range groupTunnels {
			if t.IsHealthy() && !seen[t.ID] {
				healthy = append(healthy, t)
				seen[t.ID] = true
			}
		}
	}

	// 2. Resolve tunnels from explicitly bound static IDs
	if len(rule.BoundTunnelIDs) > 0 {
		staticTunnels := s.pool.GetHealthyTunnels(rule.BoundTunnelIDs)
		for _, t := range staticTunnels {
			if t.IsHealthy() && !seen[t.ID] {
				healthy = append(healthy, t)
				seen[t.ID] = true
			}
		}
	}

	// Fallback: if no specific bound tunnels were chosen, or if the bound static/group IDs have no healthy tunnels currently,
	// automatically fallback to any healthy tunnel in the whole pool so proxy connections never black-hole
	if len(healthy) == 0 {
		healthy = s.pool.GetHealthyTunnels(nil)
		if len(healthy) == 0 {
			return nil
		}
	}

	// Filter by circuit breaker: prefer available (non-circuit-broken) tunnels
	var available []*tunnel.Tunnel
	for _, t := range healthy {
		if t.IsAvailable() {
			available = append(available, t)
		}
	}
	if len(available) > 0 {
		healthy = available
	} else {
		// All tunnels in the requested group/selection are currently circuit-broken or dead!
		// Fallback to any available non-circuit-broken tunnel in the entire pool so requests never black-hole
		poolTunnels := s.pool.GetHealthyTunnels(nil)
		var poolAvailable []*tunnel.Tunnel
		for _, t := range poolTunnels {
			if t.IsAvailable() {
				poolAvailable = append(poolAvailable, t)
			}
		}
		if len(poolAvailable) > 0 {
			healthy = poolAvailable
		} else if len(poolTunnels) > 0 {
			healthy = poolTunnels
		}
	}

	// 3. Filter by ConnectMode (tun vs zero_tun)
	filtered := filterTunnelsByConnectMode(healthy, rule.ConnectMode)

	// 4. Filter by PurityLevel (pure_residential, clean, datacenter)
	purityFiltered := filterTunnelsByPurity(filtered, rule.PurityLevel)

	if len(purityFiltered) > 0 {
		healthy = purityFiltered
	} else {
		// No tunnel matched the purity/mode criteria
		if rule.StrictMatch {
			// In strict mode, do not degrade! Refuse routing to prevent risk pollution
			return nil
		}
		// Graceful degradation: prefer connect_mode filtered if non-empty, otherwise healthy
		if len(filtered) > 0 {
			healthy = filtered
		}
	}

	return s.selectFromHealthy(healthy, rule.Policy, rule.IntervalSeconds)
}

func filterTunnelsByConnectMode(tunnels []*tunnel.Tunnel, mode string) []*tunnel.Tunnel {
	if mode == "" || mode == "all" {
		return tunnels
	}
	var res []*tunnel.Tunnel
	for _, t := range tunnels {
		if mode == "tun" && !t.IsUserland {
			res = append(res, t)
		} else if (mode == "zero_tun" || mode == "sstp") && t.IsUserland {
			res = append(res, t)
		}
	}
	return res
}

func filterTunnelsByPurity(tunnels []*tunnel.Tunnel, purity string) []*tunnel.Tunnel {
	if purity == "" || purity == "any" {
		return tunnels
	}
	var res []*tunnel.Tunnel
	for _, t := range tunnels {
		if t == nil || t.Node == nil {
			continue
		}
		n := t.Node
		switch purity {
		case "pure_residential":
			// Highly pure residential: is_residential && (risk_score <= 15 or pure)
			if n.IsResidential && (n.RiskScore <= 15 || n.RiskLevel == nodes.RiskLevelPure) {
				res = append(res, t)
			}
		case "clean":
			// Clean / low risk: risk_score <= 35 or clean/pure
			if n.RiskScore <= 35 || n.RiskLevel == nodes.RiskLevelPure || n.RiskLevel == nodes.RiskLevelClean {
				res = append(res, t)
			}
		case "datacenter":
			// High speed datacenter: is_hosting or ip_type hosting/datacenter
			if n.IsHosting || n.IPType == "hosting" || n.IPType == "datacenter" {
				res = append(res, t)
			}
		default:
			res = append(res, t)
		}
	}
	return res
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
