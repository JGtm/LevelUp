package replay

// objective_object_order_test.go — L'ORDRE DES VIES PUBLIEES DU CALQUE D'OBJECTIF EST TOTAL.
//
// Meme famille de defaut, meme forme de preuve que `flag_free_life_order_test.go` (item 0.4bis
// de PLAN_CUISSON_PERF, etendu le 2026-09-02). Le couple d'hier (T0, famille) ne separait
// RIEN : une seule famille est publiee aujourd'hui (`ball`), donc deux vies nees sur la meme
// frame etaient ex aequo, et `sort.SliceStable` reconduisait pour elles l'ordre d'entree —
// c'est-a-dire l'ordre d'une tranche batie en iterant une map. Le commentaire d'alors
// affirmait pourtant l'independance de ce tri vis-a-vis de `flagFreeLives` ; il ne l'avait pas.
//
// La preuve : des vies EX AEQUO sur le couple de tete, presentees dans plusieurs ordres
// d'entree, doivent rendre UNE SEULE sortie.

import (
	"reflect"
	"sort"
	"testing"
)

// vieObjectifExAequo fabrique une vie publiee qui partage le couple de tete (instant de depart,
// famille) avec ses soeurs : seul le contenu la distingue.
func vieObjectifExAequo(t1 int, en string, pts []ObjectiveObjectPoint) ObjectiveObjectLife {
	return ObjectiveObjectLife{
		Family: familleCrane,
		En:     en,
		Fr:     "Crane",
		T0:     12,
		T1:     t1,
		Pts:    pts,
	}
}

func unPointObjectif(x, y float32) []ObjectiveObjectPoint {
	return []ObjectiveObjectPoint{{T: 12, X: x, Y: y}}
}

func TestObjectiveObjectLessEstTotalSurDesViesExAequo(t *testing.T) {
	a := vieObjectifExAequo(12, "Skull", unPointObjectif(10, 20))
	b := vieObjectifExAequo(40, "Skull", unPointObjectif(10, 20)) // fin plus tardive
	c := vieObjectifExAequo(12, "Oddball", unPointObjectif(10, 20))
	d := vieObjectifExAequo(12, "Skull", unPointObjectif(30, 20)) // autre position
	e := vieObjectifExAequo(12, "Skull", append(unPointObjectif(10, 20),
		ObjectiveObjectPoint{T: 13, X: 11, Y: 21})) // un point de plus

	ordres := [][]ObjectiveObjectLife{
		{a, b, c, d, e},
		{e, d, c, b, a},
		{c, a, e, b, d},
		{b, e, a, d, c},
	}
	var reference []ObjectiveObjectLife
	for i, entree := range ordres {
		copie := append([]ObjectiveObjectLife(nil), entree...)
		sort.SliceStable(copie, func(x, y int) bool { return objectiveObjectLess(copie[x], copie[y]) })
		if i == 0 {
			reference = copie
			continue
		}
		if !reflect.DeepEqual(copie, reference) {
			t.Fatalf("ordre d'entree %d rend une sortie DIFFERENTE : le comparateur n'est pas total\n"+
				"  reference : %v\n  obtenue   : %v", i, reference, copie)
		}
	}
}

// TestObjectiveObjectLessRespecteLesCriteresDeTete : le departage ne contourne pas l'ordre
// metier — l'instant de depart, puis la famille, restent prioritaires.
func TestObjectiveObjectLessRespecteLesCriteresDeTete(t *testing.T) {
	tot := vieObjectifExAequo(99, "Zzz", unPointObjectif(99, 99))
	tot.T0 = 1
	tard := vieObjectifExAequo(0, "Aaa", unPointObjectif(0, 0))
	if !objectiveObjectLess(tot, tard) {
		t.Error("la vie la plus precoce doit passer devant, quel que soit son contenu")
	}
	memeInstant := vieObjectifExAequo(99, "Zzz", unPointObjectif(99, 99))
	memeInstant.Family = "ball"
	autreFamille := vieObjectifExAequo(0, "Aaa", unPointObjectif(0, 0))
	autreFamille.Family = "flag"
	if !objectiveObjectLess(memeInstant, autreFamille) {
		t.Error("a instant egal, la famille departage avant le contenu")
	}
}

// TestObjectiveObjectLessNeSeparePasDeuxViesIdentiques : deux vies identiques champ pour champ
// ne se separent pas.
func TestObjectiveObjectLessNeSeparePasDeuxViesIdentiques(t *testing.T) {
	x := vieObjectifExAequo(12, "Skull", unPointObjectif(10, 20))
	y := vieObjectifExAequo(12, "Skull", unPointObjectif(10, 20))
	if objectiveObjectLess(x, y) || objectiveObjectLess(y, x) {
		t.Error("deux vies identiques ne doivent pas se departager")
	}
}
