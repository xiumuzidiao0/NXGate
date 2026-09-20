package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/stats"
	"aimili-vpngate-go/pkg/tunnel"
)

type CheckUpdateResponse struct {
	OK             bool   `json:"ok"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	HasUpdate      bool   `json:"has_update"`
	ReleaseName    string `json:"release_name,omitempty"`
	ReleaseNotes   string `json:"release_notes,omitempty"`
	ReleaseURL     string `json:"release_url,omitempty"`
	PublishedAt    string `json:"published_at,omitempty"`
	Error          string `json:"error,omitempty"`
}

type TriggerUpdateRequest struct {
	Force bool `json:"force"`
}

type TriggerUpdateResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Version string `json:"version,omitempty"`
	Error   string `json:"error,omitempty"`
}

type UpdateStatusResponse struct {
	InProgress bool   `json:"in_progress"`
	Step       string `json:"step"`
	Version    string `json:"version"`
	Error      string `json:"error,omitempty"`
}

var updateState = struct {
	sync.RWMutex
	inProgress bool
	step       string
	targetVer  string
	err        string
}{
	step: "就绪",
}

func setUpdateStatus(inProgress bool, step, targetVer, errMsg string) {
	updateState.Lock()
	defer updateState.Unlock()
	updateState.inProgress = inProgress
	updateState.step = step
	updateState.targetVer = targetVer
	updateState.err = errMsg
}

func getUpdateStatus() UpdateStatusResponse {
	updateState.RLock()
	defer updateState.RUnlock()
	return UpdateStatusResponse{
		InProgress: updateState.inProgress,
		Step:       updateState.step,
		Version:    updateState.targetVer,
		Error:      updateState.err,
	}
}

// GitHubReleaseInfo captures the relevant fields returned by GitHub releases API.
type GitHubReleaseInfo struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
}

func (s *Server) handleCheckUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	info, err := FetchLatestRelease(ctx)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(CheckUpdateResponse{
			OK:             false,
			CurrentVersion: config.Version,
			Error:          fmt.Sprintf("检查更新失败: %v", err),
		})
		return
	}

	cleanLatest := strings.TrimPrefix(strings.TrimSpace(info.TagName), "v")
	cleanCurrent := strings.TrimPrefix(strings.TrimSpace(config.Version), "v")

	hasUpdate := CompareVersions(cleanLatest, cleanCurrent) > 0

	_ = json.NewEncoder(w).Encode(CheckUpdateResponse{
		OK:             true,
		CurrentVersion: config.Version,
		LatestVersion:  cleanLatest,
		HasUpdate:      hasUpdate,
		ReleaseName:    info.Name,
		ReleaseNotes:   info.Body,
		ReleaseURL:     info.HTMLURL,
		PublishedAt:    info.PublishedAt,
	})
}

func (s *Server) handleGetUpdateStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(getUpdateStatus())
}

func (s *Server) handleTriggerUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req TriggerUpdateRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	updateState.Lock()
	if updateState.inProgress {
		updateState.Unlock()
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(TriggerUpdateResponse{
			OK:    false,
			Error: "更新正在进行中，请勿重复操作",
		})
		return
	}
	updateState.inProgress = true
	updateState.step = "正在检查最新版本信息..."
	updateState.err = ""
	updateState.Unlock()

	defer func() {
		updateState.Lock()
		updateState.inProgress = false
		updateState.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	setUpdateStatus(true, "正在获取最新版本信息...", "", "")
	info, err := FetchLatestRelease(ctx)
	if err != nil {
		setUpdateStatus(false, "获取版本失败", "", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(TriggerUpdateResponse{
			OK:    false,
			Error: fmt.Sprintf("获取最新版本失败: %v", err),
		})
		return
	}

	cleanLatest := strings.TrimPrefix(strings.TrimSpace(info.TagName), "v")
	cleanCurrent := strings.TrimPrefix(strings.TrimSpace(config.Version), "v")

	if !req.Force && CompareVersions(cleanLatest, cleanCurrent) <= 0 {
		setUpdateStatus(false, "已是最新版本", cleanLatest, "")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(TriggerUpdateResponse{
			OK:      false,
			Message: "当前已是最新版本，无需更新",
			Version: cleanLatest,
		})
		return
	}

	setUpdateStatus(true, fmt.Sprintf("正在拉取最新程序 (v%s)...", cleanLatest), cleanLatest, "")
	stats.LogInfo("Update", "开始执行在线自更新 -> 目标版本: v%s (当前: v%s)", cleanLatest, config.Version)

	if err := PerformSelfUpdate(ctx, cleanLatest); err != nil {
		setUpdateStatus(false, "更新失败", cleanLatest, err.Error())
		stats.LogError("Update", "自更新失败: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(TriggerUpdateResponse{
			OK:    false,
			Error: fmt.Sprintf("更新失败: %v", err),
		})
		return
	}

	setUpdateStatus(false, "更新完成，服务正在重启...", cleanLatest, "")
	stats.LogInfo("Update", "版本更新成功完成，正在通知 systemd 重启守护进程...")

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(TriggerUpdateResponse{
		OK:      true,
		Message: fmt.Sprintf("🎉 已成功升级至 v%s！网关守护进程正在平滑热重启...", cleanLatest),
		Version: cleanLatest,
	})

	// Graceful trigger of systemd or OpenRC service restart
	go func() {
		time.Sleep(1200 * time.Millisecond)
		if runtime.GOOS == "linux" {
			if _, err := exec.LookPath("systemctl"); err == nil {
				_ = exec.Command("systemctl", "restart", "aimilivpn").Run()
			} else if _, err := exec.LookPath("rc-service"); err == nil {
				_ = exec.Command("rc-service", "aimilivpn", "restart").Run()
			}
		}
	}()
}

// CompareVersions compares two semantic version strings (e.g. "2.5.6" and "2.5.7").
// Returns 1 if v1 > v2, -1 if v1 < v2, and 0 if equal.
func CompareVersions(v1, v2 string) int {
	clean1 := strings.TrimPrefix(strings.TrimSpace(v1), "v")
	clean2 := strings.TrimPrefix(strings.TrimSpace(v2), "v")

	parts1 := strings.Split(clean1, ".")
	parts2 := strings.Split(clean2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			n1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			n2, _ = strconv.Atoi(parts2[i])
		}
		if n1 > n2 {
			return 1
		}
		if n1 < n2 {
			return -1
		}
	}
	return 0
}

// FetchLatestRelease attempts to get release info from GitHub API, with fallback to raw VERSION file.
func FetchLatestRelease(ctx context.Context) (*GitHubReleaseInfo, error) {
	client := &http.Client{Timeout: 6 * time.Second}

	apiURLs := []string{
		"https://api.github.com/repos/xiumuzidiao0/NXGate/releases/latest",
	}

	for _, u := range apiURLs {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "NXGate-UpdateChecker")
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var info GitHubReleaseInfo
			if err := json.NewDecoder(resp.Body).Decode(&info); err == nil && info.TagName != "" {
				return &info, nil
			}
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	// Fallback to raw VERSION file
	versionURLs := []string{
		"https://raw.githubusercontent.com/xiumuzidiao0/NXGate/main/VERSION",
		"https://ghproxy.net/https://raw.githubusercontent.com/xiumuzidiao0/NXGate/main/VERSION",
		"https://mirror.ghproxy.com/https://raw.githubusercontent.com/xiumuzidiao0/NXGate/main/VERSION",
	}

	for _, u := range versionURLs {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			continue
		}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
			resp.Body.Close()
			tag := strings.TrimSpace(string(body))
			if tag != "" {
				return &GitHubReleaseInfo{
					TagName:     "v" + strings.TrimPrefix(tag, "v"),
					Name:        fmt.Sprintf("v%s 正式构建", tag),
					Body:        "已检测到远端主干最新稳定版本。",
					HTMLURL:     "https://github.com/xiumuzidiao0/NXGate/releases/latest",
					PublishedAt: time.Now().Format(time.RFC3339),
				}, nil
			}
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	return nil, errors.New("无法连接 GitHub 官方源及镜像加速节点")
}

// PerformSelfUpdate downloads the corresponding architecture release binary, verifies its SHA-256
// checksum and ELF header, atomically overwrites the target executable, and updates VERSION.
func PerformSelfUpdate(ctx context.Context, targetVer string) error {
	arch := runtime.GOARCH
	if arch == "" {
		arch = "amd64"
	}

	fileName := fmt.Sprintf("nxgate_linux_%s", arch)

	// 1. Fetch expected SHA256 (prioritize nxgate_*, fallback to aimilivpn_*)
	setUpdateStatus(true, "正在拉取 SHA256 校验和清单...", targetVer, "")
	expectedHash, err := fetchExpectedSHA256(ctx, targetVer, fileName)
	if err != nil {
		fallbackName := fmt.Sprintf("aimilivpn_linux_%s", arch)
		expectedHash, err = fetchExpectedSHA256(ctx, targetVer, fallbackName)
		if err != nil {
			return fmt.Errorf("获取 SHA256 校验清单失败: %w", err)
		}
		fileName = fallbackName
	}

	// 2. Download binary to tmp file
	setUpdateStatus(true, fmt.Sprintf("正在下载目标程序 (%s)...", fileName), targetVer, "")
	binData, err := downloadBinaryWithMirrors(ctx, targetVer, fileName)
	if err != nil {
		return fmt.Errorf("下载二进制可执行文件失败: %w", err)
	}

	// 3. Verify SHA256 hash
	hasher := sha256.New()
	hasher.Write(binData)
	actualHash := hex.EncodeToString(hasher.Sum(nil))

	if !strings.EqualFold(actualHash, expectedHash) {
		return fmt.Errorf("SHA-256 校验不匹配 (期望: %s, 实际: %s)", expectedHash, actualHash)
	}

	// 4. Verify ELF magic
	if len(binData) < 4 || !bytes.Equal(binData[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		return errors.New("下载产物非有效的 Linux ELF 二进制程序")
	}

	// 5. Determine target binary destination
	targetBin := resolveBinaryDestination()
	dir := filepath.Dir(targetBin)
	_ = os.MkdirAll(dir, 0755)

	tmpFile := targetBin + ".tmp"
	if err := os.WriteFile(tmpFile, binData, 0755); err != nil {
		return fmt.Errorf("写入临时升级文件失败: %w", err)
	}

	// 6. Atomic replacement
	if err := os.Rename(tmpFile, targetBin); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("原子替换新程序失败: %w", err)
	}
	_ = os.Chmod(targetBin, 0755)

	// 7. Update auxiliary files if /opt/aimilivpn exists
	installDir := "/opt/aimilivpn"
	if fi, err := os.Stat(installDir); err == nil && fi.IsDir() {
		_ = os.WriteFile(filepath.Join(installDir, "VERSION"), []byte(targetVer+"\n"), 0644)

		// Download latest install.sh quietly
		go func() {
			scriptCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if data, err := downloadHTTPText(scriptCtx, "https://raw.githubusercontent.com/xiumuzidiao0/NXGate/main/install.sh"); err == nil && len(data) > 0 {
				_ = os.WriteFile(filepath.Join(installDir, "install.sh"), []byte(data), 0755)
			}
		}()
	}

	// 8. Re-assert SSH rules protection
	tunnel.EnsureSSHPolicyRouting()

	return nil
}

func resolveBinaryDestination() string {
	// 1. If standard production directory /opt/nxgate exists, target it
	if fi, err := os.Stat("/opt/nxgate"); err == nil && fi.IsDir() {
		return "/opt/nxgate/nxgate"
	}
	// 2. If legacy production directory /opt/aimilivpn exists, target it
	if fi, err := os.Stat("/opt/aimilivpn"); err == nil && fi.IsDir() {
		return "/opt/aimilivpn/aimilivpn"
	}
	// 3. Otherwise use the currently running executable location, resolving any symlinks
	execPath, err := os.Executable()
	if err == nil && execPath != "" {
		if realPath, err := filepath.EvalSymlinks(execPath); err == nil && realPath != "" {
			return realPath
		}
		return execPath
	}
	return "/opt/nxgate/nxgate"
}

func fetchExpectedSHA256(ctx context.Context, targetVer, targetFileName string) (string, error) {
	urls := []string{
		fmt.Sprintf("https://github.com/xiumuzidiao0/NXGate/releases/download/v%s/SHA256SUMS.txt", targetVer),
		fmt.Sprintf("https://ghproxy.net/https://github.com/xiumuzidiao0/NXGate/releases/download/v%s/SHA256SUMS.txt", targetVer),
		fmt.Sprintf("https://mirror.ghproxy.com/https://github.com/xiumuzidiao0/NXGate/releases/download/v%s/SHA256SUMS.txt", targetVer),
		"https://github.com/xiumuzidiao0/NXGate/releases/latest/download/SHA256SUMS.txt",
		"https://ghproxy.net/https://github.com/xiumuzidiao0/NXGate/releases/latest/download/SHA256SUMS.txt",
	}

	for _, u := range urls {
		content, err := downloadHTTPText(ctx, u)
		if err == nil && content != "" {
			for _, line := range strings.Split(content, "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					var hash, name string
					if len(fields[0]) == 64 {
						hash = strings.ToLower(fields[0])
						name = filepath.Base(strings.TrimPrefix(fields[1], "*"))
					} else if len(fields[1]) == 64 {
						hash = strings.ToLower(fields[1])
						name = filepath.Base(strings.TrimPrefix(fields[0], "*"))
					}
					if name == targetFileName && len(hash) == 64 {
						return hash, nil
					}
				}
			}
		}
	}
	return "", errors.New("未能从任何镜像源解析到目标架构的 SHA-256 校验和")
}

func downloadBinaryWithMirrors(ctx context.Context, targetVer, targetFileName string) ([]byte, error) {
	urls := []string{
		fmt.Sprintf("https://github.com/xiumuzidiao0/NXGate/releases/download/v%s/%s", targetVer, targetFileName),
		fmt.Sprintf("https://ghproxy.net/https://github.com/xiumuzidiao0/NXGate/releases/download/v%s/%s", targetVer, targetFileName),
		fmt.Sprintf("https://mirror.ghproxy.com/https://github.com/xiumuzidiao0/NXGate/releases/download/v%s/%s", targetVer, targetFileName),
		fmt.Sprintf("https://github.com/xiumuzidiao0/NXGate/releases/latest/download/%s", targetFileName),
		fmt.Sprintf("https://ghproxy.net/https://github.com/xiumuzidiao0/NXGate/releases/latest/download/%s", targetFileName),
	}

	client := &http.Client{Timeout: 45 * time.Second}

	for _, u := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "NXGate-Updater")

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			data, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr == nil && len(data) > 1024*1024 { // binaries are >= 1MB
				return data, nil
			}
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	return nil, errors.New("所有镜像源下载均失败或文件大小异常")
}

func downloadHTTPText(ctx context.Context, targetURL string) (string, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "NXGate-Updater")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
	if err != nil {
		return "", err
	}
	return string(body), nil
}
