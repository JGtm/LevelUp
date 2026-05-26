// Package duckdb â€” queries_home_citations.go : requÃªtes page Home, citations et mÃ©dias.
package duckdb

import (
	"regexp"
	"strings"
)

// Q26 — Home : matchs d'un joueur avec KPIs pour le hero card.
//
// Phase 3.bis plan stabilisation 2026-05-22 : split en 2 queries Go-side pour
// éviter le mix cross-DB (player + shared) qui forçait l'ATTACH shared sur la
// player conn — pattern interdit depuis ADR 0016 (cf. AUDIT §3 + §5).
//
//   - Q26HomeMatchesSharedPart : tables shared uniquement (match_participants,
//     match_registry, medals_earned via CTE perfect). Exécutée via SharedReader.
//     Tri chronologique + LIMIT 150 = source de vérité de la liste.
//   - Q26HomeMatchesPlayerEnrichTpl : tables player uniquement
//     (player_match_enrichment, match_skill_rank), filtrée par match_id IN (%s).
//     Exécutée sur pdb.Player.
//
// Le merge Go-side reconstruit la HomeMatchRow complète. Cf. LoadHomeMatches.
//
// Paramètres :
//   - Q26HomeMatchesSharedPart : ?1 = xuid (CTE perfect), ?2 = xuid (WHERE mp.xuid)
//   - Q26HomeMatchesPlayerEnrichTpl : pas de paramètre, juste IN (%s) match_ids
const Q26HomeMatchesSharedPart = `
WITH perfect AS (
    SELECT match_id, COALESCE(SUM(count), 0) AS perfect_kills
    FROM medals_earned
    WHERE xuid = ? AND medal_name_id = 1512363953
    GROUP BY match_id
)
SELECT
    mp.match_id,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
    COALESCE(r.map_id, '')                                  AS map_id,
    COALESCE(r.map_name, '')                                AS map_name,
    COALESCE(r.map_name_fr, r.map_name, '')                 AS map_name_fr,
    COALESCE(r.pair_id, '')                                 AS pair_id,
    COALESCE(r.pair_name, '')                               AS pair_name,
    COALESCE(r.pair_name_fr, r.pair_name, '')               AS pair_name_fr,
    COALESCE(r.game_variant_id, '')                         AS game_variant_id,
    COALESCE(r.game_variant_name, '')                       AS game_variant_name,
    COALESCE(r.playlist_id, '')                             AS playlist_id,
    COALESCE(r.playlist_name, '')                           AS playlist_name,
    COALESCE(r.playlist_name_fr, r.playlist_name, '')       AS playlist_name_fr,
    COALESCE(r.is_firefight, FALSE)                         AS is_firefight,
    CASE
        WHEN COALESCE(r.is_ranked, FALSE)
            OR STRPOS(LOWER(COALESCE(r.playlist_name, '')), 'ranked') > 0
            OR STRPOS(LOWER(COALESCE(r.pair_name, '')), 'ranked') > 0
        THEN TRUE
        ELSE FALSE
    END                                                     AS is_ranked,
    COALESCE(mp.outcome, 0)                                 AS outcome,
    COALESCE(mp.team_id, -1)                                AS team_id,
    COALESCE(r.team_0_score, -1)                            AS team_0_score,
    COALESCE(r.team_1_score, -1)                            AS team_1_score,
    COALESCE(mp.kills, 0)                                   AS kills,
    COALESCE(mp.deaths, 0)                                  AS deaths,
    COALESCE(mp.assists, 0)                                 AS assists,
    mp.kda,
    CASE WHEN COALESCE(mp.deaths, 0) > 0
         THEN CAST(COALESCE(mp.kills, 0) AS DOUBLE) / CAST(mp.deaths AS DOUBLE)
         ELSE CAST(COALESCE(mp.kills, 0) AS DOUBLE) END     AS ratio,
    mp.accuracy,
    mp.avg_life_seconds,
    NULLIF(COALESCE(NULLIF(mp.time_played_seconds, 0), r.playable_duration_seconds), 0) AS time_played_seconds,
    mp.damage_dealt,
    mp.damage_taken,
    mp.team_mmr,
    mp.enemy_mmr,
    mp.rank                                                 AS rank_in_team,
    COALESCE(mp.headshot_kills, 0)                          AS headshot_kills,
    COALESCE(perfect.perfect_kills, 0)                      AS perfect_kills,
    mp.max_killing_spree                                    AS max_killing_spree
FROM match_participants mp
JOIN match_registry r ON r.match_id = mp.match_id
LEFT JOIN perfect ON perfect.match_id = mp.match_id
WHERE mp.xuid = ?
ORDER BY COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') DESC
LIMIT 150`

