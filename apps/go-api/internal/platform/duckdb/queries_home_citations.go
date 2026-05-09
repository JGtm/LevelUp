// Package duckdb â€” queries_home_citations.go : requÃªtes page Home, citations et mÃ©dias.
package duckdb

import (
	"regexp"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite"
)

// Q26 : Home â€” matchs d un joueur avec KPIs pour le hero card.
// Parametre : ?1 = xuid du joueur.
// LIMIT 150 : couvre hero card, highlights, recent matches et summaries de sessions rÃ©centes.
// Le total rÃ©el vient de Q26bCountPlayerMatches (requÃªte sÃ©parÃ©e).
// Les sessions complÃ¨tes sont chargÃ©es indÃ©pendamment via Q27HomeSessions.
const Q26HomeMatches = `
WITH perfect AS (
    SELECT match_id, COALESCE(SUM(count), 0) AS perfect_kills
    FROM shared.medals_earned
    WHERE xuid = ? AND medal_name_id = 1512363953
    GROUP BY match_id
)
SELECT
    mp.match_id,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
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
    COALESCE(perfect.perfect_kills, 0)                      AS perfect_kills,
    mp.max_killing_spree                                    AS max_killing_spree
FROM shared.match_participants mp
JOIN shared.match_registry r ON r.match_id = mp.match_id
LEFT JOIN player_match_enrichment pme ON pme.match_id = mp.match_id
LEFT JOIN match_skill_rank msr ON msr.match_id = mp.match_id
LEFT JOIN perfect ON perfect.match_id = mp.match_id
WHERE mp.xuid = ?
ORDER BY COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') DESC
LIMIT 150`

// Q26h : Home â€” mÃ©dailles par match pour un joueur, lots de match_id.
// ParamÃ¨tres : ?1 = xuid. Les match_id sont injectÃ©s dynamiquement via IN (%s).
// RequÃªte sur pdb.Player (shared attachÃ©) ; labels rÃ©solus ensuite via metadata.
const Q26hMatchMedalsTemplate = `
SELECT
    me.match_id,
    me.medal_name_id,
    COALESCE(me.count, 1) AS count
FROM shared.medals_earned me
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
SELECT COUNT(*) FROM shared.match_participants WHERE xuid = ?`

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

// Q26e : Home -- meilleur rating historique par type (CSR ou LUSR).
// ParamÃ¨tre : ?1 = rating_type.
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

// Q26g : Home â€” 3 derniÃ¨res playlists distinctes jouÃ©es avec leur dernier rang compÃ©titif.
// ParamÃ¨tre : ?1 = xuid du joueur.
// Retourne (playlist_id, playlist_name, is_ranked, rating_type, rating_value, tier, tier_fr, sub_tier, tier_label).
// playlist_name_fr est rÃ©solu en Go depuis asset_translations (mÃªme source que les tuiles de matchs).
// rating_* sont NULL pour les playlists sans rang calculÃ©.
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
	ls.tier_label
FROM recent_playlists rp
LEFT JOIN last_skill ls ON ls.playlist_id = rp.playlist_id AND ls.rn = 1
ORDER BY rp.last_played DESC`

// Q27 : Home â€” sessions depuis player_match_enrichment.
// Pas de parametre (les donnees sont dans la DB joueur).
// Retourne les matchs avec un label de session pour le resumÃ© solo/escouade.
const Q27HomeSessions = `
SELECT
    pme.match_id,
    pme.session_id,
    pme.session_label,
    COALESCE(pme.is_with_friends, FALSE)    AS is_with_friends,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time
FROM player_match_enrichment pme
LEFT JOIN shared.match_registry r ON r.match_id = pme.match_id
WHERE pme.session_label IS NOT NULL
ORDER BY COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') DESC`

// Q28 : Home â€” medias recents depuis media_files + media_match_associations.
// Parametre : ?1 = LIMIT (nombre de medias).
// Retourne uniquement les medias actifs, triÃ©s par date de modification desc.
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

