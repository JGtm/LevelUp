package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// inventory_test.go — les propriétés de l'inventaire, pas ses valeurs de sortie.
//
// CE QUE CES TESTS PROTÈGENT AVANT TOUT : la distinction entre « non lu », « zéro » et
// « aucune ». Elle est l'objet même de ce calque, et c'est elle qu'un refactor casse en
// silence — un `omitempty` mal placé, un `if !v` au lieu d'un `if v == nil`, et une arme sans
// chargeur devient une arme au chargeur vide.

// bitWriter écrit un flux de bits, gros-boutiste, comme le fait le format du film.
type bitWriter struct {
	buf []byte
	n   int
}

func (w *bitWriter) put(v uint32, width int) {
	for i := width - 1; i >= 0; i-- {
		if w.n%8 == 0 {
			w.buf = append(w.buf, 0)
		}
		if v>>uint(i)&1 == 1 {
			w.buf[w.n/8] |= 1 << uint(7-w.n%8)
		}
		w.n++
	}
}

// writeAmmoSlot écrit un emplacement selon la grammaire : union chargeur/jauge, réserve R(11),
// drapeaux R(2), surchauffe R(7). Les portes sont ACTIVE-BAS (0 = la branche est présente).
func writeAmmoSlot(w *bitWriter, mag *uint32, gauge *uint32, res uint32) {
	if mag != nil {
		w.put(0, 1)
		w.put(*mag, 8)
	} else {
		w.put(1, 1)
	}
	if gauge != nil {
		w.put(0, 1)
		w.put(*gauge, 12)
	} else {
		w.put(1, 1)
	}
	w.put(res, 11)
	w.put(0, 2) // drapeaux
	w.put(0, 7) // surchauffe
}

// buildAmmoBlock écrit un bloc complet : deux emplacements armés, deux vides, puis i42.
//
// Il rend AUSSI la longueur en BITS, et c'est nécessaire : l'écriture complète le dernier
// octet, donc `len(buf)*8` dépasse la vraie fin du bloc de 0 à 7 bits de bourrage. Confondre
// les deux ferait échouer le critère d'atterrissage — sur le banc d'essai, pas sur un film.
func buildAmmoBlock(mag0, res0 uint32, gauge1 *uint32, sel int) ([]byte, int) {
	w := &bitWriter{}
	m := mag0
	writeAmmoSlot(w, &m, nil, res0)
	writeAmmoSlot(w, nil, gauge1, 0)
	writeAmmoSlot(w, nil, nil, 0) // emplacement 2 : structurellement vide
	writeAmmoSlot(w, nil, nil, 0) // emplacement 3 : idem
	w.put(0, 3)                   // i42 : en-tête
	if sel >= 0 {
		w.put(0, 1) // porte active-bas : valeur présente
		w.put(uint32(sel), 2)
	} else {
		w.put(1, 1)
	}
	w.put(1, 1) // seconde porte, fermée
	return w.buf, w.n
}

func TestParseAmmoBlockReadsTheGrammar(t *testing.T) {
	g := uint32(2048)
	pay, total := buildAmmoBlock(25, 75, &g, 1)
	st, sel, _, ok := invParseAmmoBlock(pay, 0, total)
	if !ok {
		t.Fatal("le bloc construit selon la grammaire doit se parser")
	}
	if st[0].Mag == nil || *st[0].Mag != 25 || st[0].Res == nil || *st[0].Res != 75 {
		t.Errorf("emplacement 0 : chargeur/reserve mal lus : %+v", st[0])
	}
	if st[0].Gauge != nil {
		t.Error("une arme a chargeur ne porte pas de jauge : les deux branches s'excluent")
	}
	if st[1].Gauge == nil {
		t.Fatal("emplacement 1 : jauge non lue")
	}
	if v := *st[1].Gauge; v < 0.49 || v > 0.51 {
		t.Errorf("la jauge est une FRACTION dans [0,1], obtenu %v", v)
	}
	if sel != 1 {
		t.Errorf("selecteur attendu 1, obtenu %d", sel)
	}
}

func TestParseAmmoBlockDistinguishesNoneFromZero(t *testing.T) {
	// LE CAS DU MARTEAU : il n'emet ni chargeur ni jauge. « Aucune » n'est pas « zero » —
	// publier 0 affirmerait un chargeur vide la ou il n'y a pas de chargeur.
	pay, total := buildAmmoBlock(25, 75, nil, 0)
	st, _, _, ok := invParseAmmoBlock(pay, 0, total)
	if !ok {
		t.Fatal("parse")
	}
	if st[1].Mag != nil || st[1].Gauge != nil {
		t.Errorf("l'emplacement sans munition ne doit porter NI chargeur NI jauge : %+v", st[1])
	}
}

