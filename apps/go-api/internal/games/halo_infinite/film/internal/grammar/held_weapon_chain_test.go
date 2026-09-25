package grammar

// held_weapon_chain_test.go — LA QUALIFICATION DES ÉMISSIONS D'ARME CONTRE LA DOTATION DE
// NAISSANCE, et la chaîne coupée à chaque vie (lot M3.2 de la campagne « retours rejeu »,
// 2026-09-23).

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// naissanceAB : une vie née à t=100, arme A à l'emplacement 0, B au 1, le 2 vide.
func naissanceAB() SpawnState {
	return SpawnState{
		Families:       map[uint32]bool{0xA: true, 0xB: true},
		ParEmplacement: map[int]uint32{0: 0xA, 1: 0xB, 2: noVariant},
		DebutDeVie:     100,
	}
}

// TestPremiereEmissionContreLaDotationDeNaissance : la première émission d'un emplacement se lit
// contre l'arme de naissance de CET emplacement, qui devient `Previous`.
func TestPremiereEmissionContreLaDotationDeNaissance(t *testing.T) {
	spawn := func(uint32, uint64) (SpawnState, bool) { return naissanceAB(), true }
	for _, c := range []struct {
		nom         string
		emplacement int
		famille     uint32
		kind        types.HeldWeaponChangeKind
		previous    uint32
	}{
		{"meme arme", 0, 0xA, types.HeldWeaponRestated, noVariant},
		{"echange", 0, 0xC, types.HeldWeaponSwapped, 0xA},
		{"prise sur emplacement vide", 2, 0xC, types.HeldWeaponTaken, noVariant},
		{"lacher nomme", 1, noVariant, types.HeldWeaponDropped, 0xB},
	} {
		ch := types.HeldWeaponChange{TimestampUS: 200, Slot: 9, SlotIndex: 43 + c.emplacement,
			Emplacement: c.emplacement, Family: c.famille, Previous: noVariant}
		newHeldWeaponChain(spawn).qualifier(&ch)
		if ch.Kind != c.kind || ch.Previous != c.previous {
			t.Errorf("%s : %s depuis %08x, attendu %s depuis %08x", c.nom, ch.Kind, ch.Previous,
				c.kind, c.previous)
		}
	}
}

// TestLaChaineDesEmissionsSeCoupeAChaqueVie : un slot se réattribue. La première émission de la
// SECONDE vie se lit contre SA naissance — pas contre la dernière famille de la vie précédente,
// qui en ferait un « échange » depuis une arme que ce corps n'a jamais portée.
func TestLaChaineDesEmissionsSeCoupeAChaqueVie(t *testing.T) {
	spawn := func(_ uint32, at uint64) (SpawnState, bool) {
		if at >= 500 {
			return SpawnState{Families: map[uint32]bool{0xB: true},
				ParEmplacement: map[int]uint32{0: 0xB}, DebutDeVie: 500}, true
		}
		return naissanceAB(), true
	}
	chaine := newHeldWeaponChain(spawn)
	vie1 := types.HeldWeaponChange{TimestampUS: 200, Slot: 9, SlotIndex: 43, Family: 0xC, Previous: noVariant}
	chaine.qualifier(&vie1)
	if vie1.Kind != types.HeldWeaponSwapped || vie1.Previous != 0xA {
		t.Fatalf("vie 1 : %s depuis %08x, attendu swapped depuis a", vie1.Kind, vie1.Previous)
	}
	vie2 := types.HeldWeaponChange{TimestampUS: 600, Slot: 9, SlotIndex: 43, Family: 0xB, Previous: noVariant}
	if chaine.qualifier(&vie2) {
		t.Error("la première émission d'une vie ne peut pas « répéter » la vie précédente")
	}
	if vie2.Kind != types.HeldWeaponRestated || vie2.Previous != noVariant {
		t.Fatalf("vie 2 : %s depuis %08x, attendu restated (l'arme de sa naissance)", vie2.Kind,
			vie2.Previous)
	}
	suite := types.HeldWeaponChange{TimestampUS: 700, Slot: 9, SlotIndex: 43, Family: 0xD, Previous: noVariant}
	chaine.qualifier(&suite)
	if suite.Kind != types.HeldWeaponSwapped || suite.Previous != 0xB {
		t.Fatalf("suite de la vie 2 : %s depuis %08x, attendu swapped depuis b", suite.Kind, suite.Previous)
	}
}
