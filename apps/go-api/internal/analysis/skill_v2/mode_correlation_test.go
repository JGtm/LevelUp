package skill_v2

import (
	"math"
	"testing"
)

func TestApplyCrossModeLeak_LinearShift(t *testing.T) {
	// Player joue slayer, μ passe de 25 → 26.5 (delta +1.5).
	// Avec w_d = 0.3, son μ en chaos devrait monter de 0.3 · 1.5 = 0.45.
	got := ApplyCrossModeLeak(24.0, 25.0, 26.5, 0.3)
	want := 24.0 + 0.45
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("ApplyCrossModeLeak = %v, want %v", got, want)
	}
}

func TestApplyCrossModeLeak_NegativeDelta(t *testing.T) {
	// Match perdu, μ slayer baisse de 25 → 23 (delta -2.0).
	// Avec w_d = 0.3, μ chaos doit baisser de 0.3 · 2 = 0.6.
	got := ApplyCrossModeLeak(24.0, 25.0, 23.0, 0.3)
	want := 24.0 - 0.6
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("ApplyCrossModeLeak = %v, want %v", got, want)
	}
}

func TestApplyCrossModeLeak_ClampsAtMax(t *testing.T) {
	// Tentative d'utiliser w_d = 0.8 → DOIT être clampé à 0.4.
	got := ApplyCrossModeLeak(24.0, 25.0, 30.0, 0.8) // delta +5
	want := 24.0 + 0.4*5.0                           // clampé à 0.4 → +2.0
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("ApplyCrossModeLeak (over-cap) = %v, want %v (clamped)", got, want)
	}
}

func TestApplyCrossModeLeak_NegativeWeightTreatedAsZero(t *testing.T) {
	// w_d < 0 doit être clampé à 0 (pas d'inversion de signe).
	got := ApplyCrossModeLeak(24.0, 25.0, 30.0, -0.5)
	want := 24.0 // pas de leak
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("ApplyCrossModeLeak (negative weight) = %v, want %v", got, want)
	}
}

func TestApplyCrossModeLeak_ZeroWeight(t *testing.T) {
	// w_d = 0 → pas de leak (mode totalement décorrélé).
	got := ApplyCrossModeLeak(24.0, 25.0, 30.0, 0.0)
	if math.Abs(got-24.0) > 1e-9 {
		t.Errorf("ApplyCrossModeLeak(w=0) = %v, want 24.0", got)
	}
}

func TestDefaultModeCouplingWeight_InRange(t *testing.T) {
	if DefaultModeCouplingWeight < 0 || DefaultModeCouplingWeight > Phase4ModeCouplingMaxWeight {
		t.Errorf("DefaultModeCouplingWeight = %v hors [0, %v]", DefaultModeCouplingWeight, Phase4ModeCouplingMaxWeight)
	}
}