// Q26HomeMatchesPlayerEnrichTpl : enrichissement player (pme + msr) pour un
// lot de match_ids. À utiliser via fmt.Sprintf(tpl, strings.Join(placeholders, ", ")).
const Q26HomeMatchesPlayerEnrichTpl = `
SELECT
    pme.match_id,
    pme.session_label,
    COALESCE(pme.is_with_friends, FALSE) AS is_with_friends,
    COALESCE(pme.dominance_flag, 0)      AS dominance_flag,
    pme.performance_score,
    msr.rating_type,
    msr.rating_value,
    msr.tier,
    COALESCE(msr.sub_tier, 0)            AS sub_tier,
    msr.tier_label,
    msr.rating_delta,
    msr.playlist_group
FROM player_match_enrichment pme
LEFT JOIN match_skill_rank msr ON msr.match_id = pme.match_id
WHERE pme.match_id IN (%s)`

// Q27HomeSessionsPlayerPart : Phase A de Q27 — sessions du joueur depuis
// player_match_enrichment uniquement. Le start_time est récupéré via une
// 2e query SharedReader (cf. Q27HomeSessionsSharedStartTimes).
//
// Phase 3.bis plan stabilisation 2026-05-22 : split de Q27HomeSessions
// (qui mixait player + shared) en 2 phases Go-side. Pattern aligné sur Q26.
const Q27HomeSessionsPlayerPart = `
SELECT
    pme.match_id,
    pme.session_id,
    pme.session_label,
    COALESCE(pme.is_with_friends, FALSE) AS is_with_friends
FROM player_match_enrichment pme
WHERE pme.session_label IS NOT NULL`

// Q27HomeSessionsSharedStartTimesTpl : Phase B de Q27 — start_time pour
// un lot de match_ids depuis match_registry (shared).
const Q27HomeSessionsSharedStartTimesTpl = `
SELECT
    match_id,
    COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC') AS start_time
FROM match_registry
WHERE match_id IN (%s)`

// Q26h : Home â€” mÃ©dailles par match pour un joueur, lots de match_id.
// ParamÃ¨tres : ?1 = xuid. Les match_id sont injectÃ©s dynamiquement via IN (%s).
// RequÃªte sur pdb.Player (shared attachÃ©) ; labels rÃ©solus ensuite via metadata.
const Q26hMatchMedalsTemplate = `
SELECT
    me.match_id,
    me.medal_name_id,
    COALESCE(me.count, 1) AS count
FROM medals_earned me
WHERE me.xuid = ?
  AND me.match_id IN (%s)
ORDER BY me.match_id, me.count DESC`

// Q26i : Home â€” citations progressÃ©es par match, avec cumul global au moment du match.
// Les match_id sont injectÃ©s dynamiquement via IN (%s).
// RequÃªte sur pdb.Player uniquement (match_citations est dans stats.duckdb).
const Q26iMatchCitationsTemplate = `
SELECT
    mc.match_id,
    mc.citation_name_norm,
    mc.value                AS match_delta,
    cum.total               AS cumulative_total
FROM match_citations mc
JOIN (
    SELECT citation_name_norm, SUM(value) AS total
    FROM match_citations
    GROUP BY citation_name_norm
) cum ON cum.citation_name_norm = mc.citation_name_norm
WHERE mc.match_id IN (%s)
  AND mc.value > 0
ORDER BY mc.match_id, mc.value DESC`

// Q26j : Home â€” mÃ©tadonnÃ©es citations depuis metadata.duckdb pour un ensemble de norms.
// Les citation_name_norm sont injectÃ©s dynamiquement via IN (%s).
// GROUP BY car une citation peut avoir plusieurs medal_id rows.
const Q26jCitationMappingsForNormsTemplate = `
SELECT
    citation_name_norm,
    citation_name_display,
    COALESCE(image_path, '')   AS image_path,
    COALESCE(tier_targets, '') AS tier_targets,
    COALESCE(MAX(description), '') AS description
FROM citation_mappings
WHERE citation_name_norm IN (%s)
  AND enabled IS NOT FALSE
GROUP BY citation_name_norm, citation_name_display, image_path, tier_targets`

