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
    ms.pair_id,
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
           r.map_name_fr, r.pair_name, r.pair_name_fr, r.pair_id,
           r.playlist_name, r.playlist_name_fr,
           COALESCE(r.is_firefight, FALSE) AS is_firefight,
           COALESCE(r.is_ranked, FALSE)    AS is_ranked
    FROM shared.v_match_full r
    JOIN shared.match_participants p ON r.match_id = p.match_id
    WHERE p.xuid = ?
) ms
LEFT JOIN player_match_enrichment_latest pme ON ms.match_id = pme.match_id
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
    ms.pair_id,
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
           pair_name, pair_name_fr, pair_id,
           playlist_name, playlist_name_fr,
           is_firefight, is_ranked
    FROM shared.mv_player_matches
    WHERE xuid = ?
) ms
LEFT JOIN player_match_enrichment_latest pme ON ms.match_id = pme.match_id
ORDER BY ms.start_time DESC`

// Q4Shared : (ADR 0016) — partie shared du split LoadMatchesForFilters.
// Cross-DB JOIN historique (v_match_full ⨝ match_participants ⨝
// player_match_enrichment) découpé pour passer par SharedReader sans toucher
// le pool ATTACH.
//
// Cette query est lancée via pdb.SharedReadDB().Get(ctx) — toutes les tables
// référencées sont dans shared_matches_v2.duckdb au niveau root (pas de
// préfixe `shared.` car la conn cible directement le catalogue de la DB).
// Le merge avec player_match_enrichment se fait en Go (Q4PlayerEnrichmentForMatches).
//
// Paramètre : ? = xuid.
const Q4SharedMatchesForFilters = `
SELECT
    r.match_id,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
    r.map_name,
    COALESCE(r.map_name_fr, r.map_name)                AS map_name_fr,
    r.pair_name,
    COALESCE(r.pair_name_fr, r.pair_name)              AS pair_name_fr,
    r.pair_id,
    COALESCE(r.playlist_name_fr, r.playlist_name)      AS playlist_name,
    COALESCE(r.is_firefight, FALSE)                    AS is_firefight,
    COALESCE(r.is_ranked, FALSE)                       AS is_ranked,
    r.playlist_name                                    AS playlist_name_en
FROM v_match_full r
JOIN match_participants p ON r.match_id = p.match_id
WHERE p.xuid = ?
ORDER BY start_time DESC`

// Q4MVSharedMatchesForFilters : variante du split avec mv_player_matches.
// Paramètre : ? = xuid.
const Q4MVSharedMatchesForFilters = `
SELECT
    match_id,
    COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC') AS start_time,
    map_name,
    COALESCE(map_name_fr, map_name)                AS map_name_fr,
    pair_name,
    COALESCE(pair_name_fr, pair_name)              AS pair_name_fr,
    pair_id,
    COALESCE(playlist_name_fr, playlist_name)      AS playlist_name,
    COALESCE(is_firefight, FALSE)                  AS is_firefight,
    COALESCE(is_ranked, FALSE)                     AS is_ranked,
    playlist_name                                  AS playlist_name_en
