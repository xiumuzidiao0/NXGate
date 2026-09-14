package tunnel

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"aimili-vpngate-go/pkg/nodes"
	"aimili-vpngate-go/pkg/stats"
)

type ServiceUnlockStatus = nodes.ServiceUnlockStatus

const (
	StatusUnlocked = nodes.StatusUnlocked
	StatusBlocked  = nodes.StatusBlocked
	StatusUnknown  = nodes.StatusUnknown
)

type UnlockResult = nodes.UnlockResult

type UnlockDetector struct {
	cachePath string
	mu        sync.RWMutex
	cache     map[string]*UnlockResult
}

func NewUnlockDetector(dataDir string) *UnlockDetector {
	d := &UnlockDetector{
		cachePath: filepath.Join(dataDir, "unlock_cache.json"),
		cache:     make(map[string]*UnlockResult),
	}
	d.load()
	return d
}

func (d *UnlockDetector) load() {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := os.ReadFile(d.cachePath)
	if err != nil {
		return
	}

	var raw map[string]*UnlockResult
	if err := json.Unmarshal(data, &raw); err == nil {
		now := time.Now()
		cleaned := make(map[string]*UnlockResult)
		for ip, res := range raw {
			// 12-hour TTL
			if res != nil && now.Sub(res.CheckedAt) < 12*time.Hour {
				cleaned[ip] = res
			}
		}
		d.cache = cleaned
	}
}

func (d *UnlockDetector) saveLocked() {
	data, err := json.MarshalIndent(d.cache, "", "  ")
	if err != nil {
		return
	}
	tmp := d.cachePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err == nil {
		_ = os.Chmod(tmp, 0600)
		_ = os.Rename(tmp, d.cachePath)
	}
}

func (d *UnlockDetector) GetUnlock(ip string) *UnlockResult {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if res, ok := d.cache[ip]; ok && time.Since(res.CheckedAt) < 12*time.Hour {
		cp := *res
		return &cp
	}
	return nil
}

func (d *UnlockDetector) GetAllCached() map[string]*UnlockResult {
	d.mu.RLock()
	defer d.mu.RUnlock()

	res := make(map[string]*UnlockResult, len(d.cache))
	for ip, r := range d.cache {
		cp := *r
		res[ip] = &cp
	}
	return res
}

var openAISupportedCountries = map[string]bool{
	"US": true, "JP": true, "KR": true, "TW": true, "SG": true, "GB": true,
	"DE": true, "FR": true, "CA": true, "AU": true, "NL": true, "IN": true,
	"VN": true, "TH": true, "MY": true, "PH": true, "ID": true, "BR": true,
	"IT": true, "ES": true, "SE": true, "NO": true, "FI": true, "PL": true,
	"CH": true, "AT": true, "BE": true, "DK": true, "NZ": true, "IE": true,
}

var claudeSupportedCountries = map[string]bool{
	"US": true, "JP": true, "KR": true, "TW": true, "SG": true, "GB": true,
	"DE": true, "FR": true, "CA": true, "AU": true, "NL": true, "IN": true,
	"NZ": true, "IE": true, "CH": true, "AT": true, "BE": true, "DK": true,
	"NO": true, "SE": true, "FI": true, "PL": true, "IT": true, "ES": true,
	"BR": true, "IL": true, "AE": true, "ZA": true,
}

