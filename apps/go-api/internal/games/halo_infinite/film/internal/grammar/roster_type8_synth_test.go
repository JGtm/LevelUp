package grammar

// roster_type8_synth_test.go — LE TEST SYNTHETIQUE DU PAQUET DE TYPE 8.
//
// Il fabrique le flux EXACTEMENT comme `FUN_142987bd4` le lit (en-tete, puis N entrees dans
// l ordre de `FUN_1407eeba4`), et il oppose au port les deux seules choses qui font foi : les
// VALEURS des champs nommes, et le NOMBRE DE BITS consomme. Le decompte de bits est calcule ici
// depuis les largeurs du desassemblage — si une largeur du port derive, l egalite casse.

import "testing"

// rosterSynthEntree : ce qu une entree synthetique porte.
type rosterSynthEntree struct {
	porte          bool // version >= 0x15 : `true` = porte OUVERTE (bit 1, pas de R(5))
	option         uint64
	identite       uint64
	drapeaux       []bool
	octets         []byte
	mots           []uint32
	etiquette      []uint16 // sans le terminateur : le writer le pose
	representation uint32
	cle            uint64
	f10, f14       uint64
	f6, octet, f7  uint64
	bit            bool
}

// rosterEcrire pose un paquet de type 8 complet et rend ses octets avec le nombre de bits ECRITS.
func rosterEcrire(entrees []rosterSynthEntree, versionAvecPorte bool) ([]byte, int) {
	w := &bitWriter{}
	w.bits(uint64(len(entrees)), rosterCountBits)
	for _, e := range entrees {
		if len(e.drapeaux) == 0 {
			// `FUN_142bdeddc` rend `R(11) + 1` : un cardinal nul n est pas representable, et le
			// modele synthetique doit porter la meme contrainte que l ecrivain.
			e.drapeaux = []bool{false}
		}
		if versionAvecPorte {
			if e.porte {
				w.bit(1)
			} else {
				w.bit(0)
				w.bits(e.option, rosterOptWidth)
			}
		}
		rosterEcrireMot(w, e.identite, rosterIdentityBits)
		// FUN_1407f01c4
		w.bits(uint64(len(e.drapeaux)-1), rosterFlagCountBits) // FUN_142bdeddc rend R(11) + 1
		for _, d := range e.drapeaux {
			if d {
				w.bit(1)
			} else {
				w.bit(0)
			}
		}
		w.bits(uint64(len(e.octets)), rosterByteLenBits)
		for _, o := range e.octets {
			w.bits(uint64(o), 8)
		}
		w.bits(uint64(len(e.mots)), rosterWordLenBits)
		for _, m := range e.mots {
			w.bits(uint64(m), 32)
		}
		rosterEcrireQueue(w, e)
	}
	return w.buf, w.n
}

// rosterEcrireQueue pose la suite de `FUN_1407eeba4` apres `FUN_1407f01c4`.
func rosterEcrireQueue(w *bitWriter, e rosterSynthEntree) {
	w.bits(0, rosterBlobABits)
	for _, u := range e.etiquette {
		w.bits(uint64(u), rosterLabelUnitBits)
	}
	if len(e.etiquette) < rosterLabelMaxWords {
		w.bits(0, rosterLabelUnitBits) // le terminateur, LU par le jeu
	}
	w.bits(0, rosterBlobBBits)
	w.bits(uint64(e.representation), rosterRepresentationBits)
	rosterEcrireMot(w, e.cle, rosterKeyBits)
	w.bits(e.f10, rosterF10Bits)
	w.bits(e.f14, rosterF14Bits)
	w.bits(e.f6, rosterF6Bits)
	w.bits(e.octet, rosterByteBits)
	w.bits(e.f7, rosterF7Bits)
	if e.bit {
		w.bit(1)
	} else {
		w.bit(0)
	}
	w.bits(0, rosterBlobCBits)
	w.bits(0, rosterBlobDBits)
}

// rosterEcrireMot pose un mot de `FUN_1406d676c` : les octets du champ, du moins significatif au
// plus significatif, dans l ordre du flux (cf. `rosterLireMot`).
func rosterEcrireMot(w *bitWriter, v uint64, bits int) {
	for i := 0; i < bits/8; i++ {
		w.bits((v>>uint(8*i))&0xff, 8)
	}
}

// TestRosterType8Synthetique : deux entrees, la porte de version ouverte puis fermee, et le
// decompte de bits au bit.
func TestRosterType8Synthetique(t *testing.T) {
	entrees := []rosterSynthEntree{
		{
			porte: true, identite: 0x0009_0000_1234_5678,
			drapeaux: []bool{true, false, true}, octets: []byte{0xAB, 0x01},
			mots:           []uint32{0xDEADBEEF},
			etiquette:      []uint16{'J', 'G', 't', 'm'},
			representation: 0x0BADF00D, cle: 0x0009_0000_ABCD_EF01,
			f10: 0x2AA, f14: 0x1555, f6: 0x21, octet: 0x5A, f7: 0x55, bit: true,
		},
		{
			porte: false, option: 0x13, identite: 2,
			drapeaux:       []bool{false},
			etiquette:      []uint16{'a'},
			representation: 7, cle: 9,
			f6: 0, octet: 0xFF, bit: false,
		},
	}
	pay, bits := rosterEcrire(entrees, true)
	got, bilan := DecodeRoster(pay, rosterOptGateVersion)
	if bilan.Debordement {
		t.Fatalf("debordement inattendu : %+v", bilan)
	}
	if bilan.Annonce != 2 || bilan.Entrees != 2 {
		t.Fatalf("annonce/entrees = %d/%d, attendu 2/2", bilan.Annonce, bilan.Entrees)
	}
	if bilan.BitsLus != bits {
		t.Errorf("bits lus = %d, ECRITS = %d (une largeur du port derive)", bilan.BitsLus, bits)
	}
	rosterVerifier(t, got[0], entrees[0], -1)
	rosterVerifier(t, got[1], entrees[1], 0x13)
}

