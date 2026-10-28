//go:build research

package himap

// chemin_depot_research_test.go — la remontee vers le DEPOT HOTE, partagee par les instruments
// de recherche de la brique (D1 lot 2c, 2026-10-28).
//
// Ces instruments lisent des donnees que la brique ne possede pas : le dump de retro-ingenierie
// `.ai/re_dump` (non versionne) et les fichiers du depot hote (`data/...`). Ils se retrouvent
// par remontee depuis le repertoire de test ; le fichier absent fait sauter ou echouer
// l'instrument selon sa propre regle, jamais une absence silencieuse.

import (
	"fmt"
	"os"
	"path/filepath"
)

// cheminDepuisDepot remonte depuis le repertoire de test jusqu'a trouver un chemin relatif au
// depot.
func cheminDepuisDepot(rel string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for d := wd; ; {
		c := filepath.Join(d, rel)
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", fmt.Errorf("%s introuvable depuis %s", rel, wd)
		}
		d = parent
	}
}
