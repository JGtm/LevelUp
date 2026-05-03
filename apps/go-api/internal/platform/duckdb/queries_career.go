// Package duckdb — queries_career.go : requêtes filtres, historique, carrière et stats.
package duckdb

// Q4 : Filtres — chargement de tous les matchs d'un joueur pour la résolution.
// Paramètres : ?1 = xuid (match_participants), ?2 = xuid (match_participants).
// La vue mv_player_matches est préférée si disponible (test séparé Q4b).
const Q4MatchesForFilters = `
SELECT
    ms.match_id,
    ms.start_time,
    ms.map_name,
    COALESCE(ms.map_name_fr, ms.map_name)                AS map_name_fr,
    ms.pair_name,
    COALESCE(ms.pair_name_fr, ms.pair_name)              AS pair_name_fr,
    COALESCE(ms.playlist_name_fr, ms.playlist_name)      AS playlist_name,
    COALESCE(ms.is_firefight, FALSE)                     AS is_firefight,
    COALESCE(ms.is_ranked, FALSE)                        AS is_ranked,
    pme.session_id,
    pme.session_label,
    COALESCE(pme.is_with_friends, FALSE)                 AS is_with_friends,
    ms.playlist_name                                     AS playlist_name_en
FROM (
    SELECT r.match_id,
           COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
           r.map_name,
           r.map_name_fr, r.pair_name, r.pair_name_fr,
           r.playlist_name, r.playlist_name_fr,
           COALESCE(r.is_firefight, FALSE) AS is_firefight,
           COALESCE(r.is_ranked, FALSE)    AS is_ranked
    FROM shared.v_match_full r
    JOIN shared.match_participants p ON r.match_id = p.match_id
    WHERE p.xuid = ?
) ms
LEFT JOIN player_match_enrichment pme ON ms.match_id = pme.match_id
ORDER BY ms.start_time DESC`

// Q4MV : Filtres — variante avec mv_player_matches (vue matérialisée optimisée).
// Paramètre : ? = xuid.
const Q4MVMatchesForFilters = `
SELECT
    ms.match_id,
    ms.start_time,
    ms.map_name,
    COALESCE(ms.map_name_fr, ms.map_name)                AS map_name_fr,
    ms.pair_name,
    COALESCE(ms.pair_name_fr, ms.pair_name)              AS pair_name_fr,
    COALESCE(ms.playlist_name_fr, ms.playlist_name)      AS playlist_name,
    COALESCE(ms.is_firefight, FALSE)                     AS is_firefight,
    COALESCE(ms.is_ranked, FALSE)                        AS is_ranked,
    pme.session_id,
    pme.session_label,
    COALESCE(pme.is_with_friends, FALSE)                 AS is_with_friends,
    ms.playlist_name                                     AS playlist_name_en
FROM (
    SELECT match_id,
           COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC') AS start_time,
           map_id, map_name, map_name_fr,
           pair_name, pair_name_fr, playlist_name, playlist_name_fr,
           is_firefight, is_ranked
    FROM shared.mv_player_matches
    WHERE xuid = ?
) ms
LEFT JOIN player_match_enrichment pme ON ms.match_id = pme.match_id
ORDER BY ms.start_time DESC`

// Q5 : Historique — chargement complet avec stats du joueur.
// Paramètres : ? = xuid (match_participants), ? = xuid (enrichment).
const Q5MatchHistory = `
SELECT
    ms.match_id,
    ms.start_time,
    ms.map_name,
    COALESCE(ms.map_name_fr, ms.map_name)                AS map_name_fr,
    ms.pair_name,
    COALESCE(ms.pair_name_fr, ms.pair_name)              AS pair_name_fr,
    COALESCE(ms.playlist_name_fr, ms.playlist_name)      AS playlist_name,
    COALESCE(ms.is_firefight, FALSE)                     AS is_firefight,
    COALESCE(ms.is_ranked, FALSE)                        AS is_ranked,
    pme.session_id,
    pme.session_label,
    COALESCE(pme.is_with_friends, FALSE)                 AS is_with_friends,
    COALESCE(pme.is_excluded, FALSE)                     AS is_excluded,
    COALESCE(p.outcome, 0)                               AS outcome,
    p.team_mmr,
    p.enemy_mmr,
    COALESCE(p.kills, 0)                                 AS kills,
    COALESCE(p.deaths, 0)                                AS deaths,
    COALESCE(p.assists, 0)                               AS assists,
    p.kda,
    p.accuracy,
    p.personal_score,
    p.avg_life_seconds                                   AS average_life_seconds,
    p.time_played_seconds
FROM (
    SELECT r.match_id,
           COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
           r.map_name,
           r.map_name_fr, r.pair_name, r.pair_name_fr,
           r.playlist_name, r.playlist_name_fr,
           COALESCE(r.is_firefight, FALSE) AS is_firefight,
           COALESCE(r.is_ranked, FALSE)    AS is_ranked
    FROM shared.v_match_full r
    JOIN shared.match_participants p ON r.match_id = p.match_id
    WHERE p.xuid = ?
) ms
LEFT JOIN shared.match_participants p
    ON ms.match_id = p.match_id AND p.xuid = ?
LEFT JOIN player_match_enrichment pme
    ON ms.match_id = pme.match_id
ORDER BY ms.start_time DESC`