// rosterVerifier oppose une entree decodee a son modele.
func rosterVerifier(t *testing.T, g RosterEntry, e rosterSynthEntree, option int) {
	t.Helper()
	if g.Identite != e.identite || g.Cle != e.cle {
		t.Errorf("identite/cle = %#x/%#x, attendu %#x/%#x", g.Identite, g.Cle, e.identite, e.cle)
	}
	if g.Etiquette != string(rosterRunes(e.etiquette)) {
		t.Errorf("etiquette = %q, attendu %q", g.Etiquette, string(rosterRunes(e.etiquette)))
	}
	if g.Representation != e.representation {
		t.Errorf("representation = %#x, attendu %#x", g.Representation, e.representation)
	}
	if g.Option != option {
		t.Errorf("option = %d, attendu %d", g.Option, option)
	}
	if uint64(g.F10) != e.f10 || uint64(g.F14) != e.f14 {
		t.Errorf("F10/F14 = %#x/%#x, attendu %#x/%#x", g.F10, g.F14, e.f10, e.f14)
	}
	// `FUN_1407ef724` decremente : le port doit rendre `R(6) - 1` modulo 256.
	if g.F6 != uint8(e.f6)-1 {
		t.Errorf("F6 = %#x, attendu %#x (R(6) - 1)", g.F6, uint8(e.f6)-1)
	}
	if uint64(g.Octet) != e.octet || uint64(g.F7) != e.f7 || g.Bit != e.bit {
		t.Errorf("octet/F7/bit = %#x/%#x/%v, attendu %#x/%#x/%v",
			g.Octet, g.F7, g.Bit, e.octet, e.f7, e.bit)
	}
	if len(g.Drapeaux) != len(e.drapeaux) || len(g.Octets) != len(e.octets) ||
		len(g.Mots) != len(e.mots) {
		t.Errorf("cardinaux = %d/%d/%d, attendu %d/%d/%d",
			len(g.Drapeaux), len(g.Octets), len(g.Mots),
			len(e.drapeaux), len(e.octets), len(e.mots))
	}
}

// rosterRunes decode une etiquette de reference sans passer par le port.
func rosterRunes(u []uint16) []rune {
	out := make([]rune, 0, len(u))
	for _, w := range u {
		out = append(out, rune(w))
	}
	return out
}

// TestRosterType8EtiquettePleine : seize unites non nulles ne portent PAS de terminateur
// (`FUN_1407f0094` s arrete sur son cardinal), donc la chaine coute exactement 256 bits.
func TestRosterType8EtiquettePleine(t *testing.T) {
	pleine := make([]uint16, rosterLabelMaxWords)
	for i := range pleine {
		pleine[i] = uint16('A' + i)
	}
	pay, bits := rosterEcrire([]rosterSynthEntree{
		{porte: true, drapeaux: []bool{false}, etiquette: pleine}}, true)
	got, bilan := DecodeRoster(pay, rosterOptGateVersion)
	if bilan.BitsLus != bits {
		t.Errorf("bits lus = %d, ECRITS = %d", bilan.BitsLus, bits)
	}
	if len(got) != 1 || len([]rune(got[0].Etiquette)) != rosterLabelMaxWords {
		t.Fatalf("etiquette = %q (%d unites), attendu %d",
			got[0].Etiquette, len([]rune(got[0].Etiquette)), rosterLabelMaxWords)
	}
}

// TestRosterType8SansPorte : sous une version de format < 0x15, l entree N OUVRE PAS la porte de
// tete — un bit de moins par entree, et le port doit le savoir.
func TestRosterType8SansPorte(t *testing.T) {
	e := []rosterSynthEntree{{identite: 42, cle: 43, drapeaux: []bool{true},
		etiquette: []uint16{'x'}}}
	pay, bits := rosterEcrire(e, false)
	got, bilan := DecodeRoster(pay, rosterOptGateVersion-1)
	if bilan.BitsLus != bits {
		t.Errorf("bits lus = %d, ECRITS = %d", bilan.BitsLus, bits)
	}
	if len(got) != 1 || got[0].Identite != 42 || got[0].Cle != 43 || got[0].Option != -1 {
		t.Fatalf("entree = %+v", got[0])
	}
}

// TestRosterType8Debordement : un cardinal qui depasse ce que le payload porte est un
// DEBORDEMENT — le verdict du jeu (@142987d6e), et la marche s arrete au payload.
func TestRosterType8Debordement(t *testing.T) {
	w := &bitWriter{}
	w.bits(1000, rosterCountBits)
	w.bits(0, 64)
	_, bilan := DecodeRoster(w.buf, rosterOptGateVersion)
	if !bilan.Debordement {
		t.Fatalf("un cardinal de 1000 sur %d octets doit deborder : %+v", len(w.buf), bilan)
	}
	if bilan.Entrees >= bilan.Annonce {
		t.Errorf("entrees = %d, annonce = %d : la marche doit s arreter au payload",
			bilan.Entrees, bilan.Annonce)
	}
}
