// Package duckdb â€” media_repo.go : accÃ¨s DB pour la galerie mÃ©dias.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/port"
)

// MediaRepo implÃ©mente port.MediaRepository.
type MediaRepo struct {
	pdb *PlayerDB
}

// NewMediaRepo crÃ©e un MediaRepo pour un joueur.
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

// LoadMediaFiles charge une page de fichiers mÃ©dias avec filtres dynamiques (Q37).
func (r *MediaRepo) LoadMediaFiles(ctx context.Context, filters domain.MediaFilters, limit, offset int) ([]domain.MediaFileRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	q, args := buildQ37MediaQuery(filters, limit, offset, r.queryConfig())
	rows, err := r.socialDB().QueryRecovered(ctx, q, args...)
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
			&row.PairNameRaw,
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
	r.enrichMediaModeCategories(result)
	return result, nil
}

// enrichMediaModeCategories remplace ModeName par la catÃ©gorie custom infÃ©rÃ©e
// depuis pair_name brut (Assassin/Fiesta/BTB/Ranked/Firefight/Other). Cf.
// halo_infinite.InferModeCategoryFromPairName pour la logique.
func (r *MediaRepo) enrichMediaModeCategories(rows []domain.MediaFileRow) {
	for i := range rows {
		if rows[i].PairNameRaw == nil || strings.TrimSpace(*rows[i].PairNameRaw) == "" {
			rows[i].ModeName = nil
			continue
		}
		cat := halo_infinite.InferModeCategoryFromPairName(*rows[i].PairNameRaw)
		if cat != "" {
			c := cat
			rows[i].ModeName = &c
		}
	}
}

// CountMediaFiles retourne le nombre total de fichiers mÃ©dias actifs selon les filtres.
func (r *MediaRepo) CountMediaFiles(ctx context.Context, filters domain.MediaFilters) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	q, args := buildQ37MediaCountQuery(filters, r.queryConfig())
	var count int
	err := r.socialDB().QueryRow(ctx, q, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("CountMediaFiles: %w", err)
	}
	return count, nil
}

// LoadMediaFilterOptions retourne les valeurs distinctes des filtres carte/mode,
// avec libellÃ©s enrichis en FR (asset_translations + mode_name_tr de metadata.duckdb)
// et dÃ©duplication par libellÃ© FR. Pour les modes, plusieurs raw EN qui se
// normalisent vers le mÃªme FR (ex: "Capture the Flag", "CTF - Ranked", "CTF on
// Bazaar" â†’ "Capture du drapeau") sont fusionnÃ©s en une seule entrÃ©e.
func (r *MediaRepo) LoadMediaFilterOptions(ctx context.Context, filters domain.MediaFilters) (domain.MediaFilterOptions, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

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

	playlistQuery, playlistArgs := buildQ37MediaPlaylistOptionsQuery(filters, queryCfg)
	playlistPairs, err := r.loadMediaIDLabelPairs(ctx, playlistQuery, playlistArgs)
	if err != nil {
		return domain.MediaFilterOptions{}, fmt.Errorf("LoadMediaFilterOptions playlists: %w", err)
	}
	playlists := r.translatePlaylistFilterOptions(ctx, playlistPairs)

	return domain.MediaFilterOptions{Playlists: playlists, Maps: maps, Modes: modes}, nil
}

