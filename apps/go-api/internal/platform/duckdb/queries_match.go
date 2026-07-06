// Package duckdb — queries_match.go : requêtes vue match (scoreboard, events, armes).
package duckdb

import "levelup/go-api/internal/analysis"

// Q10 : Career — encounters (adversaires et coéquipiers fréquents).
// Paramètre : ? = xuid du joueur.
//
// Résolveur canonique : v_gamertag_lookup gère bots + cascade
// xuid_aliases / match_participants. Fallback masqué "Joueur ####" (jamais de
// xuid brut, miroir de analysis.MaskedXuidLabelSQL) pour les xuids orphelins
// (absents de la vue) — garantit aussi gamertag NON NULL pour le scan Go.
//
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
const Q10Encounters = `
SELECT
    p2.xuid,
    COALESCE(vg.gamertag, ('Joueur ' || RIGHT(p2.xuid, 4))) AS gamertag,
    COUNT(*) AS match_count,
    SUM(CASE WHEN p2.team_id = p1.team_id THEN 1 ELSE 0 END) AS as_teammate,
    SUM(CASE WHEN p2.team_id != p1.team_id THEN 1 ELSE 0 END) AS as_enemy,
    AVG(p2.kda)                                               AS avg_kda
FROM match_participants p1
JOIN match_participants p2
    ON p1.match_id = p2.match_id AND p2.xuid != p1.xuid
LEFT JOIN v_gamertag_lookup vg ON vg.xuid = p2.xuid
WHERE p1.xuid = ?
GROUP BY p2.xuid, vg.gamertag
HAVING COUNT(*) >= 2
ORDER BY match_count DESC
LIMIT 50`

// Q12 : Match view — scoreboard complet d'un match.
// Paramètres : ?1 = match_id (medals), ?2 = match_id (weapons), ?3 = match_id (WHERE).
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
// Le token /*__PERFECT_KILL_IN__*/ est résolu au moment de l'exécution vers le
// set de médailles « frag parfait » du titre du joueur (perfectKillMedalInClause ;
// HINF byte-identique = medal_name_id IN (1512363953)).
var Q12MatchScoreboard = `
WITH me_perfect AS (
    SELECT xuid, COALESCE(SUM(count), 0) AS perfect_kills
    FROM medals_earned
    WHERE match_id = ? AND /*__PERFECT_KILL_IN__*/
    GROUP BY xuid
),
top_weapons AS (
    SELECT xuid, wid AS top_weapon_id
    FROM (
        SELECT xuid, effective_weapon_id AS wid, COUNT(*) AS wk,
               ROW_NUMBER() OVER (PARTITION BY xuid ORDER BY COUNT(*) DESC) AS rn
        FROM v_weapon_kills
        WHERE match_id = ? AND effective_weapon_id NOT IN (0, 1, 2)
        GROUP BY xuid, effective_weapon_id
    ) t WHERE rn = 1
)
SELECT
    p.xuid,
    -- Résolveur canonique : v_gamertag_lookup gère bots ('bid(N.0)' → '343 Bot N')
    -- + cascade xuid_aliases / match_participants. Fallback masqué "Joueur ####"
    -- (jamais de xuid brut, miroir analysis.MaskedXuidLabelSQL) pour les orphelins
    -- + garantit gamertag NON NULL pour le scan. Plus de CASE WHEN bot ad-hoc ici.
    COALESCE(vg.gamertag, ('Joueur ' || RIGHT(p.xuid, 4))) AS gamertag,
    (` + analysis.SQLIsBotCol("p.xuid") + `) AS is_bot,
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
    w.top_weapon_id,
    p.kills_expected,
    p.deaths_expected,
    p.kills_stddev,
    p.deaths_stddev
FROM match_participants p
LEFT JOIN v_gamertag_lookup vg ON vg.xuid = p.xuid
LEFT JOIN me_perfect m ON p.xuid = m.xuid
LEFT JOIN top_weapons w ON p.xuid = w.xuid
WHERE p.match_id = ?
  AND NOT (
    COALESCE(p.kills, 0) = 0
    AND COALESCE(p.deaths, 0) = 0
    AND COALESCE(p.assists, 0) = 0
    AND COALESCE(p.personal_score, 0) = 0
    AND (p.kills IS NOT NULL OR p.deaths IS NOT NULL
         OR p.assists IS NOT NULL OR p.personal_score IS NOT NULL)
  )
ORDER BY p.team_id ASC NULLS LAST, p.rank ASC NULLS LAST`

