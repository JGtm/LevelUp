package grammar

import "testing"

// debut_de_liste_test.go — le debut de liste par la CHAINE de ses records NEW de tete
// (`debut_de_liste.go`, lot M4b), sur des trames synthetiques a index de 13 bits.

// neuf13 ecrit un record NEW d un archetype SANS composant ni etat par defaut : type `0` puis
// `01`, slot R(13), generation R(2), archetype R(6), porte R(1)=0, masque vide.
func (w *bitWriter) neuf13(slot uint32, ti uint64) {
	w.bit(0)
	w.bits(recNew, 2)
	w.bits(uint64(slot), 13)
	w.bits(1, 2)
	w.bits(ti, 6)
	w.bit(0)     // porte du record NEW
	w.bit(0)     // masque clairseme
	w.bits(0, 3) // aucun composant
}

// delta13 ecrit un record DELTA vide : type `1`, slot R(13), generation R(2), selecteur de base
// R(1)=0, masque vide.
func (w *bitWriter) delta13(slot uint32) {
	w.bit(1)
	w.bits(uint64(slot), 13)
	w.bits(1, 2)
	w.bit(0)
	w.bit(0)
	w.bits(0, 3)
}

// mondeDeTete rend un monde ou les slots 123 et 124 sont vivants (archetype 4) et ou le slot 300
// est, dans les images-cles, un slot de l archetype 2 : sa BANDE.
func mondeDeTete() *World {
	w := NewWorld(&Registry{Archetypes: []Archetype{{Index: 0}, {Index: 1}, {Index: 2}, {Index: 3}, {Index: 4}}})
	w.BindImageCle(1, 123, 4)
	w.BindImageCle(1, 124, 4)
	t := NouvelleTableAnticipee()
	t.archetypesDuSlot[300] = map[uint32]bool{2: true}
	w.PoserTableAnticipee(t)
	return w
}

var cadreDeTete = FrameConfig{IDLowBits: 13, Profil: ProfilDeBalayageParDefaut()}

// TestLaListeCommenceASonRecordNeufDeTete : un paquet dont la liste commence par la creation du
// slot 300, puis un delta du slot 124, puis le record du slot 123 ou le localisateur se pose. La
// chaine qui part du NEW finit EXACTEMENT sur ce record : la liste commence au NEW. Decale d un
// bit, le meme debut ne tient plus. MUTATION : `return pos == debut` -> `return pos >= debut` dans
// [chaineJusqua] : le cas decale passe, ROUGE.
func TestLaListeCommenceASonRecordNeufDeTete(t *testing.T) {
	var bw bitWriter
	bw.bits(0x1f, 5) // la fin d un message de la vue A : cinq bits qui ne sont pas un en-tete
	neuf := bw.n
	bw.neuf13(300, 2)
	bw.delta13(124)
	debut := bw.n
	bw.delta13(123)
	bw.bit(0) // terminateur de la vue B
	bw.bits(0, 2)
	w := mondeDeTete()

	cands := candidatsDeTete(bw.buf, debut, w)
	if len(cands) != 1 || cands[0] != neuf {
		t.Fatalf("candidats %v, attendu [%d] : seul l en-tete du slot 300 est dans sa bande", cands, neuf)
	}
	if got, ok := debutParChaine(bw.buf, debut, cands, w, cadreDeTete); !ok || got != neuf {
		t.Fatalf("debut %d (%v), attendu %d : la chaine NEW -> delta 124 finit sur le debut localise", got, ok, neuf)
	}
	if got, ok := debutParChaine(bw.buf, debut+1, cands, w, cadreDeTete); ok || got != debut+1 {
		t.Fatalf("debut decale : %d (%v), attendu %d garde — une chaine qui ne tombe pas au bit pres ne prouve rien", got, ok, debut+1)
	}
	if _, lie := w.ArchetypeForSlot(300); lie {
		t.Fatal("la chaine d ESSAI a lie le slot 300 : seule la marche lie")
	}
}

// TestUneChaineQuiRencontreUnSlotInconnuNeProuveRien : entre le NEW et le debut localise, un delta
// d un slot que le monde ne connait pas. La chaine s arrete : le debut du localisateur est garde.
func TestUneChaineQuiRencontreUnSlotInconnuNeProuveRien(t *testing.T) {
	var bw bitWriter
	bw.bits(0x1f, 5)
	bw.neuf13(300, 2)
	bw.delta13(555) // slot inconnu du monde
	debut := bw.n
	bw.delta13(123)
	bw.bit(0)
	bw.bits(0, 2)
	w := mondeDeTete()
	if got, ok := debutParChaine(bw.buf, debut, candidatsDeTete(bw.buf, debut, w), w, cadreDeTete); ok || got != debut {
		t.Fatalf("debut %d (%v), attendu %d garde", got, ok, debut)
	}
}
