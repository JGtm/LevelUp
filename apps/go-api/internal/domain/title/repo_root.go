package title

import (
	"fmt"
	"os"
	"path/filepath"
)

// repoRootMarker est le fichier qui marque la racine du dépôt.
const repoRootMarker = "db_profiles.json"

// FindRepoRoot renvoie la racine du dépôt : LEVELUP_REPO_ROOT si la variable est définie
// (même variable que le serveur, garantit une résolution de chemins identique), sinon en
// remontant depuis le répertoire courant jusqu'au marqueur db_profiles.json.
//
// Destiné aux OUTILS HORS LIGNE (cmd/*-build) qui n'ouvrent pas de DB : il évite
// d'importer internal/config, dont la chaîne DuckDB impose CGO.
func FindRepoRoot() (string, error) {
	if r := os.Getenv("LEVELUP_REPO_ROOT"); r != "" {
		return r, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, repoRootMarker)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("LEVELUP_REPO_ROOT non défini et %s introuvable en remontant depuis le cwd",
				repoRootMarker)
		}
		dir = parent
	}
}
