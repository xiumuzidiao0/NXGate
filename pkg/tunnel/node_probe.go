package tunnel

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"aimili-vpngate-go/pkg/nodes"
	"aimili-vpngate-go/pkg/sstp"
	"aimili-vpngate-go/pkg/stats"
)

// NodeDeepProbeResult records connectivity, TCP/tunnel RTT, and physical unlock findings
// for a single on-demand node probe.
type NodeDeepProbeResult struct {
	NodeID          string              `json:"node_id"`
	IP              string              `json:"ip"`
	Port            int                 `json:"port"`
	Proto           string              `json:"proto"`
	CountryShort    string              `json:"country_short"`
	Status          string              `json:"status"` // "pending", "running", "ok", "partial", "failed", "cancelled"
	Driver          string              `json:"driver,omitempty"` // "existing", "sstp", "openvpn", "tcp_only"
	TCPReachable    bool                `json:"tcp_reachable"`
	TCPLatencyMs    int                 `json:"tcp_latency_ms"`
	TunnelConnected bool                `json:"tunnel_connected"`
	EgressReachable bool                `json:"egress_reachable"`
	EgressLatencyMs int                 `json:"egress_latency_ms"`
	SSTPPort        int                 `json:"sstp_port,omitempty"`
	Unlock          *nodes.UnlockResult `json:"unlock,omitempty"`
	Error           string              `json:"error,omitempty"`
	CheckedAt       time.Time           `json:"checked_at,omitempty"`
	DurationMs      int64               `json:"duration_ms,omitempty"`
}

// ProbeNodeDeep tests TCP reachability, tunnel connectivity + outbound RTT, and service unlocks
// for node without affecting blacklists or leaking temporary tunnels into active pools.
func (p *Pool) ProbeNodeDeep(ctx context.Context, node *nodes.Node) *NodeDeepProbeResult {
	started := time.Now()
	if node == nil {
		return &NodeDeepProbeResult{
			Status:     "failed",
			Error:      "节点不存在",
			CheckedAt:  time.Now(),
			DurationMs: time.Since(started).Milliseconds(),
		}
	}

	res := &NodeDeepProbeResult{
		NodeID:       node.ID,
		IP:           node.IP,
		Port:         node.Port,
		Proto:        node.Proto,
		CountryShort: node.CountryShort,
		SSTPPort:     node.SSTPPort,
		Status:       "running",
	}

	// 1. Direct TCP latency probe when the relay exposes a TCP port.
	if node.Port > 0 && !strings.EqualFold(node.Proto, "udp") {
		if ms, ok := probeDirectTCP(ctx, node.IP, node.Port, 3*time.Second); ok {
			res.TCPReachable = true
			res.TCPLatencyMs = ms
		} else if ctx.Err() == nil {
			res.TCPLatencyMs = -1
		}
	}

	// 2. Reuse an already-established healthy non-probe tunnel when testing an active exit.
	if existing := p.findHealthyTunnelForNode(node.ID, node.IP); existing != nil {
		res.Driver = "existing"
		res.TunnelConnected = true
		existing.mu.RLock()
		isUserland := existing.IsUserland
		dialer := existing.Dialer
		devName := existing.DevName
		existing.mu.RUnlock()

		var egressOK bool
		var egressRTT int
		if isUserland && dialer != nil {
			connCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
			egressOK, egressRTT, _ = MeasureDialerConnectivityContext(connCtx, dialer, 2500*time.Millisecond)
			cancel()
		} else {
			connCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			egressOK, egressRTT, _ = MeasureTunnelConnectivityContext(connCtx, devName, 3*time.Second)
			cancel()
		}
		res.EgressReachable = egressOK
		res.EgressLatencyMs = egressRTT
		if egressOK {
			if p.unlockDetector != nil {
				unlockCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
				unlockRes := p.unlockDetector.ProbeTunnelInstance(unlockCtx, existing)
				cancel()
				res.Unlock = unlockRes
				existing.SetUnlock(unlockRes)
			}
			res.Status = "ok"
		} else {
			res.Status = "partial"
			res.Error = "隧道已建立但无法转发公网流量"
		}
		res.CheckedAt = time.Now()
		res.DurationMs = time.Since(started).Milliseconds()
		p.persistDeepProbe(res)
		return res
	}

	// 3. Try ephemeral Userland SSTP probe (fast, zero kernel routing side effects).
	sstpErr := p.probeViaEphemeralSSTP(ctx, node, res)
	if res.TunnelConnected {
		res.CheckedAt = time.Now()
		res.DurationMs = time.Since(started).Milliseconds()
		p.persistDeepProbe(res)
		return res
	}

	// 4. Fall back to ephemeral Kernel OpenVPN probe if SSTP failed and OpenVPN TUN is available.
	var ovpnErr error
	if ctx.Err() == nil && p.CanUseKernelTUN() && strings.TrimSpace(node.ConfigData) != "" {
		ovpnErr = p.probeViaEphemeralOpenVPN(ctx, node, res)
		if res.TunnelConnected {
			res.CheckedAt = time.Now()
			res.DurationMs = time.Since(started).Milliseconds()
			p.persistDeepProbe(res)
			return res
		}
	}

	res.Driver = "tcp_only"
	if ctx.Err() != nil {
		res.Status = "cancelled"
		res.Error = "检测已取消或超时"
	} else {
		res.Status = "failed"
		switch {
		case ovpnErr != nil && sstpErr != nil:
			res.Error = fmt.Sprintf("SSTP 失败 (%v); OpenVPN 失败 (%v)", sstpErr, ovpnErr)
		case sstpErr != nil:
			res.Error = fmt.Sprintf("SSTP 握手失败: %v", sstpErr)
		case ovpnErr != nil:
			res.Error = fmt.Sprintf("OpenVPN 握手失败: %v", ovpnErr)
		default:
			res.Error = "无法建立探测隧道"
		}
	}
	res.CheckedAt = time.Now()
	res.DurationMs = time.Since(started).Milliseconds()
	p.persistDeepProbe(res)
	return res
}