// Q26k : Home â€” arme favorite (kills totaux) du joueur toutes armes confondues.
// ParamÃ¨tre : ?1 = xuid.
// RequÃªte sur pdb.Player (shared attachÃ©). Label rÃ©solu ensuite via pdb.Metadata.
const Q26kFavoriteWeapon = `
SELECT
    wk.effective_weapon_id AS weapon_id,
    COUNT(*)               AS total_kills
FROM shared.v_weapon_kills wk
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
    tier_targets
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

// Q37 : MÃ©dias â€” fichiers actifs paginÃ©s depuis media_files + associations + match_registry.
// RemplacÃ© par BuildQ37MediaQuery pour les filtres/tri dynamiques.
// ConservÃ© pour compatibilitÃ© Ã©ventuelle.
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
    ` + q37MediaModeLabelExpr + ` AS mode_name,
    mr.map_id
` + q37MediaFromClause + `
WHERE mf.status = 'active'
ORDER BY mf.mtime DESC
LIMIT ? OFFSET ?`

// Q37Count : MÃ©dias â€” nombre total de fichiers actifs.
const Q37MediaCount = `SELECT COUNT(*) FROM media_files WHERE status = 'active'`

const q37LegacyMediaFromClause = `FROM media_files mf
LEFT JOIN media_match_associations mma ON mf.file_path = mma.media_path
LEFT JOIN shared.match_registry mr ON mma.match_id = mr.match_id`

const q37SharedSocialFromClause = `FROM media_files mf
LEFT JOIN media_match_associations mma ON mf.id = mma.media_file_id
LEFT JOIN shared.match_registry mr ON mma.match_id = mr.match_id`

const q37MediaFromClause = q37LegacyMediaFromClause

// q37MediaMapLabelExpr normalise le nom de carte pour grouper les variantes
// (ex: "Recharge v3" â†’ "Recharge"). Strip conservateur :
//   - ` v\d+$`           : versions Forge ("Recharge v3" â†’ "Recharge")
//   - ` - Forge.*$`      : variante Forge avec dash ("Recharge - Forge" â†’ "Recharge")
//   - ` - Ranked.*$`     : variante Ranked avec dash
//
// On ne strip PAS les suffixes ambigus comme " Annex", " Beta", `:`, `-` gÃ©nÃ©riques
// (sinon "Forge: Argyle" deviendrait "Forge", "Recharge Annex" deviendrait "Recharge").
const q37MediaMapLabelExpr = `NULLIF(TRIM(regexp_replace(regexp_replace(regexp_replace(COALESCE(mr.map_name_fr, mr.map_name, ''), '\s+v\d+$', '', 'i'), '\s*-\s*Forge.*$', '', 'i'), '\s*-\s*Ranked.*$', '', 'i')), '')`

// q37MediaModeLabelExpr extrait le mode "parent" depuis pair_name :
//
//   - Si le pair_name contient ":" : on garde UNIQUEMENT le prÃ©fixe avant
//     ("Arena:Slayer on Bazaar" -> "Arena", "Super Fiesta:Slayer - Forge" -> "Super Fiesta",
//     "Community:Team Slayer" -> "Community"). Les sous-modes (Slayer/CTF/KOTH)
//     sont considÃ©rÃ©s comme des sous-genres de la grande catÃ©gorie de matchmaking.
//   - Sinon : strip suffixes carte/Forge/Ranked et garde le label canonique
//     (ex: "Husky Raid" -> "Husky Raid", "Slayer on Bazaar" -> "Slayer").
const q37MediaModeLabelExpr = `NULLIF(TRIM(
	CASE
		WHEN POSITION(':' IN COALESCE(mr.pair_name_fr, mr.pair_name, '')) > 0
		THEN regexp_replace(COALESCE(mr.pair_name_fr, mr.pair_name, ''), ':.*$', '', '')
		ELSE regexp_replace(regexp_replace(regexp_replace(COALESCE(mr.pair_name_fr, mr.pair_name, ''), ' on .+$', '', 'i'), '\s*-\s*Forge\b.*$', '', 'i'), '\s*-\s*Ranked\b.*$', '', 'i')
	END
), '')`

