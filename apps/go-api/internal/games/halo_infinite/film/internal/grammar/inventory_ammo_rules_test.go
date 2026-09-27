package grammar

// inventory_ammo_rules_test.go — LE BLOC MUNITIONS (i30..i42) : ses proprietes, pas ses valeurs
// de sortie.
//
// DESCENDU DE `replay/inventory_test.go` AU LOT J4.2 (2026-09-26) avec la lecture de
// l inventaire. Le lecteur de flux fabrique est le `bitWriter` du paquet
// (`frame_chain_infer_test.go`), MEME forme (tampon, compte de bits, poids fort d abord) : la
// methode `put` ci-dessous est l ecriture que ces tests employaient, posee sur lui plutot qu une
// seconde copie du type.
//
// CE QUE CES TESTS PROTÈGENT AVANT TOUT : la distinction entre « non lu », « zéro » et
// « aucune ». Elle est l'objet même de ce calque, et c'est elle qu'un refactor casse en
// silence — un `omitempty` mal placé, un `if !v` au lieu d'un `if v == nil`, et une arme sans
// chargeur devient une arme au chargeur vide.

import "testing"

// put ecrit les `width` bits de poids faible de v, poids fort d abord.
func (w *bitWriter) put(v uint32, width int) { w.bits(uint64(v), width) }

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
