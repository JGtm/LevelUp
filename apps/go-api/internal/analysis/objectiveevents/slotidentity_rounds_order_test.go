package objectiveevents

// slotidentity_rounds_order_test.go — L'ORDRE DES DEBUTS DE MANCHE EST TOTAL.
//
// Meme famille de defaut, meme forme de preuve que `filmdec/projectile_track_order_test.go`
// (item 0.4bis de PLAN_CUISSON_PERF, 2026-09-02) : `roundStartsOf` batit sa tranche en iterant
// la MAP `min` — ordre aleatoire a chaque execution — et `sort.Slice` n'est pas stable. Trier
// sur le SEUL instant de debut laissait donc l'alea decider du rang de deux manches qui
// commencent au meme instant (manche vide, ou deux enregistrements au meme horodatage), et
// `RoundIdentity.At` place les instants d'apres cette liste.
//
// La preuve tient en deux temps : plusieurs ordres d'entree rendent UNE SEULE sortie, et la
// meme entree, rejouee, la rend encore — c'est le second qui secoue l'iteration de la map.

import (
	"reflect"
	"testing"
)

// recsExAequo : quatre manches dont TROIS commencent au meme instant (5 000 ms). L'instant seul
// ne les separe pas.
func recsExAequo() []StatRecord {
	return []StatRecord{
		{TimeMS: 5000, Slot: 10, Round: 0},
		{TimeMS: 9000, Slot: 11, Round: 0},
		{TimeMS: 7000, Slot: 10, Round: 1},
		{TimeMS: 8000, Slot: 11, Round: 1},
		{TimeMS: 5000, Slot: 10, Round: 2},
		{TimeMS: 5000, Slot: 10, Round: 3},
	}
}

// byRoundExAequo declare les quatre manches ; le contenu des tables d'identite n'entre pas dans
// le tri, seule leur PRESENCE compte (`roundStartsOf` ignore un enregistrement d'une manche
// absente).
func byRoundExAequo() map[int]map[int]string {
	return map[int]map[int]string{
		0: {10: "a"}, 1: {10: "b"}, 2: {10: "c"}, 3: {10: "d"},
	}
}

func TestRoundStartsOfEstTotalSurDesManchesExAequo(t *testing.T) {
	base := recsExAequo()
	ordres := [][]StatRecord{
		base,
		{base[5], base[4], base[3], base[2], base[1], base[0]},
		{base[2], base[0], base[5], base[1], base[3], base[4]},
		{base[4], base[3], base[1], base[5], base[0], base[2]},
	}
	attendu := []roundStart{
		{round: 0, startMS: 5000},
		{round: 2, startMS: 5000},
		{round: 3, startMS: 5000},
		{round: 1, startMS: 7000},
	}
	for i, entree := range ordres {
		got := roundStartsOf(entree, byRoundExAequo())
		if !reflect.DeepEqual(got, attendu) {
			t.Fatalf("ordre d'entree %d rend une sortie DIFFERENTE : le tri n'est pas total\n"+
				"  attendu : %v\n  obtenue : %v", i, attendu, got)
		}
	}
	// L'ITERATION DE MAP EST ALEATOIRE : rejouer la MEME entree est ce qui secoue reellement
	// l'ordre dans lequel `out` est bati avant le tri.
	for i := 0; i < 50; i++ {
		if got := roundStartsOf(base, byRoundExAequo()); !reflect.DeepEqual(got, attendu) {
			t.Fatalf("tour %d : sortie instable pour la meme entree\n  attendu : %v\n  obtenue : %v",
				i, attendu, got)
		}
	}
}

// TestRoundStartsOfGardeLInstantEnPremierCritere : le departage par numero de manche n'a pas pris
// le pas sur l'ordre metier — c'est l'instant de debut qui ordonne, le numero ne fait que
// trancher les egalites.
func TestRoundStartsOfGardeLInstantEnPremierCritere(t *testing.T) {
	recs := []StatRecord{
		{TimeMS: 9000, Slot: 10, Round: 0},
		{TimeMS: 1000, Slot: 10, Round: 1},
	}
	got := roundStartsOf(recs, map[int]map[int]string{0: {10: "a"}, 1: {10: "b"}})
	attendu := []roundStart{{round: 1, startMS: 1000}, {round: 0, startMS: 9000}}
	if !reflect.DeepEqual(got, attendu) {
		t.Fatalf("la manche qui commence le plus tot doit venir en premier\n  attendu : %v\n  obtenue : %v",
			attendu, got)
	}
}

// TestRoundStartsOfSansMultiManche : une seule manche (ou aucune) ne produit aucune borne — `At`
// rend alors toujours l'unique manche, sans regarder le temps.
func TestRoundStartsOfSansMultiManche(t *testing.T) {
	if got := roundStartsOf(recsExAequo(), map[int]map[int]string{0: {10: "a"}}); got != nil {
		t.Errorf("une seule manche ne doit produire aucune borne, obtenu %v", got)
	}
	if got := roundStartsOf(recsExAequo(), nil); got != nil {
		t.Errorf("aucune manche ne doit produire aucune borne, obtenu %v", got)
	}
}
