package singbox

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseProtocolsJSON(t *testing.T) {
	raw := `{
		"ok": true,
		"protocols": [
			{
				"id": "reality",
				"name": "VLESS-REALITY",
				"recommended": true,
				"transport": "tcp",
				"tls": "reality",
				"description": "抗封锁协议",
				"args": ["port", "uuid", "sni"]
			}
		]
	}`

	var resp ProtocolsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("failed to unmarshal protocols: %v", err)
	}
	if len(resp.Protocols) != 1 || resp.Protocols[0].ID != "reality" {
		t.Fatalf("unexpected protocols: %+v", resp)
	}
}

func TestParseNodesJSON(t *testing.T) {
	raw := `{
		"ok": true,
		"count": 1,
		"nodes": [
			{
				"name": "VLESS-REALITY-8443.json",
				"protocol": "VLESS-REALITY",
				"port": 8443,
				"outbound": "socks5://127.0.0.1:7928",
				"url": "vless://test@1.1.1.1:8443"
			}
		]
	}`

	var resp NodesResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("failed to unmarshal nodes: %v", err)
	}
	if resp.Count != 1 || resp.Nodes[0].Port != 8443 || resp.Nodes[0].Outbound != "socks5://127.0.0.1:7928" {
		t.Fatalf("unexpected nodes: %+v", resp)
	}
}

func TestParseSubJSON(t *testing.T) {
	raw := `{
		"ok": true,
		"enabled": true,
		"sub_url": "http://1.1.1.1:12345/tok/sub.yaml",
		"port": 12345,
		"node_count": 2,
		"nodes": ["vless://node1", "hysteria2://node2"]
	}`

	var resp SubResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("failed to unmarshal sub: %v", err)
	}
	if !resp.Enabled || resp.NodeCount != 2 || len(resp.Nodes) != 2 {
		t.Fatalf("unexpected sub: %+v", resp)
	}
}

func TestGetStatusCacheAndInvalidation(t *testing.T) {
	dir := t.TempDir()
	counterFile := filepath.Join(dir, "count")
	script := filepath.Join(dir, "sing-box-mock.sh")

	body := "#!/usr/bin/env sh\n" +
		"echo x >> \"" + counterFile + "\"\n" +
		"echo '{\"ok\":true,\"node_count\":2,\"subscription_enabled\":true,\"core\":{\"name\":\"sing-box\",\"version\":\"1.0.0\",\"running\":true,\"pid\":123}}'\n"
	if err := os.WriteFile(script, []byte(body), 0700); err != nil {
		t.Fatalf("write mock script failed: %v", err)
	}

	c := &Client{
		binaryPath:     script,
		statusCacheTTL: 5 * time.Second,
	}

	ctx := context.Background()
	st1, err := c.GetStatus(ctx)
	if err != nil || !st1.Core.Running || st1.NodeCount != 2 {
		t.Fatalf("first GetStatus failed: st=%+v err=%v", st1, err)
	}
	st2, err := c.GetStatus(ctx)
	if err != nil || !st2.Core.Running {
		t.Fatalf("cached GetStatus failed: st=%+v err=%v", st2, err)
	}

	rawCount, err := os.ReadFile(counterFile)
	if err != nil {
		t.Fatalf("read count failed: %v", err)
	}
	if got := len(strings.Fields(string(rawCount))); got != 1 {
		t.Fatalf("expected cached second GetStatus to avoid fork, exec count=%d", got)
	}

	c.InvalidateStatusCache()
	if _, err := c.GetStatus(ctx); err != nil {
		t.Fatalf("GetStatus after invalidate failed: %v", err)
	}
	rawCount2, _ := os.ReadFile(counterFile)
	if got := len(strings.Fields(string(rawCount2))); got != 2 {
		t.Fatalf("expected exec count=2 after cache invalidation, got %d", got)
	}
}
