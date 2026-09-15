package tunnel

import (
	"testing"

	"aimili-vpngate-go/pkg/nodes"
)

func TestTripleAIValidation(t *testing.T) {
	tests := []struct {
		name     string
		unlock   *nodes.UnlockResult
		filter   string
		expected bool
	}{
		{
			name: "三大 AI 全部解锁 - 应通过",
			unlock: &nodes.UnlockResult{
				OpenAI:   nodes.StatusUnlocked,
				Claude:   nodes.StatusUnlocked,
				Gemini:   nodes.StatusUnlocked,
				IsProbed: true,
			},
			filter:   "ai",
			expected: true,
		},
		{
			name: "Claude 被阻断 - 应拒绝（现实场景）",
			unlock: &nodes.UnlockResult{
				OpenAI:   nodes.StatusUnlocked,
				Claude:   nodes.StatusBlocked,
				Gemini:   nodes.StatusUnlocked,
				IsProbed: true,
			},
			filter:   "ai",
			expected: false,
		},
		{
			name: "Gemini 被阻断 - 应拒绝",
			unlock: &nodes.UnlockResult{
				OpenAI:   nodes.StatusUnlocked,
				Claude:   nodes.StatusUnlocked,
				Gemini:   nodes.StatusBlocked,
				IsProbed: true,
			},
			filter:   "ai",
			expected: false,
		},
		{
			name: "OpenAI 被阻断 - 应拒绝",
			unlock: &nodes.UnlockResult{
				OpenAI:   nodes.StatusBlocked,
				Claude:   nodes.StatusUnlocked,
				Gemini:   nodes.StatusUnlocked,
				IsProbed: true,
			},
			filter:   "ai",
			expected: false,
		},
		{
			name: "全部被阻断 - 应拒绝",
			unlock: &nodes.UnlockResult{
				OpenAI:   nodes.StatusBlocked,
				Claude:   nodes.StatusBlocked,
				Gemini:   nodes.StatusBlocked,
				IsProbed: true,
			},
			filter:   "ai",
			expected: false,
		},
		{
			name: "未探测节点且无明确封锁 - 应通过（预判阶段）",
			unlock: &nodes.UnlockResult{
				OpenAI:   nodes.StatusUnknown,
				Claude:   nodes.StatusUnknown,
				Gemini:   nodes.StatusUnknown,
				IsProbed: false,
			},
			filter:   "ai",
			expected: true,
		},
		{
			name: "未探测但 Claude 已知封锁 - 应拒绝",
			unlock: &nodes.UnlockResult{
				OpenAI:   nodes.StatusUnknown,
				Claude:   nodes.StatusBlocked,
				Gemini:   nodes.StatusUnknown,
				IsProbed: false,
			},
			filter:   "ai",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.unlock.MatchFilter(tt.filter)
			if result != tt.expected {
				t.Errorf("MatchFilter(%q) = %v, 期望 %v\n"+
					"  OpenAI=%s, Claude=%s, Gemini=%s, IsProbed=%v",
					tt.filter, result, tt.expected,
					tt.unlock.OpenAI, tt.unlock.Claude, tt.unlock.Gemini, tt.unlock.IsProbed)
			}
		})
	}
}

func TestGeminiOnlyValidation(t *testing.T) {
	tests := []struct {
		name     string
		unlock   *nodes.UnlockResult
		filter   string
		expected bool
	}{
		{
			name: "Gemini 已解锁 - 应通过",
			unlock: &nodes.UnlockResult{
				Gemini:   nodes.StatusUnlocked,
				IsProbed: true,
			},
			filter:   "gemini",
			expected: true,
		},
		{
			name: "Gemini 被阻断 - 应拒绝",
			unlock: &nodes.UnlockResult{
				Gemini:   nodes.StatusBlocked,
				IsProbed: true,
			},
			filter:   "gemini",
			expected: false,
		},
		{
			name: "Gemini 未探测 - 应通过（预判阶段）",
			unlock: &nodes.UnlockResult{
				Gemini:   nodes.StatusUnknown,
				IsProbed: false,
			},
			filter:   "gemini",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.unlock.MatchFilter(tt.filter)
			if result != tt.expected {
				t.Errorf("MatchFilter(%q) = %v, 期望 %v (Gemini=%s)",
					tt.filter, result, tt.expected, tt.unlock.Gemini)
			}
		})
	}
}

func TestFullUnlockValidation(t *testing.T) {
	// "full" 过滤器：AI 全部解锁 + 至少一个流媒体解锁
	fullUnlocked := &nodes.UnlockResult{
		OpenAI:   nodes.StatusUnlocked,
		Claude:   nodes.StatusUnlocked,
		Gemini:   nodes.StatusUnlocked,
		Netflix:  nodes.StatusUnlocked,
		Google:   nodes.StatusUnlocked,
		IsProbed: true,
	}

	if !fullUnlocked.MatchFilter("full") {
		t.Error("全部解锁的节点应通过 full 过滤器")
	}

	// AI 全通 + Netflix 解锁（Google 被阻断）- 应通过
	aiPlusNetflix := &nodes.UnlockResult{
		OpenAI:   nodes.StatusUnlocked,
		Claude:   nodes.StatusUnlocked,
		Gemini:   nodes.StatusUnlocked,
		Netflix:  nodes.StatusUnlocked,
		Google:   nodes.StatusBlocked,
		IsProbed: true,
	}

	if !aiPlusNetflix.MatchFilter("full") {
		t.Error("AI 全通 + Netflix 解锁应通过 full 过滤器")
	}

	// AI 全通但流媒体全部被阻断 - 应拒绝
	aiOnlyNoStreaming := &nodes.UnlockResult{
		OpenAI:   nodes.StatusUnlocked,
		Claude:   nodes.StatusUnlocked,
		Gemini:   nodes.StatusUnlocked,
		Netflix:  nodes.StatusBlocked,
		Google:   nodes.StatusBlocked,
		IsProbed: true,
	}

	if aiOnlyNoStreaming.MatchFilter("full") {
		t.Error("流媒体全部被阻断的节点不应通过 full 过滤器")
	}

	// Claude 被阻断但流媒体全通 - 应拒绝（AI 不完整）
	claudeBlockedWithStreaming := &nodes.UnlockResult{
		OpenAI:   nodes.StatusUnlocked,
		Claude:   nodes.StatusBlocked,
		Gemini:   nodes.StatusUnlocked,
		Netflix:  nodes.StatusUnlocked,
		Google:   nodes.StatusUnlocked,
		IsProbed: true,
	}

	if claudeBlockedWithStreaming.MatchFilter("full") {
		t.Error("AI 未全通的节点不应通过 full 过滤器")
	}
}
