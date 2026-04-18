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
// Paramètre : ? = match_id.
const Q12MatchScoreboard = `
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
    p.power_weapon_kills
FROM shared.match_participants p
LEFT JOIN shared.xuid_aliases xa ON p.xuid = xa.xuid
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
    COALESCE(r.is_ranked, FALSE)    AS is_ranked
FROM shared.match_registry r
WHERE r.match_id = ?`

// Q14 : Médailles d'un joueur pour un match.
// Paramètres : ?1 = xuid, ?2 = match_id.
const Q14MatchMedals = `
SELECT
    me.medal_name_id,
    me.count,
    COALESCE(cm.citation_name_display, CAST(me.medal_name_id AS VARCHAR)) AS label
FROM shared.medals_earned me
LEFT JOIN meta.citation_mappings cm ON me.medal_name_id = cm.medal_id
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
LEFT JOIN meta.weapon_labels wl ON wk.weapon_id = wl.weapon_id
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
