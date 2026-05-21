//go:build integration

package sync

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openBackfillTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func seedBackfillShared(t *testing.T, db *sql.DB) {
	t.Helper()
	ddls := []string{
		`CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMPTZ,
			backfill_completed INTEGER DEFAULT 0,
			events_loaded BOOLEAN DEFAULT FALSE
		)`,
		`CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			gamertag VARCHAR,
			backfill_bits INTEGER DEFAULT 0
		)`,
		`CREATE TABLE medals_earned (match_id VARCHAR, xuid VARCHAR, medal_name_id VARCHAR, count INTEGER)`,
	}
	for _, q := range ddls {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("DDL: %v\nSQL: %s", err, q)
		}
	}
	inserts := []string{
		`INSERT INTO match_registry VALUES
			('m1', '2025-01-10 14:00:00+00', 0, FALSE),
			('m2', '2025-01-11 18:00:00+00', 0, FALSE),
			('m3', '2025-01-12 20:00:00+00', 0, TRUE)`,
		`INSERT INTO match_participants VALUES
			('m1', 'xuid001', 'Player1', 0),
			('m2', 'xuid001', 'Player1', 0),
			('m3', 'xuid001', 'Player1', 0)`,
	}
	for _, q := range inserts {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("INSERT: %v\nSQL: %s", err, q)
		}
	}
}

func seedBackfillPlayer(t *testing.T, db *sql.DB) {
	t.Helper()
	ddls := []string{
		`CREATE TABLE player_match_enrichment (match_id VARCHAR PRIMARY KEY, performance_score DOUBLE)`,
		`CREATE TABLE personal_score_awards (id INTEGER, match_id VARCHAR, xuid VARCHAR)`,
		`CREATE TABLE match_skill_rank (match_id VARCHAR PRIMARY KEY)`,
	}
	for _, q := range ddls {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("DDL: %v\nSQL: %s", err, q)
		}
	}
}

// ── getMatchSource ───────────────────────────────────────────────────────────

func TestGetMatchSource_MatchRegistry(t *testing.T) {
	db := openBackfillTestDB(t)
	seedBackfillShared(t, db)

	src := getMatchSource(t.Context(), db)
	if src != "match_registry" {
		t.Fatalf("expected match_registry, got %s", src)
	}
}

func TestGetMatchSource_VMatchFull(t *testing.T) {
	db := openBackfillTestDB(t)
	seedBackfillShared(t, db)
	if _, err := db.Exec(`CREATE VIEW v_match_full AS SELECT * FROM match_registry`); err != nil {
		t.Fatal(err)
	}

	src := getMatchSource(t.Context(), db)
	if src != "v_match_full" {
		t.Fatalf("expected v_match_full, got %s", src)
	}
}

// ── hasBackfillCompletedColumn ───────────────────────────────────────────────

func TestHasBackfillCompletedColumn_True(t *testing.T) {
	db := openBackfillTestDB(t)
	seedBackfillShared(t, db)

	if !hasBackfillCompletedColumn(t.Context(), db) {
		t.Fatal("expected true")
	}
}

func TestHasBackfillCompletedColumn_False(t *testing.T) {
	db := openBackfillTestDB(t)
	if _, err := db.Exec(`CREATE TABLE match_registry (match_id VARCHAR)`); err != nil {
		t.Fatal(err)
	}

	if hasBackfillCompletedColumn(t.Context(), db) {
		t.Fatal("expected false")
	}
}

// ── playerDoneGuard ──────────────────────────────────────────────────────────

func TestPlayerDoneGuard_Empty(t *testing.T) {
	db := openBackfillTestDB(t)
	if _, err := db.Exec(`CREATE TABLE player_match_enrichment (match_id VARCHAR PRIMARY KEY, performance_score DOUBLE)`); err != nil {
		t.Fatal(err)
	}

	guard := playerDoneGuard(t.Context(), db, "player_match_enrichment", "performance_score")
	if guard != "1=1" {
		t.Fatalf("expected 1=1, got %s", guard)
	}
}

