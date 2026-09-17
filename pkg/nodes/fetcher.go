package nodes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"aimili-vpngate-go/pkg/stats"
)

type Fetcher struct {
	apiURL    string
	mirrorURL string
	snapshot  *SnapshotManager
	client    *http.Client
}

func NewFetcher(apiURL, mirrorURL string, snapshot *SnapshotManager) *Fetcher {
	return &Fetcher{
		apiURL:    apiURL,
		mirrorURL: mirrorURL,
		snapshot:  snapshot,
		client: &http.Client{
			Timeout: 25 * time.Second,
		},
	}
}

func (f *Fetcher) fetchURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "AimiliVPN-Go/2.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected http status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxSnapshotBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > MaxSnapshotBytes {
		return nil, fmt.Errorf("response exceeds max snapshot size %d bytes", MaxSnapshotBytes)
	}

	// Validate content is actually VPNGate CSV and not an ISP intercept error
	trimmed := bytes.TrimSpace(body)
	if bytes.HasPrefix(trimmed, []byte("<")) || bytes.HasPrefix(trimmed, []byte("<!DOCTYPE")) || bytes.HasPrefix(trimmed, []byte("<html")) {
		return nil, fmt.Errorf("response contains HTML tags, not a valid VPNGate CSV (possible ISP interception)")
	}

	if !bytes.Contains(body, []byte("HostName")) && !bytes.Contains(body, []byte("vpn_servers")) {
		return nil, fmt.Errorf("response does not contain required VPNGate CSV headers")
	}

	return body, nil
}

type FetchResult struct {
	Data   []byte
	Source string
}

func (f *Fetcher) FetchNodes(ctx context.Context) (*FetchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	sources := []struct {
		name string
		url  string
	}{
		{"GitHub Pages 实时镜像", f.mirrorURL},
		{"VPNGate 官方 HTTPS API", f.apiURL},
		{"VPNGate 官方 HTTP API", "http://www.vpngate.net/api/iphone/"},
		{"GitHub Raw 直链镜像", "https://raw.githubusercontent.com/baoweise-bot/aimili-vpngate/main/mirror/vpngate.csv"},
		{"Fastly 全球加速 CDN", "https://fastly.jsdelivr.net/gh/baoweise-bot/aimili-vpngate@main/mirror/vpngate.csv"},
		{"GitHub 镜像加速源", "https://ghproxy.net/https://raw.githubusercontent.com/baoweise-bot/aimili-vpngate/main/mirror/vpngate.csv"},
		{"用户 GitHub 镜像源", "https://raw.githubusercontent.com/xiumuzidiao0/NXGate/main/mirror/vpngate.csv"},
		{"jsDelivr 全球加速 CDN", "https://cdn.jsdelivr.net/gh/baoweise-bot/aimili-vpngate@main/mirror/vpngate.csv"},
	}

	type sourceData struct {
		name string
		data []byte
	}

	var mu sync.Mutex
	var successes []sourceData

	var wg sync.WaitGroup
	for _, src := range sources {
		if src.url == "" {
			continue
		}
		wg.Add(1)
		go func(name, u string) {
			defer wg.Done()
			fetchCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
			defer cancel()

			data, err := f.fetchURL(fetchCtx, u)
			if err == nil && len(data) > 0 {
				mu.Lock()
				successes = append(successes, sourceData{name: name, data: data})
				mu.Unlock()
				stats.LogInfo("Nodes", "成功从 [%s] 拉取到节点数据 (%d 字节)", name, len(data))
			}
		}(src.name, src.url)
	}
	wg.Wait()

	if len(successes) > 0 {
		var combined bytes.Buffer
		combined.WriteString("*vpn_servers\n#HostName,IP,Score,Ping,Speed,CountryLong,CountryShort,NumVpnSessions,Uptime,TotalUsers,TotalTraffic,LogType,Operator,Message,OpenVPN_ConfigData_Base64\n")

		sourceNames := make([]string, 0, len(successes))
		seenIPs := make(map[string]bool)
		for _, s := range successes {
			sourceNames = append(sourceNames, s.name)
			lines := strings.Split(string(s.data), "\n")
			for _, l := range lines {
				trimmed := strings.TrimSpace(l)
				if trimmed == "" || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "#") {
					continue
				}
				parts := strings.SplitN(trimmed, ",", 3)
				if len(parts) >= 2 {
					ip := strings.TrimSpace(parts[1])
					if ip != "" && !seenIPs[ip] {
						seenIPs[ip] = true
						combined.WriteString(trimmed)
						combined.WriteString("\n")
					}
				}
			}
		}

		summarySource := strings.Join(sourceNames, " + ")
		if len(sourceNames) > 2 {
			summarySource = fmt.Sprintf("多源聚合 (%s 等 %d 个在线数据源)", sourceNames[0], len(sourceNames))
		}

		stats.LogInfo("Nodes", "多源聚合拉取完成，合并来自 %d 个数据源，总数据包: %d 字节", len(successes), combined.Len())
		return &FetchResult{
			Data:   combined.Bytes(),
			Source: summarySource,
		}, nil
	}

	// Fallback to local snapshot
	stats.LogInfo("Nodes", "所有网络源均不可用，尝试从本地快照恢复节点列表...")
	data, meta, err := f.snapshot.Load()
	if err == nil && len(data) > 0 {
		sourceName := "本地快照"
		if meta != nil && meta.Source != "" {
			sourceName = fmt.Sprintf("本地快照 (%s)", meta.Source)
		}
		stats.LogInfo("Nodes", "成功加载本地快照 (%d 字节)", len(data))
		return &FetchResult{
			Data:   data,
			Source: sourceName,
		}, nil
	}

	return nil, fmt.Errorf("所有在线镜像源及本地快照均不可用")
}
