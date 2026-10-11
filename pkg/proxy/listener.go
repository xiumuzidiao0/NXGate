package proxy

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/stats"
	"aimili-vpngate-go/pkg/tunnel"
)

type PortRule struct {
	Port                int        `json:"port"`
	Enabled             bool       `json:"enabled"`
	BoundTunnelIDs      []string   `json:"bound_tunnel_ids"`
	BoundGroupIDs       []string   `json:"bound_group_ids"` // Dynamic group IDs
	BoundCustomProxyIDs []string   `json:"bound_custom_proxy_ids,omitempty"` // Custom external proxy IDs
	Policy              PortPolicy `json:"policy"`
	IntervalSeconds     int        `json:"interval_seconds"`
	AuthMode            string     `json:"auth_mode"` // "random", "custom", "none" (legacy "default_web")
	AuthUser            string     `json:"auth_user"`
	AuthPass            string     `json:"auth_pass"`
	ConnectMode         string     `json:"connect_mode,omitempty"` // "all" (默认), "tun" (仅内核TUN), "zero_tun" (仅免TUN直连)
	PurityLevel         string     `json:"purity_level,omitempty"` // "any" (默认), "pure_residential" (仅纯净住宅), "clean" (纯净/低风险), "datacenter" (仅高速机房)
	StrictMatch         bool       `json:"strict_match,omitempty"` // 严格模式: 无满足条件的出口时拒绝转发(防风控)
}

type PortListener struct {
	rule           PortRule
	cfg            *config.Config
	host           string
	scheduler      TunnelSelector
	customProxyMgr *CustomProxyManager
	userlandDialer func(ctx context.Context, network, addr string) (net.Conn, error)
	userlandPacketConn func() (net.PacketConn, error)
	listener       net.Listener
	sem            chan struct{}
	cancel         context.CancelFunc
	mu             sync.Mutex
	closed         bool
}

func NewPortListener(rule PortRule, cfg *config.Config, scheduler TunnelSelector) *PortListener {
	maxConn := cfg.ProxyMaxConnections
	if maxConn <= 0 {
		maxConn = 512
	}
	host := strings.TrimSpace(cfg.ProxyHost)
	if host == "" {
		host = "127.0.0.1"
	}
	host = strings.Trim(host, "[]")

	return &PortListener{
		rule:      rule,
		cfg:       cfg,
		host:      host,
		scheduler: scheduler,
		sem:       make(chan struct{}, maxConn),
	}
}

func (l *PortListener) SetCustomProxyManager(m *CustomProxyManager) {
	l.customProxyMgr = m
}

func (l *PortListener) SetUserlandDialer(dialer func(ctx context.Context, network, addr string) (net.Conn, error)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.userlandDialer = dialer
}

func (l *PortListener) SetUserlandPacketConn(packetConn func() (net.PacketConn, error)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.userlandPacketConn = packetConn
}

func (l *PortListener) getAuthenticator() *Authenticator {
	mode := l.rule.AuthMode
	if mode == "none" {
		return NewAuthenticator("", "") // Disabled
	}
	if mode == "custom" {
		return NewAuthenticator(l.rule.AuthUser, l.rule.AuthPass)
	}
	// "random" (and legacy "default_web"): use independent persistent random proxy credentials
	user, pass := l.cfg.GetProxyCredentials()
	return NewAuthenticator(user, pass)
}

func (l *PortListener) Start(ctx context.Context) error {
	listenCtx, cancel := context.WithCancel(ctx)
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		cancel()
		return nil
	}
	l.cancel = cancel
	l.mu.Unlock()

	addr := net.JoinHostPort(l.host, strconv.Itoa(l.rule.Port))
	lc := net.ListenConfig{}
	ln, err := lc.Listen(listenCtx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("端口 %d 监听失败: %w", l.rule.Port, err)
	}

	l.mu.Lock()
	l.listener = ln
	l.closed = false
	l.mu.Unlock()

	stats.LogInfo("Proxy", "代理端口 [%d] 已就绪监听于 %s (策略: %s, 绑定隧道数: %d)",
		l.rule.Port, ln.Addr(), l.rule.Policy, len(l.rule.BoundTunnelIDs))

	go func() {
		<-listenCtx.Done()
		l.Close()
	}()

	for {
		client, err := ln.Accept()
		if err != nil {
			l.mu.Lock()
			isClosed := l.closed
			l.mu.Unlock()
			if isClosed {
				return nil
			}
			stats.LogWarn("Proxy", "端口 %d 接收连接异常: %v", l.rule.Port, err)
			continue
		}

		if tc, ok := client.(*net.TCPConn); ok {
			_ = tc.SetNoDelay(true)
		}
		_ = client.SetDeadline(time.Now().Add(30 * time.Second))

		select {
		case l.sem <- struct{}{}:
			stats.GetTrafficTracker().IncConn()
			go func(c net.Conn) {
				defer func() {
					<-l.sem
					stats.GetTrafficTracker().DecConn()
				}()
				l.dispatch(c)
			}(client)
		default:
			stats.LogWarn("Proxy", "端口 %d 并发超限，丢弃连接: %s", l.rule.Port, client.RemoteAddr())
			_ = client.Close()
		}
	}
}

func (l *PortListener) dispatch(client net.Conn) {
	br := bufio.NewReader(client)

	firstByte, err := br.Peek(1)
	if err != nil {
		_ = client.Close()
		return
	}
	bConn := &bufferedConn{
		Conn: client,
		br:   br,
	}

	auth := l.getAuthenticator()

	var dialer func(ctx context.Context, network, addr string) (net.Conn, error)
	devName := ""
	var selectedTun *tunnel.Tunnel

	// 1. Check if this port rule is bound to custom proxies
	if len(l.rule.BoundCustomProxyIDs) > 0 && l.customProxyMgr != nil {
		var candidates []*CustomProxy
		for _, id := range l.rule.BoundCustomProxyIDs {
			if cp := l.customProxyMgr.Get(id); cp != nil && cp.Enabled && cp.Healthy {
				candidates = append(candidates, cp)
			}
		}
		if len(candidates) > 0 {
			chosen := candidates[time.Now().UnixNano()%int64(len(candidates))]
			dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return chosen.DialContext(ctx, network, addr)
			}
		}
	}

	// 2. Check if userland dialer (SSTP zero-TUN) is active
	var userlandPacketConn func() (net.PacketConn, error)
	l.mu.Lock()
	uDialer := l.userlandDialer
	uPacketConn := l.userlandPacketConn
	l.mu.Unlock()
	if dialer == nil && uDialer != nil {
		dialer = uDialer
	}
	if uPacketConn != nil {
		userlandPacketConn = uPacketConn
	}

	// 3. Fall back to standard tunnel scheduler
	if dialer == nil && l.scheduler != nil {
		if tun := l.scheduler.SelectTunnelForRule(l.rule); tun != nil {
			selectedTun = tun
			devName = tun.DevName
		}
	}

	if firstByte[0] == socks5Version {
		_ = handleSocks5(bConn, auth, devName, selectedTun, dialer, userlandPacketConn)
	} else {
		_ = handleHTTP(bConn, br, auth, devName, selectedTun, dialer)
	}
}

func (l *PortListener) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.closed {
		l.closed = true
		if l.cancel != nil {
			l.cancel()
		}
		if l.listener != nil {
			_ = l.listener.Close()
		}
		stats.LogInfo("Proxy", "代理端口 [%d] 已停止监听", l.rule.Port)
	}
}
