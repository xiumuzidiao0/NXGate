package vpn

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/nodes"
	"aimili-vpngate-go/pkg/sstp"
	"aimili-vpngate-go/pkg/stats"
	"aimili-vpngate-go/pkg/tunnel"
)

type UserlandProxyAttacher interface {
	SetUserlandDialer(dialer func(ctx context.Context, network, addr string) (net.Conn, error))
	SetUserlandPacketConn(packetConn func() (net.PacketConn, error))
}

type Manager struct {
	cfg        *config.Config
	pool       *nodes.NodePool
	tunnelPool *tunnel.Pool
	attacher   UserlandProxyAttacher
	sstpTunnel *sstp.Tunnel
	driverMode DriverMode

	// failoverDelegate, when set and returning true, takes over primary egress
	// recovery so the unconstrained global "best node" pick never overrides a
	// governing dynamic group's country / IP-type / purity / unlock constraints.
	failoverDelegate func() bool

	// lastEgressProbe throttles the userland forwarding check so it runs at most
	// once per userlandEgressProbeInterval instead of on every health tick.
	lastEgressProbe time.Time

	mu           sync.RWMutex
	epoch        uint64
	primaryID    string
	status       ConnectionStatus
	activeNode   *nodes.Node
	connectedAt  time.Time
	lastMessage  string
	reconnects   int
	isConnecting bool
}

// userlandEgressProbes are dialled through a userland tunnel using literal IPs,
// so the check exercises IP packet forwarding only and never depends on DNS or
// on the tunnel's own DoH resolver.
var userlandEgressProbes = []string{"1.1.1.1:443", "8.8.8.8:443", "9.9.9.9:443"}

const (
	userlandEgressProbeInterval = 60 * time.Second
	userlandEgressProbeTimeout  = 5 * time.Second
)

// probeUserlandEgress verifies that a userland tunnel actually forwards IP
// packets. A SoftEther relay can complete the SSTP/PPP handshake, stay perfectly
// responsive to keepalives and still never forward traffic (restricted relay or
// upstream filtering); such a tunnel must never be published as a working exit.
func probeUserlandEgress(tun *sstp.Tunnel) bool {
	if tun == nil {
		return false
	}
	for _, target := range userlandEgressProbes {
		ctx, cancel := context.WithTimeout(context.Background(), userlandEgressProbeTimeout)
		conn, err := tun.Dial(ctx, "tcp", target)
		cancel()
		if err == nil {
			_ = conn.Close()
			return true
		}
	}
	return false
}

// AutoDetectDriverMode inspects system environment and capabilities to select appropriate driver mode.
func AutoDetectDriverMode(cfg *config.Config) DriverMode {
	// 1. Explicit override via DRIVER_MODE env var
	envMode := strings.ToLower(strings.TrimSpace(os.Getenv("DRIVER_MODE")))
	if envMode == string(DriverUserland) || envMode == "zero_tun" || envMode == "sstp" {
		return DriverUserland
	}
	if envMode == string(DriverKernel) || envMode == "tun" || envMode == "openvpn" {
		return DriverKernel
	}

	// 2. Non-Linux OS lacks Linux table 100+N and OpenVPN isolation routes
	if runtime.GOOS != "linux" {
		return DriverUserland
	}

	// 3. Inspect /dev/net/tun availability
	if err := CheckTUNDevice(); err != nil {
		stats.LogWarn("VPN", "检测到系统缺失 /dev/net/tun 设备 (%v)，网络驱动自动降级为用户态免 TUN 模式", err)
		return DriverUserland
	}

	// 4. Inspect openvpn command binary
	openvpnCmd := "openvpn"
	if cfg != nil && cfg.OpenVPNCommand != "" {
		openvpnCmd = cfg.OpenVPNCommand
	}
	if _, err := exec.LookPath(openvpnCmd); err != nil {
		stats.LogWarn("VPN", "未检测到 OpenVPN 可执行文件 (%s: %v)，网络驱动自动降级为用户态免 TUN 模式", openvpnCmd, err)
		return DriverUserland
	}

	return DriverKernel
}

func NewManager(cfg *config.Config, pool *nodes.NodePool, tp *tunnel.Pool) *Manager {
	initialMode := AutoDetectDriverMode(cfg)
	return &Manager{
		cfg:         cfg,
		pool:        pool,
		tunnelPool:  tp,
		driverMode:  initialMode,
		status:      StatusDisconnected,
		lastMessage: "VPN 服务已就绪，未连接",
	}
}

func (m *Manager) SetProxyAttacher(attacher UserlandProxyAttacher) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attacher = attacher
}