// Q26b : Home -- nombre total de matchs d un joueur (pas de LIMIT).
// Parametre : ?1 = xuid du joueur.
const Q26bCountPlayerMatches = `
SELECT COUNT(*) FROM match_participants WHERE xuid = ?`

// Q26c : Home -- identitÃ© record compacte depuis career_progression.
// Un seul scan via ARG_MAX â€” remplace les 5 sous-requÃªtes corrÃ©lÃ©es de l'ancienne version.
// ParamÃ¨tre : aucun.
const Q26cHomeSpartanIdentity = `
SELECT
    ARG_MAX(rank,             recorded_at)                                                                  AS rank,
    COALESCE(ARG_MAX(current_xp,      recorded_at), 0)                                                     AS current_xp,
    COALESCE(ARG_MAX(xp_for_next_rank, recorded_at), 0)                                                    AS xp_for_next_rank,
    COALESCE(ARG_MAX(is_max_rank,     recorded_at), FALSE)                                                  AS is_max_rank,
    ARG_MAX(spartan_id,       recorded_at) FILTER (WHERE NULLIF(TRIM(spartan_id),       '') IS NOT NULL)    AS spartan_id,
    NULLIF(TRIM(ARG_MAX(rank_name,    recorded_at)), '')                                                    AS rank_name,
    NULLIF(TRIM(ARG_MAX(rank_tier,    recorded_at)), '')                                                    AS rank_tier,
    ARG_MAX(banner_image_url,  recorded_at) FILTER (WHERE NULLIF(TRIM(banner_image_url),  '') IS NOT NULL)  AS banner_image_url,
    ARG_MAX(emblem_image_url,  recorded_at) FILTER (WHERE NULLIF(TRIM(emblem_image_url),  '') IS NOT NULL)  AS emblem_image_url,
    ARG_MAX(backdrop_image_url, recorded_at) FILTER (WHERE NULLIF(TRIM(backdrop_image_url),'') IS NOT NULL) AS backdrop_image_url,
    ARG_MAX(adornment_path,   recorded_at) FILTER (WHERE NULLIF(TRIM(adornment_path),   '') IS NOT NULL)    AS adornment_path
FROM career_progression`

// Q26d : Home -- assets visuels du rang carriÃ¨re courant depuis metadata.duckdb.
//
// Les libellÃ©s (title FR/EN, next_rank_title) ne sont PAS lus ici : ils
// proviennent du TitleSemanticAdapter (career_rank_translations) cÃ´tÃ© service.
// Le repo storage reste exclusivement responsable des paths d'assets.
//
// ParamÃ¨tre : ?1 = rank_id.
const Q26dHomeCareerRankMeta = `
SELECT
	COALESCE(
		NULLIF(TRIM(large_icon_path), ''),
		NULLIF(TRIM(icon_path), '')
	) AS image_path,
	NULLIF(TRIM(adornment_icon_path), '') AS adornment_path
FROM career_ranks
WHERE rank_id = ?
LIMIT 1`

// Q26e : Home -- meilleur rating historique par type (CSR ou LUSR), avec
// statut de placement par playlist_group.
//
// ParamÃ¨tre : ?1 = rating_type ('CSR' ou 'LUSR', case-insensitive).
//
// Retourne 1 ligne max :
//
//	rating_value, tier_label, tier, sub_tier, placement_remaining
//
// Logique placement (LUSR principalement, CSR secondaire en fallback) :
//   - chaque playlist_group a sa propre phase de placement de 10 matchs
//   - match_count = COUNT(*) des rows par playlist_group pour le type donné
//   - placement_remaining = GREATEST(0, 10 - match_count)
//   - on prÃ©fÃ¨re le meilleur rating d'un groupe matured (match_count >= 10) ;
//     sinon on retourne le meilleur rating du groupe le plus jouÃ© en placement,
//     pour que le badge unranked_(10-remaining).png puisse Ãªtre construit.
//
// **NULL handling — fix bug prod 2026-05-20** : `playlist_group` peut être NULL
// pour les anciennes rows LUSR/CSR (avant introduction de la colonne ou pour
// les CSR fetchés sans group). Sans COALESCE en sentinel, le `JOIN ... ON
// gc.playlist_group = bpg.playlist_group` éliminerait TOUTES les rows à
// playlist_group NULL (sémantique SQL `NULL = NULL` → NULL → exclu de l'inner
// JOIN). C'est exactement le bug observé pour JGtm : 758 rows LUSR existaient
// mais highest_lusr ressortait null. On normalise via COALESCE(..., '_unknown').
//
// Le CASE de classification CSR/LUSR garde le fallback heuristique sur
// playlist_name/pair_name (régression historique is_ranked=FALSE non corrigée).
// Q26ePeakPhaseAPlayer : Phase A (player-only) — match_skill_rank brut.
// Sprint P7 / ADR 0016 (2026-05-20) : exécutée via pdb.Player, sans
// shared. Classification CSR/LUSR faite côté Go après Phase B.
const Q26ePeakPhaseAPlayer = `
SELECT
	msr.match_id,
	msr.playlist_group,
	msr.rating_value,
	msr.rating_type,
	msr.tier,
	msr.sub_tier,
	msr.tier_label,
	COALESCE(msr.updated_at, msr.start_time, msr.created_at) AS recency
FROM match_skill_rank msr
WHERE msr.rating_value IS NOT NULL`

