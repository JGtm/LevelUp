package replay

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestUneSeuleEcritureDeLEmpriseJouee — LE GARDE-RAIL DE LA CENTRALISATION (lot M1 des retours du
// rejeu, 2026-09-23 ; regle n° 6 du depot). La garde « centiles plus douze etendues centrales » a
// trois usages depuis ce lot (bornes, positions de bipede, positions de vehicule) : elle ne se
// construit QUE par `empriseDesAxes`, et ni ses centiles ni sa constante ne se recopient.
//
//	`guardOf(`                  appele dans emprise_jouee.go seul (defini dans geometry.go)
//	`boundsRejectSpreads *`     la multiplication de la marge, dans geometry.go seul
//	`[99*n/100]`                le centile haut, dans geometry.go seul
func TestUneSeuleEcritureDeLEmpriseJouee(t *testing.T) {
	regles := []struct {
		motif    *regexp.Regexp
		autorise map[string]bool
		quoi     string
	}{
		{regexp.MustCompile(`guardOf\(`), map[string]bool{"emprise_jouee.go": true, "geometry.go": true},
			"construction d une garde d axe"},
		{regexp.MustCompile(`boundsRejectSpreads\s*\*`), map[string]bool{"geometry.go": true},
			"marge de l emprise"},
		{regexp.MustCompile(`\[\s*99\s*\*\s*n\s*/\s*100\s*\]`), map[string]bool{"geometry.go": true},
			"centile haut de l emprise"},
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	vus := make([]bool, len(regles))
	for _, e := range entries {
		file := e.Name()
		if e.IsDir() || !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			t.Fatalf("lecture %s : %v", file, err)
		}
		for i, r := range regles {
			if !r.motif.Match(raw) {
				continue
			}
			if r.autorise[file] {
				vus[i] = true
				continue
			}
			t.Errorf("%s : %s ECRITE hors de l emprise centralisee (%v). Passer par "+
				"`empriseDesAxes` / `empriseJouee.rejette` (emprise_jouee.go)", file, r.quoi, r.autorise)
		}
	}
	for i, r := range regles {
		if !vus[i] {
			t.Errorf("le motif %q a disparu de %v : le garde-rail ne verifie plus rien (renomme ?)",
				r.motif, r.autorise)
		}
	}
}

// TestEmpriseDesarmeeNeRejetteRien : sous `boundsMinSamples`, la garde n ecarte rien — et c est la
// meme reponse pour les trois usages.
func TestEmpriseDesarmeeNeRejetteRien(t *testing.T) {
	e := empriseDesAxes([]float32{0, 1}, []float32{0, 1}, []float32{0, 1})
	if e.armee || e.rejette(1e6, 1e6, 1e6) {
		t.Errorf("emprise sur deux points : armee=%t, attendu desarmee et muette", e.armee)
	}
}