// SetPrimaryFailoverDelegate installs a handler consulted before the built-in
// unconstrained "best node" failover pick. When it reports true, primary egress
// recovery is left entirely to that handler: the system-primary dynamic group
// owns the country / IP-type / purity / unlock constraints, and the global pick
// would silently violate them. The handler must not block for long and must be
// safe to call without holding the manager lock.
func (m *Manager) SetPrimaryFailoverDelegate(fn func() bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failoverDelegate = fn
}

func (m *Manager) Snapshot() StateSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := m.status
	lastMsg := m.lastMessage
	activeNode := m.activeNode
	connectedAt := m.connectedAt

	if m.primaryID != "" && m.tunnelPool != nil {
		if t := m.tunnelPool.GetTunnel(m.primaryID); t != nil {
			if t.Status == tunnel.StatusConnected {
				status = StatusConnected
			} else if t.Status == tunnel.StatusConnecting {
				status = StatusConnecting
			} else if t.Status == tunnel.StatusFailed {
				status = StatusFailed
			}
			activeNode = t.Node
			connectedAt = t.ConnectedAt
			if t.Message != "" {
				lastMsg = t.Message
			}
		}
	}

	var uptime int64
	if status == StatusConnected && !connectedAt.IsZero() {
		uptime = int64(time.Since(connectedAt).Seconds())
	}

	var activeID string
	if activeNode != nil {
		activeID = activeNode.ID
	}

	driverMode := m.driverMode
	if driverMode == "" {
		driverMode = DriverKernel
	}

	if driverMode == DriverUserland {
		if m.sstpTunnel != nil && m.sstpTunnel.IsHealthy() {
			status = StatusConnected
		} else if status == StatusConnected {
			// The userland session died (peer gone or silently black-holed). Never
			// keep advertising a working exit while the proxy cannot forward.
			status = StatusReconnecting
		}
	} else if m.primaryID != "" && m.tunnelPool != nil {
		if t := m.tunnelPool.GetTunnel(m.primaryID); t != nil {
			if t.Status == tunnel.StatusConnected {
				status = StatusConnected
			} else if t.Status == tunnel.StatusConnecting {
				status = StatusConnecting
			} else if t.Status == tunnel.StatusFailed {
				status = StatusFailed
			}
			activeNode = t.Node
			connectedAt = t.ConnectedAt
			if t.Message != "" {
				lastMsg = t.Message
			}
		}
	}

	if status == StatusConnected && !connectedAt.IsZero() {
		uptime = int64(time.Since(connectedAt).Seconds())
	}

	return StateSnapshot{
		Status:         status,
		StatusText:     string(status),
		ActiveNodeID:   activeID,
		ActiveNode:     activeNode,
		DriverMode:     driverMode,
		ZeroTUN:        driverMode == DriverUserland,
		TunnelReady:    status == StatusConnected,
		ProxyReady:     true,
		ConnectedAt:    connectedAt,
		UptimeSeconds:  uptime,
		LastMessage:    lastMsg,
		ReconnectCount: m.reconnects,
	}
}

func (m *Manager) Connect(target *nodes.Node) error {
	m.mu.RLock()
	mode := m.driverMode
	m.mu.RUnlock()
	return m.ConnectWithMode(target, string(mode))
}

