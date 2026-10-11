package dns

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDualEngineResolver_DirectIP(t *testing.T) {
	r := NewDualEngineResolver(nil, nil)

	ctx := context.Background()
	ips, err := r.LookupIP(ctx, "1.2.3.4")
	if err != nil {
		t.Fatalf("LookupIP direct IPv4 failed: %v", err)
	}
	if len(ips) != 1 || ips[0].String() != "1.2.3.4" {
		t.Errorf("expected [1.2.3.4], got %v", ips)
	}

	ips6, err := r.LookupIP(ctx, "[2001:db8::1]")
	if err != nil {
		t.Fatalf("LookupIP direct IPv6 failed: %v", err)
	}
	if len(ips6) != 1 || ips6[0].String() != "2001:db8::1" {
		t.Errorf("expected [2001:db8::1], got %v", ips6)
	}
}

func TestDualEngineResolver_MockDoHSuccessAndCache(t *testing.T) {
	mockCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockCalls++
		name := r.URL.Query().Get("name")
		qType := r.URL.Query().Get("type")

		if name != "example.com" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		resp := DoHResponse{
			Status: 0,
		}

		if qType == TypeA {
			resp.Answer = []DoHAnswer{
				{Name: "example.com", Type: 1, TTL: 60, Data: "93.184.216.34"},
			}
		} else if qType == TypeAAAA {
			resp.Answer = []DoHAnswer{
				{Name: "example.com", Type: 28, TTL: 60, Data: "2606:2800:220:1:248:1893:25c8:1946"},
			}
		}

		w.Header().Set("Content-Type", "application/dns-json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	resolver := NewDualEngineResolver([]string{server.URL}, server.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 1. First lookup: hits mock server
	ips, err := resolver.LookupIP(ctx, "example.com")
	if err != nil {
		t.Fatalf("LookupIP failed: %v", err)
	}
	if len(ips) < 2 {
		t.Fatalf("expected at least 2 IPs (A & AAAA), got %d", len(ips))
	}

	firstCallCount := mockCalls

	// 2. Second lookup: should hit in-memory cache, mockCalls must not increment!
	ipsCached, err := resolver.LookupIP(ctx, "example.com")
	if err != nil {
		t.Fatalf("cached LookupIP failed: %v", err)
	}
	if len(ipsCached) != len(ips) {
		t.Errorf("cached IPs length mismatch")
	}
	if mockCalls != firstCallCount {
		t.Errorf("expected cache hit with no new HTTP calls, before: %d, after: %d", firstCallCount, mockCalls)
	}

	// 3. Test LookupIPv4
	v4s, err := resolver.LookupIPv4(ctx, "example.com")
	if err != nil || len(v4s) != 1 || v4s[0].String() != "93.184.216.34" {
		t.Errorf("LookupIPv4 failed: %v, got %v", err, v4s)
	}

	// 4. Test LookupIPv6
	v6s, err := resolver.LookupIPv6(ctx, "example.com")
	if err != nil || len(v6s) != 1 || v6s[0].String() != "2606:2800:220:1:248:1893:25c8:1946" {
		t.Errorf("LookupIPv6 failed: %v, got %v", err, v6s)
	}
}

func TestDualEngineResolver_Failover(t *testing.T) {
	// Endpoint 1: Always fails (500 Internal Error)
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Cloudflare DoH down", http.StatusInternalServerError)
	}))
	defer server1.Close()

	// Endpoint 2: Works (Google DoH simulation)
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := DoHResponse{
			Status: 0,
			Answer: []DoHAnswer{
				{Name: "google-doh.test", Type: 1, TTL: 120, Data: "198.51.100.1"},
			},
		}
		w.Header().Set("Content-Type", "application/dns-json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server2.Close()

	resolver := NewDualEngineResolver([]string{server1.URL, server2.URL}, http.DefaultClient)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ips, err := resolver.LookupIPv4(ctx, "google-doh.test")
	if err != nil {
		t.Fatalf("failover resolution failed: %v", err)
	}
	if len(ips) != 1 || ips[0].String() != "198.51.100.1" {
		t.Errorf("expected [198.51.100.1] from backup DoH endpoint, got %v", ips)
	}
}
