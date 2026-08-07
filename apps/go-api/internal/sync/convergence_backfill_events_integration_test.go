//go:build integration

// Package sync — convergence_backfill_events_integration_test.go : tests
// d'intégration de RunEventsConvergenceBackfill contre DuckDB in-memory + un
// mock fetcher de chunk film. Couvre le flux de contrôle (détection, 404
// définitif vs récent, erreur réseau, happy-path réel via fixture, idempotence).
package sync

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability"
)

// mockHLChunkFetcher implémente highlightChunkFetcher avec des réponses canned.
type mockHLChunkFetcher struct {
	resp map[string]hlChunkResp
}
type hlChunkResp struct {
	data  []byte
	ver   int
	found bool
	err   error
}

func (m *mockHLChunkFetcher) GetHighlightEventsChunk(_ context.Context, matchID string) ([]byte, int, bool, error) {
	r, ok := m.resp[matchID]
	if !ok {
		return nil, 0, false, nil // défaut : film absent
	}
	return r.data, r.ver, r.found, r.err
}

// eventsBackfillRegistryDDL : DDL nominal de match_registry pour ces tests
// (events_empty inclus — cible de MarkEventsEmptyDefinitive en prod).
const eventsBackfillRegistryDDL = `
		CREATE TABLE match_registry (
			match_id           VARCHAR PRIMARY KEY,
			start_time         TIMESTAMPTZ,
			events_loaded      BOOLEAN DEFAULT FALSE,
			events_empty       BOOLEAN DEFAULT FALSE,
			backfill_completed BIGINT  DEFAULT 0
		);`

// eventsBackfillRegistryDDLMarkFails : même table + une contrainte CHECK qui fait
// ÉCHOUER tout UPDATE de marquage définitif (les deux Mark*Definitive posent
// `backfill_completed |= MBitEvents`). C'est l'injection de défaillance des tests de
// compteur : le persister est construit dans processChunk, il n'est pas injectable —
// la seule façon de le faire échouer est de rendre son écriture invalide côté DB.
const eventsBackfillRegistryDDLMarkFails = `
		CREATE TABLE match_registry (
			match_id           VARCHAR PRIMARY KEY,
			start_time         TIMESTAMPTZ,
			events_loaded      BOOLEAN DEFAULT FALSE,
			events_empty       BOOLEAN DEFAULT FALSE,
			backfill_completed BIGINT  DEFAULT 0 CHECK (backfill_completed = 0)
		);`

// openEventsBackfillDBs ouvre shared (match_registry + match_participants +
// highlight_events + killer_victim_pairs) + player (minimal) in-memory.
func openEventsBackfillDBs(t *testing.T) (playerDB, sharedDB *sql.DB) {
	t.Helper()
	return openEventsBackfillDBsWithRegistry(t, eventsBackfillRegistryDDL)
}

// openEventsBackfillDBsWithRegistry : idem, avec un DDL de match_registry au choix
// (permet d'injecter une table dont le marquage définitif échoue).
func openEventsBackfillDBsWithRegistry(t *testing.T, registryDDL string) (playerDB, sharedDB *sql.DB) {
	t.Helper()
	pdb, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	pdb.SetMaxOpenConns(1)
	t.Cleanup(func() { pdb.Close() })
	// player_match_enrichment : cible des dominance_flag (seed + BatchUpdateColumn).
	if err := execScript(t.Context(), pdb, `
		CREATE TABLE player_match_enrichment (
			match_id       VARCHAR PRIMARY KEY,
			dominance_flag INTEGER,
			updated_at     TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
	`); err != nil {
		t.Fatal(err)
	}
	// Append-only #23046 : convertit player_match_enrichment (id+stage) + vue _latest
	// (backfillDominanceFlagsBatch INSÈRE stage='dominance' ; les readers lisent _latest).
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(pdb); err != nil {
		t.Fatal(err)
	}

	sdb, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	sdb.SetMaxOpenConns(1)
	t.Cleanup(func() { sdb.Close() })
	if err := execScript(t.Context(), sdb, registryDDL+`
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid     VARCHAR,
			team_id  INTEGER,
			outcome  INTEGER
		);
		CREATE TABLE medals_earned (
			match_id      VARCHAR,
			xuid          VARCHAR,
			medal_name_id UBIGINT,
			count         INTEGER
		);
		CREATE SEQUENCE he_seq START 1;
		CREATE TABLE highlight_events (
			id         BIGINT DEFAULT nextval('he_seq') PRIMARY KEY,
			match_id   VARCHAR,
			event_type VARCHAR,
			time_ms    INTEGER,
			xuid       VARCHAR,
			type_hint  INTEGER
		);
		CREATE TABLE killer_victim_pairs (
			match_id        VARCHAR,
			killer_xuid     VARCHAR,
			killer_gamertag VARCHAR,
			victim_xuid     VARCHAR,
			victim_gamertag VARCHAR,
			time_ms         BIGINT,
			created_at      TIMESTAMP
		);
	`); err != nil {
		t.Fatal(err)
	}
	// match_kill_events : seconde destination des couples depuis le 2026-08-02 (double
	// écriture datée). Sans elle la complétion combat échoue et le match est SKIPPÉ —
	// silencieusement, puisqu'un skip n'est pas une erreur.
	if err := migration.EnsureMatchKillEvents(sdb); err != nil {
		t.Fatal(err)
	}
	return pdb, sdb
}

