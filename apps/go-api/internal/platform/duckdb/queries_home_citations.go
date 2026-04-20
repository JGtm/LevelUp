// Package duckdb — queries_home_citations.go : requêtes page Home, citations et médias.
package duckdb

import "levelup/go-api/internal/domain"

// Q26 : Home — matchs recents d un joueur avec KPIs pour le hero card.
// Parametre : ?1 = xuid du joueur.
// Retourne les 200 derniers matchs (hero card, highlights, recent matches, sessions).
const Q26HomeMatches = `
SELECT
    mp.match_id,
    r.start_time,
    COALESCE(r.map_name, '')                                AS map_name,
    COALESCE(r.map_name_fr, r.map_name, '')                 AS map_name_fr,
    COALESCE(r.pair_name, '')                               AS pair_name,
    COALESCE(r.pair_name_fr, r.pair_name, '')               AS pair_name_fr,
    COALESCE(r.playlist_name, '')                           AS playlist_name,
    COALESCE(r.is_firefight, FALSE)                         AS is_firefight,
    COALESCE(r.is_ranked, FALSE)                            AS is_ranked,
    pme.session_label,
    COALESCE(pme.is_with_friends, FALSE)                    AS is_with_friends,
    COALESCE(mp.outcome, 0)                                 AS outcome,
    COALESCE(mp.kills, 0)                                   AS kills,
    COALESCE(mp.deaths, 0)                                  AS deaths,
    COALESCE(mp.assists, 0)                                 AS assists,
    mp.kda,
    CASE WHEN COALESCE(mp.deaths, 0) > 0
         THEN CAST(COALESCE(mp.kills, 0) AS DOUBLE) / CAST(mp.deaths AS DOUBLE)
         ELSE CAST(COALESCE(mp.kills, 0) AS DOUBLE) END     AS ratio,
    mp.accuracy,
    mp.time_played_seconds
FROM shared.match_participants mp
JOIN shared.v_match_full r ON r.match_id = mp.match_id
LEFT JOIN player_match_enrichment pme ON pme.match_id = mp.match_id
WHERE mp.xuid = ?
ORDER BY r.start_time DESC
LIMIT 200`

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
    mr.map_name,
    mr.pair_name AS mode_name
FROM media_files mf
LEFT JOIN media_match_associations mma ON mf.file_path = mma.media_path
LEFT JOIN shared.match_registry mr ON mma.match_id = mr.match_id
WHERE mf.status = 'active'
ORDER BY mf.mtime DESC
LIMIT ? OFFSET ?`

// Q37Count : Médias — nombre total de fichiers actifs.
const Q37MediaCount = `SELECT COUNT(*) FROM media_files WHERE status = 'active'`

// BuildQ37MediaQuery construit dynamiquement la query médias avec filtres et tri.
// Retourne la query SQL et les args à passer (dans l'ordre : filtres..., limit, offset).
func BuildQ37MediaQuery(f domain.MediaFilters, limit, offset int) (string, []any) {
	where := []string{"mf.status = 'active'"}
	args := []any{}

	if f.KindFilter != "" {
		where = append(where, "mf.kind = ?")
		args = append(args, f.KindFilter)
	}
	if f.LikedOnly {
		where = append(where, "COALESCE(mf.liked, FALSE) = TRUE")
	}
	if f.MapFilter != "" {
		where = append(where, "mr.map_name ILIKE ?")
		args = append(args, "%"+f.MapFilter+"%")
	}
	if f.ModeFilter != "" {
		where = append(where, "mr.pair_name ILIKE ?")
		args = append(args, "%"+f.ModeFilter+"%")
	}

	orderBy := "mf.mtime DESC"
	switch f.Sort {
	case "date_asc":
		orderBy = "mf.mtime ASC"
	case "map_asc":
		orderBy = "COALESCE(mr.map_name, '') ASC, mf.mtime DESC"
	case "mode_asc":
		orderBy = "COALESCE(mr.pair_name, '') ASC, mf.mtime DESC"
	}

	whereClause := ""
	for i, w := range where {
		if i == 0 {
			whereClause = "WHERE " + w
		} else {
			whereClause += " AND " + w
		}
	}

	q := `SELECT
    mf.file_path,
    mf.file_name,
    mf.kind,
    mf.thumbnail_path,
    mf.capture_end_utc,
    mma.match_id,
    mma.match_start_time,
    COALESCE(mf.liked, FALSE) AS liked,
    mr.map_name,
    mr.pair_name AS mode_name
FROM media_files mf
LEFT JOIN media_match_associations mma ON mf.file_path = mma.media_path
LEFT JOIN shared.match_registry mr ON mma.match_id = mr.match_id
` + whereClause + `
ORDER BY ` + orderBy + `
LIMIT ? OFFSET ?`

	args = append(args, limit, offset)
	return q, args
}

// BuildQ37MediaCountQuery construit la query COUNT correspondante aux filtres actifs.
func BuildQ37MediaCountQuery(f domain.MediaFilters) (string, []any) {
	where := []string{"mf.status = 'active'"}
	args := []any{}

	if f.KindFilter != "" {
		where = append(where, "mf.kind = ?")
		args = append(args, f.KindFilter)
	}
	if f.LikedOnly {
		where = append(where, "COALESCE(mf.liked, FALSE) = TRUE")
	}
	if f.MapFilter != "" {
		where = append(where, "mr.map_name ILIKE ?")
		args = append(args, "%"+f.MapFilter+"%")
	}
	if f.ModeFilter != "" {
		where = append(where, "mr.pair_name ILIKE ?")
		args = append(args, "%"+f.ModeFilter+"%")
	}

	whereClause := ""
	for i, w := range where {
		if i == 0 {
			whereClause = "WHERE " + w
		} else {
			whereClause += " AND " + w
		}
	}

	q := `SELECT COUNT(*)
FROM media_files mf
LEFT JOIN media_match_associations mma ON mf.file_path = mma.media_path
LEFT JOIN shared.match_registry mr ON mma.match_id = mr.match_id
` + whereClause

	return q, args
}
