// Package ops — season_catalog_refresh_cgo_test.go : RefreshSeasonCatalog sur
// DuckDB :memory: (driver CGO). Vérifie le peuplement, la résolution FR (fallback
// EN), le parsing du numéro de saison, et l'idempotence ART-safe (un 2e passage
// emprunte le chemin UPDATE SELECT-then-write, sans doublon ni crash).
package ops

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/domain"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openSeasonCatalogTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE season_catalog (
		title_slug VARCHAR NOT NULL, season_id VARCHAR NOT NULL,
		display_name VARCHAR, name_fr VARCHAR,
		season_major INTEGER, season_minor INTEGER,
		first_seen_at TIMESTAMP, last_fetched_at TIMESTAMP,
		PRIMARY KEY (title_slug, season_id))`); err != nil {
		t.Fatalf("create season_catalog: %v", err)
	}
	return db
}

func TestRefreshSeasonCatalog_UpsertAndIdempotent(t *testing.T) {
	db := openSeasonCatalogTestDB(t)
	ctx := context.Background()
	seasons := []domain.WorldSeasonRef{
		{SeasonID: "csrseason12-1", DisplayName: "Shadows", NameFR: "Ombres"},
		{SeasonID: "csrseason13-2", DisplayName: "Infinite", NameFR: ""}, // FR vide → fallback EN
		{SeasonID: "", DisplayName: "ignorée"},                           // id vide → skip
	}

	n, err := RefreshSeasonCatalog(ctx, db, "halo_infinite", seasons)
	if err != nil {
		t.Fatalf("RefreshSeasonCatalog: %v", err)
	}
	if n != 2 {
		t.Errorf("upserts = %d, attendu 2 (l'entrée id vide est ignorée)", n)
	}

	// Ligne Shadows : FR persisté, numéro parsé.
	var displayName, nameFR string
	var major, minor int
	if err := db.QueryRow(`SELECT display_name, name_fr, season_major, season_minor
		FROM season_catalog WHERE title_slug = ? AND season_id = ?`,
		"halo_infinite", "csrseason12-1").Scan(&displayName, &nameFR, &major, &minor); err != nil {
		t.Fatalf("select shadows: %v", err)
	}
	if displayName != "Shadows" || nameFR != "Ombres" || major != 12 || minor != 1 {
		t.Errorf("csrseason12-1 = {%q,%q,%d,%d}, attendu {Shadows,Ombres,12,1}", displayName, nameFR, major, minor)
	}

	// Ligne Infinite : FR vide → fallback EN.
	if err := db.QueryRow(`SELECT name_fr FROM season_catalog WHERE season_id = ?`,
		"csrseason13-2").Scan(&nameFR); err != nil {
		t.Fatalf("select infinite: %v", err)
	}
	if nameFR != "Infinite" {
		t.Errorf("csrseason13-2 name_fr = %q, attendu fallback \"Infinite\"", nameFR)
	}

	// Idempotence : 2e passage (chemin UPDATE), nom FR corrigé, aucun doublon.
	seasons[0].NameFR = "Ténèbres"
	if _, err := RefreshSeasonCatalog(ctx, db, "halo_infinite", seasons); err != nil {
		t.Fatalf("RefreshSeasonCatalog (2e): %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM season_catalog`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("lignes = %d après 2e passage, attendu 2 (pas de doublon)", count)
	}
	if err := db.QueryRow(`SELECT name_fr FROM season_catalog WHERE season_id = ?`,
		"csrseason12-1").Scan(&nameFR); err != nil {
		t.Fatalf("select shadows (2e): %v", err)
	}
	if nameFR != "Ténèbres" {
		t.Errorf("name_fr après update = %q, attendu \"Ténèbres\"", nameFR)
	}
}
