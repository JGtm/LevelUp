package filmdec

// projectile_track_order_test.go — L'ORDRE DES VIES EST TOTAL, DONC REPRODUCTIBLE.
//
// Le defaut corrige le 2026-09-02 (item 0.4bis de PLAN_CUISSON_PERF) : `lessTrack` ne comparait
// que (naissance, slot, generation), triplet que `splitLives` peut rendre EX AEQUO — il coupe
// une vie sur un trou de 250 ms mais AUSSI sur un record `at rest`, si bien que plusieurs
// segments d'un meme couple (slot, gen) commencent au meme horodatage. Comme la tranche triee
// est batie en iterant une map et que `sort.Slice` n'est pas stable, l'ordre publie changeait a
// chaque execution. Mesure d'origine sur 000d5950 : 3 pistes sur 549 en ex aequo (bande ti=42)
// et 3 sur 477 (bande ti=37).

import (
	"reflect"
	"sort"
	"testing"
)

// pisteExAequo fabrique une vie de trois points qui NAIT au meme instant que ses soeurs : seul
// le contenu la distingue.
func pisteExAequo(slot, gen uint32, x float32, atRest bool) ProjectileTrack {
	return ProjectileTrack{Slot: slot, Gen: gen, Pts: []ProjectileSample{
		{TimestampUS: 1000, Chunk: 1, X: x, Y: 2, Z: 3},
		{TimestampUS: 1016, Chunk: 1, X: x + 1, Y: 2, Z: 3},
		{TimestampUS: 1032, Chunk: 1, X: x + 2, Y: 2, Z: 3, AtRest: atRest},
	}}
}

func TestLessTrackEstTotalSurDesViesExAequo(t *testing.T) {
	// Quatre vies du MEME couple (slot, gen) qui naissent au MEME instant : le triplet
	// (naissance, slot, generation) ne les separe pas du tout.
	a := pisteExAequo(7, 1, 10, false)
	b := pisteExAequo(7, 1, 20, false)
	c := pisteExAequo(7, 1, 20, true)
	d := pisteExAequo(7, 1, 10, false)
	d.Pts = d.Pts[:len(d.Pts)-1] // meme debut, moins de points

	ordres := [][]ProjectileTrack{
		{a, b, c, d},
		{d, c, b, a},
		{c, a, d, b},
		{b, d, a, c},
	}
	var reference []ProjectileTrack
	for i, entree := range ordres {
		copie := append([]ProjectileTrack(nil), entree...)
		sort.Slice(copie, func(x, y int) bool { return lessTrack(copie[x], copie[y]) })
		if i == 0 {
			reference = copie
			continue
		}
		if !reflect.DeepEqual(copie, reference) {
			t.Fatalf("ordre d'entree %d rend une sortie DIFFERENTE : le comparateur n'est pas total\n"+
				"  reference : %v\n  obtenue   : %v", i, resume(reference), resume(copie))
		}
	}
	// La plus courte d'abord, puis par contenu : l'ordre ne doit rien devoir a l'arrivee.
	if len(reference[0].Pts) != 2 {
		t.Errorf("la vie la plus courte doit venir en premier, obtenu %d points", len(reference[0].Pts))
	}
}

// TestLessTrackRespecteLesCritetresDeTete verifie que le departage n'a pas pris le pas sur
// l'ordre metier : naissance, puis slot, puis generation restent prioritaires.
func TestLessTrackRespecteLesCriteresDeTete(t *testing.T) {
	tot := pisteExAequo(9, 3, 99, false)
	tot.Pts[0].TimestampUS = 500
	tard := pisteExAequo(1, 0, 1, false)
	if !lessTrack(tot, tard) {
		t.Error("la naissance la plus precoce doit passer devant, quel que soit le slot")
	}
	memeInstant := pisteExAequo(2, 0, 1, false)
	autreSlot := pisteExAequo(3, 0, 1, false)
	if !lessTrack(memeInstant, autreSlot) {
		t.Error("a naissance egale, le plus petit slot passe devant")
	}
	gen0 := pisteExAequo(2, 0, 1, false)
	gen1 := pisteExAequo(2, 1, 1, false)
	if !lessTrack(gen0, gen1) {
		t.Error("a naissance et slot egaux, la plus petite generation passe devant")
	}
}

// TestLessTrackNeSeparePasDeuxViesIdentiques : deux vies identiques champ pour champ ne se
// separent pas — les echanger ne peut donc pas changer la sortie.
func TestLessTrackNeSeparePasDeuxViesIdentiques(t *testing.T) {
	x := pisteExAequo(4, 2, 5, false)
	y := pisteExAequo(4, 2, 5, false)
	if lessTrack(x, y) || lessTrack(y, x) {
		t.Error("deux vies identiques ne doivent pas se departager")
	}
}

// resume rend une forme lisible d'une liste de vies, pour les messages d'echec.
func resume(l []ProjectileTrack) []string {
	out := make([]string, 0, len(l))
	for _, tr := range l {
		out = append(out, string(rune('A'+len(tr.Pts)))+"/"+string(rune('0'+int(tr.Pts[0].X)%10)))
	}
	return out
}
