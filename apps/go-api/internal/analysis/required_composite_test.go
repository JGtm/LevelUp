package analysis

import (
	"math"
	"testing"
)

// TestRequiredCompositeForTier_StableAtCurrent : si target=current, la fonction
// retourne 0.5 (composite stable).
func TestRequiredCompositeForTier_StableAtCurrent(t *testing.T) {
	got := RequiredCompositeForTier(1500, 1500, 150)
	if math.Abs(got-0.5) > 0.01 {
		t.Errorf("RequiredCompositeForTier(1500, 1500, 150) = %.3f, want 0.5", got)
	}
}

// TestRequiredCompositeForTier_NextSubTier : sub-tier suivant (+33.3 pts), la
// composite requise est > 0.5 mais clampée à 1.0 car le delta dépasse kElo/2.
func TestRequiredCompositeForTier_NextSubTier(t *testing.T) {
	// Gold IV (1500) → Gold V (1533.33). Delta = 33.33 > kElo/2 (16). Clampé à 1.0.
	got := RequiredCompositeForTier(1500, 1533.33, 150)
	if got != 1.0 {
		t.Errorf("RequiredCompositeForTier(1500, 1533.33, 150) = %.3f, want 1.0 (clamp)", got)
	}
}

// TestRequiredCompositeForTier_ReachableSmallDelta : pour un petit delta
// atteignable (10 pts ≈ kElo/3), la composite doit être entre 0.5 et 1.0.
func TestRequiredCompositeForTier_ReachableSmallDelta(t *testing.T) {
	got := RequiredCompositeForTier(1500, 1510, 150)
	if got <= 0.5 || got >= 1.0 {
		t.Errorf("RequiredCompositeForTier(1500, 1510, 150) = %.3f, want in (0.5, 1.0)", got)
	}
	// Vérifier la cohérence : trueskillUpdate(mu=1500, composite=got) ≈ 1510.
	muAfter, _ := trueskillUpdate(1500, 150, 1500, 150, got, 1.0)
	if math.Abs(muAfter-1510) > 0.5 {
		t.Errorf("trueskillUpdate avec composite=%.3f donne mu=%.2f, want ~1510", got, muAfter)
	}
}

// TestRequiredCompositeForTier_TargetBelow : si target < current, composite < 0.5.
func TestRequiredCompositeForTier_TargetBelow(t *testing.T) {
	got := RequiredCompositeForTier(1500, 1490, 150)
	if got >= 0.5 {
		t.Errorf("RequiredCompositeForTier(1500, 1490, 150) = %.3f, want < 0.5", got)
	}
}

// TestRequiredCompositeForTier_FarBelow : target très inférieur → 0.0 (clamp).
func TestRequiredCompositeForTier_FarBelow(t *testing.T) {
	got := RequiredCompositeForTier(1500, 1400, 150)
	if got != 0.0 {
		t.Errorf("RequiredCompositeForTier(1500, 1400, 150) = %.3f, want 0.0 (clamp)", got)
	}
}

// TestRequiredCompositeForTier_AllTiersEntry : pour chaque tier (point d'entrée),
// la composite requise depuis 50 pts en-dessous doit être > 0.5 (et probablement
// clampée à 1.0 car 50 > kElo/2).
func TestRequiredCompositeForTier_AllTiersEntry(t *testing.T) {
	tierEntries := []struct {
		name      string
		current   float64
		targetMu  float64
	}{
		{"Bronze entry from 950", 950, 1000},
		{"Silver entry from 1150", 1150, 1200},
		{"Gold entry from 1350", 1350, 1400},
		{"Platinum entry from 1550", 1550, 1600},
		{"Diamond entry from 1750", 1750, 1800},
		{"Onyx entry from 1950", 1950, 2000},
	}
	for _, tc := range tierEntries {
		t.Run(tc.name, func(t *testing.T) {
			got := RequiredCompositeForTier(tc.current, tc.targetMu, 150)
			if got < 0.5 {
				t.Errorf("composite = %.3f, want > 0.5 for upward delta", got)
			}
		})
	}
}

// TestClamp01_Bounds : helper privé clamp01 sur valeurs limites.
func TestClamp01_Bounds(t *testing.T) {
	cases := []struct {
		in, want float64
	}{
		{-1, 0},
		{0, 0},
		{0.5, 0.5},
		{1, 1},
		{2, 1},
	}
	for _, c := range cases {
		if got := clamp01(c.in); got != c.want {
			t.Errorf("clamp01(%.2f) = %.2f, want %.2f", c.in, got, c.want)
		}
	}
}
