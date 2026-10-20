package config

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	mu sync.RWMutex

	// API & Mirror sources
	ApiURL        string
	MirrorURL     string
	MirrorMetaURL string

	// Web UI settings
	UIHost     string
	UIPort     int
	UIPath     string
	UIUsername string
	UIPassword string
	SubToken   string // 专属随机安全订阅 Token / 路径

	// age 订阅安全加密
	AgeEncryptEnabled bool
	AgePublicKey      string

	// Proxy gateway settings
	ProxyHost                 string
	ProxyPort                 int
	ProxyUser                 string
	ProxyPass                 string
	ProxyMaxConnections       int
	ProxyGlobalMaxConnections int
	proxyActive               int
	proxyRejected             uint64

	// Intervals & Limits
	FetchInterval      time.Duration
	CheckInterval      time.Duration
	TargetValidNodes   int
	MaxScanRows        int
	InvalidBackoff     time.Duration
	DiscoveryCountries []string

	// 自动轮换策略
	AutoRotateMinutes int    // 0 = 禁用, > 0 轮换周期分钟
	AutoRotateIPType  string // all / residential / hosting

	// Telegram 机器人告警与远程交互
	TelegramBotToken string
	TelegramChatID   string

	// OpenVPN parameters
	OpenVPNCommand  string
	OpenVPNAuthUser string
	OpenVPNAuthPass string

	// Data directory
	DataDir string
}

type SettingsDTO struct {
	ProxyGlobalMaxConnections *int     `json:"proxy_global_max_connections,omitempty"`
	UIPort                    int      `json:"ui_port"`
	UIPath                    string   `json:"ui_path"`
	UIUsername                string   `json:"ui_username"`
	UIPassword                string   `json:"ui_password,omitempty"`
	SubToken                  string   `json:"sub_token,omitempty"`
	AgeEncryptEnabled         bool     `json:"age_encrypt_enabled"`
	AgePublicKey              string   `json:"age_public_key"`
	ProxyPort                 int      `json:"proxy_port"`
	ProxyUser                 string   `json:"proxy_user"`
	ProxyPass                 string   `json:"proxy_pass,omitempty"`
	AutoRotateMinutes         int      `json:"auto_rotate_minutes"`
	AutoRotateIPType          string   `json:"auto_rotate_ip_type"`
	DiscoveryCountries        []string `json:"discovery_countries"`
	TelegramBotToken          string   `json:"telegram_bot_token"`
	TelegramChatID            string   `json:"telegram_chat_id"`
}

func getEnv(key, defaultVal string, fallbackKeys ...string) string {
	val := os.Getenv(key)
	if val != "" {
		return strings.TrimSpace(val)
	}
	for _, fb := range fallbackKeys {
		if fbVal := os.Getenv(fb); fbVal != "" {
			return strings.TrimSpace(fbVal)
		}
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal, minVal, maxVal int, fallbackKeys ...string) int {
	raw := os.Getenv(key)
	if raw == "" {
		for _, fb := range fallbackKeys {
			if fbVal := os.Getenv(fb); fbVal != "" {
				raw = fbVal
				break
			}
		}
	}
	if raw == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return defaultVal
	}
	if minVal != 0 || maxVal != 0 {
		if val < minVal {
			return minVal
		}
		if val > maxVal {
			return maxVal
		}
	}
	return val
}

func randomSecretPath() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "nxgate"
	}
	return hex.EncodeToString(b)
}

