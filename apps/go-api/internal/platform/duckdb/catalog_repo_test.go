//go:build integration

// catalog_repo_test.go — tests Phase F plan catalogue (lecture).
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func setupCatalogRepoDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Seed.
	db.Exec(`INSERT INTO playlists_catalog (title_slug, playlist_asset_id, name_canonical, experience, is_ranked, is_active)
	         VALUES ('halo_infinite', 'pl-1', 'Quick Play', 'social', FALSE, TRUE)`)
	db.Exec(`INSERT INTO playlists_catalog (title_slug, playlist_asset_id, name_canonical, experience, is_ranked, is_active)
	         VALUES ('halo_infinite', 'pl-2', 'Ranked Arena', 'ranked', TRUE, TRUE)`)
	db.Exec(`INSERT INTO playlists_catalog (title_slug, playlist_asset_id, name_canonical, is_active)
	         VALUES ('halo_infinite', 'pl-old', 'Old Playlist', FALSE)`)
	db.Exec(`INSERT INTO playlists_catalog (title_slug, playlist_asset_id, name_canonical, is_active)
	         VALUES ('synthetic_title_b', 'pl-1', 'Synth Quick Play', TRUE)`)

	db.Exec(`INSERT INTO maps_catalog (title_slug, map_asset_id, name_canonical, image_url)
	         VALUES ('halo_infinite', 'm-1', 'Bazaar', '/api/v1/assets/maps/bazaar.png')`)
	db.Exec(`INSERT INTO maps_catalog (title_slug, map_asset_id, name_canonical)
	         VALUES ('halo_infinite', 'm-2', 'Live Fire')`)

	db.Exec(`INSERT INTO map_mode_pair_definitions (title_slug, pair_asset_id, name_canonical, map_asset_id, game_variant_asset_id, mode_category)
	         VALUES ('halo_infinite', 'pa-1', 'Arena:Slayer on Bazaar', 'm-1', 'gv-1', 'Assassin')`)
	db.Exec(`INSERT INTO playlist_pair_links (title_slug, playlist_asset_id, pair_asset_id, weight)
	         VALUES ('halo_infinite', 'pl-1', 'pa-1', 1.0)`)
	return db
}

func TestCatalogRepo_PlaylistsByTitle_FullCatalog(t *testing.T) {
	ctx := context.Background()
	db := setupCatalogRepoDB(t)
	r := NewCatalogRepo(db, nil)

	pls, err := r.PlaylistsByTitle(ctx, "halo_infinite", "", false)
	if err != nil {
		t.Fatalf("PlaylistsByTitle: %v", err)
	}
	// Doit retourner les 2 actives, exclure pl-old (is_active=false).
	if len(pls) != 2 {
		t.Errorf("len = %d, want 2 (excl is_active=false)", len(pls))
	}
	for _, p := range pls {
		if p.PlaylistAssetID == "pl-old" {
			t.Error("pl-old (inactive) ne devrait pas être retourné")
		}
		if p.TitleSlug != "halo_infinite" {
			t.Errorf("title_slug = %q (cross-leak)", p.TitleSlug)
		}
	}
}

func TestCatalogRepo_PlaylistsByTitle_TitleIsolation(t *testing.T) {
	ctx := context.Background()
	db := setupCatalogRepoDB(t)
	r := NewCatalogRepo(db, nil)

	pls, err := r.PlaylistsByTitle(ctx, "synthetic_title_b", "", false)
	if err != nil {
		t.Fatalf("PlaylistsByTitle: %v", err)
	}
	if len(pls) != 1 {
		t.Errorf("len = %d, want 1 (synth only)", len(pls))
	}
	if pls[0].Name != "Synth Quick Play" {
		t.Errorf("name = %q (isolation broken)", pls[0].Name)
	}
}

func TestCatalogRepo_PairsByPlaylist(t *testing.T) {
	ctx := context.Background()
	db := setupCatalogRepoDB(t)
	r := NewCatalogRepo(db, nil)

	pairs, err := r.PairsByPlaylist(ctx, "halo_infinite", "pl-1")
	if err != nil {
		t.Fatalf("PairsByPlaylist: %v", err)
	}
	if len(pairs) != 1 {
		t.Errorf("len = %d, want 1", len(pairs))
	}
	if pairs[0].PairAssetID != "pa-1" {
		t.Errorf("pair = %q", pairs[0].PairAssetID)
	}
	if pairs[0].Weight != 1.0 {
		t.Errorf("weight = %f, want 1.0", pairs[0].Weight)
	}
	if pairs[0].ModeCategory != "Assassin" {
		t.Errorf("mode_category = %q, want Assassin", pairs[0].ModeCategory)
	}
}

func TestCatalogRepo_MapsByTitle(t *testing.T) {
	ctx := context.Background()
	db := setupCatalogRepoDB(t)
	r := NewCatalogRepo(db, nil)

	maps, err := r.MapsByTitle(ctx, "halo_infinite", "", false)
	if err != nil {
		t.Fatalf("MapsByTitle: %v", err)
	}
	if len(maps) != 2 {
		t.Errorf("len = %d, want 2", len(maps))
	}
}

func TestCatalogRepo_CountCatalogEntries(t *testing.T) {
	ctx := context.Background()
	db := setupCatalogRepoDB(t)
	r := NewCatalogRepo(db, nil)

	n, err := r.CountCatalogEntries(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("CountCatalogEntries: %v", err)
	}
	if n != 2 {
		t.Errorf("count = %d, want 2 (active only)", n)
	}

	n, _ = r.CountCatalogEntries(ctx, "synthetic_title_b")
	if n != 1 {
		t.Errorf("synth count = %d, want 1", n)
	}

	n, _ = r.CountCatalogEntries(ctx, "nonexistent")
	if n != 0 {
		t.Errorf("nonexistent count = %d, want 0", n)
	}
}