// EvaluateNodeUnlock evaluates unlock prediction for a node and caches it
func (d *UnlockDetector) EvaluateNodeUnlock(node *nodes.Node) *UnlockResult {
	if node == nil || node.IP == "" {
		return nil
	}

	d.mu.RLock()
	if cached, ok := d.cache[node.IP]; ok && time.Since(cached.CheckedAt) < 12*time.Hour {
		d.mu.RUnlock()
		node.Unlock = cached
		return cached
	}
	d.mu.RUnlock()

	res := &UnlockResult{
		IP:            node.IP,
		OpenAI:        StatusUnknown,
		Claude:        StatusUnknown,
		Gemini:        StatusUnknown,
		Google:        StatusUnknown,
		Netflix:       StatusUnknown,
		NetflixRegion: node.CountryShort,
		IsProbed:      false,
		CheckedAt:     time.Now(),
	}

	c := strings.ToUpper(strings.TrimSpace(node.CountryShort))
	isOpenAIOk := openAISupportedCountries[c]
	isClaudeOk := claudeSupportedCountries[c]
	isGeminiOk := c != "CN" && c != "HK" && c != "IR" && c != "KP" && c != "RU"
	isGoogleOk := c != "CN" && c != "IR" && c != "KP"
	isNetflixOk := c != "" && c != "CN"

	if node.LatencyMs > 0 {
		if node.IPType == "residential" {
			// 住宅宽带初步推测：在官方支持国家内标记为预测可用
			if isOpenAIOk {
				res.OpenAI = StatusUnlocked
			} else {
				res.OpenAI = StatusBlocked
			}

			if isClaudeOk {
				res.Claude = StatusUnlocked
			} else {
				res.Claude = StatusBlocked
			}

			if isGeminiOk {
				res.Gemini = StatusUnlocked
			} else {
				res.Gemini = StatusBlocked
			}

			if isGoogleOk {
				res.Google = StatusUnlocked
			} else {
				res.Google = StatusBlocked
			}

			if isNetflixOk {
				res.Netflix = StatusUnlocked
			} else {
				res.Netflix = StatusBlocked
			}
		} else if node.IPType == "hosting" {
			// 数据中心机房IP
			if isGoogleOk {
				res.Google = StatusUnlocked
			}
			res.Netflix = StatusBlocked
			res.Claude = StatusBlocked
			if isOpenAIOk && node.ReputationScore >= 70 {
				res.OpenAI = StatusUnlocked
			} else {
				res.OpenAI = StatusBlocked
			}
			if isGeminiOk && node.ReputationScore >= 70 {
				res.Gemini = StatusUnlocked
			} else {
				res.Gemini = StatusBlocked
			}
		} else {
			if isOpenAIOk {
				res.OpenAI = StatusUnlocked
			}
			if isClaudeOk {
				res.Claude = StatusUnlocked
			}
			if isGeminiOk {
				res.Gemini = StatusUnlocked
			}
			if isGoogleOk {
				res.Google = StatusUnlocked
			}
			if isNetflixOk {
				res.Netflix = StatusUnlocked
			}
		}
	}

	node.Unlock = res
	return res
}

