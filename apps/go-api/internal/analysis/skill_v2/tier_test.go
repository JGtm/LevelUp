package skill_v2

import "testing"

func TestDefaultTierBoundaries_Ordered(t *testing.T) {
	bs := DefaultTierBoundaries()
	if len(bs) != 6 {
		t.Fatalf("expected 6 tiers, got %d", len(bs))
	}
	for i := 1; i < len(bs); i++ {
		if bs[i].MinMu <= bs[i-1].MinMu {
			t.Errorf("tier %s MinMu=%v not strictly > previous %s MinMu=%v",
				bs[i].Name, bs[i].MinMu, bs[i-1].Name, bs[i-1].MinMu)
		}
	}
}

func TestInferTier_ReferencePlayers(t *testing.T) {
	// Vérifie que les 4 joueurs trackés (Phase 3d v3) tombent dans leurs tiers
	// cibles. Critère ⭐ de Phase 3e v2 — c'est pour ça qu'on a calibré les seuils.
	//
	// Grille B v2 (post-feedback utilisateur) : Bronze large, Or = bulk,
	// Platine étroit, Diamant large pour matcher les CSR connus.
	bs := DefaultTierBoundaries()
	cases := []struct {
		name     string
		mu       float64
		wantTier string
	}{
		{"Madina BTB", 26.17, "Diamond"},                  // cible Diamant ✓ (CSR Diamant 4-5)
		{"Madina arena_slayer", 26.12, "Diamond"},         // cible Diamant ✓
		{"Madina arena_objectif", 25.75, "Platinum"},      // proche Diamant, dans Platine étroit
		{"Chocoboflor arena_slayer", 23.81, "Gold"},       // cible Or ✓
		{"JGtm arena_slayer", 23.52, "Gold"},              // cible Or ✓
		{"XxDaemonGamerxX arena_slayer", 20.38, "Bronze"}, // cible Bronze ✓
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tier, _ := InferTier(c.mu, bs)
			if tier.Name != c.wantTier {
				t.Errorf("InferTier(%v) = %s, want %s", c.mu, tier.Name, c.wantTier)
			}
		})
	}
}

func TestInferTier_Boundaries(t *testing.T) {
	bs := DefaultTierBoundaries()
	// Tests sur les bornes exactes (grille Phase 3e v2).
	// Bronze [0, 21[, Argent [21, 22[, Or [22, 25[, Platine [25, 25.8[,
	// Diamant [25.8, 27[, Onyx [27, ∞[.
	cases := []struct {
		mu       float64
		wantTier string
	}{
		{-100, "Bronze"}, // très bas → Bronze
		{0, "Bronze"},
		{20.99, "Bronze"}, // juste sous Silver
		{21.0, "Silver"},  // borne inclusive
		{21.99, "Silver"},
		{22.0, "Gold"},
		{24.99, "Gold"},
		{25.0, "Platinum"},
		{25.79, "Platinum"},
		{25.8, "Diamond"},
		{26.99, "Diamond"},
		{27.0, "Onyx"},
		{100.0, "Onyx"}, // très haut → Onyx
	}
	for _, c := range cases {
		tier, _ := InferTier(c.mu, bs)
		if tier.Name != c.wantTier {
			t.Errorf("μ=%v → %s, want %s", c.mu, tier.Name, c.wantTier)
		}
	}
}

func TestInferTier_SubTiers_DistributesEvenly(t *testing.T) {
	bs := DefaultTierBoundaries()
	// Or = [22.0, 25.0[ (grille v2), 6 sous-tiers → width = 0.5 par sous-tier.
	// μ = 22.0 → sub 1 ; μ = 22.5 → sub 2 ; μ = 23.0 → sub 3 ; etc.
	cases := []struct {
		mu      float64
		wantSub int
	}{
		{22.0, 1},
		{22.49, 1},
		{22.5, 2},
		{23.0, 3},
		{23.52, 4}, // JGtm
		{24.0, 5},
		{24.5, 6},
		{24.99, 6},
	}
	for _, c := range cases {
		tier, sub := InferTier(c.mu, bs)
		if tier.Name != "Gold" {
			t.Errorf("μ=%v not Gold (%s)", c.mu, tier.Name)
			continue
		}
		if sub != c.wantSub {
			t.Errorf("μ=%v Or → sub %d, want %d", c.mu, sub, c.wantSub)
		}
	}
}

