//go:build integration

package sync

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func setupBackfillRegistryDBs(t *testing.T) (sharedDB, metadataDB *sql.DB) {
	t.Helper()
	var err error
	sharedDB, err = sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	t.Cleanup(func() { sharedDB.Close() })
	metadataDB, err = sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open meta: %v", err)
	}
	t.Cleanup(func() { metadataDB.Close() })

	if _, err := sharedDB.Exec(`CREATE TABLE match_registry (
		match_id VARCHAR PRIMARY KEY,
		playlist_id VARCHAR, playlist_name VARCHAR,
		map_id VARCHAR, map_name VARCHAR,
		pair_id VARCHAR, pair_name VARCHAR,
		game_variant_id VARCHAR, game_variant_name VARCHAR)`); err != nil {
		t.Fatalf("schema shared: %v", err)
	}
	if _, err := metadataDB.Exec(`CREATE TABLE asset_translations (
		asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR,
		PRIMARY KEY (asset_id, asset_type, lang))`); err != nil {
		t.Fatalf("schema meta: %v", err)
	}
	return sharedDB, metadataDB
}

func TestBackfillRegistryNames_FixesUUIDFallback(t *testing.T) {
	ctx := context.Background()
	sharedDB, metaDB := setupBackfillRegistryDBs(t)

	// Match avec UUID brut comme nom (cas du fallback `coalesceStrPtr`).
	const playlistID = "uuid-quick-play"
	const mapID = "uuid-aquarius"
	if _, err := sharedDB.Exec(`INSERT INTO match_registry VALUES
		('m1', ?, ?, ?, ?, NULL, NULL, NULL, NULL),
		('m2', ?, ?, NULL, NULL, NULL, NULL, NULL, NULL),
		('m3', ?, 'Quick Play', ?, 'Aquarius', NULL, NULL, NULL, NULL)`,
		playlistID, playlistID, mapID, mapID, // m1: les deux UUID
		playlistID, playlistID, // m2: playlist UUID
		playlistID, mapID); err != nil { // m3: déjà résolus
		t.Fatalf("seed shared: %v", err)
	}
	if _, err := metaDB.Exec(`INSERT INTO asset_translations VALUES
		(?, 'playlist', 'en-US', 'Quick Play'),
		(?, 'map', 'en-US', 'Aquarius')`, playlistID, mapID); err != nil {
		t.Fatalf("seed meta: %v", err)
	}

	stats, err := BackfillRegistryNames(ctx, sharedDB, metaDB)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	// Stats : 1 playlist_id distinct à fixer (uuid-quick-play présent dans m1+m2 mais distinct=1)
	if stats.PlaylistsScanned != 1 || stats.PlaylistsFixed != 1 {
		t.Errorf("playlists scanned=%d fixed=%d, want 1/1", stats.PlaylistsScanned, stats.PlaylistsFixed)
	}
	if stats.MapsScanned != 1 || stats.MapsFixed != 1 {
		t.Errorf("maps scanned=%d fixed=%d, want 1/1", stats.MapsScanned, stats.MapsFixed)
	}

	// Vérifie que m1 et m2 ont leur playlist_name remplacé par "Quick Play".
	var name string
	if err := sharedDB.QueryRow(`SELECT playlist_name FROM match_registry WHERE match_id = 'm1'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Quick Play" {
		t.Errorf("m1 playlist_name = %q, want Quick Play", name)
	}
	if err := sharedDB.QueryRow(`SELECT playlist_name FROM match_registry WHERE match_id = 'm2'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Quick Play" {
		t.Errorf("m2 playlist_name = %q, want Quick Play", name)
	}
	// m3 doit rester "Quick Play" (était déjà correct)
	if err := sharedDB.QueryRow(`SELECT playlist_name FROM match_registry WHERE match_id = 'm3'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Quick Play" {
		t.Errorf("m3 playlist_name = %q (préservation)", name)
	}
}

func TestBackfillRegistryNames_NoTranslation_KeepUUID(t *testing.T) {
	ctx := context.Background()
	sharedDB, metaDB := setupBackfillRegistryDBs(t)

	const unknownID = "uuid-unknown"
	if _, err := sharedDB.Exec(`INSERT INTO match_registry VALUES
		('m1', ?, ?, NULL, NULL, NULL, NULL, NULL, NULL)`,
		unknownID, unknownID); err != nil {
		t.Fatal(err)
	}
	// Pas d'asset_translations pour ce UUID.

	stats, err := BackfillRegistryNames(ctx, sharedDB, metaDB)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if stats.PlaylistsScanned != 1 || stats.PlaylistsFixed != 0 {
		t.Errorf("scanned=%d fixed=%d, want 1/0", stats.PlaylistsScanned, stats.PlaylistsFixed)
	}
	var name string
	if err := sharedDB.QueryRow(`SELECT playlist_name FROM match_registry WHERE match_id = 'm1'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != unknownID {
		t.Errorf("UUID inconnu : got %q, want %q (préservation)", name, unknownID)
	}
}

func TestBackfillRegistryNames_Idempotent(t *testing.T) {
	ctx := context.Background()
	sharedDB, metaDB := setupBackfillRegistryDBs(t)
	const playlistID = "uuid-quick-play"
	if _, err := sharedDB.Exec(`INSERT INTO match_registry VALUES
		('m1', ?, ?, NULL, NULL, NULL, NULL, NULL, NULL)`,
		playlistID, playlistID); err != nil {
		t.Fatal(err)
	}
	if _, err := metaDB.Exec(`INSERT INTO asset_translations VALUES
		(?, 'playlist', 'en-US', 'Quick Play')`, playlistID); err != nil {
		t.Fatal(err)
	}

	stats1, _ := BackfillRegistryNames(ctx, sharedDB, metaDB)
	stats2, _ := BackfillRegistryNames(ctx, sharedDB, metaDB)
	if stats1.PlaylistsFixed != 1 {
		t.Errorf("run 1 fixed=%d, want 1", stats1.PlaylistsFixed)
	}
	if stats2.PlaylistsScanned != 0 || stats2.PlaylistsFixed != 0 {
		t.Errorf("run 2 (idempotent) scanned=%d fixed=%d, want 0/0",
			stats2.PlaylistsScanned, stats2.PlaylistsFixed)
	}
}

// TestBackfillRegistryNames_ConstructsPairFromParts : la paire est absente
// d'asset_translations (reste un GUID après la passe colonne) mais map +
// game_variant y sont → l'étape de construction réécrit pair_name en
// "{game_variant} on {map}". Cas du nouveau contenu Halo (maps inédites).
func TestBackfillRegistryNames_ConstructsPairFromParts(t *testing.T) {
	ctx := context.Background()
	sharedDB, metaDB := setupBackfillRegistryDBs(t)

	const pairGUID = "uuid-pair-absent"
	const gvID = "uuid-gv-slayer"
	const mapID = "uuid-map-chasm"
	// m1 : pair/gv/map tous en GUID. m2 : pair a une vraie traduction → pas de construction.
	if _, err := sharedDB.Exec(`INSERT INTO match_registry VALUES
		('m1', NULL, NULL, ?, ?, ?, ?, ?, ?),
		('m2', NULL, NULL, ?, ?, 'pair-real', 'Arena:Slayer on Chasm', ?, ?)`,
		mapID, mapID, pairGUID, pairGUID, gvID, gvID,
		mapID, mapID, gvID, gvID); err != nil {
		t.Fatalf("seed shared: %v", err)
	}
	// asset_translations a map + game_variant, mais PAS la paire.
	if _, err := metaDB.Exec(`INSERT INTO asset_translations VALUES
		(?, 'map', 'en-US', 'Chasm'),
		(?, 'game_variant', 'en-US', 'Slayer')`, mapID, gvID); err != nil {
		t.Fatalf("seed meta: %v", err)
	}

	stats, err := BackfillRegistryNames(ctx, sharedDB, metaDB)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if stats.PairsFixed != 1 {
		t.Errorf("PairsFixed = %d, want 1 (construction)", stats.PairsFixed)
	}
	var name string
	if err := sharedDB.QueryRow(`SELECT pair_name FROM match_registry WHERE match_id = 'm1'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Slayer on Chasm" {
		t.Errorf("m1 pair_name = %q, want %q", name, "Slayer on Chasm")
	}
	// m2 garde sa vraie traduction (pas de construction).
	if err := sharedDB.QueryRow(`SELECT pair_name FROM match_registry WHERE match_id = 'm2'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Arena:Slayer on Chasm" {
		t.Errorf("m2 pair_name = %q, want préservé", name)
	}

	// Idempotence : un 2e run ne reconstruit rien (pair_name != pair_id désormais).
	stats2, _ := BackfillRegistryNames(ctx, sharedDB, metaDB)
	if stats2.PairsFixed != 0 {
		t.Errorf("run 2 PairsFixed = %d, want 0 (idempotent)", stats2.PairsFixed)
	}
}

func TestBackfillRegistryNames_NilMetadata_NoOp(t *testing.T) {
	ctx := context.Background()
	sharedDB, _ := setupBackfillRegistryDBs(t)
	if _, err := sharedDB.Exec(`INSERT INTO match_registry VALUES
		('m1', 'uuid', 'uuid', NULL, NULL, NULL, NULL, NULL, NULL)`); err != nil {
		t.Fatal(err)
	}
	stats, err := BackfillRegistryNames(ctx, sharedDB, nil)
	if err != nil {
		t.Fatalf("nil metadata should be no-op: %v", err)
	}
	if stats.Total() != 0 {
		t.Errorf("total = %d, want 0", stats.Total())
	}
}
