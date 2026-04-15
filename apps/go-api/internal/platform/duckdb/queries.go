// Package duckdb — queries.go : requêtes SQL canoniques Q1-Q16.
//
// Chaque constante correspond à une requête de la cartographie Go-migration.
// Les paramètres positionnels DuckDB utilisent '?' (database/sql style).
// Toutes les requêtes supposent que shared est ATTACH-é sous l'alias "shared".
package duckdb

// Q1 : Bootstrap — nombre de matchs dans shared_matches_v2.
const Q1MatchCount = `SELECT COUNT(*) FROM shared.match_registry`

// Q2 : Bootstrap — version DuckDB embarquée.
const Q2DBVersion = `SELECT version()`

// Q3 : Résolution XUID depuis sync_meta du joueur.
const Q3ResolveXUID = `SELECT value FROM sync_meta WHERE key = 'xuid'`

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
    COALESCE(pme.is_with_friends, FALSE)                 AS is_with_friends
FROM (
    SELECT r.match_id, r.start_time, r.map_name,
           NULL AS map_name_fr, r.pair_name, NULL AS pair_name_fr,
           r.playlist_name, NULL AS playlist_name_fr,
           COALESCE(r.is_firefight, FALSE) AS is_firefight,
           COALESCE(r.is_ranked, FALSE)    AS is_ranked
    FROM shared.match_registry r
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
    COALESCE(pme.is_with_friends, FALSE)                 AS is_with_friends
FROM (
    SELECT match_id, start_time, map_id, map_name, map_name_fr,
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
    SELECT r.match_id, r.start_time, r.map_name,
           NULL AS map_name_fr, r.pair_name, NULL AS pair_name_fr,
           r.playlist_name, NULL AS playlist_name_fr,
           COALESCE(r.is_firefight, FALSE) AS is_firefight,
           COALESCE(r.is_ranked, FALSE)    AS is_ranked
    FROM shared.match_registry r
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
    cp.rank_number,
    cp.current_xp,
    cp.recorded_at,
    cr.rank_label,
    cr.rank_name,
    cr.rank_tier,
    cr.xp_for_next_rank,
    cr.xp_total,
    cr.is_max_rank
FROM career_progression cp
LEFT JOIN shared.career_ranks cr ON cp.rank_number = cr.rank_number
ORDER BY cp.recorded_at DESC
LIMIT 1`

// Q7 : Career — historique XP complet.
const Q7CareerXPHistory = `
SELECT
    cp.recorded_at,
    cp.rank_number,
    cp.current_xp,
    COALESCE(cr.xp_total, cp.rank_number * 1000) AS xp_total_cumulative
FROM career_progression cp
LEFT JOIN shared.career_ranks cr ON cp.rank_number = cr.rank_number
ORDER BY cp.recorded_at ASC`

// Q8 : Career — LUSR checkpoints (match_skill_rank).
// Paramètre : ? = xuid (pour compatibilité avec la v5.3 qui filtre par joueur).
const Q8LUSRHistory = `
SELECT
    msr.match_id,
    msr.rating_value,
    msr.tier_label,
    msr.playlist_group,
    r.start_time AS recorded_at
FROM match_skill_rank msr
LEFT JOIN shared.match_registry r ON msr.match_id = r.match_id
ORDER BY r.start_time ASC`

// Q9 : Career — top matches (meilleur performance_score).
// Paramètre : ? = xuid.
const Q9TopMatches = `
SELECT
    pme.match_id,
    pme.performance_score,
    r.start_time,
    r.map_name,
    r.pair_name,
    r.playlist_name,
    COALESCE(p.outcome, 0)   AS outcome,
    COALESCE(p.kills, 0)     AS kills,
    COALESCE(p.deaths, 0)    AS deaths,
    p.kda,
    p.team_mmr,
    p.enemy_mmr
FROM player_match_enrichment pme
JOIN shared.match_registry r      ON pme.match_id = r.match_id
LEFT JOIN shared.match_participants p
    ON pme.match_id = p.match_id AND p.xuid = ?
WHERE pme.performance_score IS NOT NULL
ORDER BY pme.performance_score DESC
LIMIT 20`

// Q10 : Career — encounters (adversaires et coéquipiers fréquents).
// Paramètre : ? = xuid du joueur.
const Q10Encounters = `
SELECT
    p2.xuid,
    COALESCE(xa.gamertag, p2.xuid) AS gamertag,
    COUNT(*) AS match_count,
    SUM(CASE WHEN p2.team_id = p1.team_id THEN 1 ELSE 0 END) AS as_teammate,
    SUM(CASE WHEN p2.team_id != p1.team_id THEN 1 ELSE 0 END) AS as_enemy,
    AVG(p2.kda)                                               AS avg_kda
FROM shared.match_participants p1
JOIN shared.match_participants p2
    ON p1.match_id = p2.match_id AND p2.xuid != p1.xuid
LEFT JOIN shared.xuid_aliases xa ON p2.xuid = xa.xuid
WHERE p1.xuid = ?
GROUP BY p2.xuid, gamertag
HAVING match_count >= 2
ORDER BY match_count DESC
LIMIT 50`

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

// Q12 : Match view — scoreboard complet d'un match.
// Paramètre : ? = match_id.
const Q12MatchScoreboard = `
SELECT
    p.xuid,
    COALESCE(xa.gamertag, p.xuid)  AS gamertag,
    p.team_id,
    p.rank_in_team,
    COALESCE(p.outcome, 0)         AS outcome,
    p.personal_score,
    COALESCE(p.kills, 0)           AS kills,
    COALESCE(p.deaths, 0)          AS deaths,
    COALESCE(p.assists, 0)         AS assists,
    p.kda,
    p.accuracy,
    p.time_played_seconds,
    p.team_mmr,
    p.enemy_mmr
FROM shared.match_participants p
LEFT JOIN shared.xuid_aliases xa ON p.xuid = xa.xuid
WHERE p.match_id = ?
ORDER BY p.team_id, p.rank_in_team NULLS LAST`

// Q13 : Match view — métadonnées du match.
// Paramètre : ? = match_id.
const Q13MatchMeta = `
SELECT
    r.match_id,
    r.start_time,
    r.duration_seconds,
    r.map_name,
    r.pair_name,
    r.playlist_name,
    COALESCE(r.is_firefight, FALSE) AS is_firefight,
    COALESCE(r.is_ranked, FALSE)    AS is_ranked
FROM shared.match_registry r
WHERE r.match_id = ?`

// Q14 : Médailles d'un joueur pour un match.
// Paramètres : ?1 = xuid, ?2 = match_id.
const Q14MatchMedals = `
SELECT
    me.medal_id,
    me.count,
    COALESCE(cm.citation_label, me.medal_id) AS label
FROM shared.medals_earned me
LEFT JOIN shared.citation_mappings cm ON me.medal_id = cm.medal_id
WHERE me.xuid = ? AND me.match_id = ?
ORDER BY me.count DESC`

// Q15 : Events highlight d'un joueur pour un match.
// Paramètres : ?1 = xuid, ?2 = match_id.
const Q15MatchEvents = `
SELECT
    he.event_type,
    he.tick_count,
    he.timestamp_utc
FROM shared.highlight_events he
WHERE he.match_id = ?
ORDER BY he.tick_count ASC`

// Q16 : Weapon kills d'un joueur pour un match.
// Paramètres : ?1 = xuid, ?2 = match_id.
const Q16WeaponKills = `
SELECT
    wk.weapon_id,
    COALESCE(wl.label_fr, wl.label_en, CAST(wk.weapon_id AS VARCHAR)) AS weapon_label,
    SUM(wk.kill_count) AS kills
FROM shared.weapon_kills wk
LEFT JOIN metadata.weapon_labels wl ON wk.weapon_id = wl.weapon_id
WHERE wk.xuid = ? AND wk.match_id = ?
GROUP BY wk.weapon_id, weapon_label
ORDER BY kills DESC`

// Q17 : Stats d'un joueur pour un match spécifique (match_participants).
// Paramètres : ?1 = xuid, ?2 = match_id.
// Retourne 15 colonnes : outcome_code, team_id, rank_in_team, kills, deaths,
// assists, kda, accuracy, personal_score, avg_life_seconds, time_played_seconds,
// shots_fired, shots_hit, damage_dealt, damage_taken.
const Q17PlayerMatchStats = `
SELECT
    COALESCE(p.outcome, 0)         AS outcome_code,
    p.team_id,
    p.rank_in_team,
    COALESCE(p.kills, 0)           AS kills,
    COALESCE(p.deaths, 0)          AS deaths,
    COALESCE(p.assists, 0)         AS assists,
    p.kda,
    p.accuracy,
    p.personal_score,
    p.avg_life_seconds,
    p.time_played_seconds,
    p.shots_fired,
    p.shots_hit,
    p.damage_dealt,
    p.damage_taken
FROM shared.match_participants p
WHERE p.match_id = ? AND p.xuid = ?`

// Q18 : Enrichissement joueur pour un match (player_match_enrichment).
// Paramètre : ? = match_id.
// Retourne 2 colonnes : performance_score, is_with_friends.
const Q18MatchEnrichment = `
SELECT
    pme.performance_score,
    COALESCE(pme.is_with_friends, FALSE) AS is_with_friends
FROM player_match_enrichment pme
WHERE pme.match_id = ?`

// Q19 : Matchs communs entre 2 joueurs.
// Paramètres : ?1 = xuid joueur principal, ?2 = xuid autre joueur.
// Retourne 7 colonnes : match_id, start_time, map_ui, mode_ui,
// player1_team_id, player2_team_id, player1_outcome.
const Q19CommonMatches = `
SELECT
    r.match_id,
    r.start_time,
    COALESCE(r.map_name, '')   AS map_ui,
    COALESCE(r.pair_name, '')  AS mode_ui,
    p1.team_id                 AS player1_team_id,
    p2.team_id                 AS player2_team_id,
    COALESCE(p1.outcome, 0)    AS player1_outcome
FROM shared.match_registry r
JOIN shared.match_participants p1 ON r.match_id = p1.match_id AND p1.xuid = ?
JOIN shared.match_participants p2 ON r.match_id = p2.match_id AND p2.xuid = ?
ORDER BY r.start_time DESC
LIMIT 100`

// Q20 : Paires killer→victim pour un match (vue v6 v_killer_victim_full).
// Paramètre : ? = match_id.
// Retourne 6 colonnes : killer_xuid, killer_gamertag, victim_xuid,
// victim_gamertag, kill_count, time_ms.
const Q20KVPairs = `
SELECT
    kvf.killer_xuid,
    kvf.killer_gamertag,
    kvf.victim_xuid,
    kvf.victim_gamertag,
    kvf.kill_count,
    kvf.time_ms
FROM shared.v_killer_victim_full kvf
WHERE kvf.match_id = ?
ORDER BY kvf.time_ms ASC`

// Q21 : Événements highlight avec xuid pour un match complet.
// Paramètre : ? = match_id.
// Utilisé pour afficher les events dans le combat tab.
const Q21MatchEventsWithXUID = `
SELECT
    he.event_type,
    he.tick_count,
    he.timestamp_utc,
    he.xuid
FROM shared.highlight_events he
WHERE he.match_id = ?
ORDER BY he.tick_count ASC`


// Q22 : Sessions — chargement des matchs pour le calcul des sessions.
// Parametre : ?1 = xuid du joueur.
// Retourne 6 colonnes : match_id, start_time, teammates_sig, is_ranked,
// time_played_seconds, end_time (NULL si absent).
const Q22SessionMatches = `
SELECT
    mp.match_id,
    r.start_time,
    -- Signature des coeequipiers : XUIDs tries concatenes (hors joueur lui-meme)
    (SELECT string_agg(t.xuid, ',' ORDER BY t.xuid)
     FROM shared.match_participants t
     WHERE t.match_id = mp.match_id AND t.team_id = mp.team_id AND t.xuid <> ?
    )                                                   AS teammates_sig,
    COALESCE(mp.is_ranked, FALSE)                       AS is_ranked,
    mp.time_played_seconds,
    CASE WHEN mp.time_played_seconds IS NOT NULL
         THEN r.start_time + INTERVAL (mp.time_played_seconds || ' seconds')
         ELSE NULL
    END                                                 AS end_time
FROM shared.match_participants mp
JOIN shared.match_registry r ON r.match_id = mp.match_id
WHERE mp.xuid = ?
ORDER BY r.start_time ASC`

// Q23 : Stats series — chargement des matchs avec metriques pour perf score.
// Parametre : ?1 = xuid du joueur.
// Retourne toutes les colonnes necessaires a compute_relative_performance_score.
const Q23StatsMatches = `
SELECT
    mp.match_id,
    r.start_time,
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
    mp.is_ranked,
    COALESCE(r.playlist_name, '')    AS playlist_name,
    COALESCE(r.pair_name, '')        AS pair_name,
    mp.team_id,
    pme.performance_score             AS performance_score_computed,
    pme.session_id,
    pme.session_label
FROM shared.match_participants mp
JOIN shared.match_registry r ON r.match_id = mp.match_id
LEFT JOIN player_match_enrichment pme ON pme.match_id = mp.match_id
WHERE mp.xuid = ?
ORDER BY r.start_time ASC`

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

// Q26 : Home — matchs recents d un joueur avec KPIs pour le hero card.
// Parametre : ?1 = xuid du joueur.
// Retourne les 200 derniers matchs (hero card, highlights, recent matches, sessions).
const Q26HomeMatches = `
SELECT
    mp.match_id,
    r.start_time,
    COALESCE(r.map_name, '')                                AS map_name,
    COALESCE(r.pair_name, '')                               AS pair_name,
    COALESCE(r.playlist_name, '')                           AS playlist_name,
    COALESCE(r.is_firefight, FALSE)                         AS is_firefight,
    COALESCE(r.is_ranked, FALSE)                            AS is_ranked,
    pme.session_label,
    COALESCE(pme.is_with_friends, FALSE)                    AS is_with_friends,
    COALESCE(mp.outcome, 0)                                 AS outcome,
    COALESCE(mp.kills, 0)                                   AS kills,
    COALESCE(mp.deaths, 0)                                  AS deaths,
    COALESCE(mp.assists, 0)                                 AS assists,
    mp.kda,
    CASE WHEN COALESCE(mp.deaths, 0) > 0
         THEN CAST(COALESCE(mp.kills, 0) AS DOUBLE) / CAST(mp.deaths AS DOUBLE)
         ELSE CAST(COALESCE(mp.kills, 0) AS DOUBLE) END     AS ratio,
    mp.accuracy,
    mp.time_played_seconds
FROM shared.match_participants mp
JOIN shared.match_registry r ON r.match_id = mp.match_id
LEFT JOIN player_match_enrichment pme ON pme.match_id = mp.match_id
WHERE mp.xuid = ?
ORDER BY r.start_time DESC
LIMIT 200`

// Q27 : Home — sessions depuis player_match_enrichment.
// Pas de parametre (les donnees sont dans la DB joueur).
// Retourne les matchs avec un label de session pour le resumé solo/escouade.
const Q27HomeSessions = `
SELECT
    pme.match_id,
    pme.session_id,
    pme.session_label,
    COALESCE(pme.is_with_friends, FALSE)    AS is_with_friends,
    r.start_time
FROM player_match_enrichment pme
LEFT JOIN shared.match_registry r ON r.match_id = pme.match_id
WHERE pme.session_label IS NOT NULL
ORDER BY r.start_time DESC`

// Q28 : Home — medias recents depuis media_files + media_match_associations.
// Parametre : ?1 = LIMIT (nombre de medias).
// Retourne uniquement les medias actifs, triés par date de modification desc.
const Q28RecentMedia = `
SELECT
    mf.file_name,
    mma.match_id,
    mma.match_start_time
FROM media_files mf
LEFT JOIN media_match_associations mma ON mf.file_path = mma.media_path
WHERE mf.status = 'active'
ORDER BY mf.mtime DESC
LIMIT ?`

// =============================================================================
// Sprint 12 — Escouade + Synthèse
// =============================================================================

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
GROUP BY p2.xuid, vg.gamertag
ORDER BY games_together DESC
LIMIT 10`

// Q30 : Squad — matchs communs entre le joueur et un coéquipier spécifique.
// Paramètres : ?1 = xuid coéquipier (p2), ?2 = xuid joueur principal (p1).
const Q30SquadMatches = `
SELECT
    p1.match_id,
    r.start_time,
    COALESCE(r.map_name, '')                                     AS map_name,
    COALESCE(r.map_name_fr, r.map_name, '')                      AS map_ui,
    COALESCE(r.pair_name, '')                                    AS pair_name,
    COALESCE(r.playlist_name, '')                                AS playlist_name,
    COALESCE(r.is_firefight, FALSE)                              AS is_firefight,
    COALESCE(r.is_ranked, FALSE)                                 AS is_ranked,
    COALESCE(p1.outcome, 0)                                      AS outcome,
    COALESCE(p1.kills, 0)                                        AS kills,
    COALESCE(p1.deaths, 0)                                       AS deaths,
    COALESCE(p1.assists, 0)                                      AS assists,
    p1.kda,
    p1.accuracy,
    COALESCE(p1.time_played_seconds, 0)                          AS time_played_seconds,
    COALESCE(p1.team_mmr, 0.0)                                   AS team_mmr,
    pme.session_id,
    pme.session_label,
    pme.performance_score,
    COALESCE(pme.is_with_friends, FALSE)                         AS is_with_friends
FROM shared.match_participants p1
JOIN shared.match_registry r ON r.match_id = p1.match_id
JOIN shared.match_participants p2
    ON p2.match_id = p1.match_id
    AND p2.team_id  = p1.team_id
    AND p2.xuid     = ?
LEFT JOIN player_match_enrichment pme ON pme.match_id = p1.match_id
WHERE p1.xuid = ?
ORDER BY r.start_time DESC`

// Q31 : Squad — stats d'un coéquipier sur les matchs communs.
// Paramètres : ?1 = xuid joueur principal (p_main), ?2 = xuid coéquipier (p).
const Q31TeammateMatches = `
SELECT
    p.match_id,
    r.start_time,
    COALESCE(r.map_name_fr, r.map_name, '')                      AS map_ui,
    COALESCE(r.pair_name, '')                                    AS pair_name,
    COALESCE(p.outcome, 0)                                       AS outcome,
    COALESCE(p.kills, 0)                                         AS kills,
    COALESCE(p.deaths, 0)                                        AS deaths,
    COALESCE(p.assists, 0)                                       AS assists,
    p.kda                                                        AS ratio,
    COALESCE(p.time_played_seconds, 0)                           AS time_played_seconds,
    COALESCE(p.team_mmr, 0.0)                                    AS team_mmr,
    p.accuracy
FROM shared.match_participants p
JOIN shared.match_registry r ON r.match_id = p.match_id
JOIN shared.match_participants p_main
    ON p_main.match_id = p.match_id
    AND p_main.team_id  = p.team_id
    AND p_main.xuid     = ?
WHERE p.xuid = ?
ORDER BY r.start_time DESC`

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
FROM shared.highlight_events he
LEFT JOIN shared.v_gamertag_lookup vg ON vg.xuid = he.xuid
WHERE he.match_id IN (%s)
ORDER BY he.match_id, he.time_ms`

// Q33 : Synthèse — heatmap win rate par combinaison carte × mode.
// Paramètre : ?1 = xuid du joueur.
const Q33SynthesisHeatmap = `
SELECT
    COALESCE(r.map_name_fr, r.map_name, 'Unknown')    AS map_name,
    COALESCE(r.pair_name_fr, r.pair_name, 'Unknown')  AS mode_name,
    COUNT(DISTINCT p.match_id)                         AS match_count,
    SUM(CASE WHEN p.outcome = 2 THEN 1 ELSE 0 END)    AS wins
FROM shared.match_participants p
JOIN shared.match_registry r ON r.match_id = p.match_id
WHERE p.xuid = ?
GROUP BY map_name, mode_name
ORDER BY match_count DESC`

// =============================================================================
// Sprint 13 — Citations + Médias
// =============================================================================

// Q34 : Citations — mappings de citation depuis metadata.duckdb.
// Paramètre : aucun. Requête sur pdb.Metadata (pas pdb.Player).
const Q34CitationMappings = `
SELECT
    citation_name_norm,
    citation_name_display,
    mapping_type,
    COALESCE(category, 'misc')    AS category,
    image_path,
    description,
    tier_targets
FROM citation_mappings
WHERE enabled IS NOT FALSE
ORDER BY category, citation_name_display`

// Q35 : Citations — totaux agrégés depuis match_citations (player stats.duckdb).
// Paramètre : aucun.
const Q35CitationTotals = `
SELECT
    citation_name_norm,
    SUM(value) AS total
FROM match_citations
GROUP BY citation_name_norm
ORDER BY total DESC`

// Q36a : Commendations — total de médailles gagnées par medal_id (xuid du joueur).
// Paramètre : ?1 = xuid du joueur. Requête sur pdb.Player (shared attaché).
const Q36aMedalTotals = `
SELECT
    medal_id,
    SUM(count) AS total_count
FROM shared.medals_earned
WHERE xuid = ?
GROUP BY medal_id
ORDER BY total_count DESC`

// Q36b : Commendations — mappings médaille→citation depuis metadata.duckdb.
// Paramètre : aucun. Requête sur pdb.Metadata.
const Q36bMedalCitationMappings = `
SELECT
    medal_id,
    citation_name_display,
    COALESCE(category, 'misc')  AS category,
    image_path
FROM citation_mappings
WHERE mapping_type = 'medal'
  AND enabled IS NOT FALSE
  AND medal_id IS NOT NULL`

// Q37 : Médias — fichiers actifs paginés depuis media_files + associations.
// Paramètres : ?1 = LIMIT, ?2 = OFFSET.
const Q37MediaFiles = `
SELECT
    mf.file_path,
    mf.file_name,
    mf.kind,
    mf.thumbnail_path,
    mf.capture_end_utc,
    mma.match_id,
    mma.match_start_time
FROM media_files mf
LEFT JOIN media_match_associations mma ON mf.file_path = mma.media_path
WHERE mf.status = 'active'
ORDER BY mf.mtime DESC
LIMIT ? OFFSET ?`

// Q37Count : Médias — nombre total de fichiers actifs.
const Q37MediaCount = `SELECT COUNT(*) FROM media_files WHERE status = 'active'`