func probeDirectTCP(ctx context.Context, ip string, port int, timeout time.Duration) (int, bool) {
	if ip == "" || port <= 0 || ctx.Err() != nil {
		return 0, false
	}
	addr := net.JoinHostPort(ip, strconv.Itoa(port))
	dialer := net.Dialer{Timeout: timeout}
	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, false
	}
	_ = conn.Close()
	return max(1, int(time.Since(start).Milliseconds())), true
}

func (p *Pool) findHealthyTunnelForNode(nodeID, ip string) *Tunnel {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, t := range p.tunnels {
		if t == nil || t.IsProbe || t.Node == nil {
			continue
		}
		if (t.Node.ID == nodeID || (ip != "" && t.Node.IP == ip)) && t.IsHealthy() {
			return t
		}
	}
	return nil
}

func (p *Pool) probeViaEphemeralSSTP(ctx context.Context, node *nodes.Node, res *NodeDeepProbeResult) error {
	sstpCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var (
		dialer func(context.Context, string, string) (net.Conn, error)
		closer interface{ Close() error }
		port   int
		err    error
	)

	if p.sstpProbeDialer != nil {
		dialer, closer, port, err = p.sstpProbeDialer(sstpCtx, node)
	} else {
		st, connErr := sstp.ConnectNodeWithPreferred(sstpCtx, node.IP, node.SSTPPort, node.Port, node.Proto, "vpn", "vpn")
		err = connErr
		if st != nil {
			dialer = st.Dial
			closer = st
			port = st.Port()
		}
	}
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}
	if err != nil || dialer == nil {
		if err == nil {
			err = fmt.Errorf("missing dialer")
		}
		return err
	}

	res.Driver = "sstp"
	res.TunnelConnected = true
	if port > 0 {
		res.SSTPPort = port
	}
	if !res.TCPReachable && port > 0 {
		if ms, ok := probeDirectTCP(ctx, node.IP, port, 2*time.Second); ok {
			res.TCPReachable = true
			res.TCPLatencyMs = ms
		}
	}

	// Probe budget (8s) exceeds targetCount * perTargetTimeout (3 * 2.5s = 7.5s)
	// so black-holed relays return explicit false instead of context.DeadlineExceeded.
	connCtx, connCancel := context.WithTimeout(ctx, 8*time.Second)
	egressOK, egressRTT, _ := MeasureDialerConnectivityContext(connCtx, dialer, 2500*time.Millisecond)
	connCancel()

	res.EgressReachable = egressOK
	res.EgressLatencyMs = egressRTT
	if !egressOK {
		res.Status = "partial"
		res.Error = "SSTP 握手成功但无法转发公网流量"
		return nil
	}

	if p.unlockDetector != nil {
		unlockCtx, unlockCancel := context.WithTimeout(ctx, 12*time.Second)
		res.Unlock = p.unlockDetector.ProbeWithDialer(unlockCtx, dialer, node.IP, fmt.Sprintf("probe-sstp-%d", port))
		unlockCancel()
	}
	res.Status = "ok"
	return nil
}

