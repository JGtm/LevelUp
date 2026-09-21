package grammar

import "testing"

// components_biped_action_loop2_test.go — LE GARDE-RAIL DU SECOND TOUR D `i63` (lot 5.11.0-a).
//
// # CE QU IL FIGE, ET CONTRE QUOI
//
// Le compte du second tour d `i63 biped-action-component` a ete tenu pour IRRECUPERABLE pendant
// des mois (`bipedActionLoop2Count = 0`, commentaire « POPCOUNT of a 73-bit RAM bitmask ... It
// cannot be recovered from the delta bits »). C etait une doc inversee : `FUN_142f21b10` ECRIT
// ses trois `R(32)` dans le tampon que `FUN_1409fe718(etat, 0x49)` popcompte ensuite — le masque
// est le PREMIER CHAMP du composant, pas un etat de RAM.
//
// Ce test fige les deux choses qu un « nettoyage » pourrait reperdre :
//
//	LA FENETRE DE 73 BITS. Trente-deux bits du premier mot, trente-deux du second, et NEUF
//	  seulement du troisieme (`p[2] & 0x1ff`). Un test qui popcompterait les 96 bits passerait
//	  sur un masque creux et mentirait sur un masque dense.
//	LE COUT EN BITS DU TOUR. Chaque iteration lit `R(1)` et, si le bit est pose, `R(2)` de plus
//	  (`FUN_1406cf008` puis `FUN_14076e304`, tous deux relus au bit le 2026-09-21).
//
// Le cas commun MESURE reste le masque NUL : le film temoin `dad793c7` (un seul bipede, zero
// desync) ne porte qu une declaration d `i63` et son masque vaut zero — 96 + 4 + 96 = 196 bits.

// TestBipedActionLoop2CountFenetre73Bits fige la fenetre du popcount.
func TestBipedActionLoop2CountFenetre73Bits(t *testing.T) {
	cas := []struct {
		nom  string
		mots [3]uint64
		want int
	}{
		{"masque nul (cas commun mesure)", [3]uint64{0, 0, 0}, 0},
		{"premier mot plein", [3]uint64{0xffffffff, 0, 0}, 32},
		{"deux premiers mots pleins", [3]uint64{0xffffffff, 0xffffffff, 0}, 64},
		{"les neuf bits utiles du troisieme mot", [3]uint64{0, 0, 0x1ff}, 9},
		{"73 bits poses = le maximum", [3]uint64{0xffffffff, 0xffffffff, 0x1ff}, 73},
		// LA PIECE DU LOT : les 23 bits de poids fort du troisieme mot sont HORS FENETRE.
		// `FUN_1409fe718(p, 0x49)` masque par `0xffffffff >> (0x20 - (0x49 & 0x1f))` = `0x1ff`.
		{"les 23 bits de poids fort du troisieme mot ne comptent pas",
			[3]uint64{0, 0, 0xfffffe00}, 0},
		{"troisieme mot plein = neuf bits comptes", [3]uint64{0, 0, 0xffffffff}, 9},
		{"un bit par mot", [3]uint64{1, 1, 1}, 3},
	}
	for _, c := range cas {
		if got := bipedActionLoop2Count(c.mots); got != c.want {
			t.Errorf("%s : bipedActionLoop2Count(%08x %08x %08x) = %d, attendu %d",
				c.nom, c.mots[0], c.mots[1], c.mots[2], got, c.want)
		}
	}
}

// TestBipedActionCoutEnBits fige le cout en bits d `i63` en fonction de son masque de tete et de
// son compte de premier tour, sur des entrees synthetiques.
func TestBipedActionCoutEnBits(t *testing.T) {
	cas := []struct {
		nom    string
		mots   [3]uint64
		portes []bool // un par iteration du second tour : le bit de garde `R(1)`
		want   int
	}{
		// 96 (masque) + 4 (count1 = 0) + 96 (queue) = 196 : le cas commun.
		{"masque nul, count1 nul", [3]uint64{0, 0, 0}, nil, 196},
		// Un bit pose -> une iteration -> R(1) a zero -> 197.
		{"un bit, garde fermee", [3]uint64{1, 0, 0}, []bool{false}, 197},
		// Un bit pose -> une iteration -> R(1) a un puis R(2) -> 199.
		{"un bit, garde ouverte", [3]uint64{1, 0, 0}, []bool{true}, 199},
		// Trois bits, gardes mixtes : 196 + (1) + (1+2) + (1) = 201.
		{"trois bits, gardes mixtes", [3]uint64{7, 0, 0}, []bool{false, true, false}, 201},
		// Un bit HORS fenetre (bit 9 du troisieme mot = rang 73) : zero iteration.
		{"un bit hors fenetre", [3]uint64{0, 0, 0x200}, nil, 196},
	}
	for _, c := range cas {
		w := &bitWriterMSB{}
		for _, m := range c.mots {
			w.put(m, 32)
		}
		w.put(0, 4) // count1 = 0 : le premier tour n est pas le sujet de ce test
		for _, p := range c.portes {
			if p {
				w.put(1, 1)
				w.put(0, 2) // FUN_14076e304
			} else {
				w.put(0, 1)
			}
		}
		for i := 0; i < 3; i++ {
			w.put(0, 32) // la queue FUN_142f21b10
		}
		br := LecteurSur(w.buf)
		consumeBipedAction(br)
		if got := br.BitPos(); got != c.want {
			t.Errorf("%s : consumeBipedAction consomme %d bits, attendu %d", c.nom, got, c.want)
		}
	}
}
