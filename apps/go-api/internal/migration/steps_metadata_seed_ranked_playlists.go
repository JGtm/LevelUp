// Package migration — steps_metadata_seed_ranked_playlists.go : seed autoritatif
// des playlists CLASSÉES dans playlists_catalog (metadata.duckdb).
//
// Problème récurrent corrigé : les playlists ranked finissaient marquées
// is_ranked=FALSE / experience='social' car le catalogue était peuplé depuis
// match_registry (classif douteuse) et le seed CSR (career.go) utilisait
// ON CONFLICT DO NOTHING — incapable de corriger une ligne déjà fausse.
//
// Solution (cf. SpartanRecord / Grunt) : la référence rankedplaylists est la
// source de vérité unique de is_ranked. Ce seed UPSERT en DO UPDATE force
// is_ranked=TRUE, experience='ranked', is_active=<référence> pour chaque playlist
// classée connue — corrige les lignes existantes ET crée les playlists jamais
// jouées (Duo classé, etc.). Tous les autres chemins d'écriture consultent
// rankedplaylists.IsRanked() pour ne plus jamais rétrograder.
//
// Le fichier trie après steps_metadata_catalog.go (init alphabétique) : la table
// playlists_catalog existe donc déjà.
package migration

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
)

// titleSlug du catalogue : metadata.duckdb est per-titre (data/titles/halo_infinite/…).
const rankedSeedTitleSlug = "halo_infinite"

func init() {
	Register(Migration{
		Name:        "seed_ranked_playlists_catalog",
		TargetDB:    TargetMetadata,
		Description: "Seed autoritatif des playlists classées (is_ranked=TRUE) depuis la référence rankedplaylists — corrige le bug récurrent is_ranked=false",
		ApplySchema: applyRankedPlaylistSeeds,
	})
}

// applyRankedPlaylistSeeds force, pour chaque playlist classée de référence, sa
// présence dans playlists_catalog avec is_ranked=TRUE + sa traduction FR dans
// asset_translations. Idempotent (UPSERT).
func applyRankedPlaylistSeeds(db *sql.DB) error {
	ctx := bootCtx()
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
// asset_translations si NameFR est renseigné. Permet à enrichCSRPlaylistNames de
// résoudre le nom FR pour les playlists jamais jouées (absentes sinon).
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
