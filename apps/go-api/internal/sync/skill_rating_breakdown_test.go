package sync

import (
	"testing"
)

// Tests unitaires de computeCompositeScoreWithBreakdown (V2 §1).
// Vérifie que le breakdown map contient les composantes calculées.

func TestCompositeScoreBreakdown_AllZeros_EmptyBreakdown(t *testing.T) {
	row := &compositeMatchRow{}
	composite, breakdown := computeCompositeScoreWithBreakdown(row, nil, nil, nil, nil, nil, nil)
	if composite != 0.5 {
		t.Errorf("composite: got %.2f, want 0.5", composite)
	}
	if len(breakdown) != 0 {
		t.Errorf("breakdown: got %d entries, want 0 (no inputs valid)", len(breakdown))
	}
}

func TestCompositeScoreBreakdown_ClearWin_BreakdownPopulated(t *testing.T) {
	outcome := 2 // WIN
	row := &compositeMatchRow{
		Kills:          20,
		Deaths:         5,
		KillsExpected:  10,
		DeathsExpected: 10,
		Outcome:        &outcome,
		DamageDealt:    5000,
		DamageTaken:    2000,
		Accuracy:       55.0,
	}
	avgAcc := 45.0
	avgDE := 0.5
	composite, breakdown := computeCompositeScoreWithBreakdown(row, &avgAcc, nil, &avgDE, nil, nil, nil)

	if composite <= 0.5 {
		t.Errorf("composite: got %.2f, want > 0.5", composite)
	}

	// 5 composantes attendues (les inputs sont fournis pour 5 sur 8) :
	// kills_vs_expected, deaths_vs_expected, win_factor, damage_efficiency,
	// accuracy_delta. Pas medal_exploit / offensive_conversion /
	// defensive_resistance (champs à 0).
	expectedKeys := []string{
		"kills_vs_expected",
		"deaths_vs_expected",
		"win_factor",
		"damage_efficiency",
		"accuracy_delta",
	}
	for _, k := range expectedKeys {
		if _, ok := breakdown[k]; !ok {
			t.Errorf("breakdown missing key %q", k)
		}
	}
	if len(breakdown) != len(expectedKeys) {
		t.Errorf("breakdown len: got %d (%v), want %d", len(breakdown), keysOf(breakdown), len(expectedKeys))
	}

	// Toutes les valeurs doivent être dans [0,1].
	for k, v := range breakdown {
		if v < 0 || v > 1 {
			t.Errorf("breakdown[%q] = %.3f out of [0,1]", k, v)
		}
	}
}

func TestCompositeScoreBreakdown_AllEight_Populated(t *testing.T) {
	outcome := 2
	row := &compositeMatchRow{
		Kills:               15,
		Deaths:              8,
		KillsExpected:       10,
		DeathsExpected:      10,
		Outcome:             &outcome,
		DamageDealt:         4000,
		DamageTaken:         3000,
		Accuracy:            50.0,
		MedalExploitScore:   8.0,
		OffensiveConversion: 0.6,
		DefensiveResistance: 0.4,
	}
	avgAcc := 45.0
	avgDE := 0.5
	_, breakdown := computeCompositeScoreWithBreakdown(row, &avgAcc, nil, &avgDE, nil, nil, nil)
	all := []string{
		"kills_vs_expected", "deaths_vs_expected", "win_factor", "damage_efficiency",
		"accuracy_delta", "medal_exploit", "offensive_conversion", "defensive_resistance",
	}
	if len(breakdown) != 8 {
		t.Errorf("breakdown: got %d entries (%v), want 8", len(breakdown), keysOf(breakdown))
	}
	for _, k := range all {
		if _, ok := breakdown[k]; !ok {
			t.Errorf("breakdown missing key %q", k)
		}
	}
}

// TestCompositeScore_BackwardCompat — le wrapper rétro-compatible
// computeCompositeScore retourne le même composite que la version breakdown.
func TestCompositeScore_BackwardCompat(t *testing.T) {
	outcome := 2
	row := &compositeMatchRow{
		Kills: 15, Deaths: 8, KillsExpected: 10, DeathsExpected: 10,
		Outcome: &outcome, DamageDealt: 4000, DamageTaken: 3000, Accuracy: 50,
	}
	avgAcc := 45.0
	avgDE := 0.5
	wrapped := computeCompositeScore(row, &avgAcc, nil, &avgDE, nil, nil, nil)
	full, _ := computeCompositeScoreWithBreakdown(row, &avgAcc, nil, &avgDE, nil, nil, nil)
	if wrapped != full {
		t.Errorf("wrapper diverged: wrapped=%.4f full=%.4f", wrapped, full)
	}
}

func keysOf(m map[string]float64) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