func (m *Manager) ConnectWithMode(target *nodes.Node, mode string) error {
	if mode == "" || mode == "auto" {
		m.mu.RLock()
		mode = string(m.driverMode)
		m.mu.RUnlock()
	}

	if mode == string(DriverUserland) || mode == "zero_tun" || mode == "sstp" {
		return m.ConnectUserland(target)
	}

	// Validate kernel dependencies before attempting kernel connection; auto-downgrade to userland if missing
	if err := CheckTUNDevice(); err != nil {
		stats.LogWarn("VPN", "系统不支持内核 TUN (%v)，主网关自适应降级为用户态免 TUN 模式", err)
		return m.ConnectUserland(target)
	}
	openvpnCmd := "openvpn"
	if m.cfg != nil && m.cfg.OpenVPNCommand != "" {
		openvpnCmd = m.cfg.OpenVPNCommand
	}
	if _, err := exec.LookPath(openvpnCmd); err != nil {
		stats.LogWarn("VPN", "系统未安装 OpenVPN (%s: %v)，主网关自适应降级为用户态免 TUN 模式", openvpnCmd, err)
		return m.ConnectUserland(target)
	}

	m.mu.Lock()
	m.epoch++
	currentEpoch := m.epoch

	// Clean up previous userland sstp tunnel if active
	if m.sstpTunnel != nil {
		_ = m.sstpTunnel.Close()
		m.sstpTunnel = nil
	}
	if m.attacher != nil {
		m.attacher.SetUserlandDialer(nil)
		m.attacher.SetUserlandPacketConn(nil)
	}

	// Stop previous primary tunnel if running
	if m.primaryID != "" && m.tunnelPool != nil {
		_ = m.tunnelPool.StopTunnel(m.primaryID)
		m.primaryID = ""
	}

	m.driverMode = DriverKernel
	m.activeNode = target
	m.status = StatusConnecting
	m.isConnecting = true
	m.lastMessage = fmt.Sprintf("正在发起对节点 %s (%s) 的内核 OpenVPN 连接...", target.ID, target.CountryShort)
	m.mu.Unlock()

	stats.LogInfo("VPN", "开始连接主节点: %s (%s %s)", target.ID, target.CountryShort, target.HostName)

	if m.tunnelPool == nil {
		m.mu.Lock()
		m.status = StatusFailed
		m.isConnecting = false
		m.lastMessage = "隧道池未初始化"
		m.mu.Unlock()
		return fmt.Errorf("tunnel pool is nil")
	}

	tun, err := m.tunnelPool.StartPrimaryTunnel(target)
	if err != nil {
		m.mu.Lock()
		if m.epoch == currentEpoch {
			m.status = StatusFailed
			m.isConnecting = false
			m.lastMessage = fmt.Sprintf("启动主隧道失败: %v", err)
		}
		m.mu.Unlock()
		return err
	}

	m.mu.Lock()
	if m.epoch == currentEpoch {
		m.primaryID = tun.ID
	}
	m.mu.Unlock()

	// Monitor until ready or failed
	go func() {
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()

		timeout := time.After(20 * time.Second)

		for {
			select {
			case <-timeout:
				m.mu.Lock()
				if m.epoch == currentEpoch && m.status == StatusConnecting {
					m.status = StatusFailed
					m.isConnecting = false
					m.lastMessage = "连接超时"
				}
				m.mu.Unlock()
				return
			case <-ticker.C:
				m.mu.RLock()
				curEpoch := m.epoch
				pID := m.primaryID
				m.mu.RUnlock()

				if curEpoch != currentEpoch {
					return
				}

				t := m.tunnelPool.GetTunnel(pID)
				if t == nil {
					m.mu.Lock()
					if m.epoch == currentEpoch {
						m.status = StatusFailed
						m.isConnecting = false
						m.lastMessage = "连接异常中断 (进程已退出)"
					}
					m.mu.Unlock()
					go m.TriggerAutoFailover()
					return
				}

				status := t.GetStatus()
				if status == tunnel.StatusConnected {
					m.mu.Lock()
					if m.epoch == currentEpoch {
						m.status = StatusConnected
						m.isConnecting = false
						m.connectedAt = time.Now()
						m.lastMessage = "VPN 隧道已成功连通并就绪"
					}
					m.mu.Unlock()
					return
				} else if status == tunnel.StatusFailed {
					m.mu.Lock()
					if m.epoch == currentEpoch {
						m.status = StatusFailed
						m.isConnecting = false
						m.lastMessage = t.Message
					}
					m.mu.Unlock()
					go m.TriggerAutoFailover()
					return
				}
			}
		}
	}()

	return nil
}

