package grammar

// roster_type8_synth_test.go — LE TEST SYNTHETIQUE DU PAQUET DE TYPE 8.
//
// Il fabrique le flux EXACTEMENT comme `FUN_142987bd4` le lit (le mot de tete, puis N entrees :
// la porte de version, le XUID en ordre d octets du flux, et le corps commun de
// `player_table_record.go`), et il oppose au port les deux seules choses qui font foi : les
// VALEURS des champs nommes et le NOMBRE DE BITS consomme.
//
// L ECRIVAIN SYNTHETIQUE EMPLOIE LES MEMES CONSTANTES QUE LE LECTEUR (`slot*Bits`), parce que
// deux tables de largeurs qui se recopient divergent. Ce que le test verifie est donc le CADRAGE
// de l entree et l ordre des champs, pas les largeurs du corps : celles-la sont figees par la
// table de `chunk_00` et son corpus de 1 351 films.

import "testing"

// rosterSynthEntree : ce qu une entree synthetique porte.
type rosterSynthEntree struct {
	porte          bool // version >= 0x15 : `true` = porte OUVERTE (bit 1, pas de R(5))
	option         uint64
	identite       uint64
	drapeaux       int // le cardinal du masque ; `FUN_142bdeddc` rend R(11) + 1, donc >= 1
	octets         int // la longueur N de la premiere liste, en octets
	mots           int // la longueur M de la seconde liste, en mots de 32 bits
	etiquette      []uint16
	representation uint32
	cle            uint64
	f10, f14       uint64
	f6, octet, f7  uint64
	bit            bool
}

// rosterPersoSynth : une largeur de bloc de personnalisation courte, pour que les tampons de test
// restent petits. La vraie valeur vient du profil de build (1 852 octets sur `HI_1_13_0`).
const rosterPersoSynth = 64

// rosterEcrire pose un paquet de type 8 complet et rend ses octets avec le nombre de bits ECRITS.
func rosterEcrire(entrees []rosterSynthEntree, avecPorte bool, persoBits int) ([]byte, int) {
	w := &bitWriter{}
	w.bits(uint64(len(entrees)), rosterCountBits)
	for _, e := range entrees {
		if e.drapeaux < 1 {
			e.drapeaux = 1 // `FUN_142bdeddc` rend R(11) + 1 : un cardinal nul n est pas ecrivable
		}
		if avecPorte {
			if e.porte {
				w.bit(1)
			} else {
				w.bit(0)
				w.bits(e.option, rosterOptWidth)
			}
		}
		rosterEcrireMot(w, e.identite, slotXUIDBits)
		rosterEcrireCorps(w, e, persoBits)
	}
	return w.buf, w.n
}

// rosterEcrireMot pose un champ de `FUN_1406d676c` : les octets du champ, du moins significatif
// au plus significatif, dans l ordre du flux (cf. `rosterOctetsDuFlux`).
func rosterEcrireMot(w *bitWriter, v uint64, bits int) {
	for i := 0; i < bits/8; i++ {
		w.bits((v>>uint(8*i))&0xff, 8)
	}
}

// rosterEcrireCorps pose le corps que `decodeSlotListes` puis `decodeSlotCorps` lisent.
func rosterEcrireCorps(w *bitWriter, e rosterSynthEntree, persoBits int) {
	w.bits(uint64(e.drapeaux-1), slotMaskPrefixBits)
	w.bits(0, e.drapeaux)
	w.bits(uint64(e.octets), slotListNBits)
	w.bits(0, e.octets*8)
	w.bits(uint64(e.mots), slotListMBits)
	w.bits(0, e.mots*32)
	w.bits(0, slotBloc104Bits)
	for _, u := range e.etiquette {
		w.bits(uint64(u), 16)
	}
	if len(e.etiquette) < slotGamertagMaxUnits {
		w.bits(0, 16) // le terminateur, LU par le jeu
	}
	w.bits(0, slotBloc16Bits)
	w.bits(uint64(e.representation), slotReprBits)
	w.bits(e.cle, slotQ64Bits)
	w.bits(e.f10, 10)
	w.bits(e.f14, 14)
	w.bits(e.f6, 6)
	w.bits(e.octet, 8)
	w.bits(e.f7, 7)
	if e.bit {
		w.bit(1)
	} else {
		w.bit(0)
	}
	w.bits(0, persoBits)
	w.bits(0, slotBloc44Bits)
}

