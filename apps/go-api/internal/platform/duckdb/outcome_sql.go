package duckdb

import (
	"context"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// outcomeSQLEq résout l'expression SQL « col vaut le code de l'issue o » pour le
// titre courant (lu dans le ctx) via le resolver d'issues partagé (PMT-5 / MT-06).
// Fallback sur `legacy` (littéral Halo, ex. "outcome = 2") si aucun resolver n'est
// câblé OU si le titre ne déclare pas de raw_code pour cette issue → byte-identique
// pour halo_infinite (le resolver Halo rend exactement le littéral legacy). Évite
// tout `outcome = 2` codé en dur dans les agrégats wins/losses/draws des repos.
//
// `col` et `legacy` doivent être des littéraux de confiance (jamais d'entrée
// utilisateur) ; le resolver n'interpole qu'un entier (raw_code) dans l'expression.
func outcomeSQLEq(ctx context.Context, col string, o canonical.Outcome, legacy string) string {
	return outcomeSQLEqSlug(ctxkeys.TitleSlug(ctx), col, o, legacy)
}

// OutcomeSQLEqSlug est la variante EXPORTÉE (slug explicite) du seam d'issues,
// pour les consommateurs HORS package duckdb qui construisent du SQL agrégé et
// doivent éviter le littéral `outcome = 2` codé en dur (ex. l'orchestrateur
// post-sync `internal/api`, K1a). Même contrat byte-identique + fallback legacy.
func OutcomeSQLEqSlug(slug, col string, o canonical.Outcome, legacy string) string {
	return outcomeSQLEqSlug(slug, col, o, legacy)
}

// outcomeSQLEqSlug est la variante à slug EXPLICITE, pour les repos qui reçoivent
// déjà un titleSlug en paramètre (ex. CompareRepo.GetLocalStats) plutôt que via le
// ctx. Même contrat byte-identique + fallback legacy que outcomeSQLEq.
func outcomeSQLEqSlug(slug, col string, o canonical.Outcome, legacy string) string {
	res := games.DefaultOutcomeResolver()
	if res == nil {
		return legacy
	}
	if expr, ok := res.SQLEq(slug, col, o); ok {
		return expr
	}
	return legacy
}
