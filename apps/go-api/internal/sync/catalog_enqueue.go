package sync

// catalog_enqueue.go — Phase E du plan PLAN_PLAYLISTS_CATALOG.md.
//
// Hook post-ExtractRegistry : enqueue dans metadata.catalog_fetch_queue les
// asset IDs (playlist, pair, map, game_variant) qui ne sont pas encore dans
// le catalogue. Pattern Kinds : zéro fetch HTTP au sync, drain mensuel via CLI.
//
// Coût : 0 ou 1 INSERT OR IGNORE par asset par match (typiquement 0 dans 95 %
// des cas car le joueur rejoue les mêmes playlists).
//
// Erreurs DB sur la queue ne doivent JAMAIS bloquer l'ingestion (sync) — log
// warning et continue. C'est un cache best-effort.

import (
	"context"
	"database/sql"
	"log/slog"

	"levelup/go-api/internal/games"
)

// EnqueueCatalogAssets vérifie si les asset IDs du match sont absents du
// catalogue et les ajoute à catalog_fetch_queue. Appelle au plus 4 INSERT OR
// IGNORE par match (typiquement 0 si tout est déjà connu).
//
// metadataDB doit pointer sur metadata.duckdb (ou être ATTACHé en `meta`).
// Si nil ou indisponible, la fonction loggue un warning et retourne nil.
func EnqueueCatalogAssets(ctx context.Context, metadataDB *sql.DB, titleSlug string, row MatchRegistryRow) error {
	if metadataDB == nil {
		// Best-effort : si pas de connexion metadata disponible, skip silencieusement.
		// Cas connus : tests d'unit qui ne wirent pas la DB metadata.
		return nil
	}
	if titleSlug == "" {
		return nil
	}

	// Liste des couples (asset_type, asset_id, version_id) à vérifier.
	type asset struct {
		assetType string
		assetID   *string
		versionID *string
	}
	candidates := []asset{
		{games.AssetKindPlaylist, row.PlaylistID, row.PlaylistVersionID},
		{games.AssetKindPair, row.PairID, row.PairVersionID},
		{games.AssetKindMap, row.MapID, row.MapVersionID},
		{games.AssetKindGameVariant, row.GameVariantID, row.GameVariantVersionID},
	}

	for _, c := range candidates {
		if c.assetID == nil || *c.assetID == "" {
			continue
		}
		exists, err := catalogAssetExists(ctx, metadataDB, titleSlug, c.assetType, *c.assetID)
		if err != nil {
			slog.WarnContext(ctx, "catalog enqueue: existence check failed",
				"err", err, "asset_type", c.assetType, "asset_id", *c.assetID, "title_slug", titleSlug)
			continue
		}
		if exists {
			continue
		}
		var versionID string
		if c.versionID != nil {
			versionID = *c.versionID
		}
		if _, err := metadataDB.ExecContext(ctx,
			`INSERT OR IGNORE INTO catalog_fetch_queue (title_slug, asset_type, asset_id, version_id)
			 VALUES (?, ?, ?, ?)`,
			titleSlug, c.assetType, *c.assetID, versionID,
		); err != nil {
			slog.WarnContext(ctx, "catalog enqueue: INSERT failed",
				"err", err, "asset_type", c.assetType, "asset_id", *c.assetID, "title_slug", titleSlug)
			continue
		}
	}
	return nil
}

// catalogAssetExists retourne true si l'asset existe dans la table catalogue
// correspondante (playlists_catalog / maps_catalog / game_variants_catalog /
// map_mode_pair_definitions).
func catalogAssetExists(ctx context.Context, db *sql.DB, titleSlug, assetType, assetID string) (bool, error) {
	var query string
	switch assetType {
	case games.AssetKindPlaylist:
		query = `SELECT EXISTS(SELECT 1 FROM playlists_catalog WHERE title_slug = ? AND playlist_asset_id = ?)`
	case games.AssetKindPair:
		query = `SELECT EXISTS(SELECT 1 FROM map_mode_pair_definitions WHERE title_slug = ? AND pair_asset_id = ?)`
	case games.AssetKindMap:
		query = `SELECT EXISTS(SELECT 1 FROM maps_catalog WHERE title_slug = ? AND map_asset_id = ?)`
	case games.AssetKindGameVariant:
		query = `SELECT EXISTS(SELECT 1 FROM game_variants_catalog WHERE title_slug = ? AND game_variant_asset_id = ?)`
	default:
		return false, nil
	}
	var exists bool
	if err := db.QueryRowContext(ctx, query, titleSlug, assetID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
