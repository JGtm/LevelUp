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

// TestWeaponClassHasAccuracy fige le prédicat d'inclusion du graphe « Précision par arme »
// (déplacé depuis le package service, exporté pour être partagé avec service/teammates).
// Les classes SANS « tir au but » — grenade/mêlée/capacités spartanes/non attribué — et les
// buckets non-combat (véhicule/tourelle/environnement/autre) sont EXCLUS ; les classes gun
// et la classe non résolue ("" — bénéfice du doute) sont INCLUSES.
func TestWeaponClassHasAccuracy(t *testing.T) {
	excluded := []string{
		FragClassGrenade, FragClassMelee, FragClassSpartanAbility, FragClassUnattributed,
		FragClassVehicle, FragClassTurret, FragClassEnvironmental, FragClassOther,
	}
	for _, c := range excluded {
		if WeaponClassHasAccuracy(c) {
			t.Errorf("WeaponClassHasAccuracy(%q) = true, want false", c)
		}
	}
	included := []string{
		FragClassShoulder, FragClassSidearm, FragClassHeavy,
		"", "precision", "automatic", "sniper", "shotgun",
	}
	for _, c := range included {
		if !WeaponClassHasAccuracy(c) {
			t.Errorf("WeaponClassHasAccuracy(%q) = false, want true", c)
		}
	}
}
