package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"aimili-vpngate-go/pkg/singbox"
	"aimili-vpngate-go/pkg/tunnel"
)

var (
	reLeadingGroup   = regexp.MustCompile(`^\[[^\]]+\]\s*`)
	reCountryPrefix  = regexp.MustCompile(`^(?i)(JP|KR|US|SG|HK|TW|UK|GB|DE|FR|CA|AU|DIRECT|FAV)[-_]`)
	reDigits         = regexp.MustCompile(`\d+`)
)

// isEnglishRequest checks if the incoming HTTP request explicitly or implicitly prefers English.
func isEnglishRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.URL != nil && r.URL.Query().Get("lang") == "en" {
		return true
	}
	al := strings.ToLower(r.Header.Get("Accept-Language"))
	return strings.HasPrefix(al, "en")
}

// prepareSubscriptionNodes filters and decorates sing-box nodes for client subscription generation.
func (s *Server) prepareSubscriptionNodes(ctx context.Context, r *http.Request) ([]singbox.Node, error) {
	if s.singboxClient == nil {
		s.singboxClient = singbox.NewClient()
	}

	nodes, err := s.singboxClient.ListNodes(ctx)
	if err != nil {
		return nil, err
	}

	var q url.Values
	if r != nil && r.URL != nil {
		q = r.URL.Query()
	}

	filtered := s.FilterSubscriptionNodes(nodes, q)
	namingStyle := ""
	if q != nil {
		namingStyle = q.Get("naming")
	}
	decorated := s.DecorateSubscriptionNodesWithOptions(filtered, isEnglishRequest(r), namingStyle)
	return decorated, nil
}

