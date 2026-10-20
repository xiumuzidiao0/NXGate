package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"aimili-vpngate-go/pkg/stats"
)

// CustomProxy represents a user-managed external outbound proxy (SOCKS5/HTTP/HTTPS).
type CustomProxy struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // "socks5", "http", "https"
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Username    string    `json:"username,omitempty"`
	Password    string    `json:"password,omitempty"`
	RawURL      string    `json:"raw_url"`
	Tag         string    `json:"tag"`
	Enabled     bool      `json:"enabled"`
	Healthy     bool      `json:"healthy"`
	LatencyMs   int       `json:"latency_ms"`
	ExitIP      string    `json:"exit_ip,omitempty"`
	Country     string    `json:"country,omitempty"`
	ISP         string    `json:"isp,omitempty"`
	RiskLevel   string    `json:"risk_level,omitempty"`
	LastError   string    `json:"last_error,omitempty"`
	LastChecked time.Time `json:"last_checked,omitempty"`
}

// ParseProxyURL parses a raw proxy address into a CustomProxy object.
// Supported schemes: socks5://, http://, https://.
// If scheme is omitted, socks5:// is assumed as default.
func ParseProxyURL(raw string) (*CustomProxy, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("代理地址不能为空")
	}

	input := raw
	if !strings.Contains(input, "://") {
		input = "socks5://" + input
	}

	u, err := url.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("解析代理地址失败: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "socks5" && scheme != "socks" && scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("不支持的代理协议: %s (仅支持 socks5, http, https)", scheme)
	}
	if scheme == "socks" {
		scheme = "socks5"
	}

	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("代理地址缺少主机名或 IP")
	}

	portStr := u.Port()
	port := 0
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil || p <= 0 || p > 65535 {
			return nil, fmt.Errorf("无效的代理端口: %s", portStr)
		}
		port = p
	} else {
		switch scheme {
		case "socks5":
			port = 1080
		case "http":
			port = 80
		case "https":
			port = 443
		}
	}

	username := ""
	password := ""
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}

	id := fmt.Sprintf("%s-%s-%d", scheme, host, port)

	return &CustomProxy{
		ID:        id,
		Type:      scheme,
		Host:      host,
		Port:      port,
		Username:  username,
		Password:  password,
		RawURL:    raw,
		Enabled:   true,
		LatencyMs: -1,
	}, nil
}

// DialContext establishes a connection to the target address through this custom proxy.
func (p *CustomProxy) DialContext(ctx context.Context, network, targetAddr string) (net.Conn, error) {
	proxyAddr := net.JoinHostPort(p.Host, strconv.Itoa(p.Port))
	d := net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	rawConn, err := d.DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("连接上游代理 %s 失败: %w", proxyAddr, err)
	}

	switch p.Type {
	case "socks5":
		conn, err := p.handshakeSOCKS5(ctx, rawConn, targetAddr)
		if err != nil {
			_ = rawConn.Close()
			return nil, err
		}
		return conn, nil

	case "http", "https":
		conn, err := p.handshakeHTTP(ctx, rawConn, targetAddr, p.Type == "https")
		if err != nil {
			_ = rawConn.Close()
			return nil, err
		}
		return conn, nil

	default:
		_ = rawConn.Close()
		return nil, fmt.Errorf("未知代理协议: %s", p.Type)
	}
}

