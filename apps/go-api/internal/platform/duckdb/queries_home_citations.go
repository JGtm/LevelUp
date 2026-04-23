// Package duckdb — queries_home_citations.go : requêtes page Home, citations et médias.
package duckdb

import (
	"strings"

	"levelup/go-api/internal/domain"
)

// Q26 : Home — matchs d un joueur avec KPIs pour le hero card.
// Parametre : ?1 = xuid du joueur.
// Pas de LIMIT : tous les matchs sont chargés (hero card, highlights, recent matches, sessions).
const Q26HomeMatches = `
WITH perfect AS (
    SELECT match_id, COALESCE(SUM(count), 0) AS perfect_kills
    FROM shared.medals_earned
    WHERE xuid = ? AND medal_name_id = 1512363953
    GROUP BY match_id
)
SELECT
    mp.match_id,
    r.start_time,
    COALESCE(r.map_id, '')                                  AS map_id,
    COALESCE(r.map_name, '')                                AS map_name,
    COALESCE(r.map_name_fr, r.map_name, '')                 AS map_name_fr,
	COALESCE(r.pair_id, '')                                  AS pair_id,
    COALESCE(r.pair_name, '')                               AS pair_name,
    COALESCE(r.pair_name_fr, r.pair_name, '')               AS pair_name_fr,
	COALESCE(r.game_variant_id, '')                          AS game_variant_id,
	COALESCE(r.game_variant_name, '')                        AS game_variant_name,
	COALESCE(r.playlist_id, '')                              AS playlist_id,
	COALESCE(r.playlist_name, '')                            AS playlist_name,
	COALESCE(r.playlist_name_fr, r.playlist_name, '')       AS playlist_name_fr,
    COALESCE(r.is_firefight, FALSE)                         AS is_firefight,
    CASE
		WHEN COALESCE(r.is_ranked, FALSE)
			OR STRPOS(LOWER(COALESCE(r.playlist_name, '')), 'ranked') > 0
			OR STRPOS(LOWER(COALESCE(r.pair_name, '')), 'ranked') > 0
		THEN TRUE
		ELSE FALSE
	END                                                      AS is_ranked,
    pme.session_label,
    COALESCE(pme.is_with_friends, FALSE)                    AS is_with_friends,
    COALESCE(mp.outcome, 0)                                 AS outcome,
	COALESCE(mp.team_id, -1)                                AS team_id,
	COALESCE(r.team_0_score, -1)                            AS team_0_score,
	COALESCE(r.team_1_score, -1)                            AS team_1_score,
	COALESCE(pme.dominance_flag, 0)                         AS dominance_flag,
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
    pme.performance_score,
    msr.rating_value                                        AS skill_rating_value,
    CASE
        WHEN COALESCE(r.is_ranked, FALSE)
            OR STRPOS(LOWER(COALESCE(r.playlist_name, '')), 'ranked') > 0
            OR STRPOS(LOWER(COALESCE(r.pair_name, '')), 'ranked') > 0
        THEN 'CSR'
        WHEN UPPER(COALESCE(NULLIF(TRIM(msr.rating_type), ''), '')) = 'CSR' THEN 'CSR'
        ELSE 'LUSR'
    END                                                      AS skill_rating_type,
    msr.tier                                                AS skill_tier,
    COALESCE(msr.sub_tier, 0)                               AS skill_sub_tier,
    msr.tier_label                                          AS skill_tier_label,
    msr.rating_delta                                        AS skill_rating_delta,
    msr.playlist_group                                      AS skill_playlist_group,
    mp.rank                                                 AS rank_in_team,
    COALESCE(mp.headshot_kills, 0)                          AS headshot_kills,
    COALESCE(perfect.perfect_kills, 0)                      AS perfect_kills
FROM shared.match_participants mp
JOIN shared.match_registry r ON r.match_id = mp.match_id
LEFT JOIN player_match_enrichment pme ON pme.match_id = mp.match_id
LEFT JOIN match_skill_rank msr ON msr.match_id = mp.match_id
LEFT JOIN perfect ON perfect.match_id = mp.match_id
WHERE mp.xuid = ?
ORDER BY r.start_time DESC`

