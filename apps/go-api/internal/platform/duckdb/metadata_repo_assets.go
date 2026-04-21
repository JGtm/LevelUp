// Package duckdb — metadata_repo_assets.go : gestion asset_translations multilingues.
//
// Sprint 54 : peuplement asset_translations depuis Discovery UGC API.
package duckdb

import (
	"context"
	"fmt"
)

// GetDistinctAssetIDs retourne les asset_ids distincts depuis match_registry.
// Nécessite une connexion à shared_matches_v2.duckdb (paramètre sharedDB).
//
// Colonnes par asset_type :
//   - "map" → map_id
//   - "playlist" → playlist_id
//   - "pair" → pair_id
//   - "game_variant" → game_variant_id
func (r *MetadataRepo) GetDistinctAssetIDs(
	ctx context.Context,
	assetType string,
	sharedDB *DB,
) ([]string, error) {
	columnMap := map[string]string{
		"map":          "map_id",
		"playlist":     "playlist_id",
		"pair":         "pair_id",
		"game_variant": "game_variant_id",
	}

	column, ok := columnMap[assetType]
	if !ok {
		return nil, fmt.Errorf("asset_type invalide : %s", assetType)
	}

	query := fmt.Sprintf(
		"SELECT DISTINCT %s FROM match_registry WHERE %s IS NOT NULL",
		column, column,
	)

	rows, err := sharedDB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetDistinctAssetIDs(%s): %w", assetType, err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan asset_id: %w", err)
		}
		if id != "" {
			ids = append(ids, id)
		}
	}

	return ids, rows.Err()
}

// GetExistingTranslations retourne un set des asset_ids déjà présents et frais.
// Fraîcheur = fetched_at >= now() - freshnessDays jours.
// Retourne map[asset_id]bool pour lookup O(1).
func (r *MetadataRepo) GetExistingTranslations(
	ctx context.Context,
	assetType string,
	lang string,
	freshnessDays int,
) (map[string]bool, error) {
	query := `
		SELECT asset_id
		FROM asset_translations
		WHERE asset_type = ?
		  AND lang = ?
		  AND fetched_at >= now() - INTERVAL ? DAY
	`

	rows, err := r.meta.Query(ctx, query, assetType, lang, freshnessDays)
	if err != nil {
		return nil, fmt.Errorf("GetExistingTranslations(%s, %s): %w", assetType, lang, err)
	}
	defer rows.Close()

	existing := make(map[string]bool)
	for rows.Next() {
		var assetID string
		if err := rows.Scan(&assetID); err != nil {
			return nil, fmt.Errorf("scan asset_id: %w", err)
		}
		existing[assetID] = true
	}

	return existing, rows.Err()
}

// UpsertAssetTranslation insère ou met à jour une traduction d'asset.
func (r *MetadataRepo) UpsertAssetTranslation(
	ctx context.Context,
	assetID string,
	assetType string,
	lang string,
	name string,
	description string,
) error {
	query := `
		INSERT INTO asset_translations (asset_id, asset_type, lang, name, description, fetched_at)
		VALUES (?, ?, ?, ?, ?, now())
		ON CONFLICT (asset_id, asset_type, lang) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			fetched_at = now()
	`

	_, err := r.meta.Exec(ctx, query, assetID, assetType, lang, name, description)
	if err != nil {
		return fmt.Errorf("UpsertAssetTranslation(%s, %s, %s): %w", assetType, assetID, lang, err)
	}
	return nil
}

// GetAssetTranslationCount retourne le nombre de traductions par langue pour un asset_type.
// Utile pour vérifier le peuplement après populate-assets.
func (r *MetadataRepo) GetAssetTranslationCount(
	ctx context.Context,
	assetType string,
) (map[string]int, error) {
	query := `
		SELECT lang, COUNT(*) as count
		FROM asset_translations
		WHERE asset_type = ?
		GROUP BY lang
		ORDER BY lang
	`

	rows, err := r.meta.Query(ctx, query, assetType)
	if err != nil {
		return nil, fmt.Errorf("GetAssetTranslationCount(%s): %w", assetType, err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var lang string
		var count int
		if err := rows.Scan(&lang, &count); err != nil {
			return nil, fmt.Errorf("scan count: %w", err)
		}
		counts[lang] = count
	}

	return counts, rows.Err()
}
