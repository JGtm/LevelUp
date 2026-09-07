// Package analysis — sql_fragments.go : fragments SQL canoniques.
//
// Centralise les expressions SQL répétées dans les repositories DuckDB pour
// éviter les divergences (revue 2026-04-29 axe 6 DETTE 11 — `IsBot` SQL
// répété 8 fois, bugs causés par ces duplications).
//
// Convention :
//   - Préfixer les noms par SQL (ex: SQLIsBot)
//   - Ne pas paramétrer les noms de tables/alias — laisser le repository
//     les composer. Exceptions (fonctions) : SQLStartTimeCanonical(alias) —
//     l'expression référence la colonne aliasée deux fois ; SQLIsBotCol(col) /
//     SQLIsNotBotCol(col) — les copies utilisaient des préfixes variables
//     (mp.xuid, opp.xuid), un const nu re-divergeait (34 copies constatées).
//   - Toujours utiliser ces fragments via concaténation explicite, pas via
//     fmt.Sprintf (lisibilité + audit grep).
package analysis

import "time"

// SQLIsBotCol construit le prédicat SQL « colonne = xuid de bot » pour la colonne
// donnée (préfixe d'alias inclus : "xuid", "mp.xuid", "opp.xuid"…). Source unique
// du prédicat bot (préfixe bid(*), aligné sur analysis.IsBot côté Go). Le garde-rail
// archlint/no_raw_isbot_literal_test.go interdit le littéral brut ailleurs.
//
// Deux régimes d'usage selon le consommateur :
//   - chaîne backtick exécutée directement : concaténer
//     `... WHERE ` + analysis.SQLIsNotBotCol("mp.xuid") + ` ...`
//   - template fmt.Sprintf : l'injecter comme ARGUMENT %s (le `%` interne de
//     'bid(%' n'est PAS réinterprété par Sprintf), JAMAIS dans la chaîne de format.
//
// H2 (2026-07-04) : remplace les ex-const nues SQLIsBot/SQLIsNotBot (0 consommateur
// SQL — centralisation abandonnée, 34 copies littérales re-divergées) — cf. leçon
// CLAUDE.md règle 6.
func SQLIsBotCol(col string) string { return col + " LIKE 'bid(%'" }

// SQLIsNotBotCol est le prédicat opposé (exclusion des bots) pour la colonne donnée.
func SQLIsNotBotCol(col string) string { return col + " NOT LIKE 'bid(%'" }

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
// qualifiée. Les garde-rails vivent dans
// internal/archlint/no_raw_start_time_literal_test.go (le chemin
// analysis/start_time_canonical_test.go cité ici jusqu'au 2026-08-03 n'a jamais
// existé) : ils interdisent, hors de ce helper et de son délégué
// duckdb.StartTimeCanonicalSQL, la recopie du littéral, l'ORDER BY / CAST AS DATE
// sur start_time brut, et toute expression composée à la main autour de
// start_time_utc — c'est cette dernière forme qui a produit le bug 1.3
// (COALESCE mal parenthésé, AT TIME ZONE hors de la parenthèse).
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

// SQLDansFenetreRetention rend le predicat « ce match est DANS la fenetre de retention des
// artefacts de rejeu », a comparer a la borne rendue par [BorneRetention].
//
// # UNE SEULE DEFINITION DE « DANS LA FENETRE »
//
// La file de cuisson (`sync/replayartifacts.requeteQueueRecente`) et la lecture tactique
// s'en servent toutes les deux, et elles DOIVENT dire la meme chose : la premiere decide ce
// qui sera cuit, la seconde annonce a l'utilisateur ce qui va l'etre. Deux formulations
// auraient fini par diverger d'un `>` a un `>=` ou d'un fuseau, et la page aurait promis
// une cuisson qui n'arrive jamais — ou tu l'aurait dite impossible alors qu'elle est en
// file. Le garde-rail `internal/archlint/no_retention_window_inline_test.go` interdit toute
// autre formulation.
//
// L'horodatage passe par le fragment canonique (regle n°8) : `start_time` brut trierait et
// filtrerait faux.
func SQLDansFenetreRetention(alias string) string {
	return SQLStartTimeCanonical(alias) + " >= ?"
}

// BorneRetention rend l'instant a partir duquel un match est DANS la fenetre, et `true` si
// la fenetre est bornee.
//
// `mois <= 0` VEUT DIRE ILLIMITEE, jamais « zero mois » : c'est le reglage par defaut
// (`ReplayRetentionMonths`), et un match n'en sort alors jamais.
func BorneRetention(mois int) (time.Time, bool) {
	return BorneRetentionDepuis(time.Now().UTC(), mois)
}

// BorneRetentionDepuis est [BorneRetention] avec un instant de reference explicite.
//
// ELLE EXISTE POUR L'HORLOGE INJECTEE du cron de purge, qui doit pouvoir se placer a une
// date choisie dans ses tests. Le calcul reste ICI : c'est lui, et non l'appel a
// `time.Now`, qui doit rester unique — un `AddDate` recopie chez l'appelant est exactement
// la divergence que le garde-rail `no_retention_window_inline_test` interdit.
func BorneRetentionDepuis(now time.Time, mois int) (time.Time, bool) {
	if mois <= 0 {
		return time.Time{}, false
	}
	return now.AddDate(0, -mois, 0), true
}
