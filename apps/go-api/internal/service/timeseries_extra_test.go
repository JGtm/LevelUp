package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/legacymatch"
)

// ---------- computeRegressionStats ----------

func TestComputeRegressionStats_NotEnough(t *testing.T) {
	matches := make([]legacymatch.StatsMatchRow, 10) // < 20
	result := computeRegressionStats(matches)
	if result.HasEnoughForTrend {
		t.Error("expected HasEnoughForTrend=false for <20 matches")
	}
}

func TestComputeRegressionStats_Improving(t *testing.T) {
	matches := make([]legacymatch.StatsMatchRow, 30)
	for i := range matches {
		matches[i] = legacymatch.StatsMatchRow{
			MatchID:   "m",
			StartTime: time.Now(),
			Kills:     i + 1, // increasing kills
			Deaths:    5,
		}
	}
	result := computeRegressionStats(matches)
	if !result.HasEnoughForTrend {
		t.Fatal("expected HasEnoughForTrend=true")
	}
	if result.Trend == nil || *result.Trend != "improving" {
		t.Errorf("expected improving trend, got %v", result.Trend)
	}
	if result.KDSlope == nil || *result.KDSlope <= 0 {
		t.Errorf("expected positive slope, got %v", result.KDSlope)
	}
}

func TestComputeRegressionStats_Stable(t *testing.T) {
	matches := make([]legacymatch.StatsMatchRow, 25)
	for i := range matches {
		matches[i] = legacymatch.StatsMatchRow{
			MatchID:   "m",
			StartTime: time.Now(),
			Kills:     10,
			Deaths:    10,
		}
	}
	result := computeRegressionStats(matches)
	if !result.HasEnoughForTrend {
		t.Fatal("expected HasEnoughForTrend=true")
	}
	if result.Trend == nil || *result.Trend != "stable" {
		t.Errorf("expected stable, got %v", result.Trend)
	}
}

func TestComputeRegressionStats_Declining(t *testing.T) {
	matches := make([]legacymatch.StatsMatchRow, 30)
	for i := range matches {
		matches[i] = legacymatch.StatsMatchRow{
			MatchID:   "m",
			StartTime: time.Now(),
			Kills:     30 - i, // decreasing kills
			Deaths:    5,
		}
	}
	result := computeRegressionStats(matches)
	if result.Trend == nil || *result.Trend != "declining" {
		t.Errorf("expected declining, got %v", result.Trend)
	}
}

func TestComputeRegressionStats_AllZeroDeaths(t *testing.T) {
	matches := make([]legacymatch.StatsMatchRow, 25)
	for i := range matches {
		matches[i] = legacymatch.StatsMatchRow{
			MatchID:   "m",
			StartTime: time.Now(),
			Kills:     10,
			Deaths:    0, // kd = 0 (division guard)
		}
	}
	result := computeRegressionStats(matches)
	// Should not crash, kd=0 for all
	if !result.HasEnoughForTrend {
		t.Error("expected HasEnoughForTrend=true")
	}
}
