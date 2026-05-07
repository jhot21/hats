package templates

import (
	"testing"

	"codeberg.org/jhot/hats/internal/cmdstats"
)

func TestSparklineHeights_normalizes(t *testing.T) {
	var hours [24]int64
	hours[23] = 10 // current hour: max
	hours[22] = 5
	heights := SparklineHeights(hours)
	if heights[7] != 20 {
		t.Errorf("expected max bar=20, got %d", heights[7])
	}
	if heights[6] != 10 {
		t.Errorf("expected half-max bar=10, got %d", heights[6])
	}
}

func TestSparklineHeights_allZero(t *testing.T) {
	var hours [24]int64
	heights := SparklineHeights(hours)
	for i, h := range heights {
		if h != 2 {
			t.Errorf("expected min bar=2 at index %d, got %d", i, h)
		}
	}
}

func TestGroupByEndpoint(t *testing.T) {
	hash1 := "a" + string(make([]byte, 63))
	hash2 := "b" + string(make([]byte, 63))
	stats := []cmdstats.CommandStat{
		{CommandName: "alert", TokenHash: hash1, Hours: [24]int64{23: 10}},
		{CommandName: "alert", TokenHash: hash2, Hours: [24]int64{23: 5}},
		{CommandName: "unlock", TokenHash: hash1, Hours: [24]int64{23: 3}},
	}
	endpoints := GroupByEndpoint(stats)
	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}
	if endpoints[0].CommandName != "alert" {
		t.Errorf("expected alert first (15 calls), got %s", endpoints[0].CommandName)
	}
	if endpoints[0].TotalCalls24h != 15 {
		t.Errorf("expected 15 total calls for alert, got %d", endpoints[0].TotalCalls24h)
	}
	if endpoints[0].TokenCount != 2 {
		t.Errorf("expected 2 tokens for alert, got %d", endpoints[0].TokenCount)
	}
}

func TestTruncateHash(t *testing.T) {
	hash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	got := TruncateHash(hash)
	if got != "abcdef12…7890" {
		t.Errorf("unexpected truncated hash: %s", got)
	}
}
