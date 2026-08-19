package testutil

// repo_root.go — LA RACINE DU DEPOT POUR LES TESTS, sans variable d'environnement et sans
// marqueur gitignore.
//
// Pourquoi pas title.FindRepoRoot() : ce helper de PRODUCTION cherche `db_profiles.json`
// (gitignore, absent d'un checkout neuf) ou LEVELUP_REPO_ROOT (jamais pose en CI). Un test
// qui l'appelle pour lire un fichier VERSIONNE se skippe donc en silence sur une machine
// d'integration — c'est ainsi que les gardes du lot C-ter volet 2 ne tournaient jamais en
// CI (revue adversariale ronde 1, R1-1, 2026-08-19).
//
// Ici la racine se deduit de l'EMPLACEMENT DE CE FICHIER SOURCE : il est dans l'arbre
// versionne par construction, donc la racine est trouvee quel que soit le repertoire
// courant, sans configuration ni fichier ignore. C'est le mecanisme CANONIQUE : les
// echelles de « ../../../.. » ecrites a la main dans les tests sont interdites par
// internal/archlint (TestNoAdHocRepoRootLadderInTests).

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// repoRootDepth : nombre de repertoires entre le repertoire de CE fichier et la racine du
// depot — internal/testutil -> internal -> apps/go-api -> apps -> racine.
const repoRootDepth = 4

// repoRootMarker : un repertoire VERSIONNE de la racine. Il ne sert pas a chercher (la
// racine est deduite, pas devinee) mais a echouer clairement si ce fichier change de place.
var repoRootMarker = filepath.Join("config", "titles")

// RepoRoot renvoie la racine du depot pour un test qui lit un fichier versionne
// (`data/titles/...`, `config/titles/...`). L'erreur est destinee a un t.Fatalf : un
// fichier versionne absent est une installation cassee, pas un cas nominal a skipper.
func RepoRoot() (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("testutil: runtime.Caller a echoue, racine du depot indeductible")
	}
	root := filepath.Dir(thisFile)
	for i := 0; i < repoRootDepth; i++ {
		root = filepath.Dir(root)
	}
	if _, err := os.Stat(filepath.Join(root, repoRootMarker)); err != nil {
		return "", fmt.Errorf("testutil: %q ne porte pas %s — ce fichier a-t-il change de place ? : %w",
			root, repoRootMarker, err)
	}
	return root, nil
}
