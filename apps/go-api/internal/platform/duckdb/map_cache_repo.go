// Package duckdb — map_cache_repo.go : cache-aside des images de maps.
//
// Pattern : oneshot fetch Waypoint → stockage local → service direct depuis cache.
// Multi-titres via title_id. Singleflight dans le handler (pas ici).
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// MapImageEntry représente une entrée du cache local d'images de maps.
type MapImageEntry struct {
	TitleID     string
	MapID       string // asset_id de la map (UUID)
	ImageURL    string // URL source Waypoint ou CDN
	LocalPath   string // chemin local relatif (ex: /static/maps/Aquarius.png)
	FetchedAt   time.Time
	ContentHash string
}

// EnsureMapImageCacheTable crée la table map_images_registry si absente (idempotent).
func (r *MetadataRepo) EnsureMapImageCacheTable(ctx context.Context) error {
	_, err := r.meta.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS map_images_registry (
			title_id     VARCHAR NOT NULL,
			map_id       VARCHAR NOT NULL,
			image_url    VARCHAR NOT NULL DEFAULT '',
			local_path   VARCHAR NOT NULL DEFAULT '',
			fetched_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
			content_hash VARCHAR NOT NULL DEFAULT '',
			PRIMARY KEY (title_id, map_id)
		)`)
	if err != nil {
		return fmt.Errorf("EnsureMapImageCacheTable: %w", err)
	}
	return nil
}

// GetMapImageCache retourne l'entrée cache pour (titleID, mapID), nil si absente.
func (r *MetadataRepo) GetMapImageCache(ctx context.Context, titleID string, mapID string) (*MapImageEntry, error) {
	row := r.meta.QueryRow(ctx, `
		SELECT title_id, map_id, image_url, local_path, fetched_at, content_hash
		FROM map_images_registry
		WHERE title_id = ? AND map_id = ?`, titleID, mapID)

	var e MapImageEntry
	err := row.Scan(&e.TitleID, &e.MapID, &e.ImageURL, &e.LocalPath, &e.FetchedAt, &e.ContentHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetMapImageCache: %w", err)
	}
	return &e, nil
}

// UpsertMapImageCache insère ou met à jour une entrée dans map_images_registry.
func (r *MetadataRepo) UpsertMapImageCache(ctx context.Context, e MapImageEntry) error {
	_, err := r.meta.Exec(ctx, `
		INSERT INTO map_images_registry (title_id, map_id, image_url, local_path, fetched_at, content_hash)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (title_id, map_id) DO UPDATE SET
			image_url    = excluded.image_url,
			local_path   = excluded.local_path,
			fetched_at   = excluded.fetched_at,
			content_hash = excluded.content_hash`,
		e.TitleID, e.MapID, e.ImageURL, e.LocalPath, e.FetchedAt, e.ContentHash)
	if err != nil {
		return fmt.Errorf("UpsertMapImageCache: %w", err)
	}
	return nil
}
