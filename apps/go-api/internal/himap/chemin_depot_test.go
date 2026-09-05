package himap

// chemin_depot_test.go — le seul helper de chemin partage par les DEUX populations de tests du
// paquet.
//
// POURQUOI IL VIT HORS DU TAG `gamefiles`. Il localise le DEPOT, pas l'installation du jeu :
// `cle_forge_test.go` s'en sert pour lire les fonds versionnes sous
// `data/titles/halo_infinite/reference/`, et ces tests-la tournent partout. Le laisser dans
// `carte_gate_gamefiles_test.go` (son emplacement d'origine) cassait la compilation du paquet
// des que le corpus gamefiles passait derriere son tag.

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