type mediaWhereConfig struct {
	includeMapFilter      bool
	includeModeFilter     bool
	includePlaylistFilter bool
}

// q37MediaPlaylistLabelExpr renvoie le label playlist (FR si dispo, EN sinon).
const q37MediaPlaylistLabelExpr = `NULLIF(TRIM(COALESCE(mr.playlist_name_fr, mr.playlist_name, '')), '')`

// mediaKindEquivalents retourne les valeurs DB qui doivent matcher un filtre
// de type donnÃ©, en couvrant Ã  la fois la convention legacy ("clip"/"screenshot")
// et la nouvelle ("video"/"image"). Les mÃ©dias indexÃ©s par les anciennes
// versions ont l'une, les nouveaux uploads ont l'autre â€” sans cette translation
// le filtre type ne retourne 0 rÃ©sultat.
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

// Strips appliquÃ©s cÃ´tÃ© Go pour normaliser une valeur de filtre map.
// Doit rester en miroir de q37MediaMapLabelExpr (sinon filtre "Recharge v3"
// ne matcherait pas le label "Recharge" dÃ©jÃ  normalisÃ© cÃ´tÃ© SQL).
var (
	mediaMapForgeSuffixRe   = regexp.MustCompile(`(?i)\s*-\s*Forge.*$`)
	mediaMapRankedSuffixRe  = regexp.MustCompile(`(?i)\s*-\s*Ranked.*$`)
	mediaMapVersionSuffixRe = regexp.MustCompile(`(?i)\s+v\d+$`)
)

// normalizeMediaMapName strippe les suffixes de variante pour grouper
// "Recharge v3" / "Recharge - Forge" / "Recharge" sous le mÃªme nom canonique.
// Conservatif : ne touche pas " Annex", `:`, etc. (cf. q37MediaMapLabelExpr).
func normalizeMediaMapName(s string) string {
	s = mediaMapForgeSuffixRe.ReplaceAllString(s, "")
	s = mediaMapRankedSuffixRe.ReplaceAllString(s, "")
	s = mediaMapVersionSuffixRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
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

// baseWhereClause renvoie les contraintes de base + le filtre de section (ownership).
//
//	"" (vide)   â†’ sources visibles : mine + teammate (pas de contrainte player_slug)
//	"mine"      â†’ uniquement player_slug courant
//	"teammate"  â†’ uniquement les autres (player_slug != courant)
//
// En schÃ©ma legacy (pas de player_slug), seul "mine" et "" donnent des rÃ©sultats ;
// "teammate" force WHERE FALSE pour cohÃ©rence (rien Ã  montrer).
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
		// Tous auteurs â€” pas de contrainte sur player_slug
		return nil, nil
	}
}

func (cfg mediaQueryConfig) matchStartExpr() string {
	if cfg.useSharedSocialSchema() {
		return "COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC')"
	}
	return "mma.match_start_time"
}

func (cfg mediaQueryConfig) timeOrderExpr() string {
	if cfg.useSharedSocialSchema() {
		return "COALESCE(mf.capture_end_utc, mf.mtime, mf.indexed_at)"
	}
	return "COALESCE(mf.capture_end_utc, mf.mtime)"
}

// groupOrderExpr retourne l'expression de tri primaire pour grouper les mÃ©dias.
// Le tri secondaire (date / map / mode) reste appliquÃ© aprÃ¨s. Retourne ("", "")
// si le groupement est inconnu ou ne s'applique pas au schÃ©ma courant.
func (cfg mediaQueryConfig) groupOrderExpr(groupBy string) (expr, direction string) {
	switch groupBy {
	case "owner":
		if cfg.useSharedSocialSchema() {
			return "mf.player_slug", "ASC"
		}
		return "", ""
	case "map":
		return "COALESCE(" + q37MediaMapLabelExpr + ", '~zzz')", "ASC"
	case "mode":
		return "COALESCE(" + q37MediaModeLabelExpr + ", '~zzz')", "ASC"
	case "session":
		// Le groupement par session est calculÃ© cÃ´tÃ© frontend (proximitÃ© temporelle) ;
		// pour le ORDER BY backend, on s'aligne juste sur la date pour que les sessions
		// apparaissent contiguÃ«s. Pas de troncature SQL â€” la heuristique cÃ´tÃ© UI fait
		// foi.
		return cfg.timeOrderExpr(), "DESC"
	case "liked":
		return "COALESCE(mf.liked, FALSE)", "DESC"
	}
	return "", ""
}

