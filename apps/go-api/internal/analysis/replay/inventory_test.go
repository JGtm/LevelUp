package replay

import "testing"

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
		{TimestampUS: 500_000, Slot: 512, AbilityIndex: 3, DrawnSlot: 0}, // avant l'origine
		{TimestampUS: 1_600_000, Slot: 513, AbilityIndex: -1, DrawnSlot: -1,
			Grenades: [4]uint32{0, 2, 0, 0}, GrenadesRead: true},
		{TimestampUS: 1_200_000, Slot: 512, AbilityIndex: 4, DrawnSlot: 2,
			AmmoRead: true, Ammo: [4]SlotAmmo{{Mag: &mag}}, AmmoCandidates: 1},
	}
	got := buildInventory(raw, origin, step)
	if len(got) != 2 {
		t.Fatalf("l'etat anterieur a l'origine doit etre ecarte, obtenu %d etats", len(got))
	}
	// Tri par image puis par slot : un artefact reproductible d'un build a l'autre.
	if got[0].T != 2 || got[0].Slot != 512 || got[1].T != 6 {
		t.Errorf("projection ou tri incorrects : %+v", got)
	}
	if got[0].A == nil || *got[0].A != 4 {
		t.Errorf("capacite non portee : %+v", got[0])
	}
	if got[0].D == nil || *got[0].D != 2 {
		t.Error("le selecteur 2 (AUCUNE arme degainee) est une VALEUR, il doit etre publie")
	}
	if got[1].A != nil {
		t.Error("une capacite non lue doit rester absente, jamais valoir 0")
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
	raw := []KeyframeInventory{{TimestampUS: 0, Slot: 1, AbilityIndex: -1, DrawnSlot: -1,
		Grenades: [4]uint32{0, 0, 0, 0}, GrenadesRead: true}}
	got := buildInventory(raw, 0, 100_000)
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

func TestAbilityLabelsUsedNamesOnlyWhatItKnows(t *testing.T) {
	// LA TABLE EST PARTIELLE : 4 index observes pour 11 capacites. Un index hors table garde
	// son numero a l'ecran — le combler par le nom d'une capacite voisine se lirait comme une
	// certitude.
	known, unknown := 4, 9
	got := abilityLabelsUsed([]Inventory{{A: &known}, {A: &unknown}, {}})
	if got["4"] != "grappin" {
		t.Errorf("index connu non nomme : %+v", got)
	}
	if _, named := got["9"]; named {
		t.Error("un index hors table ne doit PAS etre nomme")
	}
	if abilityLabelsUsed(nil) != nil {
		t.Error("sans inventaire, pas de table inventee")
	}
}
