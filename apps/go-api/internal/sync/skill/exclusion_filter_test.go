//go:build integration

package skill

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func openExclusionDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	if err := execScript(t.Context(), db, `
		CREATE TABLE player_match_enrichment (
			match_id VARCHAR PRIMARY KEY,
			is_excluded BOOLEAN DEFAULT FALSE,
			performance_score DOUBLE,
			performance_chain VARCHAR
		);
	`); err != nil {
		t.Fatal(err)
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestLoadExcludedMatchIDs_Empty(t *testing.T) {
	db := openExclusionDB(t)

	result, err := LoadExcludedMatchIDs(t.Context(), db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result must be a non-nil empty map (callers do excluded[id] without nil-check)")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(result))
	}
}

func TestLoadExcludedMatchIDs_OnlyExcludedReturned(t *testing.T) {
	db := openExclusionDB(t)
	if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id, is_excluded) VALUES
		('m_excl_1', TRUE),
		('m_excl_2', TRUE),
		('m_active', FALSE),
		('m_default', NULL)`); err != nil {
		t.Fatal(err)
	}

	result, err := LoadExcludedMatchIDs(t.Context(), db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 excluded ids, got %d (%v)", len(result), result)
	}
	if !result["m_excl_1"] || !result["m_excl_2"] {
		t.Errorf("missing expected excluded ids in %v", result)
	}
	if result["m_active"] || result["m_default"] {
		t.Errorf("unexpected non-excluded ids in %v", result)
	}
}

func TestLoadExcludedMatchIDs_NullIsExcludedTreatedAsFalse(t *testing.T) {
	db := openExclusionDB(t)
	// Une row sans is_excluded explicite : COALESCE doit traiter NULL = FALSE.
	if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id, is_excluded) VALUES ('m1', NULL)`); err != nil {
		t.Fatal(err)
	}
	result, err := LoadExcludedMatchIDs(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}
	if result["m1"] {
		t.Error("NULL is_excluded should not be treated as excluded")
	}
}
