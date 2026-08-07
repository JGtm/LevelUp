//go:build integration

package sync

import (
	"database/sql"
	"testing"

	"levelup/go-api/internal/analysis"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openEventsDB crée une DB DuckDB in-memory avec le schéma minimal pour les tests.
func openEventsDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ddl := `
		CREATE SEQUENCE highlight_events_id_seq;
		CREATE TABLE highlight_events (
			id         INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
			match_id   VARCHAR NOT NULL,
			event_type VARCHAR NOT NULL,
			time_ms    INTEGER,
			xuid       VARCHAR,
			type_hint  INTEGER,
			raw_json   VARCHAR,
			UNIQUE (match_id, xuid, time_ms, event_type)
		);
		CREATE TABLE killer_victim_pairs (
			match_id        VARCHAR NOT NULL,
			killer_xuid     VARCHAR NOT NULL,
			killer_gamertag VARCHAR,
			victim_xuid     VARCHAR NOT NULL,
			victim_gamertag VARCHAR,
			kill_count      INTEGER DEFAULT 1,
			time_ms         INTEGER,
			is_validated    BOOLEAN DEFAULT FALSE,
			created_at      TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE TABLE xuid_aliases (
			xuid     VARCHAR PRIMARY KEY,
			gamertag VARCHAR
		);
		CREATE TABLE match_registry (
			match_id          VARCHAR PRIMARY KEY,
			backfill_completed INTEGER DEFAULT 0,
			events_loaded      BOOLEAN DEFAULT FALSE
		);
	`
	if err := execScript(t.Context(), db, ddl); err != nil {
		t.Fatal(err)
	}
	return db
}

// ─── InsertHighlightEvents ────────────────────────────────────────────────────

func TestInsertHighlightEvents_Empty(t *testing.T) {
	db := openEventsDB(t)
	n, err := InsertHighlightEvents(t.Context(), db, "m1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 inserted, got %d", n)
	}
}

func TestInsertHighlightEvents_InsertAndCount(t *testing.T) {
	db := openEventsDB(t)
	events := []analysis.HighlightEvent{
		{XUID: 2_500_000_000_000_001, Gamertag: "PlayerA", EventType: "kill", TypeHint: 50, TimeMS: 1000},
		{XUID: 2_500_000_000_000_002, Gamertag: "PlayerB", EventType: "death", TypeHint: 20, TimeMS: 2000},
		{XUID: 2_500_000_000_000_001, Gamertag: "PlayerA", EventType: "medal", TypeHint: 50, TimeMS: 3000, IsMedal: true, MedalType: 100},
	}
	n, err := InsertHighlightEvents(t.Context(), db, "m1", events)
	if err != nil {
		t.Fatalf("InsertHighlightEvents: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 inserted, got %d", n)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM highlight_events WHERE match_id = 'm1'").Scan(&count)
	if count != 3 {
		t.Errorf("expected 3 rows in DB, got %d", count)
	}
}

func TestInsertHighlightEvents_IdempotentOnDuplicate(t *testing.T) {
	db := openEventsDB(t)
	events := []analysis.HighlightEvent{
		{XUID: 2_500_000_000_000_001, Gamertag: "PlayerA", EventType: "kill", TypeHint: 50, TimeMS: 1000},
	}
	n1, err := InsertHighlightEvents(t.Context(), db, "m1", events)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	// Deuxième insert identique → INSERT OR IGNORE sur la contrainte UNIQUE (match_id, xuid, time_ms, event_type).
	// L'event est ignoré (n=0), pas d'erreur.
	n2, err := InsertHighlightEvents(t.Context(), db, "m1", events)
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if n1 != 1 {
		t.Errorf("first insert: expected 1 inserted, got %d", n1)
	}
	if n2 != 0 {
		t.Errorf("second insert (duplicate): expected 0 inserted, got %d", n2)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM highlight_events WHERE match_id = 'm1'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 row in DB after idempotent insert, got %d", count)
	}
}

// NB : MarkEventsLoaded a été supprimée à la décommission des heals (2026-06-01).
// Le marquage events_loaded + MBitEvents est désormais fait atomiquement par
// persist.EventsCompletionPersister (cf. events_completion_persister_test.go).