// Q26h : Home — médailles par match pour un joueur, lots de match_id.
// Paramètres : ?1 = xuid. Les match_id sont injectés dynamiquement via IN (%s).
// Requête sur pdb.Player (shared attaché) ; labels résolus ensuite via metadata.
const Q26hMatchMedalsTemplate = `
SELECT
    me.match_id,
    me.medal_name_id,
    COALESCE(me.count, 1) AS count
FROM shared.medals_earned me
WHERE me.xuid = ?
  AND me.match_id IN (%s)
ORDER BY me.match_id, me.count DESC`

// Q26i : Home — citations progressées par match, avec cumul global au moment du match.
// Les match_id sont injectés dynamiquement via IN (%s).
// Requête sur pdb.Player uniquement (match_citations est dans stats.duckdb).
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

// Q26j : Home — métadonnées citations depuis metadata.duckdb pour un ensemble de norms.
// Les citation_name_norm sont injectés dynamiquement via IN (%s).
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
SELECT COUNT(*) FROM shared.match_participants WHERE xuid = ?`

// Q26c : Home -- identité record compacte depuis career_progression.
// Paramètre : aucun.
const Q26cHomeSpartanIdentity = `
WITH latest AS (
	SELECT
		cp.rank,
		COALESCE(cp.current_xp, 0)        AS current_xp,
		COALESCE(cp.xp_for_next_rank, 0) AS xp_for_next_rank,
		COALESCE(cp.is_max_rank, FALSE)  AS is_max_rank,
		NULLIF(TRIM(cp.rank_name), '')   AS rank_name,
		NULLIF(TRIM(cp.rank_tier), '')   AS rank_tier
	FROM career_progression cp
	ORDER BY cp.recorded_at DESC
	LIMIT 1
),
latest_adornment AS (
	SELECT NULLIF(TRIM(cp.adornment_path), '') AS adornment_path
	FROM career_progression cp
	WHERE NULLIF(TRIM(cp.adornment_path), '') IS NOT NULL
	ORDER BY cp.recorded_at DESC
	LIMIT 1
)
SELECT
	latest.rank,
	latest.current_xp,
	latest.xp_for_next_rank,
	latest.is_max_rank,
	(
		SELECT NULLIF(TRIM(cp.spartan_id), '')
		FROM career_progression cp
		WHERE NULLIF(TRIM(cp.spartan_id), '') IS NOT NULL
		ORDER BY cp.recorded_at DESC
		LIMIT 1
	) AS spartan_id,
	latest.rank_name,
	latest.rank_tier,
	(
		SELECT NULLIF(TRIM(cp.banner_image_url), '')
		FROM career_progression cp
		WHERE NULLIF(TRIM(cp.banner_image_url), '') IS NOT NULL
		ORDER BY cp.recorded_at DESC
		LIMIT 1
	) AS banner_image_url,
	(
		SELECT NULLIF(TRIM(cp.emblem_image_url), '')
		FROM career_progression cp
		WHERE NULLIF(TRIM(cp.emblem_image_url), '') IS NOT NULL
		ORDER BY cp.recorded_at DESC
		LIMIT 1
	) AS emblem_image_url,
	(
		SELECT NULLIF(TRIM(cp.backdrop_image_url), '')
		FROM career_progression cp
		WHERE NULLIF(TRIM(cp.backdrop_image_url), '') IS NOT NULL
		ORDER BY cp.recorded_at DESC
		LIMIT 1
	) AS backdrop_image_url,
	(
		SELECT adornment_path
		FROM latest_adornment
	) AS adornment_path
