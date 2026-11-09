//go:build research

package mapvar

// depot_research_test.go — ou trouver le dump local des variantes `.mvar`, pour les instruments
// de recherche de la brique (D1 lot 2c, 2026-10-28).
//
// Les variantes d'autres cartes que les trois de `testdata/` vivent dans `.ai/re_dump/mapvar` :
// un dump de retro-ingenierie du depot hote, non versionne. Les instruments le lisent par une
// remontee depuis le repertoire de test ; absent, `cheminDump` rend un chemin relatif qui
// n'existe pas et chaque instrument se declare absent (lecture impossible -> t.Skip).

import (
	"os"
	"path/filepath"
)

// dumpMapvarDepot : le dump, relativement a la racine du depot hote.
const dumpMapvarDepot = ".ai/re_dump/mapvar"

// cheminDump rend le chemin de la variante `nom` dans le dump, ou son chemin relatif nu quand
// aucun parent du repertoire de test ne porte le dump.
func cheminDump(nom string) string {
	rel := filepath.FromSlash(dumpMapvarDepot)
	wd, err := os.Getwd()
	if err != nil {
		return filepath.Join(rel, nom)
	}
	for d := wd; ; {
		c := filepath.Join(d, rel)
		if _, err := os.Stat(c); err == nil {
			return filepath.Join(c, nom)
		}
		parent := filepath.Dir(d)
		if parent == d {
			return filepath.Join(rel, nom)
		}
		d = parent
	}
}
