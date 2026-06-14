// Package ops — catalog_refresh_cgo_test.go : CatalogRefreshFromRegistry sur
// DuckDB :memory: (driver CGO). Vérifie le peuplement des tables catalog ET
// l'idempotence ART-safe : un 2e passage emprunte le chemin UPDATE (SELECT-then-
// write) sans crash ni doublon — l'ancien `INSERT ... ON CONFLICT DO UPDATE`
// pouvait FATAL-invalider metadata.duckdb (bug ART).
package ops

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openCatalogRefreshTestDBs(t *testing.T) (shared, meta *sql.DB) {
	t.Helper()
	shared, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	t.Cleanup(func() { shared.Close() })
	if _, err := shared.Exec(`CREATE TABLE match_registry (
		match_id VARCHAR, start_time TIMESTAMP,
		playlist_id VARCHAR, playlist_version_id VARCHAR, playlist_name VARCHAR, is_ranked BOOLEAN,
		map_id VARCHAR, map_version_id VARCHAR, map_name VARCHAR,
		game_variant_id VARCHAR, game_variant_version_id VARCHAR, game_variant_name VARCHAR,
		pair_id VARCHAR, pair_version_id VARCHAR, pair_name VARCHAR, pair_name_fr VARCHAR,
		mode_category VARCHAR)`); err != nil {
		t.Fatalf("create match_registry: %v", err)
	}

	meta, err = sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open meta: %v", err)
	}
	t.Cleanup(func() { meta.Close() })
	stmts := []string{
		`CREATE TABLE playlists_catalog (title_slug VARCHAR, playlist_asset_id VARCHAR,
			current_version_id VARCHAR, name_canonical VARCHAR, experience VARCHAR,
			is_ranked BOOLEAN, is_active BOOLEAN, first_seen_at TIMESTAMP, last_seen_at TIMESTAMP,
			last_fetched_at TIMESTAMP, PRIMARY KEY (title_slug, playlist_asset_id))`,
		`CREATE TABLE maps_catalog (title_slug VARCHAR, map_asset_id VARCHAR,
			current_version_id VARCHAR, name_canonical VARCHAR, last_fetched_at TIMESTAMP,
			PRIMARY KEY (title_slug, map_asset_id))`,
		`CREATE TABLE game_variants_catalog (title_slug VARCHAR, game_variant_asset_id VARCHAR,
			current_version_id VARCHAR, name_canonical VARCHAR, mode_canonical VARCHAR,
			last_fetched_at TIMESTAMP, PRIMARY KEY (title_slug, game_variant_asset_id))`,
		`CREATE TABLE map_mode_pair_definitions (title_slug VARCHAR, pair_asset_id VARCHAR,
			current_version_id VARCHAR, name_canonical VARCHAR, map_asset_id VARCHAR,
			game_variant_asset_id VARCHAR, mode_category VARCHAR, last_fetched_at TIMESTAMP,
			PRIMARY KEY (title_slug, pair_asset_id))`,
		`CREATE TABLE pair_mode_label_translations (title_slug VARCHAR, pair_asset_id VARCHAR,
			lang VARCHAR, label VARCHAR, PRIMARY KEY (title_slug, pair_asset_id, lang))`,
	}
	for _, s := range stmts {
		if _, err := meta.Exec(s); err != nil {
			t.Fatalf("create catalog table: %v", err)
		}
	}
	return shared, meta
}

func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func TestCatalogRefreshFromRegistry_PopulatesAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	shared, meta := openCatalogRefreshTestDBs(t)

	if _, err := shared.Exec(`INSERT INTO match_registry VALUES
		('m1', '2026-05-01 20:00:00',
		 'pl1', 'plv1', 'Arena', TRUE,
		 'map1', 'mv1', 'Live Fire',
		 'gv1', 'gvv1', 'Slayer',
		 'pair1', 'prv1', 'Arena:Slayer on Live Fire', 'Assassin sur Live Fire',
		 'Assassin')`); err != nil {
		t.Fatalf("seed match: %v", err)
	}

	// 1er passage → INSERT.
	r, err := CatalogRefreshFromRegistry(ctx, meta, shared, "halo_infinite")
	if err != nil {
		t.Fatalf("1er refresh: %v", err)
	}
	if r.Playlists != 1 || r.Maps != 1 || r.GameVariants != 1 || r.Pairs != 1 {
		t.Fatalf("counts = %d/%d/%d/%d, want 1/1/1/1", r.Playlists, r.Maps, r.GameVariants, r.Pairs)
	}

	// 2e passage → chemin UPDATE (SELECT-then-write). Doit réussir SANS crash ni doublon.
	if _, err := CatalogRefreshFromRegistry(ctx, meta, shared, "halo_infinite"); err != nil {
		t.Fatalf("2e refresh (UPDATE path) — crash ART potentiel : %v", err)
	}

	// Idempotent : 1 ligne par table (UPDATE, pas un 2e INSERT).
	for _, tbl := range []string{"playlists_catalog", "maps_catalog", "game_variants_catalog", "map_mode_pair_definitions"} {
		if got := countRows(t, meta, tbl); got != 1 {
			t.Errorf("%s = %d lignes, want 1 (idempotent)", tbl, got)
		}
	}
	// Labels EN + FR de la paire.
	if got := countRows(t, meta, "pair_mode_label_translations"); got != 2 {
		t.Errorf("pair_mode_label_translations = %d, want 2 (en + fr)", got)
	}

	// Valeurs upsertées correctement.
	var name, exp string
	var ranked bool
	if err := meta.QueryRow(`SELECT name_canonical, experience, is_ranked FROM playlists_catalog WHERE playlist_asset_id='pl1'`).
		Scan(&name, &exp, &ranked); err != nil {
		t.Fatalf("read playlist: %v", err)
	}
	if name != "Arena" || exp != "ranked" || !ranked {
		t.Errorf("playlist = (%q,%q,%v), want (Arena,ranked,true)", name, exp, ranked)
	}
}
