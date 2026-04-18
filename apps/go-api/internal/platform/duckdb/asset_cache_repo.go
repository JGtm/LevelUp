// Package duckdb — asset_cache_repo.go : cache-aside des assets Waypoint génériques.
//
// Sprint 54 D2 : table waypoint_assets_raw dans metadata.duckdb.
// Pattern oneshot : fetch Waypoint → stockage local → service depuis cache.
// Multi-titres via title_id. Singleflight dans le handler.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// AssetEntry représente une entrée brute d'asset Waypoint.
type AssetEntry struct {
	TitleID     string
	AssetID     string
	AssetType   string
	VersionID   string
	Name        string
	Description string
	RawJSON     string
	FetchedAt   time.Time
	ContentHash string
}

// AssetDiffReport résume les changements détectés entre Waypoint et la DB locale.
type AssetDiffReport struct {
	New       []AssetEntry
	Modified  []AssetEntry
	Unchanged int
}

// GetAssetByID retourne un asset depuis waypoint_assets_raw, nil si absent.
func (r *MetadataRepo) GetAssetByID(ctx context.Context, titleID, assetID string) (*AssetEntry, error) {
	row := r.meta.QueryRow(ctx, `
		SELECT title_id, asset_id, asset_type, version_id, name, description, raw_json, fetched_at, content_hash
		FROM waypoint_assets_raw
		WHERE title_id = ? AND asset_id = ?
		ORDER BY fetched_at DESC
		LIMIT 1`, titleID, assetID)

	var e AssetEntry
	err := row.Scan(&e.TitleID, &e.AssetID, &e.AssetType, &e.VersionID,
		&e.Name, &e.Description, &e.RawJSON, &e.FetchedAt, &e.ContentHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetAssetByID: %w", err)
	}
	return &e, nil
}

// UpsertAsset insère ou met à jour un asset dans waypoint_assets_raw.
func (r *MetadataRepo) UpsertAsset(ctx context.Context, e AssetEntry) error {
	_, err := r.meta.Exec(ctx, `
		INSERT INTO waypoint_assets_raw
			(title_id, asset_id, asset_type, version_id, name, description, raw_json, fetched_at, content_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (title_id, asset_id, version_id) DO UPDATE SET
			asset_type   = excluded.asset_type,
			name         = excluded.name,
			description  = excluded.description,
			raw_json     = excluded.raw_json,
			fetched_at   = excluded.fetched_at,
			content_hash = excluded.content_hash`,
		e.TitleID, e.AssetID, e.AssetType, e.VersionID,
		e.Name, e.Description, e.RawJSON, e.FetchedAt, e.ContentHash)
	if err != nil {
		return fmt.Errorf("UpsertAsset %s: %w", e.AssetID, err)
	}
	return nil
}

// ListAssets retourne tous les assets d'un titre (dernière version par asset_id).
func (r *MetadataRepo) ListAssets(ctx context.Context, titleID string) ([]AssetEntry, error) {
	rows, err := r.meta.Query(ctx, `
		SELECT DISTINCT ON (asset_id)
			title_id, asset_id, asset_type, version_id, name, description, raw_json, fetched_at, content_hash
		FROM waypoint_assets_raw
		WHERE title_id = ?
		ORDER BY asset_id, fetched_at DESC`, titleID)
	if err != nil {
		return nil, fmt.Errorf("ListAssets: %w", err)
	}
	defer rows.Close()

	var result []AssetEntry
	for rows.Next() {
		var e AssetEntry
		if err := rows.Scan(&e.TitleID, &e.AssetID, &e.AssetType, &e.VersionID,
			&e.Name, &e.Description, &e.RawJSON, &e.FetchedAt, &e.ContentHash); err != nil {
			return nil, fmt.Errorf("ListAssets scan: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// ComputeAssetDiff compare une liste d'assets Waypoint avec la DB locale.
// Retourne un rapport des nouveaux, modifiés et inchangés — sans écriture.
func ComputeAssetDiff(local []AssetEntry, incoming []AssetEntry) AssetDiffReport {
	localIndex := make(map[string]AssetEntry, len(local))
	for _, e := range local {
		localIndex[e.AssetID] = e
	}

	var report AssetDiffReport
	for _, inc := range incoming {
		existing, found := localIndex[inc.AssetID]
		if !found {
			report.New = append(report.New, inc)
		} else if existing.ContentHash != inc.ContentHash {
			report.Modified = append(report.Modified, inc)
		} else {
			report.Unchanged++
		}
	}
	return report
}
