// Package duckdb — season_pass_repo.go : accès DB pour la page Season Pass.
package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

const gameCMSImageBaseURL = "https://gamecms-hacs.svc.halowaypoint.com/hi/images/file/"

// SeasonPassRepo implémente port.SeasonPassRepository.
type SeasonPassRepo struct {
	pdb *PlayerDB
}

// NewSeasonPassRepo crée un SeasonPassRepo pour un joueur.
func NewSeasonPassRepo(pdb *PlayerDB) *SeasonPassRepo {
	return &SeasonPassRepo{pdb: pdb}
}

// trackSnapshotState est l'état le plus récent d'un reward track pour un joueur.
type trackSnapshotState struct {
	Rank              int
	Partial           int
	IsOwned           bool
	HasReachedMaxRank bool
	IsActive          bool
}

// trackProgressMap mappe reward_track_path → progression joueur récente.
type trackProgressMap map[string]trackSnapshotState

// loadTrackSnapshots charge la progression du joueur depuis battlepass_snapshots.
// Retourne une map vide (sans erreur) si aucune entrée n'existe.
func (r *SeasonPassRepo) loadTrackSnapshots(ctx context.Context) (trackProgressMap, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.pdb.Player.Query(ctx, `
		SELECT reward_track_path, is_active, current_rank, partial_progress,
		       is_owned, has_reached_max_rank, snapshot_at
		FROM (
			SELECT reward_track_path, is_active, current_rank, partial_progress,
			       is_owned, has_reached_max_rank, snapshot_at,
			       ROW_NUMBER() OVER (PARTITION BY reward_track_path ORDER BY snapshot_at DESC) AS rn
			FROM battlepass_snapshots
			WHERE xuid = ?
		) t
		WHERE rn = 1`, r.pdb.XUID)
	if err != nil {
		if isTableNotFoundErr(err) {
			return trackProgressMap{}, "", nil
		}
		return nil, "", fmt.Errorf("season_pass_repo: track snapshots query: %w", err)
	}
	defer rows.Close()

	progressMap := trackProgressMap{}
	activeTrackPath := ""
	var activeSeenAt *time.Time
	for rows.Next() {
		var path string
		var state trackSnapshotState
		var snapshotAt time.Time
		if err := rows.Scan(
			&path,
			&state.IsActive,
			&state.Rank,
			&state.Partial,
			&state.IsOwned,
			&state.HasReachedMaxRank,
			&snapshotAt,
		); err != nil {
			return nil, "", fmt.Errorf("season_pass_repo: track snapshots scan: %w", err)
		}
		progressMap[path] = state
		if state.IsActive && (activeSeenAt == nil || snapshotAt.After(*activeSeenAt)) {
			t := snapshotAt
			activeSeenAt = &t
			activeTrackPath = path
		}
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return progressMap, activeTrackPath, nil
}

// seasonPassTrackRow représente une ligne du JOIN tracks + translations.
type seasonPassTrackRow struct {
	rewardTrackPath     string
	xpPerRank           sql.NullInt64
	trackName           sql.NullString
	battlepassImagePath sql.NullString
	backgroundImagePath sql.NullString
	rawPayloadJSON      sql.NullString
}

type seasonPassTrackPayload struct {
	Name                any                 `json:"Name"`
	Description         any                 `json:"Description"`
	XpPerRank           int                 `json:"XpPerRank"`
	BattlePassImage     string              `json:"BattlePassImage"`
	BackgroundImagePath string              `json:"BackgroundImagePath"`
	Ranks               []seasonPassRankRaw `json:"Ranks"`
}

type seasonPassRankRaw struct {
	Rank        int                    `json:"Rank"`
	FreeRewards seasonPassRewardBucket `json:"FreeRewards"`
	PaidRewards seasonPassRewardBucket `json:"PaidRewards"`
}

type seasonPassRewardBucket struct {
	InventoryRewards []seasonPassInventoryReward `json:"InventoryRewards"`
}

type seasonPassInventoryReward struct {
	InventoryItemPath string `json:"InventoryItemPath"`
}

type seasonPassItemMeta struct {
	Title       string
	Description *string
	ImageURL    *string
}

// LoadSeasonPassTracks charge toutes les tracks connues avec traductions.
// La progression joueur est injectée depuis le payload Waypoint persisté.
func (r *SeasonPassRepo) LoadSeasonPassTracks(ctx context.Context, _, _ string) ([]domain.SeasonPassTrackSummary, error) {
	progressMap, activeTrackPath, err := r.loadTrackSnapshots(ctx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Récupère la définition la plus récente par track + traduction FR (fallback EN).
	const query = `
		WITH latest AS (
			SELECT reward_track_path, content_hash, xp_per_rank, is_current, last_seen_at,
			       battlepass_image_path, background_image_path, raw_payload_json,
			       ROW_NUMBER() OVER (PARTITION BY reward_track_path ORDER BY last_seen_at DESC) AS rn
			FROM battlepass_track_definitions
		)
		SELECT d.reward_track_path, d.xp_per_rank,
		       COALESCE(t_fr.track_name, t_en.track_name) AS track_name
		       , d.battlepass_image_path, d.background_image_path, d.raw_payload_json
		FROM latest d
		LEFT JOIN battlepass_track_translations t_fr
		       ON t_fr.reward_track_path = d.reward_track_path
		      AND t_fr.content_hash = d.content_hash
		      AND t_fr.lang = 'fr'
		LEFT JOIN battlepass_track_translations t_en
		       ON t_en.reward_track_path = d.reward_track_path
		      AND t_en.content_hash = d.content_hash
		      AND t_en.lang = 'en'
		WHERE d.rn = 1
		ORDER BY d.is_current DESC, d.last_seen_at DESC`

	rows, err := r.pdb.Metadata.Query(ctx, query)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("season_pass_repo: tracks query: %w", err)
	}
	defer rows.Close()

	var tracks []domain.SeasonPassTrackSummary
	itemPaths := map[string]struct{}{}
	trackRows := make([]seasonPassTrackRow, 0)
	for rows.Next() {
		var row seasonPassTrackRow
		if err := rows.Scan(
			&row.rewardTrackPath,
			&row.xpPerRank,
			&row.trackName,
			&row.battlepassImagePath,
			&row.backgroundImagePath,
			&row.rawPayloadJSON,
		); err != nil {
			return nil, fmt.Errorf("season_pass_repo: scan: %w", err)
		}
		trackRows = append(trackRows, row)
		for _, itemPath := range collectTrackItemPaths(parseTrackPayload(row.rawPayloadJSON)) {
			itemPaths[itemPath] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	itemMap, err := r.loadItemMetadataMap(ctx, mapKeys(itemPaths))
	if err != nil {
		return nil, err
	}

	for _, row := range trackRows {
		prog := progressMap[row.rewardTrackPath]
		prog.IsActive = row.rewardTrackPath == activeTrackPath
		summary := buildTrackSummary(row, prog, itemMap)
		tracks = append(tracks, summary)
	}
	return tracks, nil
}

func (r *SeasonPassRepo) loadItemMetadataMap(
	ctx context.Context,
	itemPaths []string,
) (map[string]seasonPassItemMeta, error) {
	if len(itemPaths) == 0 {
		return map[string]seasonPassItemMeta{}, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(itemPaths)), ",")
	query := fmt.Sprintf(`
		WITH latest AS (
			SELECT inventory_item_path, content_hash, display_path, last_seen_at,
			       ROW_NUMBER() OVER (PARTITION BY inventory_item_path ORDER BY last_seen_at DESC) AS rn
			FROM battlepass_item_definitions
			WHERE is_current = TRUE
		)
		SELECT d.inventory_item_path,
		       d.display_path,
		       COALESCE(t_fr.title, t_en.title) AS title,
		       COALESCE(t_fr.description, t_en.description) AS description
		FROM latest d
		LEFT JOIN battlepass_item_translations t_fr
		       ON t_fr.inventory_item_path = d.inventory_item_path
		      AND t_fr.content_hash = d.content_hash
		      AND t_fr.lang = 'fr'
		LEFT JOIN battlepass_item_translations t_en
		       ON t_en.inventory_item_path = d.inventory_item_path
		      AND t_en.content_hash = d.content_hash
		      AND t_en.lang = 'en'
		WHERE d.rn = 1 AND d.inventory_item_path IN (%s)`, placeholders)

	args := make([]any, 0, len(itemPaths))
	for _, itemPath := range itemPaths {
		args = append(args, itemPath)
	}

	rows, err := r.pdb.Metadata.Query(ctx, query, args...)
	if err != nil {
		if isTableNotFoundErr(err) {
			return map[string]seasonPassItemMeta{}, nil
		}
		return nil, fmt.Errorf("season_pass_repo: items query: %w", err)
	}
	defer rows.Close()

	itemMap := make(map[string]seasonPassItemMeta, len(itemPaths))
	for rows.Next() {
		var itemPath sql.NullString
		var displayPath sql.NullString
		var title sql.NullString
		var description sql.NullString
		if err := rows.Scan(&itemPath, &displayPath, &title, &description); err != nil {
			return nil, fmt.Errorf("season_pass_repo: items scan: %w", err)
		}
		if !itemPath.Valid || itemPath.String == "" {
			continue
		}
		itemMap[itemPath.String] = seasonPassItemMeta{
			Title:       coalesceNullString(title),
			Description: nullStringPtr(description),
			ImageURL:    normalizeGameCMSImageURL(coalesceNullString(displayPath)),
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return itemMap, nil
}

// buildTrackSummary construit un SeasonPassTrackSummary depuis les données brutes.
func buildTrackSummary(
	row seasonPassTrackRow,
	state trackSnapshotState,
	itemMap map[string]seasonPassItemMeta,
) domain.SeasonPassTrackSummary {
	payload := parseTrackPayload(row.rawPayloadJSON)
	name := row.rewardTrackPath
	if row.trackName.Valid && row.trackName.String != "" {
		name = row.trackName.String
	} else if payloadName := localizedText(payloadNameValue(payload)); payloadName != "" {
		name = payloadName
	}
	description := payloadDescription(payload)
	xpPerRank := resolveXPPerRank(row, payload)
	maxRank := resolveMaxRank(payload, state)
	tiers, activeTierRank := buildTierSummaries(payload, itemMap, state)
	activeTierProgressPercent := computeActiveTierProgressPercent(state, xpPerRank, activeTierRank)

	status := computeSeasonPassStatus(state)
	isOwned := state.IsOwned || state.Rank > 0 || state.IsActive

	s := domain.SeasonPassTrackSummary{
		RewardTrackPath:           row.rewardTrackPath,
		Name:                      name,
		Description:               description,
		Status:                    status,
		IsActive:                  state.IsActive,
		IsOwned:                   isOwned,
		HasReachedMaxRank:         state.HasReachedMaxRank,
		CurrentRank:               state.Rank,
		PartialProgress:           state.Partial,
		MaxRank:                   maxRank,
		CompletionPercent:         computeCompletionPercent(state, maxRank),
		ActiveTierRank:            activeTierRank,
		ActiveTierProgressPercent: activeTierProgressPercent,
		Tiers:                     tiers,
	}
	if xpPerRank != nil {
		v := *xpPerRank
		s.XPPerRank = &v
	}
	if imageURL := resolveTrackImageURL(row, payload); imageURL != nil {
		s.ImageURL = imageURL
	}
	if backgroundURL := resolveTrackBackgroundURL(row, payload); backgroundURL != nil {
		s.BackgroundImageURL = backgroundURL
	}
	return s
}

func parseTrackPayload(raw sql.NullString) *seasonPassTrackPayload {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}
	var payload seasonPassTrackPayload
	if err := json.Unmarshal([]byte(raw.String), &payload); err != nil {
		return nil
	}
	if len(payload.Ranks) == 0 && payload.BattlePassImage == "" && payload.BackgroundImagePath == "" {
		return nil
	}
	return &payload
}

func collectTrackItemPaths(payload *seasonPassTrackPayload) []string {
	if payload == nil || len(payload.Ranks) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	for _, rank := range payload.Ranks {
		collectRewardBucketItemPaths(rank.FreeRewards, seen)
		collectRewardBucketItemPaths(rank.PaidRewards, seen)
	}
	return mapKeys(seen)
}

func collectRewardBucketItemPaths(bucket seasonPassRewardBucket, seen map[string]struct{}) {
	for _, reward := range bucket.InventoryRewards {
		path := strings.TrimSpace(reward.InventoryItemPath)
		if path == "" {
			continue
		}
		seen[path] = struct{}{}
	}
}

func buildTierSummaries(
	payload *seasonPassTrackPayload,
	itemMap map[string]seasonPassItemMeta,
	state trackSnapshotState,
) ([]domain.SeasonPassTierSummary, *int) {
	if payload == nil || len(payload.Ranks) == 0 {
		return nil, nil
	}
	ranks := append([]seasonPassRankRaw(nil), payload.Ranks...)
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].Rank < ranks[j].Rank
	})
	maxRank := ranks[len(ranks)-1].Rank
	activeTierRank := resolveActiveTierRank(state, maxRank)
	tiers := make([]domain.SeasonPassTierSummary, 0, len(ranks))
	for _, rank := range ranks {
		tiers = append(tiers, buildTierSummary(rank, itemMap, state, activeTierRank))
	}
	return tiers, intPtr(activeTierRank)
}

