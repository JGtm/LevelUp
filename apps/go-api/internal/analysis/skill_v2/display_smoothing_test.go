package skill_v2

import "testing"

// TestTierOrdinal_RoundTrip vérifie que TierOrdinal et TierSubFromOrdinal sont
// inverses l'un de l'autre sur toute la grille (sous-paliers non uniformes).
func TestTierOrdinal_RoundTrip(t *testing.T) {
	bs := DefaultTierBoundaries()
	// total d'ordinaux = somme des sous-paliers (Onyx compté 1).
	total := 0
	for _, b := range bs {
		n := b.SubTiers
		if n < 1 {
			n = 1
		}
		total += n
	}
	for ord := 0; ord < total; ord++ {
		tier, sub := TierSubFromOrdinal(bs, ord)
		back := TierOrdinal(bs, tier.Name, sub)
		if back != ord {
			t.Errorf("ord=%d → %s sub=%d → ordinal %d (round-trip cassé)", ord, tier.Name, sub, back)
		}
	}
}

// TestTierOrdinal_Monotonic : l'ordinal croît avec μ.
func TestTierOrdinal_Monotonic(t *testing.T) {
	bs := DefaultTierBoundaries()
	prev := -1
	for mu := 0.0; mu < 30; mu += 0.1 {
		tier, sub := InferTier(mu, bs)
		ord := TierOrdinal(bs, tier.Name, sub)
		if ord < prev {
			t.Fatalf("μ=%.1f ordinal=%d < précédent %d (non monotone)", mu, ord, prev)
		}
		prev = ord
	}
}

func TestTierOrdinal_UnknownTier(t *testing.T) {
	if got := TierOrdinal(DefaultTierBoundaries(), "Mythic", 1); got != -1 {
		t.Errorf("tier inconnu → %d, want -1", got)
	}
}

// TestSmoothDisplayedOrdinal_PromotionImmediate : la montée s'applique tout de suite.
func TestSmoothDisplayedOrdinal_PromotionImmediate(t *testing.T) {
	got := SmoothDisplayedOrdinal(5 /*prev*/, 9 /*target*/, 50 /*exp post-placement*/)
	if got != 9 {
		t.Errorf("promotion : got %d, want 9 (immédiate)", got)
	}
}

// TestSmoothDisplayedOrdinal_DemotionRateLimited : la descente est bridée à 1/match.
func TestSmoothDisplayedOrdinal_DemotionRateLimited(t *testing.T) {
	// chute cible de 9 → 2 ; affiché ne descend que d'un cran.
	got := SmoothDisplayedOrdinal(9, 2, 50)
	if got != 8 {
		t.Errorf("descente bridée : got %d, want 8 (prev-1)", got)
	}
	// une chute d'exactement 1 cran passe telle quelle.
	if got := SmoothDisplayedOrdinal(9, 8, 50); got != 8 {
		t.Errorf("descente 1 cran : got %d, want 8", got)
	}
}

// TestSmoothDisplayedOrdinal_PlacementBypass : pendant le placement, pas de lissage.
func TestSmoothDisplayedOrdinal_PlacementBypass(t *testing.T) {
	for exp := 1; exp <= PlacementMatches; exp++ {
		if got := SmoothDisplayedOrdinal(9, 2, exp); got != 2 {
			t.Errorf("placement exp=%d : got %d, want 2 (cible, pas de bridage)", exp, got)
		}
	}
	// juste après le placement, le bridage reprend.
	if got := SmoothDisplayedOrdinal(9, 2, PlacementMatches+1); got != 8 {
		t.Errorf("post-placement : got %d, want 8 (bridé)", got)
	}
}

// TestSmoothDisplayedOrdinal_NoPrevious : sans précédent (-1), on affiche la cible.
func TestSmoothDisplayedOrdinal_NoPrevious(t *testing.T) {
	if got := SmoothDisplayedOrdinal(-1, 2, 50); got != 2 {
		t.Errorf("pas de précédent : got %d, want 2 (cible)", got)
	}
}

// TestSmoothDisplayedOrdinal_ConvergesDownOverMatches : sur une chute brutale de
// μ, l'affiché rejoint la cible en N matchs (1 cran/match), sans à-coup.
func TestSmoothDisplayedOrdinal_ConvergesDownOverMatches(t *testing.T) {
	target := 3
	disp := 12
	steps := 0
	for disp != target {
		disp = SmoothDisplayedOrdinal(disp, target, 50)
		steps++
		if steps > 50 {
			t.Fatal("ne converge pas")
		}
	}
	if steps != 9 { // 12 → 3 = 9 crans, 1/match
		t.Errorf("convergence en %d matchs, want 9", steps)
	}
}

func TestFormatTierSubLabel(t *testing.T) {
	bs := DefaultTierBoundaries()
	var gold, onyx, plat TierBoundary
	for _, b := range bs {
		switch b.Name {
		case "Gold":
			gold = b
		case "Onyx":
			onyx = b
		case "Platinum":
			plat = b
		}
	}
	if got := FormatTierSubLabel(gold, 3); got != "Or III" {
		t.Errorf("Or III : got %q", got)
	}
	if got := FormatTierSubLabel(plat, 2); got != "Platine II" {
		t.Errorf("Platine II : got %q", got)
	}
	if got := FormatTierSubLabel(onyx, 0); got != "Onyx" {
		t.Errorf("Onyx : got %q", got)
	}
	if got := FormatTierSubLabel(TierBoundary{}, 1); got != "Non classé" {
		t.Errorf("tier vide : got %q", got)
	}
}

// TestMapTierSubToLegacyRating cohérent avec mapMuToLegacyRating pour un μ donné.
func TestMapTierSubToLegacyRating_ConsistentWithMu(t *testing.T) {
	bs := DefaultTierBoundaries()
	for _, mu := range []float64{20.38, 23.52, 24.0, 25.5, 26.17} {
		tier, sub := InferTier(mu, bs)
		viaTierSub := mapTierSubToLegacyRating(tier, sub)
		viaMu := mapMuToLegacyRating(mu, bs)
		if viaTierSub != viaMu {
			t.Errorf("μ=%v : MapTierSub=%v != MapMu=%v", mu, viaTierSub, viaMu)
		}
	}
}