FROM latest`

// Q26d : Home -- métadonnées du rang carrière courant depuis metadata.duckdb.
// Paramètre : ?1 = rank_id.
const Q26dHomeCareerRankMeta = `
SELECT
	NULLIF(TRIM(title_en), '') AS title_en,
	NULLIF(TRIM(title_fr), '') AS title_fr,
	COALESCE(
		NULLIF(TRIM(large_icon_path), ''),
		NULLIF(TRIM(icon_path), '')
	) AS image_path,
	NULLIF(TRIM(adornment_icon_path), '') AS adornment_path
FROM career_ranks
WHERE rank_id = ?
LIMIT 1`

// Q26e : Home -- meilleur rating historique par type (CSR ou LUSR).
// Paramètre : ?1 = rating_type.
const Q26eHomeSkillPeakByType = `
SELECT
	msr.rating_value,
	NULLIF(TRIM(msr.tier_label), '') AS tier_label,
	NULLIF(TRIM(msr.tier), '') AS tier,
	COALESCE(msr.sub_tier, 0) AS sub_tier
FROM match_skill_rank msr
LEFT JOIN shared.match_registry mr ON mr.match_id = msr.match_id
WHERE UPPER(
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
	END
) = UPPER(?)
	AND msr.rating_value IS NOT NULL
ORDER BY
	msr.rating_value DESC,
	COALESCE(msr.updated_at, msr.start_time, msr.created_at) DESC,
	COALESCE(msr.sub_tier, 0) DESC,
	msr.match_id DESC
LIMIT 1`

// Q26g : Home — 3 dernières playlists distinctes jouées avec leur dernier rang compétitif.
// Paramètre : ?1 = xuid du joueur.
// Retourne (playlist_name, is_ranked, rating_type, rating_value, tier, tier_fr, sub_tier, tier_label).
// rating_* sont NULL pour les playlists sans rang calculé.
const Q26gHomePlaylistRanks = `
WITH recent_playlists AS (
	SELECT
		r.playlist_id,
		COALESCE(MAX(r.playlist_name_fr), MAX(r.playlist_name), '') AS playlist_name,
		MAX(CASE
			WHEN COALESCE(r.is_ranked, FALSE)
				OR STRPOS(LOWER(COALESCE(r.playlist_name, '')), 'ranked') > 0
				OR STRPOS(LOWER(COALESCE(r.pair_name, '')), 'ranked') > 0
			THEN 1 ELSE 0
		END) > 0                                                     AS is_ranked,
		MAX(r.start_time)                                            AS last_played
	FROM shared.match_participants mp
	JOIN shared.match_registry r ON r.match_id = mp.match_id
	WHERE mp.xuid = ?
	  AND NULLIF(TRIM(COALESCE(r.playlist_id, '')), '') IS NOT NULL
	GROUP BY r.playlist_id
	ORDER BY MAX(r.start_time) DESC
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
)
SELECT
	rp.playlist_name,
	rp.is_ranked,
	ls.rating_type,
	ls.rating_value,
	ls.tier,
	ls.tier_fr,
	ls.sub_tier,
	ls.tier_label
FROM recent_playlists rp
LEFT JOIN last_skill ls ON ls.playlist_id = rp.playlist_id AND ls.rn = 1
ORDER BY rp.last_played DESC`

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

// Q37 : Médias — fichiers actifs paginés depuis media_files + associations + match_registry.
// Remplacé par BuildQ37MediaQuery pour les filtres/tri dynamiques.
// Conservé pour compatibilité éventuelle.
const Q37MediaFiles = `
SELECT
    mf.file_path,
    mf.file_name,
    mf.kind,
    mf.thumbnail_path,
    mf.capture_end_utc,
    mma.match_id,
    mma.match_start_time,
    COALESCE(mf.liked, FALSE) AS liked,
    ` + q37MediaMapLabelExpr + ` AS map_name,
    ` + q37MediaModeLabelExpr + ` AS mode_name