func buildTierSummary(
	rank seasonPassRankRaw,
	itemMap map[string]seasonPassItemMeta,
	state trackSnapshotState,
	activeTierRank int,
) domain.SeasonPassTierSummary {
	meta, isPremium := selectTierPreview(rank, itemMap, state.IsOwned)
	title := meta.Title
	if title == "" {
		title = fmt.Sprintf("Palier %d", rank.Rank)
	}
	return domain.SeasonPassTierSummary{
		Rank:        rank.Rank,
		Title:       title,
		Description: meta.Description,
		ImageURL:    meta.ImageURL,
		IsObtained:  rank.Rank <= state.Rank,
		IsCurrent:   rank.Rank == activeTierRank,
		IsPremium:   isPremium,
	}
}

func selectTierPreview(
	rank seasonPassRankRaw,
	itemMap map[string]seasonPassItemMeta,
	preferPremium bool,
) (seasonPassItemMeta, bool) {
	buckets := []struct {
		bucket    seasonPassRewardBucket
		isPremium bool
	}{
		{bucket: rank.FreeRewards, isPremium: false},
		{bucket: rank.PaidRewards, isPremium: true},
	}
	if preferPremium {
		buckets[0], buckets[1] = buckets[1], buckets[0]
	}
	for _, entry := range buckets {
		for _, reward := range entry.bucket.InventoryRewards {
			path := strings.TrimSpace(reward.InventoryItemPath)
			if path == "" {
				continue
			}
			meta, ok := itemMap[path]
			if ok {
				return meta, entry.isPremium
			}
		}
	}
	return seasonPassItemMeta{}, false
}