// Q13 : Match view — métadonnées du match.
// Paramètre : ? = match_id.
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
var Q13MatchMeta = `
SELECT
    r.match_id,
    ` + StartTimeCanonicalSQL("r") + ` AS start_time,
    r.duration_seconds,
    r.map_name,
    r.pair_name,
    r.playlist_name,
    COALESCE(r.is_firefight, FALSE) AS is_firefight,
    CASE
        WHEN COALESCE(r.is_ranked, FALSE)
            OR STRPOS(LOWER(COALESCE(r.playlist_name, '')), 'ranked') > 0
            OR STRPOS(LOWER(COALESCE(r.pair_name, '')), 'ranked') > 0
        THEN TRUE
        ELSE FALSE
    END AS is_ranked,
    r.playable_duration_seconds,
    r.map_id,
    r.game_variant_name,
    r.playlist_id,
    r.team_0_score,
    r.team_1_score,
    COALESCE(r.pair_name_fr, r.pair_name) AS pair_name_fr,
    r.pair_id,
    r.game_variant_id,
    -- T0 offset (Match Timeline T0, Phase 3) : countdown pré-match en ms.
    -- NULL si real_start_time absent → fallback runtime T0=0.
    CASE
        WHEN r.real_start_time IS NOT NULL THEN
            epoch_ms(r.real_start_time AT TIME ZONE 'UTC')
            - epoch_ms(` + StartTimeCanonicalSQL("r") + `)
    END AS t0_ms,
    r.map_version_id
FROM match_registry r
WHERE r.match_id = ?`

// Q14 : Médailles d'un joueur pour un match.
// Paramètres : ?1 = xuid, ?2 = match_id.
// Les labels sont résolus ensuite via pdb.Metadata.
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
const Q14MatchMedals = `
SELECT
    me.medal_name_id,
    me.count
FROM medals_earned me
WHERE me.xuid = ? AND me.match_id = ?
ORDER BY me.count DESC`

// Q15 : Events highlight d'un match (tous les joueurs).
// Paramètre : ?1 = match_id.
const Q15MatchEvents = `
SELECT
    he.event_type,
    he.time_ms,
    he.xuid,
    he.type_hint
FROM shared.highlight_events he
WHERE he.match_id = ?
ORDER BY he.time_ms ASC`

// Q16 : Weapon kills d'un joueur pour un match.
// Paramètres : ?1 = xuid, ?2 = match_id.
// Les labels sont résolus ensuite via pdb.Metadata.
// Utilise v_weapon_kills (effective_weapon_id = COALESCE(reconciled_as, weapon_id))
// pour appliquer la fusion d'armes (M392→Bandit Evo, Fuel Rod→M41 SPNKr, etc.).
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
const Q16WeaponKills = `
SELECT
    wk.effective_weapon_id AS weapon_id,
    COUNT(*) AS kills
FROM v_weapon_kills wk
WHERE wk.xuid = ? AND wk.match_id = ?
  AND wk.effective_weapon_id NOT IN (0, 1, 2)
GROUP BY wk.effective_weapon_id
ORDER BY kills DESC`

// Q17 : Stats d'un joueur pour un match spécifique (match_participants).
// Paramètres : ?1 = match_id, ?2 = xuid.
// Retourne 19 colonnes : outcome_code, team_id, rank_in_team, kills, deaths,
// assists, kda, accuracy, personal_score, avg_life_seconds, time_played_seconds,
// shots_fired, shots_hit, damage_dealt, damage_taken, team_mmr, enemy_mmr,
// headshot_kills, max_killing_spree.
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
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
    p.damage_taken,
    p.team_mmr,
    p.enemy_mmr,
    p.headshot_kills,
    p.max_killing_spree,
    COALESCE(p.backfill_bits, 0) AS backfill_bits
