package migration

// title_set.go — routage des migrations PAR TITRE (PMT-9, MT-23).
//
// Un titre NON-DÉFAUT (ex. un 2e jeu) enregistre son PROPRE jeu de migrations
// (steps + ordre canonique) via RegisterMigrationSet ; le runner route par slug
// avec un simple lookup de map — JAMAIS par comparaison de slug (cf.
// archlint/no_slug_comparison, ADR 0025 §7). Le routage est data-driven : la clé
// est une donnée (le slug), pas un littéral codé dans une branche.
//
// Le titre PAR DÉFAUT (Halo Infinite) n'a PAS besoin d'être enregistré : quand
// aucun set n'est trouvé pour le slug, RunForTitleDB retombe sur le chemin legacy
// (registre global + titleStepsProvider, ordonné par canonicalOrder d'order.go),
// strictement byte-identique au comportement historique de RunForDB. C'est
// volontaire : la liste canonicalOrder est l'ordre UNIFIÉ couvrant les steps Halo
// encore globaux (rebuild_match_participants + seeds dynamiques) ET les steps
// title-owned — la garder côté runner pour le défaut est plus robuste que d'exiger
// que chaque call-site enregistre un set.

// DefaultSlug est le slug du titre par défaut (Halo Infinite). Dupliqué depuis
// title.DefaultSlug pour garder le package migration sans dépendance sur le
// package domain/title ; l'égalité est verrouillée par default_slug_test.go.
const DefaultSlug = "halo_infinite"

// TitleMigrationSet décrit le jeu de migrations d'un titre : son ordre canonique
// (noms, imposé au tri d'exécution) et ses steps par target. Couche domain-pure
// (zéro DuckDB). Pour un titre non-défaut, Steps ne doit retourner QUE les steps
// du titre (pas le registre global Halo) — c'est ce qui garantit l'isolation.
type TitleMigrationSet struct {
	Slug           string
	CanonicalOrder []string
	Steps          func(TargetDB) []Migration
}

// migrationSets : jeux enregistrés par slug (titres non-défaut). Le défaut Halo
// est ABSENT volontairement (il passe par le chemin legacy de RunForTitleDB).
var migrationSets = map[string]TitleMigrationSet{}

// RegisterMigrationSet enregistre le jeu de migrations d'un titre non-défaut.
// À appeler au boot (ou en test) avant RunForTitleDB. Idempotent (dernier gagne).
// No-op si Slug vide.
func RegisterMigrationSet(s TitleMigrationSet) {
	if s.Slug == "" {
		return
	}
	migrationSets[s.Slug] = s
}

// migrationSetFor retourne le set enregistré pour slug, le cas échéant.
func migrationSetFor(slug string) (TitleMigrationSet, bool) {
	s, ok := migrationSets[slug]
	return s, ok
}
