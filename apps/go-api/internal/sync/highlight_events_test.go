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
			created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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

// ─── InsertKillerVictimPairsFromEvents ────────────────────────────────────────

func TestInsertKillerVictimPairsFromEvents_Empty(t *testing.T) {
	db := openEventsDB(t)
	err := InsertKillerVictimPairsFromEvents(t.Context(), db, "m1", nil)
	if err != nil {
		t.Fatalf("unexpected error for nil events: %v", err)
	}
	var count int
	db.QueryRow("SELECT COUNT(*) FROM killer_victim_pairs").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 pairs, got %d", count)
	}
}

func TestInsertKillerVictimPairsFromEvents_WithKillAndDeath(t *testing.T) {
	db := openEventsDB(t)
	// Un kill de PlayerA au ms=5000, une death de PlayerB au ms=5003 (dans la tolérance).
	events := []analysis.HighlightEvent{
		{XUID: 2_500_000_000_000_001, Gamertag: "PlayerA", EventType: "kill", TimeMS: 5000},
		{XUID: 2_500_000_000_000_002, Gamertag: "PlayerB", EventType: "death", TimeMS: 5003},
	}
	err := InsertKillerVictimPairsFromEvents(t.Context(), db, "m1", events)
	if err != nil {
		t.Fatalf("InsertKillerVictimPairsFromEvents: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM killer_victim_pairs WHERE match_id = 'm1'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 pair, got %d", count)
	}

	var killerXUID, victimXUID string
	db.QueryRow("SELECT killer_xuid, victim_xuid FROM killer_victim_pairs WHERE match_id = 'm1'").
		Scan(&killerXUID, &victimXUID)
	if killerXUID == "" || victimXUID == "" {
		t.Errorf("expected killer/victim XUIDs, got killer=%q victim=%q", killerXUID, victimXUID)
	}
}

func TestInsertKillerVictimPairsFromEvents_OnlyMedals_NoPairs(t *testing.T) {
	db := openEventsDB(t)
	events := []analysis.HighlightEvent{
		{XUID: 2_500_000_000_000_001, Gamertag: "PlayerA", EventType: "medal", TimeMS: 5000, IsMedal: true},
	}
	err := InsertKillerVictimPairsFromEvents(t.Context(), db, "m1", events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var count int
	db.QueryRow("SELECT COUNT(*) FROM killer_victim_pairs").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 pairs for medals-only events, got %d", count)
	}
}

// ─── MarkEventsLoaded ────────────────────────────────────────────────────────

func TestMarkEventsLoaded_SetsFlag(t *testing.T) {
	db := openEventsDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`)

	if err := MarkEventsLoaded(t.Context(), db, "m1"); err != nil {
		t.Fatalf("MarkEventsLoaded: %v", err)
	}

	var loaded bool
	var bits int
	db.QueryRow("SELECT events_loaded, backfill_completed FROM match_registry WHERE match_id = 'm1'").
		Scan(&loaded, &bits)

	if !loaded {
		t.Error("expected events_loaded = TRUE")
	}
	if bits&MBitEvents == 0 {
		t.Errorf("expected MBitEvents bit set in backfill_completed, got %d", bits)
	}
}

func TestMarkKillerVictimLoaded_SetsBit(t *testing.T) {
	db := openEventsDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`)

	if err := MarkKillerVictimLoaded(t.Context(), db, "m1"); err != nil {
		t.Fatalf("MarkKillerVictimLoaded: %v", err)
	}

	var bits int
	db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id = 'm1'").Scan(&bits)
	if bits&MBitKillerVictim == 0 {
		t.Errorf("expected MBitKillerVictim bit set, got %d", bits)
	}
}
