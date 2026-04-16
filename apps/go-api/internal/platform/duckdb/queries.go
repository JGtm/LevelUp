// Package duckdb — queries.go : requêtes SQL bootstrap et recherche gamertag.
//
// Chaque constante correspond à une requête de la cartographie Go-migration.
// Les paramètres positionnels DuckDB utilisent '?' (database/sql style).
// Toutes les requêtes supposent que shared est ATTACH-é sous l'alias "shared".
//
// Requêtes par domaine :
//   - queries_career.go         : Q4/Q4MV/Q5, Q6-Q9, Q22-Q25 (filtres, historique, carrière, stats)
//   - queries_match.go          : Q10, Q12-Q21 (encounters, scoreboard, events, armes)
//   - queries_squad.go          : Q29-Q33b (escouade, coéquipiers, synthèse)
//   - queries_home_citations.go : Q26-Q28, Q34-Q37 (home, citations, médias)
package duckdb

// Q1 : Bootstrap — nombre de matchs dans shared_matches_v2.
const Q1MatchCount = `SELECT COUNT(*) FROM shared.match_registry`

// Q2 : Bootstrap — version DuckDB embarquée.
const Q2DBVersion = `SELECT version()`

// Q3 : Résolution XUID depuis sync_meta du joueur.
const Q3ResolveXUID = `SELECT value FROM sync_meta WHERE key = 'xuid'`

// Q11 : Gamertag search — recherche partielle dans xuid_aliases.
// Paramètre : ? = terme (substring, ILIKE).
const Q11GamertagSearch = `
SELECT
    xa.gamertag,
    xa.xuid,
    COUNT(DISTINCT mp.match_id) AS match_count
FROM shared.xuid_aliases xa
LEFT JOIN shared.match_participants mp ON xa.xuid = mp.xuid
WHERE xa.gamertag ILIKE '%' || ? || '%'
GROUP BY xa.gamertag, xa.xuid
ORDER BY match_count DESC, xa.gamertag ASC
LIMIT 20`
