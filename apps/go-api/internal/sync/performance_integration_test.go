//go:build integration

package sync

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openPerfDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ddl := `
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMPTZ,
			pair_name VARCHAR,
			is_ranked BOOLEAN DEFAULT FALSE,
			is_firefight BOOLEAN DEFAULT FALSE
		);
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			outcome INTEGER,
			kills INTEGER, deaths INTEGER, assists INTEGER,
			kda DOUBLE, accuracy DOUBLE,
			time_played_seconds INTEGER,
			personal_score INTEGER, damage_dealt DOUBLE, damage_taken DOUBLE,
			rank INTEGER,
			team_mmr DOUBLE, enemy_mmr DOUBLE,
			kills_expected DOUBLE, deaths_expected DOUBLE
		);
		CREATE TABLE player_match_enrichment (
			match_id VARCHAR PRIMARY KEY,
			performance_score DOUBLE,
			performance_chain VARCHAR,
			updated_at TIMESTAMPTZ
		);
	`
	if err := execScript(db, ddl); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedPerfMatches(t *testing.T, db *sql.DB, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		mid := fmt.Sprintf("m%04d", i)
		ts := fmt.Sprintf("2025-01-%02dT%02d:00:00Z", (i/24)+1, i%24)
		// pair_name "Arena:Slayer" → chaîne arena_slayer pour tous (cohérence test legacy).
		db.Exec(
			"INSERT INTO match_registry (match_id, start_time, pair_name, is_ranked, is_firefight) VALUES (?, ?::TIMESTAMPTZ, ?, ?, ?)",
			mid, ts, "Arena:Slayer", false, false)
		db.Exec(`INSERT INTO match_participants (match_id, xuid, outcome, kills, deaths, assists, kda, accuracy, time_played_seconds, personal_score, damage_dealt, damage_taken, rank, team_mmr, enemy_mmr, kills_expected, deaths_expected) VALUES (?, 'xuid1', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			mid, 2, 10+i, 5, 3, 1.5, 0.5, 600, 1000+i, 2000.0, 500.0, 1, 1500.0, 1500.0, 10.0, 5.0)
	}
}

func TestLoadHistoryForPerf_Empty(t *testing.T) {
	db := openPerfDB(t)
	rows, err := loadHistoryForPerf(db, "xuid_none")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0, got %d", len(rows))
	}
}

func TestLoadHistoryForPerf_WithData(t *testing.T) {
	db := openPerfDB(t)
	seedPerfMatches(t, db, 5)
	rows, err := loadHistoryForPerf(db, "xuid1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5 {
		t.Fatalf("expected 5, got %d", len(rows))
	}
}

func TestBatchComputePerformanceScores_Empty(t *testing.T) {
	db := openPerfDB(t)
	n, err := batchComputePerformanceScores(db, db, "xuid_none", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}
}

func TestBatchComputePerformanceScores_WithData(t *testing.T) {
	db := openPerfDB(t)
	// Need >MinMatchesForRelative matches for any scoring
	seedPerfMatches(t, db, MinMatchesForRelative+10)
	n, err := batchComputePerformanceScores(db, db, "xuid1", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	// Some matches should get scored (those after MinMatchesForRelative)
	if n == 0 {
		t.Fatal("expected some matches to be scored")
	}
}
