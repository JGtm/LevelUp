//go:build cgo

// Package ops — archive_restore_cgo_test.go : tests CGO pour listArchivableYears
// et restoreTable sur des DuckDB temporaires.
package ops

import (
	"context"
	"testing"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// listArchivableYears — retourne [] sur une DB sans les tables requises
// ─────────────────────────────────────────────────────────────────────────────

func TestListArchivableYears_DBEmpty(t *testing.T) {
	_, db := openDiagDB(t)
	// Créer les tables nécessaires mais vides
	if _, err := db.Exec(`CREATE TABLE match_participants (xuid VARCHAR, match_id VARCHAR)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE match_registry (match_id VARCHAR, start_time TIMESTAMPTZ)`); err != nil {
		t.Fatal(err)
	}

	cutoff := time.Now()
	years, err := listArchivableYears(context.Background(), db, "xuid_test", cutoff)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if len(years) != 0 {
		t.Errorf("expected 0 années, got %v", years)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// restoreTable — erreur sur parquet invalide (chemin inexistant)
// ─────────────────────────────────────────────────────────────────────────────

func TestRestoreTable_InvalidParquet(t *testing.T) {
	_, db := openDiagDB(t)
	err := restoreTable(context.Background(), db, "new_table", "/nonexistent/data.parquet", false)
	if err == nil {
		t.Error("expected error pour parquet invalide")
	}
}

func TestRestoreTable_ReplaceWithInvalidParquet(t *testing.T) {
	_, db := openDiagDB(t)
	// Créer la table initiale
	if _, err := db.Exec("CREATE TABLE my_table (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	// Avec replace=true : DROP TABLE IF EXISTS doit passer, puis CREATE échoue
	err := restoreTable(context.Background(), db, "my_table", "/nonexistent/data.parquet", true)
	if err == nil {
		t.Error("expected error pour parquet invalide")
	}
}
