// Package duckdb — season_pass_repo.go : accès DB pour la page Season Pass.
package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// SeasonPassRepo implémente port.SeasonPassRepository.
type SeasonPassRepo struct {
	pdb *PlayerDB
}

// NewSeasonPassRepo crée un SeasonPassRepo pour un joueur.
func NewSeasonPassRepo(pdb *PlayerDB) *SeasonPassRepo {
	return &SeasonPassRepo{pdb: pdb}
}

// localBPImageURL retourne l'URL proxy locale incluant le chemin GameCMS complet.
// Le handler décodera ce chemin pour construire l'URL GameCMS exacte lors du fetch.
// Ne retourne jamais une URL GameCMS directe — le browser ne peut pas la charger.
func localBPImageURL(gameCMSPath, subDir string) *string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(gameCMSPath, "\\", "/"))
	if trimmed == "" {
		return nil
	}
	// Vérification de sécurité : pas de traversal de répertoire.
	cleaned := path.Clean("/" + strings.TrimLeft(trimmed, "/"))
	if cleaned == "/" || cleaned == "." {
		return nil
	}
	// Toujours retourner l'URL proxy — Go gérera le cache ou le fetch GameCMS.
	// Le chemin complet est inclus pour que le handler construise la bonne URL GameCMS.
	u := "/api/v1/assets/battlepass/" + subDir + cleaned
	return &u
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

	rows, err := r.pdb.ReadDB().Query(ctx, `
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
	SummaryImagePath    string              `json:"SummaryImagePath"`
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
	CurrencyRewards  []seasonPassCurrencyReward  `json:"CurrencyRewards"`
}

type seasonPassInventoryReward struct {
	InventoryItemPath string `json:"InventoryItemPath"`
}

// seasonPassCurrencyReward représente une récompense en monnaie virtuelle (cR, XP boost…).
// Certains paliers de Battle Pass donnent uniquement de la monnaie, sans InventoryItem.
type seasonPassCurrencyReward struct {
	CurrencyPath string `json:"CurrencyPath"`
	Amount       int    `json:"Amount"`
}

type seasonPassItemMeta struct {
	Title       string
	Description *string
	ImageURL    *string
	Quality     *string
	ItemType    *string
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
		      AND t_fr.lang = 'fr-FR'
		LEFT JOIN battlepass_track_translations t_en
		       ON t_en.reward_track_path = d.reward_track_path
		      AND t_en.content_hash = d.content_hash
		      AND t_en.lang = 'en-US'
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

	// Fallback : si battlepass_track_definitions est vide mais que des snapshots
	// de progression existent en DB joueur, construire des résumés minimaux.
	// Cela évite l'état "Non disponible" quand les définitions Waypoint n'ont jamais
	// été persistées (ex: aucun appel live Go authentifié), mais que des données
	// de progression sont déjà présentes depuis un sync Python antérieur.
	if len(trackRows) == 0 && len(progressMap) > 0 {
		for path, state := range progressMap {
			state.IsActive = path == activeTrackPath
			tracks = append(tracks, buildMinimalTrackSummary(path, state))
		}
		sort.Slice(tracks, func(i, j int) bool {
			if tracks[i].IsActive != tracks[j].IsActive {
				return tracks[i].IsActive
			}
			return tracks[i].CurrentRank > tracks[j].CurrentRank
		})
	}

	return tracks, nil
}

// buildMinimalTrackSummary construit un SeasonPassTrackSummary depuis battlepass_snapshots
// uniquement, quand battlepass_track_definitions est vide (aucun appel Waypoint persisté).
func buildMinimalTrackSummary(path string, state trackSnapshotState) domain.SeasonPassTrackSummary {
	name := path
	if parts := strings.Split(path, "/"); len(parts) > 0 {
		name = parts[len(parts)-1]
	}
	status := computeSeasonPassStatus(state)
	isOwned := state.IsOwned || state.Rank > 0 || state.IsActive
	return domain.SeasonPassTrackSummary{
		RewardTrackPath:   path,
		Name:              name,
		Status:            status,
		IsActive:          state.IsActive,
		IsOwned:           isOwned,
		HasReachedMaxRank: state.HasReachedMaxRank,
		CurrentRank:       state.Rank,
		PartialProgress:   state.Partial,
	}
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
			SELECT inventory_item_path, content_hash, quality, item_type, display_path, last_seen_at,
			       ROW_NUMBER() OVER (PARTITION BY inventory_item_path ORDER BY last_seen_at DESC) AS rn
			FROM battlepass_item_definitions
			WHERE is_current = TRUE
		)
		SELECT d.inventory_item_path,
		       d.display_path,
		       COALESCE(t_fr.title, t_en.title)       AS title,
		       COALESCE(t_fr.description, t_en.description) AS description,
		       d.quality,
		       d.item_type
		FROM latest d
		LEFT JOIN battlepass_item_translations t_fr
		       ON t_fr.inventory_item_path = d.inventory_item_path
		      AND t_fr.content_hash = d.content_hash
		      AND t_fr.lang = 'fr-FR'
		LEFT JOIN battlepass_item_translations t_en
		       ON t_en.inventory_item_path = d.inventory_item_path
		      AND t_en.content_hash = d.content_hash
		      AND t_en.lang = 'en-US'
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
		var quality sql.NullString
		var itemType sql.NullString
		if err := rows.Scan(&itemPath, &displayPath, &title, &description, &quality, &itemType); err != nil {
			return nil, fmt.Errorf("season_pass_repo: items scan: %w", err)
		}
		if !itemPath.Valid || itemPath.String == "" {
			continue
		}
		itemMap[itemPath.String] = seasonPassItemMeta{
			Title:       coalesceNullString(title),
			Description: nullStringPtr(description),
			ImageURL:    localBPImageURL(coalesceNullString(displayPath), "tier"),
			Quality:     nullStringPtr(quality),
			ItemType:    nullStringPtr(itemType),
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fallback : pour les items absents de battlepass_item_definitions (ex: joueurs sans
	// investigation JSON), chercher dans asset_index où warmBPTrackAssets stocke les JSONs
	// fetches en arrière-plan (kind='track-def').
	missing := make([]string, 0, len(itemPaths))
	for _, p := range itemPaths {
		if _, found := itemMap[p]; !found {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		r.fillItemsFromAssetIndex(ctx, missing, itemMap)
	}

	return itemMap, nil
}

// fillItemsFromAssetIndex complète itemMap avec les items présents dans asset_index
// (stockés par warmBPTrackAssets lors d'appels précédents).
// Cherche d'abord dans le nouveau kind 'bp-item-def', puis dans l'ancien 'track-def'
// pour la rétrocompatibilité avec les items mis en cache avant ce déploiement.
// TODO(expiry:2026-08-01): supprimer 'track-def' de la liste une fois tous les items
// migrés vers 'bp-item-def' via le live flow ou le backfill.
// Best-effort : toute erreur est silencieusement ignorée.
func (r *SeasonPassRepo) fillItemsFromAssetIndex(
	ctx context.Context,
	paths []string,
	itemMap map[string]seasonPassItemMeta,
) {
	placeholders := strings.TrimRight(strings.Repeat("?,", len(paths)), ",")
	query := fmt.Sprintf(`
		SELECT id,
		       json_extract_string(raw_json, '$.CommonData.DisplayPath.Media.MediaUrl.Path') AS display_path,
		       COALESCE(
		           json_extract_string(raw_json, '$.CommonData.Title.translations.fr-FR'),
		           json_extract_string(raw_json, '$.CommonData.Title.translations.en-US'),
		           json_extract_string(raw_json, '$.CommonData.Title.value')
		       )                                                                            AS title,
		       COALESCE(
		           json_extract_string(raw_json, '$.CommonData.Description.translations.fr-FR'),
		           json_extract_string(raw_json, '$.CommonData.Description.value')
		       )                                                                            AS description,
		       json_extract_string(raw_json, '$.CommonData.Quality')                        AS quality,
		       COALESCE(
		           json_extract_string(raw_json, '$.CommonData.Type'),
		           json_extract_string(raw_json, '$.CommonData.ItemType')
		       )                                                                            AS item_type
		FROM asset_index
		WHERE kind IN ('bp-item-def', 'track-def')
		  AND id IN (%s)
		  AND raw_json IS NOT NULL`, placeholders)

	args := make([]any, len(paths))
	for i, p := range paths {
		args[i] = p
	}

	rows, err := r.pdb.Metadata.Query(ctx, query, args...)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id sql.NullString
		var displayPath sql.NullString
		var title sql.NullString
		var description sql.NullString
		var quality sql.NullString
		var itemType sql.NullString
		if err := rows.Scan(&id, &displayPath, &title, &description, &quality, &itemType); err != nil {
			continue
		}
		if !id.Valid || id.String == "" {
			continue
		}
		if _, already := itemMap[id.String]; already {
			continue
		}
		itemMap[id.String] = seasonPassItemMeta{
			Title:       coalesceNullString(title),
			Description: nullStringPtr(description),
			ImageURL:    localBPImageURL(coalesceNullString(displayPath), "tier"),
			Quality:     nullStringPtr(quality),
			ItemType:    nullStringPtr(itemType),
		}
	}
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
		Quality:     meta.Quality,
		ItemType:    meta.ItemType,
		IsObtained:  rank.Rank <= state.Rank,
		IsCurrent:   rank.Rank == activeTierRank,
		IsPremium:   isPremium,
		FreeRewards: buildFreeRewardSummaries(rank.FreeRewards, itemMap),
	}
}

// buildFreeRewardSummaries construit la liste des récompenses gratuites d'un palier.
// Retourne nil si le bucket est vide (omitempty côté JSON).
func buildFreeRewardSummaries(
	bucket seasonPassRewardBucket,
	itemMap map[string]seasonPassItemMeta,
) []domain.SeasonPassItemSummary {
	var items []domain.SeasonPassItemSummary
	for _, reward := range bucket.InventoryRewards {
		path := strings.TrimSpace(reward.InventoryItemPath)
		if path == "" {
			continue
		}
		meta, ok := itemMap[path]
		if !ok {
			continue
		}
		title := meta.Title
		if title == "" {
			title = path
		}
		items = append(items, domain.SeasonPassItemSummary{
			Title:       title,
			Description: meta.Description,
			ImageURL:    meta.ImageURL,
			Quality:     meta.Quality,
			ItemType:    meta.ItemType,
		})
	}
	// Currency rewards gratuits
	for _, reward := range bucket.CurrencyRewards {
		meta, ok := currencyRewardMeta(reward)
		if !ok {
			continue
		}
		items = append(items, domain.SeasonPassItemSummary{
			Title:    meta.Title,
			ImageURL: meta.ImageURL,
		})
	}
	return items
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
	// 1. Inventory items (cosmétiques, armes…) avec méta résolue.
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
	// 2. Fallback : currency rewards (cR, xpboost, rerollcurrency, softcurrency).
	// Pour les paliers « purement monnaie », on rend le tier visible avec un titre
	// localisé et, quand l'asset existe localement, une miniature.
	for _, entry := range buckets {
		for _, reward := range entry.bucket.CurrencyRewards {
			if meta, ok := currencyRewardMeta(reward); ok {
				return meta, entry.isPremium
			}
		}
	}
	return seasonPassItemMeta{}, false
}

// currencyImagePath mappe le slug d'une currency (basename lowercase du
// CurrencyPath GameCMS, ex: "xpboost") vers son image officielle GameCMS,
// telle que publiée dans /hi/Progression/file/metadata/metadata.json
// (cf. https://den.dev/blog/halo-infinite-exchange-spartan-points/).
//
// Ces chemins sont relayés par le proxy /api/v1/assets/battlepass/tracks/{path}
// qui les télécharge à la 1ère demande via le resolver puis sert depuis le
// cache fichier — aucun asset à committer dans static/.
var currencyImagePath = map[string]string{
	"cr":             "progression/Currencies/Credit_Coin-SM.png",
	"softcurrency":   "progression/StoreContent/ToggleTiles/SpartanPoints_Common_4x4.png",
	"rerollcurrency": "progression/Currencies/1104-000-data-pad-e39bef84-2x2.png",
	"xpboost":        "progression/Currencies/1103-000-xp-boost-5e92621a-2x2.png",
	"xpgrant":        "progression/Currencies/1102-000-xp-grant-c77c6396-2x2.png",
}

// currencyTitleFR retourne le libellé français d'une currency, par slug lowercase.
func currencyTitleFR(slug string, amount int) string {
	var label string
	switch slug {
	case "cr":
		label = "Crédits"
	case "softcurrency":
		label = "Crédits Spartan"
	case "xpboost":
		label = "Boost XP"
	case "rerollcurrency":
		label = "Relance défi"
	default:
		if slug == "" {
			return ""
		}
		label = slug
	}
	if amount > 0 {
		return fmt.Sprintf("%d × %s", amount, label)
	}
	return label
}

// currencyRewardMeta convertit une CurrencyReward en seasonPassItemMeta.
// Retourne ok=false si le CurrencyPath est vide.
func currencyRewardMeta(reward seasonPassCurrencyReward) (seasonPassItemMeta, bool) {
	cleanPath := strings.TrimSpace(reward.CurrencyPath)
	if cleanPath == "" {
		return seasonPassItemMeta{}, false
	}
	slug := currencySlug(cleanPath)
	title := currencyTitleFR(slug, reward.Amount)
	meta := seasonPassItemMeta{Title: title}
	if imgPath, ok := currencyImagePath[slug]; ok {
		meta.ImageURL = localBPImageURL(imgPath, "tracks")
	}
	return meta, true
}

// currencySlug extrait le slug lowercase d'un CurrencyPath
// (ex: "Currency/Currencies/xpboost.json" → "xpboost").
func currencySlug(currencyPath string) string {
	base := path.Base(strings.ReplaceAll(currencyPath, "\\", "/"))
	if idx := strings.LastIndexByte(base, '.'); idx > 0 {
		base = base[:idx]
	}
	return strings.ToLower(strings.TrimSpace(base))
}

func resolveTrackImageURL(row seasonPassTrackRow, payload *seasonPassTrackPayload) *string {
	p := coalesceNullString(row.battlepassImagePath)
	if p == "" && payload != nil {
		p = payload.BattlePassImage
		if p == "" {
			p = payload.SummaryImagePath
		}
	}
	return localBPImageURL(p, "tracks")
}

func resolveTrackBackgroundURL(row seasonPassTrackRow, payload *seasonPassTrackPayload) *string {
	p := coalesceNullString(row.backgroundImagePath)
	if p == "" && payload != nil {
		p = payload.BackgroundImagePath
	}
	return localBPImageURL(p, "background")
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
		// Format résolu Halo : {"value": "Operation: Ground Zero", "status": "Resolved"}
		// La clé "value" contient le texte déjà localisé.
		if v, ok := typed["value"].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
		// Clés de langue explicites (stockées par le système de traduction interne).
		for _, key := range []string{"fr", "en", "default"} {
			candidate, ok := typed[key].(string)
			if ok && strings.TrimSpace(candidate) != "" {
				return strings.TrimSpace(candidate)
			}
		}
		// Pas de fallback générique sur toutes les valeurs : évite de retourner des
		// champs méta Halo comme "Status": "Ready" ou "StringId": "...".
	}
	return ""
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
