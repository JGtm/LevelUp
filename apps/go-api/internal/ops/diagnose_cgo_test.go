//go:build cgo

// Package ops — diagnose_cgo_test.go : tests de DiagnoseDB sur une DuckDB
// de fichier temporaire.
//
// Lancer avec : go test ./internal/ops/ -v (CGO_ENABLED=1 requis)
package ops

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openDiagDB crée une DuckDB temporaire de diagnostic (distincte de openTempDB).
func openDiagDB(t *testing.T) (path string, db *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	path = filepath.Join(dir, "diag.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("openDiagDB: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("openDiagDB ping: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return path, db
}

// ─────────────────────────────────────────────────────────────────────────────
// DiagnoseDB
// ─────────────────────────────────────────────────────────────────────────────

func TestDiagnoseDB_EmptyDB(t *testing.T) {
	path, db := openDiagDB(t)
	db.Close() // fermer avant DiagnoseDB (read_only)

	report, err := DiagnoseDB(context.Background(), DiagnoseOptions{DBPath: path})
	if err != nil {
		t.Fatalf("DiagnoseDB inattendu: %v", err)
	}
	if report.DBPath != path {
		t.Errorf("DBPath = %q, want %q", report.DBPath, path)
	}
	if len(report.Tables) != 0 {
		t.Errorf("expected 0 tables sur DB vide, got %d", len(report.Tables))
	}
}

func TestDiagnoseDB_WithTable(t *testing.T) {
	path, db := openDiagDB(t)
	if _, err := db.Exec("CREATE TABLE test_diag (id INTEGER, name TEXT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO test_diag VALUES (1, 'alpha'), (2, 'beta')"); err != nil {
		t.Fatal(err)
	}
	db.Close()

	report, err := DiagnoseDB(context.Background(), DiagnoseOptions{DBPath: path, Verbose: true})
	if err != nil {
		t.Fatalf("DiagnoseDB inattendu: %v", err)
	}
	if len(report.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(report.Tables))
	}
	if report.Tables[0].Name != "test_diag" {
		t.Errorf("table name = %q, want test_diag", report.Tables[0].Name)
	}
	if report.Tables[0].RowCount != 2 {
		t.Errorf("row count = %d, want 2", report.Tables[0].RowCount)
	}
}

func TestDiagnoseDB_WithView(t *testing.T) {
	path, db := openDiagDB(t)
	if _, err := db.Exec("CREATE TABLE base_tbl (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE VIEW v_test_diag AS SELECT * FROM base_tbl"); err != nil {
		t.Fatal(err)
	}
	db.Close()

	report, err := DiagnoseDB(context.Background(), DiagnoseOptions{DBPath: path})
	if err != nil {
		t.Fatalf("DiagnoseDB inattendu: %v", err)
	}
	found := false
	for _, v := range report.Views {
		if v == "v_test_diag" {
			found = true
		}
	}
	if !found {
		t.Errorf("vue v_test_diag absente du rapport: %v", report.Views)
	}
}

func TestDiagnoseDB_InvalidPath(t *testing.T) {
	_, err := DiagnoseDB(context.Background(), DiagnoseOptions{DBPath: "/totally/nonexistent/path.duckdb"})
	if err == nil {
		t.Error("expected error pour path invalide")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// describeAllTables / listViews / listIndexes (indirectement via DiagnoseDB)
// ─────────────────────────────────────────────────────────────────────────────

func TestDescribeAllTables_Empty(t *testing.T) {
	_, db := openDiagDB(t)
	tables, err := describeAllTables(context.Background(), db, false)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if len(tables) != 0 {
		t.Errorf("expected 0, got %d", len(tables))
	}
}

func TestListViews_Empty(t *testing.T) {
	_, db := openDiagDB(t)
	views, err := listViews(context.Background(), db)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if len(views) != 0 {
		t.Errorf("expected 0 vues, got %d", len(views))
	}
}

func TestListIndexes_Empty(t *testing.T) {
	_, db := openDiagDB(t)
	indexes, err := listIndexes(context.Background(), db)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if len(indexes) != 0 {
		t.Errorf("expected 0 index, got %d", len(indexes))
	}
}

func TestListIndexes_WithIndex(t *testing.T) {
	_, db := openDiagDB(t)
	if _, err := db.Exec("CREATE TABLE idx_tbl (id INTEGER, name TEXT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE INDEX idx_name ON idx_tbl(name)"); err != nil {
		t.Fatal(err)
	}
	// listIndexes n'échoue pas même si information_schema.statistics varie
	_, err := listIndexes(context.Background(), db)
	if err != nil {
		t.Fatalf("listIndexes inattendu: %v", err)
	}
}

func TestFormatDiagnoseReport_Basic(t *testing.T) {
	report := DiagnoseReport{
		DBPath: "/test/db.duckdb",
		Tables: []TableSchema{
			{Name: "match_registry", Columns: []ColumnInfo{{Name: "id", DataType: "VARCHAR"}}, RowCount: 42},
			// RowCount = -1 teste la branche "pas d'affichage count"
			{Name: "empty_table", Columns: []ColumnInfo{{Name: "x", DataType: "INTEGER"}}, RowCount: -1},
		},
		Views:   []string{"v_match_full"},
		Indexes: nil,
	}
	out := FormatDiagnoseReport(report)
	if out == "" {
		t.Error("FormatDiagnoseReport ne doit pas retourner vide")
	}
	if !stringContains(out, "match_registry") {
		t.Errorf("rapport ne contient pas match_registry")
	}
	if !stringContains(out, "v_match_full") {
		t.Errorf("rapport ne contient pas v_match_full")
	}
}

func stringContains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
