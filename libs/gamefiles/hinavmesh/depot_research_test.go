//go:build research

package hinavmesh

// depot_research_test.go — ou trouver le dump local des blobs navmesh, pour les instruments de
// recherche de la brique (D1 lot 2c, 2026-10-28).
//
// Les blobs d'autres cartes que les deux de `testdata/` vivent dans `.ai/re_dump/navmesh` : un
// dump de retro-ingenierie du depot hote, non versionne. Les instruments le lisent par une
// remontee depuis le repertoire de test ; absent, `racineDumpNavmesh` rend un chemin relatif
// qui n'existe pas et chaque instrument se declare absent (lecture impossible -> t.Skip).

import (
	"os"
	"path/filepath"
)

// dumpNavmeshDepot : le dump, relativement a la racine du depot hote.
const dumpNavmeshDepot = ".ai/re_dump/navmesh"

// racineDumpNavmesh rend le dossier du dump, ou son chemin relatif nu quand aucun parent du
// repertoire de test ne le porte.
func racineDumpNavmesh() string {
	wd, err := os.Getwd()
	if err != nil {
		return filepath.FromSlash(dumpNavmeshDepot)
	}
	for d := wd; ; {
		c := filepath.Join(d, filepath.FromSlash(dumpNavmeshDepot))
		if _, err := os.Stat(c); err == nil {
			return c
		}
		parent := filepath.Dir(d)
		if parent == d {
			return filepath.FromSlash(dumpNavmeshDepot)
		}
		d = parent
	}
}