// Q6 : Career — progression de rang (dernière entrée).
// Paramètre : aucun (lit toujours le rang le plus récent).
const Q6CareerLatestRank = `
SELECT
    cp.rank          AS rank_number,
    cp.current_xp,
    cp.recorded_at,
    cp.rank_name     AS rank_label,
    cp.rank_name,
    cp.rank_tier,
    cp.xp_for_next_rank,
    cp.xp_total,
    cp.is_max_rank
FROM career_progression cp
ORDER BY cp.recorded_at DESC
LIMIT 1`

// Q7 : Career — historique XP complet.
const Q7CareerXPHistory = `
SELECT
    cp.recorded_at,
    cp.rank          AS rank_number,
    cp.current_xp,
    COALESCE(cp.xp_total, cp.rank * 1000) AS xp_total_cumulative
FROM career_progression cp
ORDER BY cp.recorded_at ASC`

// Q8 : Career — LUSR checkpoints (match_skill_rank).
// Paramètre : aucun (filtre global, une DB = un joueur).
// rating_delta calculé par LAG par playlist_group pour retrouver la progression Python.
const Q8LUSRHistory = `
SELECT
    msr.match_id,
    msr.rating_value,
    msr.tier_label,
    msr.playlist_group,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS recorded_at,
    msr.rating_value - LAG(msr.rating_value) OVER (
        PARTITION BY msr.playlist_group ORDER BY COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC')
    ) AS rating_delta
FROM match_skill_rank msr
LEFT JOIN shared.match_registry r ON msr.match_id = r.match_id
ORDER BY COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') ASC`

// Q9 : Career — top matches : 10 meilleurs (WIN) + 10 moins bons (LOSS).
// Paramètres : ?1 = xuid (section WIN), ?2 = xuid (section LOSS).
// Filtres Python portés : had_bot_teammate=FALSE, time_played>=180s, is_firefight=FALSE.
// Tri badge priority : WIN → flags 5/3/1 (CONTRE_REMONTADA/REMONTADA/DOMINATION) DESC ;
//
//	LOSS → flags 4/2 (DEBANDADE/HUMILIATION) DESC.
//
// _s sert uniquement à séparer les sections (1=best, 2=worst) dans l'ORDER BY final.
const Q9TopMatches = `
SELECT match_id, performance_score, start_time, map_name, pair_name, playlist_name,
       outcome, kills, deaths, kda, team_mmr, enemy_mmr, dominance_flag, _s
FROM (
    (
        SELECT
            pme.match_id,
            pme.performance_score,
            COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
            r.map_name,
            r.pair_name,
            r.playlist_name,
            COALESCE(p.outcome, 0)           AS outcome,
            COALESCE(p.kills, 0)             AS kills,
            COALESCE(p.deaths, 0)            AS deaths,
            p.kda,
            p.team_mmr,
            p.enemy_mmr,
            COALESCE(pme.dominance_flag, 0)  AS dominance_flag,
            1                                AS _s
        FROM player_match_enrichment pme
        JOIN shared.match_registry r ON pme.match_id = r.match_id
        LEFT JOIN shared.match_participants p
            ON pme.match_id = p.match_id AND p.xuid = ?
        WHERE pme.performance_score IS NOT NULL
          AND COALESCE(pme.had_bot_teammate, FALSE) = FALSE
          AND COALESCE(p.time_played_seconds, 0) >= 180
          AND COALESCE(r.is_firefight, FALSE) = FALSE
          AND COALESCE(p.outcome, 0) = 2
        ORDER BY
            CASE WHEN COALESCE(pme.dominance_flag, 0) IN (5, 3, 1)
                 THEN COALESCE(pme.dominance_flag, 0) ELSE 0 END DESC,
            pme.performance_score DESC
        LIMIT 10
    )
    UNION ALL
    (
        SELECT
            pme.match_id,
            pme.performance_score,
            COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
            r.map_name,
            r.pair_name,
            r.playlist_name,
            COALESCE(p.outcome, 0)           AS outcome,
            COALESCE(p.kills, 0)             AS kills,
            COALESCE(p.deaths, 0)            AS deaths,
            p.kda,
            p.team_mmr,
            p.enemy_mmr,
            COALESCE(pme.dominance_flag, 0)  AS dominance_flag,
            2                                AS _s
        FROM player_match_enrichment pme
        JOIN shared.match_registry r ON pme.match_id = r.match_id
        LEFT JOIN shared.match_participants p
            ON pme.match_id = p.match_id AND p.xuid = ?
        WHERE pme.performance_score IS NOT NULL
          AND COALESCE(pme.had_bot_teammate, FALSE) = FALSE
          AND COALESCE(p.time_played_seconds, 0) >= 180
          AND COALESCE(r.is_firefight, FALSE) = FALSE
          AND COALESCE(p.outcome, 0) = 3
        ORDER BY
            CASE WHEN COALESCE(pme.dominance_flag, 0) IN (4, 2)
                 THEN COALESCE(pme.dominance_flag, 0) ELSE 0 END DESC,
            pme.performance_score ASC
        LIMIT 10
    )
)
ORDER BY _s ASC`

