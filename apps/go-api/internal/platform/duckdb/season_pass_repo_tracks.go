// Package duckdb - season_pass_repo_tracks.go : LoadSeasonPassTracks + helpers
// de chargement metadata items. Decoupe de season_pass_repo.go (god-file
// split, refactor 2026-05-27).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// bpPreferEN indique si la locale de requête (header X-LevelUp-Locale → ctxkeys)
// privilégie l'anglais pour les libellés de contenu Battle Pass. GameCMS stocke
// les traductions fr-FR ET en-US (battlepass_{track,item}_translations) : on
// ordonne le COALESCE selon la locale, au lieu du FR-first figé historique
// (« Rewards des battlepass pas traduits en ENG »).
func bpPreferEN(ctx context.Context) bool {
	return strings.EqualFold(ctxkeys.Locale(ctx), "en")
}

// bpLocaleOrderedCoalesce ordonne un COALESCE par préférence de locale : les sources
// de la langue préférée d'abord, puis les fallbacks de l'autre langue. enExprs = sources
// ANGLAISES (dont value = chaîne canonique en-US Waypoint), frExprs = sources françaises.
// Centralise l'ordre locale-aware des libellés d'items Battle Pass (loadItemMetadataMap
// + fillItemsFromAssetIndex) — évite une 3e copie divergente de l'ordonnancement.
func bpLocaleOrderedCoalesce(preferEN bool, enExprs, frExprs []string) string {
	ordered := make([]string, 0, len(enExprs)+len(frExprs))
	if preferEN {
		ordered = append(append(ordered, enExprs...), frExprs...)
	} else {
		ordered = append(append(ordered, frExprs...), enExprs...)
	}
	return "COALESCE(" + strings.Join(ordered, ", ") + ")"
}

// bpItemFieldCoalesce construit un COALESCE ordonné par locale pour un champ textuel
// d'item Battle Pass : traduction dénormalisée puis fallbacks JSON du payload brut, la
// langue préférée d'abord. jsonKey = clé CommonData ("Title"/"Description"), col =
// colonne battlepass_item_translations.
//
// PIÈGE Waypoint : la chaîne anglaise vit dans `.value` (canonique en-US), PAS dans
// `.translations.en-US` (absent des payloads). value est donc une source ANGLAISE et
// doit précéder les traductions étrangères en préférence EN — sinon EN retombe sur
// `.translations.fr-FR` (bug « récompenses Battle Pass restent en FR quand l'UI passe EN »).
func bpItemFieldCoalesce(ctx context.Context, jsonKey, col string) string {
	frJSON := fmt.Sprintf("json_extract_string(d.raw_payload_json, '$.CommonData.%s.translations.fr-FR')", jsonKey)
	enJSON := fmt.Sprintf("json_extract_string(d.raw_payload_json, '$.CommonData.%s.translations.en-US')", jsonKey)
	val := fmt.Sprintf("json_extract_string(d.raw_payload_json, '$.CommonData.%s.value')", jsonKey)
	en := []string{"t_en." + col, enJSON, val}
	fr := []string{"t_fr." + col, frJSON}
	return bpLocaleOrderedCoalesce(bpPreferEN(ctx), en, fr)
}

