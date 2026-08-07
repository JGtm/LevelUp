// Package sync — events_replay_test.go : tests pour
// FindBrokenHighlightEventMatches et ReplayHighlightEventsForMatches.
//
// CGO_ENABLED=1 requis (duckdb-go) — pas de tag `integration` pour rester
// cohérent avec engine_e2e_test.go (même contrainte). Pas d'appel réseau
// réel : le chunk highlight events est lu depuis le fixture commité
// `internal/analysis/testdata/v41_chunk_he.bin`, servi par le mock client.
package sync

import (
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/migration"
)

// openReplayShared crée une DB DuckDB in-memory avec le schéma minimal exercé
// par events_replay (match_registry + match_participants + highlight_events +
// killer_victim_pairs + match_kill_events + xuid_aliases). Aligné sur le schéma
// prod (cf. internal/migration/steps_shared.go).
func openReplayShared(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ddl := `
		CREATE TABLE match_registry (
			match_id           VARCHAR PRIMARY KEY,
			start_time         TIMESTAMPTZ DEFAULT now(),
			start_time_utc     TIMESTAMP,
			events_loaded      BOOLEAN DEFAULT FALSE,
			backfill_completed INTEGER DEFAULT 0
		);
		CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR
		);
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
		-- killer_victim_pairs RESTE : les producteurs y écrivent toujours (base crédit).
		-- Depuis le 2026-08-03 ce n'est plus elle que la sonde de présence interroge — la
		-- canonique arrive juste en dessous, par les migrations réelles.
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
	`
	if err := execScript(t.Context(), db, ddl); err != nil {
		t.Fatal(err)
	}
	// match_kill_events : seconde destination du kill-feed depuis le 2026-08-02 (double
	// écriture datée). DDL issue des migrations réelles.
	if err := migration.EnsureMatchKillEvents(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func insertReplayMatch(t *testing.T, db *sql.DB, matchID string, eventsLoaded bool, withParticipant bool) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO match_registry (match_id, events_loaded) VALUES (?, ?)`, matchID, eventsLoaded)
	if err != nil {
		t.Fatal(err)
	}
	if withParticipant {
		_, err := db.Exec(`INSERT INTO match_participants (match_id, xuid, gamertag) VALUES (?, 'xuid001', 'PlayerA')`, matchID)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func insertHighlightRow(t *testing.T, db *sql.DB, matchID, eventType string, timeMS int) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) VALUES (?, ?, ?, 'xuid001')`,
		matchID, eventType, timeMS)
	if err != nil {
		t.Fatal(err)
	}
}

func insertKVPair(t *testing.T, db *sql.DB, matchID string) {
	t.Helper()
	// La sonde de présence interroge la CANONIQUE depuis le 2026-08-03 : c'est donc elle
	// qu'un match « sain » doit porter. On passe par la table réelle (la vue _latest la
	// sert), avec les colonnes NOT NULL que le schéma exige.
	_, err := db.Exec(`INSERT INTO match_kill_events
		(match_id, decode_pass, decoder_rev, publishable, time_ms,
		 victim_gamertag, victim_xuid, feed_killer_xuid, feed_present,
		 assist_known, read_path, read_origin)
		VALUES (?, 'p1', 'test', TRUE, 100, 'V', 'v', 'k', TRUE, FALSE, ?, ?)`,
		matchID, killscope.ReadPathLiveFeed, killscope.OriginCreditOnly)
	if err != nil {
		t.Fatal(err)
	}
}

// ─── FindBrokenHighlightEventMatches ─────────────────────────────────────────

func TestFindBrokenHighlightEventMatches_DetectsBothCases(t *testing.T) {
	db := openReplayShared(t)

	// case A : events_loaded=TRUE + 0 highlight_events → DOIT être détecté
	insertReplayMatch(t, db, "broken-a", true, true)

	// case B : events_loaded=TRUE + kills présents + 0 kvp → DOIT être détecté
	insertReplayMatch(t, db, "broken-b", true, true)
	insertHighlightRow(t, db, "broken-b", "kill", 100)
	insertHighlightRow(t, db, "broken-b", "kill", 200)

	// case C : events_loaded=FALSE → ignoré (heal s'en charge)
	insertReplayMatch(t, db, "false-c", false, true)

	// case D : events_loaded=TRUE + highlight_events + kvp présents → sain
	insertReplayMatch(t, db, "ok-d", true, true)
	insertHighlightRow(t, db, "ok-d", "kill", 100)
	insertKVPair(t, db, "ok-d")

	// case E : events_loaded=TRUE + 0 he MAIS sans participant → ignoré
	// (placeholder, pas un vrai match).
	insertReplayMatch(t, db, "noparticipant-e", true, false)

	// case F : events_loaded=TRUE + only 'mode'/'death' events (pas de kill)
	// + 0 kvp → sain (kvp est attendu vide quand il n'y a pas de kill).
	insertReplayMatch(t, db, "modeonly-f", true, true)
	insertHighlightRow(t, db, "modeonly-f", "death", 100)
	insertHighlightRow(t, db, "modeonly-f", "mode", 200)

	got, err := FindBrokenHighlightEventMatches(context.Background(), db, 100)
	if err != nil {
		t.Fatalf("FindBrokenHighlightEventMatches: %v", err)
	}

	want := map[string]bool{"broken-a": true, "broken-b": true}
	if len(got) != len(want) {
		t.Errorf("got %d broken matches, want %d. got=%v", len(got), len(want), got)
	}
	for _, id := range got {
		if !want[id] {
			t.Errorf("unexpected broken match: %s", id)
		}
	}
}

