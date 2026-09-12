package filmdec

// navpoint_rise_order_test.go — L'ORDRE DES MONTEES DE L'ANNEAU EST TOTAL.
//
// Meme famille de defaut, meme forme de preuve que `projectile_track_order_test.go` (item 0.4bis
// de PLAN_CUISSON_PERF, etendu le 2026-09-02) : `NavpointContiguousRises` bat sa tranche en
// iterant la MAP `series` — dont l'ordre change a chaque execution — et `sort.Slice` n'est pas
// stable. Le couple d'hier (EndMS, Slot) N'EST PAS total : la serie d'UN slot peut porter deux
// lectures a la MEME milliseconde, et deux montees successives du meme slot finissent alors sur
// la meme borne. Leur rang etait tire au sort, et il compte : `replay/bomb_armings.go` lit ces
// montees dans l'ordre pour dater les armements de bombe.
//
// La preuve : des montees EX AEQUO sur le couple de tete, presentees dans plusieurs ordres
// d'entree, doivent rendre UNE SEULE sortie ; et le balayage complet, rejoue, rend la meme.

import (
	"reflect"
	"sort"
	"testing"
)

// monteeExAequo fabrique une montee qui partage le couple de tete (fin, slot) avec ses soeurs :
// seul son contenu la distingue.
func monteeExAequo(startMS int32, qStart, qEnd uint8, samples int) NavpointRise {
	return NavpointRise{
		Slot: 12, EndMS: 5_000, StartMS: startMS, QStart: qStart, QEnd: qEnd, Samples: samples,
	}
}

func TestLessNavpointRiseEstTotalSurDesMonteesExAequo(t *testing.T) {
	a := monteeExAequo(1_000, 10, 200, 5)
	b := monteeExAequo(2_000, 10, 200, 5) // depart plus tardif
	c := monteeExAequo(1_000, 20, 200, 5) // autre quantum de depart
	d := monteeExAequo(1_000, 10, 240, 5) // autre quantum d'arrivee
	e := monteeExAequo(1_000, 10, 200, 9) // plus d'echantillons

	ordres := [][]NavpointRise{
		{a, b, c, d, e},
		{e, d, c, b, a},
		{c, a, e, b, d},
		{b, e, a, d, c},
	}
	var reference []NavpointRise
	for i, entree := range ordres {
		copie := append([]NavpointRise(nil), entree...)
		sort.Slice(copie, func(x, y int) bool { return lessNavpointRise(copie[x], copie[y]) })
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

// TestLessNavpointRiseRespecteLesCriteresDeTete : le departage ne contourne pas l'ordre metier —
// la fin de montee, puis le slot, restent prioritaires.
func TestLessNavpointRiseRespecteLesCriteresDeTete(t *testing.T) {
	tot := monteeExAequo(9_000, 255, 255, 99)
	tot.EndMS = 1_000
	tard := monteeExAequo(0, 0, 0, 0)
	if !lessNavpointRise(tot, tard) {
		t.Error("la montee qui finit le plus tot doit passer devant, quel que soit son contenu")
	}
	memeFin := monteeExAequo(9_000, 255, 255, 99)
	memeFin.Slot = 3
	autreSlot := monteeExAequo(0, 0, 0, 0)
	autreSlot.Slot = 4
	if !lessNavpointRise(memeFin, autreSlot) {
		t.Error("a fin egale, le plus petit slot passe devant")
	}
}

// TestLessNavpointRiseNeSeparePasDeuxMonteesIdentiques : deux montees identiques champ pour
// champ ne se separent pas.
func TestLessNavpointRiseNeSeparePasDeuxMonteesIdentiques(t *testing.T) {
	x := monteeExAequo(1_000, 10, 200, 5)
	y := monteeExAequo(1_000, 10, 200, 5)
	if lessNavpointRise(x, y) || lessNavpointRise(y, x) {
		t.Error("deux montees identiques ne doivent pas se departager")
	}
}

// TestNavpointContiguousRisesRejouePareil : le balayage complet, dont la tranche est batie en
// iterant une map, rend la MEME sortie a chaque rejeu.
func TestNavpointContiguousRisesRejouePareil(t *testing.T) {
	// Deux slots dont les montees finissent a la MEME milliseconde : c'est la paire de
	// navpoints du protocole (+12, un par camp), et elle porte le meme anneau.
	var reads []NavpointRadialRead
	for _, slot := range []uint32{12, 24} {
		for i := int32(0); i < 4; i++ {
			reads = append(reads, NavpointRadialRead{
				Slot: slot, TMS: 1_000 + i*100, Q: uint8(10 + i*NavpointRiseMinQuanta),
			})
		}
	}
	reference := NavpointContiguousRises(reads)
	if len(reference) != 2 {
		t.Fatalf("deux montees attendues (une par slot), obtenues %d : %v", len(reference), reference)
	}
	if reference[0].EndMS != reference[1].EndMS {
		t.Fatalf("le temoin doit porter deux montees EX AEQUO sur la fin : %v", reference)
	}
	for tour := 0; tour < 50; tour++ {
		if got := NavpointContiguousRises(reads); !reflect.DeepEqual(got, reference) {
			t.Fatalf("tour %d : sortie differente\n  reference : %v\n  obtenue   : %v",
				tour, reference, got)
		}
	}
}
