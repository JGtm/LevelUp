// Package duckdb — home_repo.go : accès DB pour la page d'accueil Mission Control.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
)

const homeIdentityAssetBasePath = "/api/v1/assets/spartan"

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
	var bannerImageURL sql.NullString
	var emblemImageURL sql.NullString
	var backdropImageURL sql.NullString
	var adornmentImagePath sql.NullString

	err := r.pdb.Player.QueryRow(ctx, Q26cHomeSpartanIdentity).Scan(
		&row.RankNumber,
		&row.CurrentXP,
		&row.XPForNextRank,
		&row.IsMaxRank,
		&spartanID,
		&rankName,
		&rankTier,
		&bannerImageURL,
		&emblemImageURL,
		&backdropImageURL,
		&adornmentImagePath,
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
	if bannerImageURL.Valid {
		row.BannerImageURL = buildHomeIdentityAssetURL("banner", r.titleSlug(), bannerImageURL.String)
	}
	if emblemImageURL.Valid {
		row.EmblemImageURL = buildHomeIdentityAssetURL("emblem", r.titleSlug(), emblemImageURL.String)
	}
	if backdropImageURL.Valid {
		row.BackdropImageURL = buildHomeIdentityAssetURL("backdrop", r.titleSlug(), backdropImageURL.String)
	}
	if adornmentImagePath.Valid {
		row.AdornmentImageURL = buildHomeIdentityAssetURL("career-rank", r.titleSlug(), adornmentImagePath.String)
	}

	r.enrichSpartanIdentity(ctx, &row)
	row.HighestCSR = r.loadHomeSkillPeak(ctx, "CSR")
	row.HighestLUSR = r.loadHomeSkillPeak(ctx, "LUSR")

	if row.SpartanID == nil && row.RankNumber <= 0 && row.BannerImageURL == nil && row.EmblemImageURL == nil && row.BackdropImageURL == nil && row.HighestCSR == nil && row.HighestLUSR == nil {
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
	var adornmentPath sql.NullString
	if err := r.pdb.Metadata.QueryRow(ctx, Q26dHomeCareerRankMeta, row.RankNumber).Scan(&titleEN, &titleFR, &imagePath, &adornmentPath); err != nil {
		return
	}
	if titleEN.Valid {
		row.RankTitleEN = stringPtr(titleEN.String)
	}
	if titleFR.Valid {
		row.RankTitleFR = stringPtr(titleFR.String)
	}
	if imagePath.Valid {
		row.RankImageURL = buildHomeIdentityAssetURL("career-rank", r.titleSlug(), imagePath.String)
	}
	if row.AdornmentImageURL == nil && adornmentPath.Valid {
		row.AdornmentImageURL = buildHomeIdentityAssetURL("career-rank", r.titleSlug(), adornmentPath.String)
	}
}

func (r *HomeRepo) loadHomeSkillPeak(ctx context.Context, ratingType string) *domain.HomeSkillPeakRow {
	if r == nil || r.pdb == nil || r.pdb.Player == nil {
		return nil
	}

	var ratingValue sql.NullFloat64
	var tierLabel sql.NullString
	var tier sql.NullString
	var subTier sql.NullInt16
	if err := r.pdb.Player.QueryRow(ctx, Q26eHomeSkillPeakByType, ratingType).Scan(&ratingValue, &tierLabel, &tier, &subTier); err != nil {
		if err == sql.ErrNoRows || isTableNotFoundErr(err) {
			return nil
		}
		return nil
	}
	if !ratingValue.Valid {
		return nil
	}

	peak := &domain.HomeSkillPeakRow{RatingValue: ratingValue.Float64}
	if tierLabel.Valid {
		peak.TierLabel = stringPtr(tierLabel.String)
	}
	peak.BadgeImageURL = buildHomeSkillPeakBadgeURL(optionalNullStringValue(tier), optionalNullStringValue(tierLabel), optionalNullInt16Value(subTier))
	return peak
}

// LoadRecentPlaylistRanks retourne les 3 dernières playlists distinctes jouées avec leur
// dernier rang compétitif connu (Q26g). Retourne (nil, nil) si aucune donnée.
func (r *HomeRepo) LoadRecentPlaylistRanks(ctx context.Context) ([]domain.HomePlaylistRank, error) {
	if r == nil || r.pdb == nil || r.pdb.Player == nil {
		return nil, nil
	}

	rows, err := r.pdb.Player.Query(ctx, Q26gHomePlaylistRanks, r.pdb.XUID)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var result []domain.HomePlaylistRank
	for rows.Next() {
		var playlistName sql.NullString
		var isRanked sql.NullBool
		var ratingType sql.NullString
		var ratingValue sql.NullFloat64
		var tier sql.NullString
		var tierFR sql.NullString
		var subTier sql.NullInt16
		var tierLabel sql.NullString

		if err := rows.Scan(&playlistName, &isRanked, &ratingType, &ratingValue, &tier, &tierFR, &subTier, &tierLabel); err != nil {
			return nil, err
		}

		item := domain.HomePlaylistRank{
			PlaylistName: playlistName.String,
			IsRanked:     isRanked.Bool,
		}
		if ratingValue.Valid {
			rt := strings.ToUpper(strings.TrimSpace(ratingType.String))
			item.RatingType = &rt
			item.RatingValue = &ratingValue.Float64
			if tierLabel.Valid {
				item.TierLabel = stringPtr(tierLabel.String)
			}
			item.BadgeImageURL = buildHomeSkillPeakBadgeURL(
				optionalNullStringValue(tier),
				optionalNullStringValue(tierLabel),
				optionalNullInt16Value(subTier),
			)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *HomeRepo) titleSlug() string {
	if r == nil || r.pdb == nil {
		return titlepkg.DefaultSlug
	}
	trimmed := strings.TrimSpace(r.pdb.TitleSlug)
	if trimmed == "" {
		return titlepkg.DefaultSlug
	}
	return trimmed
}

func buildHomeIdentityAssetURL(imageType string, titleID string, value string) *string {
	cleaned := normalizeHomeIdentityAssetPath(value)
	if cleaned == "" {
		return nil
	}

	resolved := fmt.Sprintf("%s/%s/%s/%s", homeIdentityAssetBasePath, imageType, titleID, cleaned)
	return &resolved
}

func normalizeHomeIdentityAssetPath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	lower := strings.ToLower(trimmed)
	if strings.HasSuffix(lower, ".json") {
		return ""
	}
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return ""
		}
		trimmed = strings.TrimSpace(parsed.Path)
	}

	cleaned := path.Clean(strings.TrimLeft(trimmed, "/"))
	if cleaned == "" || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return ""
	}
	return cleaned
}

func stringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalNullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return strings.TrimSpace(value.String)
}

func optionalNullInt16Value(value sql.NullInt16) int {
	if !value.Valid {
		return 0
	}
	return int(value.Int16)
}

func buildHomeSkillPeakBadgeURL(tier string, tierLabel string, subTier int) *string {
	normalizedTier, normalizedSubTier := normalizeHomeSkillPeakBadgeParts(tier, tierLabel, subTier)
	if normalizedTier == "" {
		return nil
	}
	if strings.EqualFold(normalizedTier, "Onyx") {
		path := "/static/ranks/120px-HINF-CSR_Onyx.png"
		return &path
	}
	if normalizedSubTier < 1 || normalizedSubTier > 6 {
		return nil
	}
	path := fmt.Sprintf("/static/ranks/120px-HINF-CSR_%s%d.png", normalizedTier, normalizedSubTier)
	return &path
}

func normalizeHomeSkillPeakBadgeParts(tier string, tierLabel string, subTier int) (string, int) {
	normalizedTier := canonicalHomeSkillTierName(tier)
	derivedSubTier := subTier
	if normalizedTier == "" && strings.TrimSpace(tierLabel) != "" {
		parts := strings.Fields(strings.TrimSpace(tierLabel))
		if len(parts) > 0 {
			normalizedTier = canonicalHomeSkillTierName(parts[0])
		}
		if derivedSubTier <= 0 && len(parts) > 1 {
			derivedSubTier = parseHomeSkillPeakSubTier(parts[len(parts)-1])
		}
	}
	return normalizedTier, derivedSubTier
}

func canonicalHomeSkillTierName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bronze":
		return "Bronze"
	case "silver":
		return "Silver"
	case "gold":
		return "Gold"
	case "platinum":
		return "Platinum"
	case "diamond":
		return "Diamond"
	case "onyx":
		return "Onyx"
	default:
		return ""
	}
}

func parseHomeSkillPeakSubTier(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	if numeric, err := strconv.Atoi(trimmed); err == nil {
		return numeric
	}
	switch strings.ToUpper(trimmed) {
	case "I":
		return 1
	case "II":
		return 2
	case "III":
		return 3
	case "IV":
		return 4
	case "V":
		return 5
	case "VI":
		return 6
	default:
		return 0
	}
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