// handshakeSOCKS5 performs RFC 1928 + RFC 1929 client handshake.
func (p *CustomProxy) handshakeSOCKS5(ctx context.Context, conn net.Conn, targetAddr string) (net.Conn, error) {
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	// 1. Send method negotiation
	var methods []byte
	if p.Username != "" && p.Password != "" {
		methods = []byte{0x05, 0x02, 0x00, 0x02} // No-auth and User/Password
	} else {
		methods = []byte{0x05, 0x01, 0x00} // No-auth only
	}

	if _, err := conn.Write(methods); err != nil {
		return nil, fmt.Errorf("SOCKS5 握手认证协商写入失败: %w", err)
	}

	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return nil, fmt.Errorf("SOCKS5 握手认证协商读取失败: %w", err)
	}
	if resp[0] != 0x05 {
		return nil, fmt.Errorf("SOCKS5 协议版本错误: %d", resp[0])
	}

	selectedMethod := resp[1]
	if selectedMethod == 0x02 { // Username/Password (RFC 1929)
		uBytes := []byte(p.Username)
		pBytes := []byte(p.Password)
		if len(uBytes) > 255 || len(pBytes) > 255 {
			return nil, fmt.Errorf("SOCKS5 用户名或密码长度超出 255 字节限制")
		}

		authBuf := append([]byte{0x01, byte(len(uBytes))}, uBytes...)
		authBuf = append(authBuf, byte(len(pBytes)))
		authBuf = append(authBuf, pBytes...)

		if _, err := conn.Write(authBuf); err != nil {
			return nil, fmt.Errorf("SOCKS5 认证凭据写入失败: %w", err)
		}

		authResp := make([]byte, 2)
		if _, err := io.ReadFull(conn, authResp); err != nil {
			return nil, fmt.Errorf("SOCKS5 认证响应读取失败: %w", err)
		}
		if authResp[1] != 0x00 {
			return nil, fmt.Errorf("SOCKS5 用户名/密码认证被拒绝 (status %d)", authResp[1])
		}
	} else if selectedMethod != 0x00 {
		return nil, fmt.Errorf("SOCKS5 上游不支持的认证方式: 0x%02x", selectedMethod)
	}

	// 2. Send CONNECT command (RFC 1928)
	host, portStr, err := net.SplitHostPort(targetAddr)
	if err != nil {
		return nil, fmt.Errorf("解析目标地址 %s 失败: %w", targetAddr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("解析目标端口 %s 失败: %w", portStr, err)
	}

	var req []byte
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			req = append([]byte{0x05, 0x01, 0x00, 0x01}, ip4...)
		} else {
			req = append([]byte{0x05, 0x01, 0x00, 0x04}, ip.To16()...)
		}
	} else {
		if len(host) > 255 {
			return nil, fmt.Errorf("目标主机域名超出 255 字节限制: %s", host)
		}
		req = append([]byte{0x05, 0x01, 0x00, 0x03, byte(len(host))}, []byte(host)...)
	}
	req = append(req, byte(port>>8), byte(port&0xff))

	if _, err := conn.Write(req); err != nil {
		return nil, fmt.Errorf("SOCKS5 CONNECT 请求发送失败: %w", err)
	}

	// Read CONNECT reply
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, fmt.Errorf("读取 SOCKS5 CONNECT 响应头失败: %w", err)
	}
	if header[1] != 0x00 {
		return nil, fmt.Errorf("SOCKS5 CONNECT 连接目标失败 (rep 0x%02x)", header[1])
	}

	// Read bound address according to type
	switch header[3] {
	case 0x01: // IPv4
		bnd := make([]byte, 4+2)
		_, _ = io.ReadFull(conn, bnd)
	case 0x03: // Domain
		dLen := make([]byte, 1)
		_, _ = io.ReadFull(conn, dLen)
		bnd := make([]byte, int(dLen[0])+2)
		_, _ = io.ReadFull(conn, bnd)
	case 0x04: // IPv6
		bnd := make([]byte, 16+2)
		_, _ = io.ReadFull(conn, bnd)
	}

	_ = conn.SetDeadline(time.Time{})
	return conn, nil
}

// handshakeHTTP performs HTTP/HTTPS CONNECT tunnel handshake.
func (p *CustomProxy) handshakeHTTP(ctx context.Context, rawConn net.Conn, targetAddr string, isTLS bool) (net.Conn, error) {
	conn := rawConn
	if isTLS {
		tlsConn := tls.Client(rawConn, &tls.Config{
			ServerName:         p.Host,
			InsecureSkipVerify: false,
		})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			return nil, fmt.Errorf("HTTPS 代理 TLS 握手失败: %w", err)
		}
		conn = tlsConn
	}

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	var authHeader string
	if p.Username != "" && p.Password != "" {
		creds := base64.StdEncoding.EncodeToString([]byte(p.Username + ":" + p.Password))
		authHeader = fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", creds)
	}

	req := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n%sUser-Agent: NXGate/2.6\r\nProxy-Connection: Keep-Alive\r\n\r\n",
		targetAddr, targetAddr, authHeader)

	if _, err := io.WriteString(conn, req); err != nil {
		return nil, fmt.Errorf("发送 HTTP CONNECT 请求失败: %w", err)
	}

	reader := bufio.NewReader(conn)
	respLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("读取 HTTP CONNECT 响应状态行失败: %w", err)
	}

	parts := strings.SplitN(respLine, " ", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("非法 HTTP 响应状态: %s", strings.TrimSpace(respLine))
	}
	statusCode, err := strconv.Atoi(parts[1])
	if err != nil || statusCode != 200 {
		return nil, fmt.Errorf("HTTP 代理隧道建立失败 (HTTP %s)", strings.TrimSpace(respLine))
	}

	// Consume remaining HTTP headers until \r\n\r\n
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("读取 HTTP 响应头失败: %w", err)
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	_ = conn.SetDeadline(time.Time{})
	return conn, nil
}

// Probe performs an active connectivity test through this custom proxy, measuring RTT
// and identifying the actual public exit IP.
func (p *CustomProxy) Probe(ctx context.Context, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	// Connect to 1.1.1.1:80 for fast cloudflare cdn-cgi/trace probe
	conn, err := p.DialContext(probeCtx, "tcp", "1.1.1.1:80")
	if err != nil {
		p.Healthy = false
		p.LatencyMs = -1
		p.LastError = err.Error()
		p.LastChecked = time.Now()
		return err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))

	req := "GET /cdn-cgi/trace HTTP/1.1\r\nHost: 1.1.1.1\r\nUser-Agent: NXGate-Probe\r\nConnection: close\r\n\r\n"
	if _, err := io.WriteString(conn, req); err != nil {
		p.Healthy = false
		p.LatencyMs = -1
		p.LastError = err.Error()
		p.LastChecked = time.Now()
		return err
	}

	respBytes, err := io.ReadAll(io.LimitReader(conn, 4096))
	if err != nil {
		p.Healthy = false
		p.LatencyMs = -1
		p.LastError = err.Error()
		p.LastChecked = time.Now()
		return err
	}

	rtt := int(time.Since(start).Milliseconds())
	p.LatencyMs = rtt
	p.Healthy = true
	p.LastError = ""
	p.LastChecked = time.Now()

	// Parse ip=... in cdn-cgi/trace
	respStr := string(respBytes)
	for _, line := range strings.Split(respStr, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ip=") {
			p.ExitIP = strings.TrimPrefix(line, "ip=")
			break
		}
	}

	return nil
}

