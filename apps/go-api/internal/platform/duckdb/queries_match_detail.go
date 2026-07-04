package duckdb

const Q22aMatchSkillRankPlayer = `
SELECT
    UPPER(COALESCE(NULLIF(TRIM(msr.rating_type), ''), '')) AS rating_type_raw,
    msr.tier_label,
    msr.rating_value,
    msr.rating_delta,
    msr.playlist_group,
    msr.tier,
    msr.sub_tier,
    msr.expected_win_prob
FROM match_skill_rank_latest msr
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
           COALESCE(vg.gamertag, ('Joueur ' || RIGHT(p.xuid, 4))) AS gamertag,
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
var Q23bMatchEncounterStats = `
WITH this_match AS (
    SELECT p.xuid, p.team_id,
           COALESCE(vg.gamertag, ('Joueur ' || RIGHT(p.xuid, 4))) AS gamertag
    FROM match_participants p
    LEFT JOIN v_gamertag_lookup vg ON vg.xuid = p.xuid
    WHERE p.match_id = ?
      AND p.xuid != ?
      -- Bots exclus : pas d'historique cross-match pertinent (cf. Q23).
      -- %% car ce template est désormais consommé via fmt.Sprintf (PMT-5).
      AND p.xuid NOT LIKE 'bid(%%'
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
        ` + StartTimeCanonicalSQL("mr") + ` AS hist_start_time
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
        COUNT(DISTINCT CASE WHEN eh.is_ally_in_hist AND %s THEN eh.match_id END) AS wins_as_ally,
        COUNT(DISTINCT CASE WHEN eh.is_ally_in_hist AND %s THEN eh.match_id END) AS losses_as_ally,
        COUNT(DISTINCT CASE WHEN NOT eh.is_ally_in_hist AND %s THEN eh.match_id END) AS wins_vs_enemy,
        COUNT(DISTINCT CASE WHEN NOT eh.is_ally_in_hist AND %s THEN eh.match_id END) AS losses_vs_enemy,
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
    mf.kind,
    mf.thumbnail_path,
    mf.capture_end_utc,
    COALESCE(mf.liked, FALSE) AS liked
FROM media_files mf
JOIN media_match_associations_latest mma ON mf.id = mma.media_file_id
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
// Requête sur v_weapon_kills (append-only #23046 Phase 2 : la vue ne retourne
// que la dernière génération par (match_id,xuid) — sinon COUNT(*) fan-out).
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
const Q28BulkWeaponKills = `
SELECT
    wk.xuid,
    wk.effective_weapon_id AS weapon_id,
    COUNT(*) AS kills
FROM v_weapon_kills wk
WHERE wk.match_id = ?
  AND wk.effective_weapon_id NOT IN (0, 1, 2)
GROUP BY wk.xuid, wk.effective_weapon_id
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
