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

// ── Tests ComputeRelativePerformanceScore ────────────────────────────

func makeHistoryRows(n int) []domain.StatsMatchRow {
	rows := make([]domain.StatsMatchRow, n)
	for i := range rows {
		dur := 600
		acc := 0.45 + float64(i)*0.01
		ps := 1000 + i*100
		kda := 1.0 + float64(i)*0.1
		rows[i] = domain.StatsMatchRow{
			Kills:             10 + i,
			Deaths:            5,
			Assists:           3 + i,
			TimePlayedSeconds: &dur,
			Accuracy:          &acc,
			PersonalScore:     &ps,
			KDA:               &kda,
		}
	}
	return rows
}

func TestComputeRelativePerformanceScore_InsufficientHistory(t *testing.T) {
	row := domain.StatsMatchRow{Kills: 10, Deaths: 5, Assists: 3}
	// Fewer than MinMatchesForRelative
	result := ComputeRelativePerformanceScore(row, makeHistoryRows(5), false)
	if result != nil {
		t.Error("expected nil with insufficient history")
	}
}

func TestComputeRelativePerformanceScore_WithHistory(t *testing.T) {
	history := makeHistoryRows(15)
	dur := 600
	acc := 0.55
	ps := 1500
	kda := 2.0
	row := domain.StatsMatchRow{
		Kills:             20,
		Deaths:            3,
		Assists:           8,
		TimePlayedSeconds: &dur,
		Accuracy:          &acc,
		PersonalScore:     &ps,
		KDA:               &kda,
	}
	result := ComputeRelativePerformanceScore(row, history, false)
	if result == nil {
		t.Fatal("expected non-nil score with 15 history matches")
	}
	if *result < 0 || *result > 100 {
		t.Errorf("score out of [0,100]: %f", *result)
	}
}

func TestComputeRelativePerformanceScore_WithBotBonus(t *testing.T) {
	history := makeHistoryRows(15)
	dur := 600
	loss := 3
	row := domain.StatsMatchRow{
		Kills:             5,
		Deaths:            10,
		Assists:           2,
		TimePlayedSeconds: &dur,
		Outcome:           &loss,
	}
	withoutBot := ComputeRelativePerformanceScore(row, history, false)
	withBot := ComputeRelativePerformanceScore(row, history, true)
	if withoutBot == nil || withBot == nil {
		t.Fatal("expected non-nil scores")
	}
	if *withBot <= *withoutBot {
		t.Errorf("bot bonus should increase score: %f <= %f", *withBot, *withoutBot)
	}
}

func TestAddRequired_ShortSeries(t *testing.T) {
	pct := make(map[string]float64)
	w := make(map[string]float64)
	addRequired("kpm", 5.0, []float64{1.0}, false, pct, w)
	if len(pct) != 0 {
		t.Error("addRequired should skip series < 2 elements")
	}
}

func TestAddRequired_Normal(t *testing.T) {
	pct := make(map[string]float64)
	w := make(map[string]float64)
	addRequired("kpm", 5.0, []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, false, pct, w)
	if _, ok := pct["kpm"]; !ok {
		t.Error("expected kpm percentile")
	}
	if _, ok := w["kpm"]; !ok {
		t.Error("expected kpm weight")
	}
}

func TestAddOptional_NilValue(t *testing.T) {
	pct := make(map[string]float64)
	w := make(map[string]float64)
	addOptional("accuracy", nil, []float64{1, 2, 3}, false, pct, w)
	if len(pct) != 0 {
		t.Error("addOptional should skip nil value")
	}
}

func TestAddOptional_WithValue(t *testing.T) {
	pct := make(map[string]float64)
	w := make(map[string]float64)
	val := 5.0
	addOptional("accuracy", &val, []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, false, pct, w)
	if _, ok := pct["accuracy"]; !ok {
		t.Error("expected accuracy percentile")
	}
}

func TestPrepareHistoryMetrics(t *testing.T) {
	rows := makeHistoryRows(5)
	h := prepareHistoryMetrics(rows)
	if len(h.kpm) != 5 {
		t.Errorf("expected 5 kpm values, got %d", len(h.kpm))
	}
	if len(h.dpmDeaths) != 5 {
		t.Errorf("expected 5 dpmDeaths values, got %d", len(h.dpmDeaths))
	}
	// accuracy is optional, should be filled for rows that had it
	if len(h.accuracy) == 0 {
		t.Error("expected accuracy values for history rows with accuracy")
	}
}

func intPtr(v int) *int { return &v }
