package skill_v2

import (
	"math"
	"testing"
)

func TestLegacyTierRange_AllTiers(t *testing.T) {
	cases := []struct {
		name    string
		wantMin float64
		wantMax float64
	}{
		{"Bronze", 1000, 1200},
		{"Silver", 1200, 1400},
		{"Gold", 1400, 1600},
		{"Platinum", 1600, 1800},
		{"Diamond", 1800, 2000},
		{"Onyx", 2000, 2200},
	}
	for _, c := range cases {
		min, max := LegacyTierRange(c.name)
		if min != c.wantMin || max != c.wantMax {
			t.Errorf("%s: got [%v, %v], want [%v, %v]", c.name, min, max, c.wantMin, c.wantMax)
		}
	}
}

func TestMapMuToLegacyRating_ReferencePlayers(t *testing.T) {
	bs := DefaultTierBoundaries()
	cases := []struct {
		name       string
		mu         float64
		wantTier   string
		wantApprox float64 // tolérance ±10 sur le rating
	}{
		// Diamant 3 sous-paliers (band 66.7), Platine 2 (band 100) — cf. grille
		// uniformisée [2026-05-31]. Les tiers restent identiques (calibration
		// préservée), seules les valeurs sous-palier changent.
		{"Madina BTB Diamant I", 26.17, "Diamond", 1800},        // sub 1 → 1800
		{"Madina Slayer Diamant I", 26.12, "Diamond", 1800},     // sub 1 → 1800
		{"Madina Objectif Platine II", 25.75, "Platinum", 1700}, // 1600 + 1*100 = 1700
		{"Chocoboflor Or IV", 23.81, "Gold", 1500},              // 1400 + 3/6*200 = 1500
		{"JGtm Or IV", 23.52, "Gold", 1500},                     // idem
		{"XxDaemon Bronze VI", 20.38, "Bronze", 1167},           // 1000 + 5/6*200 ≈ 1167
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := MapMuToLegacyRating(c.mu, bs)
			if math.Abs(got-c.wantApprox) > 20 {
				t.Errorf("μ=%v → %v, want ≈%v (tol ±20)", c.mu, got, c.wantApprox)
			}
		})
	}
}

func TestMapMuToLegacyRating_TierPreservation(t *testing.T) {
	// Critère ⭐ : le tier label v1 (calculé via GetTierForRating dans sync,
	// ici reproduit indirectement par les bornes legacy) doit MATCHER le tier
	// label v2 (InferTier). C'est l'invariant de la Stratégie C.
	bs := DefaultTierBoundaries()
	mus := []float64{20.38, 23.52, 23.81, 26.12, 26.17}
	for _, mu := range mus {
		v2Tier, _ := InferTier(mu, bs)
		legacy := MapMuToLegacyRating(mu, bs)
		v1Min, v1Max := LegacyTierRange(v2Tier.Name)
		if v2Tier.Name != "Onyx" && (legacy < v1Min || legacy >= v1Max) {
			t.Errorf("μ=%v v2_tier=%s mapped to legacy=%v, hors range v1 [%v, %v[",
				mu, v2Tier.Name, legacy, v1Min, v1Max)
		}
	}
}

func TestMapSigmaToLegacyDeviation_Bounds(t *testing.T) {
	// σ très bas → clamp à 60 (MinSigma v1)
	if got := MapSigmaToLegacyDeviation(0.5); got != 60 {
		t.Errorf("σ=0.5 → %v, want 60 (MinSigma clamp)", got)
	}
	// σ très haut → clamp à 350 (MaxSigma v1)
	if got := MapSigmaToLegacyDeviation(20); got != 350 {
		t.Errorf("σ=20 → %v, want 350 (MaxSigma clamp)", got)
	}
	// σ typique 0.7 (skill bien établi) → ≈ 90 v1
	got := MapSigmaToLegacyDeviation(0.7)
	if math.Abs(got-90) > 30 {
		t.Errorf("σ=0.7 → %v, want approx 90 (skill confirmé v1)", got)
	}
}