FROM match_participants p
WHERE p.match_id = ? AND p.xuid = ?`

// Q17bIsParticipant : EXISTS léger pour le gating "match non-participé" (ADR 0029,
// Couche B). Exécutée sur SharedReader. Index (match_id, xuid) → coût négligeable.
const Q17bIsParticipant = `
SELECT EXISTS(
    SELECT 1 FROM match_participants WHERE match_id = ? AND xuid = ?
)`

// Q29 : Historique récent (50 matchs) pour moyennes K/D/A + spree/headshots/perfect.
// Paramètres : ?1 = xuid (recent CTE), ?2 = xuid (perfect CTE).
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
const Q29HistoryForAvg = `
WITH recent AS (
    SELECT
        p.match_id,
        COALESCE(p.kills, 0)           AS kills,
        COALESCE(p.deaths, 0)          AS deaths,
        COALESCE(p.assists, 0)         AS assists,
        p.headshot_kills,
        p.max_killing_spree,
        COALESCE(r.pair_name, '')      AS pair_name,
        COALESCE(r.is_firefight, FALSE) AS is_firefight,
        COALESCE(r.is_ranked, FALSE)    AS is_ranked,
        COALESCE(r.duration_seconds, 0) AS duration_seconds
    FROM match_participants p
    JOIN match_registry r ON r.match_id = p.match_id
    WHERE p.xuid = ?
    ORDER BY r.start_time DESC NULLS LAST
    LIMIT 50
),
perfect AS (
    SELECT m.match_id, COALESCE(SUM(m.count), 0) AS perfect_kills
    FROM medals_earned m
    WHERE m.xuid = ?
      AND m.match_id IN (SELECT match_id FROM recent)
      AND /*__PERFECT_KILL_IN__*/
    GROUP BY m.match_id
)
SELECT
    rc.kills,
    rc.deaths,
    rc.assists,
    rc.headshot_kills,
    rc.max_killing_spree,
    COALESCE(p.perfect_kills, 0) AS perfect_kills,
    rc.pair_name,
    rc.is_firefight,
    rc.is_ranked,
    rc.duration_seconds
FROM recent rc
LEFT JOIN perfect p ON p.match_id = rc.match_id`

// Q29HistoryForAvgBulkTpl : variante MULTI-xuid de Q29 (J3). Remplace ~8
// exécutions séquentielles de GetHistoryForAvg (une par ami du scoreboard) par une
// seule requête. `%s` = placeholders de la clause IN (répétés 2x : recent + perfect).
// Sémantique identique à Q29 par xuid : 50 derniers matchs (ROW_NUMBER PARTITION BY
// xuid au lieu de LIMIT 50) + perfect kills agrégés. La colonne xuid en tête permet
// de regrouper les lignes par joueur côté Go.
const Q29HistoryForAvgBulkTpl = `
WITH recent AS (
    SELECT
        p.xuid,
        p.match_id,
        COALESCE(p.kills, 0)           AS kills,
        COALESCE(p.deaths, 0)          AS deaths,
        COALESCE(p.assists, 0)         AS assists,
        p.headshot_kills,
        p.max_killing_spree,
        COALESCE(r.pair_name, '')      AS pair_name,
        COALESCE(r.is_firefight, FALSE) AS is_firefight,
        COALESCE(r.is_ranked, FALSE)    AS is_ranked,
        COALESCE(r.duration_seconds, 0) AS duration_seconds,
        ROW_NUMBER() OVER (PARTITION BY p.xuid ORDER BY r.start_time DESC NULLS LAST) AS rn
    FROM match_participants p
    JOIN match_registry r ON r.match_id = p.match_id
    WHERE p.xuid IN (%s)
),
recent_capped AS (
    SELECT * FROM recent WHERE rn <= 50
),
perfect AS (
    SELECT m.xuid, m.match_id, COALESCE(SUM(m.count), 0) AS perfect_kills
    FROM medals_earned m
    WHERE m.xuid IN (%s)
      AND m.match_id IN (SELECT match_id FROM recent_capped)
      AND /*__PERFECT_KILL_IN__*/
    GROUP BY m.xuid, m.match_id
)
SELECT
    rc.xuid,
    rc.kills,
    rc.deaths,
    rc.assists,
    rc.headshot_kills,
    rc.max_killing_spree,
    COALESCE(p.perfect_kills, 0) AS perfect_kills,
    rc.pair_name,
    rc.is_firefight,
    rc.is_ranked,
    rc.duration_seconds