FROM mv_player_matches
WHERE xuid = ?
ORDER BY start_time DESC`

// Q5SharedHistory : (ADR 0016) — partie shared du split LoadAll
// (match_history_repo). Cross-DB JOIN Q5MatchHistory découpé : la partie
// shared retourne 25 colonnes (match metadata + participant stats + team_id
// pour le calcul my/enemy_team_score côté Go).
//
// Tables référencées au niveau root (pas de préfixe `shared.`) — la conn
// SharedReader cible directement le catalogue de shared_matches_v2.duckdb.
//
// Paramètre : ? = xuid (utilisé pour le filtre p.xuid).
const Q5SharedHistory = `
SELECT
    r.match_id,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
    r.map_name,
    COALESCE(r.map_name_fr, r.map_name)                AS map_name_fr,
    r.pair_name,
    COALESCE(r.pair_name_fr, r.pair_name)              AS pair_name_fr,
    r.playlist_name                                    AS playlist_name_en,
    COALESCE(r.playlist_name_fr, r.playlist_name)      AS playlist_name,
    r.map_id,
    r.pair_id,
    r.playlist_id,
    r.season_id,
    COALESCE(r.is_firefight, FALSE)                    AS is_firefight,
    COALESCE(r.is_ranked, FALSE)                       AS is_ranked,
    COALESCE(p.outcome, 0)                             AS outcome,
    p.team_mmr,
    p.enemy_mmr,
    COALESCE(p.kills, 0)                               AS kills,
    COALESCE(p.deaths, 0)                              AS deaths,
    COALESCE(p.assists, 0)                             AS assists,
    p.kda,
    p.accuracy,
    p.personal_score,
    p.avg_life_seconds                                 AS average_life_seconds,
    p.time_played_seconds,
    p.team_id,
    r.team_0_score,
    r.team_1_score
FROM v_match_full r
JOIN match_participants p ON r.match_id = p.match_id
WHERE p.xuid = ?
ORDER BY start_time DESC`

// Q5PlayerSkillRankHistoryTpl : étape 2b — match_skill_rank pour les match_ids.
const Q5PlayerSkillRankHistoryTpl = `
SELECT
    match_id,
    NULLIF(TRIM(COALESCE(tier, '')), '')             AS skill_tier,
    NULLIF(TRIM(COALESCE(tier_fr, '')), '')          AS skill_tier_fr,
    NULLIF(TRIM(COALESCE(rating_type, '')), '')      AS skill_rating_type,
    NULLIF(TRIM(COALESCE(tier_label, '')), '')       AS skill_tier_label,
    expected_win_prob
FROM match_skill_rank
WHERE match_id IN (%s)`

// (Q5MatchHistory supprimée en P7-5 — code mort, aucun caller. Le pipeline
//  history actuel utilise Q5SharedHistory + Q5PlayerSkillRankHistoryTpl.)

// Q6 : Career — progression de rang (dernier état connu, per-field-merged).
// Paramètre : aucun.
//
// Pattern ARG_MAX(col, recorded_at) identique à qLoadLastCareerRank (flux home)
// et Q26cHomeSpartanIdentity. PAS un `ORDER BY recorded_at DESC LIMIT 1` naïf :
// le snapshot live le plus récent est souvent un partial "customization-only"
// (cf. career_progression_partial.go) où rank/current_xp sont écrits mais
// xp_for_next_rank/xp_total restent NULL (jamais renvoyés bruts par l'API live —
// ce sont des dérivés metadata). Un LIMIT 1 naïf piochait cette ligne partielle
// → jauges "progression rang/Héros" et "XP prochain rang" à 0. ARG_MAX prend la
// dernière valeur NON-NULL par colonne indépendamment.
//
// xp_for_next_rank / xp_total restent recalculés depuis career_ranks
// (CareerLiveRepo.EnrichFromMetadata) côté Go : ARG_MAX seul ne suffit pas car
// ces colonnes peuvent valoir 0 dans toutes les lignes (DEFAULT 0 réintroduit
// par la migration rebuild_career_progression + write-path live qui ne les
// écrit jamais).
//
// MAX(recorded_at) NULL ⟺ table vide : le caller traduit en sql.ErrNoRows.
const Q6CareerLatestRank = `
SELECT
    COALESCE(ARG_MAX(rank,             recorded_at), 0)     AS rank_number,
    COALESCE(ARG_MAX(current_xp,       recorded_at), 0)     AS current_xp,
    MAX(recorded_at)                                        AS recorded_at,
    NULLIF(TRIM(ARG_MAX(rank_name,     recorded_at)), '')   AS rank_name,
    NULLIF(TRIM(ARG_MAX(rank_tier,     recorded_at)), '')   AS rank_tier,
    COALESCE(ARG_MAX(xp_for_next_rank, recorded_at), 0)     AS xp_for_next_rank,
    COALESCE(ARG_MAX(xp_total,         recorded_at), 0)     AS xp_total,
    COALESCE(ARG_MAX(is_max_rank,      recorded_at), FALSE) AS is_max_rank
