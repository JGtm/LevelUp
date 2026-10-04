package grammar

import "testing"

// debut_de_liste_masque_test.go — la preuve par chaine ([debutParChaine], [pasDEssai]) refuse un
// record dont le masque contredit `FUN_142e2da44`, l ecrivain du masque, quelle que soit la regle
// contredite : un bit au-dela du dernier composant de l archetype, un masque dense qui annonce sept
// composants au plus, un masque epars a index non croissants ([EntityTrace.MasqueNonEcrit]).

// neuf13Avec ecrit l en-tete d un record NEW comme [bitWriter.neuf13], puis le masque que `masque`
// ecrit ; les bits des composants presents sont a la charge de l appelant.
func (w *bitWriter) neuf13Avec(slot uint32, ti uint64, masque func(w *bitWriter)) {
	w.bit(0)
	w.bits(recNew, 2)
	w.bits(uint64(slot), 13)
	w.bits(1, 2)
	w.bits(ti, 6)
	w.bit(0) // porte du record NEW
	masque(w)
}

// delta13Avec ecrit l en-tete d un record DELTA comme [bitWriter.delta13], puis le masque que
// `masque` ecrit.
func (w *bitWriter) delta13Avec(slot uint32, masque func(w *bitWriter)) {
	w.bit(1)
	w.bits(uint64(slot), 13)
	w.bits(1, 2)
	w.bit(0)
	masque(w)
}

// epars : selecteur `0`, compte R(3), puis les index R(6) dans l ordre donne.
func epars(idx ...uint64) func(w *bitWriter) {
	return func(w *bitWriter) {
		w.bit(0)
		w.bits(uint64(len(idx)), 3)
		for _, i := range idx {
			w.bits(i, 6)
		}
	}
}

// dense : selecteur `1`, puis le masque R(64).
func dense(m uint64) func(w *bitWriter) {
	return func(w *bitWriter) {
		w.bit(1)
		w.bits(m, 64)
	}
}

// chaineDeTete construit un paquet : cinq bits de fin de message de la vue A, puis `tete` (qui
// ecrit les records de tete), puis le record du slot 123 ou le localisateur se pose, puis le
// terminateur. Rend le tampon, la position du premier record de tete et celle du debut localise.
func chaineDeTete(tete func(w *bitWriter)) ([]byte, int, int) {
	var bw bitWriter
	bw.bits(0x1f, 5)
	neuf := bw.n
	tete(&bw)
	debut := bw.n
	bw.delta13(123)
	bw.bit(0)
	bw.bits(0, 2)
	return bw.buf, neuf, debut
}

// composantsDUnBit : huit composants lus sur un bit chacun (`FUN_142f04850`, un drapeau), de quoi
// poser un masque dense que l ecrivain ecrit (huit composants) sans sortir de l archetype.
var composantsDUnBit = []string{
	"projectile-tether-state", "projectile-tether-state", "projectile-tether-state",
	"projectile-tether-state", "projectile-tether-state", "projectile-tether-state",
	"projectile-tether-state", "projectile-tether-state",
}

// mondeDeTeteAComposants : [mondeDeTete], dont les archetypes 2 (le NEW de tete) et 4 (les deltas)
// portent [composantsDUnBit].
func mondeDeTeteAComposants() *World {
	w := NewWorld(&Registry{Archetypes: []Archetype{{Index: 0}, {Index: 1},
		{Index: 2, Components: composantsDUnBit}, {Index: 3}, {Index: 4, Components: composantsDUnBit}}})
	w.BindImageCle(1, 123, 4)
	w.BindImageCle(1, 124, 4)
	t := NouvelleTableAnticipee()
	t.archetypesDuSlot[300] = map[uint32]bool{2: true}
	w.PoserTableAnticipee(t)
	return w
}

// valeursDeComposants ecrit `n` composants d un bit.
func valeursDeComposants(w *bitWriter, n int) {
	for range n {
		w.bit(1)
	}
}

// casDeMasque : une chaine de tete, le monde ou elle se lit, et si l ecrivain peut l avoir ecrite.
type casDeMasque struct {
	nom    string
	monde  func() *World
	tete   func(w *bitWriter)
	ecrite bool
}

