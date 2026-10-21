package duckdb

// resumes_parite_anciens_test.go — LES ANCIENNES REQUÊTES (transitoire, supprimé avec la parité).

// Q21dAssistPairs : texte de match_view_repo_assist_pairs.go au commit 2fed26886 (avant la bascule).
const ancien_Q21dAssistPairs = `
WITH scope AS (
    SELECT
        COUNT(*)                                             AS match_deaths,
        COUNT(*) FILTER (WHERE publishable AND assist_known)  AS measured_deaths,
        COUNT(*) FILTER (WHERE publishable)                   AS publishable_deaths
    FROM ` + KillEventsCanonicalTable + `
    WHERE match_id = ?
),
pairs AS (
    SELECT
        kv.assist_xuid,
        kv.assist_gamertag,
        kv.feed_killer_xuid,
        kv.feed_killer_gamertag,
        COUNT(*)                                                             AS assist_count,
        COUNT(*) FILTER (WHERE kv.assist_damage_pct > kv.killer_damage_pct)  AS stolen_count,
        CAST(ROUND(AVG(kv.assist_damage_pct)) AS INTEGER)                    AS avg_assist_pct
    FROM ` + KillEventsCanonicalTable + ` kv
    WHERE kv.match_id = ?
      AND kv.publishable
      AND kv.assist_known
      AND (kv.assist_xuid IS NOT NULL OR kv.assist_gamertag IS NOT NULL)
      AND (kv.feed_killer_xuid IS NOT NULL OR kv.feed_killer_gamertag IS NOT NULL)
    GROUP BY kv.assist_xuid, kv.assist_gamertag, kv.feed_killer_xuid, kv.feed_killer_gamertag
)
SELECT
    s.match_deaths,
    s.measured_deaths,
    s.publishable_deaths,
    p.assist_xuid,
    p.assist_gamertag,
    p.feed_killer_xuid,
    p.feed_killer_gamertag,
    p.assist_count,
    p.stolen_count,
    p.avg_assist_pct
FROM scope s
LEFT JOIN pairs p ON TRUE
ORDER BY p.assist_count DESC, p.assist_gamertag, p.feed_killer_xuid, p.feed_killer_gamertag`

// Q32dSquadAssistPairsTemplate : texte de queries_squad.go au commit 2fed26886 (avant la bascule).
const ancien_Q32dSquadAssistPairsTemplate = `
WITH scope AS (
    SELECT COUNT(DISTINCT match_id) FILTER (
        WHERE publishable AND assist_known
    ) AS matches_measured
    FROM ` + KillEventsCanonicalTable + `
    WHERE match_id IN (%s)
),
pairs AS (
    SELECT
        assist_xuid,
        feed_killer_xuid,
        COUNT(*)                                                       AS assist_count,
        COUNT(*) FILTER (WHERE assist_damage_pct > killer_damage_pct)  AS stolen_count
    FROM ` + KillEventsCanonicalTable + `
    WHERE match_id IN (%s)
      AND publishable
      AND assist_known
      AND assist_xuid      IN (%s)
      AND feed_killer_xuid IN (%s)
    GROUP BY assist_xuid, feed_killer_xuid
)
SELECT
    s.matches_measured,
    p.assist_xuid,
    p.feed_killer_xuid,
    p.assist_count,
    p.stolen_count
FROM scope s
LEFT JOIN pairs p ON TRUE
ORDER BY p.assist_count DESC, p.assist_xuid, p.feed_killer_xuid`

