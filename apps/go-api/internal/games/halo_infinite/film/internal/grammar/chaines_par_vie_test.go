package grammar

// chaines_par_vie_test.go — LES CANAUX DELTA SE CHAINENT PAR VIE, PAS PAR SLOT (lot J5.3 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25).
//
// Depuis le lot J5.2, un slot bipede porte ses corps SUCCESSIFS — generation 1, puis 2 quand le pool
// reboucle. Un canal qui chaine ses emissions par slot ferait lire la premiere emission du corps
// suivant contre la derniere du precedent : un « echange » depuis une arme que ce corps n a jamais
// portee, une capacite heritee, un compteur R(3) qui « repete » ou « saute ». La cle est la vie,
// `types.LifeKey` (slot, generation du handle).

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestEquipmentChanges_NouvelleVieNHeritePasDeLaPrecedente : deux corps du MEME slot, generations 1
// puis 2. La premiere emission du second ouvre SA vie : aucun `Previous`, aucun pas de compteur
// compte contre le corps precedent (le compteur repart a 5 a chaque vie), deux vies.
func TestEquipmentChanges_NouvelleVieNHeritePasDeLaPrecedente(t *testing.T) {
	const slot uint32 = 517
	vie1 := emissAt(slot, 1_000, 1, 1, equipmentFirstCounter, 4)
	vie1.Gen = 1
	vie2 := emissAt(slot, 90_000, 5, 1, equipmentFirstCounter, 11)
	vie2.Gen = 2
	out, st := assembleEquipmentChanges([]abilityEmission{vie2, vie1}, nil, nil)
	if len(out) != 2 {
		t.Fatalf("%d changement(s), attendu 2", len(out))
	}
	second := out[1]
	if second.Previous != AbilitySetNoRank {
		t.Errorf("la premiere emission de la vie (slot %d, gen 2) herite de la capacite %d de la vie "+
			"(slot %d, gen 1) : elle doit ouvrir sa vie sans `Previous`", slot, second.Previous, slot)
	}
	if second.Gap != 0 || st.Repeats != 0 || st.CounterJumps != 0 {
		t.Errorf("pas de compteur compte a travers la frontiere de corps : gap %d, repetitions %d, "+
			"sauts %d (attendu 0, 0, 0)", second.Gap, st.Repeats, st.CounterJumps)
	}
	if st.Lives != 2 {
		t.Errorf("%d vie(s) comptee(s), attendu 2 : deux generations du handle sont deux corps", st.Lives)
	}
}

// TestHeldWeaponChanges_ChaineParVie : la chaine des emissions d arme d un emplacement se tient par
// VIE. Sans temoin de naissance (spawn nil, donc aucun `DebutDeVie` pour couper), la premiere
// emission de la generation 2 ne se lit pas contre la derniere de la generation 1 ; la suite de la
// generation 2 se lit bien contre la generation 2.
func TestHeldWeaponChanges_ChaineParVie(t *testing.T) {
	chaine := newHeldWeaponChain(nil)
	gen1 := types.HeldWeaponChange{TimestampUS: 200, Slot: 9, SlotIndex: 43, Family: 0xA, Previous: noVariant}
	chaine.qualifier(&gen1, 1)
	gen2 := types.HeldWeaponChange{TimestampUS: 900, Slot: 9, SlotIndex: 43, Family: 0xA, Previous: noVariant}
	if chaine.qualifier(&gen2, 2) {
		t.Error("la premiere emission de la generation 2 « repete » la generation 1 : les deux corps " +
			"sont chaines comme une seule vie")
	}
	if gen2.Previous != noVariant || gen2.Kind != types.HeldWeaponTaken {
		t.Fatalf("premiere emission de la generation 2 : %s depuis %08x, attendu taken sans Previous "+
			"(aucun relevé de spawn, et rien de la vie precedente)", gen2.Kind, gen2.Previous)
	}
	suite := types.HeldWeaponChange{TimestampUS: 950, Slot: 9, SlotIndex: 43, Family: 0xB, Previous: noVariant}
	chaine.qualifier(&suite, 2)
	if suite.Kind != types.HeldWeaponSwapped || suite.Previous != 0xA {
		t.Fatalf("suite de la generation 2 : %s depuis %08x, attendu swapped depuis a", suite.Kind, suite.Previous)
	}
}
