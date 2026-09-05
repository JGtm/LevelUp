package replay

// ground_weapon_object_order_test.go — L'ORDRE DES OBJETS AU SOL EST TOTAL.
//
// Meme famille de defaut, meme forme de preuve que `filmdec/projectile_track_order_test.go`
// (item 0.4bis de PLAN_CUISSON_PERF, etendu le 2026-09-02) : `buildGroundWeaponObjects` bat sa
// tranche en iterant la MAP `byKey` — dont l'ordre change a chaque execution — et `sort.Slice`
// n'est pas stable. La cle de tri d'hier etait la seule APPARITION, qui ne porte ni la cle de
// vie, ni le bornage, ni le ramasseur, ni le lacheur : deux armes de la meme famille nees au
// meme instant a la meme position quantifiee etaient strictement ex aequo, et leur rang etait
// tire au sort. Cet ordre n'est pas cosmetique — `gwItemLinkPickups` parcourt la tranche DANS
// SON ORDRE et attribue la prise au PREMIER objet a distance minimale.
//
// La preuve : des objets EX AEQUO sur l'apparition, presentes dans plusieurs ordres d'entree,
// doivent rendre UNE SEULE sortie.

import (
	"reflect"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// objetExAequo fabrique un objet au sol qui partage SON APPARITION ENTIERE avec ses soeurs :
// seul le reste de l'objet le distingue.
func objetExAequo(gen uint32, familyID uint32, status string, dropper int) gwPickupObject {
	return gwPickupObject{
		Key: filmdec.EquipmentLifeKey{Slot: 42, Gen: gen},
		Appar: gwPadApparition{
			Kind: gwPadKindWeapon, Family: "sniper",
			X: 10, Y: 20, Z: 30, TUS: 1_000_000,
			Class: gwClassSpawned,
		},
		FamilyID:    familyID,
		Pos:         [3]float32{10, 20, 30},
		Bounds:      gwPickupBounds{LowUS: 1_000_000, HighUS: 2_000_000, SeenKF: 1},
		Picker:      gwPickupHit{Slot: 7, TUS: 1_500_000, DistM: 1.2, Found: true},
		Status:      status,
		DropperSlot: dropper,
	}
}

func TestGwPickupLessEstTotalSurDesObjetsExAequo(t *testing.T) {
	// Cinq objets a l'apparition IDENTIQUE : chacun ne se separe de la reference que par UN
	// champ de departage, et il faut donc que chacun de ces champs compte.
	a := objetExAequo(0, 100, gwPickupStatusDated, -1)
	b := objetExAequo(1, 100, gwPickupStatusDated, -1) // autre generation
	c := objetExAequo(0, 200, gwPickupStatusDated, -1) // autre identite d'arme
	d := objetExAequo(0, 100, gwPickupStatusNever, -1) // autre issue d'occupation
	e := objetExAequo(0, 100, gwPickupStatusDated, 5)  // un lacheur nomme
	f := objetExAequo(0, 100, gwPickupStatusDated, -1) // ne differe que par le bornage
	f.Bounds.HighUS = 3_000_000

	ordres := [][]gwPickupObject{
		{a, b, c, d, e, f},
		{f, e, d, c, b, a},
		{c, a, f, e, b, d},
		{d, f, b, a, e, c},
	}
	var reference []gwPickupObject
	for i, entree := range ordres {
		copie := append([]gwPickupObject(nil), entree...)
		sort.Slice(copie, func(x, y int) bool { return gwPickupLess(copie[x], copie[y]) })
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

// TestGwPickupLessRespecteLApparition verifie que le departage n'a pas pris le pas sur l'ordre
// metier : l'apparition (instant en tete) reste prioritaire sur tout le reste de l'objet.
func TestGwPickupLessRespecteLApparition(t *testing.T) {
	tot := objetExAequo(3, 999, gwPickupStatusNever, 99)
	tot.Appar.TUS = 500_000
	tard := objetExAequo(0, 0, gwPickupStatusDated, -1)
	if !gwPickupLess(tot, tard) {
		t.Error("l'objet apparu le plus tot doit passer devant, quel que soit son contenu")
	}
	if gwPickupLess(tard, tot) {
		t.Error("l'ordre doit etre antisymetrique sur l'instant d'apparition")
	}
}

// TestGwPickupLessNeSeparePasDeuxObjetsIdentiques : deux objets identiques champ pour champ ne
// se separent pas — les echanger ne peut donc pas changer la sortie.
func TestGwPickupLessNeSeparePasDeuxObjetsIdentiques(t *testing.T) {
	x := objetExAequo(0, 100, gwPickupStatusDated, -1)
	y := objetExAequo(0, 100, gwPickupStatusDated, -1)
	if gwPickupLess(x, y) || gwPickupLess(y, x) {
		t.Error("deux objets identiques ne doivent pas se departager")
	}
}

// TestGwPadsLessSepareClasseEtVieDelta : l'apparition elle-meme etait incomplete — sa classe et
// sa vie delta restaient hors de la comparaison alors qu'elles font partie de la structure.
func TestGwPadsLessSepareClasseEtVieDelta(t *testing.T) {
	base := objetExAequo(0, 100, gwPickupStatusDated, -1).Appar
	autreClasse := base
	autreClasse.Class = gwClassDropped
	if !gwPadsLess(autreClasse, base) || gwPadsLess(base, autreClasse) {
		t.Errorf("la classe doit departager (%q avant %q)", gwClassDropped, gwClassSpawned)
	}
	avecDelta := base
	avecDelta.HasDelta = true
	if !gwPadsLess(base, avecDelta) || gwPadsLess(avecDelta, base) {
		t.Error("la vie delta doit departager deux apparitions autrement identiques")
	}
	if gwPadsLess(base, base) {
		t.Error("une apparition ne se departage pas d'elle-meme")
	}
}
