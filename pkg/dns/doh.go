package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	CloudflareDoHEndpoint = "https://cloudflare-dns.com/dns-query"
	CloudflareDoHIP       = "https://1.1.1.1/dns-query"
	GoogleDoHEndpoint     = "https://dns.google/resolve"
	GoogleDoHIP           = "https://8.8.8.8/resolve"

	TypeA    = "A"    // 1
	TypeAAAA = "AAAA" // 28
	TypeTXT  = "TXT"  // 16

	MinCacheTTL = 15 * time.Second
	MaxCacheTTL = 3600 * time.Second
)

// DoHAnswer represents a single DNS record answer returned by Cloudflare/Google DoH JSON API.
type DoHAnswer struct {
	Name string `json:"name"`
	Type int    `json:"type"`
	TTL  int    `json:"TTL"`
	Data string `json:"data"`
}

// DoHResponse represents the RFC 8427 / DNS-over-HTTPS JSON response structure.
type DoHResponse struct {
	Status   int         `json:"Status"`
	TC       bool        `json:"TC"`
	RD       bool        `json:"RD"`
	RA       bool        `json:"RA"`
	AD       bool        `json:"AD"`
	CD       bool        `json:"CD"`
	Question []any       `json:"Question"`
	Answer   []DoHAnswer `json:"Answer,omitempty"`
}

type cacheEntry struct {
	ips       []net.IP
	expiresAt time.Time
}

// DualEngineResolver provides anti-pollution, high-availability DNS resolution via
// Cloudflare and Google DoH, with graceful fallback to system resolver.
type DualEngineResolver struct {
	client     *http.Client
	mu         sync.RWMutex
	cache      map[string]*cacheEntry
	endpoints  []string
	systemFall *net.Resolver
}

var (
	defaultResolver     *DualEngineResolver
	defaultResolverOnce sync.Once
)

// Default returns the global singleton DualEngineResolver.
func Default() *DualEngineResolver {
	defaultResolverOnce.Do(func() {
		defaultResolver = NewDualEngineResolver(
			[]string{CloudflareDoHEndpoint, GoogleDoHEndpoint, CloudflareDoHIP, GoogleDoHIP},
			nil,
		)
	})
	return defaultResolver
}

// NewDualEngineResolver creates a custom DualEngineResolver with custom endpoints or HTTP client.
func NewDualEngineResolver(endpoints []string, client *http.Client) *DualEngineResolver {
	if client == nil {
		client = &http.Client{
			Timeout: 5 * time.Second,
		}
	}
	if len(endpoints) == 0 {
		endpoints = []string{CloudflareDoHEndpoint, GoogleDoHEndpoint}
	}
	return &DualEngineResolver{
		client:     client,
		cache:      make(map[string]*cacheEntry),
		endpoints:  endpoints,
		systemFall: net.DefaultResolver,
	}
}

func stripBrackets(host string) string {
	return strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
}

// LookupIP resolves a hostname to a list of IP addresses (IPv4 & IPv6).
func (r *DualEngineResolver) LookupIP(ctx context.Context, host string) ([]net.IP, error) {
	cleanHost := strings.TrimSpace(stripBrackets(host))
	if cleanHost == "" {
		return nil, fmt.Errorf("hostname cannot be empty")
	}

	// 1. If it's already an IP address, return immediately
	if ip := net.ParseIP(cleanHost); ip != nil {
		return []net.IP{ip}, nil
	}

	// 2. Check local memory cache
	r.mu.RLock()
	if entry, ok := r.cache[cleanHost]; ok && time.Now().Before(entry.expiresAt) {
		r.mu.RUnlock()
		return entry.ips, nil
	}
	r.mu.RUnlock()

	// 3. Query DoH endpoints with failover
	ips, ttl, err := r.queryDoHWithFailover(ctx, cleanHost)
	if err == nil && len(ips) > 0 {
		r.saveCache(cleanHost, ips, ttl)
		return ips, nil
	}

	// 4. Fallback to system resolver
	sysIPs, sysErr := r.systemFall.LookupIP(ctx, "ip", cleanHost)
	if sysErr == nil && len(sysIPs) > 0 {
		r.saveCache(cleanHost, sysIPs, 60*time.Second)
		return sysIPs, nil
	}

	if err != nil {
		return nil, err
	}
	return nil, sysErr
}

