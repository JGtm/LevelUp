// Package duckdb — queries_squad.go : requêtes page Escouade et Synthèse.
package duckdb

// Q29 : Squad — top 10 coéquipiers les plus fréquents en escouade.
// Paramètres : ?1 = xuid (p1 joueur principal), ?2 = xuid (exclure le joueur principal de p2).
// Lit player_match_enrichment (player DB) JOIN shared.match_participants (x2).
const Q29TopTeammates = `
SELECT
    p2.xuid,
    COALESCE(vg.gamertag, p2.xuid)                              AS gamertag,
    COUNT(DISTINCT p1.match_id)                                  AS games_together,
    SUM(CASE WHEN p1.outcome = 2 THEN 1 ELSE 0 END)             AS wins_together,
    ROUND(
        SUM(CASE WHEN p1.outcome = 2 THEN 1 ELSE 0 END) * 100.0
        / NULLIF(COUNT(DISTINCT p1.match_id), 0), 1
    )                                                            AS win_rate,
    COALESCE(AVG(CAST(p2.kills AS DOUBLE)), 0.0)                 AS avg_kills,
    COALESCE(AVG(CAST(p2.deaths AS DOUBLE)), 0.0)                AS avg_deaths,
    COALESCE(AVG(p2.kda), 0.0)                                   AS avg_kda
FROM player_match_enrichment pme
JOIN shared.match_participants p1
    ON p1.match_id = pme.match_id AND p1.xuid = ?
JOIN shared.match_participants p2
    ON p2.match_id = p1.match_id
    AND p2.team_id  = p1.team_id
    AND p2.xuid    != ?
LEFT JOIN shared.v_gamertag_lookup vg ON vg.xuid = p2.xuid
WHERE pme.is_with_friends = TRUE
  AND p2.xuid NOT LIKE 'bid(%'
GROUP BY p2.xuid, vg.gamertag
ORDER BY games_together DESC
LIMIT 50`

// Q30 : Squad — matchs communs entre le joueur et un coéquipier spécifique.
// Paramètres : ?1 = xuid coéquipier (p2), ?2 = xuid joueur principal (p1).
const Q30SquadMatches = `
SELECT
    p1.match_id,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
    COALESCE(r.map_name, '')                                     AS map_name,
    COALESCE(r.map_name_fr, r.map_name, '')                      AS map_ui,
    COALESCE(r.pair_name_fr, r.pair_name, '')                    AS pair_name,
    COALESCE(r.playlist_name_fr, r.playlist_name, '')            AS playlist_name,
    COALESCE(r.is_firefight, FALSE)                              AS is_firefight,
    COALESCE(r.is_ranked, FALSE)                                 AS is_ranked,
    COALESCE(p1.outcome, 0)                                      AS outcome,
    COALESCE(p1.kills, 0)                                        AS kills,
    COALESCE(p1.deaths, 0)                                       AS deaths,
    COALESCE(p1.assists, 0)                                      AS assists,
    p1.kda,
    p1.accuracy,
    COALESCE(p1.time_played_seconds, 0)                          AS time_played_seconds,
    COALESCE(r.duration_seconds, 0)                              AS duration_seconds,
    COALESCE(p1.team_mmr, 0.0)                                   AS team_mmr,
    pme.session_id,
    pme.session_label,
    pme.performance_score,
    COALESCE(pme.is_with_friends, FALSE)                         AS is_with_friends,
    COALESCE(p1.headshot_kills, 0)                               AS headshot_kills,
    -- perfect_kills n'est pas une colonne de shared.match_participants ;
    -- on l'agrège depuis shared.medals_earned (medal_name_id = 1512363953
    -- "Perfect"), même approche que Q12MatchScoreboard.
    COALESCE((
        SELECT SUM(me.count)
        FROM shared.medals_earned me
        WHERE me.match_id = p1.match_id
          AND me.xuid = p1.xuid
          AND me.medal_name_id = 1512363953
    ), 0)::INTEGER                                              AS perfect_kills,
    p1.enemy_mmr,
    CASE WHEN p1.team_id = 0 THEN r.team_0_score ELSE r.team_1_score END AS my_team_score,
    CASE WHEN p1.team_id = 0 THEN r.team_1_score ELSE r.team_0_score END AS enemy_team_score,
    COALESCE(r.map_id, '')                                               AS map_id,
    COALESCE(r.playlist_id, '')                                          AS playlist_id
FROM shared.match_participants p1
JOIN shared.v_match_full r ON r.match_id = p1.match_id
JOIN shared.match_participants p2
    ON p2.match_id = p1.match_id
    AND p2.team_id  = p1.team_id
    AND p2.xuid     = ?
LEFT JOIN player_match_enrichment pme ON pme.match_id = p1.match_id
WHERE p1.xuid = ?
ORDER BY COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') DESC`