func (d *UnlockDetector) ProbeTunnel(ctx context.Context, devName string, ip string) *UnlockResult {
	result := &UnlockResult{
		IP:        ip,
		OpenAI:    StatusUnknown,
		Claude:    StatusUnknown,
		Gemini:    StatusUnknown,
		Google:    StatusUnknown,
		Netflix:   StatusUnknown,
		IsProbed:  true,
		CheckedAt: time.Now(),
	}

	client := newTunnelHTTPClient(devName, 6*time.Second)

	var wg sync.WaitGroup
	wg.Add(5)

	// 1. OpenAI / ChatGPT
	go func() {
		defer wg.Done()
		req, err := http.NewRequestWithContext(ctx, "GET", "https://ios.chat.openai.com/public-api/mobile/server_status/v1", nil)
		if err == nil {
			req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)")
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 401 {
					body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
					if !strings.Contains(string(body), "blocked") && !strings.Contains(string(body), "Cloudflare") {
						result.OpenAI = StatusUnlocked
						return
					}
				}
				if resp.StatusCode == 403 {
					result.OpenAI = StatusBlocked
					return
				}
			}
		}

		// Fallback check to cdn-cgi trace
		reqTrace, errTrace := http.NewRequestWithContext(ctx, "GET", "https://chatgpt.com/cdn-cgi/trace", nil)
		if errTrace == nil {
			resp, err := client.Do(reqTrace)
			if err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
				if resp.StatusCode == 200 && strings.Contains(string(body), "loc=") && !strings.Contains(string(body), "loc=CN") && !strings.Contains(string(body), "loc=HK") {
					result.OpenAI = StatusUnlocked
					return
				}
				if resp.StatusCode == 403 {
					result.OpenAI = StatusBlocked
					return
				}
			}
		}
	}()

	// 2. Claude / Anthropic
	go func() {
		defer wg.Done()
		// 优先探测 Anthropic 核心消息路由接口 (Anthropic API / Claude 客户端公共路由)
		// 正常允许访问的非封锁 IP 返回 HTTP 401 并携带 authentication_error
		// 处于封锁地区或硬封禁的 IP 返回 HTTP 403 或 permission_error
		req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", strings.NewReader(`{}`))
		if err == nil {
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("anthropic-version", "2023-06-01")
			req.Header.Set("x-api-key", "ant-api-probe-check")
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
				bodyStr := string(body)
				if resp.StatusCode == 401 && strings.Contains(bodyStr, "authentication_error") {
					result.Claude = StatusUnlocked
					return
				}
				if resp.StatusCode == 403 || strings.Contains(bodyStr, "permission_error") || strings.Contains(bodyStr, "not available in your region") {
					result.Claude = StatusBlocked
					return
				}
			}
		}

		// 备用探测：获取 Claude Cloudflare Edge 真实出口归属
		reqTrace, errTrace := http.NewRequestWithContext(ctx, "GET", "https://claude.ai/cdn-cgi/trace", nil)
		if errTrace == nil {
			reqTrace.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
			resp, err := client.Do(reqTrace)
			if err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
				bodyStr := string(body)
				if resp.StatusCode == 200 && strings.Contains(bodyStr, "loc=") {
					for _, line := range strings.Split(bodyStr, "\n") {
						if strings.HasPrefix(line, "loc=") {
							loc := strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(line, "loc=")))
							if claudeSupportedCountries[loc] {
								result.Claude = StatusUnlocked
								return
							} else {
								result.Claude = StatusBlocked
								return
							}
						}
					}
				}
			}
		}

		result.Claude = StatusBlocked
	}()

	// 3. Google Gemini
	go func() {
		defer wg.Done()
		req, err := http.NewRequestWithContext(ctx, "GET", "https://gemini.google.com/", nil)
		if err == nil {
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					result.Gemini = StatusUnlocked
				} else if resp.StatusCode == 403 || resp.StatusCode == 429 {
					result.Gemini = StatusBlocked
				}
				return
			}
		}
		result.Gemini = StatusBlocked
	}()

	// 4. Google Search (Generate 204)
	go func() {
		defer wg.Done()
		req, err := http.NewRequestWithContext(ctx, "GET", "https://www.google.com/generate_204", nil)
		if err == nil {
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == 204 {
					result.Google = StatusUnlocked
				} else if resp.StatusCode == 429 {
					result.Google = StatusBlocked
				}
			}
		}
	}()

	// 5. Netflix
	go func() {
		defer wg.Done()
		req, err := http.NewRequestWithContext(ctx, "GET", "https://www.netflix.com/title/81280792", nil)
		if err == nil {
			req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					result.Netflix = StatusUnlocked
				} else if resp.StatusCode == 403 {
					result.Netflix = StatusBlocked
				}
			}
		}
	}()

	wg.Wait()

	// Update cache with real physical probe result
	d.mu.Lock()
	d.cache[ip] = result
	d.saveLocked()
	d.mu.Unlock()

	stats.LogInfo("UnlockDetector", "[%s:%s] 实测解锁结果: ChatGPT=%s, Claude=%s, Gemini=%s, Google=%s, Netflix=%s",
		devName, ip, result.OpenAI, result.Claude, result.Gemini, result.Google, result.Netflix)

	return result
}
