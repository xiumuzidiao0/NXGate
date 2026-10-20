package nodes

import (
	"fmt"
	"time"
)

type Node struct {
	ID             string    `json:"id"`
	HostName       string    `json:"hostname"`
	IP             string    `json:"ip"`
	Score          int64     `json:"score"`
	Ping           int       `json:"ping"`
	Speed          int64     `json:"speed"` // in bps
	CountryLong    string    `json:"country_long"`
	CountryShort   string    `json:"country_short"`
	NumVpnSessions int       `json:"num_vpn_sessions"`
	Uptime         int64     `json:"uptime"`
	TotalUsers     int64     `json:"total_users"`
	TotalTraffic   int64     `json:"total_traffic"`
	LogType        string    `json:"log_type"`
	Operator       string    `json:"operator"`
	Message        string    `json:"message"`
	ConfigData     string    `json:"config_data,omitempty"` // Raw ovpn config string
	Proto          string    `json:"proto"`
	Port           int       `json:"port"`
	SSTPPort       int       `json:"sstp_port,omitempty"`      // 预热探测已验证可用的 SSTP 监听端口
	ProbeStatus    string    `json:"probe_status,omitempty"`   // unknown, reachable (TCP only), unreachable, connected
	LastConnected  time.Time `json:"last_connected,omitempty"` // actual successful VPN connection, not source Ping
	LatencyMs      int       `json:"latency_ms"`               // Real-time measured TCP ping in ms (-1 if unreachable)
	LastChecked    time.Time `json:"last_checked"`

	// IP 归属地与网络类型探测
	IPType          string        `json:"ip_type"` // residential (住宅) / hosting (机房) / mobile (移动) / unknown
	ISP             string        `json:"isp"`
	City            string        `json:"city"`
	Region          string        `json:"region"`
	Latitude        float64       `json:"latitude"`
	Longitude       float64       `json:"longitude"`
	ASN             int           `json:"asn"`            // 自治系统编号
	RDNS            string        `json:"rdns,omitempty"` // 反向DNS记录
	IsHosting       bool          `json:"is_hosting"`
	IsResidential   bool          `json:"is_residential"` // 家宽/住宅IP标识
	ResConfidence   int           `json:"res_confidence"` // 家宽判定置信度 (0-100)
	ResReason       string        `json:"res_reason"`     // 家宽判定依据
	IsFavorite      bool          `json:"is_favorite"`
	ReputationScore int           `json:"reputation_score"`      // 历史综合稳定性信誉评分 (0-100)
	RiskScore       float64       `json:"risk_score"`            // 综合风险评分 (0.0 ~ 100.0)
	RiskLevel       RiskLevel     `json:"risk_level"`            // pure / clean / elevated / high / critical
	RiskLabel       string        `json:"risk_label"`            // 极度纯净 / 纯净住宅 / 轻微风险 / 高风险 / 极度危险
	FraudFlags      []string      `json:"fraud_flags,omitempty"` // 命中风险标签 (bogon, datacenter, proxy...)
	Unlock          *UnlockResult `json:"unlock,omitempty"`      // AI与主流流媒体解锁状态

	// 增量更新与生命周期跟踪
	FirstSeen time.Time `json:"first_seen,omitempty"`
	LastSeen  time.Time `json:"last_seen,omitempty"`
	FailCount int       `json:"fail_count,omitempty"` // 连续探测失败次数
}

func (n *Node) String() string {
	return fmt.Sprintf("[%s] %s (%s:%d %s, Score: %d, Speed: %.2f Mbps, Ping: %dms)",
		n.CountryShort, n.HostName, n.IP, n.Port, n.Proto, n.Score, float64(n.Speed)/1000000.0, n.Ping)
}
