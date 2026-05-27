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
	// cibles. Critère ⭐ de Phase 3e — c'est pour ça qu'on a calibré les seuils.
	bs := DefaultTierBoundaries()
	cases := []struct {
		name     string
		mu       float64
		wantTier string
	}{
		{"Madina BTB", 26.17, "Platinum"},        // cible Platine ✓
		{"Madina arena_slayer", 26.12, "Platinum"},
		{"Madina arena_objectif", 25.75, "Gold"}, // tout juste sous Platine
		{"Chocoboflor arena_slayer", 23.81, "Gold"}, // cible Or ✓
		{"JGtm arena_slayer", 23.52, "Gold"},     // cible Or ✓
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
	// Tests sur les bornes exactes.
	cases := []struct {
		mu       float64
		wantTier string
	}{
		{-100, "Bronze"},  // très bas → Bronze
		{0, "Bronze"},
		{21.99, "Bronze"}, // juste sous Silver
		{22.0, "Silver"},  // borne inclusive
		{23.49, "Silver"},
		{23.5, "Gold"},
		{25.99, "Gold"},
		{26.0, "Platinum"},
		{28.99, "Platinum"},
		{29.0, "Diamond"},
		{30.99, "Diamond"},
		{31.0, "Onyx"},
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
	// Or = [23.5, 26.0[, 6 sous-tiers → width = 0.4167 par sous-tier.
	// μ = 23.5 → sub 1 ; μ = 23.9 → sub 1 ; μ = 24.0 → sub 2 ; etc.
	cases := []struct {
		mu      float64
		wantSub int
	}{
		{23.5, 1},
		{23.9, 1},
		{24.0, 2},   // 23.5 + 1*0.4167 = 23.92, donc 24.0 = sub 2
		{24.5, 3},
		{25.0, 4},
		{25.5, 5},
		{25.9, 6},
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
	for _, mu := range []float64{31.0, 35.0, 50.0} {
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
	cases := []struct {
		mu   float64
		want string
	}{
		// Bronze couvre [0, 22[, width = 22/6 ≈ 3.67
		// μ = 1 → sub 1 (bas Bronze)
		// μ = 20 → sub 6 (haut Bronze, proche Silver)
		{1.0, "Bronze I"},
		{20.0, "Bronze VI"},
		// Or couvre [23.5, 26[, width = 2.5/6 ≈ 0.417
		// μ = 23.5 → sub 1, μ = 24.0 → sub 2, μ = 25.9 → sub 6
		{23.5, "Or I"},
		{24.0, "Or II"},
		{25.9, "Or VI"},
		// Platine couvre [26, 29[, width = 3/6 = 0.5
		{26.0, "Platine I"},
		// Onyx pas de sub
		{31.0, "Onyx"},
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
		"tier_boundary_gold":     24.0, // au lieu de 23.5
		"tier_boundary_platinum": 27.0, // au lieu de 26.0
	}
	bs := TierBoundariesFromHyperparams(hp)
	for _, b := range bs {
		switch b.Name {
		case "Gold":
			if b.MinMu != 24.0 {
				t.Errorf("Gold override échoué : %v", b.MinMu)
			}
		case "Platinum":
			if b.MinMu != 27.0 {
				t.Errorf("Platinum override échoué : %v", b.MinMu)
			}
		case "Silver":
			if b.MinMu != 22.0 {
				t.Errorf("Silver default écrasé : %v", b.MinMu)
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
