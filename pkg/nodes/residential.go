package nodes

import (
	"net"
	"strings"
)

// NodeType 节点类型
type NodeType string

const (
	TypeResidential NodeType = "residential" // 家庭宽带/住宅IP
	TypeDatacenter  NodeType = "datacenter"  // 机房/数据中心
	TypeMobile      NodeType = "mobile"      // 移动蜂窝网络
	TypeUnknown     NodeType = "unknown"     // 未知
)

// Classification 分类结果
type Classification struct {
	Type       NodeType
	Confidence int    // 置信度 0-100
	Reason     string // 判定依据
}

// ResidentialDetector 家宽/住宅 IP 智能甄选引擎
// 基于 freesub 项目的五层瀑布式判定机制
type ResidentialDetector struct {
	cdnNetworks     []*net.IPNet // CDN 网段黑名单
	datacenterASNs  map[int]bool // 云厂商 ASN 黑名单
	residentialASNs map[int]bool // 主流运营商 ASN 白名单
	idcNamePatterns []string     // 机房 ISP 名称关键词
	resNamePatterns []string     // 家宽 ISP 名称关键词
}

func NewResidentialDetector() *ResidentialDetector {
	d := &ResidentialDetector{
		cdnNetworks:     make([]*net.IPNet, 0),
		datacenterASNs:  make(map[int]bool),
		residentialASNs: make(map[int]bool),
	}

	d.initCDNNetworks()
	d.initDatacenterASNs()
	d.initResidentialASNs()
	d.initNamePatterns()

	return d
}

// initCDNNetworks 初始化 CDN 网段黑名单
func (d *ResidentialDetector) initCDNNetworks() {
	cdnCIDRs := []string{
		// Cloudflare
		"173.245.48.0/20", "103.21.244.0/22", "103.22.200.0/22",
		"103.31.4.0/22", "141.101.64.0/18", "108.162.192.0/18",
		"190.93.240.0/20", "188.114.96.0/20", "197.234.240.0/22",
		"198.41.128.0/17", "162.158.0.0/15", "104.16.0.0/13",
		"104.24.0.0/14", "172.64.0.0/13", "131.0.72.0/22",

		// Fastly
		"23.235.32.0/20", "43.249.72.0/22", "103.244.50.0/24",
		"103.245.222.0/23", "103.245.224.0/24", "104.156.80.0/20",
		"151.101.0.0/16", "157.52.64.0/18", "167.82.0.0/17",
		"167.82.128.0/20", "167.82.160.0/20", "167.82.224.0/20",

		// Akamai (部分核心段)
		"23.0.0.0/12", "104.64.0.0/10",
	}

	for _, cidr := range cdnCIDRs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err == nil {
			d.cdnNetworks = append(d.cdnNetworks, ipnet)
		}
	}
}

// initDatacenterASNs 初始化云厂商 ASN 黑名单（78 个主流云服务商）
func (d *ResidentialDetector) initDatacenterASNs() {
	dcASNs := []int{
		// 美国云厂商
		16509, 14618, // AWS
		15169, 396982, // Google Cloud, GCP
		8075,  // Microsoft Azure
		13335, // Cloudflare
		20940, // Akamai
		16625, // Akamai

		// 欧洲云厂商
		24940, 213230, // Hetzner
		16276,                      // OVH
		51167, 12876, 12989, 16276, // OVH SAS
		60068, // CDN77

		// 亚洲云厂商
		45102, 132203, 55990, // Alibaba Cloud
		56046, 132203, // Tencent Cloud
		4808, 4847, // China Telecom/Unicom Cloud
		4134, 4812, // China Telecom IDC

		// 美国 VPS/IDC
		19318, 20473, // Vultr, ChopaVPS
		63949, 46606, 62904, // Linode/Akamai
		14061, 63473, // DigitalOcean
		36351, 20326, // SoftLayer (IBM Cloud)
		26496, // GoDaddy

		// 欧洲 IDC
		29802, // HVC/nl
		31034, // Aruba IT
		56630, // Melbicom
		43350, // NForce

		// 其他知名 VPS/CDN
		209242, 202425, 54290, 33724, 36007, // BuyVM/FranTech, SiteGround
		21859, 54113, 33182, // Zenlayer
		133752, 134548, 138915, // LeaseWeb
		54994, 54641, // Psychz Networks
		19531, // Pandora/YISP
	}

	for _, asn := range dcASNs {
		d.datacenterASNs[asn] = true
	}
}