func resolveTrackImageURL(row seasonPassTrackRow, payload *seasonPassTrackPayload) *string {
	if url := normalizeGameCMSImageURL(coalesceNullString(row.battlepassImagePath)); url != nil {
		return url
	}
	if payload == nil {
		return nil
	}
	return normalizeGameCMSImageURL(payload.BattlePassImage)
}

func resolveTrackBackgroundURL(row seasonPassTrackRow, payload *seasonPassTrackPayload) *string {
	if url := normalizeGameCMSImageURL(coalesceNullString(row.backgroundImagePath)); url != nil {
		return url
	}
	if payload == nil {
		return nil
	}
	return normalizeGameCMSImageURL(payload.BackgroundImagePath)
}

func resolveXPPerRank(row seasonPassTrackRow, payload *seasonPassTrackPayload) *int {
	if row.xpPerRank.Valid {
		return intPtr(int(row.xpPerRank.Int64))
	}
	if payload != nil && payload.XpPerRank > 0 {
		return intPtr(payload.XpPerRank)
	}
	return nil
}

func resolveMaxRank(payload *seasonPassTrackPayload, state trackSnapshotState) *int {
	maxRank := 0
	if payload != nil {
		for _, rank := range payload.Ranks {
			if rank.Rank > maxRank {
				maxRank = rank.Rank
			}
		}
	}
	if maxRank == 0 {
		maxRank = state.Rank
	}
	if maxRank <= 0 {
		return nil
	}
	return intPtr(maxRank)
}