func buildQ37MediaWhereClause(
	f domain.MediaFilters,
	whereCfg mediaWhereConfig,
	queryCfg mediaQueryConfig,
) (string, []any) {
	// AuthorSlugs prend le pas sur SectionFilter quand non vide (whitelist
	// explicite plus restrictive que mine/teammate/all).
	var where []string
	var args []any
	if len(f.AuthorSlugs) > 0 && queryCfg.useSharedSocialSchema() {
		placeholders := make([]string, len(f.AuthorSlugs))
		for i, slug := range f.AuthorSlugs {
			placeholders[i] = "?"
			args = append(args, slug)
		}
		where = []string{"mf.player_slug IN (" + strings.Join(placeholders, ",") + ")"}
	} else {
		where, args = queryCfg.baseWhereClause(f.SectionFilter)
		if len(where) == 0 {
			where = []string{"TRUE"}
		}
	}

	if f.KindFilter != "" {
		// Compat schÃ©mas : legacy stocke "clip"/"screenshot", nouveau stocke "video"/"image".
		// Le frontend envoie les valeurs legacy ("clip"/"screenshot") historiquement.
		// On accepte les deux conventions pour matcher quelle que soit l'origine de la ligne.
		equivalents := mediaKindEquivalents(f.KindFilter)
		placeholders := make([]string, len(equivalents))
		for i, eq := range equivalents {
			placeholders[i] = "?"
			args = append(args, eq)
		}
		where = append(where, "mf.kind IN ("+strings.Join(placeholders, ",")+")")
	}
	if f.LikedOnly {
		where = append(where, "COALESCE(mf.liked, FALSE) = TRUE")
	}
	if f.UnassignedOnly {
		where = append(where, "mma.match_id IS NULL")
	}
	if whereCfg.includePlaylistFilter && f.PlaylistFilter != "" {
		// Match flexible : playlist_id (UUID stable, value du dropdown) OU label brut.
		where = append(where, "(mr.playlist_id = ? OR LOWER("+q37MediaPlaylistLabelExpr+") = LOWER(?))")
		args = append(args, f.PlaylistFilter, f.PlaylistFilter)
	}
	if whereCfg.includeMapFilter && f.MapFilter != "" {
		// MapFilter peut Ãªtre un map_id (cas standard, value du dropdown) OU un
		// label brut (fallback pour mÃ©dias sans map_id, ou requÃªte manuelle).
		// On matche les deux pour rester compatible. Sans cette double tentative,
		// "Altitude" FR ne matcherait jamais "High Ground" raw EN.
		where = append(where, "(mr.map_id = ? OR LOWER("+q37MediaMapLabelExpr+") = LOWER(?))")
		args = append(args, f.MapFilter, normalizeMediaMapName(f.MapFilter))
	}
	if whereCfg.includeModeFilter && f.ModeFilter != "" {
		// 2 formats acceptÃ©s :
		//   "Assassin"        â†’ catÃ©gorie entiÃ¨re (reverse-mapping vers prÃ©fixes pair_name)
		//   "Assassin/Slayer" â†’ catÃ©gorie + sous-mode normalisÃ© (filtre granulaire)
		//
		// Pour le format avec sous-mode, on conserve le filtre catÃ©gorie ET on
		// ajoute un AND sur le sous-mode normalisÃ© (extrait via q37MediaModeLabelExpr).
		// "Other" = NOT IN les prÃ©fixes connus.
		category, submode, hasSubmode := strings.Cut(f.ModeFilter, "/")
		prefixes := halo_infinite.PairNamePrefixesForCategory(category)
		if len(prefixes) > 0 {
			parts := make([]string, 0, len(prefixes)*2)
			for _, p := range prefixes {
				parts = append(parts, "LOWER(mr.pair_name) LIKE LOWER(?)")
				args = append(args, p+":%")
				parts = append(parts, "LOWER(mr.pair_name) = LOWER(?)")
				args = append(args, p)
			}
			where = append(where, "("+strings.Join(parts, " OR ")+")")
		} else if category == halo_infinite.ModeCategoryOther {
			knownParts := []string{}
			for _, p := range halo_infinite.AllKnownPairNamePrefixes() {
				knownParts = append(knownParts, "LOWER(mr.pair_name) LIKE LOWER(?)")
				args = append(args, p+":%")
				knownParts = append(knownParts, "LOWER(mr.pair_name) = LOWER(?)")
				args = append(args, p)
			}
			if len(knownParts) > 0 {
				where = append(where, "NOT ("+strings.Join(knownParts, " OR ")+")")
			}
		}
		if hasSubmode && strings.TrimSpace(submode) != "" {
			where = append(where, "LOWER("+q37MediaModeLabelExpr+") = LOWER(?)")
			args = append(args, strings.TrimSpace(submode))
		}
	}

	return "WHERE " + strings.Join(where, " AND "), args
}