func seedEventsBackfillMatch(t *testing.T, sdb *sql.DB, matchID, xuid string, start time.Time) {
	t.Helper()
	if _, err := sdb.Exec(`INSERT INTO match_registry (match_id, start_time, events_loaded) VALUES (?, ?, FALSE)`, matchID, start); err != nil {
		t.Fatalf("seed registry %s: %v", matchID, err)
	}
	if _, err := sdb.Exec(`INSERT INTO match_participants (match_id, xuid, team_id, outcome) VALUES (?, ?, 0, 2)`, matchID, xuid); err != nil {
		t.Fatalf("seed participant %s: %v", matchID, err)
	}
}

func newEventsBackfillCfg(playerDB, sharedDB *sql.DB, fetcher *mockHLChunkFetcher, xuid string) EventsConvergenceConfig {
	return EventsConvergenceConfig{
		Client:        fetcher,
		AcquireShared: func(_ context.Context) (*sql.DB, func(), error) { return sharedDB, func() {}, nil },
		PlayerDB:      playerDB,
		XUID:          xuid,
		Gamertag:      "TestGT",
		ChunkSize:     5,
		YieldInterval: time.Millisecond, // pas d'attente réelle en test
	}
}

func readEventsLoadedFlag(t *testing.T, sdb *sql.DB, matchID string) bool {
	t.Helper()
	var loaded sql.NullBool
	if err := sdb.QueryRow(`SELECT events_loaded FROM match_registry WHERE match_id = ?`, matchID).Scan(&loaded); err != nil {
		t.Fatalf("read events_loaded %s: %v", matchID, err)
	}
	return loaded.Valid && loaded.Bool
}

// TestEventsConvergence_NoFilm_DefinitiveVsRetry : un match ancien sans film est
// marqué définitif (events_loaded TRUE) ; un match récent sans film est laissé
// pour la passe suivante (events_loaded FALSE).
func TestEventsConvergence_NoFilm_DefinitiveVsRetry(t *testing.T) {
	pdb, sdb := openEventsBackfillDBs(t)
	xuid := "2533274823110022"
	old := time.Now().UTC().Add(-60 * 24 * time.Hour) // > fenêtre 30j → définitif
	recent := time.Now().UTC().Add(-2 * time.Minute)  // récent → retry
	seedEventsBackfillMatch(t, sdb, "m_old", xuid, old)
	seedEventsBackfillMatch(t, sdb, "m_recent", xuid, recent)

	cfg := newEventsBackfillCfg(pdb, sdb, &mockHLChunkFetcher{}, xuid) // tout not-found
	res, err := RunEventsConvergenceBackfill(t.Context(), cfg)
	if err != nil {
		t.Fatalf("RunEventsConvergenceBackfill: %v", err)
	}
	if res.Detected != 2 {
		t.Errorf("Detected: want 2, got %d", res.Detected)
	}
	if res.NoFilmFinal != 1 {
		t.Errorf("NoFilmFinal: want 1 (m_old), got %d", res.NoFilmFinal)
	}
	if !readEventsLoadedFlag(t, sdb, "m_old") {
		t.Error("m_old: events_loaded devrait être TRUE (no-film définitif)")
	}
	if readEventsLoadedFlag(t, sdb, "m_recent") {
		t.Error("m_recent: events_loaded devrait rester FALSE (retry)")
	}
}

