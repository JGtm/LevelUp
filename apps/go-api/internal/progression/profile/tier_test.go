package profile

import (
	"math"
	"testing"

	"levelup/go-api/internal/sync"
)

// tier_test.go — tests de comportement pour la conversion μ → TierState.
// Référentiel : sync.SkillTiers (Bronze 1000..1200 6 subs, …, Onyx 2000..9999 1 sub).

const epsilon = 1e-6

func approxEq(a, b float64) bool { return math.Abs(a-b) < epsilon }

func TestTierFromMu_BelowFirstTier(t *testing.T) {
	// μ sous le 1er tier (Bronze commence à 1000) → TierState vide.
	got := TierFromMu(999.0)
	if !got.IsEmpty() {
		t.Errorf("TierFromMu(999) = %+v, want empty", got)
	}
}

func TestTierFromMu_BronzeSubTier(t *testing.T) {
	// Bronze = [1000,1200), 6 subs, rangePerSub ≈ 33.333.
	// μ=1050 → sub = int((1050-1000)/33.333)+1 = int(1.5)+1 = 2.
	got := TierFromMu(1050.0)
	if got.Name != sync.TierBronze {
		t.Fatalf("Name = %q, want Bronze", got.Name)
	}
	if got.SubTier != 2 {
		t.Errorf("SubTier = %d, want 2", got.SubTier)
	}
	if got.Label != "Bronze II" {
		t.Errorf("Label = %q, want 'Bronze II'", got.Label)
	}
	// Bornes du sub-tier 2 : [1000+33.33, 1000+66.66].
	wantLower := 1000.0 + (200.0/6.0)*1
	wantUpper := 1000.0 + (200.0/6.0)*2
	if !approxEq(got.LowerMu, wantLower) {
		t.Errorf("LowerMu = %.4f, want %.4f", got.LowerMu, wantLower)
	}
	if !approxEq(got.UpperMu, wantUpper) {
		t.Errorf("UpperMu = %.4f, want %.4f", got.UpperMu, wantUpper)
	}
	// μ doit être dans la borne du sub-tier (invariant).
	if !(1050.0 >= got.LowerMu && 1050.0 <= got.UpperMu) {
		t.Errorf("μ=1050 hors bornes [%.2f, %.2f]", got.LowerMu, got.UpperMu)
	}
}

func TestTierFromMu_DiamondSubTier(t *testing.T) {
	// Diamond = [1800,2000), 6 subs, rangePerSub ≈ 33.333.
	// μ=1950 → sub = int((1950-1800)/33.333)+1 = int(4.5)+1 = 5.
	got := TierFromMu(1950.0)
	if got.Name != sync.TierDiamond {
		t.Fatalf("Name = %q, want Diamond", got.Name)
	}
	if got.SubTier != 5 {
		t.Errorf("SubTier = %d, want 5", got.SubTier)
	}
	if got.Label != "Diamond V" {
		t.Errorf("Label = %q, want 'Diamond V'", got.Label)
	}
}

func TestTierFromMu_OnyxNoSubTier(t *testing.T) {
	// Onyx = [2000,9999), SubTiers=1 → sub=0, bornes pleines du tier.
	got := TierFromMu(2500.0)
	if got.Name != sync.TierOnyx {
		t.Fatalf("Name = %q, want Onyx", got.Name)
	}
	if got.SubTier != 0 {
		t.Errorf("SubTier = %d, want 0 (Onyx sans sous-paliers)", got.SubTier)
	}
	if got.Label != "Onyx" {
		t.Errorf("Label = %q, want 'Onyx' (pas de chiffre romain)", got.Label)
	}
	if !approxEq(got.LowerMu, 2000.0) || !approxEq(got.UpperMu, 9999.0) {
		t.Errorf("bornes Onyx = [%.1f,%.1f], want [2000,9999]", got.LowerMu, got.UpperMu)
	}
}

func TestNextTierFromMu_InternalSubTierTransition(t *testing.T) {
	// μ=1050 → Bronze II. Next = Bronze III (transition sub interne).
	got := NextTierFromMu(1050.0)
	if got.Name != sync.TierBronze || got.SubTier != 3 {
		t.Errorf("next = %s %d, want Bronze 3", got.Name, got.SubTier)
	}
	if got.Label != "Bronze III" {
		t.Errorf("Label = %q, want 'Bronze III'", got.Label)
	}
	// Invariant : LowerMu du next == UpperMu du sub courant (continuité).
	cur := TierFromMu(1050.0)
	if !approxEq(got.LowerMu, cur.UpperMu) {
		t.Errorf("discontinuité : next.LowerMu=%.4f != cur.UpperMu=%.4f", got.LowerMu, cur.UpperMu)
	}
}

