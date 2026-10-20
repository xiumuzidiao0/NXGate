package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestOptimizationCacheBoundedTTLAndCopies(t *testing.T) {
	r := NewTunnelResolver(nil, nil)
	r.capacity = 2
	for _, h := range []string{"a", "b", "c"} {
		r.saveCache(h, []net.IP{net.ParseIP("192.0.2.1")}, time.Hour)
	}
	if s := r.Stats(); s.Entries != 2 || s.Evictions != 1 {
		t.Fatal(s)
	}
	r.saveCache("zero", []net.IP{net.ParseIP("192.0.2.2")}, 0)
	if r.Stats().Entries != 2 {
		t.Fatal("zero TTL cached")
	}
	r.Clear()
	r.saveCache("a", []net.IP{net.ParseIP("192.0.2.1")}, time.Second)
	ips, err := r.LookupIP(context.Background(), "A.")
	if err != nil {
		t.Fatal(err)
	}
	ips[0][0] = 0
	ips, _ = r.LookupIP(context.Background(), "a")
	if ips[0].String() != "192.0.2.1" {
		t.Fatal("mutable cache")
	}
	r.mu.Lock()
	r.cache["a"].expiresAt = time.Now().Add(-time.Second)
	r.mu.Unlock()
	r.saveCache("b", []net.IP{net.ParseIP("192.0.2.1")}, time.Hour)
	if r.Stats().Entries != 1 {
		t.Fatal("expired entries not pruned")
	}
}
func TestOptimizationDNSCoalescingAndCancelIsolation(t *testing.T) {
	var calls atomic.Int32
	gate := make(chan struct{})
	entered := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case entered <- struct{}{}:
		default:
		}
		select {
		case <-gate:
		case <-r.Context().Done():
			return
		}
		answers := []DoHAnswer{}
		if r.URL.Query().Get("type") == TypeA {
			answers = []DoHAnswer{{Type: 1, Data: "192.0.2.1", TTL: 1}}
		}
		json.NewEncoder(w).Encode(DoHResponse{Answer: answers})
	}))
	defer server.Close()
	r := NewTunnelResolver([]string{server.URL}, server.Client())
	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan error, 1)
	go func() { _, e := r.LookupIP(ctx, "test.invalid"); first <- e }()
	<-entered
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := r.LookupIP(context.Background(), "test.invalid"); errs <- e }()
	}
	deadline := time.Now().Add(time.Second)
	for r.Stats().Shared < 20 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if r.Stats().Shared != 20 {
		t.Fatal("waiters not joined")
	}
	cancel()
	if e := <-first; e != context.Canceled {
		t.Fatal(e)
	}
	close(gate)
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("expected one A+AAAA query", calls.Load())
	}
	if r.systemFall != nil {
		t.Fatal("tunnel host DNS fallback")
	}
	if r.Stats().Entries != 1 {
		t.Fatal(r.Stats())
	}
}
func TestOptimizationDNSClearCancelsAndIsolates(t *testing.T) {
	entered := make(chan struct{}, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { entered <- struct{}{}; <-r.Context().Done() }))
	defer server.Close()
	r := NewTunnelResolver([]string{server.URL}, server.Client())
	done := make(chan error, 1)
	go func() { _, e := r.LookupIP(context.Background(), "gone.invalid"); done <- e }()
	<-entered
	r.Clear()
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("clear success")
		}
	case <-time.After(time.Second):
		t.Fatal("worker not canceled")
	}
	if r.Stats().Entries != 0 || r.Stats().Pending != 0 {
		t.Fatal(r.Stats())
	}
	r2 := NewTunnelResolver(nil, nil)
	r.saveCache("same", []net.IP{net.ParseIP("192.0.2.1")}, time.Hour)
	if r2.Stats().Entries != 0 {
		t.Fatal("shared egress cache")
	}
}
func TestOptimizationDNSFailuresNotCached(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"Status":2}`) }))
	defer server.Close()
	r := NewTunnelResolver([]string{server.URL}, server.Client())
	if _, e := r.LookupIP(context.Background(), "bad.invalid"); e == nil {
		t.Fatal("expected failure")
	}
	if r.Stats().Entries != 0 || r.Stats().Pending != 0 {
		t.Fatal(r.Stats())
	}
}
