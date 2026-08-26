package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// grenade_reads_test.go — LES GARDE-RAILS DE L'AXE DES GRENADES.
//
// CE QU'ILS VERROUILLENT, et pourquoi c'est le point sensible du lot : l'axe reçoit DEUX
// canaux sur la MÊME grandeur. Une conversion oubliée (le flux code la sélection en base 1) ou
// une lecture partielle publiée comme pleine reproduirait, sous une autre forme, le défaut que
// la version 19 vient de fermer.

func kfGren(ts uint64, slot uint32, g [4]uint32, sel int) KeyframeInventory {
	return KeyframeInventory{
		TimestampUS: ts, Slot: slot, Grenades: g, GrenadesRead: true,
		SelectedGrenadeRank: sel, AbilityRank: -1, DrawnSlot: -1,
	}
}

func TestBuildGrenadeReadsPublieLesDeuxCanauxAvecLeurSource(t *testing.T) {
	kf := []KeyframeInventory{kfGren(1_000_000, 7, [4]uint32{1, 0, 2, 0}, 2)}
	deltas := []filmdec.InventoryDelta{{
		Slot: 7, TimestampUS: 1_500_000, Grenades: []uint32{1, 0, 1, 0},
		SelRead: true, Sel: 2, Mask: 0b0101,
	}}
	out := buildGrenadeReads(kf, deltas, 1_000_000, 100_000)
	if len(out) != 2 {
		t.Fatalf("%d lectures publiées, attendu 2 : %+v", len(out), out)
	}
	if out[0].Src != GrenadeSrcKeyframe || out[0].T != 0 {
		t.Fatalf("première lecture %+v, attendue kf à T=0", out[0])
	}
	if out[1].Src != GrenadeSrcDelta || out[1].T != 5 {
		t.Fatalf("seconde lecture %+v, attendue delta à T=5", out[1])
	}
	if out[1].Gs == nil || *out[1].Gs != 2 {
		t.Fatalf("sélection delta %v, attendue 2 (rang base 0)", out[1].Gs)
	}
	// LA GRANDEUR EST LA MÊME DES DEUX CÔTÉS : même longueur, même ordre de rangs.
	if len(out[0].G) != len(out[1].G) {
		t.Fatalf("les deux canaux ne publient pas la même forme : %v contre %v", out[0].G, out[1].G)
	}
}

// TestBuildGrenadeReadsEcarteCeQuiPrecedeLOrigine : une lecture antérieure n'a pas de place sur
// l'axe, et lui en inventer une la poserait sur la première image comme si elle y avait été
// mesurée.
func TestBuildGrenadeReadsEcarteCeQuiPrecedeLOrigine(t *testing.T) {
	kf := []KeyframeInventory{kfGren(500_000, 7, [4]uint32{1, 0, 0, 0}, -1)}
	deltas := []filmdec.InventoryDelta{
		{Slot: 7, TimestampUS: 400_000, Grenades: []uint32{2, 0, 0, 0}},
		{Slot: 7, TimestampUS: 2_000_000, Grenades: []uint32{0, 0, 0, 0}},
	}
	out := buildGrenadeReads(kf, deltas, 1_000_000, 100_000)
	if len(out) != 1 || out[0].Src != GrenadeSrcDelta {
		t.Fatalf("lectures publiées %+v, attendue la seule delta postérieure à l'origine", out)
	}
	// UN QUADRUPLET NUL EST UNE MESURE, pas une absence : il doit être publié.
	if len(out[0].G) != 4 {
		t.Fatalf("le quadruplet nul n'a pas été publié : %+v", out[0])
	}
}

// TestBuildGrenadeReadsTaitUneLectureDeltaSansCompteurs : sur cet axe la grandeur EST le
// quadruplet. Une lecture delta qui ne porte que la sélection n'a rien à y dire — publier un
// tableau vide se lirait « plus aucune grenade ».
func TestBuildGrenadeReadsTaitUneLectureDeltaSansCompteurs(t *testing.T) {
	deltas := []filmdec.InventoryDelta{
		{Slot: 3, TimestampUS: 2_000_000, Grenades: nil, SelRead: true, Sel: 1, Mask: 0b0010},
	}
	if out := buildGrenadeReads(nil, deltas, 1_000_000, 100_000); out != nil {
		t.Fatalf("une lecture sans compteurs a été publiée : %+v", out)
	}
}

// TestBuildGrenadeReadsNePubliePasDeSelectionDevinee : `InventoryDeltaNoSel` dit que le film ne
// désigne AUCUN type. Le publier comme un rang mettrait une grenade sélectionnée là où le film
// n'en désigne pas.
func TestBuildGrenadeReadsNePubliePasDeSelectionDevinee(t *testing.T) {
	deltas := []filmdec.InventoryDelta{{
		Slot: 3, TimestampUS: 2_000_000, Grenades: []uint32{0, 0, 0, 0},
		SelRead: true, Sel: filmdec.InventoryDeltaNoSel,
	}}
	out := buildGrenadeReads(nil, deltas, 1_000_000, 100_000)
	if len(out) != 1 {
		t.Fatalf("lectures %+v", out)
	}
	if out[0].Gs != nil {
		t.Fatalf("sélection publiée %v alors que le film n'en désigne aucune", *out[0].Gs)
	}
}

// TestGrenadeReadsNAffectePasLInventaire fige la décision d'architecture du lot : les lectures
// delta ne doivent JAMAIS entrer dans `Inventory`, sous peine de masquer une lecture pleine par
// une lecture partielle et de vider la cellule de munitions (défaut fermé en v19).
func TestGrenadeReadsNAffectePasLInventaire(t *testing.T) {
	kf := []KeyframeInventory{{
		TimestampUS: 1_000_000, Slot: 7, Grenades: [4]uint32{1, 0, 0, 0}, GrenadesRead: true,
		SelectedGrenadeRank: -1, AbilityRank: -1, DrawnSlot: 0, AmmoRead: true,
		Ammo: [4]SlotAmmo{}, AmmoCandidates: 1,
	}}
	inv, dropped := buildInventory(kf, 1_000_000, 100_000)
	if dropped != 0 || len(inv) != 1 {
		t.Fatalf("inventaire %+v (écartés %d)", inv, dropped)
	}
	before := len(inv)
	_ = buildGrenadeReads(kf, []filmdec.InventoryDelta{
		{Slot: 7, TimestampUS: 1_500_000, Grenades: []uint32{0, 0, 0, 0}},
	}, 1_000_000, 100_000)
	inv2, _ := buildInventory(kf, 1_000_000, 100_000)
	if len(inv2) != before {
		t.Fatalf("le calque inventaire a changé de taille : %d -> %d", before, len(inv2))
	}
}