// Q26ePeakPhaseBRegistryTpl : Phase B (shared-only) — registry pour
// classification. Exécutée via pdb.SharedReadDB().Get() — pas de préfixe.
const Q26ePeakPhaseBRegistryTpl = `
SELECT
	mr.match_id,
	COALESCE(mr.is_ranked, FALSE)  AS is_ranked,
	COALESCE(mr.playlist_name, '') AS playlist_name,
	COALESCE(mr.pair_name, '')     AS pair_name
FROM match_registry mr
WHERE mr.match_id IN (%s)`

const Q26eHomeSkillPeakByType = `
WITH classified AS (
	SELECT
		msr.match_id,
		COALESCE(msr.playlist_group, '_unknown') AS playlist_group,
		msr.rating_value,
		msr.tier,
		msr.sub_tier,
		msr.tier_label,
		msr.created_at,
		msr.updated_at,
		msr.start_time,
		CASE
			WHEN mr.match_id IS NOT NULL THEN CASE
				WHEN COALESCE(mr.is_ranked, FALSE)
					OR STRPOS(LOWER(COALESCE(mr.playlist_name, '')), 'ranked') > 0
					OR STRPOS(LOWER(COALESCE(mr.pair_name, '')), 'ranked') > 0
				THEN 'CSR'
				ELSE 'LUSR'
			END
			WHEN UPPER(COALESCE(NULLIF(TRIM(msr.rating_type), ''), '')) = 'CSR' THEN 'CSR'
			ELSE 'LUSR'
		END AS effective_type
	FROM match_skill_rank msr
	LEFT JOIN shared.match_registry mr ON mr.match_id = msr.match_id
	WHERE msr.rating_value IS NOT NULL
),
typed AS (
	SELECT * FROM classified WHERE effective_type = UPPER(?)
),
group_counts AS (
	SELECT playlist_group, COUNT(*) AS match_count
	FROM typed
	GROUP BY playlist_group
),
best_per_group AS (
	SELECT
		t.playlist_group,
		t.rating_value,
		t.tier,
		t.sub_tier,
		t.tier_label,
		t.match_id,
		COALESCE(t.updated_at, t.start_time, t.created_at) AS recency,
		ROW_NUMBER() OVER (
			PARTITION BY t.playlist_group
			ORDER BY
				t.rating_value DESC,
				COALESCE(t.updated_at, t.start_time, t.created_at) DESC,
				COALESCE(t.sub_tier, 0) DESC,
				t.match_id DESC
		) AS rn
	FROM typed t
)
SELECT
	bpg.rating_value,
	NULLIF(TRIM(bpg.tier_label), '') AS tier_label,
	NULLIF(TRIM(bpg.tier), '') AS tier,
	COALESCE(bpg.sub_tier, 0) AS sub_tier,
	GREATEST(0, 10 - gc.match_count) AS placement_remaining
FROM best_per_group bpg
JOIN group_counts gc ON gc.playlist_group = bpg.playlist_group
WHERE bpg.rn = 1
ORDER BY
	CASE WHEN gc.match_count >= 10 THEN 1 ELSE 0 END DESC,
	bpg.rating_value DESC,
	bpg.recency DESC
LIMIT 1`

