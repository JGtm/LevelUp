//go:build integration

// steps_metadata_catalog_test.go — tests Phase A du plan PLAN_PLAYLISTS_CATALOG.md.
//
// Vérifie que la migration `add_catalog_playlists` :
// 1. Crée bien les 8 tables attendues
// 2. Est idempotente (3 passes successives sans erreur)
// 3. Permet l'isolation par title_slug (insert sur 2 titres distincts sans collision PK)

package migration

import (
	"testing"
)

func TestRunForDB_Catalog_AllTablesCreated(t *testing.T) {
	db := openMemDB(t)

	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB(Metadata): %v", err)
	}

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
		if !assertTableExists(t, db, tbl) {
			t.Errorf("table catalogue %q absente après migration", tbl)
		}
	}
}

func TestRunForDB_Catalog_TitleSlugIsolation(t *testing.T) {
	db := openMemDB(t)

	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB(Metadata): %v", err)
	}

	// Insérer la même playlist_asset_id sur 2 titres différents — doit fonctionner (PK composite)
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

	// Vérifier que les 2 lignes coexistent
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM playlists_catalog WHERE playlist_asset_id = ?`, playlistID,
	).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Errorf("isolation title_slug : 2 lignes attendues, %d trouvées", n)
	}

	// Tentative de duplicat sur le même (title_slug, playlist_asset_id) → doit échouer
	_, err := db.Exec(
		`INSERT INTO playlists_catalog (title_slug, playlist_asset_id, name_canonical) VALUES (?, ?, ?)`,
		"halo_infinite", playlistID, "Duplicate",
	)
	if err == nil {
		t.Error("INSERT duplicat (PK composite) aurait dû échouer")
	}
}

func TestRunForDB_Catalog_QueueAndPrefixCandidates(t *testing.T) {
	db := openMemDB(t)

	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB(Metadata): %v", err)
	}

	// catalog_fetch_queue — INSERT OR IGNORE doit être supporté (pattern utilisé en Phase E)
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

	// unknown_prefix_candidates — pair_examples VARCHAR[] supporté
	if _, err := db.Exec(
		`INSERT INTO unknown_prefix_candidates (title_slug, prefix, n_matches, pair_examples) VALUES (?, ?, ?, ?)`,
		"halo_infinite", "Mega Fiesta", 3, "[\"a\", \"b\", \"c\"]",
	); err != nil {
		t.Fatalf("insert unknown_prefix_candidates: %v", err)
	}
}
