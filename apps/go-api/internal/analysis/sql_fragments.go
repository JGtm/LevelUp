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
// IL NE SUFFIT PAS A DIRE QU'UN MATCH SERA CUIT : la file en exige davantage (cf.
// [SQLEligibleALaCuisson]). Il ne s'emploie seul que pour repondre a « ce match est-il
// encore dans la fenetre », jamais a « la file le reprendra ».
//
// L'horodatage passe par le fragment canonique (regle n 8) : `start_time` brut trierait et
// filtrerait faux. ATTENTION, LE PREDICAT PEUT VALOIR NULL : les deux horodatages du
// registre sont nullables, et `NULL >= x` vaut NULL. Ne jamais le scanner seul dans un
// `bool` — [SQLEligibleALaCuisson] le protege par un `IS NOT NULL` en amont du AND.
func SQLDansFenetreRetention(alias string) string {
	return SQLStartTimeCanonical(alias) + " >= ?"
}

// SQLEligibleALaCuisson rend le predicat « la file de cuisson des artefacts de rejeu
// reprendra ce match », avec ses parametres DANS L'ORDRE.
//
// # UNE SEULE DEFINITION, PARCE QUE DEUX PROMESSES CONTRADICTOIRES SONT PIRES QU'AUCUNE
//
// Deux composants repondent a cette question : la FILE
// (`sync/replayartifacts.requeteQueueRecente`), qui decide ce qu'elle cuit, et la LECTURE
// TACTIQUE, qui annonce a l'utilisateur ce qui va l'etre (`matchs_en_attente` contre
// `matchs_non_cuisables`). Si la seconde est plus large que la premiere, la page promet une
// cuisson que rien ne fera — et l'utilisateur attend indefiniment un ecran qui ne se
// remplira pas. C'est arrive : la premiere version comptait « en attente » les matchs dont
// le FILM EST DEFINITIVEMENT PERDU. Le garde-rail
// `internal/archlint/no_retention_window_inline_test.go` interdit toute autre formulation.
//
// # LES TROIS CONDITIONS
//
//	FILM PAS PERDU      `backfill_completed & bitFilmAbsent = 0`. Le marqueur est TERMINAL :
//	                    l'etape 1.57 le pose quand le film rend 404 ou 0 chunk, et il vaut
//	                    pour ~29 % du parc. Un match marque ne sera jamais cuit.
//	DATABLE             les deux horodatages du registre sont nullables. Un match sans date
//	                    ne peut ni etre ordonne par recence ni etre situe dans la fenetre :
//	                    la file ne le prend pas.
//	DANS LA FENETRE     seulement si la retention est bornee (`mois > 0`).
//
// # LE PREDICAT NE VAUT JAMAIS NULL, ET C'EST DELIBERE
//
// `IS NOT NULL` precede la comparaison de fenetre : en logique ternaire SQL,
// `FALSE AND NULL` vaut FALSE. Le resultat se scanne donc sans risque dans un `bool` nu.
// Sans cet ordre, un match a horodatages NULL faisait echouer le scan
// (`converting NULL to bool`) et rendait 500 sur TOUTE la lecture de la carte.
func SQLEligibleALaCuisson(alias string, mois int, bitFilmAbsent int64) (string, []any) {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	sql := "COALESCE(" + prefix + "backfill_completed, 0) & ? = 0" +
		" AND " + SQLStartTimeCanonical(alias) + " IS NOT NULL"
	args := []any{bitFilmAbsent}
	if borne, bornee := BorneRetention(mois); bornee {
		sql += " AND " + SQLDansFenetreRetention(alias)
		args = append(args, borne)
	}
	return sql, args
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