func computeCompletionPercent(state trackSnapshotState, maxRank *int) *float64 {
	if maxRank == nil || *maxRank <= 0 {
		return nil
	}
	if state.HasReachedMaxRank {
		return floatPtr(100)
	}
	percent := float64(state.Rank) / float64(*maxRank) * 100
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return floatPtr(percent)
}

func computeActiveTierProgressPercent(
	state trackSnapshotState,
	xpPerRank *int,
	activeTierRank *int,
) *float64 {
	if activeTierRank == nil || xpPerRank == nil || *xpPerRank <= 0 {
		return nil
	}
	if state.HasReachedMaxRank {
		return floatPtr(100)
	}
	percent := float64(state.Partial) / float64(*xpPerRank) * 100
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return floatPtr(percent)
}

func resolveActiveTierRank(state trackSnapshotState, maxRank int) int {
	if maxRank <= 0 {
		return 0
	}
	if state.HasReachedMaxRank {
		return maxRank
	}
	activeTierRank := state.Rank + 1
	if activeTierRank <= 0 {
		activeTierRank = 1
	}
	if activeTierRank > maxRank {
		activeTierRank = maxRank
	}
	return activeTierRank
}

func payloadNameValue(payload *seasonPassTrackPayload) any {
	if payload == nil {
		return nil
	}
	return payload.Name
}