func TestParseAmmoBlockRefusesToReadPastTheLimit(t *testing.T) {
	// La limite est le critere d'arret du decodeur : un parse qui la franchit lirait les bits
	// du composant suivant et rendrait des valeurs credibles mais fausses.
	pay, _ := buildAmmoBlock(25, 75, nil, 0)
	if _, _, _, ok := invParseAmmoBlock(pay, 0, 20); ok {
		t.Error("un parse tronque doit echouer, pas rendre un etat partiel")
	}
}

func TestSolveAmmoBlockLandsExactly(t *testing.T) {
	// Le critere est un critere de LARGEUR : le parse doit atterrir AU BIT PRES sur la fin du
	// bloc. Aucune valeur n'y entre, sans quoi on choisirait la lecture qui plait.
	pay, end := buildAmmoBlock(25, 75, nil, 1)
	sols := invSolveAmmoBlock(pay, end, 0)
	if len(sols) == 0 {
		t.Fatal("le debut reel doit figurer parmi les solutions")
	}
	if sols[0] != 0 {
		t.Errorf("le debut le plus long doit venir en tete, obtenu %d", sols[0])
	}
	for _, s := range sols {
		if _, _, e, ok := invParseAmmoBlock(pay, s, end+1); !ok || e != end {
			t.Errorf("solution %d n'atterrit pas sur %d", s, end)
		}
	}
}

func TestBuildInventoryProjectsAndDropsPreOrigin(t *testing.T) {
	const origin, step = 1_000_000, 100_000
	mag := uint32(25)
	raw := []KeyframeInventory{
		{TimestampUS: 500_000, Slot: 512, AbilityRank: 19, DrawnSlot: 0}, // avant l'origine
		{TimestampUS: 1_600_000, Slot: 513, AbilityRank: -1, DrawnSlot: -1,
			Grenades: [4]uint32{0, 2, 0, 0}, GrenadesRead: true},
		{TimestampUS: 1_200_000, Slot: 512, AbilityRank: 20, DrawnSlot: 2,
			AmmoRead: true, Ammo: [4]SlotAmmo{{Mag: &mag}}, AmmoCandidates: 1},
	}
	got, dropped := buildInventory(raw, origin, step)
	if len(got) != 2 {
		t.Fatalf("l'etat anterieur a l'origine doit etre ecarte, obtenu %d etats", len(got))
	}
	if dropped != 1 {
		t.Errorf("compte des lectures ecartees avant l'origine = %d, attendu 1", dropped)
	}
	// Tri par image puis par slot : un artefact reproductible d'un build a l'autre.
	if got[0].T != 2 || got[0].Slot != 512 || got[1].T != 6 {
		t.Errorf("projection ou tri incorrects : %+v", got)
	}
	if got[0].D == nil || *got[0].D != 2 {
		t.Error("le selecteur 2 (AUCUNE arme degainee) est une VALEUR, il doit etre publie")
	}
	if got[1].D != nil {
		t.Error("un selecteur non lu doit rester absent")
	}
	if len(got[1].G) != 4 || got[1].G[1] != 2 {
		t.Errorf("compteurs de grenade mal portes : %+v", got[1].G)
	}
}

func TestBuildInventoryPublishesZeroCounters(t *testing.T) {
	// Un compteur a ZERO est une MESURE (« ce type, aucune en reserve »), a distinguer d'un
	// tableau absent (« non lu »). Le confondre effacerait l'information la plus utile.
	raw := []KeyframeInventory{{TimestampUS: 0, Slot: 1, AbilityRank: -1, DrawnSlot: -1,
		Grenades: [4]uint32{0, 0, 0, 0}, GrenadesRead: true}}
	got, _ := buildInventory(raw, 0, 100_000)
	if len(got) != 1 || len(got[0].G) != 4 {
		t.Fatalf("un etat lu a compteurs nuls doit etre publie : %+v", got)
	}
}

func TestAmmoSlotsOfKeepsOnlyArmedSlots(t *testing.T) {
	mag := uint32(7)
	r := KeyframeInventory{Ammo: [4]SlotAmmo{{Mag: &mag}, {}, {}, {}}}
	if got := ammoSlotsOf(r); len(got) != 2 {
		t.Errorf("seuls les deux emplacements porteurs d'arme se publient, obtenu %d", len(got))
	}
}

func TestKeepInventoryOfPublishedTracks(t *testing.T) {
	inv := []Inventory{{Slot: 512}, {Slot: 999}}
	got := keepInventoryOfPublishedTracks(inv, []Track{{Slot: 512}})
	if len(got) != 1 || got[0].Slot != 512 {
		t.Errorf("un inventaire sans trace ou se poser doit etre ecarte : %+v", got)
	}
	if keepInventoryOfPublishedTracks(nil, []Track{{Slot: 512}}) != nil {
		t.Error("sans inventaire, rien n'est invente")
	}
}

