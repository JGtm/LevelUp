//go:build integration

// catalog_test.go — tests Phase A du catalogue Playlists/Pairs/Maps, déplacés depuis
// internal/migration (Phase 1.5 b12). Vérifie que add_catalog_playlists crée les 8
// tables, est idempotent, et isole par title_slug. Utilise openMetadataDB (provider
// StepsFor câblé), défini dans milestones_test.go.
package migrations

import (
	"database/sql"
	"testing"
)

func catalogTableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", table,
	).Scan(&n); err != nil {
		t.Fatalf("table exists %s: %v", table, err)
	}
	return n > 0
}

func TestRunForDB_Catalog_AllTablesCreated(t *testing.T) {
	db := openMetadataDB(t)

	expectedTables := []string{
		"playlists_catalog",
		"maps_catalog",
		"game_variants_catalog",
		"map_mode_pair_definitions",
		"playlist_pair_links",
		"catalog_fetch_queue",
		"pair_mode_label_translations",
		"unknown_prefix_candidates",
	}
	for _, tbl := range expectedTables {
		if !catalogTableExists(t, db, tbl) {
			t.Errorf("table catalogue %q absente après migration", tbl)
		}
	}
}

func TestRunForDB_Catalog_TitleSlugIsolation(t *testing.T) {
	db := openMetadataDB(t)

	// Même playlist_asset_id sur 2 titres différents — doit fonctionner (PK composite).
	playlistID := "abc-123-uuid"
	if _, err := db.Exec(
		`INSERT INTO playlists_catalog (title_slug, playlist_asset_id, name_canonical) VALUES (?, ?, ?)`,
		"halo_infinite", playlistID, "Quick Play",
	); err != nil {
		t.Fatalf("insert halo_infinite: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO playlists_catalog (title_slug, playlist_asset_id, name_canonical) VALUES (?, ?, ?)`,
		"synthetic_title_b", playlistID, "Synth Quick Play",
	); err != nil {
		t.Fatalf("insert synthetic_title_b (isolation cross-titre violée): %v", err)
	}

	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM playlists_catalog WHERE playlist_asset_id = ?`, playlistID,
	).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Errorf("isolation title_slug : 2 lignes attendues, %d trouvées", n)
	}

	// Duplicat sur le même (title_slug, playlist_asset_id) → doit échouer.
	_, err := db.Exec(
		`INSERT INTO playlists_catalog (title_slug, playlist_asset_id, name_canonical) VALUES (?, ?, ?)`,
		"halo_infinite", playlistID, "Duplicate",
	)
	if err == nil {
		t.Error("INSERT duplicat (PK composite) aurait dû échouer")
	}
}

func TestRunForDB_Catalog_QueueAndPrefixCandidates(t *testing.T) {
	db := openMetadataDB(t)

	// catalog_fetch_queue — INSERT OR IGNORE supporté.
	if _, err := db.Exec(
		`INSERT OR IGNORE INTO catalog_fetch_queue (title_slug, asset_type, asset_id) VALUES (?, ?, ?)`,
		"halo_infinite", "playlist", "uuid-1",
	); err != nil {
		t.Fatalf("INSERT OR IGNORE queue: %v", err)
	}
	if _, err := db.Exec(
		`INSERT OR IGNORE INTO catalog_fetch_queue (title_slug, asset_type, asset_id) VALUES (?, ?, ?)`,
		"halo_infinite", "playlist", "uuid-1", // doublon — doit être ignoré
	); err != nil {
		t.Fatalf("INSERT OR IGNORE queue (dup): %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM catalog_fetch_queue`).Scan(&n); err != nil {
		t.Fatalf("count queue: %v", err)
	}
	if n != 1 {
		t.Errorf("INSERT OR IGNORE doublon : %d lignes attendues 1", n)
	}

	// unknown_prefix_candidates — pair_examples VARCHAR[] supporté.
	if _, err := db.Exec(
		`INSERT INTO unknown_prefix_candidates (title_slug, prefix, n_matches, pair_examples) VALUES (?, ?, ?, ?)`,
		"halo_infinite", "Mega Fiesta", 3, "[\"a\", \"b\", \"c\"]",
	); err != nil {
		t.Fatalf("insert unknown_prefix_candidates: %v", err)
	}
}
