package replay

// zone_letter_ranks_order_test.go — LES LETTRES DE ZONE SONT DETERMINISTES.
//
// Meme famille de defaut, meme forme de preuve que `ground_weapon_dropper_order_test.go` (item
// 0.4bis de PLAN_CUISSON_PERF, etendu le 2026-09-02) : `zoneLetterRanks` bat sa liste de zones
// en iterant la MAP `gauge`, puis la trie sur `gauge[ref]` SEUL. Or le slot de jauge n'est pas
// une cle : `pairGaugeSlots` garantit au plus une jauge PAR ZONE, jamais au plus une zone par
// jauge — a la difference d'`electZoneOwners`, qui tient un `held` par canal. Deux zones ex
// aequo sur le slot laissaient donc l'ordre d'iteration decider laquelle s'appelle A.
//
// La preuve : la meme entree, dont deux zones partagent le slot de jauge, rejouee cinquante
// fois avec des ordres d'insertion differents, rend les MEMES rangs.

import (
	"reflect"
	"testing"
)

func TestZoneLetterRanksDepartageDeuxZonesDuMemeSlot(t *testing.T) {
	// Deux zones (2 et 5) partagent le slot de jauge 100 ; la troisieme (9) porte le slot 101.
	// Le rang attendu suit le slot croissant, puis la reference de zone croissante.
	refs := []int{5, 2, 9}
	slots := map[int]uint32{5: 100, 2: 100, 9: 101}
	attendu := map[int]int{2: 0, 5: 1, 9: 2}
	for tour := 0; tour < 50; tour++ {
		gauge := map[int]uint32{}
		// L'ordre d'INSERTION change de tour en tour ; celui d'ITERATION change de lui-meme.
		for i := range refs {
			r := refs[(i+tour)%len(refs)]
			gauge[r] = slots[r]
		}
		got := zoneLetterRanks(gauge, len(refs), false)
		if !reflect.DeepEqual(got, attendu) {
			t.Fatalf("tour %d : rangs attendus %v, obtenus %v", tour, attendu, got)
		}
	}
}

// TestZoneLetterRanksSuitLeSlotAvantLaReference : le departage ferme les egalites, il ne prend
// pas le pas sur la regle — l'ordre reste celui des slots de jauge croissants.
func TestZoneLetterRanksSuitLeSlotAvantLaReference(t *testing.T) {
	gauge := map[int]uint32{9: 100, 2: 200}
	got := zoneLetterRanks(gauge, 2, false)
	attendu := map[int]int{9: 0, 2: 1}
	if !reflect.DeepEqual(got, attendu) {
		t.Errorf("rangs attendus %v (slot croissant), obtenus %v", attendu, got)
	}
}

// TestZoneLetterRanksPortesFermees : les trois refus documentes ne bougent pas.
func TestZoneLetterRanksPortesFermees(t *testing.T) {
	gauge := map[int]uint32{1: 10, 2: 20}
	if r := zoneLetterRanks(gauge, 2, true); r != nil {
		t.Errorf("un mode a colline ne recoit pas de lettres, obtenu %v", r)
	}
	if r := zoneLetterRanks(gauge, 3, false); r != nil {
		t.Errorf("sans bijection zone/catalogue, aucune lettre, obtenu %v", r)
	}
	quatre := map[int]uint32{1: 10, 2: 20, 3: 30, 4: 40}
	if r := zoneLetterRanks(quatre, 4, false); r != nil {
		t.Errorf("au-dela de trois zones, aucune lettre, obtenu %v", r)
	}
}
