package grammar

import "testing"

// frame_vue_controle_test.go — L ENTREE DE CONTROLE LUE EN ENTIER (lot M4b). Le flux est ecrit bit a
// bit dans l ordre de l ecrivain (`FUN_1406d0388`, `FUN_1406cd860`, `FUN_1406d025c`) : chaque
// branche que le port refusait avant ce lot y est OUVERTE, et la lecture doit aller jusqu au
// terminateur de la vue et fermer le paquet.

// entreeCompleteOpts decrit l entree ecrite par [ecrireEntreeComplete].
type entreeCompleteOpts struct {
	index      uint64
	gachette   uint64 // les six bits de gachettes, main 0 puis main 1, entree 0 d abord
	genreCible uint64 // genre de la queue `FUN_140c9e990` : 1 (categorie 1) ou 2 (categorie 2)
	modeVect   uint64 // mode du vecteur `FUN_1431a0cbc` : 1 (R(19)), 2 ou 3 (constante)
}

// ecrireEntreeComplete ecrit UNE entree `kind 0` dont les trois branches refusees jusqu ici sont
// ouvertes (troisieme champ, +0x10, drapeaux +0x14), puis son bloc d action.
func ecrireEntreeComplete(w *bitWriter, o entreeCompleteOpts) {
	w.bit(1)                         // la boucle de la vue C continue
	w.bits(kindVueCControle, 2)      // kind 0
	w.bit(0)                         // FUN_1406cdc04 : pas de R(7)
	w.bits(o.index, 5)               // index de controle
	w.bit(1)                         // a : le bloc de 0x68
	w.bit(1)                         // second champ present
	w.bits(1, largeurCourteControle) // ... R(2)
	w.bits(0x1F, 6)                  // couple analogique
	w.bits(0x1F, 6)
	w.bit(1) // troisieme champ present (refuse avant le lot M4b)
	w.bits(3, largeurTroisiemeChamp)
	w.bit(1) // +0x10 present (refuse avant le lot M4b)
	w.bits(5, largeurChamp10)
	w.bit(1) // drapeaux +0x14 presents (refuses avant le lot M4b) : le bloc d action suit
	w.bits(9, largeurDrapeaux14)
	ecrireBlocDAction(w, o)
	w.bit(0) // b : pas de bloc de 0xbc
}

// ecrireBlocDAction ecrit `FUN_1406d025c` avec une gachette de la main 0, le bloc de visee (mode du
// vecteur choisi) et la queue typee (genre choisi).
func ecrireBlocDAction(w *bitWriter, o entreeCompleteOpts) {
	w.bit(1) // garde
	w.bit(1) // gachettes
	w.bits(o.gachette, 6)
	w.bit(0) // pas de barillets
	w.bit(1) // bloc de visee
	w.bits(0, 2)
	w.bit(0) // FUN_1431a0bbc : absent
	w.bit(0) // FUN_1431a0abc : absent
	w.bits(o.modeVect, 2)
	if o.modeVect == modeVecteurDirection {
		w.bits(0x12345, largeurVecteurDirection)
	}
	w.bits(0, largeurQueue1406d0f20)
	if o.gachette>>3 != 0 { // une gachette de la main 0 ouvre son champ d arme
		w.bit(0)
		w.bits(1, largeurIndexArme)
	}
	if o.gachette&0b111 != 0 { // main 1
		w.bit(1)
	}
	w.bit(1) // queue : garde
	w.bits(o.genreCible, 2)
	switch o.genreCible {
	case genreCibleCategorie1:
		w.bit(1)            // sonde : la largeur tombe a 9
		w.bits(0x55<<2, 11) // valeur + generation
		w.bit(1)
		w.bits(0x2A, 6)
	case genreCibleCategorie2:
		w.bits(0x77<<2, 10) // categorie 2 : 8 bits + generation
	}
	w.bit(1) // f0 : FUN_140c9e738 suit
	w.bit(1) // FUN_140c9e738 : direction constante
}

// paquetDeVueC rend un payload qui ne porte que la vue C (terminateur compris), complete a
// l octet.
func paquetDeVueC(entrees ...entreeCompleteOpts) []byte {
	w := &bitWriter{}
	for _, e := range entrees {
		ecrireEntreeComplete(w, e)
	}
	w.bit(0) // terminateur de la vue C
	return w.buf
}

