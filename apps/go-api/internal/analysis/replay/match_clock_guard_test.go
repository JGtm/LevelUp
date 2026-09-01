package replay

// match_clock_guard_test.go — GARDE-RAIL (CLAUDE.md n° 6) : la conversion match <-> frames
// n'a qu'UNE implementation, celle de match_clock.go.
//
// HISTOIRE DU DEFAUT : la methode `frameOfMatchMS` a ete recopiee trois fois (drapeau, crane,
// couronne) avant d'etre centralisee a l'arrivee du quatrieme consommateur (portage de la
// bombe, schema 30). Une factorisation sans garde-rail re-diverge — lecon du predicat bot
// (8 -> 36 copies apres centralisation). Ce test grep rend la re-divergence ROUGE : toute
// nouvelle declaration d'une methode de conversion hors de match_clock.go echoue ici.

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestMatchClockSingleImplementation echoue si `frameOfMatchMS` ou `matchMSOfFrame` est
// declare ailleurs que dans match_clock.go.
func TestMatchClockSingleImplementation(t *testing.T) {
	decl := regexp.MustCompile(`func \([^)]+\) (frameOfMatchMS|matchMSOfFrame)\(`)
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" || e.Name() == "match_clock.go" {
			continue
		}
		src, err := os.ReadFile(e.Name()) //nolint:gosec // fichiers du paquet lui-meme
		if err != nil {
			t.Fatalf("lecture de %s : %v", e.Name(), err)
		}
		if m := decl.Find(src); m != nil {
			t.Errorf("%s declare %q : la conversion match <-> frames n'a qu'une implementation "+
				"(match_clock.go) — utiliser ou embarquer matchClock au lieu de recopier", e.Name(), m)
		}
	}
}
