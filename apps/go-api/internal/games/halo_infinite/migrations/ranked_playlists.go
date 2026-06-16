package migrations

// ranked_playlists.go — seed_ranked_playlists_catalog (named-func), déplacé depuis
// internal/migration/steps_metadata_seed_ranked_playlists.go (Phase 1.5 b12, voie B).
//
// Seed autoritatif des playlists CLASSÉES dans playlists_catalog (metadata.duckdb).
// La référence rankedplaylists est la source de vérité unique de is_ranked ; ce seed
// UPSERT en DO UPDATE force is_ranked=TRUE/experience='ranked' pour chaque playlist
// classée connue — corrige les lignes existantes ET crée les playlists jamais jouées.
// Le step est statique (dans Steps(), ApplySchema: applyRankedPlaylistSeeds) ; il trie
// après create_milestone_catalog_metadata / add_catalog_playlists dans canonicalOrder
// (la table playlists_catalog existe donc déjà).

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
	"levelup/go-api/internal/migration"
)

// rankedSeedTitleSlug du catalogue : metadata.duckdb est per-titre.
const rankedSeedTitleSlug = "halo_infinite"

// applyRankedPlaylistSeeds force, pour chaque playlist classée de référence, sa
// présence dans playlists_catalog avec is_ranked=TRUE + sa traduction FR dans
// asset_translations. Idempotent (UPSERT).
func applyRankedPlaylistSeeds(db *sql.DB) error {
	ctx := migration.BootCtx()
	now := time.Now().UTC()
	all := rankedplaylists.All()
	for _, p := range all {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO playlists_catalog
			  (title_slug, playlist_asset_id, current_version_id, name_canonical,
			   experience, is_ranked, is_active, first_seen_at, last_seen_at, last_fetched_at)
			VALUES (?, ?, '', ?, 'ranked', TRUE, ?, ?, ?, ?)
			ON CONFLICT (title_slug, playlist_asset_id) DO UPDATE SET
			  name_canonical = excluded.name_canonical,
			  experience     = 'ranked',
			  is_ranked      = TRUE,
			  is_active      = excluded.is_active,
			  last_fetched_at = excluded.last_fetched_at`,
			rankedSeedTitleSlug, p.AssetID, p.NameEN, p.Active, now, now, now,
		); err != nil {
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

// seedRankedPlaylistFR insère la traduction FR (fr + fr-FR) de la playlist dans
// asset_translations si NameFR est renseigné.
func seedRankedPlaylistFR(ctx context.Context, db *sql.DB, p rankedplaylists.Playlist) error {
	if p.NameFR == "" {
		return nil
	}
	for _, lang := range []string{"fr", "fr-FR"} {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO asset_translations (asset_id, asset_type, lang, name)
			VALUES (?, 'playlist', ?, ?)
			ON CONFLICT (asset_id, asset_type, lang) DO UPDATE SET name = excluded.name`,
			p.AssetID, lang, p.NameFR,
		); err != nil {
			return fmt.Errorf("seed FR playlist %s [%s]: %w", p.AssetID, lang, err)
		}
	}
	return nil
}
