// Package duckdb — medal_cache_repo.go : cache-aside des images de médailles.
//
// Sprint 54 D1 : table medal_image_cache dans metadata.duckdb.
// Pattern : oneshot fetch Waypoint → stockage local → service direct depuis cache.
// Multi-titres via title_id. Singleflight dans le handler (pas ici).
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"levelup/go-api/internal/metadata"
)

// MedalImageEntry représente une entrée du cache local d'images de médailles.
type MedalImageEntry struct {
	TitleID     string
	MedalID     int64
	ImageURL    string // URL source Waypoint
	LocalPath   string // chemin local relatif à LEVELUP_REPO_ROOT
	FetchedAt   time.Time
	ContentHash string
}

// EnsureMedalImageCacheTable crée la table medal_image_cache si absente (idempotent).
func (r *MetadataRepo) EnsureMedalImageCacheTable(ctx context.Context) error {
	_, err := r.meta.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS medal_image_cache (
			title_id     VARCHAR NOT NULL,
			medal_id     BIGINT  NOT NULL,
			image_url    VARCHAR NOT NULL DEFAULT '',
			local_path   VARCHAR NOT NULL DEFAULT '',
			fetched_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
			content_hash VARCHAR NOT NULL DEFAULT '',
			PRIMARY KEY (title_id, medal_id)
		)`)
	if err != nil {
		return fmt.Errorf("EnsureMedalImageCacheTable: %w", err)
	}
	return nil
}

// GetMedalImageCache retourne l'entrée cache pour (titleID, medalID), nil si absente.
func (r *MetadataRepo) GetMedalImageCache(ctx context.Context, titleID string, medalID int64) (*MedalImageEntry, error) {
	row := r.meta.QueryRow(ctx, `
		SELECT title_id, medal_id, image_url, local_path, fetched_at, content_hash
		FROM medal_image_cache
		WHERE title_id = ? AND medal_id = ?`, titleID, medalID)

	var e MedalImageEntry
	err := row.Scan(&e.TitleID, &e.MedalID, &e.ImageURL, &e.LocalPath, &e.FetchedAt, &e.ContentHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetMedalImageCache: %w", err)
	}
	return &e, nil
}

// UpsertMedalImageCache insère ou met à jour une entrée dans medal_image_cache.
func (r *MetadataRepo) UpsertMedalImageCache(ctx context.Context, e MedalImageEntry) error {
	_, err := r.meta.Exec(ctx, `
		INSERT INTO medal_image_cache (title_id, medal_id, image_url, local_path, fetched_at, content_hash)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (title_id, medal_id) DO UPDATE SET
			image_url    = excluded.image_url,
			local_path   = excluded.local_path,
			fetched_at   = excluded.fetched_at,
			content_hash = excluded.content_hash`,
		e.TitleID, e.MedalID, e.ImageURL, e.LocalPath, e.FetchedAt, e.ContentHash)
	if err != nil {
		return fmt.Errorf("UpsertMedalImageCache: %w", err)
	}
	return nil
}

// UpsertMedalsRaw insère ou met à jour les entrées dans waypoint_medals_raw.
func (r *MetadataRepo) UpsertMedalsRaw(ctx context.Context, entries []metadata.MedalEntry, contentHash string) error {
	for _, e := range entries {
		_, err := r.meta.Exec(ctx, `
			INSERT INTO waypoint_medals_raw
				(title_id, medal_id, name_id, description_id, sprite_index, difficulty, medal_type, personal_score, raw_json, fetched_at, content_hash)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, now(), ?)
			ON CONFLICT (title_id, medal_id) DO UPDATE SET
				name_id        = excluded.name_id,
				description_id = excluded.description_id,
				sprite_index   = excluded.sprite_index,
				difficulty     = excluded.difficulty,
				medal_type     = excluded.medal_type,
				personal_score = excluded.personal_score,
				raw_json       = excluded.raw_json,
				fetched_at     = excluded.fetched_at,
				content_hash   = excluded.content_hash`,
			e.TitleID, e.MedalID, e.Label, e.Description, e.SpriteIdx,
			e.Rarity, e.Category, 0, e.RawJSON, contentHash)
		if err != nil {
			return fmt.Errorf("UpsertMedalsRaw medal_id=%d: %w", e.MedalID, err)
		}
	}
	return nil
}

// CountMedalsRaw retourne le nombre de médailles en DB pour un title_id.
func (r *MetadataRepo) CountMedalsRaw(ctx context.Context, titleID string) (int, error) {
	row := r.meta.QueryRow(ctx, `SELECT COUNT(*) FROM waypoint_medals_raw WHERE title_id = ?`, titleID)
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, fmt.Errorf("CountMedalsRaw: %w", err)
	}
	return n, nil
}
