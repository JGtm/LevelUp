package filmdec

// transloc_events_test.go — la forme du record d'événement 117, figée sur des octets
// synthétiques : le décodage sur film réel est couvert par le test gaté
// (transloc_exemption_film_test.go) et par la validation Dynasty du plan (R1 : 18/18).

import "testing"

// translocPacket fabrique un paquet dont la tête est un événement 117 désignant idx.
//
// Trame : [1 config][1 présence][7 bits type][1 porte ref0][8 bits index][2 bits gen] —
// le type 117 (1110101) met les octets [0xFA, 0b1_1_xxxxxx...] : son bit de poids faible
// est le PREMIER bit du deuxième octet (rapport R1 §4.2).
func translocPacket(idx uint32) []byte {
	pay := make([]byte, 4)
	at := 0
	put := func(n int, v uint32) {
		for i := 0; i < n; i++ {
			if v>>(uint(n-1-i))&1 == 1 {
				pay[(at+i)/8] |= 1 << (7 - uint((at+i)%8))
			}
		}
		at += n
	}
	put(1, 1)   // configuration
	put(1, 1)   // présence : la liste porte un événement
	put(7, 117) // le type
	put(1, 1)   // porte de ref0
	put(8, idx) // l'index de l'unité (domaine 2)
	put(2, 1)   // génération
	return pay
}

func TestDecodeTranslocHead(t *testing.T) {
	pay := translocPacket(23)
	if pay[0] != translocFamilyByte {
		t.Fatalf("premier octet fabriqué %#x, attendu %#x — la trame d'essai contredit la forme"+
			" mesurée du record", pay[0], translocFamilyByte)
	}
	ev, ok := decodeTranslocHead(pay, 42_000)
	if !ok || ev.Slot != 535 || ev.TimestampUS != 42_000 {
		t.Fatalf("décodage = %+v (ok=%v), attendu slot 535 (index 23 + base 512) @42000us", ev, ok)
	}
}

func TestDecodeTranslocHeadRefuse(t *testing.T) {
	t.Run("type voisin", func(t *testing.T) {
		pay := translocPacket(23)
		pay[1] &^= 0x80 // le bit de poids faible du type : 117 -> 116
		if _, ok := decodeTranslocHead(pay, 0); ok {
			t.Fatal("un type 116 a été lu comme une téléportation")
		}
	})
	t.Run("liste vide", func(t *testing.T) {
		pay := translocPacket(23)
		pay[0] &^= 0x40 // bit de présence à 0
		if _, ok := decodeTranslocHead(pay, 0); ok {
			t.Fatal("une liste vide a rendu un événement")
		}
	})
	t.Run("unite non designee", func(t *testing.T) {
		pay := translocPacket(23)
		pay[1] &^= 0x40 // porte de ref0 à 0
		if _, ok := decodeTranslocHead(pay, 0); ok {
			t.Fatal("un événement sans référence d'unité a rendu un slot — un slot ne se devine pas")
		}
	})
}