// Q26g : Home â€” 3 derniÃ¨res playlists distinctes jouÃ©es avec leur dernier rang compÃ©titif.
// ParamÃ¨tre : ?1 = xuid du joueur.
// Retourne (playlist_id, playlist_name, is_ranked, rating_type, rating_value, tier, tier_fr,
//
//	sub_tier, tier_label, measurement_matches_remaining).
//
// playlist_name_fr est rÃ©solu en Go depuis asset_translations (mÃªme source que les tuiles de matchs).
// rating_* sont NULL pour les playlists sans rang calculÃ©.
// measurement_matches_remaining vient de player_csr_snapshots (snapshot le plus rÃ©cent par playlist)
// pour permettre d'Ã©mettre `unranked_N.png` pendant la phase de placement (10 â†’ 0 matchs restants).
// Q26gPlaylistPhaseBShared : Phase B (shared) — top 3 playlists pour xuid
// avec le dernier match_id par playlist. Sprint P7 / ADR 0016 : sans préfixe
// shared. (exécuté via pdb.SharedReadDB().Get()).
//
// Fallback playlist_name : beaucoup de matchs ont playlist_id = NULL dans
// match_registry (Social, anciens matchs non backfillés). On groupe sur
// COALESCE(playlist_id, playlist_name) pour capturer ces cas et retourner
// jusqu'à 3 playlists distinctes même sans playlist_id renseigné.
const Q26gPlaylistPhaseBShared = `
WITH per_playlist AS (
	SELECT
		COALESCE(NULLIF(TRIM(r.playlist_id), ''), NULLIF(TRIM(r.playlist_name), '')) AS group_key,
		COALESCE(MAX(NULLIF(TRIM(r.playlist_id), '')), '') AS playlist_id,
		COALESCE(MAX(r.playlist_name), '') AS playlist_name,
		MAX(CASE
			WHEN COALESCE(r.is_ranked, FALSE)
				OR STRPOS(LOWER(COALESCE(r.playlist_name, '')), 'ranked') > 0
				OR STRPOS(LOWER(COALESCE(r.pair_name, '')), 'ranked') > 0
			THEN 1 ELSE 0
		END) > 0 AS is_ranked,
		MAX(COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC')) AS last_played,
		ARG_MAX(r.match_id, COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC')) AS last_match_id,
		ARG_MAX(r.season_id, COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC')) AS last_season_id
	FROM match_participants mp
	JOIN match_registry r ON r.match_id = mp.match_id
	WHERE mp.xuid = ?
	  AND COALESCE(NULLIF(TRIM(r.playlist_id), ''), NULLIF(TRIM(r.playlist_name), '')) IS NOT NULL
	GROUP BY group_key
)
SELECT playlist_id, playlist_name, is_ranked, last_played, last_match_id, last_season_id
FROM per_playlist
ORDER BY last_played DESC
LIMIT 3`

// Q26gPlaylistPhaseAMSRTpl : Phase A1 (player) — rating + tier des last_match_id.
const Q26gPlaylistPhaseAMSRTpl = `
SELECT
	match_id,
	rating_value,
	NULLIF(TRIM(tier), '')        AS tier,
	NULLIF(TRIM(tier_fr), '')     AS tier_fr,
	COALESCE(sub_tier, 0)         AS sub_tier,
	NULLIF(TRIM(tier_label), '')  AS tier_label
FROM match_skill_rank
WHERE match_id IN (%s)
  AND rating_value IS NOT NULL`

// Q26gPlaylistPhaseASnapshotTpl : Phase A2 (player) — placement remaining
// par playlist_id depuis player_csr_snapshots (snapshot le plus récent).
const Q26gPlaylistPhaseASnapshotTpl = `
WITH ranked AS (
	SELECT
		playlist_id,
		current_measurement_remaining,
		ROW_NUMBER() OVER (
			PARTITION BY playlist_id
			ORDER BY fetched_at DESC, season_id DESC
		) AS rn
	FROM player_csr_snapshots_latest
	WHERE playlist_id IN (%s)
)
SELECT playlist_id, current_measurement_remaining
FROM ranked
WHERE rn = 1`