func TestFindBrokenHighlightEventMatches_NilDB(t *testing.T) {
	_, err := FindBrokenHighlightEventMatches(context.Background(), nil, 10)
	if err == nil {
		t.Fatal("expected error on nil db")
	}
}

func TestFindBrokenHighlightEventMatches_EmptyDB_ReturnsNil(t *testing.T) {
	db := openReplayShared(t)
	got, err := FindBrokenHighlightEventMatches(context.Background(), db, 10)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 broken matches, got %d", len(got))
	}
}

// ─── ReplayHighlightEventsForMatches ─────────────────────────────────────────

// loadV41Fixture lit le chunk v41 réel commité dans
// internal/analysis/testdata/v41_chunk_he.bin. Le path est relatif au dossier
// du package sync (les tests Go s'exécutent depuis le dossier du package).
func loadV41Fixture(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "analysis", "testdata", "v41_chunk_he.bin")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read v41 fixture: %v", err)
	}
	return data
}

// makeBenignZlibChunkForReplay produit un chunk zlib non-vide qui décompresse
// en un buffer de zéros — donc 0 events parsés (= parse_anomaly).
func makeBenignZlibChunkForReplay(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(make([]byte, 200)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestReplayHighlightEventsForMatches_HealedNoFilmAnomaly(t *testing.T) {
	db := openReplayShared(t)

	// 3 matchs avec des comportements distincts.
	for _, id := range []string{"healed", "nofilm", "anomaly"} {
		insertReplayMatch(t, db, id, true, true)
	}

	mock := &mockHaloClient{
		highlightChunkByMatch: map[string]highlightChunkResponse{
			"healed":  {data: loadV41Fixture(t), version: 41, found: true},
			"nofilm":  {data: nil, version: 0, found: false},
			"anomaly": {data: makeBenignZlibChunkForReplay(t), version: 41, found: true},
		},
	}

	res, err := ReplayHighlightEventsForMatches(context.Background(),
		mock, db, []string{"healed", "nofilm", "anomaly"}, nil)
	if err != nil {
		t.Fatalf("ReplayHighlightEventsForMatches: %v", err)
	}
	if res.Total != 3 {
		t.Errorf("Total = %d, want 3", res.Total)
	}
	if res.Healed != 1 {
		t.Errorf("Healed = %d, want 1", res.Healed)
	}
	if res.NoFilm != 1 {
		t.Errorf("NoFilm = %d, want 1", res.NoFilm)
	}
	if res.ParseAnomaly != 1 {
		t.Errorf("ParseAnomaly = %d, want 1", res.ParseAnomaly)
	}
	if res.Errors != 0 {
		t.Errorf("Errors = %d, want 0", res.Errors)
	}
	if res.EventsInserted < 100 {
		t.Errorf("EventsInserted = %d, want ≥ 100 (v41 fixture donne ~275)", res.EventsInserted)
	}

	// Vérifier que le bit MBitEvents est positionné pour le healed
	var bf int64
	if err := db.QueryRow(`SELECT backfill_completed FROM match_registry WHERE match_id = 'healed'`).Scan(&bf); err != nil {
		t.Fatal(err)
	}
	if bf&MBitEvents == 0 {
		t.Errorf("healed: MBitEvents non positionné dans backfill_completed (got %d)", bf)
	}

	// Et que pour anomaly, events_loaded a été reset par clearEventsLoaded
	// puis NON re-positionné (n=0 events insérés → pas de mark).
	var loaded bool
	if err := db.QueryRow(`SELECT events_loaded FROM match_registry WHERE match_id = 'anomaly'`).Scan(&loaded); err != nil {
		t.Fatal(err)
	}
	if loaded {
		t.Errorf("anomaly: events_loaded ne devrait PAS être TRUE après parse_anomaly")
	}
}

func TestReplayHighlightEventsForMatches_ProgressCallback(t *testing.T) {
	db := openReplayShared(t)
	for _, id := range []string{"a", "b", "c"} {
		insertReplayMatch(t, db, id, true, true)
	}

	mock := &mockHaloClient{
		highlightChunkByMatch: map[string]highlightChunkResponse{
			"a": {found: false}, "b": {found: false}, "c": {found: false},
		},
	}

	type call struct {
		done, total int
		matchID     string
		status      string
	}
	var (
		mu    sync.Mutex
		calls []call
	)
	progress := func(done, total int, matchID, status string) {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, call{done, total, matchID, status})
	}

	res, err := ReplayHighlightEventsForMatches(context.Background(),
		mock, db, []string{"a", "b", "c"}, progress)
	if err != nil {
		t.Fatal(err)
	}
	if res.NoFilm != 3 {
		t.Errorf("NoFilm = %d, want 3", res.NoFilm)
	}

	if len(calls) != 3 {
		t.Fatalf("progress callback called %d times, want 3", len(calls))
	}
	for i, c := range calls {
		if c.done != i+1 || c.total != 3 {
			t.Errorf("call %d: done=%d total=%d, want done=%d total=3", i, c.done, c.total, i+1)
		}
		if c.status != "no_film" {
			t.Errorf("call %d status = %s, want no_film", i, c.status)
		}
	}
}

func TestReplayHighlightEventsForMatches_ContextCancelled(t *testing.T) {
	db := openReplayShared(t)
	for _, id := range []string{"a", "b", "c", "d"} {
		insertReplayMatch(t, db, id, true, true)
	}

	mock := &mockHaloClient{}
	mock.highlightChunkFound = false

	ctx, cancel := context.WithCancel(context.Background())
	// On annule après le 1er match en utilisant le progressFn.
	progress := func(done, total int, matchID, status string) {
		if done == 1 {
			cancel()
		}
	}

	res, err := ReplayHighlightEventsForMatches(ctx, mock, db,
		[]string{"a", "b", "c", "d"}, progress)
	if err == nil {
		t.Fatal("expected ctx cancelled error, got nil")
	}
	if res.NoFilm < 1 {
		t.Errorf("expected at least 1 NoFilm before cancel, got %d", res.NoFilm)
	}
	if res.Healed+res.NoFilm+res.ParseAnomaly+res.Errors >= 4 {
		t.Errorf("expected to stop early, but processed all 4 matches: %+v", res)
	}
}

func TestReplayHighlightEventsForMatches_NilArgs(t *testing.T) {
	if _, err := ReplayHighlightEventsForMatches(context.Background(), nil, nil, nil, nil); err == nil {
		t.Error("expected error on nil sharedDB + nil client")
	}
	db := openReplayShared(t)
	if _, err := ReplayHighlightEventsForMatches(context.Background(), nil, db, nil, nil); err == nil {
		t.Error("expected error on nil client")
	}
}

// ─── UnionMatchIDs / SortedMatchIDs ──────────────────────────────────────────

func TestUnionMatchIDs_PreservesOrderAndDedupes(t *testing.T) {
	a := []string{"x", "y", "z"}
	b := []string{"y", "w", "x"}
	got := UnionMatchIDs(a, b)
	want := []string{"x", "y", "z", "w"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d. got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("idx %d: got %s, want %s", i, got[i], want[i])
		}
	}
}

func TestUnionMatchIDs_EmptyInputs(t *testing.T) {
	if got := UnionMatchIDs(nil, nil); len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
	a := []string{"x", "y"}
	if got := UnionMatchIDs(a, nil); len(got) != 2 {
		t.Errorf("expected 2 elements, got %v", got)
	}
	if got := UnionMatchIDs(nil, a); len(got) != 2 {
		t.Errorf("expected 2 elements, got %v", got)
	}
}

func TestSortedMatchIDs_StableCopy(t *testing.T) {
	in := []string{"c", "a", "b"}
	out := SortedMatchIDs(in)
	if in[0] != "c" || in[1] != "a" || in[2] != "b" {
		t.Errorf("input mutated: %v", in)
	}
	if out[0] != "a" || out[1] != "b" || out[2] != "c" {
		t.Errorf("not sorted: %v", out)
	}
}

// timeoutAfter est un helper pour cancel ctx après la durée donnée.
// (Présent au cas où ; pas utilisé directement par les tests ci-dessus.)
var _ = func(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
