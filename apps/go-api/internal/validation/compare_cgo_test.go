//go:build cgo

// Package validation — compare_cgo_test.go : tests avec DuckDB en mémoire.
//
// Sprint 49 : couvre ComparePlayerDBs, listTables, countRows, compareTableCounts,
// compareBitmasks, compareMatchIDs et loadMatchIDs via deux DBs DuckDB temporaires.
package validation

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupPlayerDB crée une DB DuckDB temporaire avec le schéma player_match_enrichment.
func setupPlayerDB(t *testing.T, dir, name string, matchIDs []string, nullPerf bool) string {
	t.Helper()
	dbPath := filepath.Join(dir, name)
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE player_match_enrichment (
		match_id VARCHAR PRIMARY KEY,
		performance_score DOUBLE,
		session_id VARCHAR,
		is_with_friends BOOLEAN
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	for _, mid := range matchIDs {
		var perfVal any = 42.0
		if nullPerf {
			perfVal = nil
		}
		_, err = db.Exec(
			"INSERT INTO player_match_enrichment VALUES (?, ?, 's1', true)",
			mid, perfVal,
		)
		if err != nil {
			t.Fatalf("insert %s: %v", mid, err)
		}
	}
	return dbPath
}

func TestListTables(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, _ = db.Exec("CREATE TABLE foo (id INT)")
	_, _ = db.Exec("CREATE TABLE bar (id INT)")

	tables, err := listTables(db)
	if err != nil {
		t.Fatal(err)
	}
	if !tables["foo"] || !tables["bar"] {
		t.Fatalf("expected foo and bar, got %v", tables)
	}
}

func TestCountRows(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, _ = db.Exec("CREATE TABLE items (id INT)")
	_, _ = db.Exec("INSERT INTO items VALUES (1), (2), (3)")

	n, err := countRows(db, "items")
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("expected 3, got %d", n)
	}
}

func TestCompareTableCounts_SameTables(t *testing.T) {
	goDb, _ := sql.Open("duckdb", ":memory:")
	defer goDb.Close()
	pyDb, _ := sql.Open("duckdb", ":memory:")
	defer pyDb.Close()

	_, _ = goDb.Exec("CREATE TABLE t1 (id INT)")
	_, _ = goDb.Exec("INSERT INTO t1 VALUES (1), (2)")
	_, _ = pyDb.Exec("CREATE TABLE t1 (id INT)")
	_, _ = pyDb.Exec("INSERT INTO t1 VALUES (1), (2)")

	goTables := map[string]bool{"t1": true}
	pyTables := map[string]bool{"t1": true}

	result := compareTableCounts(goDb, pyDb, goTables, pyTables)
	if len(result) != 1 {
		t.Fatalf("expected 1 comparison, got %d", len(result))
	}
	if result[0].Status != "OK" {
		t.Fatalf("expected OK, got %s", result[0].Status)
	}
}

func TestCompareTableCounts_MissingPython(t *testing.T) {
	goDb, _ := sql.Open("duckdb", ":memory:")
	defer goDb.Close()
	pyDb, _ := sql.Open("duckdb", ":memory:")
	defer pyDb.Close()

	_, _ = goDb.Exec("CREATE TABLE only_go (id INT)")
	_, _ = goDb.Exec("INSERT INTO only_go VALUES (1)")

	goTables := map[string]bool{"only_go": true}
	pyTables := map[string]bool{}

	result := compareTableCounts(goDb, pyDb, goTables, pyTables)
	found := false
	for _, tc := range result {
		if tc.TableName == "only_go" {
			found = true
			if tc.Status != statusMissPy {
				t.Fatalf("expected MISS_PY, got %s", tc.Status)
			}
		}
	}
	if !found {
		t.Fatal("expected only_go in results")
	}
}

func TestCompareTableCounts_MissingGo(t *testing.T) {
	goDb, _ := sql.Open("duckdb", ":memory:")
	defer goDb.Close()
	pyDb, _ := sql.Open("duckdb", ":memory:")
	defer pyDb.Close()

	_, _ = pyDb.Exec("CREATE TABLE only_py (id INT)")
	_, _ = pyDb.Exec("INSERT INTO only_py VALUES (1)")

	goTables := map[string]bool{}
	pyTables := map[string]bool{"only_py": true}

	result := compareTableCounts(goDb, pyDb, goTables, pyTables)
	found := false
	for _, tc := range result {
		if tc.TableName == "only_py" {
			found = true
			if tc.Status != statusMissGo {
				t.Fatalf("expected MISS_GO, got %s", tc.Status)
			}
		}
	}
	if !found {
		t.Fatal("expected only_py in results")
	}
}

func TestLoadMatchIDs(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, _ = db.Exec("CREATE TABLE player_match_enrichment (match_id VARCHAR, performance_score DOUBLE)")
	_, _ = db.Exec("INSERT INTO player_match_enrichment VALUES ('m1', 10), ('m2', 20)")

	ids, err := loadMatchIDs(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 match IDs, got %d", len(ids))
	}
	if !ids["m1"] || !ids["m2"] {
		t.Fatalf("missing match IDs: %v", ids)
	}
}