// TestEventsConvergence_FetchError_Skipped : une erreur réseau laisse le match
// events_loaded=FALSE (repris la passe suivante), comptée en Skipped — ET COMPTÉE
// AU COMPTEUR. Le `Skipped++` seul est muet : il confond « fetch échoué » avec
// « convergé par un sync parallèle », et une erreur qui se répète à chaque passe
// boucle sans signal. Le compteur est ce qui distingue les deux.
func TestEventsConvergence_FetchError_Skipped(t *testing.T) {
	pdb, sdb := openEventsBackfillDBs(t)
	xuid := "2533274823110022"
	seedEventsBackfillMatch(t, sdb, "m_err", xuid, time.Now().UTC().Add(-10*24*time.Hour))

	ctx := t.Context()
	const counter = "convergence_events_fetch_failed_total"
	before := observability.LoadCounterT(ctxkeys.TitleSlug(ctx), counter)

	fetcher := &mockHLChunkFetcher{resp: map[string]hlChunkResp{
		"m_err": {err: context.DeadlineExceeded},
	}}
	cfg := newEventsBackfillCfg(pdb, sdb, fetcher, xuid)
	res, err := RunEventsConvergenceBackfill(ctx, cfg)
	if err != nil {
		t.Fatalf("RunEventsConvergenceBackfill: %v", err)
	}
	if res.Skipped != 1 {
		t.Errorf("Skipped: want 1, got %d", res.Skipped)
	}
	if readEventsLoadedFlag(t, sdb, "m_err") {
		t.Error("m_err: events_loaded devrait rester FALSE après erreur réseau")
	}
	if got := observability.LoadCounterT(ctxkeys.TitleSlug(ctx), counter) - before; got != 1 {
		t.Errorf("%s: want +1, got +%d — l'erreur de fetch est avalée en silence", counter, got)
	}
}

// TestEventsConvergence_MarkNoFilmFailure_Counted : quand le marquage no-film
// définitif ÉCHOUE, le match reste dans le retry set — il sera re-fetché à chaque
// passe pour un film qu'on sait définitivement absent. Le `Skipped++` seul rendait
// ce trafic réseau perpétuel invisible ; le compteur est ce qui le distingue.
func TestEventsConvergence_MarkNoFilmFailure_Counted(t *testing.T) {
	pdb, sdb := openEventsBackfillDBsWithRegistry(t, eventsBackfillRegistryDDLMarkFails)
	xuid := "2533274823110022"
	seedEventsBackfillMatch(t, sdb, "m_old", xuid, time.Now().UTC().Add(-60*24*time.Hour))

	ctx := t.Context()
	const counter = "convergence_events_mark_nofilm_failed_total"
	before := observability.LoadCounterT(ctxkeys.TitleSlug(ctx), counter)

	cfg := newEventsBackfillCfg(pdb, sdb, &mockHLChunkFetcher{}, xuid) // film absent
	res, err := RunEventsConvergenceBackfill(ctx, cfg)
	if err != nil {
		t.Fatalf("RunEventsConvergenceBackfill: %v", err)
	}
	if res.NoFilmFinal != 0 {
		t.Errorf("NoFilmFinal: want 0 (marquage échoué), got %d", res.NoFilmFinal)
	}
	if res.Skipped != 1 {
		t.Errorf("Skipped: want 1, got %d", res.Skipped)
	}
	if readEventsLoadedFlag(t, sdb, "m_old") {
		t.Error("m_old: events_loaded devrait rester FALSE (marquage échoué)")
	}
	if got := observability.LoadCounterT(ctxkeys.TitleSlug(ctx), counter) - before; got != 1 {
		t.Errorf("%s: want +1, got +%d — l'échec de marquage est avalé en silence", counter, got)
	}
}

