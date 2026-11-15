package testutil

// brique_film.go — LES FICHIERS VERSIONNES DU DECODEUR DE FILM, pour les tests de LevelUp qui les
// lisent (fixtures, goldens de forme, mini-bobines). Le decodeur est le module `libs/film` (ADR
// 0041) : hors de `apps/go-api`, son dossier est celui que le resolveur des briques
// ([briques.Dir]) tire de l'outil Go, jamais une echelle de « .. » ni un chemin depuis la racine du
// depot (internal/archlint, TestNoAdHocRepoRootLadderInTests et TestCheminsDeBriqueParLeResolveur).

import "levelup/go-api/internal/briques"

// FatalT : ce dont [CheminBriqueFilm] a besoin d'un test (`*testing.T`, `*testing.B`).
type FatalT interface {
	Fatalf(format string, args ...any)
	Helper()
}

// CheminBriqueFilm rend le chemin, sur le disque, d'un element du module du decodeur de film. Une
// brique introuvable fait echouer le test : un fichier versionne absent est une installation
// cassee, pas un cas a sauter.
func CheminBriqueFilm(t FatalT, elems ...string) string {
	t.Helper()
	chemin, err := briques.Chemin("film", elems...)
	if err != nil {
		t.Fatalf("brique du decodeur de film : %v", err)
	}
	return chemin
}