FROM career_progression`

// Q7 : Career — historique XP complet, monotone.
//
// Filtre les rows sans xp_total réel : post-migration fix_career_xp_total_default_zero
// (steps_player_fix_career_xp_total_default.go), xp_total=0/NULL est toujours un
// artefact d'un INSERT partial customization-only — jamais une valeur légitime.
//
// MAX(xp_total) OVER garantit la monotonie : si un row aberrant subsiste (ex :
// xp_total chuté par bug API ou écriture concurrente), il ne fait pas régresser
// la courbe — on garde le max précédent.
//
// Pas de fallback rank * 1000 : il sous-estimait massivement la vraie valeur
// historique (un Diamant à 5M XP voyait 300_000 quand xp_total devenait NULL).
//
// FILTRE PAR XUID (?1) : comme Q26c, career_progression peut contenir des rows
// d'un autre joueur (contamination de sync historique). Sans filtre, la courbe XP
// mélangeait les historiques. Concaténer xuid à une chaîne vide défait le pushdown
// index (cf. Q26c). Paramètre : ?1 = xuid du joueur.
const Q7CareerXPHistory = `
SELECT
    cp.recorded_at,
    cp.rank          AS rank_number,
    cp.current_xp,
    MAX(cp.xp_total) OVER (
        ORDER BY cp.recorded_at
        ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
    ) AS xp_total_cumulative
FROM career_progression cp
WHERE cp.xuid || '' = ? AND cp.xp_total IS NOT NULL AND cp.xp_total > 0
ORDER BY cp.recorded_at ASC`

// Q8LUSRHistoryPlayerTpl : Phase A de Q8 — partie player (match_skill_rank)
// exécutée sur pdb.Player. Le tri start_time + le LAG rating_delta sont
// calculés côté Go après merge avec match_registry (Phase B).
const Q8LUSRHistoryPlayer = `
SELECT
    msr.match_id,
    msr.rating_type,
    msr.rating_value,
    msr.tier_label,
    msr.playlist_group,
    NULLIF(TRIM(COALESCE(msr.tier, '')), '')               AS tier,
    COALESCE(msr.sub_tier, 0)                              AS sub_tier
FROM match_skill_rank msr`

// Q8LUSRHistoryRegistryTpl : Phase B de Q8 — start_time + playlist depuis
// match_registry pour les match_ids résultants de Phase A. Exécutée via
// SharedReader.
const Q8LUSRHistoryRegistryTpl = `
SELECT
    match_id,
    COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC') AS recorded_at,
    COALESCE(playlist_name_fr, playlist_name, '')           AS playlist_name,
    COALESCE(playlist_id, '')                               AS playlist_id