// BuildQ37MediaQuery construit dynamiquement la query mÃ©dias avec filtres et tri.
// Retourne la query SQL et les args Ã  passer (dans l'ordre : filtres..., limit, offset).
func BuildQ37MediaQuery(f domain.MediaFilters, limit, offset int) (string, []any) {
	return buildQ37MediaQuery(f, limit, offset, mediaQueryConfig{})
}

func buildQ37MediaQuery(
	f domain.MediaFilters,
	limit, offset int,
	queryCfg mediaQueryConfig,
) (string, []any) {
	whereClause, args := buildQ37MediaWhereClause(f, mediaWhereConfig{
		includeMapFilter:      true,
		includeModeFilter:     true,
		includePlaylistFilter: true,
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

	if groupExpr, groupDir := queryCfg.groupOrderExpr(f.GroupBy); groupExpr != "" {
		orderBy = groupExpr + " " + groupDir + ", " + orderBy
	}

	playerSlugExpr := "NULL"
	if queryCfg.useSharedSocialSchema() {
		playerSlugExpr = "mf.player_slug"
	}

	// CRITIQUE : un mÃ©dia physique = 1 ligne. Sans QUALIFY, le LEFT JOIN sur
	// media_match_associations duplique le mÃ©dia si plusieurs matchs sont
	// associÃ©s (cas rÃ©el : capture pendant une session de plusieurs matchs
	// proches). On garde l'association la plus pertinente :
	//   - en prioritÃ© une avec match (mr.start_time non null)
	//   - parmi celles-lÃ , la plus proche temporellement de capture_end_utc
	//   - sinon stable tiebreak sur match_id
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
    ` + q37MediaModeLabelExpr + ` AS mode_name,
    COALESCE(mr.pair_name, '') AS pair_name_raw,
    mr.map_id,
    ` + playerSlugExpr + ` AS player_slug
` + queryCfg.fromClause() + `
` + whereClause + `
QUALIFY ROW_NUMBER() OVER (
    PARTITION BY mf.file_path
    ORDER BY
        CASE WHEN mr.start_time IS NULL THEN 1 ELSE 0 END,
        ABS(EXTRACT(EPOCH FROM (COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC') - mf.capture_end_utc))) ASC NULLS LAST,
        COALESCE(mma.match_id, '')
) = 1
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
		includeMapFilter:      true,
		includeModeFilter:     true,
		includePlaylistFilter: true,
	}, queryCfg)

	// COUNT(DISTINCT mf.file_path) car le LEFT JOIN duplique les mÃ©dias avec
	// plusieurs associations match. Sinon la pagination renvoie X*N pages au
	// lieu de X (oÃ¹ N = nombre moyen d'associations par mÃ©dia).
	q := `SELECT COUNT(DISTINCT mf.file_path)
` + queryCfg.fromClause() + `
` + whereClause

	return q, args
}

// BuildQ37MediaMapOptionsQuery retourne les cartes distinctes disponibles pour la galerie.
func BuildQ37MediaMapOptionsQuery(f domain.MediaFilters) (string, []any) {
	return buildQ37MediaMapOptionsQuery(f, mediaQueryConfig{})
}

func buildQ37MediaMapOptionsQuery(f domain.MediaFilters, queryCfg mediaQueryConfig) (string, []any) {
	// Cartes restreintes par playlist + mode courants (pas par carte elle-mÃªme)
	whereClause, args := buildQ37MediaWhereClause(f, mediaWhereConfig{
		includeMapFilter:      false,
		includeModeFilter:     true,
		includePlaylistFilter: true,
	}, queryCfg)

	// Retourne (map_id, label_raw) pour permettre l'enrichissement FR via asset_translations.
	q := `SELECT DISTINCT COALESCE(mr.map_id, '') AS map_id, ` + q37MediaMapLabelExpr + ` AS label
` + queryCfg.fromClause() + `
` + whereClause + `
  AND ` + q37MediaMapLabelExpr + ` IS NOT NULL
ORDER BY label ASC`

	return q, args
}

// BuildQ37MediaModeOptionsQuery retourne les modes normalisÃ©s distincts disponibles.
func BuildQ37MediaModeOptionsQuery(f domain.MediaFilters) (string, []any) {
	return buildQ37MediaModeOptionsQuery(f, mediaQueryConfig{})
}

func buildQ37MediaModeOptionsQuery(f domain.MediaFilters, queryCfg mediaQueryConfig) (string, []any) {
	// Modes restreints par playlist + carte courantes (pas par mode lui-mÃªme)
	whereClause, args := buildQ37MediaWhereClause(f, mediaWhereConfig{
		includeMapFilter:      true,
		includeModeFilter:     false,
		includePlaylistFilter: true,
	}, queryCfg)

	// Retourne (pair_name_raw, label_normalisÃ©) pour permettre normalisation
	// canonique (NormalizeModeLabel) puis lookup FR (mode_name_tr) cÃ´tÃ© Go.
	q := `SELECT DISTINCT COALESCE(mr.pair_name, '') AS pair_name_raw, ` + q37MediaModeLabelExpr + ` AS label
` + queryCfg.fromClause() + `
` + whereClause + `
  AND ` + q37MediaModeLabelExpr + ` IS NOT NULL
ORDER BY label ASC`

	return q, args
}

// BuildQ37MediaPlaylistOptionsQuery retourne les playlists distinctes disponibles.
func BuildQ37MediaPlaylistOptionsQuery(f domain.MediaFilters) (string, []any) {
	return buildQ37MediaPlaylistOptionsQuery(f, mediaQueryConfig{})
}

func buildQ37MediaPlaylistOptionsQuery(f domain.MediaFilters, queryCfg mediaQueryConfig) (string, []any) {
	// Playlists restreintes par carte + mode courants (pas par playlist elle-mÃªme)
	whereClause, args := buildQ37MediaWhereClause(f, mediaWhereConfig{
		includeMapFilter:      true,
		includeModeFilter:     true,
		includePlaylistFilter: false,
	}, queryCfg)

	// Retourne (playlist_id, label_raw) pour Value=playlist_id stable + Label FR enrichi.
	q := `SELECT DISTINCT COALESCE(mr.playlist_id, '') AS playlist_id, ` + q37MediaPlaylistLabelExpr + ` AS label
` + queryCfg.fromClause() + `
` + whereClause + `
  AND ` + q37MediaPlaylistLabelExpr + ` IS NOT NULL
ORDER BY label ASC`

	return q, args
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