func TestNextTierFromMu_PromoteToNextTier(t *testing.T) {
	// μ dans le DERNIER sub-tier d'un tier → next = 1er sub du tier suivant.
	// Bronze VI : μ proche de 1200. 1000 + 33.33*5 = 1166.66 ≤ μ < 1200.
	got := NextTierFromMu(1190.0)
	if got.Name != sync.TierSilver {
		t.Fatalf("next.Name = %q, want Silver", got.Name)
	}
	if got.SubTier != 1 {
		t.Errorf("next.SubTier = %d, want 1", got.SubTier)
	}
	if !approxEq(got.LowerMu, 1200.0) {
		t.Errorf("next.LowerMu = %.2f, want 1200 (entrée Silver)", got.LowerMu)
	}
}

func TestNextTierFromMu_AtMaxOnyx(t *testing.T) {
	// Déjà Onyx (tier max, dernier de la slice) → TierState vide.
	got := NextTierFromMu(2500.0)
	if !got.IsEmpty() {
		t.Errorf("next depuis Onyx = %+v, want empty (pas de tier au-dessus)", got)
	}
}

func TestNextTierFromMu_OutOfRangeReturnsFirstTier(t *testing.T) {
	// μ hors plage (sous Bronze) → cible = 1er sub-tier (Bronze I).
	got := NextTierFromMu(500.0)
	if got.Name != sync.TierBronze || got.SubTier != 1 {
		t.Errorf("next hors plage = %s %d, want Bronze 1", got.Name, got.SubTier)
	}
	if !approxEq(got.LowerMu, 1000.0) {
		t.Errorf("LowerMu = %.2f, want 1000 (MinRating Bronze)", got.LowerMu)
	}
}

func TestSubTierBounds_FullTierWhenNoSubs(t *testing.T) {
	onyx := &sync.SkillTiers[len(sync.SkillTiers)-1] // Onyx, SubTiers=1
	// sub<=0 → bornes pleines.
	lo, hi := subTierBounds(onyx, 0)
	if !approxEq(lo, onyx.MinRating) || !approxEq(hi, onyx.MaxRating) {
		t.Errorf("sub=0 bounds = [%.1f,%.1f], want [%.1f,%.1f]", lo, hi, onyx.MinRating, onyx.MaxRating)
	}
	// SubTiers<=1 → bornes pleines même avec sub>0.
	lo, hi = subTierBounds(onyx, 1)
	if !approxEq(lo, onyx.MinRating) || !approxEq(hi, onyx.MaxRating) {
		t.Errorf("SubTiers<=1 bounds = [%.1f,%.1f], want full tier", lo, hi)
	}
}

func TestSubTierBounds_SplitWhenSubsPresent(t *testing.T) {
	gold := &sync.SkillTiers[2] // Gold [1400,1600), 6 subs
	lo, hi := subTierBounds(gold, 3)
	rangePerSub := (1600.0 - 1400.0) / 6.0
	wantLo := 1400.0 + rangePerSub*2
	wantHi := 1400.0 + rangePerSub*3
	if !approxEq(lo, wantLo) || !approxEq(hi, wantHi) {
		t.Errorf("Gold sub3 = [%.4f,%.4f], want [%.4f,%.4f]", lo, hi, wantLo, wantHi)
	}
	// Invariant : upper(sub) == lower(sub+1).
	_, hi3 := subTierBounds(gold, 3)
	lo4, _ := subTierBounds(gold, 4)
	if !approxEq(hi3, lo4) {
		t.Errorf("discontinuité sub3.upper=%.4f != sub4.lower=%.4f", hi3, lo4)
	}
}

func TestSubTierLowerUpper_Arithmetic(t *testing.T) {
	silver := &sync.SkillTiers[1] // Silver [1200,1400), 6 subs
	rangePerSub := (1400.0 - 1200.0) / 6.0
	// sub1 lower == MinRating ; sub6 upper == MaxRating (invariants bornes).
	if !approxEq(subTierLower(silver, 1), 1200.0) {
		t.Errorf("sub1 lower = %.4f, want 1200", subTierLower(silver, 1))
	}
	if !approxEq(subTierUpper(silver, 6), 1400.0) {
		t.Errorf("sub6 upper = %.4f, want 1400", subTierUpper(silver, 6))
	}
	if !approxEq(subTierUpper(silver, 2), 1200.0+rangePerSub*2) {
		t.Errorf("sub2 upper arithmétique faux : %.4f", subTierUpper(silver, 2))
	}
}

func TestFormatLabel(t *testing.T) {
	tests := []struct {
		name string
		tier string
		sub  int
		want string
	}{
		{"sub 0 → nom seul", "Onyx", 0, "Onyx"},
		{"sub négatif → nom seul", "Onyx", -1, "Onyx"},
		{"sub valide → roman", "Diamond", 3, "Diamond III"},
		{"sub 1 → roman I", "Bronze", 1, "Bronze I"},
		{"sub 6 → roman VI", "Gold", 6, "Gold VI"},
		{"sub hors map (7) → nom seul", "Gold", 7, "Gold"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatLabel(tt.tier, tt.sub)
			if got != tt.want {
				t.Errorf("formatLabel(%q,%d) = %q, want %q", tt.tier, tt.sub, got, tt.want)
			}
		})
	}
}
