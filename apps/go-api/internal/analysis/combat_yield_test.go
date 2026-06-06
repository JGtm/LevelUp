package analysis

import (
	"math"
	"testing"
)

func TestComputeCombatYield_nominal(t *testing.T) {
	// 10 kills, 6 assists, 2000 damage dealt, 1800 damage taken, 4 deaths
	cy := ComputeCombatYield(10, 6, 2000, 1800, 4)

	// offensive_conversion = 225 * (10 + 6/3) / 2000 = 225 * 12 / 2000 = 1.35
	wantOC := 225.0 * 12.0 / 2000.0
	if math.Abs(cy.OffensiveConversion-wantOC) > 1e-9 {
		t.Errorf("OffensiveConversion = %.6f, want %.6f", cy.OffensiveConversion, wantOC)
	}

	// defensive_resistance = 1800 / (225 * 4) = 1800/900 = 2.0
	wantDR := 1800.0 / (225.0 * 4.0)
	if math.Abs(cy.DefensiveResistance-wantDR) > 1e-9 {
		t.Errorf("DefensiveResistance = %.6f, want %.6f", cy.DefensiveResistance, wantDR)
	}

	// offensive_finishing = 225 * 10 / 2000 = 1.125
	wantOF := 225.0 * 10.0 / 2000.0
	if math.Abs(cy.OffensiveFinishing-wantOF) > 1e-9 {
		t.Errorf("OffensiveFinishing = %.6f, want %.6f", cy.OffensiveFinishing, wantOF)
	}
}

func TestComputeCombatYield_zeroDamageDealt(t *testing.T) {
	cy := ComputeCombatYield(5, 2, 0, 1000, 3)
	if cy.OffensiveConversion != 0 {
		t.Errorf("OffensiveConversion should be 0 when damage_dealt=0, got %f", cy.OffensiveConversion)
	}
	if cy.OffensiveFinishing != 0 {
		t.Errorf("OffensiveFinishing should be 0 when damage_dealt=0, got %f", cy.OffensiveFinishing)
	}
}

func TestComputeCombatYield_zeroDeaths(t *testing.T) {
	cy := ComputeCombatYield(5, 2, 1000, 500, 0)
	if cy.DefensiveResistance != 0 {
		t.Errorf("DefensiveResistance should be 0 when deaths=0, got %f", cy.DefensiveResistance)
	}
}

func TestComputeCombatYield_zeroDamageTaken(t *testing.T) {
	cy := ComputeCombatYield(5, 2, 1000, 0, 3)
	if cy.DefensiveResistance != 0 {
		t.Errorf("DefensiveResistance should be 0 when damage_taken=0, got %f", cy.DefensiveResistance)
	}
}

func TestComputeCombatYield_zeroKillsWithAssists(t *testing.T) {
	// 0 kills, 9 assists → offensive_conversion > 0 (profil teamshot)
	cy := ComputeCombatYield(0, 9, 1000, 800, 2)
	wantOC := 225.0 * 3.0 / 1000.0 // 225 * (0 + 9/3) / 1000
	if math.Abs(cy.OffensiveConversion-wantOC) > 1e-9 {
		t.Errorf("OffensiveConversion = %.6f, want %.6f (assists-only profile)", cy.OffensiveConversion, wantOC)
	}
	if cy.OffensiveFinishing != 0 {
		t.Errorf("OffensiveFinishing should be 0 with 0 kills, got %f", cy.OffensiveFinishing)
	}
}

func TestComputeCombatYield_allZero(t *testing.T) {
	cy := ComputeCombatYield(0, 0, 0, 0, 0)
	if cy.OffensiveConversion != 0 || cy.DefensiveResistance != 0 || cy.OffensiveFinishing != 0 {
		t.Errorf("all-zero input should produce all-zero output, got %+v", cy)
	}
}

func TestComputeCombatYield_assistCoefficient(t *testing.T) {
	// Vérification que le coefficient exact est 1/3 (convention Halo Infinite)
	cy := ComputeCombatYield(0, 3, 225, 0, 0)
	// offensive_conversion = 225 * 1 / 225 = 1.0
	if math.Abs(cy.OffensiveConversion-1.0) > 1e-9 {
		t.Errorf("1/3 assist coefficient: OffensiveConversion = %.9f, want 1.0", cy.OffensiveConversion)
	}
}