func TestInferTier_Onyx_NoSubTier(t *testing.T) {
	bs := DefaultTierBoundaries()
	for _, mu := range []float64{27.0, 28.5, 35.0, 50.0} {
		tier, sub := InferTier(mu, bs)
		if tier.Name != "Onyx" {
			t.Errorf("μ=%v should be Onyx", mu)
		}
		if sub != 0 {
			t.Errorf("Onyx sub-tier should be 0, got %d", sub)
		}
	}
}

func TestFormatTierLabel(t *testing.T) {
	bs := DefaultTierBoundaries()
	// Convention Halo CSR : sub I (1) = bas du tier, sub VI (6) = haut.
	// Donc μ proche du seuil supérieur du tier produit un sub élevé.
	//
	// Grille v2 (sous-paliers uniformisés [2026-05-31]) :
	// Bronze [0, 21[ × 6 = 3.5 μ/sous-tier
	// Or [22, 25[ × 6 = 0.5 μ/sous-tier
	// Platine [25, 25.8[ × 2 = 0.4 μ/sous-tier
	// Diamant [25.8, 27[ × 3 = 0.4 μ/sous-tier
	cases := []struct {
		mu   float64
		want string
	}{
		{1.0, "Bronze I"},
		{20.0, "Bronze VI"}, // (20-0)/3.5+1 = 5.71+1 → clamp 6
		{22.0, "Or I"},      // borne inclusive
		{22.5, "Or II"},
		{23.52, "Or IV"}, // JGtm
		{24.99, "Or VI"},
		{25.0, "Platine I"},  // Platine 2 sous-paliers (width 0.4)
		{25.5, "Platine II"}, // (25.5-25)/0.4 = 1.25 → 2
		{25.8, "Diamant I"},  // borne Diamant (3 sous-paliers, width 0.4)
		{26.0, "Diamant I"},  // (26-25.8)/0.4 = 0.5 → 1
		{26.17, "Diamant I"}, // Madina BTB → (26.17-25.8)/0.4 = 0.925 → 1
		{26.5, "Diamant II"}, // (26.5-25.8)/0.4 = 1.75 → 2
		{27.0, "Onyx"},
		{45.0, "Onyx"},
	}
	for _, c := range cases {
		if got := FormatTierLabel(c.mu, bs); got != c.want {
			t.Errorf("FormatTierLabel(%v) = %q, want %q", c.mu, got, c.want)
		}
	}
}

func TestTierBoundariesFromHyperparams_OverridesDefaults(t *testing.T) {
	hp := map[string]float64{
		"tier_boundary_gold":     22.5, // au lieu de 22.0
		"tier_boundary_platinum": 25.5, // au lieu de 25.0
	}
	bs := TierBoundariesFromHyperparams(hp)
	for _, b := range bs {
		switch b.Name {
		case "Gold":
			if b.MinMu != 22.5 {
				t.Errorf("Gold override échoué : %v", b.MinMu)
			}
		case "Platinum":
			if b.MinMu != 25.5 {
				t.Errorf("Platinum override échoué : %v", b.MinMu)
			}
		case "Silver":
			if b.MinMu != 21.0 {
				t.Errorf("Silver default écrasé : %v (attendu 21.0)", b.MinMu)
			}
		}
	}
}

func TestTierBoundariesFromHyperparams_EmptyDefaults(t *testing.T) {
	// hp vide → identique aux defaults.
	bs1 := TierBoundariesFromHyperparams(nil)
	bs2 := DefaultTierBoundaries()
	if len(bs1) != len(bs2) {
		t.Fatalf("len diff")
	}
	for i := range bs1 {
		if bs1[i].MinMu != bs2[i].MinMu {
			t.Errorf("tier %s: hp-loaded MinMu %v != default %v", bs1[i].Name, bs1[i].MinMu, bs2[i].MinMu)
		}
	}
}