func randomAlphaNumeric(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "proxy" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

func LoadConfig() *Config {
	dataDir := getEnv("DATA_DIR", "data")
	_ = os.MkdirAll(dataDir, 0700)
	// #nosec G302 -- 0700 is restrictive for a directory that contains config and credential files.
	_ = os.Chmod(dataDir, 0700)

	// Persisted UI settings take precedence over bootstrap environment values.
	var persisted *SettingsDTO
	saved, err := os.ReadFile(filepath.Join(dataDir, "settings.json"))
	if err == nil {
		var dto SettingsDTO
		if err := json.Unmarshal(saved, &dto); err != nil {
			panic(fmt.Errorf("invalid persisted settings: %w", err))
		}
		if strings.TrimSpace(dto.UIUsername) == "" || strings.TrimSpace(dto.UIPassword) == "" {
			panic("persisted settings must contain nonempty UI credentials")
		}
		persisted = &dto
	} else if !os.IsNotExist(err) {
		panic(fmt.Errorf("read persisted settings: %w", err))
	}
	uiPassword := ""
	if persisted != nil {
		uiPassword = persisted.UIPassword
	} else {
		uiPassword = initialUIPassword(dataDir)
	}

	uiPath := getEnv("UI_PATH", "")
	if uiPath == "" {
		uiPath = getEnv("ADMIN_SECRET_PATH", "")
	}
	if uiPath == "" {
		// Check if saved before
		secretFile := filepath.Join(dataDir, "admin_path.txt")
		// #nosec G304 -- path is rooted in the operator-controlled data directory.
		if data, err := os.ReadFile(secretFile); err == nil && len(strings.TrimSpace(string(data))) > 0 {
			uiPath = strings.TrimSpace(string(data))
		} else {
			uiPath = randomSecretPath()
			_ = os.WriteFile(secretFile, []byte(uiPath), 0600)
			_ = os.Chmod(secretFile, 0600)
		}
	}
	uiPath = strings.Trim(uiPath, "/")

	subToken := getEnv("SUB_TOKEN", "")
	if subToken == "" {
		subTokenFile := filepath.Join(dataDir, "sub_token.txt")
		// #nosec G304 -- path is rooted in the operator-controlled data directory.
		if data, err := os.ReadFile(subTokenFile); err == nil && len(strings.TrimSpace(string(data))) > 0 {
			subToken = strings.TrimSpace(string(data))
		} else {
			// Check /etc/sing-box/sub.json if token exists
			if sbData, err := os.ReadFile("/etc/sing-box/sub.json"); err == nil {
				var sbSub struct {
					Token string `json:"token"`
				}
				if json.Unmarshal(sbData, &sbSub) == nil && sbSub.Token != "" {
					subToken = strings.TrimSpace(sbSub.Token)
				}
			}
			if subToken == "" {
				subToken = randomSecretPath()
			}
			_ = os.WriteFile(subTokenFile, []byte(subToken), 0600)
			_ = os.Chmod(subTokenFile, 0600)
		}
	}
	subToken = strings.Trim(subToken, "/")

	ageEncryptEnabled := getEnv("AGE_ENCRYPT_ENABLED", "false") == "true" || getEnv("AGE_ENCRYPT_ENABLED", "0") == "1"
	agePublicKey := getEnv("AGE_PUBLIC_KEY", "")
	if agePublicKey == "" {
		ageKeyFile := filepath.Join(dataDir, "age_public_key.txt")
		// #nosec G304 -- path is rooted in the operator-controlled data directory.
		if data, err := os.ReadFile(ageKeyFile); err == nil {
			agePublicKey = strings.TrimSpace(string(data))
		}
	}

	countriesRaw := getEnv("DISCOVERY_COUNTRIES", "")
	var countries []string
	if countriesRaw != "" {
		for _, c := range strings.Split(countriesRaw, ",") {
			c = strings.ToUpper(strings.TrimSpace(c))
			if len(c) == 2 {
				countries = append(countries, c)
			}
		}
	}

	proxyUser := getEnv("LOCAL_PROXY_USER", "", "PROXY_USER")
	proxyPass := getEnv("LOCAL_PROXY_PASS", "", "PROXY_PASS")
	proxyAuthFile := filepath.Join(dataDir, "proxy_auth.txt")
	if proxyUser == "" || proxyPass == "" {
		if data, err := os.ReadFile(proxyAuthFile); err == nil {
			parts := strings.Split(strings.TrimSpace(string(data)), ":")
			if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
				if proxyUser == "" {
					proxyUser = parts[0]
				}
				if proxyPass == "" {
					proxyPass = parts[1]
				}
			}
		}
	}
	if proxyUser == "" {
		proxyUser = "vpn_" + randomAlphaNumeric(6)
	}
	if proxyPass == "" {
		proxyPass = randomAlphaNumeric(16)
	}
	_ = os.WriteFile(proxyAuthFile, []byte(proxyUser+":"+proxyPass), 0600)
	_ = os.Chmod(proxyAuthFile, 0600)

	cfg := &Config{
		ApiURL:        getEnv("VPNGATE_API_HTTPS_URL", "https://www.vpngate.net/api/iphone/"),
		MirrorURL:     getEnv("VPNGATE_MIRROR_HTTPS_URL", "https://baoweise-bot.github.io/aimili-vpngate/vpngate.csv"),
		MirrorMetaURL: getEnv("VPNGATE_MIRROR_META_URL", "https://baoweise-bot.github.io/aimili-vpngate/vpngate.meta.json"),

		UIHost:            getEnv("UI_HOST", "127.0.0.1"),
		UIPort:            getEnvInt("UI_PORT", 8787, 1, 65535),
		UIPath:            uiPath,
		UIUsername:        getEnv("UI_USERNAME", "admin"),
		UIPassword:        uiPassword,
		SubToken:          subToken,
		AgeEncryptEnabled: ageEncryptEnabled,
		AgePublicKey:      agePublicKey,

		ProxyHost:                 getEnv("LOCAL_PROXY_HOST", "127.0.0.1", "PROXY_HOST"),
		ProxyPort:                 getEnvInt("LOCAL_PROXY_PORT", 7928, 1, 65535, "PROXY_PORT"),
		ProxyUser:                 proxyUser,
		ProxyPass:                 proxyPass,
		ProxyGlobalMaxConnections: getEnvInt("LOCAL_PROXY_GLOBAL_MAX_CONNECTIONS", DefaultProxyGlobalMaxConnections, 1, MaxProxyGlobalMaxConnections),
		ProxyMaxConnections:       getEnvInt("LOCAL_PROXY_MAX_CONNECTIONS", 512, 16, 4096, "PROXY_MAX_CONNECTIONS"),

		FetchInterval: func() time.Duration {
			fetchSec := getEnvInt("FETCH_INTERVAL_SECONDS", 0, 60, 86400)
			if fetchSec == 0 {
				fetchMin := getEnvInt("FETCH_INTERVAL_MINUTES", 15, 1, 1440)
				fetchSec = fetchMin * 60
			}
			return time.Duration(fetchSec) * time.Second
		}(),
		CheckInterval:      time.Duration(getEnvInt("CHECK_INTERVAL_SECONDS", 20, 5, 300)) * time.Second,
		TargetValidNodes:   getEnvInt("TARGET_VALID_NODES", 5, 1, 50),
		MaxScanRows:        getEnvInt("MAX_SCAN_ROWS", 1000, 10, 5000),
		InvalidBackoff:     time.Duration(getEnvInt("INVALID_BACKOFF_SECONDS", 1800, 60, 86400)) * time.Second,
		DiscoveryCountries: countries,

		AutoRotateMinutes: getEnvInt("AUTO_ROTATE_INTERVAL_MINUTES", 0, 0, 1440),
		AutoRotateIPType:  getEnv("AUTO_ROTATE_IP_TYPE", "all"),

		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:   getEnv("TELEGRAM_CHAT_ID", ""),

		OpenVPNCommand:  getEnv("OPENVPN_CMD", "openvpn", "OPENVPN_COMMAND"),
		OpenVPNAuthUser: getEnv("OPENVPN_AUTH_USER", "vpn"),
		OpenVPNAuthPass: getEnv("OPENVPN_AUTH_PASS", "vpn"),

		DataDir: dataDir,
	}
	if persisted != nil {
		cfg.applySettingsLocked(*persisted)
	}
	return cfg

}

