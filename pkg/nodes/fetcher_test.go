package nodes

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
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

	// Production includes built-in public mirrors. Fixtures must never contact them.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil || host != "127.0.0.1" {
			return nil, fmt.Errorf("non-local test dial blocked: %s", addr)
		}
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}
	defer transport.CloseIdleConnections()
	fetcher.client.Transport = transport

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

func TestFetchURLDecompressesGzipCSV(t *testing.T) {
	sampleCSV := "*vpn_servers\n#HostName,IP,Score,Ping,Speed,CountryLong,CountryShort,NumVpnSessions,Uptime,TotalUsers,TotalTraffic,LogType,Operator,Message,OpenVPN_ConfigData_Base64\nvpn1,1.1.1.1,100,20,1000,Japan,JP,1,100,10,1000,2,op,msg,Y29uZmln\n"
	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	_, _ = gw.Write([]byte(sampleCSV))
	_ = gw.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/vpngate.csv.gz":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(gzBuf.Bytes())
		case "/corrupt.csv.gz":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte{0x1f, 0x8b, 0x08, 0x00, 0xff, 0xff})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	fetcher := NewFetcher("", "", NewSnapshotManager(t.TempDir()))
	got, err := fetcher.fetchURL(context.Background(), srv.URL+"/vpngate.csv.gz")
	if err != nil {
		t.Fatalf("expected gzip mirror fetch to succeed, got %v", err)
	}
	if !strings.Contains(string(got), "HostName") || !strings.Contains(string(got), "1.1.1.1") {
		t.Fatalf("expected decompressed CSV content, got %q", string(got))
	}

	if _, err := fetcher.fetchURL(context.Background(), srv.URL+"/corrupt.csv.gz"); err == nil {
		t.Fatalf("expected corrupt gzip payload to fail validation")
	}
}