FROM recent_capped rc
LEFT JOIN perfect p ON p.match_id = rc.match_id AND p.xuid = rc.xuid
ORDER BY rc.xuid, rc.rn`

// Q18 : Enrichissement joueur pour un match (player_match_enrichment).
// Paramètre : ? = match_id.
// Retourne 3 colonnes : performance_score, is_with_friends, is_excluded.
const Q18MatchEnrichment = `
SELECT
    pme.performance_score,
    COALESCE(pme.is_with_friends, FALSE) AS is_with_friends,
    COALESCE(pme.is_excluded, FALSE)     AS is_excluded,
    COALESCE(pme.dominance_flag, 0)      AS dominance_flag
FROM player_match_enrichment_latest pme
WHERE pme.match_id = ?`

// Q19 : Matchs communs entre 2 joueurs.
// Paramètres : ?1 = xuid joueur principal, ?2 = xuid autre joueur.
// Retourne 10 colonnes : match_id, start_time, map_ui, mode_ui,
// player1_team_id, player2_team_id, player1_outcome, player1_kills, player1_deaths, player1_kda.
var Q19CommonMatches = `
SELECT
    r.match_id,
    ` + StartTimeCanonicalSQL("r") + ` AS start_time,
    COALESCE(r.map_name, '')          AS map_ui,
    COALESCE(r.pair_name, '')         AS mode_ui,
    p1.team_id                        AS player1_team_id,
    p2.team_id                        AS player2_team_id,
    COALESCE(p1.outcome, 0)           AS player1_outcome,
    COALESCE(p1.kills, 0)             AS player1_kills,
    COALESCE(p1.deaths, 0)            AS player1_deaths,
    COALESCE(p1.kda, 0.0)             AS player1_kda
FROM match_registry r
JOIN match_participants p1 ON r.match_id = p1.match_id AND p1.xuid = ?
JOIN match_participants p2 ON r.match_id = p2.match_id AND p2.xuid = ?
ORDER BY ` + StartTimeCanonicalSQL("r") + ` DESC`

// Q19cTargetRecentMatches : les `limit` derniers matchs PvP (firefight exclu)
// du joueur cible, pour les graphes "profil de combat" (Explorer mode Joueur).
// Filtre PvP en SQL via match_registry.is_firefight (même pattern que
// engagement_score_repo / career highlights — plus fiable qu'un filtre Go sur
// le pair_name). perfect_kills agrégé par match via LEFT JOIN sur medals_earned ;
// le set de médailles « frag parfait » est résolu au runtime via le token
// /*__PERFECT_KILL_IN__*/ (perfectKillMedalInClause ; HINF = {1512363953}, même
// source unique que queries_squad.go / compare_repo.go). Exécutée sur
// SharedReader (shared.* sans préfixe).
// Params (ordre) : ? = xuid (perfect CTE), ? = xuid (participants), ? = limit.
var Q19cTargetRecentMatches = `
WITH perfect AS (
    SELECT match_id, COALESCE(SUM(count), 0) AS perfect_kills
    FROM medals_earned
    WHERE xuid = ? AND /*__PERFECT_KILL_IN__*/
    GROUP BY match_id
)
SELECT
    mp.match_id,
    ` + StartTimeCanonicalSQL("r") + ` AS start_time,
    COALESCE(r.map_name, '')          AS map_ui,
    r.pair_name                       AS pair_name,
    r.pair_name_fr                    AS pair_name_fr,
    COALESCE(mp.outcome, 0)           AS outcome,
    mp.rank                           AS rank,
    COALESCE(mp.kills, 0)             AS kills,
    COALESCE(mp.deaths, 0)            AS deaths,
    COALESCE(mp.assists, 0)           AS assists,
    COALESCE(mp.kda, 0.0)             AS kda,
    COALESCE(mp.personal_score, 0)    AS score,
    COALESCE(mp.damage_dealt, 0.0)    AS damage_dealt,
    COALESCE(mp.damage_taken, 0.0)    AS damage_taken,
    COALESCE(mp.max_killing_spree, 0) AS max_killing_spree,
    COALESCE(p.perfect_kills, 0)      AS perfect_kills
