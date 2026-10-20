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

	MinCacheTTL          = 15 * time.Second
	MaxCacheTTL          = 3600 * time.Second
	DefaultCacheCapacity = 1024
	maxPendingQueries    = 128
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
	client                          *http.Client
	mu                              sync.RWMutex
	cache                           map[string]*cacheEntry
	endpoints                       []string
	systemFall                      *net.Resolver
	pending                         map[string]*lookupCall
	capacity                        int
	hits, misses, shared, evictions uint64
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
		pending:    make(map[string]*lookupCall),
		capacity:   DefaultCacheCapacity,
		endpoints:  append([]string(nil), endpoints...),
		systemFall: net.DefaultResolver,
	}
}

func stripBrackets(host string) string {
	return strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
}

// Each resolver owns a cache namespace: never share one across distinct egresses.
// NewTunnelResolver disables host DNS fallback; client must dial via that tunnel.
func NewTunnelResolver(endpoints []string, client *http.Client) *DualEngineResolver {
	r := NewDualEngineResolver(endpoints, client)
	r.systemFall = nil
	return r
}

type lookupCall struct {
	done    chan struct{}
	cancel  context.CancelFunc
	waiters int
	ips     []net.IP
	err     error
}

type CacheStats struct {
	Hits, Misses, Shared, Evictions uint64
	Entries, Pending                int
}

func (r *DualEngineResolver) Stats() CacheStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return CacheStats{r.hits, r.misses, r.shared, r.evictions, len(r.cache), len(r.pending)}
}

// Clear cancels in-flight work and prevents stale results from repopulating the cache.
func (r *DualEngineResolver) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	clear(r.cache)
	for key, c := range r.pending {
		c.cancel()
		delete(r.pending, key)
	}
}
func cloneIPs(ips []net.IP) []net.IP {
	out := make([]net.IP, len(ips))
	for i, ip := range ips {
		out[i] = append(net.IP(nil), ip...)
	}
	return out
}
func (r *DualEngineResolver) pruneLocked(now time.Time) {
	for k, e := range r.cache {
		if !now.Before(e.expiresAt) {
			delete(r.cache, k)
			r.evictions++
		}
	}
}

// LookupIP coalesces concurrent cold queries without coupling one caller's deadline
// to all others. All waiters gone cancels the worker; work has its own 10s ceiling.
func (r *DualEngineResolver) LookupIP(ctx context.Context, host string) ([]net.IP, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(stripBrackets(host)), "."))
	if host == "" {
		return nil, fmt.Errorf("hostname cannot be empty")
	}
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}
	r.mu.Lock()
	r.pruneLocked(time.Now())
	if e := r.cache[host]; e != nil {
		r.hits++
		ips := cloneIPs(e.ips)
		r.mu.Unlock()
		return ips, nil
	}
	r.misses++
	c := r.pending[host]
	if c == nil {
		if len(r.pending) >= maxPendingQueries {
			r.mu.Unlock()
			return nil, fmt.Errorf("DNS lookup budget exhausted")
		}
		work, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		c = &lookupCall{done: make(chan struct{}), cancel: cancel}
		r.pending[host] = c
		go r.runLookup(work, host, c)
	} else {
		r.shared++
	}
	c.waiters++
	r.mu.Unlock()
	select {
	case <-ctx.Done():
		r.mu.Lock()
		c.waiters--
		if c.waiters == 0 && r.pending[host] == c {
			delete(r.pending, host)
			c.cancel()
		}
		r.mu.Unlock()
		return nil, ctx.Err()
	case <-c.done:
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return cloneIPs(c.ips), c.err
	}
}
func (r *DualEngineResolver) runLookup(ctx context.Context, host string, c *lookupCall) {
	defer c.cancel()
	ips, ttl, err := r.queryDoHWithFailover(ctx, host)
	if err != nil && r.systemFall != nil && ctx.Err() == nil {
		ips, err = r.systemFall.LookupIP(ctx, "ip", host)
		ttl = 60 * time.Second
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if ctx.Err() != nil {
		ips = nil
		err = ctx.Err()
	}
	c.ips, c.err = cloneIPs(ips), err
	if r.pending[host] == c {
		delete(r.pending, host)
		if err == nil && len(ips) > 0 {
			r.saveCacheLocked(host, ips, ttl)
		}
	}
	close(c.done)
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
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saveCacheLocked(host, ips, ttl)
}
func (r *DualEngineResolver) saveCacheLocked(host string, ips []net.IP, ttl time.Duration) {
	// Never extend authoritative TTLs, and do not negative-cache failures.
	if ttl <= 0 || len(ips) == 0 {
		return
	}
	if ttl > MaxCacheTTL {
		ttl = MaxCacheTTL
	}
	r.pruneLocked(time.Now())
	capacity := r.capacity
	if capacity <= 0 {
		capacity = DefaultCacheCapacity
	}
	if _, exists := r.cache[host]; !exists && len(r.cache) >= capacity {
		var oldest string
		var expiry time.Time
		for k, e := range r.cache {
			if oldest == "" || e.expiresAt.Before(expiry) {
				oldest = k
				expiry = e.expiresAt
			}
		}
		delete(r.cache, oldest)
		r.evictions++
	}
	r.cache[host] = &cacheEntry{ips: cloneIPs(ips), expiresAt: time.Now().Add(ttl)}
}

func (r *DualEngineResolver) queryDoHWithFailover(ctx context.Context, host string) ([]net.IP, time.Duration, error) {
	var lastErr error

	for _, endpoint := range r.endpoints {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
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
	minTTL := int(MaxCacheTTL / time.Second)
	for _, a := range append(append([]DoHAnswer(nil), answersA...), answersAAAA...) {
		if a.TTL >= 0 && a.TTL < minTTL {
			minTTL = a.TTL
		}
	}

	for _, a := range answersA {
		if ip := net.ParseIP(strings.TrimSpace(a.Data)); ip != nil && ip.To4() != nil {
			ips = append(ips, ip.To4())
			if a.TTL >= 0 && a.TTL < minTTL {
				minTTL = a.TTL
			}
		}
	}
	for _, a := range answersAAAA {
		if ip := net.ParseIP(strings.TrimSpace(a.Data)); ip != nil && ip.To4() == nil {
			ips = append(ips, ip)
			if a.TTL >= 0 && a.TTL < minTTL {
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

	if dohResp.Status != 0 || dohResp.TC {
		return nil, fmt.Errorf("unsuccessful or truncated DoH response: %d", dohResp.Status)
	}
	return dohResp.Answer, nil
}
