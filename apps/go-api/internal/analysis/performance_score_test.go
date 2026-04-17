// Package analysis — performance_score_test.go : tests unitaires des algorithmes de performance.
package analysis

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func TestPercentileRank_MinValue(t *testing.T) {
	series := []float64{10, 20, 30, 40, 50}
	rank := PercentileRank(5, series) // en-dessous du min → 0%
	if rank < 0 || rank > 100 {
		t.Errorf("PercentileRank out of [0,100]: %f", rank)
	}
}

func TestPercentileRank_MaxValue(t *testing.T) {
	series := []float64{10, 20, 30, 40, 50}
	rank := PercentileRank(60, series) // au-dessus du max → 100%
	if rank < 0 || rank > 100 {
		t.Errorf("PercentileRank out of [0,100]: %f", rank)
	}
	if rank != 100.0 {
		t.Errorf("PercentileRank above max should be 100, got %f", rank)
	}
}

func TestPercentileRank_MedianValue(t *testing.T) {
	series := []float64{10, 20, 30, 40, 50}
	rank := PercentileRank(30, series) // médiane → ~60%
	// 3 valeurs ≤ 30 sur 5 → 60%
	if rank < 40 || rank > 80 {
		t.Errorf("PercentileRank of median ≈ 60, got %f", rank)
	}
}

func TestPercentileRankInverse_HalfPoint(t *testing.T) {
	series := []float64{10, 20, 30, 40, 50}
	// Inverse: combien >= 30? → 3 sur 5 = 60%
	val := PercentileRankInverse(30, series)
	if val < 40 || val > 80 {
		t.Errorf("PercentileRankInverse(30) ≈ 60, got %f", val)
	}
}

func TestComputePerformanceSeries_Empty(t *testing.T) {
	result := ComputePerformanceSeries(nil)
	if result != nil && len(result) != 0 {
		t.Errorf("expected empty result on nil input, got %d elements", len(result))
	}
}

func TestComputePerformanceSeries_SingleMatch(t *testing.T) {
	row := domain.StatsMatchRow{
		Kills:   10,
		Deaths:  2,
		Assists: 3,
	}
	result := ComputePerformanceSeries([]domain.StatsMatchRow{row})
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
}

// ── Sprint 48 : tests additionnels ──────────────────────────────────

func TestComputePerformanceSeries_MultipleMatches(t *testing.T) {
	rows := []domain.StatsMatchRow{
		{Kills: 10, Deaths: 5, Assists: 2, TimePlayedSeconds: intPtr(600)},
		{Kills: 15, Deaths: 3, Assists: 5, TimePlayedSeconds: intPtr(600)},
		{Kills: 5, Deaths: 10, Assists: 1, TimePlayedSeconds: intPtr(600)},
	}
	result := ComputePerformanceSeries(rows)
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}
}

func TestComputeNormalizedMetrics_PositiveKDA(t *testing.T) {
	row := domain.StatsMatchRow{
		Kills:             20,
		Deaths:            5,
		Assists:           10,
		TimePlayedSeconds: intPtr(600),
	}
	m := computeNormalizedMetrics(row)
	if m.kpm <= 0 {
		t.Error("expected positive kills per minute")
	}
	if m.dpmDeaths < 0 {
		t.Error("deaths per minute should not be negative")
	}
}

func TestComputeNormalizedMetrics_ZeroDuration(t *testing.T) {
	row := domain.StatsMatchRow{
		Kills:             10,
		Deaths:            5,
		Assists:           3,
		TimePlayedSeconds: intPtr(0),
	}
	m := computeNormalizedMetrics(row)
	// Avec duration=0, les per-minute ne doivent pas être Inf ou NaN
	if m.kpm < 0 {
		t.Error("kpm should not be negative")
	}
}

func TestApplyBotBonus_NoMMR(t *testing.T) {
	// Sans MMR, le bonus est basé uniquement sur le résultat
	row := domain.StatsMatchRow{}
	score := applyBotBonus(75.0, row)
	// La fonction ajoute toujours un bonus (pas de branche "no bonus")
	if score < 75.0 {
		t.Errorf("expected score >= 75.0, got %f", score)
	}
}

func TestApplyBotBonus_WithMMRGap(t *testing.T) {
	team := 1200.0
	enemy := 1500.0
	win := 2 // OutcomeWin
	row := domain.StatsMatchRow{
		TeamMMR:  &team,
		EnemyMMR: &enemy,
		Outcome:  &win,
	}
	score := applyBotBonus(75.0, row)
	if score <= 75.0 {
		t.Errorf("expected bonus for high enemy MMR, got %f", score)
	}
}

func TestComputeKDAFallback_Empty(t *testing.T) {
	row := domain.StatsMatchRow{Kills: 10, Deaths: 5, Assists: 3}
	result := computeKDAFallback(row, nil)
	if result == nil {
		t.Fatal("expected non-nil KDA fallback even with empty history")
	}
}

func TestComputeKDAFallback_WithHistory(t *testing.T) {
	history := []domain.StatsMatchRow{
		{Kills: 10, Deaths: 5, Assists: 3},
		{Kills: 15, Deaths: 3, Assists: 5},
	}
	row := domain.StatsMatchRow{Kills: 8, Deaths: 8, Assists: 2}
	result := computeKDAFallback(row, history)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestClampF(t *testing.T) {
	tests := []struct {
		v, min, max, want float64
	}{
		{5, 0, 10, 5},
		{-1, 0, 10, 0},
		{15, 0, 10, 10},
		{0, 0, 10, 0},
		{10, 0, 10, 10},
	}
	for _, tt := range tests {
		got := clampF(tt.v, tt.min, tt.max)
		if got != tt.want {
			t.Errorf("clampF(%f, %f, %f) = %f, want %f", tt.v, tt.min, tt.max, got, tt.want)
		}
	}
}

func TestPercentileRank_EmptySeries(t *testing.T) {
	// Avec série vide (<2 éléments), retourne 50.0 (valeur par défaut)
	rank := PercentileRank(50, nil)
	if rank != 50.0 {
		t.Errorf("PercentileRank with empty series should be 50.0 (default), got %f", rank)
	}
}

func intPtr(v int) *int { return &v }