` + q37MediaFromClause + `
WHERE mf.status = 'active'
ORDER BY mf.mtime DESC
LIMIT ? OFFSET ?`

// Q37Count : Médias — nombre total de fichiers actifs.
const Q37MediaCount = `SELECT COUNT(*) FROM media_files WHERE status = 'active'`

const q37LegacyMediaFromClause = `FROM media_files mf
LEFT JOIN media_match_associations mma ON mf.file_path = mma.media_path
LEFT JOIN shared.match_registry mr ON mma.match_id = mr.match_id`

const q37SharedSocialFromClause = `FROM media_files mf
LEFT JOIN media_match_associations mma ON mf.id = mma.media_file_id
LEFT JOIN shared.match_registry mr ON mma.match_id = mr.match_id`

const q37MediaFromClause = q37LegacyMediaFromClause

const q37MediaMapLabelExpr = `NULLIF(TRIM(COALESCE(mr.map_name_fr, mr.map_name, '')), '')`

const q37MediaModeLabelExpr = `NULLIF(TRIM(regexp_replace(regexp_replace(regexp_replace(COALESCE(mr.pair_name_fr, mr.pair_name, ''), ' on .+$', '', 'i'), '\s*-\s*Forge\b', '', 'i'), '\s*-\s*Ranked\b', '', 'i')), '')`

type mediaWhereConfig struct {
	includeMapFilter  bool
	includeModeFilter bool
}

type mediaQueryConfig struct {
	playerSlug string
}

func (cfg mediaQueryConfig) useSharedSocialSchema() bool {
	return cfg.playerSlug != ""
}

func (cfg mediaQueryConfig) fromClause() string {
	if cfg.useSharedSocialSchema() {
		return q37SharedSocialFromClause
	}
	return q37LegacyMediaFromClause
}

func (cfg mediaQueryConfig) baseWhereClause() ([]string, []any) {
	if cfg.useSharedSocialSchema() {
		return []string{"mf.player_slug = ?"}, []any{cfg.playerSlug}
	}
	return []string{"mf.status = 'active'"}, nil
}

func (cfg mediaQueryConfig) matchStartExpr() string {
	if cfg.useSharedSocialSchema() {
		return "mr.start_time"
	}
	return "mma.match_start_time"
}

func (cfg mediaQueryConfig) timeOrderExpr() string {
	if cfg.useSharedSocialSchema() {
		return "COALESCE(mf.capture_end_utc, mf.updated_at, mf.created_at)"
	}
	return "COALESCE(mf.capture_end_utc, mf.mtime)"
}

func buildQ37MediaWhereClause(
	f domain.MediaFilters,
	whereCfg mediaWhereConfig,
	queryCfg mediaQueryConfig,
) (string, []any) {
	where, args := queryCfg.baseWhereClause()

	if f.KindFilter != "" {
		where = append(where, "mf.kind = ?")
		args = append(args, f.KindFilter)
	}
	if f.LikedOnly {
		where = append(where, "COALESCE(mf.liked, FALSE) = TRUE")
	}
	if whereCfg.includeMapFilter && f.MapFilter != "" {
		where = append(where, q37MediaMapLabelExpr+" ILIKE ?")
		args = append(args, "%"+f.MapFilter+"%")
	}
	if whereCfg.includeModeFilter && f.ModeFilter != "" {
		where = append(where, q37MediaModeLabelExpr+" ILIKE ?")
		args = append(args, "%"+f.ModeFilter+"%")
	}

	return "WHERE " + strings.Join(where, " AND "), args
}

// BuildQ37MediaQuery construit dynamiquement la query médias avec filtres et tri.
// Retourne la query SQL et les args à passer (dans l'ordre : filtres..., limit, offset).
func BuildQ37MediaQuery(f domain.MediaFilters, limit, offset int) (string, []any) {
	return buildQ37MediaQuery(f, limit, offset, mediaQueryConfig{})
}

func buildQ37MediaQuery(
	f domain.MediaFilters,
	limit, offset int,
	queryCfg mediaQueryConfig,
) (string, []any) {
	whereClause, args := buildQ37MediaWhereClause(f, mediaWhereConfig{
		includeMapFilter:  true,
		includeModeFilter: true,
	}, queryCfg)

	orderBy := queryCfg.timeOrderExpr() + " DESC"
	switch f.Sort {
	case "date_asc":
		orderBy = queryCfg.timeOrderExpr() + " ASC"
	case "map_asc":
		orderBy = "COALESCE(" + q37MediaMapLabelExpr + ", '') ASC, " + queryCfg.timeOrderExpr() + " DESC"
	case "mode_asc":
		orderBy = "COALESCE(" + q37MediaModeLabelExpr + ", '') ASC, " + queryCfg.timeOrderExpr() + " DESC"
	}

	q := `SELECT
    mf.file_path,
    mf.file_name,
    mf.kind,
    mf.thumbnail_path,
    mf.capture_end_utc,
    mma.match_id,
    ` + queryCfg.matchStartExpr() + ` AS match_start_time,
    COALESCE(mf.liked, FALSE) AS liked,
    ` + q37MediaMapLabelExpr + ` AS map_name,
    ` + q37MediaModeLabelExpr + ` AS mode_name
