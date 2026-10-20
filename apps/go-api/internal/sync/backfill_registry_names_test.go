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
		game_variant_id VARCHAR, game_variant_name VARCHAR,
		mode_category VARCHAR)`); err != nil {
		t.Fatalf("schema shared: %v", err)
	}
	if _, err := metadataDB.Exec(`CREATE TABLE asset_translations (
		asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR,
		PRIMARY KEY (asset_id, asset_type, lang))`); err != nil {
		t.Fatalf("schema meta: %v", err)
	}
	return sharedDB, metadataDB
}

func nomDuRegistre(t *testing.T, db *sql.DB, matchID, col string) sql.NullString {
	t.Helper()
	var n sql.NullString
	// col vient des tests eux-mêmes (liste fermée), jamais d'une entrée.
	if err := db.QueryRow(`SELECT `+col+` FROM match_registry WHERE match_id = ?`, matchID).Scan(&n); err != nil {
		t.Fatalf("lecture %s.%s: %v", matchID, col, err)
	}
	return n
}

// TestBackfillRegistryNames_FixesUUIDFallback : les noms égaux à l'identifiant sont remplacés
// par la traduction en-US ; les comptes sont PAR MATCH ; un vrai nom reste en place.
func TestBackfillRegistryNames_FixesUUIDFallback(t *testing.T) {
	ctx := context.Background()
	sharedDB, metaDB := setupBackfillRegistryDBs(t)

	const playlistID = "uuid-quick-play"
	const mapID = "uuid-aquarius"
	if _, err := sharedDB.Exec(`INSERT INTO match_registry VALUES
		('m1', ?, ?, ?, ?, NULL, NULL, NULL, NULL, NULL),
		('m2', ?, ?, NULL, NULL, NULL, NULL, NULL, NULL, NULL),
		('m3', ?, 'Quick Play', ?, 'Aquarius', NULL, NULL, NULL, NULL, NULL)`,
		playlistID, playlistID, mapID, mapID, // m1: les deux en identifiant
		playlistID, playlistID, // m2: playlist en identifiant
		playlistID, mapID); err != nil { // m3: déjà résolus
		t.Fatalf("seed shared: %v", err)
	}
	if _, err := metaDB.Exec(`INSERT INTO asset_translations VALUES
		(?, 'playlist', 'en-US', 'Quick Play'),
		(?, 'map', 'en-US', 'Aquarius')`, playlistID, mapID); err != nil {
		t.Fatalf("seed meta: %v", err)
	}

	stats, err := BackfillRegistryNames(ctx, sharedDB, metaDB, RegistryNamesOptions{})
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if stats.PlaylistsScanned != 2 || stats.PlaylistsFixed != 2 {
		t.Errorf("playlists scanned=%d fixed=%d, want 2/2 (m1, m2)", stats.PlaylistsScanned, stats.PlaylistsFixed)
	}
	if stats.MapsScanned != 1 || stats.MapsFixed != 1 {
		t.Errorf("maps scanned=%d fixed=%d, want 1/1", stats.MapsScanned, stats.MapsFixed)
	}
	for _, id := range []string{"m1", "m2", "m3"} {
		if got := nomDuRegistre(t, sharedDB, id, "playlist_name"); got.String != "Quick Play" {
			t.Errorf("%s playlist_name = %q, want Quick Play", id, got.String)
		}
	}
}

// TestBackfillRegistryNames_NullNames : un nom NULL (vidé par un ancien outil de réparation) est
// réécrit comme un nom égal à l'identifiant ; la catégorie suit le nom de la paire.
func TestBackfillRegistryNames_NullNames(t *testing.T) {
	ctx := context.Background()
	sharedDB, metaDB := setupBackfillRegistryDBs(t)
	if _, err := sharedDB.Exec(`INSERT INTO match_registry VALUES
		('m1', NULL, NULL, 'map-x', NULL, 'pair-x', NULL, NULL, NULL, 'Other')`); err != nil {
		t.Fatalf("seed shared: %v", err)
	}
	if _, err := metaDB.Exec(`INSERT INTO asset_translations VALUES
		('map-x', 'map', 'en-US', 'Streets'),
		('pair-x', 'pair', 'en-US', 'Arena:CTF on Streets')`); err != nil {
		t.Fatalf("seed meta: %v", err)
	}
	stats, err := BackfillRegistryNames(ctx, sharedDB, metaDB, RegistryNamesOptions{})
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if stats.MapsFixed != 1 || stats.PairsFixed != 1 || stats.PairsConstructed != 0 {
		t.Errorf("stats = %+v, want 1 carte et 1 paire traduites", stats)
	}
	if got := nomDuRegistre(t, sharedDB, "m1", "map_name"); got.String != "Streets" {
		t.Errorf("map_name = %q, want Streets", got.String)
	}
	if got := nomDuRegistre(t, sharedDB, "m1", "pair_name"); got.String != "Arena:CTF on Streets" {
		t.Errorf("pair_name = %q", got.String)
	}
	if got := nomDuRegistre(t, sharedDB, "m1", "mode_category"); got.String != "Assassin" {
		t.Errorf("mode_category = %q, want Assassin (catégorie de Arena:CTF on Streets)", got.String)
	}
}

// TestBackfillRegistryNames_CategorieNullOuIndexeeNonReecrite : une catégorie NULL (titre qui ne
// la renseigne pas) reste NULL ; et tant qu'un index couvre la colonne (base pas encore migrée),
// aucune catégorie n'est réécrite (UPDATE d'une colonne indexée = vecteur ART).
func TestBackfillRegistryNames_CategorieNullOuIndexeeNonReecrite(t *testing.T) {
	ctx := context.Background()
	sharedDB, metaDB := setupBackfillRegistryDBs(t)
	if _, err := sharedDB.Exec(`INSERT INTO match_registry VALUES
		('m1', NULL, NULL, NULL, NULL, 'pair-x', NULL, NULL, NULL, NULL),
		('m2', NULL, NULL, NULL, NULL, 'pair-x', NULL, NULL, NULL, 'Other')`); err != nil {
		t.Fatalf("seed shared: %v", err)
	}
	if _, err := metaDB.Exec(`INSERT INTO asset_translations VALUES
		('pair-x', 'pair', 'en-US', 'Ranked:CTF on Streets')`); err != nil {
		t.Fatalf("seed meta: %v", err)
	}
	if _, err := sharedDB.Exec(`CREATE INDEX idx_registry_mode_category ON match_registry(mode_category)`); err != nil {
		t.Fatalf("index: %v", err)
	}
	if _, err := BackfillRegistryNames(ctx, sharedDB, metaDB, RegistryNamesOptions{}); err != nil {
		t.Fatalf("backfill indexée: %v", err)
	}
	if got := nomDuRegistre(t, sharedDB, "m2", "mode_category"); got.String != "Other" {
		t.Errorf("colonne indexée : mode_category = %q, doit rester Other", got.String)
	}
	if got := nomDuRegistre(t, sharedDB, "m2", "pair_name"); got.String != "Ranked:CTF on Streets" {
		t.Errorf("le nom doit être écrit malgré tout, pair_name = %q", got.String)
	}
}

// TestBackfillRegistryNames_CategorieSuitLeNomSansIndex : sans index sur la colonne, la
// catégorie est réécrite avec le nom (NULL conservée pour un titre qui ne la renseigne pas).
func TestBackfillRegistryNames_CategorieSuitLeNomSansIndex(t *testing.T) {
	ctx := context.Background()
	sharedDB, metaDB := setupBackfillRegistryDBs(t)
	if _, err := sharedDB.Exec(`INSERT INTO match_registry VALUES
		('m1', NULL, NULL, NULL, NULL, 'pair-x', NULL, NULL, NULL, NULL),
		('m2', NULL, NULL, NULL, NULL, 'pair-x', NULL, NULL, NULL, 'Other')`); err != nil {
		t.Fatalf("seed shared: %v", err)
	}
	if _, err := metaDB.Exec(`INSERT INTO asset_translations VALUES
		('pair-x', 'pair', 'en-US', 'Ranked:CTF on Streets')`); err != nil {
		t.Fatalf("seed meta: %v", err)
	}
	if _, err := BackfillRegistryNames(ctx, sharedDB, metaDB, RegistryNamesOptions{}); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if got := nomDuRegistre(t, sharedDB, "m2", "mode_category"); got.String != "Ranked" {
		t.Errorf("m2 mode_category = %q, want Ranked", got.String)
	}
	if got := nomDuRegistre(t, sharedDB, "m1", "mode_category"); got.Valid {
		t.Errorf("m1 mode_category = %q, une catégorie NULL ne se remplit pas", got.String)
	}
}

// TestBackfillRegistryNames_DryRunCompteSansEcrire : la simulation rend les mêmes comptes que la
// passe réelle et n'écrit rien.
func TestBackfillRegistryNames_DryRunCompteSansEcrire(t *testing.T) {
	ctx := context.Background()
	sharedDB, metaDB := setupBackfillRegistryDBs(t)
	if _, err := sharedDB.Exec(`INSERT INTO match_registry VALUES
		('m1', NULL, NULL, 'map-x', 'map-x', 'pair-x', 'pair-x', 'gv-x', NULL, NULL),
		('m2', NULL, NULL, 'map-y', NULL, NULL, NULL, NULL, NULL, NULL)`); err != nil {
		t.Fatalf("seed shared: %v", err)
	}
	if _, err := metaDB.Exec(`INSERT INTO asset_translations VALUES
		('map-x', 'map', 'en-US', 'Streets'),
		('gv-x', 'game_variant', 'en-US', 'CTF')`); err != nil {
		t.Fatalf("seed meta: %v", err)
	}
	sim, err := BackfillRegistryNames(ctx, sharedDB, metaDB, RegistryNamesOptions{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if !sim.DryRun || sim.MapsScanned != 2 || sim.MapsFixed != 1 || sim.VariantsFixed != 1 ||
		sim.PairsFixed != 1 || sim.PairsConstructed != 1 {
		t.Errorf("simulation = %+v, want 2 cartes candidates dont 1 réparable, 1 variante, 1 paire construite", sim)
	}
	if got := nomDuRegistre(t, sharedDB, "m1", "map_name"); got.String != "map-x" {
		t.Fatalf("la simulation a écrit : map_name = %q", got.String)
	}
	reel, err := BackfillRegistryNames(ctx, sharedDB, metaDB, RegistryNamesOptions{})
	if err != nil {
		t.Fatalf("réel: %v", err)
	}
	reel.DryRun = true
	if reel != sim {
		t.Errorf("réel = %+v, want les comptes de la simulation %+v", reel, sim)
	}
}

func TestBackfillRegistryNames_NoTranslation_KeepUUID(t *testing.T) {
	ctx := context.Background()
	sharedDB, metaDB := setupBackfillRegistryDBs(t)

	const unknownID = "uuid-unknown"
	if _, err := sharedDB.Exec(`INSERT INTO match_registry VALUES
		('m1', ?, ?, NULL, NULL, NULL, NULL, NULL, NULL, NULL)`,
		unknownID, unknownID); err != nil {
		t.Fatal(err)
	}
	stats, err := BackfillRegistryNames(ctx, sharedDB, metaDB, RegistryNamesOptions{})
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if stats.PlaylistsScanned != 1 || stats.PlaylistsFixed != 0 {
		t.Errorf("scanned=%d fixed=%d, want 1/0", stats.PlaylistsScanned, stats.PlaylistsFixed)
	}
	if got := nomDuRegistre(t, sharedDB, "m1", "playlist_name"); got.String != unknownID {
		t.Errorf("identifiant inconnu : got %q, want %q (préservation)", got.String, unknownID)
	}
}

func TestBackfillRegistryNames_Idempotent(t *testing.T) {
	ctx := context.Background()
	sharedDB, metaDB := setupBackfillRegistryDBs(t)
	const playlistID = "uuid-quick-play"
	if _, err := sharedDB.Exec(`INSERT INTO match_registry VALUES
		('m1', ?, ?, NULL, NULL, NULL, NULL, NULL, NULL, NULL)`,
		playlistID, playlistID); err != nil {
		t.Fatal(err)
	}
	if _, err := metaDB.Exec(`INSERT INTO asset_translations VALUES
		(?, 'playlist', 'en-US', 'Quick Play')`, playlistID); err != nil {
		t.Fatal(err)
	}

	stats1, _ := BackfillRegistryNames(ctx, sharedDB, metaDB, RegistryNamesOptions{})
	stats2, _ := BackfillRegistryNames(ctx, sharedDB, metaDB, RegistryNamesOptions{})
	if stats1.PlaylistsFixed != 1 {
		t.Errorf("run 1 fixed=%d, want 1", stats1.PlaylistsFixed)
	}
	if stats2.PlaylistsScanned != 0 || stats2.PlaylistsFixed != 0 {
		t.Errorf("run 2 (idempotent) scanned=%d fixed=%d, want 0/0",
			stats2.PlaylistsScanned, stats2.PlaylistsFixed)
	}
}

// TestBackfillRegistryNames_ConstructsPairFromParts : la paire est absente
// d'asset_translations mais carte et variante y sont → pair_name est construit
// "{variante} on {carte}" à partir des noms convergés dans la même passe.
func TestBackfillRegistryNames_ConstructsPairFromParts(t *testing.T) {
	ctx := context.Background()
	sharedDB, metaDB := setupBackfillRegistryDBs(t)

	const pairGUID = "uuid-pair-absent"
	const gvID = "uuid-gv-slayer"
	const mapID = "uuid-map-chasm"
	// m1 : pair/gv/map tous en identifiant. m2 : la paire a un vrai nom → pas de construction.
	if _, err := sharedDB.Exec(`INSERT INTO match_registry VALUES
		('m1', NULL, NULL, ?, ?, ?, ?, ?, ?, NULL),
		('m2', NULL, NULL, ?, ?, 'pair-real', 'Arena:Slayer on Chasm', ?, ?, NULL)`,
		mapID, mapID, pairGUID, pairGUID, gvID, gvID,
		mapID, mapID, gvID, gvID); err != nil {
		t.Fatalf("seed shared: %v", err)
	}
	if _, err := metaDB.Exec(`INSERT INTO asset_translations VALUES
		(?, 'map', 'en-US', 'Chasm'),
		(?, 'game_variant', 'en-US', 'Slayer')`, mapID, gvID); err != nil {
		t.Fatalf("seed meta: %v", err)
	}

	stats, err := BackfillRegistryNames(ctx, sharedDB, metaDB, RegistryNamesOptions{})
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if stats.PairsFixed != 1 || stats.PairsConstructed != 1 {
		t.Errorf("PairsFixed=%d PairsConstructed=%d, want 1/1 (construction)", stats.PairsFixed, stats.PairsConstructed)
	}
	if got := nomDuRegistre(t, sharedDB, "m1", "pair_name"); got.String != "Slayer on Chasm" {
		t.Errorf("m1 pair_name = %q, want %q", got.String, "Slayer on Chasm")
	}
	if got := nomDuRegistre(t, sharedDB, "m2", "pair_name"); got.String != "Arena:Slayer on Chasm" {
		t.Errorf("m2 pair_name = %q, want préservé", got.String)
	}
	stats2, _ := BackfillRegistryNames(ctx, sharedDB, metaDB, RegistryNamesOptions{})
	if stats2.PairsFixed != 0 {
		t.Errorf("run 2 PairsFixed = %d, want 0 (idempotent)", stats2.PairsFixed)
	}
}

func TestBackfillRegistryNames_NilMetadata_NoOp(t *testing.T) {
	ctx := context.Background()
	sharedDB, _ := setupBackfillRegistryDBs(t)
	if _, err := sharedDB.Exec(`INSERT INTO match_registry VALUES
		('m1', 'uuid', 'uuid', NULL, NULL, NULL, NULL, NULL, NULL, NULL)`); err != nil {
		t.Fatal(err)
	}
	stats, err := BackfillRegistryNames(ctx, sharedDB, nil, RegistryNamesOptions{})
	if err != nil {
		t.Fatalf("nil metadata should be no-op: %v", err)
	}
	if stats.Total() != 0 {
		t.Errorf("total = %d, want 0", stats.Total())
	}
}

// TestBackfillRegistryNames_SimulationSansMetadonnees_CompteLesCandidats : simulation sans base
// de métadonnées — chaque colonne NULL ou égale à son identifiant est comptée candidate, aucune
// n'est réparable (pas de source), un vrai nom n'est pas compté, rien n'est écrit.
func TestBackfillRegistryNames_SimulationSansMetadonnees_CompteLesCandidats(t *testing.T) {
	ctx := context.Background()
	sharedDB, _ := setupBackfillRegistryDBs(t)
	if _, err := sharedDB.Exec(`INSERT INTO match_registry VALUES
		('m1', 'pl', 'pl', 'map', NULL, 'pair', 'pair', 'gv', 'Slayer', 'other'),
		('m2', 'pl', 'Ranked', 'map', 'map', NULL, NULL, 'gv', 'gv', 'other')`); err != nil {
		t.Fatal(err)
	}
	stats, err := BackfillRegistryNames(ctx, sharedDB, nil, RegistryNamesOptions{DryRun: true})
	if err != nil {
		t.Fatalf("simulation sans métadonnées: %v", err)
	}
	want := BackfillRegistryStats{DryRun: true, PlaylistsScanned: 1, MapsScanned: 2, PairsScanned: 1, VariantsScanned: 1}
	if stats != want {
		t.Errorf("stats = %+v, want %+v", stats, want)
	}
	var nom sql.NullString
	if err := sharedDB.QueryRow(`SELECT map_name FROM match_registry WHERE match_id = 'm1'`).Scan(&nom); err != nil || nom.Valid {
		t.Errorf("m1 map_name = %v (err %v), want NULL (aucune écriture)", nom, err)
	}
}
