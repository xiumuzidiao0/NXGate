package nodes

import (
	"testing"
)

func TestIsBogonIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"192.168.1.1", true},
		{"172.16.0.5", true},
		{"0.0.0.0", true},
		{"255.255.255.255", true},
		{"::1", true},
		{"fe80::1", true},
		{"invalid-ip", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"114.114.114.114", false},
		{"2001:4860:4860::8888", false},
	}

	for _, tc := range tests {
		got := IsBogonIP(tc.ip)
		if got != tc.expected {
			t.Errorf("IsBogonIP(%s) = %v, expected %v", tc.ip, got, tc.expected)
		}
	}
}

func TestCalculateRisk_Bogon(t *testing.T) {
	node := &Node{IP: "192.168.1.100"}
	eval := CalculateRisk(node, false)

	if eval.Level != RiskLevelCritical {
		t.Errorf("expected RiskLevelCritical for Bogon IP, got %v", eval.Level)
	}
	if eval.Score != 100.0 {
		t.Errorf("expected score 100.0, got %f", eval.Score)
	}
	if !eval.IsBogon {
		t.Errorf("expected IsBogon to be true")
	}
}

func TestCalculateRisk_PureResidential(t *testing.T) {
	node := &Node{
		IP:              "114.34.25.10",
		IsResidential:   true,
		IsHosting:       false,
		ResConfidence:   90,
		ReputationScore: 90,
		FailCount:       0,
	}

	eval := CalculateRisk(node, false)
	// baseScore = 3.0 (vpn) * 0.05 * 0.80 = 0.12 (< 0.25)
	if eval.Level != RiskLevelPure {
		t.Errorf("expected RiskLevelPure, got %v (score: %f)", eval.Level, eval.Score)
	}
	if eval.Score >= 0.25 {
		t.Errorf("expected score < 0.25 for pure residential, got %f", eval.Score)
	}
	if eval.Label != "极度纯净" {
		t.Errorf("expected label 极度纯净, got %s", eval.Label)
	}
}

func TestCalculateRisk_CleanResidential(t *testing.T) {
	node := &Node{
		IP:              "114.34.25.10",
		IsResidential:   true,
		IsHosting:       false,
		ResConfidence:   70,
		ReputationScore: 70,
		FailCount:       0,
	}

	eval := CalculateRisk(node, false)
	// baseScore = 3.0 * 0.15 = 0.45 (< 5.0)
	if eval.Level != RiskLevelClean {
		t.Errorf("expected RiskLevelClean, got %v (score: %f)", eval.Level, eval.Score)
	}
	if eval.Label != "纯净住宅" {
		t.Errorf("expected label 纯净住宅, got %s", eval.Label)
	}
}

func TestCalculateRisk_Datacenter(t *testing.T) {
	node := &Node{
		IP:            "104.28.19.5",
		IsHosting:     true,
		IsResidential: false,
	}

	eval := CalculateRisk(node, true)
	// baseScore = 30(datacenter) + 15(proxy) + 3(vpn) = 48.0 (high)
	if eval.Level != RiskLevelHigh {
		t.Errorf("expected RiskLevelHigh for hosting with proxy, got %v (score: %f)", eval.Level, eval.Score)
	}
	if !eval.IsHosting {
		t.Errorf("expected IsHosting to be true")
	}
	if !eval.IsProxy {
		t.Errorf("expected IsProxy to be true")
	}
}

func TestCalculateRisk_CriticalUnstableDatacenter(t *testing.T) {
	node := &Node{
		IP:              "104.28.19.5",
		IsHosting:       true,
		FailCount:       5,
		ReputationScore: 20,
	}

	eval := CalculateRisk(node, true)
	// 30(datacenter) + 15(proxy) + 3(vpn) + 15(unstable) + (50-20)*0.4(12) = 75.0 (critical)
	if eval.Level != RiskLevelCritical {
		t.Errorf("expected RiskLevelCritical for highly degraded node, got %v (score: %f)", eval.Level, eval.Score)
	}
	if eval.Label != "极度危险" {
		t.Errorf("expected label 极度危险, got %s", eval.Label)
	}
}

func TestCalculateRisk_EmptyNode(t *testing.T) {
	eval := CalculateRisk(nil, false)
	if eval.Level != RiskLevelUnknown {
		t.Errorf("expected RiskLevelUnknown for nil node, got %v", eval.Level)
	}
}
