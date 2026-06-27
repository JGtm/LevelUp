package duckdb

// queries_relations_moments.go — SQL du sous-endpoint « Moments & Rivalités »
// (Phase 3a). Lecture seule sur le catalogue shared (match_participants +
// match_registry + killer_victim_pairs + v_gamertag_lookup) via SharedReader.
//
// Deux requêtes :
//   - Q29RelationsHeatmap : top-N relations (par matchs communs) × heure UTC
//     canonique, pour le heatmap agrégé « Quand tu les croises ». Le bucketing
//     en day-parts (6 tranches) est fait côté Go (analysis/relations.Daypart).
//   - Q30RivalTimeline : timeline d'un rival (matchs communs joués en ennemi),
//     avec frags directionnels moi↔lui par match. Result décidé title-aware.

// Q29RelationsHeatmapTpl : heatmap relation × heure. Restreint aux TOP-N
// relations (les plus de matchs communs, bots exclus) puis compte les matchs
// communs par (xuid, heure UTC). Le filtre top-N est fait par une sous-requête
// classée par count_together DESC (plafonné pour la lisibilité du heatmap).
//
// Format string : %s = inClause my_history (segmentation Phase 2, ou vide).
//
// Placeholders ? :
//
//	?1 my_history.WHERE xuid = ?
//	?…  scope IN (my_history) — N placeholders si scope non vide
//	?  encounters JOIN p.xuid <> ?
//	?  topN (LIMIT)
//
// Colonnes SELECT (5) : xuid, gamertag, hour, dow (0=dimanche…6=samedi UTC), count.
const Q29RelationsHeatmapTpl = `
WITH my_history AS (
    SELECT match_id, team_id
    FROM match_participants
    WHERE xuid = ?%s
),
encounters AS (
    SELECT
        p.xuid,
        p.match_id,
        EXTRACT(hour FROM (COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'))::INTEGER AS hour_utc,
        EXTRACT(dow FROM (COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'))::INTEGER AS dow_utc
    FROM my_history h
    JOIN match_participants p
        ON p.match_id = h.match_id
       AND p.xuid <> ?
       AND p.xuid NOT LIKE 'bid(%%'
    LEFT JOIN match_registry r ON r.match_id = p.match_id
),
top_xuids AS (
    SELECT xuid, COUNT(DISTINCT match_id) AS count_together
    FROM encounters
    GROUP BY xuid
    HAVING COUNT(DISTINCT match_id) >= 2
    ORDER BY count_together DESC, xuid ASC
    LIMIT ?
)
SELECT
    e.xuid,
    COALESCE(vg.gamertag, ('Joueur ' || RIGHT(e.xuid, 4))) AS gamertag,
    e.hour_utc AS hour,
    e.dow_utc AS dow,
    COUNT(DISTINCT e.match_id) AS cnt
FROM encounters e
JOIN top_xuids t ON t.xuid = e.xuid
LEFT JOIN v_gamertag_lookup vg ON vg.xuid = e.xuid
WHERE e.hour_utc IS NOT NULL
GROUP BY e.xuid, gamertag, e.hour_utc, e.dow_utc
ORDER BY e.xuid ASC, e.hour_utc ASC`

// Q30RivalTimelineTpl : timeline d'un rival (matchs communs joués EN ENNEMI),
// ordonnée ancien→récent. Joint match_registry ↔ p1 (moi) ↔ p2 (rival), ne
// garde que les matchs où l'on était dans des équipes différentes (duel réel),
// et agrège les frags directionnels moi↔lui depuis killer_victim_pairs.
//
// Result est décidé title-aware via outcomeSQLEq : %s = winExpr, %s = lossExpr
// (sur p1.outcome). Le 3e %s = inClause start_time/scope optionnel (segmentation
// Phase 2 sur r.match_id, ou vide). Le 4e %s = inClause sur kv.match_id (idem).
//
// Placeholders ? :
//
//	?1 duel CTE : kv.killer_xuid = ?  (moi, kills_on_rival : moi tueur)
//	?2 duel CTE : kv.victim_xuid = ?  (moi, deaths_by_rival : moi victime)
//	?3 duel CTE : WHERE killer_xuid = ? (moi)
//	?4 duel CTE : WHERE victim_xuid = ? (rival)
//	?5 duel CTE : WHERE killer_xuid = ? (rival)
//	?6 duel CTE : WHERE victim_xuid = ? (moi)
//	?…  scope IN (kv.match_id) — N placeholders si scope non vide
//	?  p1.xuid = ? (moi)
//	?  p2.xuid = ? (rival)
//	?…  scope IN (r.match_id) — N placeholders si scope non vide
//	?  LIMIT (N derniers)
//
// On garde les N derniers duels (ORDER BY start_time DESC LIMIT N dans la
// sous-requête) puis on les ré-ordonne ancien→récent pour la frise et le WR
// glissant. Colonnes SELECT (7) : match_id, start_time, result,
// kills_on_rival, deaths_by_rival, mode (pair_name), map_name.
const Q30RivalTimelineTpl = `
WITH duel AS (
    SELECT
        kv.match_id,
        SUM(CASE WHEN kv.killer_xuid = ? THEN kv.kill_count ELSE 0 END) AS kills_on_rival,
        SUM(CASE WHEN kv.victim_xuid = ? THEN kv.kill_count ELSE 0 END) AS deaths_by_rival
    FROM killer_victim_pairs kv
    WHERE ((kv.killer_xuid = ? AND kv.victim_xuid = ?)
        OR (kv.killer_xuid = ? AND kv.victim_xuid = ?))%s
    GROUP BY kv.match_id
),
recent AS (
    SELECT
        r.match_id,
        COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
        CASE WHEN %s THEN 1 WHEN %s THEN 2 ELSE 0 END AS result,
        COALESCE(d.kills_on_rival, 0)  AS kills_on_rival,
        COALESCE(d.deaths_by_rival, 0) AS deaths_by_rival,
        COALESCE(r.pair_name, '')               AS mode,
        COALESCE(r.map_name_fr, r.map_name, '') AS map_name
    FROM match_registry r
    JOIN match_participants p1 ON r.match_id = p1.match_id AND p1.xuid = ?
    JOIN match_participants p2 ON r.match_id = p2.match_id AND p2.xuid = ?
    LEFT JOIN duel d ON d.match_id = r.match_id
    WHERE p1.team_id <> p2.team_id%s
    ORDER BY start_time DESC, match_id DESC
    LIMIT ?
)
SELECT match_id, start_time, result, kills_on_rival, deaths_by_rival, mode, map_name
FROM recent
ORDER BY start_time ASC, match_id ASC`
