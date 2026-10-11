package nodes

import (
	"context"
	"testing"
	"time"
)

func TestEnrichNodes_CacheHitAndRisk(t *testing.T) {
	tempDir := t.TempDir()
	enricher := NewIPEnricher(tempDir)

	// Pre-populate cache
	enricher.mu.Lock()
	enricher.cache["114.34.25.10"] = &IPCacheEntry{
		IP:            "114.34.25.10",
		IPType:        "residential",
		ISP:           "Chunghwa Telecom",
		City:          "Taipei",
		Region:        "Taipei City",
		Lat:           25.0330,
		Lon:           121.5654,
		ASN:           3462,
		IsResidential: true,
		ResConfidence: 90,
		Proxy:         false,
		CachedAt:      time.Now(),
	}
	enricher.mu.Unlock()

	node := &Node{
		ID:              "test-node-1",
		IP:              "114.34.25.10",
		ReputationScore: 90,
	}

	enricher.EnrichNodes(context.Background(), []*Node{node})

	if node.City != "Taipei" {
		t.Errorf("expected City Taipei, got %s", node.City)
	}
	if node.Latitude != 25.0330 || node.Longitude != 121.5654 {
		t.Errorf("expected coordinates (25.0330, 121.5654), got (%f, %f)", node.Latitude, node.Longitude)
	}
	if node.RiskLevel != RiskLevelPure {
		t.Errorf("expected RiskLevelPure, got %s (score: %f)", node.RiskLevel, node.RiskScore)
	}
	if node.RiskLabel != "极度纯净" {
		t.Errorf("expected label 极度纯净, got %s", node.RiskLabel)
	}
}
