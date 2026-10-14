// Package archlint — no_second_riposte_window_test.go : ratchet « UNE SEULE FENETRE DE RIPOSTE ».
//
// La fenetre de riposte (3 000 ms, bornes incluses) est `coordination.FenetreRiposteMs`, source
// unique ; le web recoit le delai calcule, jamais la fenetre. Une factorisation sans garde-rail
// re-diverge (CLAUDE.md regle 6) : ce test interdit tout litteral 3000 / 3_000 sur une ligne qui
// parle de riposte, de vengeur, de tueur abattu ou d'abattu en retour, hors du fichier qui declare
// la constante.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestUneSeuleFenetreDeRiposte(t *testing.T) {
	apiRoot := archlintAPIRoot(t)
	motifLitteral := regexp.MustCompile(`\b3_?000\b`)
	motifSujet := regexp.MustCompile(`(?i)riposte|vengeur|tueur.?abattu|abattu.?en.?retour`)
	source := filepath.ToSlash(filepath.Join("internal", "analysis", "coordination", "riposte.go"))

	lus := 0
	err := filepath.WalkDir(filepath.Join(apiRoot, "internal"), func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel := filepath.ToSlash(mustRel(apiRoot, path))
		if rel == source {
			return nil
		}
		octets, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		lus++
		for i, ligne := range strings.Split(string(octets), "\n") {
			code := ligne
			if k := strings.Index(code, "//"); k >= 0 {
				code = code[:k]
			}
			if motifLitteral.MatchString(code) && motifSujet.MatchString(ligne) {
				t.Errorf("%s:%d : un litteral 3000 sur une ligne de riposte — la fenetre est "+
					"coordination.FenetreRiposteMs, source unique", rel, i+1)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if lus < 100 {
		t.Fatalf("%d fichiers lus seulement : le ratchet ne mord plus sur rien", lus)
	}
	if _, err := os.Stat(filepath.Join(apiRoot, filepath.FromSlash(source))); err != nil {
		t.Fatalf("la source de la fenetre a bouge : %v", err)
	}
}
