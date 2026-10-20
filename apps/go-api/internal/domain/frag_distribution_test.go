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

// TestIsPerWeaponFragClass fige les classes dont le niveau 2 se ventile par OBJET
// (weapon_key) et non par rôle de combat.
//
// Véhicule et tourelle y sont depuis V73-3.2 : sur ces classes le registre porte
// class == role == family, un niveau 2 par rôle serait donc un arc unique sans
// information. Équipement et environnement les ont rejointes le 2026-09-01 avec la
// bascule de l'arme du kill : « Bobine à plasma » est une information, « environnement »
// n'en est pas une — même exigence.
//
// Ce prédicat décrit la FORME du niveau 2, jamais la PROVENANCE des frags : savoir SI une
// classe est servie se tranche dans fragdist.isRegistryFragClass, qui regarde d'où vient
// la ligne (verrou Halo 5 : fragdist_halo5_golden_test.go).
func TestIsPerWeaponFragClass(t *testing.T) {
	for _, c := range []string{FragClassVehicle, FragClassTurret, FragClassEquipment, FragClassEnvironmental} {
		if !IsPerWeaponFragClass(c) {
			t.Errorf("IsPerWeaponFragClass(%q) = false, want true", c)
		}
	}
	notPerWeapon := []string{
		FragClassShoulder, FragClassSidearm, FragClassHeavy, FragClassMelee,
		FragClassGrenade, FragClassSpartanAbility, FragClassUnattributed,
		FragClassOther, "", "precision",
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