// LookupIPv4 resolves a hostname to IPv4 addresses only.
func (r *DualEngineResolver) LookupIPv4(ctx context.Context, host string) ([]net.IP, error) {
	all, err := r.LookupIP(ctx, host)
	if err != nil {
		return nil, err
	}
	var ipv4s []net.IP
	for _, ip := range all {
		if ip4 := ip.To4(); ip4 != nil {
			ipv4s = append(ipv4s, ip4)
		}
	}
	if len(ipv4s) == 0 {
		return nil, fmt.Errorf("no IPv4 address found for %s", host)
	}
	return ipv4s, nil
}

// LookupIPv6 resolves a hostname to IPv6 addresses only.
func (r *DualEngineResolver) LookupIPv6(ctx context.Context, host string) ([]net.IP, error) {
	all, err := r.LookupIP(ctx, host)
	if err != nil {
		return nil, err
	}
	var ipv6s []net.IP
	for _, ip := range all {
		if ip.To4() == nil && ip.To16() != nil {
			ipv6s = append(ipv6s, ip)
		}
	}
	if len(ipv6s) == 0 {
		return nil, fmt.Errorf("no IPv6 address found for %s", host)
	}
	return ipv6s, nil
}

func (r *DualEngineResolver) saveCache(host string, ips []net.IP, ttl time.Duration) {
	if ttl < MinCacheTTL {
		ttl = MinCacheTTL
	} else if ttl > MaxCacheTTL {
		ttl = MaxCacheTTL
	}

	r.mu.Lock()
	r.cache[host] = &cacheEntry{
		ips:       ips,
		expiresAt: time.Now().Add(ttl),
	}
	r.mu.Unlock()
}

func (r *DualEngineResolver) queryDoHWithFailover(ctx context.Context, host string) ([]net.IP, time.Duration, error) {
	var lastErr error

	for _, endpoint := range r.endpoints {
		ips, ttl, err := r.querySingleEndpoint(ctx, endpoint, host)
		if err == nil && len(ips) > 0 {
			return ips, ttl, nil
		}
		if err != nil {
			lastErr = err
		}
	}

	if lastErr != nil {
		return nil, 0, lastErr
	}
	return nil, 0, fmt.Errorf("no DoH answer received from configured endpoints")
}

func (r *DualEngineResolver) querySingleEndpoint(ctx context.Context, endpoint, host string) ([]net.IP, time.Duration, error) {
	// Query IPv4 (A) record
	answersA, errA := r.queryType(ctx, endpoint, host, TypeA)
	// Query IPv6 (AAAA) record
	answersAAAA, _ := r.queryType(ctx, endpoint, host, TypeAAAA)

	if errA != nil && len(answersAAAA) == 0 {
		return nil, 0, errA
	}

	var ips []net.IP
	minTTL := 300

	for _, a := range answersA {
		if ip := net.ParseIP(strings.TrimSpace(a.Data)); ip != nil && ip.To4() != nil {
			ips = append(ips, ip.To4())
			if a.TTL > 0 && a.TTL < minTTL {
				minTTL = a.TTL
			}
		}
	}
	for _, a := range answersAAAA {
		if ip := net.ParseIP(strings.TrimSpace(a.Data)); ip != nil && ip.To4() == nil {
			ips = append(ips, ip)
			if a.TTL > 0 && a.TTL < minTTL {
				minTTL = a.TTL
			}
		}
	}

	if len(ips) == 0 {
		return nil, 0, fmt.Errorf("no valid IP answers in DoH response")
	}

	return ips, time.Duration(minTTL) * time.Second, nil
}

func (r *DualEngineResolver) queryType(ctx context.Context, endpoint, host, qType string) ([]DoHAnswer, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("name", host)
	q.Set("type", qType)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/dns-json")
	req.Header.Set("User-Agent", "NXGate-DoH/2.6")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DoH endpoint %s returned status: %d", endpoint, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, err
	}

	var dohResp DoHResponse
	if err := json.Unmarshal(body, &dohResp); err != nil {
		return nil, fmt.Errorf("failed to decode DoH json from %s: %w", endpoint, err)
	}

	return dohResp.Answer, nil
}
