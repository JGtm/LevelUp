package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openAssetMemDB ouvre une DB DuckDB in-memory pour les tests de l'Asset Drawer.
// Défini localement : repo_test.go est taggué //go:build integration.
func openAssetMemDB(t *testing.T) *DB {
	t.Helper()
	sqlDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("openAssetMemDB: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return newTestDB(sqlDB, ":memory:")
}

func TestListMapsByTitle_All(t *testing.T) {
	db := openAssetMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	setupAssetDrawerFixtures(t, db, ctx)

	maps, err := repo.ListMapsByTitle(ctx, "halo_infinite", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(maps) != 3 {
		t.Errorf("ListMapsByTitle(all): attendu 3, obtenu %d", len(maps))
	}
}

func TestListMapsByTitle_Search(t *testing.T) {
	db := openAssetMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	setupAssetDrawerFixtures(t, db, ctx)

	maps, err := repo.ListMapsByTitle(ctx, "halo_infinite", "aqu")
	if err != nil {
		t.Fatal(err)
	}
	if len(maps) != 1 {
		t.Errorf("ListMapsByTitle(aqu): attendu 1, obtenu %d", len(maps))
	}
	if maps[0].ID != "map-001" {
		t.Errorf("ID=%q, attendu map-001", maps[0].ID)
	}
	if maps[0].NameEN != "Aquarius" {
		t.Errorf("NameEN=%q, attendu Aquarius", maps[0].NameEN)
	}
}

// TestListMapsByTitle_NoTranslations_FallsBackToNameCanonical est le test de régression
// pour le cas où asset_translations est vide (populate-assets pas encore lancé).
// Les maps doivent s'afficher via maps_catalog.name_canonical même sans traductions.
func TestListMapsByTitle_NoTranslations_FallsBackToNameCanonical(t *testing.T) {
	db := openAssetMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	ddl := []string{
		`CREATE TABLE maps_catalog (
			title_slug         VARCHAR NOT NULL,
			map_asset_id       VARCHAR NOT NULL,
			current_version_id VARCHAR,
			name_canonical     VARCHAR,
			image_url          VARCHAR,
			last_fetched_at    TIMESTAMP,
			PRIMARY KEY (title_slug, map_asset_id)
		)`,
		`CREATE TABLE asset_translations (
			asset_id    VARCHAR NOT NULL,
			asset_type  VARCHAR NOT NULL,
			lang        VARCHAR NOT NULL,
			name        VARCHAR NOT NULL DEFAULT '',
			description VARCHAR NOT NULL DEFAULT '',
			fetched_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (asset_id, asset_type, lang)
		)`,
		`CREATE TABLE weapon_labels (weapon_id UBIGINT PRIMARY KEY, name_en VARCHAR, name_fr VARCHAR)`,
	}
	for _, q := range ddl {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("DDL: %v", err)
		}
	}
	// maps_catalog peuplé, asset_translations VIDE.
	fixtures := []string{
		`INSERT INTO maps_catalog (title_slug, map_asset_id, name_canonical) VALUES ('halo_infinite','map-001','Aquarius')`,
		`INSERT INTO maps_catalog (title_slug, map_asset_id, name_canonical) VALUES ('halo_infinite','map-002','Breaker')`,
	}
	for _, q := range fixtures {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}

	maps, err := repo.ListMapsByTitle(ctx, "halo_infinite", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(maps) != 2 {
		t.Errorf("sans traductions: attendu 2 maps via name_canonical, obtenu %d", len(maps))
	}
	if maps[0].NameEN != "Aquarius" {
		t.Errorf("NameEN=%q, attendu Aquarius (fallback name_canonical)", maps[0].NameEN)
	}
}

func TestListWeaponsByTitle_All(t *testing.T) {
	db := openAssetMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	setupAssetDrawerFixtures(t, db, ctx)

	weapons, err := repo.ListWeaponsByTitle(ctx, "halo_infinite", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(weapons) != 2 {
		t.Errorf("ListWeaponsByTitle(all): attendu 2, obtenu %d", len(weapons))
	}
}

func TestListWeaponsByTitle_Search(t *testing.T) {
	db := openAssetMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	setupAssetDrawerFixtures(t, db, ctx)

	weapons, err := repo.ListWeaponsByTitle(ctx, "halo_infinite", "BR75")
	if err != nil {
		t.Fatal(err)
	}
	if len(weapons) != 1 {
		t.Errorf("ListWeaponsByTitle(BR75): attendu 1, obtenu %d", len(weapons))
	}
	if weapons[0].NameEN != "BR75 Battle Rifle" {
		t.Errorf("NameEN=%q", weapons[0].NameEN)
	}
}

// setupAssetDrawerFixtures crée les tables et insère 3 maps + 2 armes en mémoire.
// maps_catalog est la source primaire ; asset_translations enrichit les noms (optionnel).
func setupAssetDrawerFixtures(t *testing.T, db *DB, ctx context.Context) {
	t.Helper()

	ddl := []string{
		`CREATE TABLE maps_catalog (
			title_slug         VARCHAR NOT NULL,
			map_asset_id       VARCHAR NOT NULL,
			current_version_id VARCHAR,
			name_canonical     VARCHAR,
			image_url          VARCHAR,
			last_fetched_at    TIMESTAMP,
			PRIMARY KEY (title_slug, map_asset_id)
		)`,
		`CREATE TABLE asset_translations (
			asset_id    VARCHAR NOT NULL,
			asset_type  VARCHAR NOT NULL,
			lang        VARCHAR NOT NULL,
			name        VARCHAR NOT NULL DEFAULT '',
			description VARCHAR NOT NULL DEFAULT '',
			fetched_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (asset_id, asset_type, lang)
		)`,
		`CREATE TABLE weapon_labels (
			weapon_id UBIGINT PRIMARY KEY,
			name_en   VARCHAR,
			name_fr   VARCHAR
		)`,
	}
	for _, q := range ddl {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("DDL: %v", err)
		}
	}

	fixtures := []string{
		// maps_catalog — source primaire (toujours peuplée)
		`INSERT INTO maps_catalog (title_slug, map_asset_id, name_canonical) VALUES ('halo_infinite','map-001','Aquarius')`,
		`INSERT INTO maps_catalog (title_slug, map_asset_id, name_canonical) VALUES ('halo_infinite','map-002','Breaker')`,
		`INSERT INTO maps_catalog (title_slug, map_asset_id, name_canonical) VALUES ('halo_infinite','map-003','Streets')`,
		// asset_translations — enrichissement optionnel (EN + FR pour map-001 uniquement)
		`INSERT INTO asset_translations VALUES ('map-001','map','en-US','Aquarius','',now())`,
		`INSERT INTO asset_translations VALUES ('map-001','map','fr-FR','Aquarius','',now())`,
		`INSERT INTO asset_translations VALUES ('map-002','map','en-US','Breaker','',now())`,
		// map-003 n'a pas de traduction → nom vient de name_canonical
		`INSERT INTO weapon_labels VALUES (100,'BR75 Battle Rifle','Fusil BR75')`,
		`INSERT INTO weapon_labels VALUES (200,'Skewer','Brochette')`,
	}
	for _, q := range fixtures {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
}

// TestListMapsByTitle_DedupeParAssetIDPasParNom fige la correction D15 (2026-09-13).
//
// CE QUI ETAIT FAUX : la requete dedupliquait par `DISTINCT ON (m.name_canonical)`. Deux
// cartes HOMONYMES d'asset_id distincts — une variante Forge republiee, une carte et son
// portage classe — s'effondraient donc en UNE SEULE entree du drawer, et l'asset_id
// survivant etait choisi par l'ordre de tri, pas par une regle. Or l'asset_id est
// precisement la cle sur laquelle le web identifie une carte : le drawer perdait une carte
// jouable et en designait une autre a sa place.
//
// CE QUE LE TEST EXIGE : les deux homonymes sortent, chacune avec SON asset_id ; et une
// carte dont asset_translations porte des doublons ne sort toujours qu'une fois (c'est
// l'unique deduplication que la requete doit faire — celle des lignes produites par ses
// propres jointures).
func TestListMapsByTitle_DedupeParAssetIDPasParNom(t *testing.T) {
	db := openAssetMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	setupAssetDrawerFixtures(t, db, ctx)

	// Deux cartes HOMONYMES de 'Aquarius' (map-001), d'asset_id distincts.
	extra := []string{
		`INSERT INTO maps_catalog (title_slug, map_asset_id, name_canonical) VALUES ('halo_infinite','map-004','Aquarius')`,
		`INSERT INTO asset_translations VALUES ('map-004','map','en-US','Aquarius','',now())`,
	}
	for _, q := range extra {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("fixture homonyme: %v", err)
		}
	}

	maps, err := repo.ListMapsByTitle(ctx, "halo_infinite", "")
	if err != nil {
		t.Fatal(err)
	}

	ids := map[string]int{}
	for _, m := range maps {
		ids[m.ID]++
	}
	for _, want := range []string{"map-001", "map-002", "map-003", "map-004"} {
		if ids[want] != 1 {
			t.Errorf("asset_id %q present %d fois, attendu 1 — les homonymes doivent sortir "+
				"toutes les deux, chacune avec son asset_id (deduplication par nom = D15)",
				want, ids[want])
		}
	}
	if len(maps) != 4 {
		t.Errorf("%d carte(s), attendu 4 : %v", len(maps), ids)
	}

	// La recherche voit elle aussi les DEUX homonymes.
	trouvees, err := repo.ListMapsByTitle(ctx, "halo_infinite", "aqu")
	if err != nil {
		t.Fatal(err)
	}
	if len(trouvees) != 2 {
		t.Errorf("recherche 'aqu' : %d resultat(s), attendu 2 (map-001 et map-004)", len(trouvees))
	}
}