// TestInventoryCoverageBalances verrouille l'invariant ECRIT dans le commentaire d'InventoryCoverage :
// `Decoded == DroppedBeforeOrigin + Unpublished + Published`, exactement.
//
// POURQUOI UN TEST ET PAS SEULEMENT UN COMMENTAIRE. Les quatre compteurs viennent de TROIS etapes
// differentes du filtrage ; intervertir deux d'entre elles, ou brancher `Published` sur la mauvaise
// tranche, rend des chiffres plausibles dont la somme ne tombe plus juste. Les comptes sont pris NON
// TRIVIAUX (une lecture au moins dans chacune des trois issues) : un jeu de zeros passerait
// l'egalite sans rien mesurer.
func TestInventoryCoverageBalances(t *testing.T) {
	const origin, step = 1_000_000, 100_000
	decoded := []KeyframeInventory{
		// Avant l'origine : ECARTEE par buildInventory.
		{TimestampUS: 400_000, Slot: 512, AbilityRank: -1, DrawnSlot: -1},
		{TimestampUS: 700_000, Slot: 512, AbilityRank: -1, DrawnSlot: -1},
		// Apres l'origine, slot d'une piste PUBLIEE : retenue jusqu'au bout.
		{TimestampUS: 1_200_000, Slot: 512, AbilityRank: -1, DrawnSlot: 0},
		{TimestampUS: 1_400_000, Slot: 512, AbilityRank: -1, DrawnSlot: 1},
		{TimestampUS: 1_600_000, Slot: 512, AbilityRank: -1, DrawnSlot: 2},
		// Apres l'origine, slot SANS piste publiee : ecartee par le second filtre.
		{TimestampUS: 1_300_000, Slot: 999, AbilityRank: -1, DrawnSlot: -1},
	}
	built, dropped := buildInventory(decoded, origin, step)
	published := keepInventoryOfPublishedTracks(built, []Track{{Slot: 512}})
	cov := buildInventoryCoverage(decoded, built, published, dropped)
	if cov.Decoded != 6 || cov.DroppedBeforeOrigin != 2 || cov.Unpublished != 1 || cov.Published != 3 {
		t.Fatalf("couverture = %+v, attendu {Decoded:6 DroppedBeforeOrigin:2 Unpublished:1 Published:3}", *cov)
	}
	if somme := cov.DroppedBeforeOrigin + cov.Unpublished + cov.Published; somme != cov.Decoded {
		t.Errorf("invariant rompu : %d ecartees avant origine + %d sans piste + %d publiees = %d, "+
			"pour %d decodees", cov.DroppedBeforeOrigin, cov.Unpublished, cov.Published, somme, cov.Decoded)
	}
}

// TestInventoryCoverageAbsentWhenNothingToRead : l'ABSENCE de couverture et une couverture a ZERO
// sont deux etats differents, et le document doit les distinguer.
//
// C'EST LA DOCTRINE DE coverage.go : « son ABSENCE dit encore autre chose — l'appelant n'a rien
// fourni a lire ». Un inventaire illisible (BuildFromFilm pose `inventory = nil`) ne doit donc PAS
// publier {0,0,0,0}, qui se lirait « lecture faite, zero trouve ».
func TestInventoryCoverageAbsentWhenNothingToRead(t *testing.T) {
	// Deux points sur un slot : le minimum pour que le document soit assemble jusqu'au bout
	// (sans position, BuildFromPositions rend un document nu, sans aucune couverture).
	in := []filmdec.BipedPosition{pos(512, 0, 10, 20, 1), pos(512, 100, 11, 21, 1)}
	sans := BuildFromPositions("m", "halo_infinite", in, nil, Options{FrameIntervalMS: 100})
	if sans.Coverage == nil {
		t.Fatal("document sans couverture : rien a juger")
	}
	if sans.Coverage.Inventory != nil {
		t.Errorf("aucun inventaire fourni : la couverture doit rester ABSENTE, obtenu %+v",
			*sans.Coverage.Inventory)
	}
	vide := BuildFromPositions("m", "halo_infinite", in, nil,
		Options{FrameIntervalMS: 100, Inventory: []KeyframeInventory{}})
	if vide.Coverage == nil || vide.Coverage.Inventory == nil {
		t.Fatal("lecture faite mais vide : la couverture doit etre PRESENTE, a zero")
	}
	if got := *vide.Coverage.Inventory; got != (InventoryCoverage{}) {
		t.Errorf("couverture d'une lecture vide = %+v, attendu quatre zeros", got)
	}
}
