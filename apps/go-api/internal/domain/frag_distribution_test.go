package domain

import "testing"

// TestIsNonCombatFragClass fige l'ensemble non-combat canonique côté Go (miroir de
// NON_COMBAT_WEAPON_ROLES web) : véhicule/tourelle/environnement/non attribué/autre
// sont exclus du breakdown par-arme ; les classes d'arme réelles ne le sont jamais.
func TestIsNonCombatFragClass(t *testing.T) {
	nonCombat := []string{
		FragClassUnattributed, FragClassVehicle, FragClassTurret,
		FragClassEnvironmental, FragClassOther,
	}
	for _, c := range nonCombat {
		if !IsNonCombatFragClass(c) {
			t.Errorf("IsNonCombatFragClass(%q) = false, want true", c)
		}
	}
	combat := []string{
		FragClassShoulder, FragClassSidearm, FragClassHeavy,
		FragClassMelee, FragClassGrenade, FragClassSpartanAbility,
		"", "precision", "automatic",
	}
	for _, c := range combat {
		if IsNonCombatFragClass(c) {
			t.Errorf("IsNonCombatFragClass(%q) = true, want false", c)
		}
	}
}