func TestFragEquivalents_assistThird(t *testing.T) {
	// 10 frags + 6 assists → 10 + 6/3 = 12 frag-équivalents.
	if got := FragEquivalents(10, 6); math.Abs(got-12) > 1e-9 {
		t.Errorf("FragEquivalents(10,6) = %f, want 12", got)
	}
}

func TestDamagePerFragEquivalent_isInverseOfOC(t *testing.T) {
	// dégâts/frag-équivalent = 2000 / (10 + 6/3) = 2000/12 ≈ 166.67
	dpfe := DamagePerFragEquivalent(2000, 10, 6)
	want := 2000.0 / 12.0
	if math.Abs(dpfe-want) > 1e-9 {
		t.Errorf("DamagePerFragEquivalent = %f, want %f", dpfe, want)
	}
	// Invariant clé : OffensiveConversion == 225 / DamagePerFragEquivalent.
	cy := ComputeCombatYield(10, 6, 2000, 1800, 4)
	if math.Abs(cy.OffensiveConversion-225.0/dpfe) > 1e-9 {
		t.Errorf("OC (%f) != 225/dmgPerFragEq (%f)", cy.OffensiveConversion, 225.0/dpfe)
	}
}

func TestDamagePerFragEquivalent_zeroDenominator(t *testing.T) {
	if got := DamagePerFragEquivalent(2000, 0, 0); got != 0 {
		t.Errorf("DamagePerFragEquivalent with 0 frags/assists should be 0, got %f", got)
	}
}

func TestNormalizeForBar_belowP80(t *testing.T) {
	n := NormalizeForBar(0.5, OffensiveConversionP80)
	want := 0.5 / OffensiveConversionP80
	if math.Abs(n-want) > 1e-9 {
		t.Errorf("NormalizeForBar below p80: got %f, want %f", n, want)
	}
}

func TestNormalizeForBar_aboveClip(t *testing.T) {
	n := NormalizeForBar(OffensiveConversionP80*3, OffensiveConversionP80)
	if n != CombatYieldClipFactor {
		t.Errorf("NormalizeForBar above clip: got %f, want %f", n, CombatYieldClipFactor)
	}
}

func TestNormalizeForBar_zeroP80(t *testing.T) {
	if NormalizeForBar(1.0, 0) != 0 {
		t.Error("NormalizeForBar with p80=0 should return 0")
	}
}

func TestTooltipDamagePer_nominal(t *testing.T) {
	d := TooltipDamagePer(1000, 4)
	if math.Abs(d-250) > 1e-9 {
		t.Errorf("TooltipDamagePer = %f, want 250", d)
	}
}

func TestTooltipDamagePer_zeroCount(t *testing.T) {
	if TooltipDamagePer(1000, 0) != 0 {
		t.Error("TooltipDamagePer with count=0 should return 0")
	}
}

// =============================================================================
// ClassifyCombatProfile
// =============================================================================

func TestClassifyCombatProfile_BelowMinMatches_NoStyles(t *testing.T) {
	// < 15 matchs → styles nil, champs scalaires bien copiés.
	block := ClassifyCombatProfile(0.90, 1.60, nil, 14)
	if block.StyleOffensive != nil || block.StyleDefensive != nil || block.StyleActivity != nil {
		t.Errorf("< 15 matchs: all styles should be nil, got off=%v def=%v act=%v",
			block.StyleOffensive, block.StyleDefensive, block.StyleActivity)
	}
	if block.AvgOC != 0.90 || block.AvgDR != 1.60 || block.MatchCount != 14 {
		t.Errorf("scalar fields not copied: AvgOC=%v AvgDR=%v MatchCount=%d", block.AvgOC, block.AvgDR, block.MatchCount)
	}
}

func TestClassifyCombatProfile_ExactlyMinMatches_HasStyles(t *testing.T) {
	// Exactement 15 matchs → styles calculés.
	block := ClassifyCombatProfile(0.90, 1.60, nil, 15)
	if block.StyleOffensive == nil || block.StyleDefensive == nil {
		t.Errorf("exactly 15 matchs: off and def should be set, got off=%v def=%v",
			block.StyleOffensive, block.StyleDefensive)
	}
}

func TestClassifyCombatProfile_PrecisResistant(t *testing.T) {
	// avgOC >= P80 → precis ; avgDR >= P80 → resistant.
	block := ClassifyCombatProfile(OffensiveConversionP80, DefensiveResistanceP80, nil, 20)
	if block.StyleOffensive == nil || *block.StyleOffensive != "precis" {
		t.Errorf("StyleOffensive: want precis, got %v", block.StyleOffensive)
	}
	if block.StyleDefensive == nil || *block.StyleDefensive != "resistant" {
		t.Errorf("StyleDefensive: want resistant, got %v", block.StyleDefensive)
	}
}

