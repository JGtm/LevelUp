// Package ops — catalog_queue_cgo_test.go : seed de la file catalog_fetch_queue
// sur DuckDB :memory: (driver CGO requis). Vérifie le dédoublonnage, le
// dénombrement par type et l'idempotence (INSERT OR IGNORE).
package ops

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openQueueTestDBs(t *testing.T) (shared, meta *sql.DB) {
	t.Helper()
	shared, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	t.Cleanup(func() { shared.Close() })
	if _, err := shared.Exec(`CREATE TABLE match_registry (
		match_id VARCHAR, playlist_id VARCHAR, playlist_version_id VARCHAR,
		pair_id VARCHAR, pair_version_id VARCHAR,
		map_id VARCHAR, map_version_id VARCHAR,
		game_variant_id VARCHAR, game_variant_version_id VARCHAR)`); err != nil {
		t.Fatalf("create match_registry: %v", err)
	}
	meta, err = sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open meta: %v", err)
	}
	t.Cleanup(func() { meta.Close() })
	if _, err := meta.Exec(`CREATE TABLE catalog_fetch_queue (
		title_slug VARCHAR, asset_type VARCHAR, asset_id VARCHAR, version_id VARCHAR,
		attempts INTEGER DEFAULT 0, last_error VARCHAR,
		PRIMARY KEY (title_slug, asset_type, asset_id, version_id))`); err != nil {
		t.Fatalf("create catalog_fetch_queue: %v", err)
	}
	return shared, meta
}

func TestSeedCatalogQueueFromRegistry(t *testing.T) {
	ctx := context.Background()
	shared, meta := openQueueTestDBs(t)

	// 2 matchs : playlist p1 partagée (dédoublonnée), pairs distinctes, 1 map.
	for _, row := range [][]any{
		{"m1", "p1", "pv1", "pair1", "prv1", "map1", "mv1", "gv1", "gvv1"},
		{"m2", "p1", "pv1", "pair2", "prv2", "map1", "mv1", "gv1", "gvv1"},
	} {
		if _, err := shared.Exec(`INSERT INTO match_registry VALUES (?,?,?,?,?,?,?,?,?)`, row...); err != nil {
			t.Fatalf("seed match: %v", err)
		}
	}

	res, err := SeedCatalogQueueFromRegistry(ctx, meta, shared, "halo_infinite")
	if err != nil {
		t.Fatalf("SeedCatalogQueueFromRegistry: %v", err)
	}
	// playlist p1 dédoublonnée → 1 ; 2 pairs ; 1 map ; 1 game_variant.
	if res.Playlists != 1 || res.Pairs != 2 || res.Maps != 1 || res.GameVariants != 1 {
		t.Fatalf("counts = %d/%d/%d/%d (attendu 1/2/1/1)",
			res.Playlists, res.Pairs, res.Maps, res.GameVariants)
	}
	if res.Total() != 5 {
		t.Errorf("Total = %d (attendu 5)", res.Total())
	}

	var queued int
	if err := meta.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_fetch_queue`).Scan(&queued); err != nil {
		t.Fatalf("count queue: %v", err)
	}
	if queued != 5 {
		t.Errorf("queue rows = %d (attendu 5)", queued)
	}

	// Idempotence : un second seed n'insère plus rien (INSERT OR IGNORE).
	again, err := SeedCatalogQueueFromRegistry(ctx, meta, shared, "halo_infinite")
	if err != nil {
		t.Fatalf("re-seed: %v", err)
	}
	if again.Total() != 0 {
		t.Errorf("re-seed Total = %d (attendu 0, idempotent)", again.Total())
	}
}