func (c *Config) GetSettings() SettingsDTO {
	c.mu.RLock()
	defer c.mu.RUnlock()

	dto := c.settingsLocked()
	dto.UIPassword = "" // never expose the management password through GET/backup
	return dto
}

func (c *Config) settingsLocked() SettingsDTO {
	limit := c.ProxyGlobalMaxConnections
	if limit <= 0 {
		limit = DefaultProxyGlobalMaxConnections
	}
	return SettingsDTO{
		ProxyGlobalMaxConnections: &limit,
		UIPort:                    c.UIPort, UIPath: c.UIPath, UIUsername: c.UIUsername, UIPassword: c.UIPassword,
		SubToken: c.SubToken, AgeEncryptEnabled: c.AgeEncryptEnabled, AgePublicKey: c.AgePublicKey,
		ProxyPort: c.ProxyPort, ProxyUser: c.ProxyUser, ProxyPass: c.ProxyPass,
		AutoRotateMinutes: c.AutoRotateMinutes, AutoRotateIPType: c.AutoRotateIPType,
		DiscoveryCountries: append([]string(nil), c.DiscoveryCountries...),
		TelegramBotToken:   c.TelegramBotToken, TelegramChatID: c.TelegramChatID,
	}
}

func (c *Config) GetSubscriptionToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.SubToken != "" {
		return c.SubToken
	}
	if c.UIPath != "" {
		return c.UIPath
	}
	return ""
}

func (c *Config) GetProxyCredentials() (string, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ProxyUser, c.ProxyPass
}

func (c *Config) GetUICredentials() (string, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.UIUsername, c.UIPassword
}

func (c *Config) VerifyUICredentials(user, pass string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.UIUsername == "" && c.UIPassword == "" {
		return true
	}

	userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(c.UIUsername)) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(c.UIPassword)) == 1
	return userMatch && passMatch
}

func (c *Config) IsUIAuthEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.UIUsername != "" || c.UIPassword != ""
}

