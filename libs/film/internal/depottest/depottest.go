// Package depottest : OU TROUVER LE DEPOT HOTE pour les tests de la brique qui en lisent les
// donnees.
//
// Certains tests de la brique lisent des donnees versionnees du depot hote (catalogues de
// reference sous `data/titles/`, sources du front sous `apps/`) ou son parc local (`data/cache/`).
// La brique n'importe rien du depot : [Racine] remonte donc depuis l'EMPLACEMENT DE CE FICHIER
// SOURCE jusqu'au premier dossier qui porte `go.work` (le fichier d'espace de travail du depot,
// versionne). Hors depot hote (la brique prise seule), aucun `go.work` n'est trouve et le test se
// declare absent (t.Skip) : il n'a pas de donnee a lire.
//
// C'est la seule remontee vers la racine de la brique (tenue par un garde-rail de l'hote).
package depottest

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Racine rend la racine du depot hote (le dossier qui porte `go.work`), ou saute le test.
func Racine(t testing.TB) string {
	t.Helper()
	_, ce, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("racine du depot hote indeductible (runtime.Caller a echoue)")
	}
	for dir := filepath.Dir(ce); ; {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skipf("aucun go.work au-dessus de %s : brique hors du depot hote", filepath.Dir(ce))
		}
		dir = parent
	}
}
