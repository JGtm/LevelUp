// Package duckdb — queries_match.go : requêtes vue match (scoreboard, events, armes).
package duckdb

// Q10 : Career — encounters (adversaires et coéquipiers fréquents).
// Paramètre : ? = xuid du joueur.
//
// Résolveur canonique : v_gamertag_lookup gère bots + cascade
// xuid_aliases / match_participants / fallback xuid raw. Caller fait juste
// `COALESCE(vg.gamertag, p2.xuid)` pour couvrir les xuids orphelins (jamais en DB).
//
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
const Q10Encounters = `
SELECT
    p2.xuid,
    COALESCE(vg.gamertag, p2.xuid) AS gamertag,
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
const Q12MatchScoreboard = `
WITH me_perfect AS (
    SELECT xuid, COALESCE(SUM(count), 0) AS perfect_kills
    FROM medals_earned
    WHERE match_id = ? AND medal_name_id = 1512363953
    GROUP BY xuid
),
top_weapons AS (
    SELECT xuid, wid AS top_weapon_id
    FROM (
        SELECT xuid, COALESCE(reconciled_as, weapon_id) AS wid, COUNT(*) AS wk,
               ROW_NUMBER() OVER (PARTITION BY xuid ORDER BY COUNT(*) DESC) AS rn
        FROM weapon_kills
        WHERE match_id = ? AND COALESCE(reconciled_as, weapon_id) NOT IN (0, 1, 2)
        GROUP BY xuid, COALESCE(reconciled_as, weapon_id)
    ) t WHERE rn = 1
)
SELECT
    p.xuid,
    -- Résolveur canonique : v_gamertag_lookup gère bots ('bid(N.0)' → '343 Bot N')
    -- + cascade xuid_aliases / match_participants. Caller fait juste fallback
    -- xuid raw pour les xuids orphelins. Plus de CASE WHEN bot ad-hoc ici.
    COALESCE(vg.gamertag, p.xuid) AS gamertag,
    (p.xuid LIKE 'bid(%') AS is_bot,
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
const Q13MatchMeta = `
SELECT
    r.match_id,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
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
            - epoch_ms(COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC'))
    END AS t0_ms
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
        COALESCE(r.is_ranked, FALSE)    AS is_ranked
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
      AND m.medal_name_id = 1512363953
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
    rc.is_ranked
FROM recent rc
LEFT JOIN perfect p ON p.match_id = rc.match_id`

// Q18 : Enrichissement joueur pour un match (player_match_enrichment).
// Paramètre : ? = match_id.
// Retourne 3 colonnes : performance_score, is_with_friends, is_excluded.
const Q18MatchEnrichment = `
SELECT
    pme.performance_score,
    COALESCE(pme.is_with_friends, FALSE) AS is_with_friends,
    COALESCE(pme.is_excluded, FALSE)     AS is_excluded,
    COALESCE(pme.dominance_flag, 0)      AS dominance_flag
FROM player_match_enrichment pme
WHERE pme.match_id = ?`

