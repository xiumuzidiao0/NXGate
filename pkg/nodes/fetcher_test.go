package nodes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchNodesHonorsCanceledContext(t *testing.T) {
	fetcher := NewFetcher("https://example.invalid/api", "", NewSnapshotManager(t.TempDir()))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := fetcher.FetchNodes(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestFetchNodesEarlyExit(t *testing.T) {
	sampleCSV := "*vpn_servers\n#HostName,IP,Score,Ping,Speed,CountryLong,CountryShort,NumVpnSessions,Uptime,TotalUsers,TotalTraffic,LogType,Operator,Message,OpenVPN_ConfigData_Base64\nvpn1,1.1.1.1,100,20,1000,Japan,JP,1,100,10,1000,2,op,msg,Y29uZmln\n"

	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleCSV))
	}))
	defer srv1.Close()

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleCSV))
	}))
	defer srv2.Close()

	fetcher := NewFetcher(srv1.URL, srv2.URL, NewSnapshotManager(t.TempDir()))

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := fetcher.FetchNodes(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || len(res.Data) == 0 {
		t.Fatalf("expected non-empty data from FetchNodes")
	}
	// With early-exit trigger, it should complete well before the 5s context deadline and 12s individual timeout
	if elapsed > 3500*time.Millisecond {
		t.Errorf("FetchNodes took %v, expected early exit within ~2 seconds", elapsed)
	}
}