// initResidentialASNs 初始化主流运营商 ASN 白名单（182 个全球家宽运营商）
func (d *ResidentialDetector) initResidentialASNs() {
	resASNs := []int{
		// === 台湾 ===
		3462,   // Chunghwa Telecom (中华电信)
		9924,   // Taiwan Mobile (台湾大哥大)
		17709,  // Aptg Telecom (亚太电信)
		4780,   // FET (远传电信)
		18049,  // Kbro (凯擘大宽频)
		131584, // Taiwan Fixed Network (台湾固网)

		// === 日本 ===
		4713,  // NTT OCN (NTT 光纤)
		2516,  // KDDI
		17676, // SoftBank BB
		4721,  // IIJ (Internet Initiative Japan)
		9605,  // NTT Communications
		7506,  // GMO Internet
		2527,  // So-net
		7684,  // Rakuten
		17511, // BIGLOBE
		10010, // Yahoo! Japan (Tokai)

		// === 韩国 ===
		4766, // KT (Korea Telecom)
		9318, // SK Broadband
		9316, // Hanaro Telecom (SK)
		3786, // LG Uplus (LG U+)
		9644, // SK Telecom

		// === 香港 ===
		9304,  // HGC (和记环球电讯)
		4515,  // PCCW-HKT (电讯盈科)
		9269,  // HKBN (香港宽频)
		55933, // Cloudie Limited

		// === 新加坡 ===
		4657,  // StarHub Cable
		9506,  // Singtel
		10111, // M1 Limited

		// === 美国 ===
		7922,                                            // Comcast Cable (康卡斯特)
		20115,                                           // Charter Communications (Spectrum)
		22773,                                           // Cox Communications
		7018,                                            // AT&T Services
		701,                                             // Verizon Business / UUNet
		6167,                                            // Verizon Wireless
		6128,                                            // Cablevision (Optimum)
		11427,                                           // Time Warner Cable (now Charter)
		33363,                                           // Bright House Networks
		11426,                                           // TWC (Time Warner Cable Midwest)
		12271,                                           // Charter Communications (legacy)
		10796,                                           // TWC Northeast
		33491, 33650, 33652, 33660, 33662, 33667, 33668, // TWC regions

		// === 英国 ===
		2856,  // BT (British Telecom)
		5089,  // Virgin Media
		2818,  // BBC
		20712, // Andrews & Arnold (A&A)
		5607,  // Sky Broadband
		31459, // TalkTalk

		// === 德国 ===
		3320,  // Deutsche Telekom
		6805,  // Telefonica Germany (O2)
		3209,  // Vodafone Germany
		8881,  // Versatel / 1&1
		29562, // Kabelbw / Unitymedia

		// === 法国 ===
		3215,  // Orange France
		5410,  // Bouygues Telecom
		15557, // SFR (Société française du radiotéléphone)
		12322, // Free SAS (Free.fr)

		// === 加拿大 ===
		812,  // Rogers Communications
		5645, // Rogers Cable
		6327, // Shaw Communications
		855,  // Bell Canada
		577,  // Bell Aliant
		6799, // Videotron (Quebec)

		// === 澳大利亚 ===
		1221, // Telstra
		4764, // TPG Internet (TPG Telecom)
		9443, // Vocus
		4739, // Internode (iiNet)
		7545, // TPG

		// === 新西兰 ===
		4648, // Spark NZ (formerly Telecom NZ)
		9503, // Vodafone NZ

		// === 印度 ===
		9498,  // Bharti Airtel
		55836, // Reliance Jio
		24560, // Airtel Broadband

		// === 其他亚洲 ===
		7473,  // SingTel (Singapore Telecommunications)
		17621, // China Unicom (中国联通)
		4837,  // China Unicom Backbone
		4134,  // China Telecom (部分家宽)
		9808,  // China Mobile (中国移动)
		23724, // IDC Frontier / Yahoo BB (Japan)

		// === 中东 ===
		8968, // Etisalat (UAE)
		5384, // Emirates Internet (UAE)

		// === 南美 ===
		7738,  // Telecom Argentina
		28573, // NET (Claro Brasil)
		8151,  // Uninet (Mexico)
	}

	for _, asn := range resASNs {
		d.residentialASNs[asn] = true
	}
}