// Q31 : Squad — stats d'un coéquipier sur les matchs communs.
// Paramètres : ?1 = xuid joueur principal (p_main), ?2 = xuid coéquipier (p).
const Q31TeammateMatches = `
SELECT
    p.match_id,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
    COALESCE(r.map_name_fr, r.map_name, '')                      AS map_ui,
    COALESCE(r.pair_name, '')                                    AS pair_name,
    COALESCE(p.outcome, 0)                                       AS outcome,
    COALESCE(p.kills, 0)                                         AS kills,
    COALESCE(p.deaths, 0)                                        AS deaths,
    COALESCE(p.assists, 0)                                       AS assists,
    p.kda                                                        AS ratio,
    COALESCE(p.time_played_seconds, 0)                           AS time_played_seconds,
    COALESCE(p.team_mmr, 0.0)                                    AS team_mmr,
    p.accuracy,
    CASE WHEN p.team_id = 0 THEN r.team_0_score ELSE r.team_1_score END AS my_team_score,
    CASE WHEN p.team_id = 0 THEN r.team_1_score ELSE r.team_0_score END AS enemy_team_score
FROM shared.match_participants p
JOIN shared.v_match_full r ON r.match_id = p.match_id
JOIN shared.match_participants p_main
    ON p_main.match_id = p.match_id
    AND p_main.team_id  = p.team_id
    AND p_main.xuid     = ?
WHERE p.xuid = ?
ORDER BY COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') DESC`

// Q32SquadImpactEventsTemplate : template SQL pour charger les events d'impact escouade.
// Les '?' positionnels sont insérés dynamiquement (fmt.Sprintf(Q32SquadImpactEventsTemplate, placeholders)).
// Ne PAS utiliser directement — passer par squad_repo.LoadImpactEvents().
const Q32SquadImpactEventsTemplate = `
SELECT
    he.match_id,
    he.xuid,
    COALESCE(vg.gamertag, he.xuid)   AS gamertag,
    he.event_type,
    COALESCE(he.time_ms, 0)           AS time_ms
FROM highlight_events he
LEFT JOIN v_gamertag_lookup vg ON vg.xuid = he.xuid
WHERE he.match_id IN (%s)
ORDER BY he.match_id, he.time_ms`

// Q32bMainTeamParticipantsTemplate : pour chaque match dans la liste, retourne
// tous les participants de la même team_id que le joueur principal (le main
// inclus). Utilisé par buildSquadImpactMatrix pour calculer les badges
// d'impact en périmètre team-wide alliée (parité Python team_xuids), au lieu
// de se limiter au squad sélectionné.
//
// Le 1er '?' est l'XUID du main player, suivi de N '?' pour les match_ids.
// Ne PAS utiliser directement — passer par squad_repo.LoadMainTeamParticipants().
const Q32bMainTeamParticipantsTemplate = `
SELECT
    p.match_id,
    p.xuid,
    COALESCE(vg.gamertag, p.xuid)        AS gamertag,
    COALESCE(p.kills, 0)                 AS kills,
    COALESCE(p.deaths, 0)                AS deaths,
    COALESCE(p.assists, 0)               AS assists,
    COALESCE(p.outcome, 0)               AS outcome
FROM match_participants p
JOIN match_participants main
    ON main.match_id = p.match_id
    AND main.xuid    = ?
    AND p.team_id    = main.team_id
LEFT JOIN v_gamertag_lookup vg ON vg.xuid = p.xuid
WHERE p.match_id IN (%s)`