// Q22 : Sessions — chargement des matchs pour le calcul des sessions.
// Parametre : ?1 = xuid du joueur.
// Retourne 6 colonnes : match_id, start_time, teammates_sig, is_ranked,
// time_played_seconds, end_time (NULL si absent).
const Q22SessionMatches = `
SELECT
    mp.match_id,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
    -- Signature des coeequipiers : XUIDs tries concatenes (hors joueur lui-meme)
    (SELECT string_agg(t.xuid, ',' ORDER BY t.xuid)
     FROM shared.match_participants t
     WHERE t.match_id = mp.match_id AND t.team_id = mp.team_id AND t.xuid <> ?
    )                                                   AS teammates_sig,
    COALESCE(r.is_ranked, FALSE)                       AS is_ranked,
    mp.time_played_seconds,
    CASE WHEN mp.time_played_seconds IS NOT NULL
         THEN COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') + INTERVAL (mp.time_played_seconds || ' seconds')
         ELSE NULL
    END                                                 AS end_time
FROM shared.match_participants mp
JOIN shared.match_registry r ON r.match_id = mp.match_id
WHERE mp.xuid = ?
ORDER BY COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') ASC`

// Q23 : Stats series — chargement des matchs avec metriques pour perf score.
// Parametre : ?1 = xuid du joueur.
// Retourne toutes les colonnes necessaires a compute_relative_performance_score.
const Q23StatsMatches = `
SELECT
    mp.match_id,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
    mp.outcome,
    COALESCE(mp.kills, 0)              AS kills,
    COALESCE(mp.deaths, 0)             AS deaths,
    COALESCE(mp.assists, 0)            AS assists,
    mp.kda,
    mp.accuracy,
    mp.personal_score,
    mp.damage_dealt,
    mp.damage_taken,
    mp.time_played_seconds,
    mp.team_mmr,
    mp.enemy_mmr,
    mp.kills_expected,
    mp.deaths_expected,
    mp.rank,
    COALESCE(r.is_ranked, FALSE)     AS is_ranked,
    COALESCE(r.playlist_name, '')    AS playlist_name,
    COALESCE(r.pair_name, '')        AS pair_name,
    mp.team_id,
    pme.performance_score             AS performance_score_computed,
    pme.session_id,
    pme.session_label,
    CASE
        WHEN COALESCE(mp.damage_dealt, 0) > 0 THEN
            225.0 * (COALESCE(mp.kills, 0) + COALESCE(mp.assists, 0) / 3.0) / mp.damage_dealt
        ELSE 0.0
    END AS offensive_conversion,
    CASE
        WHEN COALESCE(mp.damage_taken, 0) > 0 AND COALESCE(mp.deaths, 0) > 0 THEN
            mp.damage_taken / (225.0 * mp.deaths)
        ELSE 0.0
    END AS defensive_resistance
FROM shared.match_participants mp
JOIN shared.match_registry r ON r.match_id = mp.match_id
LEFT JOIN player_match_enrichment pme ON pme.match_id = mp.match_id
WHERE mp.xuid = ?
ORDER BY COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') ASC`

// Q24 : LUSR — chargement du rating par match depuis match_skill_rank.
// Parametre : ?1 = xuid du joueur (filtre via player_match_enrichment).
const Q24LUSRHistory = `
SELECT
    msr.match_id,
    msr.rating_value,
    msr.rating_deviation,
    msr.playlist_group
FROM match_skill_rank msr
ORDER BY msr.match_id ASC`

// Q25 : Stats series — participants d un ensemble de matchs (pour LUSR enemy strength).
// Parametre : ?1 = xuid joueur (pour filtrer son equipe).
// Utilise les matchs de Q23 comme sous-requete.
const Q25MatchParticipants = `
SELECT
    mp.match_id,
    mp.xuid,
    mp.team_id,
    mp.kills_expected,
    mp.deaths_expected
FROM shared.match_participants mp
WHERE mp.match_id IN (
    SELECT DISTINCT mp2.match_id
    FROM shared.match_participants mp2
    WHERE mp2.xuid = ?
)
ORDER BY mp.match_id, mp.xuid`
