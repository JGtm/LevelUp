package domain

import "testing"

// TestIsNonCombatFragClass fige l'ensemble non-combat canonique côté Go (miroir de
// NON_COMBAT_WEAPON_ROLES web) : environnement / autre / non attribué sont exclus du
// breakdown par-arme, car aucun outil de destruction n'y est identifiable. Les classes
// d'arme réelles ne le sont jamais — véhicule et tourelle INCLUS depuis V73-3.2, le
// registre les nommant par engin (h5_vehicle_warthog…).
func TestIsNonCombatFragClass(t *testing.T) {
	nonCombat := []string{
		FragClassUnattributed, FragClassEnvironmental, FragClassOther,
	}
	for _, c := range nonCombat {
		if !IsNonCombatFragClass(c) {
			t.Errorf("IsNonCombatFragClass(%q) = false, want true", c)
		}
	}
	combat := []string{
		FragClassShoulder, FragClassSidearm, FragClassHeavy,
		FragClassMelee, FragClassGrenade, FragClassSpartanAbility,
		FragClassVehicle, FragClassTurret,
		"", "precision", "automatic",
	}
	for _, c := range combat {
		if IsNonCombatFragClass(c) {
			t.Errorf("IsNonCombatFragClass(%q) = true, want false", c)
		}
	}
}

// TestIsPerWeaponFragClass fige les classes dont le niveau 2 se ventile par ENGIN
// (weapon_key) et non par rôle de combat : véhicule et tourelle SEULEMENT. Sur ces
// classes le registre porte class == role == family, un niveau 2 par rôle serait donc
// un arc unique sans information (V73-3.2).
func TestIsPerWeaponFragClass(t *testing.T) {
	for _, c := range []string{FragClassVehicle, FragClassTurret} {
		if !IsPerWeaponFragClass(c) {
			t.Errorf("IsPerWeaponFragClass(%q) = false, want true", c)
		}
	}
	notPerWeapon := []string{
		FragClassShoulder, FragClassSidearm, FragClassHeavy, FragClassMelee,
		FragClassGrenade, FragClassSpartanAbility, FragClassUnattributed,
		FragClassEnvironmental, FragClassOther, "", "precision",
	}
	for _, c := range notPerWeapon {
		if IsPerWeaponFragClass(c) {
			t.Errorf("IsPerWeaponFragClass(%q) = true, want false", c)
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
