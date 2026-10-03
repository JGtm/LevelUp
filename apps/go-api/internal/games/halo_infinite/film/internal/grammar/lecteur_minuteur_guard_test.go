package grammar

// lecteur_minuteur_guard_test.go — LE GARDE-RAIL DU LECTEUR DE MINUTEUR (regle des deux copies,
// CLAUDE.md n. 6). La sequence de FUN_140d580d0 — deux lectures de n bits puis la queue R(5) — n
// existe qu une fois, dans `lecteur_minuteur.go`. Ce test interdit qu elle revienne ailleurs dans
// la production du paquet sous trois formes : ecrite en ligne (trois appels consecutifs d un meme
// bloc, `ReadBits(x)`, `ReadBits(x)`, `ReadBits(5)`, quelle que soit leur mise en page : la suite est
// lue dans l arbre syntaxique, cf. `sequence_appels_test.go`), en saut litteral `Skip(37)` ou
// `Skip(53)` (n = 16), ou en saut calcule dont l argument nomme une largeur de minuteur. Un saut
// calcule a partir de litteraux seuls (`Skip(2*16 + 5)`) lui echappe.

import (
	"regexp"
	"testing"
)

// largeurNommeeDeMinuteur : un argument de `Skip` qui nomme une largeur de minuteur — la queue de
// FUN_1407f0354 ou un n que les appelants passent a FUN_140d580d0. Aucun saut de production ne
// les nomme : ces largeurs ne servent qu au lecteur unique et a ses appelants.
var largeurNommeeDeMinuteur = regexp.MustCompile(`\b(largeurQueueMinuteur|roundTimerBits|largeurMinuteurSoftKill)\b`)

// formesDeMinuteur compte, dans la suite d appels `seq` d une fonction, les lectures en ligne de
// FUN_140d580d0 et les sauts de sa forme. Le saut de 15 bits n est pas interdit :
// `consumeDevicePosition` le fait pour R(14) + R(1), une autre forme.
func formesDeMinuteur(seq []appelLu) (copies, sauts int) {
	for i, a := range seq {
		if i+2 < len(seq) && estAppel(a, "ReadBits") && estAppel(seq[i+1], "ReadBits", a.args) &&
			estAppel(seq[i+2], "ReadBits", "5", "largeurQueueMinuteur") && memeBloc(seq[i:i+3]...) {
			copies++
		}
		if estAppel(a, "Skip", "37", "53") || (a.nom == "Skip" && largeurNommeeDeMinuteur.MatchString(a.args)) {
			sauts++
		}
	}
	return copies, sauts
}

// TestLecteurDeMinuteurUnique interdit les copies hors du fichier hote.
//
// EXCLUSIONS : les `_test.go` (vecteurs et instruments de mesure, hors production). Le fichier hote
// doit porter la sequence exactement une fois, sans quoi le garde-rail garde un fantome.
func TestLecteurDeMinuteurUnique(t *testing.T) {
	hote := 0
	for _, f := range appelsDeLaProduction(t) {
		copies, sauts := 0, 0
		for _, seq := range f.fonctions {
			c, s := formesDeMinuteur(seq)
			copies, sauts = copies+c, sauts+s
		}
		if f.nom == "lecteur_minuteur.go" {
			hote = copies
			continue
		}
		if copies > 0 || sauts > 0 {
			t.Errorf("%s : %d lecture(s) en ligne et %d saut(s) de la forme de FUN_140d580d0 — "+
				"appeler lireMinuteur140d580d0 / lireMinuteur142ba78dc (lecteur_minuteur.go)",
				f.nom, copies, sauts)
		}
	}
	if hote != 1 {
		t.Errorf("lecteur_minuteur.go porte %d fois la sequence de FUN_140d580d0, 1 attendue — "+
			"deplacer le garde-rail avec le lecteur", hote)
	}
}

// TestGardeRailMinuteurInsensibleALaMiseEnPage : la copie se voit sur une ligne comme sur trois,
// commentaires intercales compris ; deux largeurs differentes, ou des lectures sous porte suivies
// d une lecture inconditionnelle (`consumeDeadStateAnimBlock`, etapes 14 et 15), ne sont pas la sequence.
func TestGardeRailMinuteurInsensibleALaMiseEnPage(t *testing.T) {
	for _, c := range []struct {
		corps         string
		copies, sauts int
	}{
		{"return Minuteur{A: br.ReadBits(n), B: br.ReadBits(n), Queue: br.ReadBits(5)}", 1, 0},
		{"a := br.ReadBits(16) // a\n\t// b\n\tb := br.ReadBits( 16 )\n\tq := br.ReadBits(largeurQueueMinuteur)\n\t_, _, _ = a, b, q", 1, 0},
		{"br.ReadBits(16); br.ReadBits(8); br.ReadBits(5)", 0, 0},
		{"if v { br.ReadBits(14); br.ReadBits(14) }; br.ReadBits(5)", 0, 0},
		{"br.Skip(37); br.Skip(2*roundTimerBits + largeurQueueMinuteur); br.Skip(15)", 0, 2},
	} {
		src := "package p\n\nfunc f(br *Lecteur, n uint, v bool) Minuteur {\n\t" + c.corps + "\n\treturn Minuteur{}\n}\n"
		copies, sauts := 0, 0
		for _, seq := range appelsDuSource(t, "vecteur.go", []byte(src)) {
			cp, s := formesDeMinuteur(seq)
			copies, sauts = copies+cp, sauts+s
		}
		if copies != c.copies || sauts != c.sauts {
			t.Errorf("%q : %d copie(s), %d saut(s), attendu %d et %d", c.corps, copies, sauts, c.copies, c.sauts)
		}
	}
}