const Q26gHomePlaylistRanks = `
WITH recent_playlists AS (
	SELECT
		r.playlist_id,
		COALESCE(MAX(r.playlist_name), '') AS playlist_name,
		MAX(CASE
			WHEN COALESCE(r.is_ranked, FALSE)
				OR STRPOS(LOWER(COALESCE(r.playlist_name, '')), 'ranked') > 0
				OR STRPOS(LOWER(COALESCE(r.pair_name, '')), 'ranked') > 0
			THEN 1 ELSE 0
		END) > 0                           AS is_ranked,
		MAX(COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC')) AS last_played
	FROM shared.match_participants mp
	JOIN shared.match_registry r ON r.match_id = mp.match_id
	WHERE mp.xuid = ?
	  AND NULLIF(TRIM(COALESCE(r.playlist_id, '')), '') IS NOT NULL
	GROUP BY r.playlist_id
	ORDER BY MAX(COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC')) DESC
	LIMIT 3
),
last_skill AS (
	SELECT
		r.playlist_id,
		CASE
			WHEN COALESCE(r.is_ranked, FALSE)
				OR STRPOS(LOWER(COALESCE(r.playlist_name, '')), 'ranked') > 0
				OR STRPOS(LOWER(COALESCE(r.pair_name, '')), 'ranked') > 0
			THEN 'CSR'
			ELSE 'LUSR'
		END AS rating_type,
		msr.rating_value,
		NULLIF(TRIM(msr.tier), '')               AS tier,
		NULLIF(TRIM(msr.tier_fr), '')            AS tier_fr,
		COALESCE(msr.sub_tier, 0)                AS sub_tier,
		NULLIF(TRIM(msr.tier_label), '')         AS tier_label,
		ROW_NUMBER() OVER (
			PARTITION BY r.playlist_id
			ORDER BY COALESCE(msr.start_time, msr.updated_at, msr.created_at) DESC
		) AS rn
	FROM match_skill_rank msr
	JOIN shared.match_registry r ON r.match_id = msr.match_id
	WHERE msr.rating_value IS NOT NULL
),
csr_snapshot AS (
	SELECT
		playlist_id,
		current_measurement_remaining,
		ROW_NUMBER() OVER (
			PARTITION BY playlist_id
			ORDER BY fetched_at DESC, season_id DESC
		) AS rn
	FROM player_csr_snapshots
)
SELECT
	rp.playlist_id,
	rp.playlist_name,
	rp.is_ranked,
	ls.rating_type,
	ls.rating_value,
	ls.tier,
	ls.tier_fr,
	ls.sub_tier,
	ls.tier_label,
	cs.current_measurement_remaining
FROM recent_playlists rp
LEFT JOIN last_skill ls ON ls.playlist_id = rp.playlist_id AND ls.rn = 1
LEFT JOIN csr_snapshot cs ON cs.playlist_id = rp.playlist_id AND cs.rn = 1
ORDER BY rp.last_played DESC`

// Q27HomeSessions : DÉPRÉCIÉ — split en 2 phases via
// Q27HomeSessionsPlayerPart + Q27HomeSessionsSharedStartTimesTpl
// (Phase 3.bis plan stabilisation 2026-05-22). Conservé en commentaire
// comme référence historique. Code mort retiré.

// Q28 : Home â€” medias recents depuis media_files + media_match_associations.
// Parametre : ?1 = LIMIT (nombre de medias).
// Retourne uniquement les medias actifs, triés par date de modification desc.
//
// Phase 3 plan stabilisation 2026-05-22 : migré vers pdb.SharedSocial. Le
// schéma shared_social.media_files + media_match_associations diffère de
// l'ancien player.media_files :
//   - JOIN via `mma.media_file_id = mf.id` (au lieu de file_path/media_path)
//   - match_start_time absent → retourné NULL (consumer doit dériver depuis
//     match_registry si besoin via une autre query Go-side)
const Q28RecentMedia = `
SELECT
    mf.file_name,
    mma.match_id,
    NULL::TIMESTAMP AS match_start_time
FROM media_files mf
LEFT JOIN media_match_associations mma ON mma.media_file_id = mf.id
WHERE mf.status = 'active'
ORDER BY mf.mtime DESC
LIMIT ?`