// Q28cRelationAssistsTpl : texte de relation_assists_repo.go au commit 2fed26886 (avant la bascule).
const ancien_Q28cRelationAssistsTpl = `
WITH me AS (
    SELECT CAST(? AS VARCHAR) AS xuid
),
lues AS MATERIALIZED (
    SELECT kv.match_id,
           kv.feed_killer_xuid  AS killer,
           kv.assist_xuid       AS assister,
           kv.assist_damage_pct AS pct
    FROM ` + KillEventsCanonicalTable + ` kv
    WHERE kv.publishable AND kv.assist_known AND %s
),
porteuses AS (
    SELECT DISTINCT match_id FROM lues
),
mates AS (
    SELECT mp.match_id,
           p.xuid                     AS partner,
           MAX(COALESCE(mp.kills, 0)) AS my_kills,
           MAX(COALESCE(p.kills, 0))  AS partner_kills
    FROM me
    JOIN match_participants mp ON mp.xuid = me.xuid
    JOIN match_participants p
        ON p.match_id = mp.match_id
       AND p.team_id  = mp.team_id
       AND p.xuid    <> mp.xuid
    WHERE %s
      AND mp.match_id IN (SELECT match_id FROM porteuses)%s%s
    GROUP BY mp.match_id, p.xuid
),
bases AS (
    SELECT partner,
           CAST(SUM(my_kills) AS BIGINT)      AS my_frags,
           CAST(SUM(partner_kills) AS BIGINT) AS partner_frags
    FROM mates
    GROUP BY partner
),
echanges AS (
    SELECT
        m.partner,
        COUNT(*) FILTER (WHERE k.killer = me.xuid AND k.assister = m.partner)  AS received_total,
        COUNT(*) FILTER (WHERE k.killer = me.xuid AND k.assister = m.partner AND k.pct < %d)                AS received_low,
        COUNT(*) FILTER (WHERE k.killer = me.xuid AND k.assister = m.partner AND k.pct >= %d AND k.pct <= %d) AS received_mid,
        COUNT(*) FILTER (WHERE k.killer = me.xuid AND k.assister = m.partner AND k.pct > %d)                AS received_high,
        COUNT(*) FILTER (WHERE k.killer = m.partner AND k.assister = me.xuid)  AS given_total,
        COUNT(*) FILTER (WHERE k.killer = m.partner AND k.assister = me.xuid AND k.pct < %d)                AS given_low,
        COUNT(*) FILTER (WHERE k.killer = m.partner AND k.assister = me.xuid AND k.pct >= %d AND k.pct <= %d) AS given_mid,
        COUNT(*) FILTER (WHERE k.killer = m.partner AND k.assister = me.xuid AND k.pct > %d)                AS given_high
    FROM mates m
    CROSS JOIN me
    JOIN lues k
        ON k.match_id = m.match_id
    GROUP BY m.partner
)
SELECT
    b.partner,
    b.my_frags,
    b.partner_frags,
    COALESCE(e.received_total, 0), COALESCE(e.received_low, 0), COALESCE(e.received_mid, 0), COALESCE(e.received_high, 0),
    COALESCE(e.given_total, 0),    COALESCE(e.given_low, 0),    COALESCE(e.given_mid, 0),    COALESCE(e.given_high, 0)
FROM bases b
LEFT JOIN echanges e ON e.partner = b.partner`

// Q26lHomeAssistedFragsTpl : texte de home_repo_assisted_frags.go au commit 2fed26886 (avant la bascule).
const ancien_Q26lHomeAssistedFragsTpl = `
WITH porteuses AS (
    SELECT DISTINCT match_id
    FROM ` + KillEventsCanonicalTable + `
    WHERE publishable AND assist_known AND %s
)
SELECT kv.match_id,
       COUNT(*)                                                                                     AS frags_film,
       COUNT(*) FILTER (WHERE %s)                                                                   AS received_total,
       COUNT(*) FILTER (WHERE %s AND kv.assist_damage_pct < %d)                                     AS received_low,
       COUNT(*) FILTER (WHERE %s AND kv.assist_damage_pct >= %d AND kv.assist_damage_pct <= %d)     AS received_mid,
       COUNT(*) FILTER (WHERE %s AND kv.assist_damage_pct > %d)                                     AS received_high
FROM ` + KillEventsCanonicalTable + ` kv
WHERE kv.publishable
  AND %s
  AND kv.match_id IN (SELECT match_id FROM porteuses)
  AND kv.feed_killer_xuid = ?
GROUP BY kv.match_id`

// qSoloJournalNonPubliable : texte de solo_lives_repo.go au commit 2fed26886 (avant la bascule).
const ancien_qSoloJournalNonPubliable = `
SELECT DISTINCT e.match_id
FROM match_kill_events_latest e
WHERE %s AND NOT e.publishable`

// matchRangeKillsTotalSQL : texte de match_range_repo.go au commit 2fed26886 (avant la bascule).
const ancien_matchRangeKillsTotalSQL = `
SELECT count(*)
FROM ` + KillEventsCanonicalTable + `
WHERE match_id IN (%s)
  AND publishable
  AND feed_killer_xuid IS NOT NULL`
