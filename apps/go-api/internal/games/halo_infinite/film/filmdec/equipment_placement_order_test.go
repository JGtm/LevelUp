package filmdec

// equipment_placement_order_test.go — L'ORDRE DES POSES D'EQUIPEMENT EST TOTAL.
//
// Meme famille de defaut, meme forme de preuve que `projectile_track_order_test.go` (item 0.4bis
// de PLAN_CUISSON_PERF, 2026-09-02) : `out` est bati en iterant la map `best` — dont l'ordre
// change a chaque execution — et `sort.Slice` n'est pas stable. Le triplet de tete
// (T0US, slot, generation) N'EST PAS total : la cle de `best` est (slot, generation, DEBUT DE
// VIE), donc deux vies distinctes du meme couple peuvent porter le meme instant de creation.
// Sans departage par le CONTENU de la pose, le rang publie etait tire au sort.
//
// La preuve : des poses EX AEQUO sur le triplet de tete, presentees dans plusieurs ordres
// d'entree, doivent rendre UNE SEULE sortie.

import (
	"reflect"
	"sort"
	"testing"
)

// poseExAequo fabrique une pose qui partage le triplet de tete (instant de pose, slot,
// generation) avec ses soeurs : seul le contenu la distingue.
func poseExAequo(t1us uint64, x float32, globalID uint32, points int) EquipmentPlacement {
	return EquipmentPlacement{
		Life:     EquipmentLifeKey{Slot: 7, Gen: 1},
		T0US:     1000,
		T1US:     t1us,
		X:        x,
		Y:        2,
		Z:        3,
		GlobalID: globalID,
		Points:   points,
	}
}

func TestLessPlacementEstTotalSurDesPosesExAequo(t *testing.T) {
	// Cinq poses du MEME couple (slot, generation) posees au MEME instant : chacune ne se
	// separe de la premiere que par UN champ de departage, et il faut donc que chacun de ces
	// champs compte.
	a := poseExAequo(2000, 10, 42, 5)
	b := poseExAequo(3000, 10, 42, 5) // fin de vie plus tardive
	c := poseExAequo(2000, 20, 42, 5) // meme fin, position differente
	d := poseExAequo(2000, 10, 99, 5) // meme position, autre identite d'objet
	e := poseExAequo(2000, 10, 42, 9) // tout pareil, plus d'echantillons

	ordres := [][]EquipmentPlacement{
		{a, b, c, d, e},
		{e, d, c, b, a},
		{c, a, e, b, d},
		{b, e, a, d, c},
	}
	var reference []EquipmentPlacement
	for i, entree := range ordres {
		copie := append([]EquipmentPlacement(nil), entree...)
		sort.Slice(copie, func(x, y int) bool { return lessPlacement(copie[x], copie[y]) })
		if i == 0 {
			reference = copie
			continue
		}
		if !reflect.DeepEqual(copie, reference) {
			t.Fatalf("ordre d'entree %d rend une sortie DIFFERENTE : le comparateur n'est pas total\n"+
				"  reference : %v\n  obtenue   : %v", i, reference, copie)
		}
	}
	// Le premier critere de departage est la fin de vie : `a`, `d` et `e` (T1US 2000, X 10)
	// precedent `c` (X 20), qui precede `b` (T1US 3000).
	if reference[len(reference)-1].T1US != 3000 {
		t.Errorf("la pose a la fin de vie la plus tardive doit venir en dernier, obtenue T1US=%d",
			reference[len(reference)-1].T1US)
	}
}

// TestLessPlacementRespecteLesCriteresDeTete verifie que le departage n'a pas pris le pas sur
// l'ordre metier : instant de pose, puis slot, puis generation restent prioritaires.
func TestLessPlacementRespecteLesCriteresDeTete(t *testing.T) {
	tot := poseExAequo(9000, 99, 999, 99)
	tot.T0US = 500
	tard := poseExAequo(0, 0, 0, 0)
	if !lessPlacement(tot, tard) {
		t.Error("la pose la plus precoce doit passer devant, quel que soit son contenu")
	}
	memeInstant := poseExAequo(9000, 99, 999, 99)
	memeInstant.Life = EquipmentLifeKey{Slot: 2, Gen: 0}
	autreSlot := poseExAequo(0, 0, 0, 0)
	autreSlot.Life = EquipmentLifeKey{Slot: 3, Gen: 0}
	if !lessPlacement(memeInstant, autreSlot) {
		t.Error("a instant de pose egal, le plus petit slot passe devant")
	}
	gen0 := poseExAequo(9000, 99, 999, 99)
	gen0.Life = EquipmentLifeKey{Slot: 2, Gen: 0}
	gen1 := poseExAequo(0, 0, 0, 0)
	gen1.Life = EquipmentLifeKey{Slot: 2, Gen: 1}
	if !lessPlacement(gen0, gen1) {
		t.Error("a instant de pose et slot egaux, la plus petite generation passe devant")
	}
}

// TestLessPlacementNeSeparePasDeuxPosesIdentiques : deux poses identiques champ pour champ ne se
// separent pas — les echanger ne peut donc pas changer la sortie.
func TestLessPlacementNeSeparePasDeuxPosesIdentiques(t *testing.T) {
	x := poseExAequo(2000, 10, 42, 5)
	y := poseExAequo(2000, 10, 42, 5)
	if lessPlacement(x, y) || lessPlacement(y, x) {
		t.Error("deux poses identiques ne doivent pas se departager")
	}
}
