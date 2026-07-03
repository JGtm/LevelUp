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
			got := mapMuToLegacyRating(c.mu, bs)
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
		legacy := mapMuToLegacyRating(mu, bs)
		v1Min, v1Max := LegacyTierRange(v2Tier.Name)
		if v2Tier.Name != "Onyx" && (legacy < v1Min || legacy >= v1Max) {
			t.Errorf("μ=%v v2_tier=%s mapped to legacy=%v, hors range v1 [%v, %v[",
				mu, v2Tier.Name, legacy, v1Min, v1Max)
		}
	}
}

func TestMapMuToContinuousRating_Monotonic(t *testing.T) {
	bs := DefaultTierBoundaries()
	prev := math.Inf(-1)
	for mu := 0.0; mu <= 30; mu += 0.05 {
		got := MapMuToContinuousRating(mu, bs)
		if got < prev-1e-9 {
			t.Fatalf("non monotone à μ=%v : %v < %v", mu, got, prev)
		}
		prev = got
	}
}

func TestMapMuToContinuousRating_ContinuousAtTierBoundaries(t *testing.T) {
	bs := DefaultTierBoundaries()
	// À chaque frontière de tier, la valeur juste en-dessous et pile à la borne
	// doivent être quasi-égales (continuité C0 — pas de saut de palier).
	for _, b := range bs[1:] { // skip Bronze (pas de frontière basse)
		below := MapMuToContinuousRating(b.MinMu-1e-4, bs)
		at := MapMuToContinuousRating(b.MinMu, bs)
		if math.Abs(at-below) > 1.0 {
			t.Errorf("discontinuité à μ=%v (%s) : below=%v at=%v", b.MinMu, b.Name, below, at)
		}
	}
}

func TestMapMuToContinuousRating_WithinTierRange(t *testing.T) {
	bs := DefaultTierBoundaries()
	mus := []float64{20.38, 23.52, 23.81, 25.75, 26.12, 26.17, 27.5}
	for _, mu := range mus {
		tier, _ := InferTier(mu, bs)
		got := MapMuToContinuousRating(mu, bs)
		min, max := LegacyTierRange(tier.Name)
		if got < min-1e-9 || got > max+1e-9 {
			t.Errorf("μ=%v (%s) → %v hors plage tier [%v, %v]", mu, tier.Name, got, min, max)
		}
	}
}

func TestMapMuToContinuousRating_DiffersFromQuantized(t *testing.T) {
	// Cœur du fix : deux μ DIFFÉRENTS dans le MÊME sous-palier donnent des valeurs
	// continues DIFFÉRENTES (≠ mapMuToLegacyRating qui les écrase au même bas de
	// sous-palier → rating_delta resterait 0). Diamant D3 = [26.6, 27[.
	bs := DefaultTierBoundaries()
	muA, muB := 26.65, 26.85
	qA, qB := mapMuToLegacyRating(muA, bs), mapMuToLegacyRating(muB, bs)
	if qA != qB {
		t.Fatalf("pré-condition : μ choisis pas dans le même sous-palier quantifié (qA=%v qB=%v)", qA, qB)
	}
	cA, cB := MapMuToContinuousRating(muA, bs), MapMuToContinuousRating(muB, bs)
	if cA == cB {
		t.Errorf("continu identique entre μ=%v et μ=%v (=%v) — le delta resterait 0", muA, muB, cA)
	}
	if cB <= cA {
		t.Errorf("continu non croissant : μ=%v→%v vs μ=%v→%v", muA, cA, muB, cB)
	}
}

func TestLegacySubTierRange(t *testing.T) {
	bs := DefaultTierBoundaries()
	get := func(name string) TierBoundary {
		for _, b := range bs {
			if b.Name == name {
				return b
			}
		}
		t.Fatalf("tier %s introuvable", name)
		return TierBoundary{}
	}
	// Onyx : sous-palier unique → plage complète du tier.
	if min, max := LegacySubTierRange(get("Onyx"), 0); min != 2000 || max != 2200 {
		t.Errorf("Onyx → [%v,%v], want [2000,2200]", min, max)
	}
	// Gold [1400,1600], 6 sous-paliers → band 33.33.
	if min, max := LegacySubTierRange(get("Gold"), 1); math.Abs(min-1400) > 1e-9 || math.Abs(max-1433.333) > 0.01 {
		t.Errorf("Gold sub1 → [%v,%v], want [1400,1433.33]", min, max)
	}
	if min, _ := LegacySubTierRange(get("Gold"), 5); math.Abs(min-1533.333) > 0.01 {
		t.Errorf("Gold sub5 min = %v, want ≈1533.33", min)
	}
}

func TestLegacyContinuousSubTierProgress(t *testing.T) {
	cases := []struct {
		name    string
		rv      float64
		wantPct float64
		wantOK  bool
	}{
		// Gold [1400,1600], 6 sous-paliers (band 33.33).
		{"Gold I début", 1400, 0.0, true},
		{"Gold I milieu", 1416.667, 0.5, true},
		{"Gold V milieu (1550)", 1550, 0.5, true},
		// Diamant [1800,2000], 3 sous-paliers (band 66.67). On évite les bornes
		// inter-sous-paliers non représentables (1866.6…/1933.3…) — ambiguës au
		// floor — et on teste une borne entière + un milieu.
		{"Diamant I début (borne tier)", 1800, 0.0, true},
		{"Diamant III milieu", 1966.667, 0.5, true},
		// Onyx [2000,2200[ — sous-palier unique → progression sur toute la bande.
		{"Onyx milieu", 2100, 0.5, true},
		// Hors grille.
		{"sous Bronze", 900, 0.0, false},
		{"au-dessus Onyx → plein", 2300, 1.0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pct, ok := LegacyContinuousSubTierProgress(c.rv)
			if ok != c.wantOK {
				t.Fatalf("rv=%v → ok=%v, want %v", c.rv, ok, c.wantOK)
			}
			if ok && math.Abs(pct-c.wantPct) > 0.01 {
				t.Errorf("rv=%v → pct=%.4f, want ≈%.4f", c.rv, pct, c.wantPct)
			}
		})
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
