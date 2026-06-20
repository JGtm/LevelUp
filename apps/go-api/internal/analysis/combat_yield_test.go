package analysis

import (
	"math"
	"testing"
)

func TestComputeCombatYield_excludeAssistsToggle(t *testing.T) {
	// Réglage global "rendement sans assistances" : OffensiveConversion doit
	// ignorer les assists et égaler OffensiveFinishing (225*kills/damage).
	SetExcludeAssistsFromYield(true)
	defer SetExcludeAssistsFromYield(false) // restaure le défaut pour les autres tests

	if !AssistsExcludedFromYield() {
		t.Fatal("AssistsExcludedFromYield should be true after Set(true)")
	}

	cy := ComputeCombatYield(10, 6, 2000, 1800, 4)

	// Sans assists : OC = 225 * 10 / 2000 = 1.125 (== OffensiveFinishing).
	wantOC := 225.0 * 10.0 / 2000.0
	if math.Abs(cy.OffensiveConversion-wantOC) > 1e-9 {
		t.Errorf("OffensiveConversion (excl. assists) = %.6f, want %.6f", cy.OffensiveConversion, wantOC)
	}
	if math.Abs(cy.OffensiveConversion-cy.OffensiveFinishing) > 1e-9 {
		t.Errorf("OC (%.6f) doit égaler OffensiveFinishing (%.6f) quand assists exclus",
			cy.OffensiveConversion, cy.OffensiveFinishing)
	}

	// DamagePerFragEquivalent doit aussi ignorer les assists (= damage/kills).
	if v := DamagePerFragEquivalent(2000, 10, 6); math.Abs(v-2000.0/10.0) > 1e-9 {
		t.Errorf("DamagePerFragEquivalent (excl. assists) = %.6f, want %.6f", v, 2000.0/10.0)
	}

	// Après restauration, le comportement par défaut (assists/3) revient.
	SetExcludeAssistsFromYield(false)
	cy2 := ComputeCombatYield(10, 6, 2000, 1800, 4)
	if math.Abs(cy2.OffensiveConversion-225.0*12.0/2000.0) > 1e-9 {
		t.Errorf("OC (défaut) = %.6f, want %.6f", cy2.OffensiveConversion, 225.0*12.0/2000.0)
	}
}

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

func TestClassifyCombatProfile_OffensiveBands(t *testing.T) {
	// 5 bandes OC : disperse <0.78 / irregulier / equilibre / precis / chirurgical >0.90.
	cases := []struct {
		oc   float64
		want string
	}{
		{0.77, "disperse"}, {0.78, "irregulier"}, {0.80, "irregulier"},
		{0.81, "equilibre"}, {0.84, "equilibre"}, {0.85, "precis"},
		{0.89, "precis"}, {0.90, "chirurgical"}, {0.98, "chirurgical"},
	}
	for _, c := range cases {
		block := ClassifyCombatProfile(c.oc, 1.40, nil, 20)
		if block.StyleOffensive == nil || string(*block.StyleOffensive) != c.want {
			t.Errorf("OC=%.2f: want %q, got %v", c.oc, c.want, block.StyleOffensive)
		}
	}
}

func TestClassifyCombatProfile_DefensiveBands(t *testing.T) {
	// 5 bandes DR : fragile <1.20 / expose / solide / resistant / inebranlable >1.65.
	cases := []struct {
		dr   float64
		want string
	}{
		{1.10, "fragile"}, {1.19, "fragile"}, {1.20, "expose"},
		{1.34, "expose"}, {1.35, "solide"}, {1.49, "solide"},
		{1.50, "resistant"}, {1.64, "resistant"}, {1.65, "inebranlable"}, {2.10, "inebranlable"},
	}
	for _, c := range cases {
		block := ClassifyCombatProfile(0.82, c.dr, nil, 20)
		if block.StyleDefensive == nil || string(*block.StyleDefensive) != c.want {
			t.Errorf("DR=%.2f: want %q, got %v", c.dr, c.want, block.StyleDefensive)
		}
	}
}

func TestClassifyCombatProfile_ActivityBands(t *testing.T) {
	// 5 bandes pace_ratio : passif <0.80 / discret / mesure / actif / agressif >1.25.
	cases := []struct {
		ratio float64
		want  string
	}{
		{0.71, "passif"}, {0.79, "passif"}, {0.80, "discret"},
		{0.91, "discret"}, {0.92, "mesure"}, {1.07, "mesure"},
		{1.08, "actif"}, {1.24, "actif"}, {1.25, "agressif"}, {1.50, "agressif"},
	}
	for _, c := range cases {
		ratio := c.ratio
		block := ClassifyCombatProfile(0.82, 1.40, &ratio, 20)
		if block.StyleActivity == nil || string(*block.StyleActivity) != c.want {
			t.Errorf("paceRatio=%.2f: want %q, got %v", c.ratio, c.want, block.StyleActivity)
		}
	}
}

func TestClassifyCombatProfile_NilPaceRatio_NilActivity(t *testing.T) {
	// StyleActivity reste nil si avgPaceRatio non fourni (dégradation gracieuse).
	block := ClassifyCombatProfile(0.82, 1.40, nil, 20)
	if block.StyleActivity != nil {
		t.Errorf("StyleActivity: want nil with nil paceRatio, got %v", block.StyleActivity)
	}
}

func TestClassifyCombatProfile_ZeroOCZeroDR(t *testing.T) {
	// Pas de données dégâts → bandes les plus basses (disperse + fragile).
	block := ClassifyCombatProfile(0, 0, nil, 20)
	if block.StyleOffensive == nil || string(*block.StyleOffensive) != "disperse" {
		t.Errorf("zero OC: want disperse, got %v", block.StyleOffensive)
	}
	if block.StyleDefensive == nil || string(*block.StyleDefensive) != "fragile" {
		t.Errorf("zero DR: want fragile, got %v", block.StyleDefensive)
	}
}