// Q19 : Matchs communs entre 2 joueurs.
// Paramètres : ?1 = xuid joueur principal, ?2 = xuid autre joueur.
// Retourne 10 colonnes : match_id, start_time, map_ui, mode_ui,
// player1_team_id, player2_team_id, player1_outcome, player1_kills, player1_deaths, player1_kda.
const Q19CommonMatches = `
SELECT
    r.match_id,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
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
ORDER BY COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') DESC`

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
const Q25NeighborMatches = `
WITH ordered AS (
    SELECT
        mr.match_id,
        COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC') AS start_time,
        ROW_NUMBER() OVER (
            ORDER BY COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC') DESC
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
const Q25NeighborMatchesTemplate = `
WITH ordered AS (
    SELECT
        mr.match_id,
        COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC') AS start_time,
        ROW_NUMBER() OVER (
            ORDER BY COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC') DESC
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
const Q22aMatchSkillRankPlayer = `
SELECT
    UPPER(COALESCE(NULLIF(TRIM(msr.rating_type), ''), '')) AS rating_type_raw,
    msr.tier_label,
    msr.rating_value,
    msr.rating_delta,
    msr.playlist_group,
    msr.tier,
    msr.sub_tier
FROM match_skill_rank msr
WHERE msr.match_id = ?
LIMIT 1`

// Q22b : Métadonnées registry minimales pour déduire rating_type (CSR/LUSR).
// Paramètre : ? = match_id. Exécutée sur SharedReader (ADR 0016).
const Q22bMatchRegistryRankedFlag = `
SELECT
    COALESCE(is_ranked, FALSE) AS is_ranked,
    COALESCE(playlist_name, '') AS playlist_name,
    COALESCE(pair_name, '')     AS pair_name
FROM match_registry
WHERE match_id = ?
LIMIT 1`

// Q23 : Rencontres historiques avec chaque participant de ce match.
// Paramètres : ?1 = match_id, ?2 = myXUID (this_match excl.), ?3 = match_id,
//
//	?4 = myXUID (my_team), ?5 = myXUID (me.xuid=?).
//
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
const Q23MatchEncounters = `
WITH this_match AS (
    SELECT p.xuid, p.team_id,
           COALESCE(vg.gamertag, p.xuid) AS gamertag,
           FALSE AS is_bot
    FROM match_participants p
    LEFT JOIN v_gamertag_lookup vg ON vg.xuid = p.xuid
    WHERE p.match_id = ?
      AND p.xuid != ?
      -- Bots exclus : leur xuid 'bid(N.0)' est unique par match → aucun
      -- "historique de rencontres" pertinent à afficher.
      AND p.xuid NOT LIKE 'bid(%'
),
my_team AS (
    SELECT team_id FROM match_participants
    WHERE match_id = ? AND xuid = ?
    LIMIT 1
)
SELECT
    tm.xuid,
    tm.gamertag,
    tm.is_bot,
    COUNT(DISTINCT hist.match_id) AS count_together,
    (tm.team_id = (SELECT team_id FROM my_team)) AS is_ally
FROM this_match tm
LEFT JOIN match_participants me ON me.xuid = ?
LEFT JOIN match_participants hist
    ON hist.match_id = me.match_id AND hist.xuid = tm.xuid
GROUP BY tm.xuid, tm.gamertag, tm.is_bot, tm.team_id
ORDER BY count_together DESC`

// Q23bMatchEncounterStats : stats riches par encounter (chunk MV4.C').
// Permet d'attribuer les badges narratifs ally_plus + tough_enemy via
// narrative.ComputeEncounterBadges.
//
// Paramètres (8 placeholders, ordre strict) :
//
//	?1 = match_id (this_match)
//	?2 = myXUID  (this_match exclude)
//	?3 = match_id (my_team)
//	?4 = myXUID  (my_team)
//	?5 = myXUID  (my_history me)
//	?6 = myXUID  (kv kills_dealt)
//	?7 = myXUID  (kv deaths_suffered)
//	?8 = myXUID  (kv join condition)
//
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
const Q23bMatchEncounterStats = `
WITH this_match AS (
    SELECT p.xuid, p.team_id,
           COALESCE(vg.gamertag, p.gamertag, p.xuid) AS gamertag
    FROM match_participants p
    LEFT JOIN v_gamertag_lookup vg ON vg.xuid = p.xuid
    WHERE p.match_id = ?
      AND p.xuid != ?
      -- Bots exclus : pas d'historique cross-match pertinent (cf. Q23).
      AND p.xuid NOT LIKE 'bid(%'
),
my_team AS (
    SELECT team_id FROM match_participants
    WHERE match_id = ? AND xuid = ?
    LIMIT 1
),
my_history AS (
    SELECT match_id, team_id, outcome
    FROM match_participants
    WHERE xuid = ?
),
encounter_history AS (
    SELECT
        tm.xuid,
        h.match_id,
        h.outcome AS me_outcome,
        (h.team_id = hist.team_id) AS is_ally_in_hist,
        COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC') AS hist_start_time
    FROM this_match tm
    JOIN my_history h ON 1=1
    JOIN match_participants hist
        ON hist.match_id = h.match_id AND hist.xuid = tm.xuid
    LEFT JOIN match_registry mr ON mr.match_id = h.match_id
),
encounter_stats AS (
    SELECT
        eh.xuid,
        COUNT(DISTINCT CASE WHEN eh.is_ally_in_hist THEN eh.match_id END) AS ally_count,
        COUNT(DISTINCT CASE WHEN NOT eh.is_ally_in_hist THEN eh.match_id END) AS enemy_count,
        COUNT(DISTINCT CASE WHEN eh.is_ally_in_hist AND eh.me_outcome = 2 THEN eh.match_id END) AS wins_as_ally,
        COUNT(DISTINCT CASE WHEN eh.is_ally_in_hist AND eh.me_outcome = 3 THEN eh.match_id END) AS losses_as_ally,
        COUNT(DISTINCT CASE WHEN NOT eh.is_ally_in_hist AND eh.me_outcome = 2 THEN eh.match_id END) AS wins_vs_enemy,
        COUNT(DISTINCT CASE WHEN NOT eh.is_ally_in_hist AND eh.me_outcome = 3 THEN eh.match_id END) AS losses_vs_enemy,
        MAX(eh.hist_start_time) AS last_seen_at
    FROM encounter_history eh
    GROUP BY eh.xuid
),
kv_stats AS (
    SELECT
        tm.xuid,
        SUM(CASE WHEN kv.killer_xuid = ? AND kv.victim_xuid = tm.xuid THEN kv.kill_count ELSE 0 END) AS kills_dealt,
        SUM(CASE WHEN kv.killer_xuid = tm.xuid AND kv.victim_xuid = ? THEN kv.kill_count ELSE 0 END) AS deaths_suffered
    FROM this_match tm
    LEFT JOIN killer_victim_pairs kv
        ON ((kv.killer_xuid = ? AND kv.victim_xuid = tm.xuid)
            OR (kv.killer_xuid = tm.xuid AND kv.victim_xuid = ?))
    GROUP BY tm.xuid
)
SELECT
    tm.xuid,
    COALESCE(es.ally_count, 0) AS ally_count,
    COALESCE(es.enemy_count, 0) AS enemy_count,
    COALESCE(es.wins_as_ally, 0) AS wins_as_ally,
    COALESCE(es.losses_as_ally, 0) AS losses_as_ally,
    COALESCE(es.wins_vs_enemy, 0) AS wins_vs_enemy,
    COALESCE(es.losses_vs_enemy, 0) AS losses_vs_enemy,
    COALESCE(kv.kills_dealt, 0) AS kills_dealt,
    COALESCE(kv.deaths_suffered, 0) AS deaths_suffered,
    es.last_seen_at
FROM this_match tm
LEFT JOIN encounter_stats es ON es.xuid = tm.xuid
LEFT JOIN kv_stats kv ON kv.xuid = tm.xuid
ORDER BY tm.xuid`

// Q24 : Médias associés à un match — tous auteurs confondus (shared_social DB).
// Le feed média est cross-joueur : si un coéquipier a uploadé un clip pour
// ce match, on l'affiche aussi sur la page match.
// Paramètre : ? = match_id.
//
// Note : pas de filtre sur mf.status — aligné sur le baseWhereClause de la
// galerie en mode shared_social (cf. queries_home_citations.go) qui n'en
// applique pas non plus. Les médias existants n'ont pas de status non-NULL
// par défaut (ALTER ADD COLUMN sans DEFAULT).
const Q24MatchMedia = `
SELECT
    mf.id               AS file_id,
    mf.file_name,
    mf.file_path,
    mf.thumbnail_path,
    mf.capture_end_utc,
    COALESCE(mf.liked, FALSE) AS liked
FROM media_files mf
JOIN media_match_associations mma ON mf.id = mma.media_file_id
WHERE mma.match_id = ?
ORDER BY mf.capture_end_utc ASC NULLS LAST`

// Q27 : Médailles de tous les joueurs d'un match (bulk).
// Paramètre : ? = match_id.
// Retourne 3 colonnes : xuid, medal_name_id, count.
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
const Q27BulkMedals = `
SELECT
    xuid,
    medal_name_id,
    SUM(count) AS count
FROM medals_earned
WHERE match_id = ?
GROUP BY xuid, medal_name_id
ORDER BY xuid, count DESC`

// Q28 : Kills par arme de tous les joueurs d'un match (bulk).
// Paramètre : ? = match_id.
// Retourne 3 colonnes : xuid, weapon_id, kills.
// Requête directe sur weapon_kills (sans passer par v_weapon_kills).
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
const Q28BulkWeaponKills = `
SELECT
    wk.xuid,
    COALESCE(wk.reconciled_as, wk.weapon_id) AS weapon_id,
    COUNT(*) AS kills
FROM weapon_kills wk
WHERE wk.match_id = ?
  AND COALESCE(wk.reconciled_as, wk.weapon_id) NOT IN (0, 1, 2)
GROUP BY wk.xuid, COALESCE(wk.reconciled_as, wk.weapon_id)
ORDER BY wk.xuid, kills DESC`

// Q30 : CSR de tous les participants d'un match ranked depuis shared.match_csrs_latest.
// Paramètre : ?1 = match_id.
// Exécutée sur SharedReader — match_csrs est dans la shared DB.
const Q30SharedMatchCSRs = `
SELECT
    xuid,
    rating_type,
    tier_label,
    rating_value,
    rating_delta,
    tier,
    sub_tier
FROM match_csrs_latest
WHERE match_id = ?`

// Q26 : Stats attendues du joueur pour ce match (match_participants expected columns).
// Paramètres : ?1 = match_id, ?2 = xuid.
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
const Q26MatchExpectedStats = `
SELECT
    p.kills_expected,
    p.deaths_expected,
    p.kills_stddev,
    p.deaths_stddev
FROM match_participants p
WHERE p.match_id = ? AND p.xuid = ?`