// translatePlaylistFilterOptions enrichit les libellÃ©s de playlists en FR via
// asset_translations (asset_type='playlist') + dÃ©dup par playlist_id. Value =
// playlist_id (stable) ; sinon fallback label brut.
func (r *MediaRepo) translatePlaylistFilterOptions(ctx context.Context, pairs []mediaFilterOptionPair) []domain.LabelValue {
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
	translations := r.loadAssetTranslationNames(ctx, "playlist", ids)

	seenIDs := make(map[string]bool)
	seenLabels := make(map[string]bool)
	options := make([]domain.LabelValue, 0, len(pairs))
	for _, p := range pairs {
		labelFR := translations[p.id]
		if labelFR == "" {
			labelFR = p.label
		}
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

// CurrentPlayerSlug retourne le slug (== gamertag) du joueur dont on lit la galerie.
func (r *MediaRepo) CurrentPlayerSlug() string {
	if r.pdb == nil {
		return ""
	}
	return r.pdb.Gamertag
}

// LoadMatchCandidatesForMedia retourne les matchs du joueur courant dans la
// fenÃªtre temporelle [capture_start - window, capture_start + window].
// Inclut les KPIs du joueur pour aider Ã  reconnaÃ®tre le bon match.
// Si capture_start_utc est nul â†’ fallback mtime, sinon liste vide.
func (r *MediaRepo) LoadMatchCandidatesForMedia(ctx context.Context, filePath string, windowMinutes int) (domain.MediaMatchCandidatesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if windowMinutes <= 0 {
		windowMinutes = 15
	}

	// Lire capture_start_utc + association actuelle du mÃ©dia.
	// Match flexible : soit file_path exact (DB absolute), soit file_name
	// (basename â€” pour quand le frontend envoie l'URL transformÃ©e et qu'on
	// reÃ§oit ".../foo.mp4" au lieu du chemin DB original).
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

	// Charger les matchs du joueur dans la fenÃªtre.
	// start_time_utc est TIMESTAMPTZ UTC garanti (migration add_start_time_utc_to_match_registry).
	// Fallback sur start_time AT TIME ZONE 'UTC' pour les matchs synchro aprÃ¨s le fix DuckDB
	// (first_sync_at >= 2026-03-01 â†’ start_time dÃ©jÃ  UTC) qui n'auraient pas encore start_time_utc.
	rows, err := r.pdb.Player.Query(ctx, fmt.Sprintf(`
		SELECT
			r.match_id,
			COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_utc,
			COALESCE(r.end_time_utc,   r.end_time   AT TIME ZONE 'UTC') AS end_utc,
			COALESCE(r.map_name_fr, r.map_name) AS map_name,
			COALESCE(r.map_id, '') AS map_id,
			COALESCE(r.pair_name_fr, r.pair_name) AS pair_name,
			COALESCE(r.playlist_name_fr, r.playlist_name) AS playlist_name,
			COALESCE(r.playlist_id, '') AS playlist_id,
			mp.outcome,
			ABS(DATEDIFF('second', ?, COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC'))) AS delta_s
		FROM shared.match_registry r
		JOIN shared.match_participants mp
			ON mp.match_id = r.match_id AND mp.xuid = ?
		WHERE COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC')
		        BETWEEN (? - INTERVAL '%d minutes')
		            AND (? + INTERVAL '%d minutes')
		ORDER BY ABS(DATEDIFF('second', ?, COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC'))) ASC
		LIMIT 50
	`, windowMinutes, windowMinutes), cap, r.pdb.XUID, cap, cap, cap)
	if err != nil {
		return resp, fmt.Errorf("LoadMatchCandidatesForMedia: query: %w", err)
	}
	defer rows.Close()

	matchIDs := []string{}
	mapIDByMatch := map[string]string{}
	mapIDSet := map[string]struct{}{}
	modeEnSet := map[string]struct{}{}
	playlistIDByMatch := map[string]string{}
	playlistIDSet := map[string]struct{}{}
	for rows.Next() {
		var c domain.MediaMatchCandidate
		var mapName, mapID, pairName, playlistName, playlistID sql.NullString
		var outcome sql.NullInt64
		var deltaS sql.NullInt64
		var startT, endT sql.NullTime
		if err := rows.Scan(&c.MatchID, &startT, &endT, &mapName, &mapID, &pairName, &playlistName,
			&playlistID, &outcome, &deltaS); err != nil {
			continue
		}
		if mapID.Valid && mapID.String != "" {
			mapIDByMatch[c.MatchID] = mapID.String
			mapIDSet[mapID.String] = struct{}{}
		}
		if playlistID.Valid && playlistID.String != "" {
			playlistIDByMatch[c.MatchID] = playlistID.String
			playlistIDSet[playlistID.String] = struct{}{}
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
			// Sous-mode EN canonique pour le picker (Slayer/CTF/KOTH/etc.) :
			// l'utilisateur a besoin du DÃ‰TAIL du mode pour distinguer entre 4
			// matchs candidats â€” pas de la catÃ©gorie parente. cf. mode_label.go
			// (NormalizeModeLabel) vs mode_category.go (InferModeCategoryFromPairName).
			if en := analysis.NormalizeModeLabel(pairName.String); en != "" {
				c.ModeName = &en
				modeEnSet[en] = struct{}{}
			}
		}
		if playlistName.Valid {
			s := strings.TrimSpace(playlistName.String)
			if s != "" {
				c.PlaylistName = &s
			}
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
		matchIDs = append(matchIDs, c.MatchID)
	}

	// Traduction FR des modes (ex: "Slayer" â†’ "Assassin") via mode_name_tr.
	// Si pair_name_fr Ã©tait dÃ©jÃ  rempli en DB, on le prÃ©serve quand mÃªme
	// puisqu'on substitue uniquement si une traduction existe.
	if len(modeEnSet) > 0 {
		enList := make([]string, 0, len(modeEnSet))
		for en := range modeEnSet {
			enList = append(enList, en)
		}
		translations := r.loadModeNameTranslations(ctx, enList)
		for i := range resp.Candidates {
			if resp.Candidates[i].ModeName == nil {
				continue
			}
			// ModeName est dÃ©jÃ  le sous-mode EN canonique (cf. boucle ci-dessus) â†’
			// lookup direct, pas besoin de re-normaliser.
			if fr, ok := translations[*resp.Candidates[i].ModeName]; ok && fr != "" {
				resp.Candidates[i].ModeName = &fr
			}
		}
	}

	// Résolution MapImageURL via map_images_registry (pattern asset kinds —
	// lookup par map_id, pas par name pour éviter les écueils FR/EN/UUID brut).
	if len(mapIDSet) > 0 {
		ids := make([]string, 0, len(mapIDSet))
		for id := range mapIDSet {
			ids = append(ids, id)
		}
		urls := r.loadMapImageURLsByID(ctx, ids)
		for i := range resp.Candidates {
			mid := mapIDByMatch[resp.Candidates[i].MatchID]
			if mid == "" {
				continue
			}
			if u, ok := urls[mid]; ok && u != "" {
				localCopy := u
				resp.Candidates[i].MapImageURL = &localCopy
			}
		}
	}

	// Traduction FR des playlists via asset_translations (asset_type='playlist').
	if len(playlistIDSet) > 0 {
		ids := make([]string, 0, len(playlistIDSet))
		for id := range playlistIDSet {
			ids = append(ids, id)
		}
		playlistTr := r.loadAssetTranslationNames(ctx, "playlist", ids)
		for i := range resp.Candidates {
			pid := playlistIDByMatch[resp.Candidates[i].MatchID]
			if pid == "" {
				continue
			}
			if fr, ok := playlistTr[pid]; ok && fr != "" {
				resp.Candidates[i].PlaylistName = &fr
			}
		}
	}

	// 2e query batch : lobby (max 12 joueurs par match â€” assez pour 4v4 + spectateurs)
	if len(matchIDs) > 0 {
		lobbies := r.loadMatchLobbies(ctx, matchIDs)
		for i := range resp.Candidates {
			resp.Candidates[i].Lobby = lobbies[resp.Candidates[i].MatchID]
		}
	}

	return resp, nil
}

// loadMatchLobbies retourne le lobby (max 12 joueurs/match) pour un set de match_ids.
// Marque is_self pour le joueur courant (xuid match).
func (r *MediaRepo) loadMatchLobbies(ctx context.Context, matchIDs []string) map[string][]domain.MediaMatchLobbyEntry {
	if len(matchIDs) == 0 {
		return nil
	}
	placeholders := make([]string, len(matchIDs))
	args := make([]any, 0, len(matchIDs)+1)
	args = append(args, r.pdb.XUID)
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	// Résolveur canonique : v_gamertag_lookup gère bots + cascade
	// xuid_aliases / match_participants. shared.xuid_aliases couvre les
	// participants jamais croisés directement par le joueur courant.
	q := `
		SELECT mp.match_id,
			COALESCE(vg.gamertag, va.gamertag, mp.xuid) AS gamertag,
			mp.team_id,
			(mp.xuid = ?) AS is_self
		FROM shared.match_participants mp
		LEFT JOIN shared.v_gamertag_lookup vg ON vg.xuid = mp.xuid
		LEFT JOIN shared.xuid_aliases va ON va.xuid = mp.xuid
		WHERE mp.match_id IN (` + joinStrings(placeholders) + `)
		ORDER BY mp.match_id, mp.team_id, mp.gamertag
	`
	rows, err := r.pdb.Player.Query(ctx, q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	out := make(map[string][]domain.MediaMatchLobbyEntry, len(matchIDs))
	for rows.Next() {
		var matchID string
		var gamertag sql.NullString
		var teamID sql.NullInt64
		var isSelf bool
		if err := rows.Scan(&matchID, &gamertag, &teamID, &isSelf); err != nil {
			continue
		}
		if len(out[matchID]) >= 12 {
			continue
		}
		name := "?"
		if gamertag.Valid {
			name = strings.TrimSpace(gamertag.String)
			if name == "" {
				name = "?"
			}
		}
		entry := domain.MediaMatchLobbyEntry{Gamertag: name, IsSelf: isSelf}
		if teamID.Valid {
			t := int(teamID.Int64)
			entry.TeamID = &t
		}
		out[matchID] = append(out[matchID], entry)
	}
	return out
}

// Miroir Go de q37MediaModeLabelExpr : extraction du mode parent.
//   - Si le label contient ":" â†’ prÃ©fixe avant (Arena:Slayer â†’ Arena)
//   - Sinon â†’ strip suffixes carte/Forge/Ranked
var (
	modeLabelOnRe     = regexp.MustCompile(`(?i)\s+on\s+.+$`)
	modeLabelForgeRe  = regexp.MustCompile(`(?i)\s*-\s*Forge\b.*$`)
	modeLabelRankedRe = regexp.MustCompile(`(?i)\s*-\s*Ranked\b.*$`)
)

func normalizeModeLabel(s string) string {
	if idx := strings.Index(s, ":"); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	s = modeLabelOnRe.ReplaceAllString(s, "")
	s = modeLabelForgeRe.ReplaceAllString(s, "")
	s = modeLabelRankedRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// SetMediaMatchAssociation force l'association d'un mÃ©dia Ã  un match prÃ©cis.
// Supprime l'association existante (si prÃ©sente) et insÃ¨re la nouvelle.
// Retourne (mapName, modeName) pour permettre au handler d'enrichir la rÃ©ponse.
func (r *MediaRepo) SetMediaMatchAssociation(ctx context.Context, filePath, matchID string) (mapName, modeName *string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// RÃ©cupÃ©rer l'id du mÃ©dia (match flexible : file_path exact OU basename)
	basename := filepath.Base(filePath)
	var mediaID string
	if err := r.socialDB().QueryRow(ctx,
		`SELECT id FROM media_files WHERE file_path = ? OR file_name = ? LIMIT 1`,
		filePath, basename,
	).Scan(&mediaID); err != nil {
		return nil, nil, fmt.Errorf("SetMediaMatchAssociation: media not found: %w", err)
	}

	// DELETE + INSERT sÃ©quentiels. DuckDB est ACID single-writer sur fichier ;
	// risque de race minimal pour une opÃ©ration manuelle utilisateur.
	// is_manual = TRUE : marque la correction utilisateur pour qu'un reassociate
	// global ultÃ©rieur ne l'Ã©crase pas.
	if _, err := r.socialDB().ExecRecovered(ctx, `DELETE FROM media_match_associations WHERE media_file_id = ?`, mediaID); err != nil {
		return nil, nil, fmt.Errorf("delete old assoc: %w", err)
	}
	if _, err := r.socialDB().ExecRecovered(ctx, `INSERT INTO media_match_associations (media_file_id, match_id, delta_seconds, is_manual) VALUES (?, ?, 0, TRUE)`, mediaID, matchID); err != nil {
		return nil, nil, fmt.Errorf("insert new assoc: %w", err)
	}

	// RÃ©cupÃ©rer map/mode du nouveau match pour le retour
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

// SetMediaLike persiste l'Ã©tat liked d'un mÃ©dia dans media_files (cache local).
func (r *MediaRepo) SetMediaLike(ctx context.Context, filePath string, liked bool) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := r.socialDB().ExecRecovered(ctx, `
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

// SetMediaLikeAtomic exécute en une seule transaction (si exec est un *sql.Tx)
// le UPDATE media_files.liked + l'INSERT/DELETE media_likes correspondant.
//
// Si likerSlug est vide, seul le UPDATE media_files.liked est exécuté
// (pas de ligne dans media_likes côté shared).
//
// Retourne true si la ligne media_files a été mise à jour (file_path existe).
//
// Cette méthode est l'usage canonique côté MediaService quand un
// WriterAcquirer est configuré : le service ouvre une *sql.Tx via
// LeasedWriter.BeginTx, l'injecte ici comme port.DBExecutor → atomicité.
//
// Cf. commit 6 du refactor leased-writer-enforcement (résout P3).
func (r *MediaRepo) SetMediaLikeAtomic(
	ctx context.Context,
	exec port.DBExecutor,
	filePath, likerSlug, likerGamertag string,
	liked bool,
) (bool, error) {
	result, err := exec.ExecContext(ctx, `
		UPDATE media_files
		SET liked = ?,
			liked_at = CASE WHEN ? THEN CURRENT_TIMESTAMP ELSE NULL END
		WHERE file_path = ?
	`, liked, liked, filePath)
	if err != nil {
		return false, fmt.Errorf("SetMediaLikeAtomic update: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("SetMediaLikeAtomic rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return false, nil // file_path inconnu — caller traduit en 404 sans toucher media_likes
	}

	if likerSlug == "" {
		return true, nil
	}

	if liked {
		_, err := exec.ExecContext(ctx, `
			INSERT INTO media_likes (media_path, liker_slug, liker_gamertag, liked_at)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (media_path, liker_slug) DO UPDATE SET
				liker_gamertag = EXCLUDED.liker_gamertag,
				liked_at = EXCLUDED.liked_at
		`, filePath, likerSlug, likerGamertag)
		if err != nil {
			return true, fmt.Errorf("SetMediaLikeAtomic insert media_likes: %w", err)
		}
		return true, nil
	}

	_, err = exec.ExecContext(ctx, `
		DELETE FROM media_likes WHERE media_path = ? AND liker_slug = ?
	`, filePath, likerSlug)
	if err != nil {
		return true, fmt.Errorf("SetMediaLikeAtomic delete media_likes: %w", err)
	}
	return true, nil
}

// ToggleSharedLike Ã©crit ou supprime un like dans media_likes (shared DB).
// Retourne l'Ã©tat liked rÃ©sultant.
func (r *MediaRepo) ToggleSharedLike(ctx context.Context, mediaPath, likerSlug, likerGamertag string, liked bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if liked {
		// Note : `liked_at = CURRENT_TIMESTAMP` dans le ON CONFLICT casse le binder
		// DuckDB qui interprÃ¨te CURRENT_TIMESTAMP comme un nom de colonne.
		// On utilise EXCLUDED.liked_at qui prend la valeur du VALUES (= CURRENT_TIMESTAMP).
		_, err := r.socialDB().ExecRecovered(ctx, `
			INSERT INTO media_likes (media_path, liker_slug, liker_gamertag, liked_at)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (media_path, liker_slug) DO UPDATE SET
				liker_gamertag = EXCLUDED.liker_gamertag,
				liked_at = EXCLUDED.liked_at
		`, mediaPath, likerSlug, likerGamertag)
		return err
	}
	_, err := r.socialDB().ExecRecovered(ctx, `
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

	rows, err := r.socialDB().QueryRecovered(ctx, q, args...)
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

// enrichMediaMapTranslations rÃ©sout les noms de cartes en franÃ§ais depuis asset_translations (metadata.duckdb).
// MÃªme mÃ©canisme que HomeRepo.enrichHomeMatchTranslations.
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
// label SQL utilisÃ© pour le filtrage et l'affichage par dÃ©faut.
type mediaFilterOptionPair struct {
	id    string
	label string
}

func (r *MediaRepo) loadMediaIDLabelPairs(ctx context.Context, query string, args []any) ([]mediaFilterOptionPair, error) {
	rows, err := r.socialDB().QueryRecovered(ctx, query, args...)
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

// translateMapFilterOptions enrichit les libellÃ©s de cartes en FR via
// asset_translations + dÃ©dup par map_id. Value = map_id (stable, structurel)
// pour permettre un filtrage non ambigu cÃ´tÃ© backend (sinon "Altitude" FR ne
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

	// DÃ©dup par map_id : si plusieurs raw labels mappent vers le mÃªme map_id
	// (ex: "High Ground" et "Altitude" pour la mÃªme carte selon match_name_fr),
	// on regroupe sous une seule entrÃ©e. Si map_id absent, fallback sur label.
	seenIDs := make(map[string]bool)
	seenLabels := make(map[string]bool)
	options := make([]domain.LabelValue, 0, len(pairs))
	for _, p := range pairs {
		labelFR := translations[p.id]
		if labelFR == "" {
			labelFR = p.label
		}
		// Value = map_id si dispo (stable), sinon label (fallback mÃ©dias sans match)
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

// translateModeFilterOptions retourne une liste hiÃ©rarchique :
//   - 1 entrÃ©e racine par catÃ©gorie prÃ©sente : {Label: "Assassin", Value: "Assassin"}
//     (label EN canonique â†’ frontend traduit via i18n local)
//   - N entrÃ©es sous-mode par catÃ©gorie : {Label: "Slayer" (ou trad FR via
//     mode_name_tr si dispo), Value: "Assassin/Slayer", Parent: "Assassin"}
//
// Le format value "CatÃ©gorie/SousMode" permet au backend de filtrer finement :
// le WHERE dÃ©tecte le sÃ©parateur "/" et applique catÃ©gorie + sous-mode normalisÃ©.
func (r *MediaRepo) translateModeFilterOptions(ctx context.Context, pairs []mediaFilterOptionPair) []domain.LabelValue {
	if len(pairs) == 0 {
		return []domain.LabelValue{}
	}

	// 1) Grouper par catÃ©gorie + collecter les sous-modes EN distincts
	type catBucket struct {
		category string
		subEN    map[string]struct{} // sous-modes EN canoniques (ex: "Slayer", "Team Slayer")
	}
	buckets := make(map[string]*catBucket)
	subEnSet := make(map[string]struct{})
	for _, p := range pairs {
		cat := halo_infinite.InferModeCategoryFromPairName(p.id)
		if cat == "" {
			cat = halo_infinite.ModeCategoryOther
		}
		if buckets[cat] == nil {
			buckets[cat] = &catBucket{category: cat, subEN: make(map[string]struct{})}
		}
		// Sous-mode EN canonique via NormalizeModeLabel ("Arena:Slayer on X" â†’ "Slayer").
		if sub := analysis.NormalizeModeLabel(p.id); sub != "" {
			buckets[cat].subEN[sub] = struct{}{}
			subEnSet[sub] = struct{}{}
		}
	}

	// 2) Traduire les sous-modes EN â†’ FR via mode_name_tr (best-effort)
	subEnList := make([]string, 0, len(subEnSet))
	for en := range subEnSet {
		subEnList = append(subEnList, en)
	}
	subTranslations := r.loadModeNameTranslations(ctx, subEnList)

	// 3) Construire la liste plate : header catÃ©gorie + sous-modes triÃ©s
	categories := make([]string, 0, len(buckets))
	for cat := range buckets {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	options := make([]domain.LabelValue, 0)
	for _, cat := range categories {
		b := buckets[cat]
		// Header catÃ©gorie (label EN, le frontend traduit via i18n.ts)
		options = append(options, domain.LabelValue{Label: cat, Value: cat})
		// Sous-modes triÃ©s par label localisÃ©
		subs := make([]domain.LabelValue, 0, len(b.subEN))
		for en := range b.subEN {
			label := en
			if fr, ok := subTranslations[en]; ok && fr != "" {
				label = fr
			}
			subs = append(subs, domain.LabelValue{
				Label:  label,
				Value:  cat + "/" + en, // value canonique EN pour matcher cÃ´tÃ© WHERE
				Parent: cat,
			})
		}
		sort.Slice(subs, func(i, j int) bool { return subs[i].Label < subs[j].Label })
		options = append(options, subs...)
	}
	return options
}

// loadAssetTranslationNames lit les traductions FR depuis metadata.asset_translations.
// Retourne map[asset_id]â†’nom FR. Best-effort.
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

// loadMapImageURLsByID lit local_path depuis map_images_registry pour les
// mapIDs donnés (pattern asset kinds — lookup par ID dans la table cache,
// peuplée par cmd/migrate-static-maps). Retourne map[map_id]→local_path.
// map_ids absents du registry sont simplement absents de la map (best-effort).
func (r *MediaRepo) loadMapImageURLsByID(ctx context.Context, mapIDs []string) map[string]string {
	out := make(map[string]string)
	if r.pdb == nil || r.pdb.Metadata == nil || len(mapIDs) == 0 {
		return out
	}
	placeholders := make([]string, len(mapIDs))
	args := make([]any, 0, len(mapIDs)+1)
	args = append(args, mediaStaticTitleSlug)
	for i, id := range mapIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	q := `SELECT map_id, local_path
FROM map_images_registry
WHERE title_id = ?
  AND TRIM(local_path) != ''
  AND map_id IN (` + joinStrings(placeholders) + `)`
	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, localPath string
		if err := rows.Scan(&id, &localPath); err != nil {
			continue
		}
		out[id] = localPath
	}
	return out
}

// mediaStaticTitleSlug est le slug de titre utilisé pour résoudre les URLs
// statiques côté media. Halo Infinite uniquement pour le moment ; quand un
// 2e titre arrivera, ce slug sera dérivé du contexte (cf. PathResolver).
const mediaStaticTitleSlug = "halo_infinite"

// loadModeNameTranslations lit les traductions FR depuis metadata.mode_name_tr,
// keyed par mode_en (dÃ©jÃ  normalisÃ© via analysis.NormalizeModeLabel).
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