func (m *Manager) ConnectUserland(target *nodes.Node) error {
	m.mu.Lock()
	m.epoch++
	currentEpoch := m.epoch

	// Stop previous primary openvpn tunnel if running
	if m.primaryID != "" && m.tunnelPool != nil {
		_ = m.tunnelPool.StopTunnel(m.primaryID)
		m.primaryID = ""
	}
	// Stop previous sstp tunnel if running
	if m.sstpTunnel != nil {
		_ = m.sstpTunnel.Close()
		m.sstpTunnel = nil
	}
	if m.attacher != nil {
		m.attacher.SetUserlandDialer(nil)
		m.attacher.SetUserlandPacketConn(nil)
	}

	m.activeNode = target
	m.driverMode = DriverUserland
	m.status = StatusConnecting
	m.isConnecting = true
	m.lastMessage = fmt.Sprintf("正在通过用户态 SSTP (免 TUN) 连接 %s (%s)...", target.ID, target.CountryShort)
	m.mu.Unlock()

	stats.LogInfo("VPN", "开始用户态免 TUN 直连: %s (%s %s:%d, 协议: %s)", target.ID, target.CountryShort, target.IP, target.Port, target.Proto)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		tun, err := sstp.ConnectNodeWithPreferred(ctx, target.IP, target.SSTPPort, target.Port, target.Proto, "vpn", "vpn")
		if err != nil {
			m.mu.Lock()
			if m.epoch == currentEpoch {
				m.status = StatusFailed
				m.isConnecting = false
				m.lastMessage = fmt.Sprintf("用户态 SSTP 直连失败: %v", err)
			}
			m.mu.Unlock()
			if m.pool != nil {
				m.pool.Blacklist().Mark(target, fmt.Sprintf("用户态免TUN直连失败: %v", err), 600*time.Second)
			}
			stats.LogWarn("VPN", "用户态直连 %s 失败: %v", target.IP, err)
			go m.TriggerAutoFailover()
			return
		}

		// The SSTP/PPP handshake alone does not prove the relay forwards traffic.
		if !probeUserlandEgress(tun) {
			_ = tun.Close()
			m.mu.Lock()
			if m.epoch == currentEpoch {
				m.status = StatusFailed
				m.isConnecting = false
				m.lastMessage = fmt.Sprintf("用户态隧道 %s 握手成功但无法转发 IP 报文", target.IP)
			}
			m.mu.Unlock()
			if m.pool != nil {
				// A verified forwarding failure is a functional defect, not a
				// transient blip, so block the node outright instead of relying on
				// the escalation counter (which only starts blocking at 3 strikes).
				m.pool.Blacklist().MarkManual(target.ID, target.IP, target.CountryShort,
					"用户态隧道握手成功但无法出网", 10*time.Minute)
			}
			stats.LogWarn("VPN", "用户态隧道 %s (%s) 握手成功但无法出网 (被对端阻断)，已拉黑并重新选路", target.IP, target.CountryShort)
			go m.TriggerAutoFailover()
			return
		}

		if target != nil {
			target.SSTPPort = tun.Port()
		}

		m.mu.Lock()
		if m.epoch != currentEpoch {
			_ = tun.Close()
			m.mu.Unlock()
			return
		}
		m.sstpTunnel = tun
		m.status = StatusConnected
		m.isConnecting = false
		m.connectedAt = time.Now()
		m.lastMessage = fmt.Sprintf("用户态免 TUN 直连已就绪 (虚拟 IP: %s, 端口: %d)", tun.AssignedIP(), tun.Port())
		m.mu.Unlock()

		if m.attacher != nil {
			m.attacher.SetUserlandDialer(tun.Dial)
			m.attacher.SetUserlandPacketConn(tun.OpenPacketConn)
		}

		stats.LogInfo("VPN", "用户态免 TUN 直连就绪: %s (%s), 虚拟 IP: %s, 端口: %d", target.ID, target.CountryShort, tun.AssignedIP(), tun.Port())
	}()

	return nil
}

func (m *Manager) Disconnect(reason string) {
	m.mu.Lock()
	m.epoch++
	if m.sstpTunnel != nil {
		_ = m.sstpTunnel.Close()
		m.sstpTunnel = nil
	}
	if m.attacher != nil {
		m.attacher.SetUserlandDialer(nil)
		m.attacher.SetUserlandPacketConn(nil)
	}
	if m.primaryID != "" && m.tunnelPool != nil {
		_ = m.tunnelPool.StopTunnel(m.primaryID)
		m.primaryID = ""
	}
	m.status = StatusDisconnected
	m.activeNode = nil
	m.isConnecting = false
	m.lastMessage = fmt.Sprintf("已断开连接: %s", reason)
	m.mu.Unlock()

	stats.LogInfo("VPN", "VPN 已主动断开: %s", reason)
}

func (m *Manager) TriggerAutoFailover() {
	m.mu.RLock()
	delegate := m.failoverDelegate
	m.mu.RUnlock()
	if delegate != nil && delegate() {
		m.mu.Lock()
		m.status = StatusReconnecting
		m.lastMessage = "检测到隧道中断，已转交主出口动态组按既定国家/纯净度约束重新选路..."
		m.mu.Unlock()
		return
	}

	m.mu.Lock()
	if m.isConnecting {
		m.mu.Unlock()
		return
	}
	m.reconnects++
	m.status = StatusReconnecting
	m.lastMessage = "检测到隧道中断，正在挑选最优新节点自动切换..."
	m.mu.Unlock()

	time.Sleep(2 * time.Second)

	best := m.pool.SelectBest()
	if best == nil {
		stats.LogWarn("VPN", "未找到可用节点，尝试刷新节点池...")
		_ = m.pool.Refresh(context.Background())
		best = m.pool.SelectBest()
	}

	m.mu.RLock()
	currentMode := m.driverMode
	m.mu.RUnlock()

	if best != nil {
		stats.LogInfo("VPN", "自动切换至新主节点: %s (%s, 模式: %s)", best.ID, best.CountryShort, currentMode)
		_ = m.ConnectWithMode(best, string(currentMode))
	} else {
		stats.LogError("VPN", "自动故障转移失败: 无任何候选节点可用")
	}
}