func (c *Config) UpdateSettings(dto SettingsDTO) error {
	if dto.ProxyGlobalMaxConnections != nil && (*dto.ProxyGlobalMaxConnections < 1 || *dto.ProxyGlobalMaxConnections > MaxProxyGlobalMaxConnections) {
		return fmt.Errorf("proxy_global_max_connections must be 1..%d", MaxProxyGlobalMaxConnections)
	}
	for name, value := range map[string]string{
		"ui_path":             dto.UIPath,
		"ui_username":         dto.UIUsername,
		"ui_password":         dto.UIPassword,
		"sub_token":           dto.SubToken,
		"proxy_user":          dto.ProxyUser,
		"proxy_pass":          dto.ProxyPass,
		"telegram_bot_token":  dto.TelegramBotToken,
		"telegram_chat_id":    dto.TelegramChatID,
		"age_public_key":      dto.AgePublicKey,
		"auto_rotate_ip_type": dto.AutoRotateIPType,
	} {
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("%s 不能包含换行符", name)
		}
	}
	for _, country := range dto.DiscoveryCountries {
		if strings.ContainsAny(country, "\r\n") {
			return fmt.Errorf("discovery_countries 不能包含换行符")
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	pending := &Config{}
	pending.applySettingsLocked(c.settingsLocked())
	pending.applySettingsLocked(dto)
	data, err := json.MarshalIndent(pending.settingsLocked(), "", "  ")
	if err == nil {
		err = atomicSettingsWrite(filepath.Join(c.DataDir, "settings.json"), data)
	}
	if err != nil {
		return fmt.Errorf("persist settings: %w", err)
	}
	c.applySettingsLocked(dto)
	return nil
}

func (c *Config) applySettingsLocked(dto SettingsDTO) {
	if dto.ProxyGlobalMaxConnections != nil {
		if *dto.ProxyGlobalMaxConnections < 1 || *dto.ProxyGlobalMaxConnections > MaxProxyGlobalMaxConnections {
			panic("invalid persisted proxy global budget")
		}
		c.ProxyGlobalMaxConnections = *dto.ProxyGlobalMaxConnections
	}
	if dto.UIPort > 0 && dto.UIPort < 65536 {
		c.UIPort = dto.UIPort
	}
	if dto.ProxyPort > 0 && dto.ProxyPort < 65536 {
		c.ProxyPort = dto.ProxyPort
	}
	if strings.TrimSpace(dto.UIPath) != "" {
		c.UIPath = strings.Trim(strings.TrimSpace(dto.UIPath), "/")
	}
	if strings.TrimSpace(dto.SubToken) != "" {
		c.SubToken = strings.Trim(strings.TrimSpace(dto.SubToken), "/")
	}
	if strings.TrimSpace(dto.UIUsername) != "" {
		c.UIUsername = strings.TrimSpace(dto.UIUsername)
	}
	if strings.TrimSpace(dto.UIPassword) != "" {
		c.UIPassword = strings.TrimSpace(dto.UIPassword)
	}
	if dto.ProxyUser != "" {
		c.ProxyUser = strings.TrimSpace(dto.ProxyUser)
	}
	if dto.ProxyPass != "" {
		c.ProxyPass = strings.TrimSpace(dto.ProxyPass)
	}
	if dto.AutoRotateMinutes >= 0 {
		c.AutoRotateMinutes = dto.AutoRotateMinutes
	}
	if dto.AutoRotateIPType != "" {
		c.AutoRotateIPType = dto.AutoRotateIPType
	}
	if dto.DiscoveryCountries != nil {
		c.DiscoveryCountries = dto.DiscoveryCountries
	}
	c.TelegramBotToken = strings.TrimSpace(dto.TelegramBotToken)
	c.TelegramChatID = strings.TrimSpace(dto.TelegramChatID)
	c.AgeEncryptEnabled = dto.AgeEncryptEnabled
	c.AgePublicKey = strings.TrimSpace(dto.AgePublicKey)
}

// Atomic rename avoids truncated credentials on interruption. Never fall back to defaults on read errors.
func atomicSettingsWrite(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".settings-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}

func initialUIPassword(dataDir string) string {
	if pass := getEnv("UI_PASSWORD", ""); pass != "" {
		return pass
	}
	path := filepath.Join(dataDir, "ui_password.txt")
	if data, err := os.ReadFile(path); err == nil {
		if pass := strings.TrimSpace(string(data)); pass != "" {
			return pass
		}
		panic("empty persisted UI password")
	} else if !os.IsNotExist(err) {
		panic(fmt.Errorf("read UI password: %w", err))
	}
	secret := make([]byte, 24)
	if _, err := rand.Read(secret); err != nil {
		panic(fmt.Errorf("generate UI password: %w", err))
	}
	pass := hex.EncodeToString(secret)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		return initialUIPassword(dataDir)
	}
	if err != nil {
		panic(fmt.Errorf("save UI password: %w", err))
	}
	_, err = f.WriteString(pass)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		panic(err)
	}
	if closeErr != nil {
		panic(closeErr)
	}
	return pass
}
