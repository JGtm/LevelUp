//go:build integration

// Package persist — events_completion_persister_test.go : tests d'intégration du
// EventsCompletionPersister sur DuckDB en mémoire.
//
// Build tag `integration` (+ CGO pour le driver duckdb). Lancer avec :
//
//	CGO_ENABLED=1 go test -tags integration ./internal/persist/...
//
// Vérifie : écriture atomique events + killer_victim (par-kill, gamertags +
// time_ms), marquage des bits + events_loaded, idempotence kv (DELETE+INSERT),
// INSERT OR IGNORE highlight_events (pas de DELETE — table ART-indexée), et le
// marquage no-film définitif.
package persist

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// Valeurs des bits passées par le caller (sync.MBitEvents / MBitKillerVictim).
// Dupliquées ici car persist n'importe pas sync ; doivent rester alignées.
const (
	testBitEvents       int64 = 1 << 16 // 65536
	testBitKillerVictim int64 = 1 << 19 // 524288
)

// openCompletionTestDB ouvre une DuckDB en mémoire avec le schéma combat réel :
// highlight_events (PK id seq), killer_victim_pairs (forme par-kill, sans PK),
// match_registry minimal.
func openCompletionTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	schema := []string{
		`CREATE SEQUENCE IF NOT EXISTS highlight_events_id_seq START 1`,
		`CREATE TABLE highlight_events (
			id INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
			match_id VARCHAR, event_type VARCHAR, time_ms INTEGER, xuid VARCHAR, type_hint INTEGER
		)`,
		`CREATE TABLE killer_victim_pairs (
			match_id        VARCHAR NOT NULL,
			killer_xuid     VARCHAR NOT NULL,
			killer_gamertag VARCHAR,
			victim_xuid     VARCHAR NOT NULL,
			victim_gamertag VARCHAR,
			kill_count      INTEGER DEFAULT 1,
			time_ms         INTEGER,
			is_validated    BOOLEAN DEFAULT FALSE,
			created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			backfill_completed BIGINT DEFAULT 0,
			events_loaded BOOLEAN DEFAULT FALSE,
			events_empty BOOLEAN DEFAULT FALSE
		)`,
	}
	for _, ddl := range schema {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			t.Fatalf("create schema: %v", err)
		}
	}
	return db
}

func seedCompletionRegistry(t *testing.T, db *sql.DB, matchID string) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(),
		`INSERT INTO match_registry (match_id) VALUES (?)`, matchID); err != nil {
		t.Fatalf("seed registry: %v", err)
	}
}

func countCompletionRows(t *testing.T, db *sql.DB, table, matchID string) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM "+table+" WHERE match_id = ?", matchID).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func completionRegistryState(t *testing.T, db *sql.DB, matchID string) (bits int64, loaded bool) {
	t.Helper()
	if err := db.QueryRowContext(context.Background(),
		`SELECT backfill_completed, events_loaded FROM match_registry WHERE match_id = ?`, matchID).
		Scan(&bits, &loaded); err != nil {
		t.Fatalf("registry state: %v", err)
	}
	return bits, loaded
}

func sampleCompletionInput(matchID string) EventsCompletionInput {
	return EventsCompletionInput{
		MatchID: matchID,
		Events: []HLEventCompletion{
			{XUID: "111", EventType: "kill", TimeMS: 1000, TypeHint: 50},
			{XUID: "222", EventType: "death", TimeMS: 1002, TypeHint: 1},
		},
		Pairs: []KVPairCompletion{
			{KillerXUID: "111", KillerGamertag: "Killer1", VictimXUID: "222", VictimGamertag: "Victim1", TimeMS: 1000},
		},
		MarkKV:          true,
		EventsBit:       testBitEvents,
		KillerVictimBit: testBitKillerVictim,
	}
}

func TestEventsCompletionPersister_Persist(t *testing.T) {
	ctx := context.Background()
	db := openCompletionTestDB(t)
	matchID := "m-complete-001"
	seedCompletionRegistry(t, db, matchID)

	n, err := NewEventsCompletionPersister(db).Persist(ctx, sampleCompletionInput(matchID))
	if err != nil {
		t.Fatalf("Persist: %v", err)
	}
	if n != 2 {
		t.Errorf("eventsInserted = %d, want 2", n)
	}
	if got := countCompletionRows(t, db, "highlight_events", matchID); got != 2 {
		t.Errorf("highlight_events = %d, want 2", got)
	}
	if got := countCompletionRows(t, db, "killer_victim_pairs", matchID); got != 1 {
		t.Errorf("killer_victim_pairs = %d, want 1", got)
	}

	bits, loaded := completionRegistryState(t, db, matchID)
	if !loaded {
		t.Error("events_loaded = false, want true")
	}
	if bits&testBitEvents == 0 {
		t.Errorf("MBitEvents non positionné: bits=%d", bits)
	}
	if bits&testBitKillerVictim == 0 {
		t.Errorf("MBitKillerVictim non positionné: bits=%d", bits)
	}

	// Vérifie que la forme par-kill (gamertags + time_ms) est bien écrite.
	var killerGT, victimGT string
	var timeMS int64
	if err := db.QueryRowContext(ctx,
		`SELECT killer_gamertag, victim_gamertag, time_ms FROM killer_victim_pairs WHERE match_id = ?`, matchID).
		Scan(&killerGT, &victimGT, &timeMS); err != nil {
		t.Fatalf("kv row: %v", err)
	}
	if killerGT != "Killer1" || victimGT != "Victim1" || timeMS != 1000 {
		t.Errorf("kv row = (%q, %q, %d), want (Killer1, Victim1, 1000)", killerGT, victimGT, timeMS)
	}
}