// initNamePatterns 初始化 ISP 名称关键词
func (d *ResidentialDetector) initNamePatterns() {
	// 家宽特征词
	d.resNamePatterns = []string{
		"broadband", "fiber", "fibre", "dsl", "cable", "pppoe",
		"residential", "home", "hinet", "chunghwa", "comcast",
		"charter", "cox", "verizon", "att", "bt.com", "virgin",
		"telecom", "telekom", "telefonica", "orange", "bouygues",
		"rogers", "shaw", "bell", "telstra", "optus", "kddi",
		"softbank", "ocn", "biglobe", "so-net", "rakuten",
		"kt.com", "hanaro", "lg-uplus", "hkbn", "pccw",
		"starhu", "singtel", "etisalat", "airtel", "jio",
		"dynamic", "dhcp", "pool", "客户", "用户", "宽带",
		"家庭", "住宅", "光纤", "电信", "联通", "移动",
	}

	// 机房特征词
	d.idcNamePatterns = []string{
		"hosting", "datacenter", "data center", "cloud", "vps",
		"dedicated", "server", "colocation", "colo", "aws",
		"amazon", "google", "azure", "digitalocean", "linode",
		"vultr", "hetzner", "ovh", "scaleway", "alibaba",
		"tencent", "baidu", "cloudflare", "akamai", "fastly",
		"cdn", "edge", "anycast", "transit", "backbone",
		"idc", "bgp", "ix", "peer", "enterprise",
	}
}

// Classify 执行五层瀑布式判定
func (d *ResidentialDetector) Classify(ip string, asn int, isp, rdns string) Classification {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return Classification{Type: TypeUnknown, Confidence: 0, Reason: "Invalid IP"}
	}

	// 1) CDN 网段黑名单（100% 置信）
	for _, ipnet := range d.cdnNetworks {
		if ipnet.Contains(parsedIP) {
			return Classification{
				Type:       TypeDatacenter,
				Confidence: 100,
				Reason:     "CDN network range",
			}
		}
	}

	// 2) ASN 白名单（90% 置信）
	if asn > 0 && d.residentialASNs[asn] {
		return Classification{
			Type:       TypeResidential,
			Confidence: 90,
			Reason:     "Residential ISP ASN whitelist",
		}
	}

	// 3) ASN 黑名单（90% 置信）
	if asn > 0 && d.datacenterASNs[asn] {
		return Classification{
			Type:       TypeDatacenter,
			Confidence: 90,
			Reason:     "Cloud provider ASN blacklist",
		}
	}

	// 4) ISP 名称关键词（70% 置信）
	ispLower := strings.ToLower(isp)

	// 机房特征优先（避免误判）
	for _, pattern := range d.idcNamePatterns {
		if strings.Contains(ispLower, pattern) {
			return Classification{
				Type:       TypeDatacenter,
				Confidence: 75,
				Reason:     "ISP name contains datacenter keyword: " + pattern,
			}
		}
	}

	// 家宽特征
	for _, pattern := range d.resNamePatterns {
		if strings.Contains(ispLower, pattern) {
			return Classification{
				Type:       TypeResidential,
				Confidence: 70,
				Reason:     "ISP name contains residential keyword: " + pattern,
			}
		}
	}

	// 5) rDNS 兜底（60% 置信）
	if rdns != "" {
		rdnsLower := strings.ToLower(rdns)

		// 动态 IP 特征（家宽常见）
		dynamicPatterns := []string{"dynamic", "dhcp", "pool", "pppoe", "客户", "用户"}
		for _, pattern := range dynamicPatterns {
			if strings.Contains(rdnsLower, pattern) {
				return Classification{
					Type:       TypeResidential,
					Confidence: 65,
					Reason:     "rDNS contains dynamic IP pattern: " + pattern,
				}
			}
		}

		// 服务器特征（机房常见）
		serverPatterns := []string{"server", "host", "vps", "cloud", "vm"}
		for _, pattern := range serverPatterns {
			if strings.Contains(rdnsLower, pattern) {
				return Classification{
					Type:       TypeDatacenter,
					Confidence: 60,
					Reason:     "rDNS contains server pattern: " + pattern,
				}
			}
		}
	}

	// 默认：信息不足，低置信度
	return Classification{
		Type:       TypeUnknown,
		Confidence: 0,
		Reason:     "Insufficient data for classification",
	}
}