func TestClassifyCombatProfile_EquilibreSolide(t *testing.T) {
	// avgOC = P80 * 0.75 → equilibre ; avgDR = P80 * 0.75 → solide.
	oc := OffensiveConversionP80 * 0.75
	dr := DefensiveResistanceP80 * 0.75
	block := ClassifyCombatProfile(oc, dr, nil, 20)
	if block.StyleOffensive == nil || *block.StyleOffensive != "equilibre" {
		t.Errorf("StyleOffensive: want equilibre, got %v", block.StyleOffensive)
	}
	if block.StyleDefensive == nil || *block.StyleDefensive != "solide" {
		t.Errorf("StyleDefensive: want solide, got %v", block.StyleDefensive)
	}
}

func TestClassifyCombatProfile_GeneruxFragile(t *testing.T) {
	// avgOC < P80 * 0.70 → genereux ; avgDR < P80 * 0.70 → fragile.
	oc := OffensiveConversionP80 * 0.50
	dr := DefensiveResistanceP80 * 0.50
	block := ClassifyCombatProfile(oc, dr, nil, 20)
	if block.StyleOffensive == nil || *block.StyleOffensive != "genereux" {
		t.Errorf("StyleOffensive: want genereux, got %v", block.StyleOffensive)
	}
	if block.StyleDefensive == nil || *block.StyleDefensive != "fragile" {
		t.Errorf("StyleDefensive: want fragile, got %v", block.StyleDefensive)
	}
}

func TestClassifyCombatProfile_ActivityActif(t *testing.T) {
	residual := 6.0
	block := ClassifyCombatProfile(OffensiveConversionP80, DefensiveResistanceP80, &residual, 20)
	if block.StyleActivity == nil || *block.StyleActivity != "actif" {
		t.Errorf("StyleActivity: want actif (residual=6), got %v", block.StyleActivity)
	}
}

func TestClassifyCombatProfile_ActivityModere(t *testing.T) {
	residual := 0.0
	block := ClassifyCombatProfile(OffensiveConversionP80, DefensiveResistanceP80, &residual, 20)
	if block.StyleActivity == nil || *block.StyleActivity != "modere" {
		t.Errorf("StyleActivity: want modere (residual=0), got %v", block.StyleActivity)
	}
}

func TestClassifyCombatProfile_ActivityDiscret(t *testing.T) {
	residual := -10.0
	block := ClassifyCombatProfile(OffensiveConversionP80, DefensiveResistanceP80, &residual, 20)
	if block.StyleActivity == nil || *block.StyleActivity != "discret" {
		t.Errorf("StyleActivity: want discret (residual=-10), got %v", block.StyleActivity)
	}
}

func TestClassifyCombatProfile_NilResidualBrut_NilActivity(t *testing.T) {
	// Même avec 20 matchs, StyleActivity reste nil si residualBrut non fourni.
	block := ClassifyCombatProfile(OffensiveConversionP80, DefensiveResistanceP80, nil, 20)
	if block.StyleActivity != nil {
		t.Errorf("StyleActivity: want nil with nil residualBrut, got %v", block.StyleActivity)
	}
}

func TestClassifyCombatProfile_ActivityBoundary_Minus5_IsModere(t *testing.T) {
	// Limite basse : -5 → modéré (>= -5).
	residual := -5.0
	block := ClassifyCombatProfile(OffensiveConversionP80, DefensiveResistanceP80, &residual, 20)
	if block.StyleActivity == nil || *block.StyleActivity != "modere" {
		t.Errorf("StyleActivity: want modere at boundary -5, got %v", block.StyleActivity)
	}
}

func TestClassifyCombatProfile_ZeroOCZeroDR(t *testing.T) {
	// Pas de données dégâts → genereux + fragile (classes, < seuil minimal).
	block := ClassifyCombatProfile(0, 0, nil, 20)
	if block.StyleOffensive == nil || *block.StyleOffensive != "genereux" {
		t.Errorf("zero OC: want genereux, got %v", block.StyleOffensive)
	}
	if block.StyleDefensive == nil || *block.StyleDefensive != "fragile" {
		t.Errorf("zero DR: want fragile, got %v", block.StyleDefensive)
	}
}