// L'idempotence killer_victim repose sur DELETE-then-INSERT (table sans index).
// Deux passages → 1 seule paire (pas de doublon).
func TestEventsCompletionPersister_KVIdempotent(t *testing.T) {
	ctx := context.Background()
	db := openCompletionTestDB(t)
	matchID := "m-idem-001"
	seedCompletionRegistry(t, db, matchID)

	p := NewEventsCompletionPersister(db)
	if _, err := p.Persist(ctx, sampleCompletionInput(matchID)); err != nil {
		t.Fatalf("Persist #1: %v", err)
	}
	if _, err := p.Persist(ctx, sampleCompletionInput(matchID)); err != nil {
		t.Fatalf("Persist #2: %v", err)
	}
	if got := countCompletionRows(t, db, "killer_victim_pairs", matchID); got != 1 {
		t.Errorf("killer_victim_pairs après 2 runs = %d, want 1 (idempotent)", got)
	}
}

// MarkKV=true positionne le bit killer_victim même sans paires (parité legacy :
// le bit était marqué dès que le calcul réussissait, y compris 0 paire).
func TestEventsCompletionPersister_EventsOnlyMarksKVBit(t *testing.T) {
	ctx := context.Background()
	db := openCompletionTestDB(t)
	matchID := "m-nopairs-001"
	seedCompletionRegistry(t, db, matchID)

	in := EventsCompletionInput{
		MatchID:         matchID,
		Events:          []HLEventCompletion{{XUID: "111", EventType: "medal", TimeMS: 500, TypeHint: 12}},
		Pairs:           nil,
		MarkKV:          true,
		EventsBit:       testBitEvents,
		KillerVictimBit: testBitKillerVictim,
	}
	n, err := NewEventsCompletionPersister(db).Persist(ctx, in)
	if err != nil {
		t.Fatalf("Persist: %v", err)
	}
	if n != 1 {
		t.Errorf("eventsInserted = %d, want 1", n)
	}
	if got := countCompletionRows(t, db, "killer_victim_pairs", matchID); got != 0 {
		t.Errorf("killer_victim_pairs = %d, want 0", got)
	}
	bits, loaded := completionRegistryState(t, db, matchID)
	if !loaded {
		t.Error("events_loaded = false, want true")
	}
	if bits&testBitEvents == 0 || bits&testBitKillerVictim == 0 {
		t.Errorf("bits = %d, want MBitEvents|MBitKillerVictim", bits)
	}
}

func TestEventsCompletionPersister_MarkNoFilmDefinitive(t *testing.T) {
	ctx := context.Background()
	db := openCompletionTestDB(t)
	matchID := "m-nofilm-001"
	seedCompletionRegistry(t, db, matchID)

	if err := NewEventsCompletionPersister(db).MarkNoFilmDefinitive(ctx, matchID, testBitEvents); err != nil {
		t.Fatalf("MarkNoFilmDefinitive: %v", err)
	}
	if got := countCompletionRows(t, db, "highlight_events", matchID); got != 0 {
		t.Errorf("highlight_events = %d, want 0 (no film)", got)
	}
	bits, loaded := completionRegistryState(t, db, matchID)
	if !loaded {
		t.Error("events_loaded = false, want true")
	}
	if bits&testBitEvents == 0 {
		t.Errorf("MBitEvents non positionné: bits=%d", bits)
	}
}

// TestEventsCompletionPersister_MarkEventsEmptyDefinitive : un chunk récupéré mais
// 0 event légitime pose events_empty=TRUE + le bit events, SANS mentir sur
// events_loaded (qui reste FALSE — aucun event chargé). Sort le match du retry set
// tout en restant honnête et auditable.
func TestEventsCompletionPersister_MarkEventsEmptyDefinitive(t *testing.T) {
	ctx := context.Background()
	db := openCompletionTestDB(t)
	matchID := "m-empty-001"
	seedCompletionRegistry(t, db, matchID)

	if err := NewEventsCompletionPersister(db).MarkEventsEmptyDefinitive(ctx, matchID, testBitEvents); err != nil {
		t.Fatalf("MarkEventsEmptyDefinitive: %v", err)
	}
	if got := countCompletionRows(t, db, "highlight_events", matchID); got != 0 {
		t.Errorf("highlight_events = %d, want 0 (vide)", got)
	}

	var eventsEmpty, eventsLoaded sql.NullBool
	var bits int64
	if err := db.QueryRowContext(ctx,
		`SELECT backfill_completed, events_loaded, events_empty FROM match_registry WHERE match_id = ?`, matchID).
		Scan(&bits, &eventsLoaded, &eventsEmpty); err != nil {
		t.Fatalf("read registry: %v", err)
	}
	if !eventsEmpty.Bool {
		t.Error("events_empty = false, want true (chunk vide définitif)")
	}
	if eventsLoaded.Bool {
		t.Error("events_loaded = true, want false — on ne doit PAS mentir : aucun event chargé")
	}
	if bits&testBitEvents == 0 {
		t.Errorf("MBitEvents non positionné (sortie du retry set): bits=%d", bits)
	}
}

func TestEventsCompletionPersister_EmptyMatchID(t *testing.T) {
	ctx := context.Background()
	db := openCompletionTestDB(t)
	if _, err := NewEventsCompletionPersister(db).Persist(ctx, EventsCompletionInput{}); err == nil {
		t.Error("Persist avec MatchID vide: attendu une erreur, got nil")
	}
}
