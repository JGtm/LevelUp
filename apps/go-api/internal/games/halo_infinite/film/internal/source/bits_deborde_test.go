package source

// bits_deborde_test.go — LE VERDICT DU MOTEUR SUR UNE LECTURE QUI DEPASSE LE TAMPON
// (`FUN_14298816c` : bits consommes > 8 x taille), garde a cote des zeros de bourrage.

import "testing"

// TestDebordeEstCollant : lire exactement le tampon ne deborde pas ; un bit de plus deborde, et
// un repositionnement en arriere ne l efface pas. Les valeurs lues hors tampon restent des zeros.
func TestDebordeEstCollant(t *testing.T) {
	b := NewBits([]byte{0xff})
	if b.ReadBits(8) != 0xff || b.Deborde() {
		t.Fatal("huit bits lus sur un octet : aucun debordement")
	}
	if b.ReadBits(4) != 0 || !b.Deborde() {
		t.Fatal("quatre bits au-dela : zeros, et debordement")
	}
	b.SetBitPos(0)
	if !b.Deborde() || b.ReadBits(8) != 0xff {
		t.Fatal("le debordement est collant ; la relecture rend les memes valeurs")
	}
	c := NewBits([]byte{0x00})
	c.Skip(9)
	c.SetBitPos(2)
	if !c.Deborde() {
		t.Fatal("un saut au-dela du tampon deborde, meme quitte par un repositionnement")
	}
	if NewBits(nil).Deborde() {
		t.Fatal("un lecteur neuf ne deborde pas")
	}
}