// TestEventsConvergence_MarkEventsEmptyFailure_Counted : symétrique, sur le chunk
// récupéré mais vide de tout event. Le marquage events_empty existe pour clore la
// boucle parse_anomaly ; s'il échoue sans compteur, la boucle repart indéfiniment
// sans signal.
func TestEventsConvergence_MarkEventsEmptyFailure_Counted(t *testing.T) {
	pdb, sdb := openEventsBackfillDBsWithRegistry(t, eventsBackfillRegistryDDLMarkFails)
	xuid := "2533274823110022"
	seedEventsBackfillMatch(t, sdb, "m_empty", xuid, time.Now().UTC().Add(-60*24*time.Hour))

	ctx := t.Context()
	const counter = "convergence_events_mark_empty_failed_total"
	before := observability.LoadCounterT(ctxkeys.TitleSlug(ctx), counter)

	// Chunk non-vide qui ne contient aucun xuid → parse OK, 0 event (anomalie légitime).
	fetcher := &mockHLChunkFetcher{resp: map[string]hlChunkResp{
		"m_empty": {data: []byte{0x01, 0x02, 0x03, 0x04}, ver: 41, found: true},
	}}
	cfg := newEventsBackfillCfg(pdb, sdb, fetcher, xuid)
	res, err := RunEventsConvergenceBackfill(ctx, cfg)
	if err != nil {
		t.Fatalf("RunEventsConvergenceBackfill: %v", err)
	}
	if res.NoFilmFinal != 0 {
		t.Errorf("NoFilmFinal: want 0 (marquage échoué), got %d", res.NoFilmFinal)
	}
	if res.Skipped != 1 {
		t.Errorf("Skipped: want 1, got %d", res.Skipped)
	}
	if got := observability.LoadCounterT(ctxkeys.TitleSlug(ctx), counter) - before; got != 1 {
		t.Errorf("%s: want +1, got +%d — l'échec de marquage est avalé en silence", counter, got)
	}
}

// TestEventsConvergence_HappyPath_WritesEventsAndConverges : avec un vrai chunk
// film (fixture v41), les events sont écrits, events_loaded passe TRUE, et la
// passe suivante ne détecte plus rien (idempotence/convergence).
func TestEventsConvergence_HappyPath_WritesEventsAndConverges(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "analysis", "testdata", "v41_chunk_he.bin"))
	if err != nil {
		t.Skipf("fixture film absente: %v", err)
	}
	pdb, sdb := openEventsBackfillDBs(t)
	xuid := "2533274823110022"
	seedEventsBackfillMatch(t, sdb, "m_film", xuid, time.Now().UTC().Add(-5*24*time.Hour))

	fetcher := &mockHLChunkFetcher{resp: map[string]hlChunkResp{
		"m_film": {data: data, ver: 41, found: true},
	}}
	cfg := newEventsBackfillCfg(pdb, sdb, fetcher, xuid)

	res, err := RunEventsConvergenceBackfill(t.Context(), cfg)
	if err != nil {
		t.Fatalf("passe 1: %v", err)
	}
	if res.EventsWritten != 1 {
		t.Fatalf("EventsWritten: want 1, got %d (skipped=%d)", res.EventsWritten, res.Skipped)
	}
	if !readEventsLoadedFlag(t, sdb, "m_film") {
		t.Error("m_film: events_loaded devrait être TRUE")
	}
	var nEvents int
	if err := sdb.QueryRow(`SELECT COUNT(*) FROM highlight_events WHERE match_id = ?`, "m_film").Scan(&nEvents); err != nil {
		t.Fatal(err)
	}
	if nEvents < 100 {
		t.Errorf("highlight_events: want >= 100 (chunk 4v4), got %d", nEvents)
	}

	// Dominance calculée après events (non-slayer, sans steaktacular → flag 0,
	// mais bien CALCULÉ et écrit en player_match_enrichment).
	if res.EventsWritten == 1 && res.DominanceUpdated != 1 {
		t.Errorf("DominanceUpdated: want 1 (calculée après events), got %d", res.DominanceUpdated)
	}
	var domFlag sql.NullInt64
	if err := pdb.QueryRow(`SELECT dominance_flag FROM player_match_enrichment WHERE match_id = ?`, "m_film").Scan(&domFlag); err != nil {
		t.Fatalf("read dominance_flag: %v", err)
	}
	if !domFlag.Valid {
		t.Error("m_film: dominance_flag devrait être calculé (non NULL) après events")
	}

	// Passe 2 : convergence — plus rien à détecter.
	res2, err := RunEventsConvergenceBackfill(t.Context(), cfg)
	if err != nil {
		t.Fatalf("passe 2: %v", err)
	}
	if res2.Detected != 0 {
		t.Errorf("passe 2 Detected: want 0 (convergé), got %d", res2.Detected)
	}
}
