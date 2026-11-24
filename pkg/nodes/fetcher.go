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
	req.Header.Set("User-Agent", "NXGate/2.0")

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
	body, err = DecompressSnapshotCSV(body)
	if err != nil {
		return nil, err
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

	type fetchSource struct {
		name   string
		url    string
		family string
	}

	rawSources := []fetchSource{
		{name: "NXGate 主镜像源", url: f.mirrorURL, family: "mirror"},
		{name: "VPNGate 官方 HTTPS API", url: f.apiURL, family: "official"},
		{name: "VPNGate 官方 HTTP API", url: "http://www.vpngate.net/api/iphone/", family: "official"},
		{name: "NXGate 实时镜像 (Data 分支)", url: "https://raw.githubusercontent.com/xiumuzidiao0/NXGate/data/mirror/vpngate.csv", family: "mirror"},
		{name: "NXGate 高速压缩镜像 (Gzip)", url: "https://raw.githubusercontent.com/xiumuzidiao0/NXGate/data/mirror/vpngate.csv.gz", family: "mirror"},
		{name: "Fastly 全球加速 CDN", url: "https://fastly.jsdelivr.net/gh/xiumuzidiao0/NXGate@data/mirror/vpngate.csv", family: "mirror"},
		{name: "jsDelivr 全球加速 CDN", url: "https://cdn.jsdelivr.net/gh/xiumuzidiao0/NXGate@data/mirror/vpngate.csv", family: "mirror"},
		{name: "GitHub 镜像加速源", url: "https://ghproxy.net/https://raw.githubusercontent.com/xiumuzidiao0/NXGate/data/mirror/vpngate.csv", family: "mirror"},
	}

	// Deduplicate identical URLs so a default mirrorURL never fetches the same feed twice.
	seenURLs := make(map[string]bool, len(rawSources))
	sources := make([]fetchSource, 0, len(rawSources))
	activeFamilies := make(map[string]bool, 2)
	for _, src := range rawSources {
		if src.url == "" || seenURLs[src.url] {
			continue
		}
		seenURLs[src.url] = true
		activeFamilies[src.family] = true
		sources = append(sources, src)
	}

	type sourceData struct {
		name string
		data []byte
	}

	var mu sync.Mutex
	familySuccess := make(map[string]*sourceData, 2)

	fetchCtx, cancelAll := context.WithCancel(ctx)
	defer cancelAll()

	officialCtx, officialCancel := context.WithCancel(fetchCtx)
	defer officialCancel()
	mirrorCtx, mirrorCancel := context.WithCancel(fetchCtx)
	defer mirrorCancel()

	var triggerEarlyExitOnce sync.Once
	doneCh := make(chan struct{})

	var wg sync.WaitGroup
	for _, src := range sources {
		wg.Add(1)
		go func(s fetchSource) {
			defer wg.Done()
			famCtx := mirrorCtx
			famCancel := mirrorCancel
			if s.family == "official" {
				famCtx = officialCtx
				famCancel = officialCancel
			}

			reqCtx, reqCancel := context.WithTimeout(famCtx, 12*time.Second)
			defer reqCancel()

			data, err := f.fetchURL(reqCtx, s.url)
			if err != nil || len(data) == 0 {
				return
			}

			mu.Lock()
			if _, already := familySuccess[s.family]; already {
				mu.Unlock()
				return
			}
			familySuccess[s.family] = &sourceData{name: s.name, data: data}
			// First win per source family cancels sibling mirrors immediately.
			famCancel()
			wonCount := len(familySuccess)
			allWon := wonCount >= len(activeFamilies)
			mu.Unlock()

			stats.LogInfo("Nodes", "成功从 [%s] 拉取到节点数据 (%d 字节)", s.name, len(data))

			if allWon {
				cancelAll()
				return
			}

			// One family succeeded; allow a short 1.5s grace window for the other family before early exit.
			triggerEarlyExitOnce.Do(func() {
				go func() {
					timer := time.NewTimer(1500 * time.Millisecond)
					defer timer.Stop()
					select {
					case <-timer.C:
						cancelAll()
					case <-doneCh:
					}
				}()
			})
		}(src)
	}

	go func() {
		wg.Wait()
		close(doneCh)
	}()

	<-doneCh

	if err := ctx.Err(); err != nil && len(familySuccess) == 0 {
		return nil, err
	}

	if len(familySuccess) > 0 {
		var combined bytes.Buffer
		combined.WriteString("*vpn_servers\n#HostName,IP,Score,Ping,Speed,CountryLong,CountryShort,NumVpnSessions,Uptime,TotalUsers,TotalTraffic,LogType,Operator,Message,OpenVPN_ConfigData_Base64\n")

		sourceNames := make([]string, 0, len(familySuccess))
		seenIPs := make(map[string]bool)

		appendCSVLines := func(data []byte) {
			for len(data) > 0 {
				var line []byte
				if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
					line = data[:idx]
					data = data[idx+1:]
				} else {
					line = data
					data = nil
				}
				trimmed := bytes.TrimSpace(line)
				if len(trimmed) == 0 || trimmed[0] == '*' || trimmed[0] == '#' {
					continue
				}
				c1 := bytes.IndexByte(trimmed, ',')
				if c1 < 0 {
					continue
				}
				rest := trimmed[c1+1:]
				var ipRaw []byte
				if c2 := bytes.IndexByte(rest, ','); c2 >= 0 {
					ipRaw = rest[:c2]
				} else {
					ipRaw = rest
				}
				ipBytes := bytes.TrimSpace(ipRaw)
				if len(ipBytes) == 0 {
					continue
				}
				ip := string(ipBytes)
				if !seenIPs[ip] {
					seenIPs[ip] = true
					combined.Write(trimmed)
					combined.WriteByte('\n')
				}
			}
		}

		for _, fam := range []string{"mirror", "official"} {
			sd := familySuccess[fam]
			if sd == nil {
				continue
			}
			sourceNames = append(sourceNames, sd.name)
			appendCSVLines(sd.data)
			sd.data = nil
		}

		summarySource := strings.Join(sourceNames, " + ")
		if len(sourceNames) > 2 {
			summarySource = fmt.Sprintf("多源聚合 (%s 等 %d 个在线数据源)", sourceNames[0], len(sourceNames))
		}

		stats.LogInfo("Nodes", "多源聚合拉取完成，合并来自 %d 个数据源，总数据包: %d 字节", len(sourceNames), combined.Len())
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
