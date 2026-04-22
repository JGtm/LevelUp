// Package duckdb — home_repo.go : accès DB pour la page d'accueil Mission Control.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

const homeCareerRankGameCMSBase = "https://gamecms-hacs.svc.halowaypoint.com"

// HomeRepo fournit les données de la page d'accueil depuis DuckDB.
type HomeRepo struct {
	pdb *PlayerDB
}

// NewHomeRepo crée un HomeRepo pour un joueur.
func NewHomeRepo(pdb *PlayerDB) *HomeRepo {
	return &HomeRepo{pdb: pdb}
}

// LoadHomeMatches charge tous les matchs du joueur (Q26).
func (r *HomeRepo) LoadHomeMatches(ctx context.Context) ([]domain.HomeMatchRow, error) {
	rows, err := r.pdb.Player.Query(ctx, Q26HomeMatches, r.pdb.XUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.HomeMatchRow
	for rows.Next() {
		var row domain.HomeMatchRow
		if err := rows.Scan(
			&row.MatchID,
			&row.StartTime,
			&row.MapID,
			&row.MapName,
			&row.MapNameFR,
			&row.PairID,
			&row.PairName,
			&row.PairNameFR,
			&row.GameVariantID,
			&row.GameVariantName,
			&row.PlaylistID,
			&row.PlaylistName,
			&row.PlaylistNameFR,
			&row.IsFirefight,
			&row.IsRanked,
			&row.SessionLabel,
			&row.IsWithFriends,
			&row.Outcome,
			&row.TeamID,
			&row.Team0Score,
			&row.Team1Score,
			&row.DominanceFlag,
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.KDA,
			&row.Ratio,
			&row.Accuracy,
			&row.TimePlayedSecs,
			&row.DamageDealt,
			&row.DamageTaken,
			&row.PerformanceScore,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	r.enrichHomeMatchTranslations(ctx, result)
	return result, nil
}

// LoadSpartanIdentity charge le bloc record compact depuis career_progression et metadata.
// Dégrade silencieusement si la carrière n'est pas synchronisée pour le joueur.
func (r *HomeRepo) LoadSpartanIdentity(ctx context.Context) (*domain.HomeSpartanIdentityRow, error) {
	var row domain.HomeSpartanIdentityRow
	var spartanID sql.NullString
	var rankName sql.NullString
	var rankTier sql.NullString

	err := r.pdb.Player.QueryRow(ctx, Q26cHomeSpartanIdentity).Scan(
		&row.RankNumber,
		&row.CurrentXP,
		&row.XPForNextRank,
		&row.IsMaxRank,
		&spartanID,
		&rankName,
		&rankTier,
	)
	if err != nil {
		if err == sql.ErrNoRows || isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}

	if spartanID.Valid {
		row.SpartanID = stringPtr(spartanID.String)
	}
	if rankName.Valid {
		row.RankName = stringPtr(rankName.String)
	}
	if rankTier.Valid {
		row.RankTier = stringPtr(rankTier.String)
	}

	r.enrichSpartanIdentity(ctx, &row)

	if row.SpartanID == nil && row.RankNumber <= 0 {
		return nil, nil
	}
	return &row, nil
}

func (r *HomeRepo) enrichSpartanIdentity(ctx context.Context, row *domain.HomeSpartanIdentityRow) {
	if row == nil || row.RankNumber <= 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return
	}

	var titleEN sql.NullString
	var titleFR sql.NullString
	var imagePath sql.NullString
	if err := r.pdb.Metadata.QueryRow(ctx, Q26dHomeCareerRankMeta, row.RankNumber).Scan(&titleEN, &titleFR, &imagePath); err != nil {
		return
	}
	if titleEN.Valid {
		row.RankTitleEN = stringPtr(titleEN.String)
	}
	if titleFR.Valid {
		row.RankTitleFR = stringPtr(titleFR.String)
	}
	if imagePath.Valid {
		row.RankImageURL = buildHomeCareerRankImageURL(imagePath.String)
	}
}

func buildHomeCareerRankImageURL(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	lower := strings.ToLower(trimmed)
	if strings.HasSuffix(lower, ".json") {
		return nil
	}
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return &trimmed
	}

	cleaned := strings.TrimLeft(trimmed, "/")
	if cleaned == "" {
		return nil
	}

	switch {
	case strings.HasPrefix(strings.ToLower(cleaned), "hi/images/file/"):
		resolved := homeCareerRankGameCMSBase + "/" + cleaned
		return &resolved
	case strings.HasPrefix(strings.ToLower(cleaned), "images/file/"):
		resolved := homeCareerRankGameCMSBase + "/hi/" + cleaned
		return &resolved
	case strings.HasPrefix(strings.ToLower(cleaned), "hi/progression/file/"):
		resolved := homeCareerRankGameCMSBase + "/" + cleaned
		return &resolved
	case strings.HasPrefix(strings.ToLower(cleaned), "progression/file/"):
		resolved := homeCareerRankGameCMSBase + "/hi/" + cleaned
		return &resolved
	default:
		resolved := homeCareerRankGameCMSBase + "/hi/images/file/" + cleaned
		return &resolved
	}
}

func stringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (r *HomeRepo) enrichHomeMatchTranslations(ctx context.Context, matches []domain.HomeMatchRow) {
	if len(matches) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return
	}

	mapNames, _ := r.loadHomeAssetTranslationNames(ctx, "map", collectMissingHomeAssetIDs(matches, "map"))
	pairNames, _ := r.loadHomeAssetTranslationNames(ctx, "pair", collectMissingHomeAssetIDs(matches, "pair"))
	gameVariantNames, _ := r.loadHomeAssetTranslationNames(ctx, "game_variant", collectMissingHomeAssetIDs(matches, "game_variant"))
	playlistNames, _ := r.loadHomeAssetTranslationNames(ctx, "playlist", collectMissingHomeAssetIDs(matches, "playlist"))

	for i := range matches {
		if needsHomeAssetTranslation(matches[i].MapNameFR, matches[i].MapName) {
			if name := strings.TrimSpace(mapNames[matches[i].MapID]); name != "" {
				matches[i].MapNameFR = name
			}
		}
		if needsHomeAssetTranslation(matches[i].PairNameFR, matches[i].PairName) {
			if name := strings.TrimSpace(pairNames[matches[i].PairID]); name != "" {
				matches[i].PairNameFR = name
			}
		}
		if needsHomeAssetTranslation(matches[i].GameVariantNameFR, matches[i].GameVariantName) {
			if name := strings.TrimSpace(gameVariantNames[matches[i].GameVariantID]); name != "" {
				matches[i].GameVariantNameFR = name
			}
		}
		if needsHomeAssetTranslation(matches[i].PlaylistNameFR, matches[i].PlaylistName) {
			if name := strings.TrimSpace(playlistNames[matches[i].PlaylistID]); name != "" {
				matches[i].PlaylistNameFR = name
			}
		}
	}
}