// TestVueCLitLEntreeComplete : les trois branches de `FUN_1406cd860` et les deux sous-lecteurs
// corriges du bloc d action se lisent jusqu au terminateur ; le paquet se FERME ; l entree rend son
// index et sa gachette principale. Avant le lot M4b, le troisieme champ arretait la vue.
func TestVueCLitLEntreeComplete(t *testing.T) {
	for _, tc := range []struct {
		nom          string
		genre, vecte uint64
	}{
		{"genre 1 (categorie 1) et vecteur en direction", genreCibleCategorie1, modeVecteurDirection},
		{"genre 2 (categorie 2) et vecteur constant", genreCibleCategorie2, 2},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			pay := paquetDeVueC(
				entreeCompleteOpts{index: 2, gachette: 0b100000, genreCible: tc.genre, modeVect: tc.vecte},
				entreeCompleteOpts{index: 5, gachette: 0, genreCible: tc.genre, modeVect: tc.vecte})
			br := LecteurSur(pay)
			flux := consumeVueC(br, len(pay)*8)
			if !flux.Porte || flux.Arret != ArretVueCAucun {
				t.Fatalf("vue C arretee (porte %v, arret %d) : une branche n est pas lue", flux.Porte,
					flux.Arret)
			}
			if !vueCFermee(pay, br.BitPos()) {
				t.Fatalf("paquet non ferme : curseur %d sur %d bits", br.BitPos(), len(pay)*8)
			}
			if len(flux.Entrees) != 2 {
				t.Fatalf("%d entrees lues, attendu 2", len(flux.Entrees))
			}
			e := flux.Entrees[0]
			if e.Index != 2 || !e.Bloc || !e.Action.Present || e.Action.Gachettes[0] != 1 {
				t.Errorf("entree 0 = %+v, attendu index 2, gachette principale de la main 0", e)
			}
			if e.Action.Arme[0] != 1 || e.Action.Arme[1] != ArmeAbsente {
				t.Errorf("armes %v, attendu [1 absente]", e.Action.Arme)
			}
			if f := flux.Entrees[1]; f.Index != 5 || f.Action.Tire() {
				t.Errorf("entree 1 = %+v, attendu index 5 sans tir", f)
			}
		})
	}
}

// TestVueCNommeSesArrets : un kind 1 ou 2, et le bloc de 0xbc, arretent la vue sur une cause
// NOMMEE — jamais un « pas de tir ».
func TestVueCNommeSesArrets(t *testing.T) {
	w := &bitWriter{}
	w.bit(1)
	w.bits(kindVueCSecond, 2)
	w.bits(0, 16)
	br := LecteurSur(w.buf)
	if f := consumeVueC(br, len(w.buf)*8); f.Porte || f.Arret != ArretVueCKindNonPorte {
		t.Errorf("kind 1 : porte %v arret %d, attendu arret %d", f.Porte, f.Arret, ArretVueCKindNonPorte)
	}
	w = &bitWriter{}
	w.bit(1)
	w.bits(kindVueCControle, 2)
	w.bit(0)
	w.bits(3, 5)
	w.bit(0) // pas de bloc de 0x68
	w.bit(1) // mais le bloc de 0xbc
	w.bits(0, 16)
	br = LecteurSur(w.buf)
	if f := consumeVueC(br, len(w.buf)*8); f.Porte || f.Arret != ArretVueCBlocBC {
		t.Errorf("bloc 0xbc : porte %v arret %d, attendu arret %d", f.Porte, f.Arret, ArretVueCBlocBC)
	}
}

// TestLaMarchePublieLeVerdictDeLaVueC : de bout en bout par la marche du frame-processeur — vue A
// vide, vue B vide, vue C portant une entree qui tire — le crochet recoit UN verdict par paquet :
// atteint et ferme avec son entree ; et le MEME paquet suivi d un octet de trop ne ferme plus, et
// ne rend AUCUNE entree.
func TestLaMarchePublieLeVerdictDeLaVueC(t *testing.T) {
	w := &bitWriter{}
	w.bit(1)     // configuration
	w.bit(0)     // vue A vide
	w.bits(0, 3) // vue B : record de type 0, la fin
	ecrireEntreeComplete(w, entreeCompleteOpts{index: 2, gachette: 0b100000,
		genreCible: genreCibleCategorie1, modeVect: modeVecteurDirection})
	w.bit(0) // terminateur de la vue C
	ferme := append([]byte(nil), w.buf...)
	trop := append(append([]byte(nil), w.buf...), 0)
	var recus []LectureVueC
	obs := NouvelleObservation()
	obs.VueControleHook = func(l LectureVueC) { recus = append(recus, l) }
	cfg := DefaultFrameConfig()
	cfg.Obs = obs
	monde := NewWorld(nil)
	DecodeFrameViewsCurseur(ferme, monde, cfg, MovementStateViews, cfg.PacketPreambleBits)
	if len(recus) != 1 || !recus[0].Atteinte || !recus[0].Fermee || len(recus[0].Entrees) != 1 {
		t.Fatalf("paquet ferme : verdicts %+v, attendu un seul, ferme, avec une entree", recus)
	}
	if e := recus[0].Entrees[0]; e.Index != 2 || e.Action.Gachettes[0] != 1 {
		t.Errorf("entree %+v, attendu l index 2 gachette tenue", e)
	}
	recus = nil
	DecodeFrameViewsCurseur(trop, monde, cfg, MovementStateViews, cfg.PacketPreambleBits)
	if len(recus) != 1 || !recus[0].Atteinte || recus[0].Fermee || len(recus[0].Entrees) != 0 {
		t.Errorf("octet de trop : verdicts %+v, attendu atteint, NON ferme, sans entree", recus)
	}
}