` + queryCfg.fromClause() + `
` + whereClause + `
ORDER BY ` + orderBy + `
LIMIT ? OFFSET ?`

	args = append(args, limit, offset)
	return q, args
}

// BuildQ37MediaCountQuery construit la query COUNT correspondante aux filtres actifs.
func BuildQ37MediaCountQuery(f domain.MediaFilters) (string, []any) {
	return buildQ37MediaCountQuery(f, mediaQueryConfig{})
}

func buildQ37MediaCountQuery(f domain.MediaFilters, queryCfg mediaQueryConfig) (string, []any) {
	whereClause, args := buildQ37MediaWhereClause(f, mediaWhereConfig{
		includeMapFilter:  true,
		includeModeFilter: true,
	}, queryCfg)

	q := `SELECT COUNT(*)
` + queryCfg.fromClause() + `
` + whereClause

	return q, args
}

// BuildQ37MediaMapOptionsQuery retourne les cartes distinctes disponibles pour la galerie.
func BuildQ37MediaMapOptionsQuery(f domain.MediaFilters) (string, []any) {
	return buildQ37MediaMapOptionsQuery(f, mediaQueryConfig{})
}

func buildQ37MediaMapOptionsQuery(f domain.MediaFilters, queryCfg mediaQueryConfig) (string, []any) {
	whereClause, args := buildQ37MediaWhereClause(f, mediaWhereConfig{
		includeMapFilter:  false,
		includeModeFilter: true,
	}, queryCfg)

	q := `SELECT DISTINCT ` + q37MediaMapLabelExpr + ` AS label
` + queryCfg.fromClause() + `
` + whereClause + `
  AND ` + q37MediaMapLabelExpr + ` IS NOT NULL
ORDER BY label ASC`

	return q, args
}

// BuildQ37MediaModeOptionsQuery retourne les modes normalisés distincts disponibles.
func BuildQ37MediaModeOptionsQuery(f domain.MediaFilters) (string, []any) {
	return buildQ37MediaModeOptionsQuery(f, mediaQueryConfig{})
}

func buildQ37MediaModeOptionsQuery(f domain.MediaFilters, queryCfg mediaQueryConfig) (string, []any) {
	whereClause, args := buildQ37MediaWhereClause(f, mediaWhereConfig{
		includeMapFilter:  true,
		includeModeFilter: false,
	}, queryCfg)

	q := `SELECT DISTINCT ` + q37MediaModeLabelExpr + ` AS label
` + queryCfg.fromClause() + `
` + whereClause + `
  AND ` + q37MediaModeLabelExpr + ` IS NOT NULL
ORDER BY label ASC`

	return q, args
}
