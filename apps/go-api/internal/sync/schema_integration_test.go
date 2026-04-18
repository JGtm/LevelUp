//go:build integration

package sync

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ── EnsurePlayerSchema ───────────────────────────────────────────────────────

func TestEnsurePlayerSchema_InMemory(t *testing.T) {
	db := openTestDB(t)
	if err := EnsurePlayerSchema(db); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}
	// Verify tables exist
	for _, tbl := range []string{
		"player_match_enrichment",
		"personal_score_awards",
		"sync_meta",
		"career_progression",
		"match_skill_rank",
	} {
		assertTableExists(t, db, tbl)
	}
}

func TestEnsurePlayerSchema_Idempotent(t *testing.T) {
	db := openTestDB(t)
	if err := EnsurePlayerSchema(db); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := EnsurePlayerSchema(db); err != nil {
		t.Fatalf("second call should be idempotent: %v", err)
	}
}

// ── EnsureSharedSchema ───────────────────────────────────────────────────────

func TestEnsureSharedSchema_InMemory(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSharedSchema(db); err != nil {
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	for _, tbl := range []string{
		"match_registry",
		"match_participants",
		"medals_earned",
		"xuid_aliases",
	} {
		assertTableExists(t, db, tbl)
	}
}

func TestEnsureSharedSchema_Idempotent(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSharedSchema(db); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := EnsureSharedSchema(db); err != nil {
		t.Fatalf("second (idempotent): %v", err)
	}
}

// ── execScript ───────────────────────────────────────────────────────────────

func TestExecScript_Simple(t *testing.T) {
	db := openTestDB(t)
	err := execScript(db, "CREATE TABLE t1 (id INT); CREATE TABLE t2 (id INT);")
	if err != nil {
		t.Fatalf("execScript: %v", err)
	}
	assertTableExists(t, db, "t1")
	assertTableExists(t, db, "t2")
}

func TestExecScript_Empty(t *testing.T) {
	db := openTestDB(t)
	if err := execScript(db, ""); err != nil {
		t.Fatalf("execScript empty: %v", err)
	}
}

func TestExecScript_InvalidSQL(t *testing.T) {
	db := openTestDB(t)
	err := execScript(db, "THIS IS NOT SQL;")
	if err == nil {
		t.Fatal("expected error for invalid SQL")
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func assertTableExists(t *testing.T, db *sql.DB, tableName string) {
	t.Helper()
	var cnt int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?",
		tableName,
	).Scan(&cnt)
	if err != nil {
		t.Fatalf("check table %s: %v", tableName, err)
	}
	if cnt == 0 {
		t.Fatalf("table %s not found", tableName)
	}
}

// ── OpenPlayerDB / OpenSharedDB ──────────────────────────────────────────────

func TestOpenPlayerDB_CreatesAndSchemas(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/sub/stats.duckdb"
	db, err := OpenPlayerDB(path)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer db.Close()
	assertTableExists(t, db, "player_match_enrichment")
	assertTableExists(t, db, "sync_meta")
}

func TestOpenSharedDB_CreatesAndSchemas(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/sub/shared.duckdb"
	db, err := OpenSharedDB(path)
	if err != nil {
		t.Fatalf("OpenSharedDB: %v", err)
	}
	defer db.Close()
	assertTableExists(t, db, "match_registry")
	assertTableExists(t, db, "match_participants")
}