func collectMissingHomeAssetIDs(matches []domain.HomeMatchRow, assetType string) []string {
	ids := make(map[string]struct{})
	for _, match := range matches {
		var assetID string
		var labelFR string
		var labelEN string

		switch assetType {
		case "map":
			assetID = match.MapID
			labelFR = match.MapNameFR
			labelEN = match.MapName
		case "pair":
			assetID = match.PairID
			labelFR = match.PairNameFR
			labelEN = match.PairName
		case "game_variant":
			assetID = match.GameVariantID
			labelFR = match.GameVariantNameFR
			labelEN = match.GameVariantName
		case "playlist":
			assetID = match.PlaylistID
			labelFR = match.PlaylistNameFR
			labelEN = match.PlaylistName
		default:
			return nil
		}

		if strings.TrimSpace(assetID) == "" || !needsHomeAssetTranslation(labelFR, labelEN) {
			continue
		}
		ids[assetID] = struct{}{}
	}

	result := make([]string, 0, len(ids))
	for assetID := range ids {
		result = append(result, assetID)
	}
	return result
}

func (r *HomeRepo) loadHomeAssetTranslationNames(ctx context.Context, assetType string, assetIDs []string) (map[string]string, error) {
	if len(assetIDs) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(assetIDs)), ",")
	query := fmt.Sprintf(`
		SELECT asset_id, name, lang
		FROM asset_translations
		WHERE asset_type = ?
		  AND lang IN ('fr-FR', 'fr')
		  AND name IS NOT NULL
		  AND TRIM(name) != ''
		  AND asset_id IN (%s)
		ORDER BY asset_id, CASE WHEN lang = 'fr-FR' THEN 0 ELSE 1 END
	`, placeholders)

	args := make([]any, 0, len(assetIDs)+1)
	args = append(args, assetType)
	for _, assetID := range assetIDs {
		args = append(args, assetID)
	}

	rows, err := r.pdb.Metadata.Query(ctx, query, args...)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	translations := make(map[string]string)
	for rows.Next() {
		var assetID string
		var name string
		var lang string
		if err := rows.Scan(&assetID, &name, &lang); err != nil {
			return nil, err
		}
		if _, exists := translations[assetID]; !exists {
			translations[assetID] = name
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return translations, nil
}

func needsHomeAssetTranslation(labelFR, labelEN string) bool {
	trimmedFR := strings.TrimSpace(labelFR)
	if trimmedFR == "" {
		return true
	}
	trimmedEN := strings.TrimSpace(labelEN)
	return trimmedEN != "" && strings.EqualFold(trimmedFR, trimmedEN)
}

// CountPlayerMatches retourne le nombre total de matchs du joueur (Q26b).
func (r *HomeRepo) CountPlayerMatches(ctx context.Context) (int, error) {
	var count int
	err := r.pdb.Player.QueryRow(ctx, Q26bCountPlayerMatches, r.pdb.XUID).Scan(&count)
	return count, err
}

// LoadHomeSessions charge les sessions avec label depuis player_match_enrichment (Q27).
func (r *HomeRepo) LoadHomeSessions(ctx context.Context) ([]domain.HomeSessionRow, error) {
	rows, err := r.pdb.Player.Query(ctx, Q27HomeSessions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.HomeSessionRow
	for rows.Next() {
		var row domain.HomeSessionRow
		if err := rows.Scan(
			&row.MatchID,
			&row.SessionID,
			&row.SessionLabel,
			&row.IsWithFriends,
			&row.StartTime,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadRecentMedia charge les médias récents du joueur (Q28).
// Retourne une liste vide si la table media_files n'existe pas.
func (r *HomeRepo) LoadRecentMedia(ctx context.Context, limit int) ([]domain.HomeMediaRow, error) {
	rows, err := r.pdb.Player.Query(ctx, Q28RecentMedia, limit)
	if err != nil {
		// La table media_files peut ne pas exister — dégradation silencieuse.
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var result []domain.HomeMediaRow
	for rows.Next() {
		var row domain.HomeMediaRow
		var matchID sql.NullString
		var matchStartTime sql.NullTime
		if err := rows.Scan(&row.FileName, &matchID, &matchStartTime); err != nil {
			return nil, err
		}
		if matchID.Valid {
			row.MatchID = &matchID.String
		}
		if matchStartTime.Valid {
			row.MatchStartTime = &matchStartTime.Time
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// isTableNotFoundErr détecte les erreurs "table not found" DuckDB.
func isTableNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Table with name") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "no such table")
}

// ---------------------------------------------------------------------------
// BattlePassCacheRepository implementation
// ---------------------------------------------------------------------------

// LoadCachedBattlePass retourne les données BP depuis battlepass_snapshots si une
// entrée récente du joueur existe dans la fenêtre ttl.
func (r *HomeRepo) LoadCachedBattlePass(ctx context.Context, ttl time.Duration) (*domain.BattlePassResponse, bool, error) {
	secs := int64(ttl.Seconds())
	query := fmt.Sprintf(`
		SELECT reward_track_path, current_rank, partial_progress
		FROM battlepass_snapshots
		WHERE xuid = ?
		  AND snapshot_at > CURRENT_TIMESTAMP - INTERVAL '%d' SECOND
		ORDER BY is_active DESC, snapshot_at DESC
		LIMIT 1`, secs)

	var trackPath string
	var rank, progress int
	err := r.pdb.Player.QueryRow(ctx, query, r.pdb.XUID).Scan(&trackPath, &rank, &progress)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		if isTableNotFoundErr(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("home_repo: cache BP query: %w", err)
	}

	resp := &domain.BattlePassResponse{
		Available:   true,
		Rank:        &rank,
		Progress:    &progress,
		RewardTrack: &trackPath,
		FromCache:   true,
	}
	return resp, true, nil
}

// challengeSnapshotRow est une ligne agrégée pour la reconstruction ChallengesResponse.
type challengeSnapshotRow struct {
	status    string
	xpReward  int
	expiresAt sql.NullTime
}

// LoadCachedChallenges retourne un résumé des snapshots récents depuis challenge_snapshots
// (la snapshot la plus récente par challenge_path dans la fenêtre ttl).
func (r *HomeRepo) LoadCachedChallenges(ctx context.Context, ttl time.Duration) (*domain.ChallengesResponse, bool, error) {
	secs := int64(ttl.Seconds())
	// Sélectionne la snapshot la plus récente par challenge_path dans la fenêtre TTL.
	query := fmt.Sprintf(`
		SELECT status, xp_reward, expires_at
		FROM (
			SELECT status, xp_reward, expires_at,
			       ROW_NUMBER() OVER (PARTITION BY challenge_path ORDER BY snapshot_at DESC) AS rn
			FROM challenge_snapshots
			WHERE xuid = ?
			  AND snapshot_at > CURRENT_TIMESTAMP - INTERVAL '%d' SECOND
		) t
		WHERE rn = 1`, secs)

	rows, err := r.pdb.Player.Query(ctx, query, r.pdb.XUID)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("home_repo: cache challenges query: %w", err)
	}
	defer rows.Close()

	var snapshots []challengeSnapshotRow
	for rows.Next() {
		var s challengeSnapshotRow
		if err := rows.Scan(&s.status, &s.xpReward, &s.expiresAt); err != nil {
			return nil, false, fmt.Errorf("home_repo: cache challenges scan: %w", err)
		}
		snapshots = append(snapshots, s)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	if len(snapshots) == 0 {
		return nil, false, nil
	}

	total := len(snapshots)
	completed := 0
	xpAvailable := 0
	var earliestExpiry *time.Time

	for _, s := range snapshots {
		if strings.EqualFold(s.status, "Completed") {
			completed++
		} else {
			xpAvailable += s.xpReward
		}
		if s.expiresAt.Valid {
			t := s.expiresAt.Time
			if earliestExpiry == nil || t.Before(*earliestExpiry) {
				earliestExpiry = &t
			}
		}
	}

	resp := &domain.ChallengesResponse{
		Available: true,
		Total:     &total,
		Completed: &completed,
		FromCache: true,
	}
	if xpAvailable > 0 {
		resp.XPAvailable = &xpAvailable
	}
	if earliestExpiry != nil {
		s := earliestExpiry.Format(time.RFC3339)
		resp.NextExpiry = &s
	}
	return resp, true, nil
}