// Q26k : Home â€” arme favorite (kills totaux) du joueur toutes armes confondues.
// ParamÃ¨tre : ?1 = xuid.
// RequÃªte sur pdb.Player (shared attachÃ©). Label rÃ©solu ensuite via pdb.Metadata.
const Q26kFavoriteWeapon = `
SELECT
    wk.effective_weapon_id AS weapon_id,
    COUNT(*)               AS total_kills
FROM v_weapon_kills wk
WHERE wk.xuid = ?
  AND wk.effective_weapon_id NOT IN (0, 1, 2)
GROUP BY wk.effective_weapon_id
ORDER BY total_kills DESC
LIMIT 1`

// =============================================================================
// Sprint 13 â€” Citations + MÃ©dias
// =============================================================================

// Q34 : Citations â€” mappings de citation depuis metadata.duckdb.
// ParamÃ¨tre : aucun. RequÃªte sur pdb.Metadata (pas pdb.Player).
const Q34CitationMappings = `
SELECT
    citation_name_norm,
    citation_name_display,
    mapping_type,
    COALESCE(category, 'misc')    AS category,
    image_path,
    description,
    tier_targets,
    composite_children
FROM citation_mappings
WHERE enabled IS NOT FALSE
ORDER BY category, citation_name_display`

// Q35 : Citations â€” totaux agrÃ©gÃ©s depuis match_citations (player stats.duckdb).
// ParamÃ¨tre : aucun.
const Q35CitationTotals = `
SELECT
    citation_name_norm,
    SUM(value) AS total
FROM match_citations
WHERE citation_name_norm NOT LIKE '\_%%' ESCAPE '\'
GROUP BY citation_name_norm
ORDER BY total DESC`

// Q36a : Commendations â€” total de mÃ©dailles gagnÃ©es par medal_id (xuid du joueur).
// ParamÃ¨tre : ?1 = xuid du joueur. RequÃªte sur pdb.Player (shared attachÃ©).
const Q36aMedalTotals = `
SELECT
    medal_id,
    SUM(count) AS total_count
FROM shared.medals_earned
WHERE xuid = ?
GROUP BY medal_id
ORDER BY total_count DESC`

// Q36b : Commendations â€” mappings mÃ©dailleâ†’citation depuis metadata.duckdb.
// ParamÃ¨tre : aucun. RequÃªte sur pdb.Metadata.
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

// Q38 : Match view â€” top citations gagnÃ©es dans un match (match_citations + citation_mappings).
// ParamÃ¨tre : ?1 = match_id.
// Retourne 3 colonnes : citation_name_norm, citation_name_display, value.
// RequÃªte sur pdb.Player (match_citations) + pdb.Metadata (citation_mappings).
const Q38MatchViewCitations = `
SELECT
    mc.citation_name_norm,
    COALESCE(cm.citation_name_display, mc.citation_name_norm) AS citation_name_display,
    mc.value
FROM match_citations mc
LEFT JOIN citation_mappings cm
    ON cm.citation_name_norm = mc.citation_name_norm
   AND cm.enabled IS NOT FALSE
WHERE mc.match_id = ?
  AND mc.citation_name_norm IS NOT NULL
  AND mc.value > 0
ORDER BY mc.value DESC
LIMIT 4`

// Q40 : Moteur citations complet â€” tous les champs de citation_mappings.
// ParamÃ¨tre : aucun. RequÃªte sur metadata.duckdb (passÃ© comme DB racine dans le sync).
// Retourne 11 colonnes pour le dispatch par mapping_type.
const Q40CitationFullMappings = `
SELECT
    citation_name_norm,
    citation_name_display,
    COALESCE(mapping_type, 'medal') AS mapping_type,
    COALESCE(category, 'misc')     AS category,
    medal_id,
    medal_ids,
    stat_name,
    award_name,
    custom_function,
    composite_children,
    tier_targets
FROM citation_mappings
WHERE enabled IS NOT FALSE
ORDER BY citation_name_norm`

// Q39 : Moteur citations â€” mappings citationâ†’medal depuis metadata.duckdb.
// ParamÃ¨tre : aucun. RequÃªte sur pdb.Metadata.
// Retourne 4 colonnes : citation_name_norm, citation_name_display, medal_id, mapping_type.
const Q39CitationMedalMappings = `
SELECT
    citation_name_norm,
    citation_name_display,
    medal_id,
    COALESCE(mapping_type, 'medal') AS mapping_type
FROM citation_mappings
WHERE enabled IS NOT FALSE
  AND medal_id IS NOT NULL
ORDER BY citation_name_norm`