func (m *Manager) StartHealthChecker(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(m.cfg.CheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.mu.RLock()
				status := m.status
				dMode := m.driverMode
				sTun := m.sstpTunnel
				m.mu.RUnlock()

				if status != StatusConnected {
					continue
				}

				if dMode == DriverUserland {
					if sTun == nil || !sTun.IsHealthy() {
						stats.LogWarn("Health", "心跳检测未通过：用户态 SSTP 链路已中断，触发自动切换")
						go m.TriggerAutoFailover()
						continue
					}
					// The SSTP session can stay alive while the relay silently stops
					// forwarding, so re-verify real egress periodically. A dead exit must
					// never be reported as connected.
					m.mu.Lock()
					due := time.Since(m.lastEgressProbe) >= userlandEgressProbeInterval
					if due {
						m.lastEgressProbe = time.Now()
					}
					m.mu.Unlock()
					if due && !probeUserlandEgress(sTun) {
						// A relay that stopped forwarding must not be picked again
						// immediately, otherwise failover just bounces straight back to it.
						m.mu.RLock()
						failedNode := m.activeNode
						m.mu.RUnlock()
						if failedNode != nil && m.pool != nil {
							m.pool.Blacklist().MarkManual(failedNode.ID, failedNode.IP, failedNode.CountryShort,
								"用户态隧道已无法转发 IP 报文", 5*time.Minute)
						}
						stats.LogWarn("Health", "心跳检测未通过：用户态隧道已无法转发 IP 报文 (%s)，已暂时降权并触发自动切换", nodeLabel(failedNode))
						go m.TriggerAutoFailover()
					}
					continue
				}

				if !CheckExternalConnectivity(6 * time.Second) {
					stats.LogWarn("Health", "心跳检测未通过：主连接外部网络连通性中断，触发重试...")
					time.Sleep(3 * time.Second)
					if !CheckExternalConnectivity(6 * time.Second) {
						stats.LogError("Health", "主连接外部网络持续不通，判定当前 VPN 节点失效，触发故障自动转移")
						go m.TriggerAutoFailover()
					}
				}
			}
		}
	}()
}

func (m *Manager) StartAutoRotator(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		var lastRotated time.Time

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				settings := m.cfg.GetSettings()
				if settings.AutoRotateMinutes <= 0 {
					continue
				}

				rotateInterval := time.Duration(settings.AutoRotateMinutes) * time.Minute
				if !lastRotated.IsZero() && time.Since(lastRotated) < rotateInterval {
					continue
				}

				m.mu.RLock()
				status := m.status
				currID := ""
				if m.activeNode != nil {
					currID = m.activeNode.ID
				}
				currentMode := m.driverMode
				m.mu.RUnlock()

				if status != StatusConnected {
					continue
				}

				best := m.pool.SelectBestWithFilter(settings.AutoRotateIPType, settings.DiscoveryCountries, true)
				if best != nil && best.ID != currID {
					stats.LogInfo("Rotator", "触发定时自动轮换策略 (周期: %d分钟)，正在平滑切换至新节点: %s (%s, 模式: %s)...",
						settings.AutoRotateMinutes, best.ID, best.CountryShort, currentMode)
					_ = m.ConnectWithMode(best, string(currentMode))
					lastRotated = time.Now()
				}
			}
		}
	}()
}

func (m *Manager) GetActiveNode() *nodes.Node {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeNode
}

func (m *Manager) GetStatus() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return string(m.status)
}

func (m *Manager) DriverMode() DriverMode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.driverMode
}

func (m *Manager) GetDefaultDriverMode() DriverMode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.driverMode
}

func (m *Manager) IsZeroTun() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.driverMode == DriverUserland
}

func (m *Manager) ConnectedAt() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connectedAt
}

func (m *Manager) GetSSTPTunnel() *sstp.Tunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sstpTunnel
}

// nodeLabel renders a node for log output, tolerating a nil node.
func nodeLabel(n *nodes.Node) string {
	if n == nil {
		return "<unknown>"
	}
	return fmt.Sprintf("%s (%s)", n.IP, n.CountryShort)
}
