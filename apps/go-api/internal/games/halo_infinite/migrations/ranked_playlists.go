package migrations

// ranked_playlists.go — seed_ranked_playlists_catalog (named-func), déplacé depuis
// internal/migration/steps_metadata_seed_ranked_playlists.go (Phase 1.5 b12, voie B).
//
// Seed autoritatif des playlists CLASSÉES dans playlists_catalog (metadata.duckdb).
// La référence rankedplaylists est la source de vérité unique de is_ranked ; ce seed
// force is_ranked=TRUE/experience='ranked' pour chaque playlist classée connue — corrige
// les lignes existantes ET crée les playlists jamais jouées. Le step est statique (dans
// Steps(), ApplySchema: applyRankedPlaylistSeeds) ; il trie après
// create_milestone_catalog_metadata / add_catalog_playlists dans canonicalOrder (la table
// playlists_catalog existe donc déjà).
//
// Écritures en SELECT-then-write (duckdbkeys.Upsert), jamais ON CONFLICT DO
// UPDATE : playlists_catalog est une table protégée (un UPSERT sur elle FATAL-invalide le
// handle metadata), y compris dans une migration — garde-rail
// internal/sync/no_art_upsert_migrations_test.go.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
	"levelup/go-api/internal/migration"
	"levelup/go-api/pkg/duckdbkeys"
)

// rankedSeedTitleSlug du catalogue : metadata.duckdb est per-titre.
const rankedSeedTitleSlug = "halo_infinite"

// applyRankedPlaylistSeeds force, pour chaque playlist classée de référence, sa
// présence dans playlists_catalog avec is_ranked=TRUE + sa traduction FR dans
// asset_translations. Idempotent.
func applyRankedPlaylistSeeds(db *sql.DB) error {
	ctx := migration.BootCtx()
	now := time.Now().UTC()
	all := rankedplaylists.All()
	for _, p := range all {
		if err := (duckdbkeys.Upsert{
			ExistsSQL:  `SELECT 1 FROM playlists_catalog WHERE title_slug = ? AND playlist_asset_id = ?`,
			ExistsArgs: []any{rankedSeedTitleSlug, p.AssetID},
			UpdateSQL: `UPDATE playlists_catalog SET
			   name_canonical = ?, experience = 'ranked', is_ranked = TRUE,
			   is_active = ?, last_fetched_at = ?
			 WHERE title_slug = ? AND playlist_asset_id = ?`,
			UpdateArgs: []any{p.NameEN(), p.Active, now, rankedSeedTitleSlug, p.AssetID},
			InsertSQL: `INSERT INTO playlists_catalog
			  (title_slug, playlist_asset_id, current_version_id, name_canonical,
			   experience, is_ranked, is_active, first_seen_at, last_seen_at, last_fetched_at)
			 VALUES (?, ?, '', ?, 'ranked', TRUE, ?, ?, ?, ?)`,
			InsertArgs: []any{rankedSeedTitleSlug, p.AssetID, p.NameEN(), p.Active, now, now, now},
		}).Exec(ctx, db); err != nil {
			return fmt.Errorf("seed ranked playlist %s: %w", p.AssetID, err)
		}
		if err := seedRankedPlaylistFR(ctx, db, p); err != nil {
			return err
		}
	}
	slog.InfoContext(ctx, "migration: playlists classées seedées (is_ranked autoritatif)",
		"total", len(all), "actives", len(rankedplaylists.Active()), "title_slug", rankedSeedTitleSlug)
	return nil
}

// seedRankedPlaylistFR écrit la traduction FR (fr + fr-FR) de la playlist dans
// asset_translations si NameFR est renseigné.
func seedRankedPlaylistFR(ctx context.Context, db *sql.DB, p rankedplaylists.Playlist) error {
	nameFR := p.NameFR()
	if nameFR == "" {
		return nil
	}
	for _, lang := range []string{"fr", "fr-FR"} {
		if err := (duckdbkeys.Upsert{
			ExistsSQL:  `SELECT 1 FROM asset_translations WHERE asset_id = ? AND asset_type = 'playlist' AND lang = ?`,
			ExistsArgs: []any{p.AssetID, lang},
			UpdateSQL:  `UPDATE asset_translations SET name = ? WHERE asset_id = ? AND asset_type = 'playlist' AND lang = ?`,
			UpdateArgs: []any{nameFR, p.AssetID, lang},
			InsertSQL:  `INSERT INTO asset_translations (asset_id, asset_type, lang, name) VALUES (?, 'playlist', ?, ?)`,
			InsertArgs: []any{p.AssetID, lang, nameFR},
		}).Exec(ctx, db); err != nil {
			return fmt.Errorf("seed FR playlist %s [%s]: %w", p.AssetID, lang, err)
		}
	}
	return nil
}
