package nodes

import (
	"math"
	"net"
)

// RiskLevel defines the qualitative risk evaluation of an egress node IP.
type RiskLevel string

const (
	RiskLevelPure     RiskLevel = "pure"     // 极度纯净 (< 0.25%)
	RiskLevelClean    RiskLevel = "clean"    // 纯净住宅 (0.25% - 5%)
	RiskLevelElevated RiskLevel = "elevated" // 轻微风险 (5% - 20%)
	RiskLevelHigh     RiskLevel = "high"     // 高风险 (20% - 50%)
	RiskLevelCritical RiskLevel = "critical" // 极度危险 (>= 50%)
	RiskLevelUnknown  RiskLevel = "unknown"  // 未知
)

// RiskEvaluation holds computed risk score and metadata.
type RiskEvaluation struct {
	Score      float64   `json:"score"`       // 归一化百分比得分 (0.0 ~ 100.0)
	Level      RiskLevel `json:"level"`       // pure, clean, elevated, high, critical, unknown
	Label      string    `json:"label"`       // 中文等级标签
	FraudFlags []string  `json:"fraud_flags"` // 命中风险标签 (bogon, datacenter, proxy, vpn, unstable, low_reputation)
	IsBogon    bool      `json:"is_bogon"`
	IsProxy    bool      `json:"is_proxy"`
	IsHosting  bool      `json:"is_hosting"`
}

// IsBogonIP checks if an IP belongs to private, loopback, link-local, multicast, or invalid ranges.
func IsBogonIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 0 || ip4[0] == 255 {
			return true
		}
	}
	return false
}

// CalculateRisk evaluates a node's IP risk score and returns a RiskEvaluation.
// Inspired by CF-Workers-CheckSocks5 exit risk evaluation and tailored for
// multi-egress residential network intelligence.
func CalculateRisk(n *Node, isProxy bool) RiskEvaluation {
	if n == nil || n.IP == "" {
		return RiskEvaluation{
			Score:      0,
			Level:      RiskLevelUnknown,
			Label:      "未知",
			FraudFlags: []string{},
		}
	}

	// 1. Bogon / Invalid IP check
	if IsBogonIP(n.IP) {
		return RiskEvaluation{
			Score:      100.0,
			Level:      RiskLevelCritical,
			Label:      "极度危险",
			FraudFlags: []string{"bogon"},
			IsBogon:    true,
			IsProxy:    isProxy,
			IsHosting:  n.IsHosting,
		}
	}

	var flags []string
	var baseScore float64

	// 2. Hosting / Datacenter Penalty
	if n.IsHosting {
		flags = append(flags, "datacenter")
		baseScore += 30.0
	}

	// 3. Proxy & VPN indicators
	if isProxy {
		flags = append(flags, "proxy")
		baseScore += 15.0
	}
	flags = append(flags, "vpn")
	baseScore += 3.0

	// 4. Stability & Reputation penalties
	if n.FailCount >= 3 {
		flags = append(flags, "unstable")
		baseScore += 15.0
	}
	if n.ReputationScore > 0 && n.ReputationScore < 50 {
		flags = append(flags, "low_reputation")
		baseScore += float64(50-n.ReputationScore) * 0.4
	}

	// 5. Residential Dampener / Discount
	// Verified high-confidence residential ISPs drastically reduce the risk score
	if n.IsResidential && !n.IsHosting {
		if n.ResConfidence >= 85 {
			baseScore *= 0.05 // Extreme purity: Top residential ISPs (Comcast, NTT, Chunghwa)
		} else if n.ResConfidence >= 70 {
			baseScore *= 0.15
		} else {
			baseScore *= 0.40
		}

		if n.ReputationScore >= 80 {
			baseScore *= 0.80
		}
	}

	// Clamp to 0.0 ~ 100.0 and round to 2 decimal places
	if baseScore < 0 {
		baseScore = 0
	} else if baseScore > 100.0 {
		baseScore = 100.0
	}
	roundedScore := math.Round(baseScore*100) / 100

	level, label := GetRiskMeta(roundedScore)

	return RiskEvaluation{
		Score:      roundedScore,
		Level:      level,
		Label:      label,
		FraudFlags: flags,
		IsBogon:    false,
		IsProxy:    isProxy,
		IsHosting:  n.IsHosting,
	}
}

// GetRiskMeta maps a numeric risk score (0.0 - 100.0) to level enum and human-readable label.
func GetRiskMeta(score float64) (RiskLevel, string) {
	switch {
	case score < 0.25:
		return RiskLevelPure, "极度纯净"
	case score < 5.0:
		return RiskLevelClean, "纯净住宅"
	case score < 20.0:
		return RiskLevelElevated, "轻微风险"
	case score < 50.0:
		return RiskLevelHigh, "高风险"
	default:
		return RiskLevelCritical, "极度危险"
	}
}
