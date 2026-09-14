package nodes

import (
	"time"
)

type ServiceUnlockStatus string

const (
	StatusUnlocked ServiceUnlockStatus = "unlocked" // 解锁可用
	StatusBlocked  ServiceUnlockStatus = "blocked"  // 风控或地区封锁
	StatusUnknown  ServiceUnlockStatus = "unknown"  // 探测超时或未知
)

type UnlockResult struct {
	IP            string              `json:"ip"`
	OpenAI        ServiceUnlockStatus `json:"openai"`         // ChatGPT / OpenAI
	Claude        ServiceUnlockStatus `json:"claude"`         // Claude / Anthropic
	Gemini        ServiceUnlockStatus `json:"gemini"`         // Google Gemini
	Google        ServiceUnlockStatus `json:"google"`         // Google Search / 204
	Netflix       ServiceUnlockStatus `json:"netflix"`        // Netflix
	NetflixRegion string              `json:"netflix_region"` // 地区代码，如 "JP", "US"
	IsProbed      bool                `json:"is_probed"`      // 是否经过物理虚拟网卡真实流量探测
	CheckedAt     time.Time           `json:"checked_at"`
}

// MatchFilter checks whether the unlock status satisfies the requested filter criteria.
func (u *UnlockResult) MatchFilter(filter string) bool {
	if filter == "" || filter == "none" || filter == "all_unlimited" {
		return true
	}
	if u == nil {
		return false
	}
	switch filter {
	case "ai":
		// 三大主流 AI 强一致性校验：物理实测必须 ChatGPT、Claude、Gemini 全部解锁
		if u.IsProbed {
			return u.OpenAI == StatusUnlocked && u.Claude == StatusUnlocked && u.Gemini == StatusUnlocked
		}
		// 尚未实测的候选节点：要求三家均不得处于明确封锁状态
		return u.OpenAI != StatusBlocked && u.Claude != StatusBlocked && u.Gemini != StatusBlocked
	case "openai", "chatgpt":
		if u.IsProbed {
			return u.OpenAI == StatusUnlocked
		}
		return u.OpenAI != StatusBlocked
	case "claude":
		if u.IsProbed {
			return u.Claude == StatusUnlocked
		}
		return u.Claude != StatusBlocked
	case "gemini":
		if u.IsProbed {
			return u.Gemini == StatusUnlocked
		}
		return u.Gemini != StatusBlocked
	case "streaming":
		if u.IsProbed {
			return u.Netflix == StatusUnlocked || u.Google == StatusUnlocked
		}
		return u.Netflix != StatusBlocked || u.Google != StatusBlocked
	case "netflix":
		if u.IsProbed {
			return u.Netflix == StatusUnlocked
		}
		return u.Netflix != StatusBlocked
	case "full", "both", "all":
		if u.IsProbed {
			hasAI := (u.OpenAI == StatusUnlocked && u.Claude == StatusUnlocked && u.Gemini == StatusUnlocked)
			hasStreaming := (u.Netflix == StatusUnlocked || u.Google == StatusUnlocked)
			return hasAI && hasStreaming
		}
		hasAI := (u.OpenAI != StatusBlocked && u.Claude != StatusBlocked && u.Gemini != StatusBlocked)
		hasStreaming := (u.Netflix != StatusBlocked || u.Google != StatusBlocked)
		return hasAI && hasStreaming
	default:
		return true
	}
}
