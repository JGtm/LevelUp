package testutil

// brique_film.go — LES FICHIERS VERSIONNES DU DECODEUR DE FILM, pour les tests de LevelUp qui les
// lisent (fixtures, goldens de forme, mini-bobines). Le decodeur est le module `libs/film` (ADR
// 0041) : hors de `apps/go-api`, il se situe depuis la racine du depot ([RepoRoot]), jamais par une
// echelle de « .. » ecrite a la main (internal/archlint, TestNoAdHocRepoRootLadderInTests).

import "path/filepath"

// briqueFilm : la racine du module du decodeur, relative a la racine du depot.
var briqueFilm = filepath.Join("libs", "film")

// FatalT : ce dont [CheminBriqueFilm] a besoin d'un test (`*testing.T`, `*testing.B`).
type FatalT interface {
	Fatalf(format string, args ...any)
	Helper()
}

// CheminBriqueFilm rend le chemin, sur le disque, d'un element du module du decodeur de film
// (`libs/film/<elems...>`). Une racine introuvable fait echouer le test : un fichier versionne
// absent est une installation cassee, pas un cas a sauter.
func CheminBriqueFilm(t FatalT, elems ...string) string {
	t.Helper()
	racine, err := RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	return filepath.Join(append([]string{racine, briqueFilm}, elems...)...)
}
