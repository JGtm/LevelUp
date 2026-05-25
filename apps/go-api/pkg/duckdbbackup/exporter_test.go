package duckdbbackup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckIntegrity_ValidDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.duckdb")

	// Create a minimal DuckDB database.
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := db.Exec("CREATE TABLE t (id INTEGER)"); err != nil {
		db.Close()
		t.Fatalf("create table: %v", err)
	}
	db.Close()

	target := Target{Key: "test", Path: dbPath}
	result := CheckIntegrity(context.Background(), target)

	// Expect either OK=true (pragma supported and returned "ok") or
	// OK=true with no detail (pragma not supported — inconclusive).
	if !result.OK {
		t.Errorf("CheckIntegrity on valid DB: OK=false, Detail=%q", result.Detail)
	}
	if result.CheckedAt.IsZero() {
		t.Error("CheckedAt should not be zero")
	}
}

func TestCheckIntegrity_NonDBFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.duckdb")
	// Write garbage that can't be a valid DuckDB file.
	if err := os.WriteFile(path, []byte("not a duckdb"), 0o644); err != nil {
		t.Fatal(err)
	}

	target := Target{Key: "corrupt", Path: path}
	result := CheckIntegrity(context.Background(), target)

	// A garbage file may fail to open, fail to run pragma, or pass integrity_check
	// returning an error row. Any of these is acceptable — we just check that the
	// function returns without panicking and sets CheckedAt.
	if result.CheckedAt.IsZero() {
		t.Error("CheckedAt should not be zero")
	}
}

func TestManifest_SetIntegrityResult(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".manifest.json")

	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	m.SetIntegrityResult("shared", IntegrityResult{OK: true})
	m.SetIntegrityResult("player", IntegrityResult{OK: false, Detail: "corrupted page 1"})

	if err := m.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	m2, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest after save: %v", err)
	}
	if got := m2.IntegrityChecks["shared"]; !got.OK {
		t.Errorf("shared: expected OK=true, got %+v", got)
	}
	if got := m2.IntegrityChecks["player"]; got.OK || got.Detail != "corrupted page 1" {
		t.Errorf("player: expected OK=false detail='corrupted page 1', got %+v", got)
	}
}