// Médias — les anciens builders SQL Q37 (BuildQ37MediaQuery/Count/Options)
// ont été supprimés en P1 (refactor pipeline Go via SharedReader, ADR 0016).
// Les expressions SQL `q37*LabelExpr` et leurs régressions cross-DB
// (`shared.match_registry`) sont remplacées par les helpers Go équivalents
// dans media_repo_q37_pipeline.go (computedMapLabel, computedModeLabel,
// computedPlaylistLabel) — chargés via loadMediaMatchRegistry sur SharedReader.

type mediaWhereConfig struct {
	includeMapFilter      bool
	includeModeFilter     bool
	includePlaylistFilter bool
}

// mediaKindEquivalents retourne les valeurs DB qui doivent matcher un filtre
// de type donné, en couvrant à la fois la convention legacy ("clip"/"screenshot")
// et la nouvelle ("video"/"image"). Les médias indexés par les anciennes
// versions ont l'une, les nouveaux uploads ont l'autre — sans cette translation
// le filtre type ne retourne 0 résultat.
func mediaKindEquivalents(kind string) []string {
	switch kind {
	case "clip", "video":
		return []string{"clip", "video"}
	case "screenshot", "image":
		return []string{"screenshot", "image"}
	default:
		return []string{kind}
	}
}

// Strips appliqués côté Go pour normaliser une valeur de filtre map.
// Pattern conservateur : ne touche pas " Annex", `:`, etc.
var (
	mediaMapForgeSuffixRe   = regexp.MustCompile(`(?i)\s*-\s*Forge.*$`)
	mediaMapRankedSuffixRe  = regexp.MustCompile(`(?i)\s*-\s*Ranked.*$`)
	mediaMapVersionSuffixRe = regexp.MustCompile(`(?i)\s+v\d+$`)
)

// normalizeMediaMapName strippe les suffixes de variante pour grouper
// "Recharge v3" / "Recharge - Forge" / "Recharge" sous le même nom canonique.
func normalizeMediaMapName(s string) string {
	s = mediaMapForgeSuffixRe.ReplaceAllString(s, "")
	s = mediaMapRankedSuffixRe.ReplaceAllString(s, "")
	s = mediaMapVersionSuffixRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// mediaQueryConfig porte les paramètres de scoping (player_slug) pour le
// pipeline Q37. Utilisé par media_repo_q37_pipeline.go.
type mediaQueryConfig struct {
	playerSlug string
}

func (cfg mediaQueryConfig) useSharedSocialSchema() bool {
	return cfg.playerSlug != ""
}

// baseWhereClause renvoie les contraintes de base + le filtre de section (ownership).
//
//	"" (vide)   → sources visibles : mine + teammate (pas de contrainte player_slug)
//	"mine"      → uniquement player_slug courant
//	"teammate"  → uniquement les autres (player_slug != courant)
//
// En schéma legacy (pas de player_slug), seul "mine" et "" donnent des résultats ;
// "teammate" force WHERE FALSE pour cohérence (rien à montrer).
func (cfg mediaQueryConfig) baseWhereClause(sectionFilter string) ([]string, []any) {
	if !cfg.useSharedSocialSchema() {
		switch sectionFilter {
		case "teammate":
			return []string{"FALSE"}, nil
		default:
			return []string{"mf.status = 'active'"}, nil
		}
	}
	switch sectionFilter {
	case "mine":
		return []string{"mf.player_slug = ?"}, []any{cfg.playerSlug}
	case "teammate":
		return []string{"mf.player_slug <> ?"}, []any{cfg.playerSlug}
	default:
		// Tous auteurs — pas de contrainte sur player_slug
		return nil, nil
	}
}

// Q41 : MatchView Summary — match_citations + cumul global pour un seul match (player DB).
// Les métadonnées (display, image_path, tier_targets, description) sont chargées
// séparément via Q26j sur la metadata DB et mergées en Go (citation_mappings n'est
// pas attachée au player DB — voir pool.go).
const Q41SummaryTabCitations = `
SELECT
    mc.citation_name_norm,
    mc.value                AS match_delta,
    cum.total               AS cumulative_total
FROM match_citations mc
JOIN (
    SELECT citation_name_norm, SUM(value) AS total
    FROM match_citations
    GROUP BY citation_name_norm
) cum ON cum.citation_name_norm = mc.citation_name_norm
WHERE mc.match_id = ?
  AND mc.value > 0
ORDER BY mc.value DESC`
