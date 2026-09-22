package grammar

import (
	"errors"
	"testing"
)

// type1_datums_test.go — LE TEMOIN SYNTHETIQUE DU BLOC DE TYPE 1 (lot 5.21.1).
//
// Il ECRIT un bloc avec la grammaire de `FUN_1429883ec` (R(6), R(8), R(32), 33 x R(1) LSB
// d abord, puis 256 bits par slot, puis 5 x R(32)) et verifie que le lecteur le referme au
// bit et rend les memes valeurs. Sans cet ecrivain, « bit-exact » ne serait qu une intention.

// ecrireBlocDeDatums fabrique un bloc de type 1 a partir d entrees et d une queue.
func ecrireBlocDeDatums(t *testing.T, ent []DatumEntry, queue [datumQueueMots]uint32) []byte {
	t.Helper()
	w := &bitWriter{}
	for _, e := range ent {
		w.bits(uint64(e.Drapeaux), datumDrapeauxBits)
		w.bits(uint64(e.Gen), datumGenBits)
		w.bits(uint64(e.Generation), datumEtatBits)
		for k := 0; k < datumMasqueBits; k++ { // LSB d abord : `1L << k` chez l ecrivain
			w.bit(e.MasqueVue >> uint(k))
		}
	}
	for _, e := range ent { // le masque de composants, BIT A BIT, LSB d abord
		for k := 0; k < datumBitmapBits; k++ {
			w.bit(e.Composants[k/64] >> uint(k%64))
		}
	}
	for _, q := range queue {
		w.bits(uint64(q), 32)
	}
	return w.buf
}

// TestBlocDeDatumsAllerRetour : ce que l ecrivain pose, le lecteur le rend — et le bloc ferme.
func TestBlocDeDatumsAllerRetour(t *testing.T) {
	ent := []DatumEntry{
		{Drapeaux: DatumAlloue | DatumPublie, Gen: 1, Generation: 2, MasqueVue: 1,
			Composants: [4]uint64{0b101010, 0, 1 << 63, 3}},
		{Drapeaux: DatumAlloue | DatumPublie | DatumLibere, Gen: 3, Generation: 4,
			MasqueVue: 1<<32 | 1<<3},
		{Drapeaux: 0, Gen: 0, Generation: DatumGenerationVierge, MasqueVue: 0},
	}
	queue := [datumQueueMots]uint32{1, 2, 3, 4, 5}
	b, err := LireBlocDeDatums(ecrireBlocDeDatums(t, ent, queue))
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	if len(b.Entrees) != len(ent) {
		t.Fatalf("cardinal derive %d, attendu %d", len(b.Entrees), len(ent))
	}
	for i, e := range ent {
		if b.Entrees[i] != e {
			t.Errorf("entree %d : %+v, attendu %+v", i, b.Entrees[i], e)
		}
	}
	if b.Bitmaps != len(ent) || b.BitmapBitsLeves != 6 {
		t.Errorf("bitmaps %d (%d bits leves), attendu %d et 6", b.Bitmaps, b.BitmapBitsLeves,
			len(ent))
	}
	// LE MASQUE EST INDEXE COMME `Archetype.Components` : le bit i est le composant i.
	for _, i := range []int{1, 3, 5, 191, 192, 193} {
		if !b.Entrees[0].Composant(i) {
			t.Errorf("composant %d : absent, attendu present", i)
		}
	}
	for _, i := range []int{0, 2, 64, 190, 194, 255, -1, 256} {
		if b.Entrees[0].Composant(i) {
			t.Errorf("composant %d : present, attendu absent", i)
		}
	}
	if b.Queue != queue {
		t.Errorf("queue %v, attendu %v", b.Queue, queue)
	}
	if b.BitsLus != len(ent)*datumPasParSlot+datumQueueBits {
		t.Errorf("bits lus %d, attendu %d", b.BitsLus, len(ent)*datumPasParSlot+datumQueueBits)
	}
	if b.Bourrage < 0 || b.Bourrage > datumBourrageMax {
		t.Errorf("bourrage %d hors [0 ; %d]", b.Bourrage, datumBourrageMax)
	}
}

// TestBlocDeDatumsVivante : le predicat de `FUN_1408f1730`, drapeau par drapeau.
func TestBlocDeDatumsVivante(t *testing.T) {
	cas := []struct {
		drapeaux uint8
		vivante  bool
	}{
		{DatumAlloue | DatumPublie, true},
		{DatumAlloue, false},                             // pas publiee
		{DatumPublie, false},                             // pas allouee
		{DatumAlloue | DatumPublie | DatumLibere, false}, // liberee
		{DatumAlloue | DatumPublie | DatumEcarte, false}, // ecartee
		{DatumAlloue | DatumPublie | 0x10, true},         // un bit libre ne decide rien
	}
	for _, c := range cas {
		if got := (DatumEntry{Drapeaux: c.drapeaux}).Vivante(); got != c.vivante {
			t.Errorf("drapeaux %#x : vivante = %v, attendu %v", c.drapeaux, got, c.vivante)
		}
	}
}

// TestBlocDeDatumsTailleRefusee : une taille qui ne referme pas la grammaire est une ERREUR,
// pas un cardinal devine.
func TestBlocDeDatumsTailleRefusee(t *testing.T) {
	for _, n := range []int{0, 1, 19, 20, 60} { // 20 octets = la seule queue, zero entree
		if _, err := LireBlocDeDatums(make([]byte, n)); !errors.Is(err, ErrBlocDeDatums) {
			t.Errorf("%d octets : err = %v, attendu %v", n, err, ErrBlocDeDatums)
		}
	}
}

// TestBlocDeDatumsTailleDuJeu : la taille CONSTANTE mesuree sur les deux films (343 019 o)
// rend EXACTEMENT 8 191 entrees — le cardinal de `DAT_144706100`. C est le ratchet de la
// grammaire : si une largeur bouge, ce compte ne tombe plus.
func TestBlocDeDatumsTailleDuJeu(t *testing.T) {
	const tailleMesuree = 343019
	b, err := LireBlocDeDatums(make([]byte, tailleMesuree))
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	if len(b.Entrees) != datumCapSlots {
		t.Fatalf("%d octets rendent %d entrees, attendu %d", tailleMesuree, len(b.Entrees),
			datumCapSlots)
	}
	if b.Bourrage != datumBourrageMax {
		t.Errorf("bourrage %d, attendu %d", b.Bourrage, datumBourrageMax)
	}
}
