package replay

// flag_free_life_order_test.go — L'ORDRE DES VIES LIBRES D'OBJET D'OBJECTIF EST TOTAL.
//
// Meme famille de defaut, meme forme de preuve que `filmdec/equipment_placement_order_test.go`
// (item 0.4bis de PLAN_CUISSON_PERF, etendu le 2026-09-02) : `flagFreeLives` bat sa tranche en
// iterant la MAP `byKey` et `sort.Slice` n'est pas stable. Le triplet de tete
// (T0US, slot, generation) N'EST PAS total : une meme cle peut porter DEUX creations au MEME
// instant — la fin de vie d'une cle etant la creation suivante de la meme cle, les deux vies
// sont alors ex aequo sur le triplet tout en portant des positions differentes.
//
// L'ORDRE COMPTE JUSQU'A L'ARTEFACT : `buildObjectiveObjects` retrie avec un `sort.SliceStable`,
// qui RECONDUIT l'ordre d'entree pour les ex aequo — l'alea se serait propage tel quel.
//
// La preuve : des vies EX AEQUO sur le triplet de tete, presentees dans plusieurs ordres
// d'entree, doivent rendre UNE SEULE sortie.

import (
	"reflect"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// vieLibreExAequo fabrique une vie libre qui partage le triplet de tete (instant de creation,
// slot, generation) avec ses soeurs : seul le contenu la distingue.
func vieLibreExAequo(id uint32, t1us uint64, pts []flagFreeSample) flagFreeLife {
	return flagFreeLife{
		ID:   id,
		Key:  filmdec.EquipmentLifeKey{Slot: 33, Gen: 2},
		T0US: 1_000_000,
		T1US: t1us,
		Pts:  pts,
	}
}

func unPoint(x, y float32) []flagFreeSample {
	return []flagFreeSample{{TUS: 1_000_000, X: x, Y: y}}
}

func TestFlagFreeLessEstTotalSurDesViesExAequo(t *testing.T) {
	// Cinq vies de la MEME cle creees au MEME instant : chacune ne se separe de la premiere que
	// par UN champ de departage.
	a := vieLibreExAequo(7, 1_000_000, unPoint(10, 20))
	b := vieLibreExAequo(7, 2_000_000, unPoint(10, 20)) // fin de vie plus tardive
	c := vieLibreExAequo(9, 1_000_000, unPoint(10, 20)) // autre identite d'objet
	d := vieLibreExAequo(7, 1_000_000, unPoint(30, 20)) // meme cle, autre position
	e := vieLibreExAequo(7, 1_000_000, append(unPoint(10, 20),
		flagFreeSample{TUS: 1_500_000, X: 11, Y: 21})) // un point de plus

	ordres := [][]flagFreeLife{
		{a, b, c, d, e},
		{e, d, c, b, a},
		{c, a, e, b, d},
		{b, e, a, d, c},
	}
	var reference []flagFreeLife
	for i, entree := range ordres {
		copie := append([]flagFreeLife(nil), entree...)
		sort.Slice(copie, func(x, y int) bool { return flagFreeLess(copie[x], copie[y]) })
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

// TestFlagFreeLessRespecteLesCriteresDeTete : le departage ne contourne pas l'ordre metier —
// instant de creation, puis slot, puis generation restent prioritaires.
func TestFlagFreeLessRespecteLesCriteresDeTete(t *testing.T) {
	tot := vieLibreExAequo(999, 9_000_000, unPoint(99, 99))
	tot.T0US = 500_000
	tard := vieLibreExAequo(0, 0, unPoint(0, 0))
	if !flagFreeLess(tot, tard) {
		t.Error("la vie la plus precoce doit passer devant, quel que soit son contenu")
	}
	memeInstant := vieLibreExAequo(999, 9_000_000, unPoint(99, 99))
	memeInstant.Key = filmdec.EquipmentLifeKey{Slot: 2, Gen: 3}
	autreSlot := vieLibreExAequo(0, 0, unPoint(0, 0))
	autreSlot.Key = filmdec.EquipmentLifeKey{Slot: 3, Gen: 0}
	if !flagFreeLess(memeInstant, autreSlot) {
		t.Error("a instant egal, le plus petit slot passe devant")
	}
	gen0 := vieLibreExAequo(999, 9_000_000, unPoint(99, 99))
	gen0.Key = filmdec.EquipmentLifeKey{Slot: 2, Gen: 0}
	gen1 := vieLibreExAequo(0, 0, unPoint(0, 0))
	gen1.Key = filmdec.EquipmentLifeKey{Slot: 2, Gen: 1}
	if !flagFreeLess(gen0, gen1) {
		t.Error("a instant et slot egaux, la plus petite generation passe devant")
	}
}

// TestFlagFreeLessNeSeparePasDeuxViesIdentiques : deux vies identiques champ pour champ ne se
// separent pas — les echanger ne peut donc pas changer la sortie.
func TestFlagFreeLessNeSeparePasDeuxViesIdentiques(t *testing.T) {
	x := vieLibreExAequo(7, 1_000_000, unPoint(10, 20))
	y := vieLibreExAequo(7, 1_000_000, unPoint(10, 20))
	if flagFreeLess(x, y) || flagFreeLess(y, x) {
		t.Error("deux vies identiques ne doivent pas se departager")
	}
}
