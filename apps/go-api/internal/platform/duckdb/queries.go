// Package duckdb — queries.go : requêtes SQL bootstrap et recherche gamertag.
//
// Chaque constante correspond à une requête de la cartographie Go-migration.
// Les paramètres positionnels DuckDB utilisent '?' (database/sql style).
// Les requêtes shared.* sont exécutées via SharedReader (ADR 0016, plus
// aucun ATTACH `shared` sur les conns du pool depuis le commit 9c.5).
//
// Requêtes par domaine :
//   - queries_career.go         : Q4Shared/Q4MVShared/Q5, Q6-Q9, Q22-Q25 (filtres, historique, carrière, stats)
//   - queries_match.go          : Q10, Q12-Q21 (encounters, scoreboard, events, armes)
//   - queries_squad.go          : Q29-Q33b (escouade, coéquipiers, synthèse)
//   - queries_home_citations.go : Q26-Q28, Q34-Q37 (home, citations, médias)
package duckdb

import "levelup/go-api/internal/analysis"

// Q1 : Bootstrap — nombre de matchs dans shared_matches_v2.
const Q1MatchCount = `SELECT COUNT(*) FROM match_registry`

// Q2 : Bootstrap — version DuckDB embarquée.
const Q2DBVersion = `SELECT version()`

// Q3 : Résolution XUID depuis sync_meta du joueur.
const Q3ResolveXUID = `SELECT value FROM sync_meta WHERE key = 'xuid'`

// Q11 : Gamertag search — fuzzy ranking sur xuid_aliases.
//
// Score combiné (du plus prioritaire au moins) :
//   - exact match (case-insensitive) : +1000
//   - prefix match                   : +200
//   - substring match                : +50
//   - similarité Jaro-Winkler        : +0..100 (typo tolerance)
//
// Filtre WHERE : substring OU jaro_winkler_similarity > 0.80
// (le seuil 0.80 attrape les typos courants — ex: "mst3rch1f" → "MasterChief").
//
// Tri secondaire : match_count DESC, puis gamertag ASC pour stabilité.
// Paramètre : ? = terme (bind unique via CTE).
//
// Perf : mesuré ~5-15ms sur 15k rows en DuckDB columnar (avril 2026).
// Pas d'index nécessaire jusqu'à ~100k rows ; au-delà, ajouter une colonne
// gamertag_lower générée + index ART dessus.
//
// Note schéma : GamertagRepo ouvre shared_matches_v2.duckdb directement (pas via
// pool player). Les tables sont dans main — pas de préfixe global./shared. ici.
var Q11GamertagSearch = `
WITH params AS (SELECT lower(?) AS q),
matched AS (
    SELECT
        xa.gamertag,
        xa.xuid,
        CASE WHEN lower(xa.gamertag) = p.q THEN 1000 ELSE 0 END
      + CASE WHEN lower(xa.gamertag) LIKE p.q || '%' THEN 200 ELSE 0 END
      + CASE WHEN lower(xa.gamertag) LIKE '%' || p.q || '%' THEN 50 ELSE 0 END
      + CAST(jaro_winkler_similarity(lower(xa.gamertag), p.q) * 100 AS INTEGER) AS score
    FROM xuid_aliases xa
    CROSS JOIN params p
    WHERE ` + analysis.SQLIsNotBotCol("xa.xuid") + `
      AND (   lower(xa.gamertag) LIKE '%' || p.q || '%'
           OR jaro_winkler_similarity(lower(xa.gamertag), p.q) > 0.80)
)
SELECT
    m.gamertag,
    m.xuid,
    COUNT(DISTINCT mp.match_id) AS match_count
FROM matched m
LEFT JOIN match_participants mp ON m.xuid = mp.xuid
GROUP BY m.gamertag, m.xuid, m.score
ORDER BY m.score DESC, match_count DESC, m.gamertag ASC
LIMIT 20`
