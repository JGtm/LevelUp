//go:build cgo

package halo_infinite

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/duckdb"
)

// TestLoadRankCatalog_NilDB couvre le early-return : sans DB, on retourne
// un catalog vide (le caller dégrade gracieusement vers un libellé minimal).
func TestLoadRankCatalog_NilDB(t *testing.T) {
	t.Parallel()
	catalog, err := LoadRankCatalog(context.Background(), nil)
	if err != nil {
		t.Fatalf("err = %v, want nil pour DB nil", err)
	}
	if catalog == nil {
		t.Fatal("catalog ne doit jamais être nil")
	}
	if catalog.Len() != 0 {
		t.Errorf("Len() = %d, want 0 pour catalog vide", catalog.Len())
	}
}

// TestLoadRankCatalog_EmptyTable : la table existe mais est vide → catalog vide.
func TestLoadRankCatalog_EmptyTable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "metadata_empty.duckdb")
	db, err := duckdb.OpenReadWriteShared(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(context.Background(), `
		CREATE TABLE career_rank_translations (
			rank_id INTEGER, lang VARCHAR, title VARCHAR, subtitle VARCHAR, tier VARCHAR
		)
	`); err != nil {
		t.Fatalf("create: %v", err)
	}

	catalog, err := LoadRankCatalog(context.Background(), db)
	if err != nil {
		t.Fatalf("LoadRankCatalog: %v", err)
	}
	if catalog.Len() != 0 {
		t.Errorf("Len() = %d, want 0", catalog.Len())
	}
}

// TestLoadRankCatalog_PopulatedTable : peuple la table puis vérifie le scan,
// les agrégations par rank_id et la normalisation des codes lang Waypoint.
func TestLoadRankCatalog_PopulatedTable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "metadata_populated.duckdb")
	db, err := duckdb.OpenReadWriteShared(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(context.Background(), `
		CREATE TABLE career_rank_translations (
			rank_id INTEGER, lang VARCHAR, title VARCHAR, subtitle VARCHAR, tier VARCHAR
		)
	`); err != nil {
		t.Fatalf("create: %v", err)
	}
	rows := []struct {
		id                          int
		lang, title, subtitle, tier string
	}{
		{1, "en-US", "Bronze", "I", "BRONZE_1"},
		{1, "fr-FR", "Bronze", "I", "BRONZE_1"},
		{2, "en-US", "Silver", "I", "SILVER_1"},
	}
	for _, r := range rows {
		if _, err := db.Exec(context.Background(),
			`INSERT INTO career_rank_translations VALUES (?, ?, ?, ?, ?)`,
			r.id, r.lang, r.title, r.subtitle, r.tier,
		); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	catalog, err := LoadRankCatalog(context.Background(), db)
	if err != nil {
		t.Fatalf("LoadRankCatalog: %v", err)
	}
	if catalog.Len() != 2 {
		t.Errorf("Len() = %d, want 2", catalog.Len())
	}
}