// CustomProxyManager coordinates external proxy CRUD, probing, and persistence.
type CustomProxyManager struct {
	filePath string
	mu       sync.RWMutex
	proxies  map[string]*CustomProxy
}

// NewCustomProxyManager initializes and loads existing custom proxies.
func NewCustomProxyManager(dataDir string) *CustomProxyManager {
	m := &CustomProxyManager{
		filePath: filepath.Join(dataDir, "custom_proxies.json"),
		proxies:  make(map[string]*CustomProxy),
	}
	m.load()
	return m
}

func (m *CustomProxyManager) load() {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return
	}

	var list []*CustomProxy
	if err := json.Unmarshal(data, &list); err == nil {
		for _, p := range list {
			if p != nil && p.ID != "" {
				m.proxies[p.ID] = p
			}
		}
	}
}

func (m *CustomProxyManager) saveLocked() {
	var list []*CustomProxy
	for _, p := range m.proxies {
		list = append(list, p)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return
	}
	tmp := m.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err == nil {
		_ = os.Chmod(tmp, 0600)
		_ = os.Rename(tmp, m.filePath)
	}
}

// AddOrUpdate parses and stores a custom proxy definition.
func (m *CustomProxyManager) AddOrUpdate(rawURL, tag string) (*CustomProxy, error) {
	p, err := ParseProxyURL(rawURL)
	if err != nil {
		return nil, err
	}
	if tag != "" {
		p.Tag = tag
	}

	m.mu.Lock()
	m.proxies[p.ID] = p
	m.saveLocked()
	m.mu.Unlock()

	stats.LogInfo("ProxyManager", "已导入自定义外部代理: [%s] %s (%s)", p.Type, p.Host, p.ID)
	snapshot := *p
	return &snapshot, nil
}

// BatchImport imports multiple proxy lines at once.
func (m *CustomProxyManager) BatchImport(lines []string, tag string) ([]*CustomProxy, []error) {
	var added []*CustomProxy
	var errs []error

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p, err := m.AddOrUpdate(line, tag)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", line, err))
		} else {
			added = append(added, p)
		}
	}
	return added, errs
}

// Delete removes a custom proxy by ID.
func (m *CustomProxyManager) Delete(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.proxies[id]; exists {
		delete(m.proxies, id)
		m.saveLocked()
		return true
	}
	return false
}

// Get returns a custom proxy by ID.
func (m *CustomProxyManager) Get(id string) *CustomProxy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p := m.proxies[id]; p != nil {
		c := *p
		return &c
	}
	return nil
}

// List returns a slice of all registered custom proxies.
func (m *CustomProxyManager) List() []*CustomProxy {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*CustomProxy
	for _, p := range m.proxies {
		// return clone to prevent caller mutation
		c := *p
		list = append(list, &c)
	}
	return list
}

// GetHealthy returns all proxies marked as healthy and enabled.
func (m *CustomProxyManager) GetHealthy() []*CustomProxy {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*CustomProxy
	for _, p := range m.proxies {
		if p.Enabled && p.Healthy {
			c := *p
			list = append(list, &c)
		}
	}
	return list
}

// Probe checks a detached copy and commits only if the definition was not replaced/deleted.
func (m *CustomProxyManager) Probe(ctx context.Context, id string, timeout time.Duration) (*CustomProxy, error) {
	m.mu.RLock()
	original := m.proxies[id]
	if original == nil {
		m.mu.RUnlock()
		return nil, fmt.Errorf("proxy not found")
	}
	snapshot := *original
	m.mu.RUnlock()
	err := snapshot.Probe(ctx, timeout)
	m.mu.Lock()
	if m.proxies[id] == original {
		// Replace, never mutate an object that may have been read by another probe.
		result := snapshot
		m.proxies[id] = &result
		m.saveLocked()
	}
	m.mu.Unlock()
	return &snapshot, err
}

// ProbeAll concurrently checks detached snapshots.
func (m *CustomProxyManager) ProbeAll(ctx context.Context) []*CustomProxy {
	targets := m.List()
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)
	for _, p := range targets {
		if !p.Enabled {
			continue
		}
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			_, _ = m.Probe(ctx, id, 6*time.Second)
		}(p.ID)
	}
	wg.Wait()
	return m.List()
}
