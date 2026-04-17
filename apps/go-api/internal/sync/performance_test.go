//go:build integration

// Package sync — performance_test.go : tests pour les calculs de performance.
//
// Sprint 47 T14 — couvrir les fonctions pures : percentileRank, extractMatchMetrics,
// computeRelativePerformanceScore, prepareHistoryMetrics.
// Note : ce package importe DuckDB transitif → ne compile pas sur Windows.
// Lancer avec : go test -tags=integration ./internal/sync/ -run TestPercentile -v
package sync

import (
	"math"
	"testing"
)

// ── Tests percentileRank ─────────────────────────────────────────────────────

func TestPercentileRank_Empty(t *testing.T) {
	// < 2 éléments → 50.0 par défaut
	if r := percentileRank(1.0, nil); r != 50.0 {
		t.Errorf("attendu 50.0, obtenu %v", r)
	}
	if r := percentileRank(1.0, []float64{5.0}); r != 50.0 {
		t.Errorf("attendu 50.0, obtenu %v", r)
	}
}

func TestPercentileRank_Median(t *testing.T) {
	series := []float64{10, 20, 30, 40, 50}
	// 30 est à la médiane
	r := percentileRank(30, series)
	if r < 50 || r > 70 {
		t.Errorf("percentile de la médiane attendu ~60%%, obtenu %.1f", r)
	}
}

func TestPercentileRank_Min(t *testing.T) {
	series := []float64{10, 20, 30, 40, 50}
	r := percentileRank(5, series) // inférieur au minimum
	if r != 0.0 {
		t.Errorf("attendu 0.0 pour valeur inférieure au min, obtenu %v", r)
	}
}

func TestPercentileRank_Max(t *testing.T) {
	series := []float64{10, 20, 30, 40, 50}
	r := percentileRank(100, series) // supérieur au maximum
	if r != 100.0 {
		t.Errorf("attendu 100.0 pour valeur supérieure au max, obtenu %v", r)
	}
}

func TestPercentileRankInverse_HighValueLowPercent(t *testing.T) {
	// Inverse : plus la valeur est haute, plus le percentile est bas
	series := []float64{1, 2, 3, 4, 5}
	rHigh := percentileRankInverse(5, series)
	rLow := percentileRankInverse(1, series)
	if rHigh >= rLow {
		t.Errorf("percentile inversé : rHigh(%v) devrait < rLow(%v)", rHigh, rLow)
	}
}

// ── Tests extractMatchMetrics ────────────────────────────────────────────────

func TestExtractMatchMetrics_Basic(t *testing.T) {
	row := &historyRow{
		MatchID:           "m1",
		Kills:             10,
		Deaths:            2,
		Assists:           3,
		TimePlayedSeconds: 600, // 10 minutes
		KDA:               0,   // calculé automatiquement
	}
	m := extractMatchMetrics(row)

	if m == nil {
		t.Fatal("extractMatchMetrics retourne nil")
	}
	// KPM = 10/10 = 1.0
	if math.Abs(m.KPM-1.0) > 0.001 {
		t.Errorf("KPM attendu 1.0, obtenu %v", m.KPM)
	}
	// DPMDeaths = 2/10 = 0.2
	if math.Abs(m.DPMDeaths-0.2) > 0.001 {
		t.Errorf("DPMDeaths attendu 0.2, obtenu %v", m.DPMDeaths)
	}
	// KDA auto = (10+3)/max(1,2) = 6.5
	if math.Abs(m.KDA-6.5) > 0.001 {
		t.Errorf("KDA attendu 6.5, obtenu %v", m.KDA)
	}
}

func TestExtractMatchMetrics_ZeroDuration(t *testing.T) {
	// duration=0 → fallback 600s
	row := &historyRow{Kills: 5, Deaths: 1, TimePlayedSeconds: 0}
	m := extractMatchMetrics(row)
	// KPM = 5/10 = 0.5
	if math.Abs(m.KPM-0.5) > 0.001 {
		t.Errorf("KPM avec duration=0 attendu 0.5, obtenu %v", m.KPM)
	}
}

func TestExtractMatchMetrics_WithOptionalFields(t *testing.T) {
	accuracy := 0.45
	score := 1200.0
	row := &historyRow{
		Kills:             8,
		Deaths:            3,
		TimePlayedSeconds: 600,
		Accuracy:          accuracy,
		PersonalScore:     score,
		DamageDealt:       3000,
		TeamMMR:           1500,
		EnemyMMR:          1450,
	}
	m := extractMatchMetrics(row)

	if m.Accuracy == nil || math.Abs(*m.Accuracy-accuracy) > 0.001 {
		t.Errorf("Accuracy attendu %v, obtenu %v", accuracy, m.Accuracy)
	}
	if m.PSPM == nil {
		t.Error("PSPM devrait être non-nil")
	}
	if m.DPMDamage == nil {
		t.Error("DPMDamage devrait être non-nil")
	}
	if m.TeamMMR == nil {
		t.Error("TeamMMR devrait être non-nil")
	}
}

// ── Tests prepareHistoryMetrics ──────────────────────────────────────────────

func TestPrepareHistoryMetrics_EmptySlice(t *testing.T) {
	metrics := prepareHistoryMetrics(nil)
	if metrics == nil {
		t.Fatal("prepareHistoryMetrics(nil) ne devrait pas retourner nil")
	}
	if _, ok := metrics["KPM"]; !ok {
		t.Error("prepareHistoryMetrics: clé KPM manquante même sur slice vide")
	}
}

func TestPrepareHistoryMetrics_WithRows(t *testing.T) {
	rows := []historyRow{
		{Kills: 10, Deaths: 2, Assists: 3, TimePlayedSeconds: 600},
		{Kills: 5, Deaths: 5, Assists: 1, TimePlayedSeconds: 600},
		{Kills: 15, Deaths: 1, Assists: 5, TimePlayedSeconds: 600},
	}
	metrics := prepareHistoryMetrics(rows)

	if len(metrics["KPM"]) != 3 {
		t.Errorf("attendu 3 entrées KPM, obtenu %d", len(metrics["KPM"]))
	}
	if len(metrics["KDA"]) != 3 {
		t.Errorf("attendu 3 entrées KDA, obtenu %d", len(metrics["KDA"]))
	}
}