// casDeMasques : pour chaque regle, sur le pas NEW et sur le pas delta, un masque qui la contredit
// et son TEMOIN — le meme paquet, aux memes composants lus, avec un masque que l ecrivain ecrit. Le
// temoin prouve que la chaine tombe au bit pres : seul le masque refuse le cas contredit.
var casDeMasques = []casDeMasque{
	{"NEW, bit au-dela de l archetype", mondeDeTete, func(w *bitWriter) {
		w.neuf13Avec(300, 2, epars(5))
		w.delta13(124)
	}, false},
	{"NEW, aucun bit (temoin du bit au-dela)", mondeDeTete, func(w *bitWriter) {
		w.neuf13Avec(300, 2, epars())
		w.delta13(124)
	}, true},
	{"delta, bit au-dela de l archetype", mondeDeTete, func(w *bitWriter) {
		w.neuf13(300, 2)
		w.delta13Avec(124, epars(3))
	}, false},
	{"NEW, dense de sept composants", mondeDeTeteAComposants, func(w *bitWriter) {
		w.neuf13Avec(300, 2, dense(0x7f))
		valeursDeComposants(w, 7)
		w.delta13(124)
	}, false},
	{"NEW, dense de huit composants (temoin)", mondeDeTeteAComposants, func(w *bitWriter) {
		w.neuf13Avec(300, 2, dense(0xff))
		valeursDeComposants(w, 8)
		w.delta13(124)
	}, true},
	{"NEW, epars non croissant", mondeDeTeteAComposants, func(w *bitWriter) {
		w.neuf13Avec(300, 2, epars(3, 1))
		valeursDeComposants(w, 2)
		w.delta13(124)
	}, false},
	{"NEW, epars croissant (temoin)", mondeDeTeteAComposants, func(w *bitWriter) {
		w.neuf13Avec(300, 2, epars(1, 3))
		valeursDeComposants(w, 2)
		w.delta13(124)
	}, true},
	{"delta, dense de sept composants", mondeDeTeteAComposants, func(w *bitWriter) {
		w.neuf13(300, 2)
		w.delta13Avec(124, dense(0xfe))
		valeursDeComposants(w, 7)
	}, false},
	{"delta, dense de huit composants (temoin)", mondeDeTeteAComposants, func(w *bitWriter) {
		w.neuf13(300, 2)
		w.delta13Avec(124, dense(0xff))
		valeursDeComposants(w, 8)
	}, true},
	{"delta, epars non croissant", mondeDeTeteAComposants, func(w *bitWriter) {
		w.neuf13(300, 2)
		w.delta13Avec(124, epars(6, 6))
		valeursDeComposants(w, 1)
	}, false},
	{"delta, epars croissant (temoin)", mondeDeTeteAComposants, func(w *bitWriter) {
		w.neuf13(300, 2)
		w.delta13Avec(124, epars(6))
		valeursDeComposants(w, 1)
	}, true},
}

// TestUneChaineQuiTraverseUnMasqueNonEcritNeProuveRien : la chaine NEW (slot 300) -> delta (slot
// 124) tombe au bit pres sur le debut localise ; quand un de ses masques ne peut pas avoir ete
// ecrit par le jeu, le debut du localisateur est garde, sinon la liste commence au NEW.
// MUTATIONS : retirer le test du masque dans [pasDEssai] (pas NEW ou pas delta), ou ne refuser que
// le bit hors archetype — ROUGE.
func TestUneChaineQuiTraverseUnMasqueNonEcritNeProuveRien(t *testing.T) {
	for _, c := range casDeMasques {
		buf, neuf, debut := chaineDeTete(c.tete)
		w := c.monde()
		got, ok := debutParChaine(buf, debut, candidatsDeTete(buf, debut, w), w, cadreDeTete)
		switch {
		case c.ecrite && (!ok || got != neuf):
			t.Errorf("%s : debut %d (%v), attendu %d — l ecrivain ecrit ce masque", c.nom, got, ok, neuf)
		case !c.ecrite && (ok || got != debut):
			t.Errorf("%s : debut %d (%v), attendu %d garde — un masque que l ecrivain n ecrit pas ne prouve rien",
				c.nom, got, ok, debut)
		}
	}
}
