//go:build integration

package sync

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openSharedForBackfill(t *testing.T) *sql.DB {
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
			end_time TIMESTAMPTZ
		);
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			gamertag VARCHAR,
			score INTEGER,
			rank INTEGER,
			kills INTEGER,
			deaths INTEGER,
			assists INTEGER,
			shots_fired INTEGER,
			shots_hit INTEGER,
			damage_dealt DOUBLE,
			damage_taken DOUBLE,
			avg_life_seconds DOUBLE
		);
	`
	if err := execScript(t.Context(), db, ddl); err != nil {
		t.Fatal(err)
	}

	inserts := `
		INSERT INTO match_registry VALUES
			('m1', '2025-01-01 00:00:00', '2025-01-01 00:30:00'),
			('m2', '2025-01-02 00:00:00', '2025-01-02 00:30:00'),
			('m3', '2025-01-03 00:00:00', '2025-01-03 00:30:00');
		INSERT INTO match_participants VALUES
			('m1', 'xuid1', 'Player1', NULL, NULL, 10, 5, 3, NULL, NULL, NULL, NULL, NULL),
			('m2', 'xuid1', 'Player1', 100, 1, NULL, NULL, NULL, 50, 20, 1000.0, 800.0, NULL),
			('m3', 'xuid1', 'Player1', 100, 1, 10, 5, 3, 50, 20, 1000.0, 800.0, 30.0);
	`
	if err := execScript(t.Context(), db, inserts); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestFindMatchesInSharedDB_ParticipantsScores(t *testing.T) {
	db := openSharedForBackfill(t)
	scope := &SyncScope{ParticipantsScores: true}
	matches, err := findMatchesInSharedDB(t.Context(), db, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0] != "m1" {
		t.Fatalf("expected [m1], got %v", matches)
	}
}

func TestFindMatchesInSharedDB_ParticipantsKDA(t *testing.T) {
	db := openSharedForBackfill(t)
	scope := &SyncScope{ParticipantsKDA: true}
	matches, err := findMatchesInSharedDB(t.Context(), db, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0] != "m2" {
		t.Fatalf("expected [m2], got %v", matches)
	}
}

func TestFindMatchesInSharedDB_ParticipantsShots(t *testing.T) {
	db := openSharedForBackfill(t)
	scope := &SyncScope{ParticipantsShots: true}
	matches, err := findMatchesInSharedDB(t.Context(), db, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0] != "m1" {
		t.Fatalf("expected [m1], got %v", matches)
	}
}

func TestFindMatchesInSharedDB_ParticipantsDamage(t *testing.T) {
	db := openSharedForBackfill(t)
	scope := &SyncScope{ParticipantsDamage: true}
	matches, err := findMatchesInSharedDB(t.Context(), db, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0] != "m1" {
		t.Fatalf("expected [m1], got %v", matches)
	}
}

func TestFindMatchesInSharedDB_ParticipantsAvgLife(t *testing.T) {
	db := openSharedForBackfill(t)
	scope := &SyncScope{ParticipantsAvgLife: true}
	matches, err := findMatchesInSharedDB(t.Context(), db, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	// m1 and m2 have NULL avg_life_seconds
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %v", matches)
	}
}

func TestFindMatchesInSharedDB_ForceShots(t *testing.T) {
	db := openSharedForBackfill(t)
	scope := &SyncScope{ParticipantsShots: true, ForceParticipantsShots: true}
	matches, err := findMatchesInSharedDB(t.Context(), db, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	// force → all 3 matches
	if len(matches) != 3 {
		t.Fatalf("expected 3 matches with force, got %v", matches)
	}
}

func TestFindMatchesInSharedDB_NoScope(t *testing.T) {
	db := openSharedForBackfill(t)
	scope := &SyncScope{}
	matches, err := findMatchesInSharedDB(t.Context(), db, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if matches != nil {
		t.Fatalf("expected nil, got %v", matches)
	}
}

func TestFindMatchesInSharedDB_MaxMatches(t *testing.T) {
	db := openSharedForBackfill(t)
	scope := &SyncScope{ParticipantsAvgLife: true, MaxMatches: 1}
	matches, err := findMatchesInSharedDB(t.Context(), db, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match with limit, got %v", matches)
	}
}
