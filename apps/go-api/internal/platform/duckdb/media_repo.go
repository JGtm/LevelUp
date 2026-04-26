// Package duckdb — media_repo.go : accès DB pour la galerie médias.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// MediaRepo implémente port.MediaRepository.
type MediaRepo struct {
	pdb *PlayerDB
}

// NewMediaRepo crée un MediaRepo pour un joueur.
func NewMediaRepo(pdb *PlayerDB) *MediaRepo {
	return &MediaRepo{pdb: pdb}
}

// socialDB retourne SharedSocial si disponible, sinon Player (fallback de transition).
func (r *MediaRepo) socialDB() *DB {
	if r.pdb.SharedSocial != nil {
		return r.pdb.SharedSocial
	}
	return r.pdb.Player
}

// LoadMediaFiles charge une page de fichiers médias avec filtres dynamiques (Q37).
func (r *MediaRepo) LoadMediaFiles(ctx context.Context, filters domain.MediaFilters, limit, offset int) ([]domain.MediaFileRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	filters = r.expandModeFilter(ctx, filters)
	q, args := buildQ37MediaQuery(filters, limit, offset, r.queryConfig())
	rows, err := r.socialDB().Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("LoadMediaFiles: %w", err)
	}
	defer rows.Close()

	var result []domain.MediaFileRow
	for rows.Next() {
		var row domain.MediaFileRow
		if err := rows.Scan(
			&row.FilePath,
			&row.FileName,
			&row.Kind,
			&row.ThumbnailPath,
			&row.CaptureEndUTC,
			&row.MatchID,
			&row.MatchStartTime,
			&row.Liked,
			&row.MapName,
			&row.ModeName,
			&row.MapID,
			&row.PlayerSlug,
		); err != nil {
			return nil, fmt.Errorf("LoadMediaFiles scan: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	r.enrichMediaMapTranslations(ctx, result)
	return result, nil
}

// CountMediaFiles retourne le nombre total de fichiers médias actifs selon les filtres.
func (r *MediaRepo) CountMediaFiles(ctx context.Context, filters domain.MediaFilters) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filters = r.expandModeFilter(ctx, filters)
	q, args := buildQ37MediaCountQuery(filters, r.queryConfig())
	var count int
	err := r.socialDB().QueryRow(ctx, q, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("CountMediaFiles: %w", err)
	}
	return count, nil
}

// LoadMediaFilterOptions retourne les valeurs distinctes des filtres carte/mode,
// avec libellés enrichis en FR (asset_translations + mode_name_tr de metadata.duckdb)
// et déduplication par libellé FR. Pour les modes, plusieurs raw EN qui se
// normalisent vers le même FR (ex: "Capture the Flag", "CTF - Ranked", "CTF on
// Bazaar" → "Capture du drapeau") sont fusionnés en une seule entrée.
func (r *MediaRepo) LoadMediaFilterOptions(ctx context.Context, filters domain.MediaFilters) (domain.MediaFilterOptions, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filters = r.expandModeFilter(ctx, filters)
	queryCfg := r.queryConfig()

	mapQuery, mapArgs := buildQ37MediaMapOptionsQuery(filters, queryCfg)
	mapPairs, err := r.loadMediaIDLabelPairs(ctx, mapQuery, mapArgs)
	if err != nil {
		return domain.MediaFilterOptions{}, fmt.Errorf("LoadMediaFilterOptions maps: %w", err)
	}
	maps := r.translateMapFilterOptions(ctx, mapPairs)

	modeQuery, modeArgs := buildQ37MediaModeOptionsQuery(filters, queryCfg)
	modePairs, err := r.loadMediaIDLabelPairs(ctx, modeQuery, modeArgs)
	if err != nil {
		return domain.MediaFilterOptions{}, fmt.Errorf("LoadMediaFilterOptions modes: %w", err)
	}
	modes := r.translateModeFilterOptions(ctx, modePairs)

	return domain.MediaFilterOptions{Maps: maps, Modes: modes}, nil
}

// CurrentPlayerSlug retourne le slug (== gamertag) du joueur dont on lit la galerie.
func (r *MediaRepo) CurrentPlayerSlug() string {
	if r.pdb == nil {
		return ""
	}
	return r.pdb.Gamertag
}

// LoadMatchCandidatesForMedia retourne les matchs du joueur courant dans la
// fenêtre temporelle [capture_start - window, capture_start + window].
// Inclut les KPIs du joueur pour aider à reconnaître le bon match.
// Si capture_start_utc est nul → fallback mtime, sinon liste vide.
func (r *MediaRepo) LoadMatchCandidatesForMedia(ctx context.Context, filePath string, windowMinutes int) (domain.MediaMatchCandidatesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if windowMinutes <= 0 {
		windowMinutes = 15
	}

	// Lire capture_start_utc + association actuelle du média.
	// Match flexible : soit file_path exact (DB absolute), soit file_name
	// (basename — pour quand le frontend envoie l'URL transformée et qu'on
	// reçoit ".../foo.mp4" au lieu du chemin DB original).
	basename := filepath.Base(filePath)
	var captureUTC sql.NullTime
	var currentMatchID sql.NullString
	err := r.socialDB().QueryRow(ctx, `
		SELECT
			COALESCE(mf.capture_start_utc, mf.capture_end_utc, mf.mtime) AS cap,
			mma.match_id
		FROM media_files mf
		LEFT JOIN media_match_associations mma ON mma.media_file_id = mf.id
		WHERE mf.file_path = ? OR mf.file_name = ?
		LIMIT 1
	`, filePath, basename).Scan(&captureUTC, &currentMatchID)
	if err != nil {
		return domain.MediaMatchCandidatesResponse{}, fmt.Errorf("LoadMatchCandidatesForMedia: lookup capture: %w", err)
	}

	resp := domain.MediaMatchCandidatesResponse{
		FilePath:      filePath,
		WindowMinutes: windowMinutes,
		Candidates:    []domain.MediaMatchCandidate{},
	}
	if !captureUTC.Valid {
		return resp, nil
	}
	cap := captureUTC.Time
	resp.CaptureUTC = &cap

	// Charger les matchs du joueur dans la fenêtre — JOIN sur match_participants
	// pour les KPIs. start_time de match_registry est en UTC (TIMESTAMPTZ).
	rows, err := r.pdb.Player.Query(ctx, fmt.Sprintf(`
		SELECT
			r.match_id,
			r.start_time,
			r.end_time,
			COALESCE(r.map_name_fr, r.map_name) AS map_name,
			COALESCE(r.pair_name_fr, r.pair_name) AS pair_name,
			COALESCE(r.playlist_name_fr, r.playlist_name) AS playlist_name,
			mp.kills, mp.deaths, mp.assists, mp.outcome,
			ABS(DATEDIFF('second', ?, r.start_time)) AS delta_s
		FROM shared.match_registry r
		JOIN shared.match_participants mp
			ON mp.match_id = r.match_id AND mp.xuid = ?
		WHERE r.start_time BETWEEN (? - INTERVAL '%d minutes')
		                       AND (? + INTERVAL '%d minutes')
		ORDER BY ABS(DATEDIFF('second', ?, r.start_time)) ASC
		LIMIT 50
	`, windowMinutes, windowMinutes), cap, r.pdb.XUID, cap, cap, cap)
	if err != nil {
		return resp, fmt.Errorf("LoadMatchCandidatesForMedia: query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var c domain.MediaMatchCandidate
		var mapName, pairName, playlistName sql.NullString
		var kills, deaths, assists, outcome sql.NullInt64
		var deltaS sql.NullInt64
		var startT, endT sql.NullTime
		if err := rows.Scan(&c.MatchID, &startT, &endT, &mapName, &pairName, &playlistName,
			&kills, &deaths, &assists, &outcome, &deltaS); err != nil {
			continue
		}
		if startT.Valid {
			t := startT.Time
			c.StartTime = &t
		}
		if endT.Valid {
			t := endT.Time
			c.EndTime = &t
		}
		if mapName.Valid {
			s := strings.TrimSpace(mapName.String)
			if s != "" {
				c.MapName = &s
			}
		}
		if pairName.Valid {
			s := strings.TrimSpace(pairName.String)
			if s != "" {
				c.ModeName = &s
			}
		}
		if playlistName.Valid {
			s := strings.TrimSpace(playlistName.String)
			if s != "" {
				c.PlaylistName = &s
			}
		}
		if kills.Valid {
			n := int(kills.Int64)
			c.Kills = &n
		}
		if deaths.Valid {
			n := int(deaths.Int64)
			c.Deaths = &n
		}
		if assists.Valid {
			n := int(assists.Int64)
			c.Assists = &n
		}
		if outcome.Valid {
			n := int(outcome.Int64)
			c.Outcome = &n
		}
		if deltaS.Valid {
			n := int(deltaS.Int64)
			c.DeltaSeconds = &n
		}
		c.IsCurrent = currentMatchID.Valid && currentMatchID.String == c.MatchID
		resp.Candidates = append(resp.Candidates, c)
	}
	return resp, nil
}

// SetMediaMatchAssociation force l'association d'un média à un match précis.
// Supprime l'association existante (si présente) et insère la nouvelle.
// Retourne (mapName, modeName) pour permettre au handler d'enrichir la réponse.
func (r *MediaRepo) SetMediaMatchAssociation(ctx context.Context, filePath, matchID string) (mapName, modeName *string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Récupérer l'id du média (match flexible : file_path exact OU basename)
	basename := filepath.Base(filePath)
	var mediaID string
	if err := r.socialDB().QueryRow(ctx,
		`SELECT id FROM media_files WHERE file_path = ? OR file_name = ? LIMIT 1`,
		filePath, basename,
	).Scan(&mediaID); err != nil {
		return nil, nil, fmt.Errorf("SetMediaMatchAssociation: media not found: %w", err)
	}

	// DELETE + INSERT séquentiels. DuckDB est ACID single-writer sur fichier ;
	// risque de race minimal pour une opération manuelle utilisateur.
	if _, err := r.socialDB().Exec(ctx, `DELETE FROM media_match_associations WHERE media_file_id = ?`, mediaID); err != nil {
		return nil, nil, fmt.Errorf("delete old assoc: %w", err)
	}
	if _, err := r.socialDB().Exec(ctx, `INSERT INTO media_match_associations (media_file_id, match_id, delta_seconds) VALUES (?, ?, 0)`, mediaID, matchID); err != nil {
		return nil, nil, fmt.Errorf("insert new assoc: %w", err)
	}

	// Récupérer map/mode du nouveau match pour le retour
	var mapN, pairN sql.NullString
	_ = r.pdb.Player.QueryRow(ctx, `
		SELECT COALESCE(r.map_name_fr, r.map_name), COALESCE(r.pair_name_fr, r.pair_name)
		FROM shared.match_registry r WHERE r.match_id = ? LIMIT 1
	`, matchID).Scan(&mapN, &pairN)
	if mapN.Valid {
		s := strings.TrimSpace(mapN.String)
		if s != "" {
			mapName = &s
		}
	}
	if pairN.Valid {
		s := strings.TrimSpace(pairN.String)
		if s != "" {
			modeName = &s
		}
	}
	return mapName, modeName, nil
}

func (r *MediaRepo) queryConfig() mediaQueryConfig {
	if r.pdb.SharedSocial != nil {
		return mediaQueryConfig{playerSlug: r.pdb.Gamertag}
	}
	return mediaQueryConfig{}
}

// SetMediaLike persiste l'état liked d'un média dans media_files (cache local).
func (r *MediaRepo) SetMediaLike(ctx context.Context, filePath string, liked bool) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := r.socialDB().Exec(ctx, `
		UPDATE media_files
		SET liked = ?,
			liked_at = CASE WHEN ? THEN CURRENT_TIMESTAMP ELSE NULL END
		WHERE file_path = ?
	`, liked, liked, filePath)
	if err != nil {
		return false, fmt.Errorf("SetMediaLike: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("SetMediaLike rows affected: %w", err)
	}
	return rowsAffected > 0, nil
}

// ToggleSharedLike écrit ou supprime un like dans media_likes (shared DB).
// Retourne l'état liked résultant.
func (r *MediaRepo) ToggleSharedLike(ctx context.Context, mediaPath, likerSlug, likerGamertag string, liked bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if liked {
		_, err := r.socialDB().Exec(ctx, `
			INSERT INTO media_likes (media_path, liker_slug, liker_gamertag, liked_at)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (media_path, liker_slug) DO UPDATE SET
				liker_gamertag = EXCLUDED.liker_gamertag,
				liked_at = CURRENT_TIMESTAMP
		`, mediaPath, likerSlug, likerGamertag)
		return err
	}
	_, err := r.socialDB().Exec(ctx, `
		DELETE FROM media_likes WHERE media_path = ? AND liker_slug = ?
	`, mediaPath, likerSlug)
	return err
}

// GetMediaLikers retourne pour chaque media_path ses likers (max 3 noms + total).
func (r *MediaRepo) GetMediaLikers(ctx context.Context, mediaPaths []string) (map[string]domain.MediaLikersInfo, error) {
	if len(mediaPaths) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Construire le placeholder IN (?, ?, ...)
	placeholders := make([]string, len(mediaPaths))
	args := make([]any, len(mediaPaths))
	for i, p := range mediaPaths {
		placeholders[i] = "?"
		args[i] = p
	}

	q := `SELECT media_path, liker_gamertag, ROW_NUMBER() OVER (PARTITION BY media_path ORDER BY liked_at) AS rn,
		COUNT(*) OVER (PARTITION BY media_path) AS total
	FROM media_likes
	WHERE media_path IN (` + joinStrings(placeholders) + `)
	ORDER BY media_path, liked_at`

	rows, err := r.socialDB().Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("GetMediaLikers: %w", err)
	}
	defer rows.Close()

	result := make(map[string]domain.MediaLikersInfo)
	for rows.Next() {
		var path, gamertag string
		var rn, total int
		if err := rows.Scan(&path, &gamertag, &rn, &total); err != nil {
			return nil, err
		}
		info := result[path]
		info.Total = total
		if rn <= 3 {
			info.Names = append(info.Names, gamertag)
		}
		result[path] = info
	}
	return result, rows.Err()
}

func joinStrings(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

// enrichMediaMapTranslations résout les noms de cartes en français depuis asset_translations (metadata.duckdb).
// Même mécanisme que HomeRepo.enrichHomeMatchTranslations.
func (r *MediaRepo) enrichMediaMapTranslations(ctx context.Context, rows []domain.MediaFileRow) {
	if r.pdb.Metadata == nil || len(rows) == 0 {
		return
	}

	seen := make(map[string]struct{})
	for _, row := range rows {
		if row.MapID != nil && *row.MapID != "" {
			seen[*row.MapID] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return
	}

	placeholders := make([]string, 0, len(seen))
	args := make([]any, 0, len(seen)+1)
	args = append(args, "map")
	for id := range seen {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}

	q := `SELECT asset_id, name
FROM asset_translations
WHERE asset_type = ?
  AND lang IN ('fr-FR', 'fr')
  AND asset_id IN (` + joinStrings(placeholders) + `)
ORDER BY lang DESC`

	dbRows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		return
	}
	defer dbRows.Close()

	translations := make(map[string]string)
	for dbRows.Next() {
		var assetID, name string
		if err := dbRows.Scan(&assetID, &name); err != nil {
			continue
		}
		if _, ok := translations[assetID]; !ok {
			translations[assetID] = name
		}
	}

	for i := range rows {
		if rows[i].MapID == nil {
			continue
		}
		if nameFR, ok := translations[*rows[i].MapID]; ok && nameFR != "" {
			rows[i].MapName = &nameFR
		}
	}
}

// mediaFilterOptionPair regroupe l'id source (map_id ou pair_name brut) et le
// label SQL utilisé pour le filtrage et l'affichage par défaut.
type mediaFilterOptionPair struct {
	id    string
	label string
}

func (r *MediaRepo) loadMediaIDLabelPairs(ctx context.Context, query string, args []any) ([]mediaFilterOptionPair, error) {
	rows, err := r.socialDB().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pairs := make([]mediaFilterOptionPair, 0)
	for rows.Next() {
		var id, label string
		if err := rows.Scan(&id, &label); err != nil {
			return nil, err
		}
		if strings.TrimSpace(label) == "" {
			continue
		}
		pairs = append(pairs, mediaFilterOptionPair{id: id, label: label})
	}
	return pairs, rows.Err()
}

// translateMapFilterOptions enrichit les libellés de cartes en FR via
// asset_translations + dédup par map_id. Value = map_id (stable, structurel)
// pour permettre un filtrage non ambigu côté backend (sinon "Altitude" FR ne
// matche pas "High Ground" raw EN dans match_registry, et le filtre devient
// inutilisable). Label = FR enrichi pour l'affichage.
func (r *MediaRepo) translateMapFilterOptions(ctx context.Context, pairs []mediaFilterOptionPair) []domain.LabelValue {
	if len(pairs) == 0 {
		return []domain.LabelValue{}
	}
	idsSet := make(map[string]struct{})
	for _, p := range pairs {
		if p.id != "" {
			idsSet[p.id] = struct{}{}
		}
	}
	ids := make([]string, 0, len(idsSet))
	for id := range idsSet {
		ids = append(ids, id)
	}
	translations := r.loadAssetTranslationNames(ctx, "map", ids)

	// Dédup par map_id : si plusieurs raw labels mappent vers le même map_id
	// (ex: "High Ground" et "Altitude" pour la même carte selon match_name_fr),
	// on regroupe sous une seule entrée. Si map_id absent, fallback sur label.
	seenIDs := make(map[string]bool)
	seenLabels := make(map[string]bool)
	options := make([]domain.LabelValue, 0, len(pairs))
	for _, p := range pairs {
		labelFR := translations[p.id]
		if labelFR == "" {
			labelFR = p.label
		}
		// Value = map_id si dispo (stable), sinon label (fallback médias sans match)
		value := p.id
		if value == "" {
			value = p.label
			if seenLabels[value] {
				continue
			}
			seenLabels[value] = true
		} else {
			if seenIDs[value] {
				continue
			}
			seenIDs[value] = true
		}
		options = append(options, domain.LabelValue{Label: labelFR, Value: value})
	}
	sort.Slice(options, func(i, j int) bool { return options[i].Label < options[j].Label })
	return options
}

// translateModeFilterOptions normalise (analysis.NormalizeModeLabel) puis traduit
// chaque mode en FR via mode_name_tr, et déduplique par libellé FR. Value reste
// le label FR : le repo ré-expanse côté ModeFilter (FR → liste raw EN candidates)
// avant la query (cf. expandModeFilter), pour que le ILIKE matche tous les variants.
func (r *MediaRepo) translateModeFilterOptions(ctx context.Context, pairs []mediaFilterOptionPair) []domain.LabelValue {
	if len(pairs) == 0 {
		return []domain.LabelValue{}
	}
	enSet := make(map[string]struct{})
	for _, p := range pairs {
		// p.id contient le pair_name brut, p.label la version regex-normalisée
		if en := analysis.NormalizeModeLabel(p.id); en != "" {
			enSet[en] = struct{}{}
		}
		if en := analysis.NormalizeModeLabel(p.label); en != "" {
			enSet[en] = struct{}{}
		}
	}
	enList := make([]string, 0, len(enSet))
	for en := range enSet {
		enList = append(enList, en)
	}
	translations := r.loadModeNameTranslations(ctx, enList)

	seen := make(map[string]bool)
	options := make([]domain.LabelValue, 0, len(pairs))
	for _, p := range pairs {
		labelFR := translations[analysis.NormalizeModeLabel(p.id)]
		if labelFR == "" {
			labelFR = translations[analysis.NormalizeModeLabel(p.label)]
		}
		if labelFR == "" {
			labelFR = p.label
		}
		if seen[labelFR] {
			continue
		}
		seen[labelFR] = true
		// Value = labelFR pour que le frontend envoie le FR au backend ; le repo
		// ré-expanse via expandModeFilter() avant la SQL query.
		options = append(options, domain.LabelValue{Label: labelFR, Value: labelFR})
	}
	sort.Slice(options, func(i, j int) bool { return options[i].Label < options[j].Label })
	return options
}

// expandModeFilter convertit un ModeFilter exprimé en FR (ex: "Capture du
// drapeau") vers la liste de raw EN candidates (ex: ["Capture the Flag", "CTF",
// …]) via reverse mode_name_tr lookup. Le SQL utilise alors un OR sur chaque
// variant ILIKE pour matcher tous les médias quel que soit leur pair_name brut.
// Si aucune correspondance trouvée, ModeFilter reste tel quel et le SQL fallback
// sur l'ILIKE simple existant.
func (r *MediaRepo) expandModeFilter(ctx context.Context, filters domain.MediaFilters) domain.MediaFilters {
	if filters.ModeFilter == "" || len(filters.ModeFilterCandidates) > 0 {
		return filters
	}
	if r.pdb == nil || r.pdb.Metadata == nil {
		return filters
	}
	candidates := r.reverseModeNameLookup(ctx, filters.ModeFilter)
	if len(candidates) > 0 {
		filters.ModeFilterCandidates = candidates
	}
	return filters
}

// reverseModeNameLookup : étant donné un nom FR (ex: "Capture du drapeau"),
// retourne la liste des mode_en présents dans mode_name_tr ayant ce libellé.
// Best-effort : retourne nil sur erreur ou table absente.
func (r *MediaRepo) reverseModeNameLookup(ctx context.Context, modeFR string) []string {
	if r.pdb == nil || r.pdb.Metadata == nil || strings.TrimSpace(modeFR) == "" {
		return nil
	}
	q := `SELECT DISTINCT mode_en FROM mode_name_tr WHERE lang = 'fr' AND name = ?`
	rows, err := r.pdb.Metadata.Query(ctx, q, modeFR)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var en string
		if err := rows.Scan(&en); err != nil {
			continue
		}
		if en = strings.TrimSpace(en); en != "" {
			out = append(out, en)
		}
	}
	return out
}

// loadAssetTranslationNames lit les traductions FR depuis metadata.asset_translations.
// Retourne map[asset_id]→nom FR. Best-effort.
func (r *MediaRepo) loadAssetTranslationNames(ctx context.Context, assetType string, assetIDs []string) map[string]string {
	out := make(map[string]string)
	if r.pdb == nil || r.pdb.Metadata == nil || len(assetIDs) == 0 {
		return out
	}

	placeholders := make([]string, len(assetIDs))
	args := make([]any, 0, len(assetIDs)+1)
	args = append(args, assetType)
	for i, id := range assetIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	q := `SELECT asset_id, name
FROM asset_translations
WHERE asset_type = ?
  AND lang IN ('fr-FR', 'fr')
  AND name IS NOT NULL
  AND TRIM(name) != ''
  AND asset_id IN (` + joinStrings(placeholders) + `)
ORDER BY asset_id, CASE WHEN lang = 'fr-FR' THEN 0 ELSE 1 END`

	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		if _, exists := out[id]; !exists {
			out[id] = name
		}
	}
	return out
}

// loadModeNameTranslations lit les traductions FR depuis metadata.mode_name_tr,
// keyed par mode_en (déjà normalisé via analysis.NormalizeModeLabel).
func (r *MediaRepo) loadModeNameTranslations(ctx context.Context, modeENNames []string) map[string]string {
	out := make(map[string]string)
	if r.pdb == nil || r.pdb.Metadata == nil || len(modeENNames) == 0 {
		return out
	}

	placeholders := make([]string, len(modeENNames))
	args := make([]any, len(modeENNames))
	for i, name := range modeENNames {
		placeholders[i] = "?"
		args[i] = name
	}

	q := `SELECT mode_en, name
FROM mode_name_tr
WHERE lang = 'fr'
  AND mode_en IN (` + joinStrings(placeholders) + `)`

	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var modeEN, name string
		if err := rows.Scan(&modeEN, &name); err != nil {
			continue
		}
		if strings.TrimSpace(name) != "" {
			out[modeEN] = name
		}
	}
	return out
}