func payloadDescription(payload *seasonPassTrackPayload) *string {
	if payload == nil {
		return nil
	}
	value := localizedText(payload.Description)
	if value == "" {
		return nil
	}
	return &value
}

func localizedText(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		for _, key := range []string{"fr", "en", "default"} {
			candidate, ok := typed[key].(string)
			if ok && strings.TrimSpace(candidate) != "" {
				return strings.TrimSpace(candidate)
			}
		}
		for _, raw := range typed {
			candidate, ok := raw.(string)
			if ok && strings.TrimSpace(candidate) != "" {
				return strings.TrimSpace(candidate)
			}
		}
	}
	return ""
}

func normalizeGameCMSImageURL(path string) *string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return &trimmed
	}
	trimmed = strings.TrimLeft(trimmed, "/")
	if trimmed == "" {
		return nil
	}
	url := gameCMSImageBaseURL + trimmed
	return &url
}

func coalesceNullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullStringPtr(value sql.NullString) *string {
	text := strings.TrimSpace(coalesceNullString(value))
	if text == "" {
		return nil
	}
	return &text
}

func mapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func intPtr(value int) *int {
	return &value
}

func floatPtr(value float64) *float64 {
	return &value
}

// computeSeasonPassStatus détermine le statut d'un track depuis les indicateurs connus.
func computeSeasonPassStatus(state trackSnapshotState) domain.SeasonPassStatus {
	if state.HasReachedMaxRank {
		return domain.SeasonPassStatusCompleted
	}
	if state.IsActive {
		return domain.SeasonPassStatusActive
	}
	if state.Rank > 0 || state.Partial > 0 {
		return domain.SeasonPassStatusInProgress
	}
	return domain.SeasonPassStatusNotStarted
}
