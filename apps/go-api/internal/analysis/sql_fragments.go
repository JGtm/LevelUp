// Package analysis — sql_fragments.go : fragments SQL canoniques.
//
// Centralise les expressions SQL répétées dans les repositories DuckDB pour
// éviter les divergences (revue 2026-04-29 axe 6 DETTE 11 — `IsBot` SQL
// répété 8 fois, bugs causés par ces duplications).
//
// Convention :
//   - Préfixer les noms par SQL (ex: SQLIsBot)
//   - Ne pas paramétrer les noms de tables/alias — laisser le repository
//     les composer. Ces fragments sont des prédicats / expressions, pas des
//     queries complètes. Exception : SQLStartTimeCanonical(alias) est une
//     fonction car l'expression référence la colonne aliasée deux fois — un
//     const ne peut pas porter l'alias sur les deux occurrences.
//   - Toujours utiliser ces fragments via concaténation explicite, pas via
//     fmt.Sprintf (lisibilité + audit grep).
package analysis

// SQLIsBot est le prédicat SQL pour identifier un xuid de bot.
// Aligné sur analysis.IsBot côté Go (préfixe bid(*).
//
// Usage typique :
//
//	WHERE xuid NOT LIKE 'bid(%'        // filtrer les bots
//	WHERE xuid NOT LIKE ` + analysis.SQLIsBot + `   // version centralisée
//
// Pour préserver la lisibilité dans les repos, préférer concaténer en
// fin de WHERE plutôt qu'au milieu d'une CTE complexe.
const SQLIsBot = `xuid LIKE 'bid(%'`

// SQLIsNotBot est le prédicat opposé — utile pour filtrer les bots sans
// double négation côté repository.
const SQLIsNotBot = `xuid NOT LIKE 'bid(%'`

// Note : les prédicats d'issue (win/loss/tie) NE sont PAS des fragments const
// (ils dépendent du titre — MT-06 / PMT-5). Construire l'expression via le
// resolver d'issues : `duckdb.outcomeSQLEq(ctx, col, canonical.OutcomeWin, "outcome = 2")`.
// Les ex-const `SQLIsWin` / `SQLWinRateExpr` (codées en dur `outcome = 2`) ont été
// retirées (0 consommateur) au profit de ce seam title-aware.

// SQLKDRExpr est l'expression SQL canonique pour calculer un K/D ratio
// agrégé (sum(kills)/max(1,sum(deaths))). Aligné sur analysis.KDR.
//
// Note : c'est un KDR sur totaux, distinct de avg(KDR per match). Pour le
// "K/D moyen affiché" produit, préférer cette agrégation totaliste qui est
// stable face aux matchs aux scores extrêmes.
const SQLKDRExpr = `CAST(SUM(kills) AS DOUBLE) / NULLIF(SUM(deaths), 0)`

// SQLStartTimeCanonical est l'expression SQL canonique du timestamp de début
// de match, en UTC. Elle applique la règle CLAUDE.md n°8 : ne JAMAIS filtrer
// ni trier sur `start_time` brut — toujours COALESCE avec `start_time_utc`
// puis interpréter `start_time` en UTC. Toute divergence de cette expression
// a causé des décalages de fuseau (DETTE first_joined_time).
//
// alias est le préfixe de table (ex "mr", "r") ; "" pour une colonne non
// qualifiée. Le garde-rail analysis/start_time_canonical_test.go interdit le
// littéral brut hors de ce helper (et de son délégué duckdb.StartTimeCanonicalSQL).
//
// Usage :
//
//	`... ORDER BY ` + analysis.SQLStartTimeCanonical("mr") + ` DESC`
func SQLStartTimeCanonical(alias string) string {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	return "COALESCE(" + prefix + "start_time_utc, " + prefix + "start_time AT TIME ZONE 'UTC')"
}