FROM match_registry
WHERE match_id IN (%s)`

// Q9 : Career — top matches : 10 meilleurs (WIN) + 10 moins bons (LOSS).
// Paramètres : ?1 = xuid (section WIN), ?2 = xuid (section LOSS).
// Filtres Python portés : had_bot_teammate=FALSE, time_played>=180s, is_firefight=FALSE.
// Tri badge priority : WIN → flags 5/3/1 (CONTRE_REMONTADA/REMONTADA/DOMINATION) DESC ;
//
//	LOSS → flags 4/2 (DEBANDADE/HUMILIATION) DESC.
//
// _s sert uniquement à séparer les sections (1=best, 2=worst) dans l'ORDER BY final.
// Q9TopMatchesPlayer : Phase A de Q9 — partie player (pme) avec filtre
// performance_score. Le tri (dominance flag + perf score) et la sélection
// par section (WIN/LOSS) sont faits côté Go après merge avec shared (Phase B).
//
// Le flag had_bot_teammate est désormais transmis au tri Go pour exclusion
// asymétrique : on garde les WIN avec bot coéquipier (perf personnelle
// méritoire malgré le handicap d'équipe), on rejette les LOSS avec bot
// coéquipier (responsabilité du joueur non isolable d'un déséquilibre 4v3).
const Q9TopMatchesPlayer = `
SELECT match_id, performance_score, COALESCE(dominance_flag, 0), COALESCE(had_bot_teammate, FALSE)
FROM player_match_enrichment_latest
WHERE performance_score IS NOT NULL`

// Q9TopMatchesShared : Phase B de Q9 — partie shared (mp + r) avec filtres
// shared-only (time_played >= 180, is_firefight = FALSE). Filtre xuid + IN
// les match_ids passés par Phase A.
const Q9TopMatchesSharedTpl = `
SELECT
    mp.match_id,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
    r.map_name,
    r.pair_name,
    r.playlist_name,
    COALESCE(mp.outcome, 0)   AS outcome,
    COALESCE(mp.kills, 0)     AS kills,
    COALESCE(mp.deaths, 0)    AS deaths,
    mp.kda,
    mp.team_mmr,
    mp.enemy_mmr
FROM match_participants mp
JOIN match_registry r ON mp.match_id = r.match_id
WHERE mp.xuid = ?
  AND COALESCE(mp.time_played_seconds, 0) >= 180
  AND COALESCE(r.is_firefight, FALSE) = FALSE
  AND mp.match_id IN (%s)`

// Q9bHighlightSharedTpl : Phase B partagée entre GetHighlightMatchIDs et
// GetHighlightPool — partie shared (mp + r) avec filtres shared-only
// (time_played >= 180, NOT is_firefight, outcome ∈ {2,3}, mp.match_id IN
// les match_ids de Phase A). Le 2e %s reçoit la clause dynamique
// (buildHighlightFilterClause) qui filtre sur r.is_ranked / r.start_time /
// r.pair_name / r.playlist_name.
const Q9bHighlightSharedTpl = `
SELECT
    mp.match_id,
    COALESCE(mp.outcome, 0)                                                AS outcome,
    COALESCE(r.is_ranked, FALSE)                                           AS is_ranked,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC')            AS start_time,
    COALESCE(NULLIF(r.pair_name_fr, ''), r.pair_name, '')                  AS pair_name_source,
    COALESCE(NULLIF(r.playlist_name_fr, ''), r.playlist_name, '')          AS playlist_name_source,
    COALESCE(r.playlist_id, '')                                            AS playlist_id
FROM match_participants mp
JOIN match_registry r ON mp.match_id = r.match_id
WHERE mp.xuid = ?
  AND COALESCE(mp.time_played_seconds, 0) >= 180
  AND COALESCE(r.is_firefight, FALSE) = FALSE
  AND COALESCE(mp.outcome, 0) IN (2, 3)
  AND mp.match_id IN (%s)
  %s`

// (Q9bHighlightMatchIDsTpl + Q9bHighlightPool supprimées en P7-5 — remplacées
//  par le pipeline split via Q9bHighlightSharedTpl + loadHighlightCandidates
//  côté Go en P7-3.)

// Q26CareerTopEncountersTpl : Career — joueurs les plus croisés au niveau global,
// hors amis configurés (FriendGamertags).
//
// Format string : %s à remplacer par la clause d'exclusion friends (vide si
// aucun ami) — ex. "AND es.xuid NOT IN (?, ?, ?)".
//
// Paramètres :
//
//	?1..?5 = xuid joueur (cf. positions ci-dessous)
//	?6..?N = xuids des amis à exclure (substitués via la clause %s)
//
// Inspiré de Q23bMatchEncounterStats mais scope global (sans contrainte sur un
// match_id particulier). Calcule count_together, ally/enemy counts, winrates,
// kills_dealt, deaths_suffered (via killer_victim_pairs), last_seen_at.
// Tri par count_together DESC, limite 10.