func (p *Pool) probeViaEphemeralOpenVPN(ctx context.Context, node *nodes.Node, res *NodeDeepProbeResult) error {
	probeTun, err := p.startProbeTunnel(node)
	if err != nil || probeTun == nil {
		return err
	}
	defer func() { _ = p.StopTunnel(probeTun.ID) }()

	connected := false
	for w := 0; w < 28; w++ {
		if !waitContext(ctx, 500*time.Millisecond) {
			return ctx.Err()
		}
		st := probeTun.GetStatus()
		if st == StatusConnected {
			connected = true
			break
		}
		if st == StatusFailed || st == StatusStopped {
			break
		}
	}
	if !connected {
		probeTun.mu.RLock()
		msg := probeTun.Message
		probeTun.mu.RUnlock()
		if msg == "" {
			msg = "OpenVPN 握手超时"
		}
		return fmt.Errorf("%s", msg)
	}

	res.Driver = "openvpn"
	res.TunnelConnected = true
	if !waitContext(ctx, 300*time.Millisecond) {
		return ctx.Err()
	}

	connCtx, connCancel := context.WithTimeout(ctx, 10*time.Second)
	egressOK, egressRTT, _ := MeasureTunnelConnectivityContext(connCtx, probeTun.DevName, 3*time.Second)
	connCancel()

	res.EgressReachable = egressOK
	res.EgressLatencyMs = egressRTT
	if !egressOK {
		res.Status = "partial"
		res.Error = "OpenVPN 握手成功但无法转发公网流量"
		return nil
	}

	if p.unlockDetector != nil {
		unlockCtx, unlockCancel := context.WithTimeout(ctx, 12*time.Second)
		res.Unlock = p.unlockDetector.ProbeTunnel(unlockCtx, probeTun.DevName, node.IP)
		unlockCancel()
	}
	res.Status = "ok"
	return nil
}

func (p *Pool) persistDeepProbe(res *NodeDeepProbeResult) {
	if p.nodePool == nil || res == nil || res.NodeID == "" {
		return
	}
	tcpLat := res.TCPLatencyMs
	if tcpLat == 0 && !res.TCPReachable && !res.TunnelConnected {
		tcpLat = -1
	}
	p.nodePool.ApplyDeepProbeResult(res.NodeID, tcpLat, res.TCPReachable, res.TunnelConnected, res.SSTPPort, res.Unlock)
	stats.LogInfo("NodeProbe", "单点实测 [%s]: status=%s driver=%s tcp=%dms egress=%dms",
		res.NodeID, res.Status, res.Driver, res.TCPLatencyMs, res.EgressLatencyMs)
}
