// Package ops — catalog_queue.go : seed de la file catalog_fetch_queue depuis
// match_registry (zéro réseau). C'est la phase 1 du drain DiscoveryUGC : on
// recense tous les asset IDs distincts vus en jeu pour qu'un drain (CLI ou
// action admin) les résolve ensuite via l'API.
//
// Extrait de cmd/populate-playlists-catalog (seedQueueFromMatchRegistry) pour
// être appelable in-process — le CLI reste le wrapper du même code.
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// CatalogQueueSeedResult compte les insertions effectives (hors doublons
// ignorés) par type d'asset.
type CatalogQueueSeedResult struct {
	Playlists, Pairs, Maps, GameVariants int
}

// Total agrège les quatre types.
func (r CatalogQueueSeedResult) Total() int {
	return r.Playlists + r.Pairs + r.Maps + r.GameVariants
}

// SeedCatalogQueueFromRegistry insère dans catalog_fetch_queue tous les asset
// IDs distincts de shared.match_registry (INSERT OR IGNORE → idempotent). Lit
// shared, écrit metadata, en Go (pas d'ATTACH). Fallback sans version_id si la
// colonne *_version_id n'existe pas encore (migration non appliquée).
func SeedCatalogQueueFromRegistry(ctx context.Context, metadataDB, sharedDB *sql.DB, titleSlug string) (CatalogQueueSeedResult, error) {
	var counters CatalogQueueSeedResult
	specs := []struct {
		assetType  string
		col        string
		versionCol string
		counter    *int
	}{
		{"playlist", "playlist_id", "playlist_version_id", &counters.Playlists},
		{"pair", "pair_id", "pair_version_id", &counters.Pairs},
		{"map", "map_id", "map_version_id", &counters.Maps},
		{"game_variant", "game_variant_id", "game_variant_version_id", &counters.GameVariants},
	}
	for _, s := range specs {
		inserted, err := seedQueueAsset(ctx, metadataDB, sharedDB, titleSlug, s.assetType, s.col, s.versionCol)
		if err != nil {
			return counters, err
		}
		*s.counter = inserted
	}
	return counters, nil
}

// seedQueueAsset traite un type d'asset (extrait pour rester < 80 lignes).
func seedQueueAsset(ctx context.Context, metadataDB, sharedDB *sql.DB, titleSlug, assetType, col, versionCol string) (int, error) {
	query := fmt.Sprintf(
		`SELECT DISTINCT %s, COALESCE(%s, '') FROM match_registry WHERE %s IS NOT NULL AND %s != ''`,
		col, versionCol, col, col,
	)
	rows, err := sharedDB.QueryContext(ctx, query)
	if err != nil {
		// Colonne version_id absente → fallback id-only.
		slog.WarnContext(ctx, "seed_queue: version_id absent, fallback id-only", "asset_type", assetType, "err", err)
		query = fmt.Sprintf(`SELECT DISTINCT %s, '' FROM match_registry WHERE %s IS NOT NULL AND %s != ''`, col, col, col)
		rows, err = sharedDB.QueryContext(ctx, query)
		if err != nil {
			return 0, fmt.Errorf("select %s (fallback): %w", assetType, err)
		}
	}
	defer rows.Close()

	var inserted int
	for rows.Next() {
		var id, ver string
		if err := rows.Scan(&id, &ver); err != nil {
			return inserted, err
		}
		res, err := metadataDB.ExecContext(ctx,
			`INSERT OR IGNORE INTO catalog_fetch_queue (title_slug, asset_type, asset_id, version_id) VALUES (?, ?, ?, ?)`,
			titleSlug, assetType, id, ver)
		if err != nil {
			slog.WarnContext(ctx, "seed_queue: insert échoué", "err", err, "asset_type", assetType, "asset_id", id)
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			inserted++
		}
	}
	return inserted, rows.Err()
}