func TestPlayerDoneGuard_WithData(t *testing.T) {
	db := openBackfillTestDB(t)
	if _, err := db.Exec(`CREATE TABLE player_match_enrichment (match_id VARCHAR PRIMARY KEY, performance_score DOUBLE)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO player_match_enrichment VALUES ('abcdef01-2345-6789-abcd-ef0123456789', 42.0)`); err != nil {
		t.Fatal(err)
	}

	guard := playerDoneGuard(t.Context(), db, "player_match_enrichment", "performance_score")
	if guard == "1=1" {
		t.Fatal("expected NOT IN clause, got 1=1")
	}
	if !strContains(guard, "NOT IN") {
		t.Fatalf("expected NOT IN, got %s", guard)
	}
}

func TestPlayerDoneGuard_NoColumn(t *testing.T) {
	db := openBackfillTestDB(t)
	if _, err := db.Exec(`CREATE TABLE match_citations (match_id VARCHAR)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO match_citations VALUES ('abcdef01-2345-6789-abcd-ef0123456789')`); err != nil {
		t.Fatal(err)
	}

	guard := playerDoneGuard(t.Context(), db, "match_citations", "")
	if guard == "1=1" {
		t.Fatal("expected NOT IN clause")
	}
}

// ── FindMatchesMissingData ──────────────────────────────────────────────────

func TestFindMatchesMissingData_NilScope(t *testing.T) {
	db := openBackfillTestDB(t)
	_, err := FindMatchesMissingData(t.Context(), db, db, "xuid001", nil)
	if err == nil {
		t.Fatal("expected error for nil scope")
	}
}

func TestFindMatchesMissingData_Events(t *testing.T) {
	sharedDB := openBackfillTestDB(t)
	playerDB := openBackfillTestDB(t)
	seedBackfillShared(t, sharedDB)
	seedBackfillPlayer(t, playerDB)

	scope := &SyncScope{Events: true, DetectionMode: "or"}
	scope.Resolve()

	matches, err := FindMatchesMissingData(t.Context(), playerDB, sharedDB, "xuid001", scope)
	if err != nil {
		t.Fatal(err)
	}
	// m1 and m2 have events_loaded=FALSE, m3 has TRUE
	if len(matches) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(matches), matches)
	}
}

// ── FindMatchesMissingParticipantBits ─────────────────────────────────────

func TestFindMatchesMissingParticipantBits_All(t *testing.T) {
	db := openBackfillTestDB(t)
	seedBackfillShared(t, db)

	matches, err := FindMatchesMissingParticipantBits(t.Context(), db, "xuid001", 1, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	// All 3 matches have backfill_bits=0, so all missing bit 1
	if len(matches) != 3 {
		t.Fatalf("expected 3, got %d", len(matches))
	}
}

func TestFindMatchesMissingParticipantBits_Force(t *testing.T) {
	db := openBackfillTestDB(t)
	seedBackfillShared(t, db)
	// Set bits for m1
	if _, err := db.Exec(`UPDATE match_participants SET backfill_bits=1 WHERE match_id='m1'`); err != nil {
		t.Fatal(err)
	}

	// Force mode should return all
	matches, err := FindMatchesMissingParticipantBits(t.Context(), db, "xuid001", 1, true, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 3 {
		t.Fatalf("force: expected 3, got %d", len(matches))
	}

	// Non-force should skip m1
	matches, err = FindMatchesMissingParticipantBits(t.Context(), db, "xuid001", 1, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("no-force: expected 2, got %d", len(matches))
	}
}

func TestFindMatchesMissingParticipantBits_Limit(t *testing.T) {
	db := openBackfillTestDB(t)
	seedBackfillShared(t, db)

	matches, err := FindMatchesMissingParticipantBits(t.Context(), db, "xuid001", 1, false, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1, got %d", len(matches))
	}
}

func strContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && strContainsHelper(s, sub))
}

func strContainsHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