// TestRosterType8Synthetique : deux entrees, la porte de version ouverte puis fermee, et le
// decompte de bits au bit.
func TestRosterType8Synthetique(t *testing.T) {
	entrees := []rosterSynthEntree{
		{
			porte: true, identite: 0x0009_01f3_c4d0_01e1,
			drapeaux: 3, octets: 2, mots: 1,
			etiquette:      []uint16{'J', 'G', 't', 'm'},
			representation: 0x0BADF00D, cle: 0x0009_0000_ABCD_EF01,
			f10: 0x2AA, f14: 0x1555, f6: 0x21, octet: 0x5A, f7: 0x55, bit: true,
		},
		{
			porte: false, option: 0x13, identite: 0x0009_0000_0000_0002,
			etiquette:      []uint16{'a'},
			representation: 7, cle: 9,
			octet: 0xFF,
		},
	}
	pay, bits := rosterEcrire(entrees, true, rosterPersoSynth)
	got, bilan := DecodeRoster(pay, rosterOptGateVersion, rosterPersoSynth)
	if bilan.Debordement || bilan.Refusees != 0 {
		t.Fatalf("debordement ou refus inattendu : %+v", bilan)
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
	if g.Identite != e.identite || g.Joueur.XUID != e.identite {
		t.Errorf("identite = %#x / XUID %#x, attendu %#x", g.Identite, g.Joueur.XUID, e.identite)
	}
	if g.Joueur.Gamertag != string(rosterRunes(e.etiquette)) {
		t.Errorf("gamertag = %q, attendu %q", g.Joueur.Gamertag, string(rosterRunes(e.etiquette)))
	}
	if g.Joueur.Shorts.Repr != e.representation {
		t.Errorf("representation = %#x, attendu %#x", g.Joueur.Shorts.Repr, e.representation)
	}
	if g.Joueur.Shorts.Q64 != e.cle {
		t.Errorf("cle = %#x, attendu %#x", g.Joueur.Shorts.Q64, e.cle)
	}
	if g.Option != option {
		t.Errorf("option = %d, attendu %d", g.Option, option)
	}
	if uint64(g.Joueur.Shorts.F10) != e.f10 || uint64(g.Joueur.Shorts.F14) != e.f14 {
		t.Errorf("F10/F14 = %#x/%#x, attendu %#x/%#x",
			g.Joueur.Shorts.F10, g.Joueur.Shorts.F14, e.f10, e.f14)
	}
	// Le champ de 6 bits est rendu DECREMENTE (`DEC R9B` @1407ef764 chez l ecrivain du type 8,
	// `- 1` dans le corps depuis le lot 1.5.2).
	if g.Joueur.Shorts.F6 != int(e.f6)-1 {
		t.Errorf("F6 = %d, attendu %d (R(6) - 1)", g.Joueur.Shorts.F6, int(e.f6)-1)
	}
	if uint64(g.Joueur.Shorts.F8) != e.octet || uint64(g.Joueur.Shorts.F7) != e.f7 {
		t.Errorf("F8/F7 = %#x/%#x, attendu %#x/%#x",
			g.Joueur.Shorts.F8, g.Joueur.Shorts.F7, e.octet, e.f7)
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
	pleine := make([]uint16, slotGamertagMaxUnits)
	for i := range pleine {
		pleine[i] = uint16('A' + i) //nolint:gosec // 'A'+15 tient dans un u16
	}
	pay, bits := rosterEcrire([]rosterSynthEntree{{porte: true, etiquette: pleine}},
		true, rosterPersoSynth)
	got, bilan := DecodeRoster(pay, rosterOptGateVersion, rosterPersoSynth)
	if bilan.BitsLus != bits {
		t.Errorf("bits lus = %d, ECRITS = %d", bilan.BitsLus, bits)
	}
	if len(got) != 1 || len([]rune(got[0].Joueur.Gamertag)) != slotGamertagMaxUnits {
		t.Fatalf("gamertag = %q (%d unites), attendu %d",
			got[0].Joueur.Gamertag, len([]rune(got[0].Joueur.Gamertag)), slotGamertagMaxUnits)
	}
}

// TestRosterType8SansPorte : sous une version de format < 0x15, l entree N OUVRE PAS la porte de
// tete — un bit de moins par entree, et le port doit le savoir.
func TestRosterType8SansPorte(t *testing.T) {
	e := []rosterSynthEntree{{identite: 42, cle: 43, etiquette: []uint16{'x'}}}
	pay, bits := rosterEcrire(e, false, rosterPersoSynth)
	got, bilan := DecodeRoster(pay, rosterOptGateVersion-1, rosterPersoSynth)
	if bilan.BitsLus != bits {
		t.Errorf("bits lus = %d, ECRITS = %d", bilan.BitsLus, bits)
	}
	if len(got) != 1 || got[0].Identite != 42 || got[0].Joueur.Shorts.Q64 != 43 ||
		got[0].Option != -1 {
		t.Fatalf("entree = %+v", got[0])
	}
}

// TestRosterType8Debordement : un cardinal qui depasse ce que le payload porte est un
// DEBORDEMENT — le verdict du jeu (@142987d6e) —, et la marche s arrete a la borne du payload
// sans rendre d entree a moitie lue.
func TestRosterType8Debordement(t *testing.T) {
	w := &bitWriter{}
	w.bits(1000, rosterCountBits)
	w.bits(0, 64)
	got, bilan := DecodeRoster(w.buf, rosterOptGateVersion, rosterPersoSynth)
	if !bilan.Debordement {
		t.Fatalf("un cardinal de 1000 sur %d octets doit deborder : %+v", len(w.buf), bilan)
	}
	if bilan.Entrees != len(got) || bilan.Entrees >= bilan.Annonce || bilan.Refusees != 1 {
		t.Errorf("entrees = %d (rendues %d), annonce = %d, refusees = %d",
			bilan.Entrees, len(got), bilan.Annonce, bilan.Refusees)
	}
}
