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
