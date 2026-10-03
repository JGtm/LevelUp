package grammar

// lecteur_minuteur_guard_test.go — LE GARDE-RAIL DU LECTEUR DE MINUTEUR (regle des deux copies,
// CLAUDE.md n. 6). La sequence de FUN_140d580d0 — deux lectures de n bits puis la queue R(5) — a
// vecu en ligne dans cinq composants ; elle n existe plus qu une fois, dans
// `lecteur_minuteur.go`. Ce test interdit qu elle revienne ailleurs dans la production du paquet
// sous trois formes : ecrite en ligne (deux `ReadBits` du meme argument puis un `ReadBits(5)`), en
// saut litteral `Skip(37)` ou `Skip(53)` (n = 16), ou en saut calcule qui nomme une largeur de
// minuteur sur la ligne du `Skip`. Un saut calcule a partir de litteraux seuls (`Skip(2*16 + 5)`)
// lui echappe.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// lectureMinuteurLitteral : deux `ReadBits` du meme argument sur deux lignes consecutives, puis
// un `ReadBits(5)` sur la ligne suivante. Les groupes capturent les deux arguments ; l egalite se
// verifie a part (RE2 n a pas de reference arriere).
var lectureMinuteurLitteral = regexp.MustCompile(
	`ReadBits\(([A-Za-z0-9_]+)\)[^\n]*\n[^\n]*ReadBits\(([A-Za-z0-9_]+)\)[^\n]*\n[^\n]*ReadBits\((5|largeurQueueMinuteur)\)`)

// sautMinuteurLitteral : le saut de la forme 2n + 5 a n = 16 (37 bits) et de la forme
// 3n + 5 a n = 16 (53 bits). Le saut de 15 bits n est pas interdit : `consumeDevicePosition` le
// fait pour R(14) + R(1), une autre forme.
var sautMinuteurLitteral = regexp.MustCompile(`Skip\((37|53)\)`)

// sautMinuteurCalcule : un `Skip` dont la ligne nomme une largeur de minuteur — la queue de
// FUN_1407f0354 ou un n que les appelants passent a FUN_140d580d0. Aucun saut de production ne
// les nomme : ces largeurs ne servent qu au lecteur unique et a ses appelants.
var sautMinuteurCalcule = regexp.MustCompile(
	`Skip\([^\n]*\b(largeurQueueMinuteur|roundTimerBits|largeurMinuteurSoftKill)\b`)

// TestLecteurDeMinuteurUnique interdit les copies hors du fichier hote.
//
// EXCLUSIONS : les `_test.go` (vecteurs et instruments de mesure, hors production). Le fichier hote
// doit porter la sequence exactement une fois, sans quoi le garde-rail garde un fantome.
func TestLecteurDeMinuteurUnique(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob : %v", err)
	}
	if len(files) == 0 {
		t.Fatal("aucun fichier Go vu : le garde-rail ne garde rien")
	}
	hote := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		data, err := os.ReadFile(f) //nolint:gosec // fichiers du paquet lui-meme
		if err != nil {
			t.Fatalf("lecture de %s : %v", f, err)
		}
		copies := 0
		for _, m := range lectureMinuteurLitteral.FindAllSubmatch(data, -1) {
			if string(m[1]) == string(m[2]) {
				copies++
			}
		}
		sauts := len(sautMinuteurLitteral.FindAll(data, -1)) + len(sautMinuteurCalcule.FindAll(data, -1))
		if f == "lecteur_minuteur.go" {
			hote = copies
			continue
		}
		if copies > 0 || sauts > 0 {
			t.Errorf("%s : %d lecture(s) en ligne et %d saut(s) de la forme de FUN_140d580d0 — "+
				"appeler lireMinuteur140d580d0 / lireMinuteur142ba78dc (lecteur_minuteur.go)",
				f, copies, sauts)
		}
	}
	if hote != 1 {
		t.Errorf("lecteur_minuteur.go porte %d fois la sequence de FUN_140d580d0, 1 attendue — "+
			"deplacer le garde-rail avec le lecteur", hote)
	}
}