// FilterSubscriptionNodes applies multi-dimensional filtering to sing-box nodes based on URL query parameters:
// - filter=favorites (or favorites=1/true): only nodes bound to favorites dynamic group
// - countries=JP,US (or country=JP): only nodes bound to dynamic groups matching target countries
// - group=dg-1 (or group_id=xxx): only nodes bound to a specific dynamic group ID or name
// - protocols=vless,hy2 (or protocol=reality): only nodes matching specified protocols
// - max_ping=150 (or max_latency=150): only nodes whose bound dynamic group has active latency <= max_ping ms
// - min_speed=50: only nodes whose bound dynamic group has active speed >= min_speed Mbps
func (s *Server) FilterSubscriptionNodes(nodes []singbox.Node, q url.Values) []singbox.Node {
	if len(nodes) == 0 || len(q) == 0 {
		return nodes
	}

	filterFav := strings.EqualFold(q.Get("filter"), "favorites") ||
		strings.EqualFold(q.Get("favorites"), "1") ||
		strings.EqualFold(q.Get("favorites"), "true")

	countriesRaw := q.Get("countries")
	if countriesRaw == "" {
		countriesRaw = q.Get("country")
	}
	var targetCountries []string
	if countriesRaw != "" {
		for _, c := range strings.Split(countriesRaw, ",") {
			c = strings.TrimSpace(strings.ToUpper(c))
			if c != "" {
				targetCountries = append(targetCountries, c)
			}
		}
	}

	targetGroup := strings.TrimSpace(q.Get("group"))
	if targetGroup == "" {
		targetGroup = strings.TrimSpace(q.Get("group_id"))
	}

	protocolsRaw := q.Get("protocols")
	if protocolsRaw == "" {
		protocolsRaw = q.Get("protocol")
	}
	var targetProtocols []string
	if protocolsRaw != "" {
		for _, p := range strings.Split(protocolsRaw, ",") {
			p = strings.TrimSpace(strings.ToLower(p))
			if p != "" {
				targetProtocols = append(targetProtocols, p)
			}
		}
	}

	maxPing := 0
	if mpStr := q.Get("max_ping"); mpStr != "" {
		maxPing, _ = strconv.Atoi(mpStr)
	} else if mlStr := q.Get("max_latency"); mlStr != "" {
		maxPing, _ = strconv.Atoi(mlStr)
	}

	minSpeed := 0
	if msStr := q.Get("min_speed"); msStr != "" {
		minSpeed, _ = strconv.Atoi(msStr)
	}

	// If no effective filters active, return nodes as is
	if !filterFav && len(targetCountries) == 0 && targetGroup == "" && len(targetProtocols) == 0 && maxPing <= 0 && minSpeed <= 0 {
		return nodes
	}

	var res []singbox.Node

	for _, n := range nodes {
		// 1. Protocol filter
		if len(targetProtocols) > 0 {
			proto := strings.ToLower(n.Protocol)
			rawProto := strings.ToLower(n.RawProtocol)
			matched := false
			for _, tp := range targetProtocols {
				if strings.Contains(proto, tp) || strings.Contains(rawProto, tp) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Determine outbound port and associated groups
		outboundRaw := strings.TrimSpace(n.Outbound)
		outboundPort := n.OutboundPort
		if outboundPort <= 0 && outboundRaw != "" && !strings.EqualFold(outboundRaw, "direct") && !strings.EqualFold(outboundRaw, "none") {
			if u, err := url.Parse(outboundRaw); err == nil && u.Port() != "" {
				outboundPort, _ = strconv.Atoi(u.Port())
			}
		}

		isDirect := outboundRaw == "direct" || outboundRaw == "none" || (outboundRaw == "" && outboundPort <= 0)

		// Collect groups associated with this node
		var boundGroups []*tunnel.DynamicGroup
		if outboundPort > 0 {
			defaultPort := 7928
			if s.cfg != nil && s.cfg.ProxyPort > 0 {
				defaultPort = s.cfg.ProxyPort
			}

			if outboundPort == defaultPort && s.dynamicMgr != nil {
				if sysGroup := s.dynamicMgr.GetGroup(tunnel.SystemPrimaryGroupID); sysGroup != nil {
					boundGroups = append(boundGroups, sysGroup)
				}
			} else if s.portMgr != nil {
				if rule := s.portMgr.GetRule(outboundPort); rule != nil && s.dynamicMgr != nil {
					for _, gid := range rule.BoundGroupIDs {
						if grp := s.dynamicMgr.GetGroup(gid); grp != nil {
							boundGroups = append(boundGroups, grp)
						}
					}
				}
			}
		}

		// 2. Favorites filter
		if filterFav {
			matched := false
			for _, bg := range boundGroups {
				if bg.Country == "FAVORITES" || strings.Contains(bg.Name, "收藏") || bg.ID == "dg-favorites" {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// 3. Country filter
		if len(targetCountries) > 0 {
			matched := false
			if isDirect {
				for _, tc := range targetCountries {
					if tc == "DIRECT" {
						matched = true
						break
					}
				}
			} else {
				for _, tc := range targetCountries {
					for _, bg := range boundGroups {
						if strings.EqualFold(bg.Country, tc) {
							matched = true
							break
						}
						// If dynamic group is FAVORITES or ANY, check active tunnel nodes
						if (bg.Country == "FAVORITES" || bg.Country == "") && s.tunnelPool != nil {
							for _, tid := range bg.ActiveTunnelIDs {
								if tun := s.tunnelPool.GetTunnel(tid); tun != nil && tun.Node != nil {
									if strings.EqualFold(tun.Node.CountryShort, tc) {
										matched = true
										break
									}
								}
							}
						}
					}
					if matched {
						break
					}
				}
			}
			if !matched {
				continue
			}
		}

		// 4. Group filter
		if targetGroup != "" {
			matched := false
			for _, bg := range boundGroups {
				if strings.EqualFold(bg.ID, targetGroup) || strings.EqualFold(bg.Name, targetGroup) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// 5. Ping filter (maxPing ms)
		if maxPing > 0 && !isDirect {
			matched := false
			if s.tunnelPool != nil {
				for _, bg := range boundGroups {
					for _, tid := range bg.ActiveTunnelIDs {
						if tun := s.tunnelPool.GetTunnel(tid); tun != nil {
							p := tun.LatencyMs
							if p <= 0 && tun.Node != nil {
								p = tun.Node.Ping
							}
							if p > 0 && p <= maxPing {
								matched = true
								break
							}
						}
					}
					if matched {
						break
					}
				}
			}
			if !matched && len(boundGroups) > 0 {
				continue
			}
		}

		// 6. Speed filter (minSpeed Mbps)
		if minSpeed > 0 && !isDirect {
			matched := false
			minBps := int64(minSpeed) * 1_000_000
			if s.tunnelPool != nil {
				for _, bg := range boundGroups {
					for _, tid := range bg.ActiveTunnelIDs {
						if tun := s.tunnelPool.GetTunnel(tid); tun != nil && tun.Node != nil {
							if tun.Node.Speed >= minBps {
								matched = true
								break
							}
						}
					}
					if matched {
						break
					}
				}
			}
			if !matched && len(boundGroups) > 0 {
				continue
			}
		}

		res = append(res, n)
	}

	return res
}

// resolveCleanProtocol returns a clean standardized protocol name (e.g. VLESS, Hysteria2, TUIC, Trojan, Shadowsocks).
func resolveCleanProtocol(n singbox.Node, cleanName string) string {
	raw := strings.ToLower(strings.TrimSpace(n.RawProtocol))
	proto := strings.ToLower(strings.TrimSpace(n.Protocol))
	clean := strings.ToLower(strings.TrimSpace(cleanName))
	combined := raw + " " + proto + " " + clean

	switch {
	case strings.Contains(combined, "reality") || strings.Contains(combined, "rh2"):
		return "VLESS"
	case strings.Contains(combined, "vless"):
		return "VLESS"
	case strings.Contains(combined, "hysteria2") || strings.Contains(combined, "hy2"):
		return "Hysteria2"
	case strings.Contains(combined, "tuic"):
		return "TUIC"
	case strings.Contains(combined, "trojan"):
		return "Trojan"
	case strings.Contains(combined, "shadowsocks") || strings.Contains(combined, "ss"):
		return "Shadowsocks"
	case strings.Contains(combined, "vmess"):
		return "VMess"
	case strings.Contains(combined, "anytls"):
		return "AnyTLS"
	case strings.Contains(combined, "socks"):
		return "Socks5"
	case strings.Contains(combined, "http"):
		return "HTTP"
	default:
		p := strings.TrimSpace(n.Protocol)
		if p != "" {
			return p
		}
		return "Proxy"
	}
}

// resolveNodeCountry determines the uppercase country code (e.g. JP, KR, US, DIRECT, FAV, GLOBAL)
// for an inbound sing-box node based on its outbound routing and active egress state.
func (s *Server) resolveNodeCountry(n singbox.Node, outboundPort int, outboundRaw string, cleanName string) string {
	if outboundRaw == "direct" || outboundRaw == "none" || (outboundRaw == "" && outboundPort <= 0) {
		return "DIRECT"
	}

	if outboundPort > 0 {
		defaultPort := 7928
		if s.cfg != nil && s.cfg.ProxyPort > 0 {
			defaultPort = s.cfg.ProxyPort
		}

		if outboundPort == defaultPort {
			if s.dynamicMgr != nil {
				if sysGroup := s.dynamicMgr.GetGroup(tunnel.SystemPrimaryGroupID); sysGroup != nil {
					if c := strings.TrimSpace(sysGroup.Country); c != "" && !strings.EqualFold(c, "FAVORITES") && !strings.EqualFold(c, "ALL") {
						return strings.ToUpper(c)
					}
					if s.tunnelPool != nil {
						for _, tid := range sysGroup.ActiveTunnelIDs {
							if tun := s.tunnelPool.GetTunnel(tid); tun != nil && tun.Node != nil && tun.Node.CountryShort != "" {
								return strings.ToUpper(tun.Node.CountryShort)
							}
						}
					}
				}
			}
			if s.tunnelPool != nil {
				for _, tun := range s.tunnelPool.ListTunnels() {
					if (tun.DevIndex == 0 || tun.DevName == "tun0") && tun.Node != nil && tun.Node.CountryShort != "" {
						return strings.ToUpper(tun.Node.CountryShort)
					}
				}
			}
			if s.vpn != nil {
				snap := s.vpn.Snapshot()
				if snap.ActiveNode != nil && snap.ActiveNode.CountryShort != "" {
					return strings.ToUpper(snap.ActiveNode.CountryShort)
				}
			}
		} else if s.portMgr != nil {
			if rule := s.portMgr.GetRule(outboundPort); rule != nil {
				if len(rule.BoundGroupIDs) > 0 && s.dynamicMgr != nil {
					for _, gid := range rule.BoundGroupIDs {
						if grp := s.dynamicMgr.GetGroup(gid); grp != nil {
							if c := strings.TrimSpace(grp.Country); c != "" && !strings.EqualFold(c, "FAVORITES") && !strings.EqualFold(c, "ALL") {
								return strings.ToUpper(c)
							}
							if s.tunnelPool != nil {
								for _, tid := range grp.ActiveTunnelIDs {
									if tun := s.tunnelPool.GetTunnel(tid); tun != nil && tun.Node != nil && tun.Node.CountryShort != "" {
										return strings.ToUpper(tun.Node.CountryShort)
									}
								}
							}
							if strings.EqualFold(grp.Country, "FAVORITES") {
								return "FAV"
							}
						}
					}
				}
				if len(rule.BoundTunnelIDs) > 0 && s.tunnelPool != nil {
					for _, tid := range rule.BoundTunnelIDs {
						if tun := s.tunnelPool.GetTunnel(tid); tun != nil && tun.Node != nil && tun.Node.CountryShort != "" {
							return strings.ToUpper(tun.Node.CountryShort)
						}
					}
				}
			}
		}
	}

	if m := reCountryPrefix.FindStringSubmatch(cleanName); len(m) > 1 {
		return strings.ToUpper(m[1])
	}

	if s.tunnelPool != nil {
		for _, tun := range s.tunnelPool.ListTunnels() {
			if tun.Node != nil && tun.Node.CountryShort != "" {
				return strings.ToUpper(tun.Node.CountryShort)
			}
		}
	}

	return "GLOBAL"
}

// DecorateSubscriptionNodes dynamically standardizes sing-box node names and URL fragments for presentation in client subscriptions.
// By default, it formats nodes as: {CountryCode}-{Protocol}-{Port} (e.g. JP-VLESS-443, KR-Hysteria2-8443, DIRECT-Trojan-443).
// Underlying sing-box configuration files and daemon tags remain strictly unmodified.
func (s *Server) DecorateSubscriptionNodes(nodes []singbox.Node, isEnglish bool) []singbox.Node {
	return s.DecorateSubscriptionNodesWithOptions(nodes, isEnglish, "")
}

// DecorateSubscriptionNodesWithOptions allows selecting naming style ("" for standard Country-Proto-Port, "group" for [Group] cleanName).
func (s *Server) DecorateSubscriptionNodesWithOptions(nodes []singbox.Node, isEnglish bool, namingStyle string) []singbox.Node {
	if len(nodes) == 0 {
		return nodes
	}

	decorated := make([]singbox.Node, len(nodes))
	usedNames := make(map[string]int)

	// Precompute baseNames and total counts for standard naming style to achieve symmetric numbering
	baseNames := make([]string, len(nodes))
	baseCounts := make(map[string]int)

	if !strings.EqualFold(namingStyle, "group") {
		for i, n := range nodes {
			cleanName := strings.TrimSpace(n.Name)
			if cleanName == "" {
				cleanName = strings.TrimSpace(n.Tag)
			}
			cleanName = strings.TrimSuffix(cleanName, ".json")
			cleanName = reLeadingGroup.ReplaceAllString(cleanName, "")
			cleanName = strings.TrimSpace(cleanName)
			if cleanName == "" {
				cleanName = fmt.Sprintf("Node-%s-%d", n.Protocol, n.Port)
			}

			outboundRaw := strings.TrimSpace(n.Outbound)
			outboundPort := n.OutboundPort
			if outboundPort <= 0 && outboundRaw != "" && !strings.EqualFold(outboundRaw, "direct") && !strings.EqualFold(outboundRaw, "none") {
				if u, err := url.Parse(outboundRaw); err == nil && u.Port() != "" {
					outboundPort, _ = strconv.Atoi(u.Port())
				} else if strings.Contains(outboundRaw, ":") {
					_, pStr, err := net.SplitHostPort(outboundRaw)
					if err == nil {
						outboundPort, _ = strconv.Atoi(pStr)
					}
				}
			}

			countryCode := s.resolveNodeCountry(n, outboundPort, outboundRaw, cleanName)
			protoName := resolveCleanProtocol(n, cleanName)
			port := n.Port
			if port <= 0 {
				matches := reDigits.FindAllString(cleanName, -1)
				if len(matches) > 0 {
					port, _ = strconv.Atoi(matches[len(matches)-1])
				}
			}
			if port <= 0 {
				port = 443
			}

			bn := fmt.Sprintf("%s-%s-%d", countryCode, protoName, port)
			baseNames[i] = bn
			baseCounts[bn]++
		}
	}

	for i, n := range nodes {
		decorated[i] = n

		var finalName string
		if strings.EqualFold(namingStyle, "group") {
			cleanName := strings.TrimSpace(n.Name)
			if cleanName == "" {
				cleanName = strings.TrimSpace(n.Tag)
			}
			cleanName = strings.TrimSuffix(cleanName, ".json")
			cleanName = reLeadingGroup.ReplaceAllString(cleanName, "")
			cleanName = strings.TrimSpace(cleanName)
			if cleanName == "" {
				cleanName = fmt.Sprintf("Node-%s-%d", n.Protocol, n.Port)
			}

			outboundRaw := strings.TrimSpace(n.Outbound)
			outboundPort := n.OutboundPort
			if outboundPort <= 0 && outboundRaw != "" && !strings.EqualFold(outboundRaw, "direct") && !strings.EqualFold(outboundRaw, "none") {
				if u, err := url.Parse(outboundRaw); err == nil && u.Port() != "" {
					outboundPort, _ = strconv.Atoi(u.Port())
				} else if strings.Contains(outboundRaw, ":") {
					_, pStr, err := net.SplitHostPort(outboundRaw)
					if err == nil {
						outboundPort, _ = strconv.Atoi(pStr)
					}
				}
			}

			groupLabel := ""
			if outboundRaw == "direct" || outboundRaw == "none" || (outboundRaw == "" && outboundPort <= 0) {
				if isEnglish {
					groupLabel = "Direct"
				} else {
					groupLabel = "直连"
				}
			} else if outboundPort > 0 {
				defaultPort := 7928
				if s.cfg != nil && s.cfg.ProxyPort > 0 {
					defaultPort = s.cfg.ProxyPort
				}

				if outboundPort == defaultPort {
					if s.dynamicMgr != nil {
						if sysGroup := s.dynamicMgr.GetGroup(tunnel.SystemPrimaryGroupID); sysGroup != nil && sysGroup.Name != "" {
							groupLabel = sysGroup.Name
						}
					}
					if groupLabel == "" {
						if isEnglish {
							groupLabel = "Default Gateway"
						} else {
							groupLabel = "默认出口"
						}
					}
				} else if s.portMgr != nil {
					if rule := s.portMgr.GetRule(outboundPort); rule != nil {
						if len(rule.BoundGroupIDs) > 0 && s.dynamicMgr != nil {
							var gNames []string
							for _, gid := range rule.BoundGroupIDs {
								if grp := s.dynamicMgr.GetGroup(gid); grp != nil && grp.Name != "" {
									gNames = append(gNames, grp.Name)
								}
							}
							if len(gNames) > 0 {
								groupLabel = strings.Join(gNames, "/")
							}
						}
						if groupLabel == "" && len(rule.BoundTunnelIDs) > 0 {
							if isEnglish {
								groupLabel = fmt.Sprintf("Tunnel Exit (%d)", len(rule.BoundTunnelIDs))
							} else {
								groupLabel = fmt.Sprintf("指定隧道出口 (%d)", len(rule.BoundTunnelIDs))
							}
						}
					}
					if groupLabel == "" {
						groupLabel = fmt.Sprintf("PORT %d", outboundPort)
					}
				} else {
					groupLabel = fmt.Sprintf("PORT %d", outboundPort)
				}
			}

			if groupLabel != "" {
				finalName = fmt.Sprintf("[%s] %s", groupLabel, cleanName)
			} else {
				finalName = cleanName
			}
		} else {
			// Standard clean format: {CountryCode}-{Protocol}-{Port} (e.g. JP-VLESS-443 or JP-VLESS-443-01 if multiple)
			baseName := baseNames[i]
			usedNames[baseName]++
			count := usedNames[baseName]
			if baseCounts[baseName] > 1 {
				finalName = fmt.Sprintf("%s-%02d", baseName, count)
			} else {
				finalName = baseName
			}
		}

		decorated[i].Name = finalName
		decorated[i].Tag = finalName

		if decorated[i].URL != "" {
			if hashIdx := strings.Index(decorated[i].URL, "#"); hashIdx != -1 {
				decorated[i].URL = decorated[i].URL[:hashIdx+1] + url.PathEscape(finalName)
			} else {
				decorated[i].URL = decorated[i].URL + "#" + url.PathEscape(finalName)
			}
		}
	}

	return decorated
}

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