// Q33 : Synthèse — heatmap win rate par combinaison carte × mode.
// Paramètre : ?1 = xuid du joueur.
const Q33SynthesisHeatmap = `
SELECT
    COALESCE(r.map_name_fr, r.map_name, 'Unknown')    AS map_name,
    COALESCE(r.pair_name_fr, r.pair_name, 'Unknown')  AS mode_name,
    COUNT(DISTINCT p.match_id)                         AS match_count,
    SUM(CASE WHEN p.outcome = 2 THEN 1 ELSE 0 END)    AS wins
FROM match_participants p
JOIN v_match_full r ON r.match_id = p.match_id
WHERE p.xuid = ?
GROUP BY 1, 2
ORDER BY match_count DESC`

// Q33b : Synthèse — matchs du joueur pour le calcul top_weeks, KPIs et bipolaire.
// Sprint 43 : enrichi avec accuracy, time_played_seconds, performance_score.
// Sprint N  : ajout session_label pour les filtres de session teammates.
// Sprint N+1 : ajout is_ranked, is_firefight, playlist_name pour le câblage cascade.
// Paramètre : ?1 = xuid du joueur.
const Q33bSynthesisMatches = `
SELECT
    r.match_id,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
    p.outcome,
    p.kills,
    p.deaths,
    p.kda,
    COALESCE(pme.is_with_friends, FALSE)  AS is_with_friends,
    p.accuracy,
    p.time_played_seconds,
    pme.performance_score,
    pme.session_label,
    COALESCE(r.is_ranked, FALSE)          AS is_ranked,
    COALESCE(r.is_firefight, FALSE)       AS is_firefight,
    COALESCE(r.playlist_name, '')         AS playlist_name
FROM shared.match_participants p
JOIN shared.match_registry r ON r.match_id = p.match_id
LEFT JOIN player_match_enrichment pme ON r.match_id = pme.match_id
WHERE p.xuid = ?
ORDER BY COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') DESC`

// Q42MapStatsForSquadTemplate : taux de victoire et performance moyenne par carte
// du joueur principal, restreint aux matchs où TOUS les xuids de l'escouade
// sélectionnée sont participants. Aucun filtre temporel — c'est l'historique
// complet "avec cette escouade exacte". Si le squad est composé d'un seul xuid
// (uniquement le main = solo), retombe sur les stats solo+squad du main.
//
// Construit dynamiquement (fmt.Sprintf) car la clause IN dépend de la taille
// du squad. Ne PAS utiliser directement — passer par squad_repo.LoadMapStatsForSquad().
//
// Paramètres positionnels :
//   - autant de '?' que de xuids dans le squad (clause IN)
//   - 1 '?' pour le main xuid (filtre des rows main)
const Q42MapStatsForSquadTemplate = `
WITH squad_matches AS (
    SELECT match_id
    FROM shared.match_participants
    WHERE xuid IN (%s)
    GROUP BY match_id
    HAVING COUNT(DISTINCT xuid) = %d
)
SELECT COALESCE(r.map_id, '')                            AS map_id,
       COUNT(*)                                           AS total,
       SUM(CASE WHEN mp.outcome = 2 THEN 1 ELSE 0 END)   AS wins,
       AVG(pme.performance_score)                         AS perf_avg
FROM shared.match_participants mp
JOIN shared.match_registry r       ON r.match_id = mp.match_id
JOIN squad_matches sm              ON sm.match_id = mp.match_id
LEFT JOIN player_match_enrichment pme ON pme.match_id = mp.match_id
WHERE mp.xuid = ?
  AND COALESCE(r.map_id, '') <> ''
GROUP BY r.map_id`
