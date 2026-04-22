// Package duckdb — queries_match.go : requêtes vue match (scoreboard, events, armes).
package duckdb

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
FROM match_participants p1
JOIN match_participants p2
    ON p1.match_id = p2.match_id AND p2.xuid != p1.xuid
LEFT JOIN xuid_aliases xa ON p2.xuid = xa.xuid
WHERE p1.xuid = ?
GROUP BY p2.xuid, COALESCE(xa.gamertag, p2.xuid)
HAVING COUNT(*) >= 2
ORDER BY match_count DESC
LIMIT 50`

// Q12 : Match view — scoreboard complet d'un match.
// Paramètres : ?1 = match_id (medals), ?2 = match_id (weapons), ?3 = match_id (WHERE).
const Q12MatchScoreboard = `
WITH me_perfect AS (
    SELECT xuid, COALESCE(SUM(count), 0) AS perfect_kills
    FROM shared.medals_earned
    WHERE match_id = ? AND medal_name_id = 1512363953
    GROUP BY xuid
),
top_weapons AS (
    SELECT xuid, effective_weapon_id AS top_weapon_id
    FROM (
        SELECT xuid, effective_weapon_id, SUM(kill_count) AS wk,
               ROW_NUMBER() OVER (PARTITION BY xuid ORDER BY SUM(kill_count) DESC) AS rn
        FROM shared.v_weapon_kills
        WHERE match_id = ? AND effective_weapon_id NOT IN (0, 1, 2)
        GROUP BY xuid, effective_weapon_id
    ) t WHERE rn = 1
)
SELECT
    p.xuid,
    COALESCE(xa.gamertag, p.xuid)  AS gamertag,
    p.team_id,
    p.rank              AS rank_in_team,
    COALESCE(p.outcome, 0)         AS outcome,
    p.personal_score,
    COALESCE(p.kills, 0)           AS kills,
    COALESCE(p.deaths, 0)          AS deaths,
    COALESCE(p.assists, 0)         AS assists,
    p.kda,
    p.accuracy,
    p.time_played_seconds,
    p.team_mmr,
    p.enemy_mmr,
    p.shots_fired,
    p.shots_hit,
    p.damage_dealt,
    p.damage_taken,
    p.avg_life_seconds,
    p.headshot_kills,
    p.max_killing_spree,
    p.grenade_kills,
    p.melee_kills,
    p.power_weapon_kills,
    COALESCE(m.perfect_kills, 0)   AS perfect_kills,
    w.top_weapon_id
FROM shared.match_participants p
LEFT JOIN shared.xuid_aliases xa ON p.xuid = xa.xuid
LEFT JOIN me_perfect m ON p.xuid = m.xuid
LEFT JOIN top_weapons w ON p.xuid = w.xuid
WHERE p.match_id = ?
ORDER BY p.team_id, p.rank NULLS LAST`

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
    COALESCE(r.is_ranked, FALSE)    AS is_ranked,
    r.playable_duration_seconds,
    r.map_id,
    r.game_variant_name
FROM shared.match_registry r
WHERE r.match_id = ?`

// Q14 : Médailles d'un joueur pour un match.
// Paramètres : ?1 = xuid, ?2 = match_id.
// Les labels sont résolus ensuite via pdb.Metadata.
const Q14MatchMedals = `
SELECT
    me.medal_name_id,
    me.count
FROM shared.medals_earned me
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
// Les labels sont résolus ensuite via pdb.Metadata.
const Q16WeaponKills = `
SELECT
    wk.weapon_id,
    SUM(wk.kill_count) AS kills
FROM shared.weapon_kills wk
WHERE wk.xuid = ? AND wk.match_id = ?
GROUP BY wk.weapon_id
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
    p.rank                         AS rank_in_team,
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
// Retourne 3 colonnes : performance_score, is_with_friends, is_excluded.
const Q18MatchEnrichment = `
SELECT
    pme.performance_score,
    COALESCE(pme.is_with_friends, FALSE) AS is_with_friends,
    COALESCE(pme.is_excluded, FALSE)     AS is_excluded
FROM player_match_enrichment pme
WHERE pme.match_id = ?`

// Q19 : Matchs communs entre 2 joueurs.
// Paramètres : ?1 = xuid joueur principal, ?2 = xuid autre joueur.
// Retourne 10 colonnes : match_id, start_time, map_ui, mode_ui,
// player1_team_id, player2_team_id, player1_outcome, player1_kills, player1_deaths, player1_kda.
const Q19CommonMatches = `
SELECT
    r.match_id,
    r.start_time,
    COALESCE(r.map_name, '')          AS map_ui,
    COALESCE(r.pair_name, '')         AS mode_ui,
    p1.team_id                        AS player1_team_id,
    p2.team_id                        AS player2_team_id,
    COALESCE(p1.outcome, 0)           AS player1_outcome,
    COALESCE(p1.kills, 0)             AS player1_kills,
    COALESCE(p1.deaths, 0)            AS player1_deaths,
    COALESCE(p1.kda, 0.0)             AS player1_kda
FROM shared.match_registry r
JOIN shared.match_participants p1 ON r.match_id = p1.match_id AND p1.xuid = ?
JOIN shared.match_participants p2 ON r.match_id = p2.match_id AND p2.xuid = ?
ORDER BY r.start_time DESC
LIMIT 100`

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
// Utilise time_ms (colonne réelle) — pas tick_count.
const Q21MatchEventsWithXUID = `
SELECT
    he.event_type,
    he.time_ms,
    he.xuid
FROM shared.highlight_events he
WHERE he.match_id = ?
ORDER BY he.time_ms ASC NULLS LAST`

// Q25 : Navigation prev/next entre matchs adjacents d'un joueur (chronologie globale).
// Paramètres : ?1 = xuid, ?2 = match_id, ?3 = xuid (réutilisé pour la CTE).
// Ordre : start_time DESC (plus récent = index 0).
const Q25NeighborMatches = `
WITH ordered AS (
    SELECT
        mr.match_id,
        mr.start_time,
        ROW_NUMBER() OVER (ORDER BY mr.start_time DESC) - 1 AS idx,
        COUNT(*) OVER () AS total
    FROM shared.match_registry mr
    JOIN shared.match_participants mp
        ON mr.match_id = mp.match_id AND mp.xuid = ?
),
current AS (
    SELECT idx, total FROM ordered WHERE match_id = ?
)
SELECT
    (SELECT match_id FROM ordered WHERE idx = c.idx - 1) AS next_match_id,
    (SELECT match_id FROM ordered WHERE idx = c.idx + 1) AS prev_match_id,
    c.idx    AS current_index,
    c.total  AS total_matches
FROM current c
LIMIT 1`
