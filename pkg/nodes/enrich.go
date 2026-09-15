package nodes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"aimili-vpngate-go/pkg/stats"
)

type IPCacheEntry struct {
	IP            string    `json:"ip"`
	IPType        string    `json:"ip_type"` // residential, hosting, mobile, unknown
	ISP           string    `json:"isp"`
	City          string    `json:"city"`
	Region        string    `json:"region"`
	ASN           int       `json:"asn"`
	RDNS          string    `json:"rdns"`
	IsHosting     bool      `json:"is_hosting"`
	IsResidential bool      `json:"is_residential"`
	ResConfidence int       `json:"res_confidence"`
	ResReason     string    `json:"res_reason"`
	CachedAt      time.Time `json:"cached_at"`
}

type IPEnricher struct {
	mu               sync.RWMutex
	cachePath        string
	cache            map[string]*IPCacheEntry
	client           *http.Client
	residentialDetector *ResidentialDetector
}

func NewIPEnricher(dataDir string) *IPEnricher {
	e := &IPEnricher{
		cachePath: filepath.Join(dataDir, "ip_cache.json"),
		cache:     make(map[string]*IPCacheEntry),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		residentialDetector: NewResidentialDetector(),
	}
	e.loadCache()
	return e
}

func (e *IPEnricher) loadCache() {
	e.mu.Lock()
	defer e.mu.Unlock()

	data, err := os.ReadFile(e.cachePath)
	if err != nil {
		return
	}

	var raw map[string]*IPCacheEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}

	now := time.Now()
	cleaned := make(map[string]*IPCacheEntry)
	for ip, entry := range raw {
		// 7-day TTL
		if entry != nil && now.Sub(entry.CachedAt) < 7*24*time.Hour {
			cleaned[ip] = entry
		}
	}
	e.cache = cleaned
}

func (e *IPEnricher) saveCacheLocked() {
	data, err := json.MarshalIndent(e.cache, "", "  ")
	if err != nil {
		return
	}
	tmp := e.cachePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err == nil {
		_ = os.Chmod(tmp, 0600)
		_ = os.Rename(tmp, e.cachePath)
	}
}

type ipApiItem struct {
	Status     string `json:"status"`
	Query      string `json:"query"`
	Country    string `json:"country"`
	RegionName string `json:"regionName"`
	City       string `json:"city"`
	ISP        string `json:"isp"`
	Org        string `json:"org"`
	AS         string `json:"as"`       // "AS15169 Google LLC"
	Reverse    string `json:"reverse"`  // rDNS
	Hosting    bool   `json:"hosting"`
	Mobile     bool   `json:"mobile"`
	Proxy      bool   `json:"proxy"`
}

func (e *IPEnricher) EnrichNodes(ctx context.Context, nodeList []*Node) {
	if len(nodeList) == 0 {
		return
	}

	// 1. Check local cache first
	e.mu.RLock()
	var toQuery []string
	now := time.Now()
	for _, n := range nodeList {
		if entry, ok := e.cache[n.IP]; ok && now.Sub(entry.CachedAt) < 7*24*time.Hour {
			n.IPType = entry.IPType
			n.ISP = entry.ISP
			n.City = entry.City
			n.Region = entry.Region
			n.ASN = entry.ASN
			n.RDNS = entry.RDNS
			n.IsHosting = entry.IsHosting
			n.IsResidential = entry.IsResidential
			n.ResConfidence = entry.ResConfidence
			n.ResReason = entry.ResReason
		} else {
			toQuery = append(toQuery, n.IP)
		}
	}
	e.mu.RUnlock()

	if len(toQuery) == 0 {
		return
	}

	// 2. Batch query in chunks of 50 to ip-api.com
	chunkSize := 50
	newResults := make(map[string]*IPCacheEntry)

	for i := 0; i < len(toQuery); i += chunkSize {
		end := i + chunkSize
		if end > len(toQuery) {
			end = len(toQuery)
		}
		chunk := toQuery[i:end]

		reqBytes, err := json.Marshal(chunk)
		if err != nil {
			continue
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			"http://ip-api.com/batch?lang=zh-CN&fields=status,query,country,regionName,city,isp,org,as,reverse,hosting,mobile,proxy",
			bytes.NewReader(reqBytes))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "AimiliVPN-Go/2.0")

		resp, err := e.client.Do(req)
		if err != nil {
			stats.LogWarn("Enrich", "批量查询 IP 类型失败: %v", err)
			continue
		}

		var items []ipApiItem
		decodeErr := json.NewDecoder(resp.Body).Decode(&items)
		_ = resp.Body.Close()
		if decodeErr != nil {
			continue
		}

		for _, item := range items {
			if item.Status != "success" || item.Query == "" {
				continue
			}

			// 解析 ASN
			asn := 0
			if item.AS != "" {
				// AS format: "AS15169 Google LLC"
				var parsed int
				if n, _ := fmt.Sscanf(item.AS, "AS%d", &parsed); n == 1 {
					asn = parsed
				}
			}

			ispName := item.ISP
			if ispName == "" {
				ispName = item.Org
			}

			// 使用家宽甄选引擎进行五层判定
			classification := e.residentialDetector.Classify(item.Query, asn, ispName, item.Reverse)

			// 基于 ip-api.com 和甄选引擎综合判定
			ipType := "residential" // 默认居民住宅/家庭宽带
			isResidential := false
			if item.Hosting {
				ipType = "hosting" // 机房/数据中心（API 明确标记）
			} else if item.Mobile {
				ipType = "mobile" // 移动蜂窝
			} else if classification.Type == TypeResidential && classification.Confidence >= 60 {
				ipType = "residential"
				isResidential = true
			} else if classification.Type == TypeDatacenter && classification.Confidence >= 70 {
				ipType = "hosting"
			}

			entry := &IPCacheEntry{
				IP:            item.Query,
				IPType:        ipType,
				ISP:           ispName,
				City:          item.City,
				Region:        item.RegionName,
				ASN:           asn,
				RDNS:          item.Reverse,
				IsHosting:     item.Hosting,
				IsResidential: isResidential,
				ResConfidence: classification.Confidence,
				ResReason:     classification.Reason,
				CachedAt:      time.Now(),
			}
			newResults[item.Query] = entry
		}

		// Avoid flooding ip-api rate limits (45 req/min)
		time.Sleep(200 * time.Millisecond)
	}

	// 3. Save new entries to cache and update nodes
	e.mu.Lock()
	for ip, entry := range newResults {
		e.cache[ip] = entry
	}
	e.saveCacheLocked()
	e.mu.Unlock()

	// 4. Populate onto node instances
	for _, n := range nodeList {
		if entry, ok := newResults[n.IP]; ok {
			n.IPType = entry.IPType
			n.ISP = entry.ISP
			n.City = entry.City
			n.Region = entry.Region
			n.ASN = entry.ASN
			n.RDNS = entry.RDNS
			n.IsHosting = entry.IsHosting
			n.IsResidential = entry.IsResidential
			n.ResConfidence = entry.ResConfidence
			n.ResReason = entry.ResReason
		} else if n.IPType == "" {
			n.IPType = "unknown"
		}
	}

	stats.LogInfo("Enrich", "已完成 %d 个节点的 IP 类型探测与信息丰富化", len(newResults))
}