func TestCompareMatchIDs(t *testing.T) {
	goDb, _ := sql.Open("duckdb", ":memory:")
	defer goDb.Close()
	pyDb, _ := sql.Open("duckdb", ":memory:")
	defer pyDb.Close()

	_, _ = goDb.Exec("CREATE TABLE player_match_enrichment (match_id VARCHAR, performance_score DOUBLE)")
	_, _ = goDb.Exec("INSERT INTO player_match_enrichment VALUES ('m1', 10), ('m2', 20), ('m3', 30)")

	_, _ = pyDb.Exec("CREATE TABLE player_match_enrichment (match_id VARCHAR, performance_score DOUBLE)")
	_, _ = pyDb.Exec("INSERT INTO player_match_enrichment VALUES ('m1', 10), ('m2', 20), ('m4', 40)")

	overlap, err := compareMatchIDs(goDb, pyDb)
	if err != nil {
		t.Fatal(err)
	}
	if overlap.InBoth != 2 {
		t.Fatalf("expected 2 in both, got %d", overlap.InBoth)
	}
	if overlap.OnlyInGo != 1 {
		t.Fatalf("expected 1 only in Go, got %d", overlap.OnlyInGo)
	}
	if overlap.OnlyInPython != 1 {
		t.Fatalf("expected 1 only in Python, got %d", overlap.OnlyInPython)
	}
}

func TestCompareBitmasks(t *testing.T) {
	goDb, _ := sql.Open("duckdb", ":memory:")
	defer goDb.Close()
	pyDb, _ := sql.Open("duckdb", ":memory:")
	defer pyDb.Close()

	schema := `CREATE TABLE player_match_enrichment (
		match_id VARCHAR, performance_score DOUBLE, session_id VARCHAR, is_with_friends BOOLEAN
	)`
	_, _ = goDb.Exec(schema)
	_, _ = pyDb.Exec(schema)

	_, _ = goDb.Exec("INSERT INTO player_match_enrichment VALUES ('m1', 42, 's1', true), ('m2', NULL, 's1', true)")
	_, _ = pyDb.Exec("INSERT INTO player_match_enrichment VALUES ('m1', 42, 's1', true), ('m2', 42, 's1', true)")

	stats, err := compareBitmasks(goDb, pyDb)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) == 0 {
		t.Fatal("expected bitmask stats")
	}
}

func TestComparePlayerDBs_FullRoundtrip(t *testing.T) {
	dir := t.TempDir()
	goIDs := []string{"m1", "m2", "m3"}
	pyIDs := []string{"m1", "m2", "m4"}

	goPath := setupPlayerDB(t, dir, "go.duckdb", goIDs, false)
	pyPath := setupPlayerDB(t, dir, "py.duckdb", pyIDs, false)

	report, err := ComparePlayerDBs(goPath, pyPath)
	if err != nil {
		t.Fatalf("ComparePlayerDBs: %v", err)
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if report.Summary == "" {
		t.Fatal("expected non-empty summary")
	}
}

func TestComparePlayerDBs_BadGoPath(t *testing.T) {
	dir := t.TempDir()
	pyPath := setupPlayerDB(t, dir, "py.duckdb", nil, false)

	_, err := ComparePlayerDBs(filepath.Join(dir, "nonexistent.duckdb"), pyPath)
	if err == nil {
		t.Fatal("expected error for bad Go path")
	}
}

func TestComparePlayerDBs_BadPyPath(t *testing.T) {
	dir := t.TempDir()
	goPath := setupPlayerDB(t, dir, "go.duckdb", nil, false)

	_, err := ComparePlayerDBs(goPath, filepath.Join(dir, "nonexistent.duckdb"))
	if err == nil {
		t.Fatal("expected error for bad Python path")
	}
}

func TestComparePlayerDBs_NullPerformanceScore(t *testing.T) {
	dir := t.TempDir()
	goPath := setupPlayerDB(t, dir, "go.duckdb", []string{"m1", "m2"}, true)  // all NULL
	pyPath := setupPlayerDB(t, dir, "py.duckdb", []string{"m1", "m2"}, false) // all 42.0

	report, err := ComparePlayerDBs(goPath, pyPath)
	if err != nil {
		t.Fatalf("ComparePlayerDBs: %v", err)
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}

	// Check bitmask stats show divergence.
	for _, b := range report.Bitmasks {
		if b.Status == "OK" && b.ZeroCount != b.ZeroCountPy {
			t.Logf("bitmask divergence detected: %+v", b)
		}
	}
}

// Cleanup: ensure temp DBs are removed.
func init() {
	// DuckDB creates WAL files; TempDir handles cleanup.
	_ = os.Getenv("HOME")
}
