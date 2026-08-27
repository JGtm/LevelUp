// Package sync — performance_unit_test.go : tests unitaires des fonctions de performance.
//
// Couvre extractMatchMetrics, percentileRank, percentileRankInverse,
// computeRelativePerformanceScore, getMetricValue, prepareHistoryMetrics,
// computeRankPerformance.
package sync

import (
	"math"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// extractMatchMetrics
// ─────────────────────────────────────────────────────────────────────────────

func TestExtractMatchMetrics_Normal(t *testing.T) {
	row := historyRow{
		TimePlayedSeconds: 600,
		Kills:             12,
		Deaths:            8,
		Assists:           5,
		DamageDealt:       4000,
		PersonalScore:     2500,
		Accuracy:          40.0,
	}
	m := extractMatchMetrics(&row)
	expectedKPM := 12.0 / 10.0 // 12 kills / 10 minutes
	if math.Abs(m.KPM-expectedKPM) > 0.01 {
		t.Errorf("KPM = %v, want %v", m.KPM, expectedKPM)
	}
	if math.Abs(m.DPMDeaths-0.8) > 0.01 {
		t.Errorf("DPMDeaths = %v, want 0.8", m.DPMDeaths)
	}
	if m.Accuracy == nil || math.Abs(*m.Accuracy-40.0) > 0.01 {
		t.Errorf("Accuracy = %v, want 40.0", m.Accuracy)
	}
	if m.DPMDamage == nil || math.Abs(*m.DPMDamage-400.0) > 0.1 {
		t.Errorf("DPMDamage = %v, want ~400.0", m.DPMDamage)
	}
}

func TestExtractMatchMetrics_ZeroDurationFallback(t *testing.T) {
	row := historyRow{
		TimePlayedSeconds: 0,
		Kills:             10,
		Deaths:            5,
	}
	m := extractMatchMetrics(&row)
	// Zero duration → falls back to 600s (10 min) → 10 kills / 10 min = 1.0 KPM.
	const expectedKPM = 1.0
	if math.Abs(m.KPM-expectedKPM) > 0.01 {
		t.Errorf("zero duration fallback: KPM = %v, want %v", m.KPM, expectedKPM)
	}
}

func TestExtractMatchMetrics_ZeroValues(t *testing.T) {
	row := historyRow{TimePlayedSeconds: 600}
	m := extractMatchMetrics(&row)
	if m.KPM != 0 || m.DPMDeaths != 0 {
		t.Errorf("zero values: got non-zero: KPM=%v DPM=%v", m.KPM, m.DPMDeaths)
	}
	if m.Accuracy != nil {
		t.Error("Accuracy should be nil when 0")
	}
}

func TestExtractMatchMetrics_KDAFallback(t *testing.T) {
	row := historyRow{
		TimePlayedSeconds: 600,
		Kills:             10,
		Deaths:            5,
		Assists:           5,
		KDA:               0, // will trigger fallback calculation
	}
	m := extractMatchMetrics(&row)
	expectedKDA := (10.0 + 5.0) / 5.0 // 3.0
	if math.Abs(m.KDA-expectedKDA) > 0.01 {
		t.Errorf("KDA fallback = %v, want %v", m.KDA, expectedKDA)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// percentileRank / percentileRankInverse
// ─────────────────────────────────────────────────────────────────────────────

func TestPercentileRank_Normal(t *testing.T) {
	series := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	got := percentileRank(50, series)
	// 5 values <= 50 → 5/10*100 = 50
	if math.Abs(got-50.0) > 1 {
		t.Errorf("percentileRank(50) = %v, want ~50", got)
	}
}

func TestPercentileRank_NilSlice(t *testing.T) {
	got := percentileRank(50, nil)
	if got != 50.0 {
		t.Errorf("empty slice: got %v, want 50.0", got)
	}
}

func TestPercentileRank_SingleElement(t *testing.T) {
	got := percentileRank(5, []float64{5})
	if got != 50.0 {
		t.Errorf("single element: got %v, want 50.0", got)
	}
}

func TestPercentileRank_AboveAll(t *testing.T) {
	series := []float64{1, 2, 3, 4, 5}
	got := percentileRank(100, series)
	if got != 100.0 {
		t.Errorf("above all: got %v, want 100.0", got)
	}
}

func TestPercentileRank_BelowAll(t *testing.T) {
	series := []float64{50, 60, 70, 80, 90}
	got := percentileRank(1, series)
	if got != 0 {
		t.Errorf("below all: got %v, want 0", got)
	}
}

func TestPercentileRankInverse_Normal(t *testing.T) {
	series := []float64{10, 20, 30, 40, 50}
	got := percentileRankInverse(10, series)
	// All 5 values >= 10 → 5/5*100 = 100
	if got != 100.0 {
		t.Errorf("inverse at min: got %v, want 100.0", got)
	}
}

func TestPercentileRankInverse_NilSlice(t *testing.T) {
	got := percentileRankInverse(5, nil)
	if got != 50.0 {
		t.Errorf("empty: got %v, want 50.0", got)
	}
}

func TestPercentileRankInverse_AtMax(t *testing.T) {
	series := []float64{10, 20, 30, 40, 50}
	got := percentileRankInverse(50, series)
	// 1 value >= 50 → 1/5*100 = 20
	if math.Abs(got-20.0) > 0.1 {
		t.Errorf("inverse at max: got %v, want 20.0", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// getMetricValue
// ─────────────────────────────────────────────────────────────────────────────

func TestGetMetricValue_AllMetrics(t *testing.T) {
	acc := 45.0
	pspm := 250.0
	dpmDmg := 400.0
	m := &matchMetrics{
		KPM:       1.5,
		DPMDeaths: 0.8,
		APM:       0.5,
		KDA:       2.0,
		Accuracy:  &acc,
		PSPM:      &pspm,
		DPMDamage: &dpmDmg,
	}
	cases := []struct {
		name string
		want float64
		ok   bool
	}{
		{"kpm", 1.5, true},
		{"dpm_deaths", 0.8, true},
		{"apm", 0.5, true},
		{"kda", 2.0, true},
		{"accuracy", 45.0, true},
		{"pspm", 250.0, true},
		{"dpm_damage", 400.0, true},
		{"unknown", 0, false},
	}
	for _, c := range cases {
		got, ok := getMetricValue(m, c.name)
		if ok != c.ok {
			t.Errorf("getMetricValue(%q): ok = %v, want %v", c.name, ok, c.ok)
		}
		if got != c.want {
			t.Errorf("getMetricValue(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestGetMetricValue_NilOptional(t *testing.T) {
	m := &matchMetrics{KPM: 1.0}
	_, ok := getMetricValue(m, "accuracy")
	if ok {
		t.Error("nil Accuracy should return ok=false")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// prepareHistoryMetrics
// ─────────────────────────────────────────────────────────────────────────────

func TestPrepareHistoryMetrics_AlwaysReturnsMap(t *testing.T) {
	// prepareHistoryMetrics always returns a map, even with empty history
	got := prepareHistoryMetrics(nil)
	if got == nil {
		t.Fatal("should return non-nil map")
	}
	if _, ok := got["kpm"]; !ok {
		t.Error("map should have kpm key")
	}
}

func TestPrepareHistoryMetrics_ExtractsCorrectly(t *testing.T) {
	history := make([]historyRow, 15)
	for i := range history {
		history[i] = historyRow{
			TimePlayedSeconds: 600,
			Kills:             float64(10 + i),
			Deaths:            5,
			Assists:           3,
			DamageDealt:       3000,
			PersonalScore:     2000,
			Accuracy:          40.0,
		}
	}
	got := prepareHistoryMetrics(history)
	kpms := got["kpm"]
	if len(kpms) != 15 {
		t.Errorf("kpm series: len = %d, want 15", len(kpms))
	}
	// Should be sorted
	for i := 1; i < len(kpms); i++ {
		if kpms[i] < kpms[i-1] {
			t.Errorf("kpm series not sorted at index %d", i)
			break
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// computeRelativePerformanceScore
// ─────────────────────────────────────────────────────────────────────────────

func TestComputeRelativePerformanceScore_NotEnoughHistory(t *testing.T) {
	current := &historyRow{TimePlayedSeconds: 600, Kills: 10, Deaths: 5}
	history := make([]historyRow, MinMatchesForRelative-1)
	for i := range history {
		history[i] = historyRow{TimePlayedSeconds: 600, Kills: 8, Deaths: 6}
	}
	got := computeRelativePerformanceScore(current, history, RelativeWeights)
	if got != nil {
		t.Errorf("not enough history: got %v, want nil", got)
	}
}

func TestComputeRelativePerformanceScore_AboveAverage(t *testing.T) {
	n := 20
	history := make([]historyRow, n)
	for i := range history {
		history[i] = historyRow{
			TimePlayedSeconds: 600,
			Kills:             float64(8 + i%5),
			Deaths:            float64(8),
			Assists:           3,
			DamageDealt:       3000,
			PersonalScore:     2000,
			Accuracy:          40.0,
		}
	}
	current := &historyRow{
		TimePlayedSeconds: 600,
		Kills:             25,
		Deaths:            3,
		Assists:           8,
		DamageDealt:       5000,
		PersonalScore:     4000,
		Accuracy:          55.0,
	}
	got := computeRelativePerformanceScore(current, history, RelativeWeights)
	if got == nil {
		t.Fatal("got nil, want non-nil score")
	}
	if *got <= 50 {
		t.Errorf("above-average: got %v, want > 50", *got)
	}
}

func TestComputeRelativePerformanceScore_BelowAverage(t *testing.T) {
	n := 20
	history := make([]historyRow, n)
	for i := range history {
		history[i] = historyRow{
			TimePlayedSeconds: 600,
			Kills:             15,
			Deaths:            5,
			Assists:           5,
			DamageDealt:       4000,
			PersonalScore:     3000,
			Accuracy:          50.0,
		}
	}
	current := &historyRow{
		TimePlayedSeconds: 600,
		Kills:             3,
		Deaths:            15,
		Assists:           1,
		DamageDealt:       1000,
		PersonalScore:     800,
		Accuracy:          20.0,
	}
	got := computeRelativePerformanceScore(current, history, RelativeWeights)
	if got == nil {
		t.Fatal("got nil, want non-nil score")
	}
	if *got >= 50 {
		t.Errorf("below-average: got %v, want < 50", *got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// computeRankPerformance
// ─────────────────────────────────────────────────────────────────────────────

func TestComputeRankPerformance_WithHistory(t *testing.T) {
	// Build a history with rank_perf_diff series
	histMetrics := map[string][]float64{
		"rank_perf_diff": {-3, -2, -1, 0, 1, 2, 3},
	}
	// rank=1, teamMMR=1500, enemyMMR=1500 → deltaMMR=0 → expectedRank=4.5 → diff=4.5-1=3.5
	got := computeRankPerformance(1, 1500, 1500, histMetrics)
	if got == nil {
		t.Fatal("got nil, want non-nil")
	}
	if *got < 50 {
		t.Errorf("rank 1 with equal MMR: got %v, want > 50", *got)
	}
}

func TestComputeRankPerformance_NoHistory(t *testing.T) {
	histMetrics := map[string][]float64{}
	got := computeRankPerformance(4, 1500, 1500, histMetrics)
	if got != nil {
		t.Errorf("no history: got %v, want nil", *got)
	}
}

func TestComputeRankPerformance_EmptySeries(t *testing.T) {
	histMetrics := map[string][]float64{
		"rank_perf_diff": {},
	}
	got := computeRankPerformance(4, 1500, 1500, histMetrics)
	if got != nil {
		t.Errorf("empty series: got %v, want nil", *got)
	}
}
