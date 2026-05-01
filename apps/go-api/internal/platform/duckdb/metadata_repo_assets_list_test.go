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
	return &DB{sqlDB: sqlDB, path: ":memory:"}
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

// TestListMapsByTitle_EmptyRegistry_StillReturnsNames est le test de régression pour le bug
// où l'INNER JOIN sur map_images_registry retournait 0 lignes quand la table était vide
// (images pas encore mises en cache via populate-assets). Les noms doivent s'afficher
// même si aucune image n'est en cache.
func TestListMapsByTitle_EmptyRegistry_StillReturnsNames(t *testing.T) {
	db := openAssetMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	// On crée les tables SANS map_images_registry peuplé (table vide).
	ddl := []string{
		`CREATE TABLE map_images_registry (
			title_id   VARCHAR NOT NULL,
			map_id     VARCHAR NOT NULL,
			local_path VARCHAR NOT NULL DEFAULT '',
			fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (title_id, map_id)
		)`,
		`CREATE TABLE asset_translations (
			asset_id   VARCHAR NOT NULL,
			asset_type VARCHAR NOT NULL,
			lang       VARCHAR NOT NULL,
			name       VARCHAR NOT NULL DEFAULT '',
			description VARCHAR NOT NULL DEFAULT '',
			fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (asset_id, asset_type, lang)
		)`,
		`CREATE TABLE weapon_labels (weapon_id UBIGINT PRIMARY KEY, name_en VARCHAR, name_fr VARCHAR)`,
	}
	for _, q := range ddl {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("DDL: %v", err)
		}
	}
	// Traductions présentes, mais map_images_registry est VIDE.
	fixtures := []string{
		`INSERT INTO asset_translations VALUES ('map-001','map','en-US','Aquarius','',now())`,
		`INSERT INTO asset_translations VALUES ('map-002','map','en-US','Breaker','',now())`,
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
		t.Errorf("registry vide: attendu 2 maps via asset_translations, obtenu %d", len(maps))
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
func setupAssetDrawerFixtures(t *testing.T, db *DB, ctx context.Context) {
	t.Helper()

	ddl := []string{
		`CREATE TABLE map_images_registry (
			title_id   VARCHAR NOT NULL,
			map_id     VARCHAR NOT NULL,
			local_path VARCHAR NOT NULL DEFAULT '',
			fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (title_id, map_id)
		)`,
		`CREATE TABLE asset_translations (
			asset_id   VARCHAR NOT NULL,
			asset_type VARCHAR NOT NULL,
			lang       VARCHAR NOT NULL,
			name       VARCHAR NOT NULL DEFAULT '',
			description VARCHAR NOT NULL DEFAULT '',
			fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
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
		`INSERT INTO map_images_registry VALUES ('halo_infinite','map-001','',now())`,
		`INSERT INTO map_images_registry VALUES ('halo_infinite','map-002','',now())`,
		`INSERT INTO map_images_registry VALUES ('halo_infinite','map-003','',now())`,
		`INSERT INTO asset_translations VALUES ('map-001','map','en-US','Aquarius','',now())`,
		`INSERT INTO asset_translations VALUES ('map-001','map','fr-FR','Aquarius','',now())`,
		`INSERT INTO asset_translations VALUES ('map-002','map','en-US','Breaker','',now())`,
		`INSERT INTO asset_translations VALUES ('map-003','map','en-US','Streets','',now())`,
		`INSERT INTO weapon_labels VALUES (100,'BR75 Battle Rifle','Fusil BR75')`,
		`INSERT INTO weapon_labels VALUES (200,'Skewer','Brochette')`,
	}
	for _, q := range fixtures {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
}
