//go:build cgo

// Package validation — gate_cgo_test.go : tests DuckDB pour les fonctions
// checkDBAccessible, checkTablesExist, checkViewsExist, checkSharedTables,
// checkSharedViews, checkMigrationsApplied, checkPlayerDB.
//
// Lancer avec : go test ./internal/validation/ -v (CGO_ENABLED=1 requis)
package validation

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openTestDB crée une DuckDB temporaire et retourne son chemin + connexion.
func openTestDB(t *testing.T) (string, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "gate_test.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return path, db
}

// openClosedDB crée une DuckDB temporaire puis la ferme (pour tests read-only).
func openClosedDB(t *testing.T) string {
	t.Helper()
	path, db := openTestDB(t)
	db.Close()
	return path
}

// ─────────────────────────────────────────────────────────────────────────────
// checkDBAccessible
// ─────────────────────────────────────────────────────────────────────────────

func TestCheckDBAccessible_Absent(t *testing.T) {
	ok, msg := checkDBAccessible("/nonexistent/path/test.duckdb")
	if ok {
		t.Error("expected false pour DB absente")
	}
	if msg == "" {
		t.Error("message ne doit pas être vide")
	}
}

func TestCheckDBAccessible_Valid(t *testing.T) {
	path := openClosedDB(t)
	ok, msg := checkDBAccessible(path)
	if !ok {
		t.Errorf("expected true pour DB valide, got msg=%q", msg)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// checkTablesExist
// ─────────────────────────────────────────────────────────────────────────────

func TestCheckTablesExist_AllMissing(t *testing.T) {
	_, db := openTestDB(t)
	missing := checkTablesExist(db, []string{"table_a", "table_b"})
	if len(missing) != 2 {
		t.Errorf("expected 2 missing, got %d: %v", len(missing), missing)
	}
}

func TestCheckTablesExist_PartialPresent(t *testing.T) {
	_, db := openTestDB(t)
	if _, err := db.Exec("CREATE TABLE present_table (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	missing := checkTablesExist(db, []string{"present_table", "absent_table"})
	if len(missing) != 1 || missing[0] != "absent_table" {
		t.Errorf("expected [absent_table], got %v", missing)
	}
}

func TestCheckTablesExist_AllPresent(t *testing.T) {
	_, db := openTestDB(t)
	if _, err := db.Exec("CREATE TABLE tbl1 (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE tbl2 (name TEXT)"); err != nil {
		t.Fatal(err)
	}
	missing := checkTablesExist(db, []string{"tbl1", "tbl2"})
	if len(missing) != 0 {
		t.Errorf("expected 0 missing, got %v", missing)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// checkViewsExist
// ─────────────────────────────────────────────────────────────────────────────

func TestCheckViewsExist_AllMissing(t *testing.T) {
	_, db := openTestDB(t)
	missing := checkViewsExist(db, []string{"v_nonexistent"})
	if len(missing) != 1 {
		t.Errorf("expected 1 missing, got %d", len(missing))
	}
}

func TestCheckViewsExist_WithView(t *testing.T) {
	_, db := openTestDB(t)
	if _, err := db.Exec("CREATE TABLE base (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE VIEW v_test AS SELECT * FROM base"); err != nil {
		t.Fatal(err)
	}
	missing := checkViewsExist(db, []string{"v_test", "v_absent"})
	if len(missing) != 1 || missing[0] != "v_absent" {
		t.Errorf("expected [v_absent], got %v", missing)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// checkSharedTables
// ─────────────────────────────────────────────────────────────────────────────

func TestCheckSharedTables_DBAbsent(t *testing.T) {
	ok, msg := checkSharedTables("/nonexistent/shared.duckdb")
	if ok {
		t.Error("expected false pour DB absente")
	}
	if msg == "" {
		t.Error("message ne doit pas être vide")
	}
}

func TestCheckSharedTables_DBEmpty(t *testing.T) {
	path := openClosedDB(t)
	ok, msg := checkSharedTables(path)
	if ok {
		t.Errorf("expected false (tables manquantes), got msg=%q", msg)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// checkSharedViews
// ─────────────────────────────────────────────────────────────────────────────

func TestCheckSharedViews_DBAbsent(t *testing.T) {
	ok, msg := checkSharedViews("/nonexistent/shared.duckdb")
	if ok {
		t.Error("expected false pour DB absente")
	}
	_ = msg
}

func TestCheckSharedViews_DBEmpty(t *testing.T) {
	path := openClosedDB(t)
	ok, msg := checkSharedViews(path)
	if ok {
		t.Errorf("expected false (vues manquantes), got msg=%q", msg)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// checkMigrationsApplied
// ─────────────────────────────────────────────────────────────────────────────

func TestCheckMigrationsApplied_DBAbsent(t *testing.T) {
	dir := t.TempDir()
	ok, msg := checkMigrationsApplied(dir, "GhostPlayer")
	if ok {
		t.Error("expected false (DB absente)")
	}
	_ = msg
}

// ─────────────────────────────────────────────────────────────────────────────
// checkPlayerDB
// ─────────────────────────────────────────────────────────────────────────────

func TestCheckPlayerDB_FileAbsent(t *testing.T) {
	ok, msg := checkPlayerDB("/nonexistent/stats.duckdb")
	if ok {
		t.Error("expected false (fichier absent)")
	}
	_ = msg
}

func TestCheckPlayerDB_TableAbsent(t *testing.T) {
	path := openClosedDB(t)
	ok, msg := checkPlayerDB(path)
	if ok {
		t.Errorf("expected false (table absente), got msg=%q", msg)
	}
}

func TestCheckPlayerDB_TableEmpty(t *testing.T) {
	_, db := openTestDB(t)
	if _, err := db.Exec("CREATE TABLE player_match_enrichment (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	path, _ := func() (string, *sql.DB) {
		return openTestDB(t)
	}()
	// Créer une DB avec la table vide
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "player_empty.duckdb")
	db2, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db2.Exec("CREATE TABLE player_match_enrichment (id INTEGER)"); err != nil {
		db2.Close()
		t.Fatal(err)
	}
	db2.Close()
	_ = path // path de openTestDB, pas utilisé

	ok, msg := checkPlayerDB(dbPath)
	if ok {
		t.Errorf("expected false (table vide), got msg=%q", msg)
	}
}