func (r *SeasonPassRepo) LoadSeasonPassTracks(ctx context.Context, _, _ string) ([]domain.SeasonPassTrackSummary, error) {
	progressMap, activeTrackPath, err := r.loadTrackSnapshots(ctx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Récupère la définition la plus récente par track + traduction dans la locale
	// de requête (fallback sur l'autre langue).
	trackNameExpr := "COALESCE(t_fr.track_name, t_en.track_name)"
	if bpPreferEN(ctx) {
		trackNameExpr = "COALESCE(t_en.track_name, t_fr.track_name)"
	}
	query := fmt.Sprintf(`
		WITH latest AS (
			SELECT reward_track_path, content_hash, xp_per_rank, is_current, last_seen_at,
			       battlepass_image_path, background_image_path, raw_payload_json,
			       ROW_NUMBER() OVER (PARTITION BY reward_track_path ORDER BY last_seen_at DESC) AS rn
			FROM battlepass_track_definitions
		)
		SELECT d.reward_track_path, d.xp_per_rank,
		       %s AS track_name
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
		ORDER BY d.is_current DESC, d.last_seen_at DESC`, trackNameExpr)

	rows, err := r.pdb.Metadata.Query(ctx, query)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("season_pass_repo: tracks query: %w", err)
	}
	defer rows.Close()

	tracks := make([]domain.SeasonPassTrackSummary, 0)
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

	preferEN := bpPreferEN(ctx)
	for _, row := range trackRows {
		prog := progressMap[row.rewardTrackPath]
		prog.IsActive = row.rewardTrackPath == activeTrackPath
		summary := buildTrackSummary(row, prog, itemMap, preferEN)
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
	s := domain.SeasonPassTrackSummary{
		RewardTrackPath: path,
		Name:            name,
		Status:          status,
		IsActive:        state.IsActive,
		IsOwned:         isOwned,
		// PremiumOwned = signal brut d'achat premium (state.IsOwned), SANS la
		// dilution progression/actif de IsOwned ci-dessus.
		PremiumOwned:      state.IsOwned,
		HasReachedMaxRank: state.HasReachedMaxRank,
		CurrentRank:       state.Rank,
		PartialProgress:   state.Partial,
	}
	if !state.SnapshotAt.IsZero() {
		ts := state.SnapshotAt.UTC().Format(time.RFC3339)
		s.SnapshotAt = &ts
	}
	return s
}

func (r *SeasonPassRepo) loadItemMetadataMap(
	ctx context.Context,
	itemPaths []string,
) (map[string]seasonPassItemMeta, error) {
	if len(itemPaths) == 0 {
		return map[string]seasonPassItemMeta{}, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(itemPaths)), ",")
	// title/description : COALESCE ordonné par locale (traduction dénormalisée puis
	// fallbacks JSON du payload), langue préférée d'abord (cf. bpItemFieldCoalesce).
	// 1. battlepass_item_translations FR/EN (cache dénormalisé)
	// 2. raw_payload_json $.CommonData.Title.translations.{fr-FR,en-US} (extraction live)
	// 3. raw_payload_json $.CommonData.Title.value (fallback non localisé)
	// La fallback JSON couvre les items dont les translations n'ont jamais été
	// peuplées dans battlepass_item_translations (cf. items pré-déploiement
	// translations table).
	titleExpr := bpItemFieldCoalesce(ctx, "Title", "title")
	descExpr := bpItemFieldCoalesce(ctx, "Description", "description")
	query := fmt.Sprintf(`
		WITH latest AS (
			SELECT inventory_item_path, content_hash, quality, item_type, display_path,
			       raw_payload_json, last_seen_at,
			       ROW_NUMBER() OVER (PARTITION BY inventory_item_path ORDER BY last_seen_at DESC) AS rn
			FROM battlepass_item_definitions
			WHERE is_current = TRUE
		)
		SELECT d.inventory_item_path,
		       d.display_path,
		       %s AS title,
		       %s AS description,
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
		WHERE d.rn = 1 AND d.inventory_item_path IN (%s)`, titleExpr, descExpr, placeholders)

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
			Description: descriptionPtr(description),
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
	// Resolution locale-aware (cf. bpItemFieldCoalesce) : value = chaine canonique en-US
	// (source ANGLAISE), translations.{loc} = etranger. asset_index n'a pas de table de
	// traduction denormalisee -> uniquement les fallbacks JSON du raw_json.
	preferEN := bpPreferEN(ctx)
	assetTitle := bpLocaleOrderedCoalesce(preferEN,
		[]string{
			"json_extract_string(raw_json, '$.CommonData.Title.translations.en-US')",
			"json_extract_string(raw_json, '$.CommonData.Title.value')",
		},
		[]string{"json_extract_string(raw_json, '$.CommonData.Title.translations.fr-FR')"},
	)
	assetDesc := bpLocaleOrderedCoalesce(preferEN,
		[]string{
			"json_extract_string(raw_json, '$.CommonData.Description.translations.en-US')",
			"json_extract_string(raw_json, '$.CommonData.Description.value')",
		},
		[]string{"json_extract_string(raw_json, '$.CommonData.Description.translations.fr-FR')"},
	)
	query := fmt.Sprintf(`
		SELECT id,
		       json_extract_string(raw_json, '$.CommonData.DisplayPath.Media.MediaUrl.Path') AS display_path,
		       %s AS title,
		       %s AS description,
		       json_extract_string(raw_json, '$.CommonData.Quality')                        AS quality,
		       COALESCE(
		           json_extract_string(raw_json, '$.CommonData.Type'),
		           json_extract_string(raw_json, '$.CommonData.ItemType')
		       )                                                                            AS item_type
		FROM asset_index
		WHERE kind IN ('bp-item-def', 'track-def')
		  AND id IN (%s)
		  AND raw_json IS NOT NULL`, assetTitle, assetDesc, placeholders)

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
			Description: descriptionPtr(description),
			ImageURL:    localBPImageURL(coalesceNullString(displayPath), "tier"),
			Quality:     nullStringPtr(quality),
			ItemType:    nullStringPtr(itemType),
		}
	}
}
