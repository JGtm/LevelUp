//go:build integration

// catalog_enqueue_test.go — tests Phase E plan catalogue.
package sync

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func setupMetadataDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForDB(Metadata): %v", err)
	}
	return db
}

func TestEnqueueCatalogAssets_AllUnknown_Enqueues4(t *testing.T) {
	ctx := context.Background()
	db := setupMetadataDB(t)

	playlistID := "pl-1"
	pairID := "pa-1"
	mapID := "ma-1"
	gvID := "gv-1"
	row := MatchRegistryRow{
		PlaylistID:    &playlistID,
		PairID:        &pairID,
		MapID:         &mapID,
		GameVariantID: &gvID,
	}

	if err := EnqueueCatalogAssets(ctx, db, "halo_infinite", row); err != nil {
		t.Fatalf("EnqueueCatalogAssets: %v", err)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM catalog_fetch_queue`).Scan(&n); err != nil {
		t.Fatalf("count queue: %v", err)
	}
	if n != 4 {
		t.Errorf("queue count = %d, want 4", n)
	}
}

func TestEnqueueCatalogAssets_PlaylistKnown_Skips(t *testing.T) {
	ctx := context.Background()
	db := setupMetadataDB(t)

	// Pré-peupler le catalogue avec une playlist déjà connue.
	if _, err := db.Exec(
		`INSERT INTO playlists_catalog (title_slug, playlist_asset_id, name_canonical) VALUES (?, ?, ?)`,
		"halo_infinite", "pl-known", "Quick Play",
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	playlistID := "pl-known"
	pairID := "pa-new"
	row := MatchRegistryRow{
		PlaylistID: &playlistID,
		PairID:     &pairID,
	}

	if err := EnqueueCatalogAssets(ctx, db, "halo_infinite", row); err != nil {
		t.Fatalf("EnqueueCatalogAssets: %v", err)
	}

	// Seul pa-new doit être enqueué (pl-known est déjà dans le catalogue).
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM catalog_fetch_queue`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("queue count = %d, want 1 (playlist known, pair new)", n)
	}

	var assetType string
	if err := db.QueryRow(`SELECT asset_type FROM catalog_fetch_queue`).Scan(&assetType); err != nil {
		t.Fatalf("read asset_type: %v", err)
	}
	if assetType != "pair" {
		t.Errorf("asset_type enqueued = %q, want pair", assetType)
	}
}

func TestEnqueueCatalogAssets_NilDB_NoError(t *testing.T) {
	ctx := context.Background()
	playlistID := "pl-1"
	row := MatchRegistryRow{PlaylistID: &playlistID}
	// Sans DB metadata disponible, doit retourner nil sans paniquer.
	if err := EnqueueCatalogAssets(ctx, nil, "halo_infinite", row); err != nil {
		t.Errorf("EnqueueCatalogAssets(nil DB): %v, want nil", err)
	}
}

func TestEnqueueCatalogAssets_EmptyTitleSlug_Skips(t *testing.T) {
	ctx := context.Background()
	db := setupMetadataDB(t)
	playlistID := "pl-1"
	row := MatchRegistryRow{PlaylistID: &playlistID}
	if err := EnqueueCatalogAssets(ctx, db, "", row); err != nil {
		t.Fatalf("EnqueueCatalogAssets: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM catalog_fetch_queue`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("queue count = %d, want 0 (empty title_slug)", n)
	}
}

func TestEnqueueCatalogAssets_NilAssetIDs_Skipped(t *testing.T) {
	ctx := context.Background()
	db := setupMetadataDB(t)

	// Tous les IDs nil → rien à enqueuer.
	row := MatchRegistryRow{}
	if err := EnqueueCatalogAssets(ctx, db, "halo_infinite", row); err != nil {
		t.Fatalf("EnqueueCatalogAssets: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM catalog_fetch_queue`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("queue count = %d, want 0 (no asset IDs)", n)
	}
}

func TestEnqueueCatalogAssets_DuplicateMatch_NoDuplicate(t *testing.T) {
	ctx := context.Background()
	db := setupMetadataDB(t)
	playlistID := "pl-dup"
	row := MatchRegistryRow{PlaylistID: &playlistID}

	for i := 0; i < 3; i++ {
		if err := EnqueueCatalogAssets(ctx, db, "halo_infinite", row); err != nil {
			t.Fatalf("EnqueueCatalogAssets pass %d: %v", i, err)
		}
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM catalog_fetch_queue`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("queue count = %d après 3 passes, want 1 (INSERT OR IGNORE)", n)
	}
}

func TestEnqueueCatalogAssets_TitleSlugIsolation(t *testing.T) {
	ctx := context.Background()
	db := setupMetadataDB(t)
	playlistID := "pl-1"
	row := MatchRegistryRow{PlaylistID: &playlistID}

	if err := EnqueueCatalogAssets(ctx, db, "halo_infinite", row); err != nil {
		t.Fatalf("enqueue halo: %v", err)
	}
	if err := EnqueueCatalogAssets(ctx, db, "synthetic_title_b", row); err != nil {
		t.Fatalf("enqueue synth: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM catalog_fetch_queue`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Errorf("queue count = %d, want 2 (isolation par title_slug)", n)
	}
}
