package server

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"aimili-vpngate-go/pkg/singbox"
)

// GenerateRawSubscription converts sing-box nodes into a slice of standard proxy URLs (e.g. vless://, hysteria2://, tuic://).
// It replaces any "auto" placeholder in the address or URL with the actual defaultServerHost.
func GenerateRawSubscription(nodes []singbox.Node, defaultServerHost string) []string {
	var urls []string

	for _, n := range nodes {
		u := strings.TrimSpace(n.URL)

		srv := strings.TrimSpace(n.Address)
		if srv == "" || srv == "0.0.0.0" || srv == "127.0.0.1" || srv == "localhost" || strings.EqualFold(srv, "auto") {
			srv = defaultServerHost
		}

		if u == "" {
			// Fallback: construct standard URL if n.URL is missing
			u = constructFallbackURL(n, srv)
		} else if srv != "" {
			// Replace @auto with the actual server host/IP
			u = strings.ReplaceAll(u, "@auto:", "@"+srv+":")
			u = strings.ReplaceAll(u, "@auto#", "@"+srv+"#")
			u = strings.ReplaceAll(u, "@auto?", "@"+srv+"?")
			u = strings.ReplaceAll(u, "-auto", "-"+srv)
		}

		if u != "" {
			urls = append(urls, u)
		}
	}

	return urls
}

// GenerateBase64Subscription converts sing-box nodes into standard Base64-encoded subscription text,
// compatible with Shadowrocket, v2rayN, Sing-box, Quantumult X, and other universal clients.
func GenerateBase64Subscription(nodes []singbox.Node, defaultServerHost string) string {
	urls := GenerateRawSubscription(nodes, defaultServerHost)
	if len(urls) == 0 {
		return ""
	}
	rawText := strings.Join(urls, "\n")
	return base64.StdEncoding.EncodeToString([]byte(rawText))
}

func constructFallbackURL(n singbox.Node, srv string) string {
	if srv == "" || n.Port <= 0 {
		return ""
	}

	name := strings.TrimSpace(n.Name)
	if name == "" {
		name = strings.TrimSpace(n.Tag)
	}
	name = strings.TrimSuffix(name, ".json")
	if name == "" {
		name = fmt.Sprintf("Node-%s-%d", n.Protocol, n.Port)
	}
	escapedName := url.PathEscape(name)

	rawProto := strings.ToLower(strings.TrimSpace(n.RawProtocol))
	fullProto := strings.ToLower(strings.TrimSpace(n.Protocol))

	// 1. VLESS-REALITY
	if strings.Contains(rawProto, "vless") || strings.Contains(fullProto, "vless") || strings.Contains(rawProto, "reality") || strings.Contains(fullProto, "reality") {
		sni := n.SNI
		if sni == "" {
			sni = "www.apple.com"
		}
		flow := n.Flow
		if flow == "" {
			flow = "xtls-rprx-vision"
		}
		return fmt.Sprintf("vless://%s@%s:%d?encryption=none&flow=%s&security=reality&sni=%s&fp=chrome&pbk=%s&type=tcp#%s",
			n.UUID, srv, n.Port, flow, sni, n.PBK, escapedName)
	}

	// 2. Hysteria2
	if strings.Contains(rawProto, "hysteria2") || strings.Contains(fullProto, "hysteria2") || strings.Contains(rawProto, "hy2") || strings.Contains(fullProto, "hy2") {
		sni := n.SNI
		if sni == "" {
			sni = srv
		}
		return fmt.Sprintf("hysteria2://%s@%s:%d?insecure=1&sni=%s#%s",
			n.Password, srv, n.Port, sni, escapedName)
	}

	// 3. TUIC
	if strings.Contains(rawProto, "tuic") || strings.Contains(fullProto, "tuic") {
		return fmt.Sprintf("tuic://%s:%s@%s:%d?congestion_control=bbr&alpn=h3&sni=%s&allow_insecure=1#%s",
			n.UUID, n.Password, srv, n.Port, srv, escapedName)
	}

	// 4. AnyTLS
	if strings.Contains(rawProto, "anytls") || strings.Contains(fullProto, "anytls") {
		return fmt.Sprintf("anytls://%s@%s:%d?insecure=1&allowInsecure=1#%s",
			n.Password, srv, n.Port, escapedName)
	}

	// 5. Shadowsocks
	if strings.Contains(rawProto, "shadowsocks") || strings.Contains(fullProto, "shadowsocks") || rawProto == "ss" {
		userInfo := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", n.SSMethod, n.Password)))
		return fmt.Sprintf("ss://%s@%s:%d#%s", userInfo, srv, n.Port, escapedName)
	}

	return ""
}
