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
//     queries complètes.
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

// SQLIsWin est le prédicat SQL pour un match gagné. Aligné sur
// domain.OutcomeWin == 2 (cf. internal/domain/outcomes.go).
//
// Usage : `SUM(CASE WHEN ` + SQLIsWin + ` THEN 1 ELSE 0 END) AS wins`
const SQLIsWin = `outcome = 2`

// SQLWinRateExpr est l'expression SQL canonique pour calculer un winrate
// 0..1 (ratio, pas pourcent). Aligné sur analysis.WinRate (ADR 0006).
//
// Usage : `SELECT ` + SQLWinRateExpr + ` AS win_rate FROM ...`
//
// Renvoie 0 si aucun match (NULLIF + COALESCE).
const SQLWinRateExpr = `COALESCE(` +
	`CAST(SUM(CASE WHEN outcome = 2 THEN 1 ELSE 0 END) AS DOUBLE) ` +
	`/ NULLIF(COUNT(*), 0), 0)`

// SQLKDRExpr est l'expression SQL canonique pour calculer un K/D ratio
// agrégé (sum(kills)/max(1,sum(deaths))). Aligné sur analysis.KDR.
//
// Note : c'est un KDR sur totaux, distinct de avg(KDR per match). Pour le
// "K/D moyen affiché" produit, préférer cette agrégation totaliste qui est
// stable face aux matchs aux scores extrêmes.
const SQLKDRExpr = `CAST(SUM(kills) AS DOUBLE) / NULLIF(SUM(deaths), 0)`