FROM match_participants mp
JOIN match_registry r ON r.match_id = mp.match_id
LEFT JOIN perfect p ON p.match_id = mp.match_id
WHERE mp.xuid = ?
  AND COALESCE(r.is_firefight, FALSE) = FALSE
ORDER BY start_time DESC
LIMIT ?`

// Q19b : Kills croisés agrégés entre deux joueurs sur l'ensemble de leurs matchs communs.
// Paramètres : ?1 = xuid joueur principal, ?2 = xuid autre joueur (répétés 2 fois chacun).
// Retourne 2 colonnes : kills_dealt, deaths_suffered.
const Q19bKillerVictimBetween = `
SELECT
    COALESCE((
        SELECT SUM(kill_count) FROM killer_victim_pairs
        WHERE killer_xuid = ? AND victim_xuid = ?
    ), 0) AS kills_dealt,
    COALESCE((
        SELECT SUM(kill_count) FROM killer_victim_pairs
        WHERE killer_xuid = ? AND victim_xuid = ?
    ), 0) AS deaths_suffered`

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
FROM v_killer_victim_full kvf
WHERE kvf.match_id = ?
ORDER BY kvf.time_ms ASC`

// Q21 : Événements highlight avec xuid + gamertag résolu pour un match complet.
// Paramètre : ? = match_id.
//
// JOIN sur shared.v_gamertag_lookup : la vue gère bots (`bid(N.0)` → "343 Bot N")
// + fallback xuid raw, donc gamertag retourné est toujours non vide quand le
// xuid est présent en DB. Pour un xuid orphelin (jamais vu en match_participants
// ni xuid_aliases), vg.gamertag est NULL → caller fallback sur xuid brut.
const Q21MatchEventsWithXUID = `
SELECT
    he.event_type,
    he.time_ms,
    he.xuid,
    vg.gamertag AS gamertag
FROM highlight_events he
LEFT JOIN v_gamertag_lookup vg ON vg.xuid = he.xuid
WHERE he.match_id = ?
ORDER BY he.time_ms ASC NULLS LAST`

// Q25 : Navigation prev/next entre matchs adjacents d'un joueur (chronologie globale).
// Paramètres : ?1 = xuid, ?2 = match_id, ?3 = xuid (réutilisé pour la CTE).
// Ordre : start_time DESC (plus récent = index 0).
//
// Executée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
var Q25NeighborMatches = `
WITH ordered AS (
    SELECT
        mr.match_id,
        ` + StartTimeCanonicalSQL("mr") + ` AS start_time,
        ROW_NUMBER() OVER (
            ORDER BY ` + StartTimeCanonicalSQL("mr") + ` DESC
        ) - 1 AS idx,
        COUNT(*) OVER () AS total
    FROM match_registry mr
    JOIN match_participants mp
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

// Q25NeighborMatchesTemplate : version paramétrable de Q25 pour Phase 2b
// (filtres MatchFilterSpec). Le marqueur `/*EXTRA_WHERE*/` est remplacé par
// le fragment SQL produit par analysis.BuildNeighborsWhereClause (vide ou
// commençant par ' AND ').
//
// Paramètres positionnels : xuid, [filtres...], match_id.
// L'ordre est important — le repo concatène les args dans cet ordre.
// Executée sur SharedReader (ADR 0016).
var Q25NeighborMatchesTemplate = `
WITH ordered AS (
    SELECT
        mr.match_id,
        ` + StartTimeCanonicalSQL("mr") + ` AS start_time,
        ROW_NUMBER() OVER (
            ORDER BY ` + StartTimeCanonicalSQL("mr") + ` DESC
        ) - 1 AS idx,
        COUNT(*) OVER () AS total
    FROM match_registry mr
    JOIN match_participants mp
        ON mr.match_id = mp.match_id AND mp.xuid = ?
    WHERE TRUE /*EXTRA_WHERE*/
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

// Q22a : Rang compétitif du joueur pour ce match (player DB seule).
// Paramètre : ? = match_id.
//
// Cross-DB split (ADR 0016) : on lit d'abord les colonnes match_skill_rank
// sur la conn Player (Q22a), puis on lit séparément match_registry sur
// SharedReader (Q22b) pour calculer le rating_type effectif côté Go.
