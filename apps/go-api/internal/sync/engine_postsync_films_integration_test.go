//go:build integration

// Package sync — engine_postsync_films_integration_test.go : garde-fous du split
// COLLECT/FLUSH des deux étapes film du post-sync (v7.3).
//
// Propriété centrale gardée : le téléchargement/parsing du film ne se fait PLUS
// sous le writer RW shared. L'acquirer est instrumenté — si un fetch (lent)
// observe un writer tenu ou une acquisition RW, le test échoue : c'est
// exactement la régression qui produisait les « writer RW tenu > 2 s » en prod.
//
// Couvre aussi la PARITÉ (les lignes collectées sont bien flushées : events
// écrits, bit weapon posé) et l'ANTI-TOCTOU events (un match convergé par un
// post-sync parallèle entre le collect et le flush n'est pas réécrit).
//
// Tag `integration` : DuckDB (CGO) requis.
package sync

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

// ─────────────────────────────────────────────────────────────────────────────
// Instrumentation : SharedAccess dont l'acquirer RW est observable
// ─────────────────────────────────────────────────────────────────────────────

// burstProbe joue l'acquirer writer + le lecteur RO d'un SharedAccess de test et
// enregistre ce qui est réellement acquis : nombre d'acquisitions RW, labels de
// télémétrie, et si le writer est tenu à un instant donné (held).
type burstProbe struct {
	db       *sql.DB
	held     atomic.Bool
	acquires atomic.Int32
	reads    atomic.Int32
	mu       sync.Mutex
	labels   []string
}

func (p *burstProbe) acquire(ctx context.Context) (*sql.DB, func(), error) {
	p.acquires.Add(1)
	p.mu.Lock()
	p.labels = append(p.labels, ctxkeys.DBWriterLabel(ctx))
	p.mu.Unlock()
	p.held.Store(true)
	return p.db, func() { p.held.Store(false) }, nil
}

func (p *burstProbe) read(_ context.Context) (*sql.DB, func(), error) {
	p.reads.Add(1)
	return p.db, func() {}, nil
}

func (p *burstProbe) seenLabel(want string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, l := range p.labels {
		if l == want {
			return true
		}
	}
	return false
}

// newFilmStepsProbe câble un postSyncFilmSteps sur des DB de test + un
// SharedAccess instrumenté (mode burst, comme le runner V2 en prod).
func newFilmStepsProbe(t *testing.T, playerDB, sharedDB *sql.DB, client HaloClient, xuid string) (postSyncFilmSteps, *burstProbe, *domain.PostSyncResult) {
	t.Helper()
	probe := &burstProbe{db: sharedDB}
	res := &domain.PostSyncResult{}
	steps := postSyncFilmSteps{
		engine:   &SyncEngine{gamertag: "ProbeGT", xuid: xuid, titleSlug: "halo_infinite"},
		playerDB: playerDB,
		shared:   NewBurstSharedAccess(probe.acquire, probe.read, "sync_v2_postsync"),
		client:   client,
		result:   res,
	}
	return steps, probe, res
}

// ─────────────────────────────────────────────────────────────────────────────
// Clients lents instrumentés
// ─────────────────────────────────────────────────────────────────────────────

// slowEventsClient simule un fetch de chunk highlight events LENT et capture
// l'état du writer AU MOMENT du fetch (c'est la mesure du test).
type slowEventsClient struct {
	mockHaloClient
	probe    *burstProbe
	latency  time.Duration
	data     []byte
	version  int
	onFetch  func(matchID string) // injection d'une convergence concurrente (TOCTOU)
	heldSeen atomic.Bool
	acqSeen  atomic.Int32
}

func (c *slowEventsClient) GetHighlightEventsChunk(_ context.Context, matchID string) ([]byte, int, bool, error) {
	if c.probe.held.Load() {
		c.heldSeen.Store(true)
	}
	c.acqSeen.Store(c.probe.acquires.Load())
	if c.onFetch != nil {
		c.onFetch(matchID)
	}
	time.Sleep(c.latency)
	return c.data, c.version, len(c.data) > 0, nil
}

// slowFilmClient simule un download film LENT (l'offenseur historique) et capture
// l'état du writer au moment du download.
type slowFilmClient struct {
	weaponTestClient
	probe    *burstProbe
	latency  time.Duration
	chunks   map[int]FilmChunkData
	heldSeen atomic.Bool
	acqSeen  atomic.Int32
}

func (c *slowFilmClient) GetMatchFilm(_ context.Context, _ string) (map[int]FilmChunkData, bool, error) {
	if c.probe.held.Load() {
		c.heldSeen.Store(true)
	}
	c.acqSeen.Store(c.probe.acquires.Load())
	time.Sleep(c.latency)
	return c.chunks, true, nil
}

// loadHighlightFixture charge le chunk film de référence (v41). Skip si absent.
func loadHighlightFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "analysis", "testdata", "v41_chunk_he.bin"))
	if err != nil {
		t.Skipf("fixture film absente: %v", err)
	}
	return data
}

// ─────────────────────────────────────────────────────────────────────────────
// Events
// ─────────────────────────────────────────────────────────────────────────────

// TestPostSyncEvents_FetchOutsideWriter_AndRowsFlushed : le fetch (lent) du chunk
// film ne voit AUCUN writer RW tenu ni acquis, et les events collectés sont bien
// écrits ensuite dans un burst labellisé sync_v2_postsync/events.
func TestPostSyncEvents_FetchOutsideWriter_AndRowsFlushed(t *testing.T) {
	fixture := loadHighlightFixture(t)
	shared := openBatchPathTestDB(t, migration.TargetShared)
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	const xuid = "x1"
	seedConvergenceMatch(t, shared, "ev-slow", xuid, false, 0)

	client := &slowEventsClient{latency: 60 * time.Millisecond, data: fixture, version: 41}
	steps, probe, res := newFilmStepsProbe(t, player, shared, client, xuid)
	client.probe = probe

	steps.runEventsConvergence(context.Background())

	if client.heldSeen.Load() {
		t.Error("RÉGRESSION : le writer RW était TENU pendant le fetch du chunk film (le fetch doit être hors lease)")
	}
	if got := client.acqSeen.Load(); got != 0 {
		t.Errorf("RÉGRESSION : %d acquisition(s) RW déjà faite(s) au moment du fetch, want 0", got)
	}
	if probe.acquires.Load() != 1 {
		t.Errorf("acquisitions RW = %d, want 1 (un seul burst de flush pour un lot)", probe.acquires.Load())
	}
	if !probe.seenLabel("sync_v2_postsync/events") {
		t.Errorf("label de télémétrie manquant sur le burst de flush, labels=%v", probe.labels)
	}

	// Parité : les lignes collectées sont bien flushées.
	if res.ConvergedEvents != 1 {
		t.Errorf("ConvergedEvents = %d, want 1", res.ConvergedEvents)
	}
	var nEvents int
	if err := shared.QueryRow(`SELECT COUNT(*) FROM highlight_events WHERE match_id = 'ev-slow'`).Scan(&nEvents); err != nil {
		t.Fatalf("count highlight_events: %v", err)
	}
	if nEvents < 100 {
		t.Errorf("highlight_events écrits = %d, want >= 100 (chunk 4v4 flushé)", nEvents)
	}
	var loaded bool
	if err := shared.QueryRow(`SELECT COALESCE(events_loaded, FALSE) FROM match_registry WHERE match_id = 'ev-slow'`).Scan(&loaded); err != nil {
		t.Fatalf("read events_loaded: %v", err)
	}
	if !loaded {
		t.Error("events_loaded devrait être TRUE après flush")
	}
}

// TestPostSyncEvents_AntiTOCTOU_ConvergedDuringCollect : un post-sync parallèle
// (simulé pendant le fetch) converge le match entre le COLLECT et le FLUSH — le
// re-check sous le burst (filterEventsStillMissing) doit empêcher toute écriture,
// sinon les highlight_events seraient dupliqués (INSERT OR IGNORE non-déduplicant
// en prod). C'est l'invariant que le split ne doit JAMAIS perdre.
func TestPostSyncEvents_AntiTOCTOU_ConvergedDuringCollect(t *testing.T) {
	fixture := loadHighlightFixture(t)
	shared := openBatchPathTestDB(t, migration.TargetShared)
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	const xuid = "x1"
	seedConvergenceMatch(t, shared, "ev-toctou", xuid, false, 0)

	client := &slowEventsClient{latency: 10 * time.Millisecond, data: fixture, version: 41}
	client.onFetch = func(matchID string) {
		// Convergence concurrente : le coéquipier a chargé les events pendant que
		// NOUS téléchargions le film.
		if _, err := shared.Exec(
			`UPDATE match_registry SET events_loaded = TRUE WHERE match_id = ?`, matchID); err != nil {
			t.Errorf("simulation convergence parallèle: %v", err)
		}
	}
	steps, probe, res := newFilmStepsProbe(t, player, shared, client, xuid)
	client.probe = probe

	steps.runEventsConvergence(context.Background())

	if res.ConvergedEvents != 0 {
		t.Errorf("ConvergedEvents = %d, want 0 (match déjà convergé → flush skippé)", res.ConvergedEvents)
	}
	var nEvents int
	if err := shared.QueryRow(`SELECT COUNT(*) FROM highlight_events WHERE match_id = 'ev-toctou'`).Scan(&nEvents); err != nil {
		t.Fatalf("count highlight_events: %v", err)
	}
	if nEvents != 0 {
		t.Errorf("ANTI-TOCTOU CASSÉ : %d events écrits alors que le match a été convergé entre collect et flush", nEvents)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Weapons
// ─────────────────────────────────────────────────────────────────────────────

// TestPostSyncWeapons_FilmDownloadOutsideWriter_AndRowsFlushed : le download film
// (lent) ne voit AUCUN writer RW tenu ni acquis ; les lignes corrélées sont
// ensuite écrites dans un burst labellisé sync_v2_postsync/weapons, avec le bit
// de complétion posé (garde bit-honnête).
func TestPostSyncWeapons_FilmDownloadOutsideWriter_AndRowsFlushed(t *testing.T) {
	shared := openBatchPathTestDB(t, migration.TargetShared)
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	const xuid = "x1"
	seedConvergenceMatch(t, shared, "wk-slow", xuid, true, 0)
	for _, ms := range []int{5000, 10000} {
		if _, err := shared.Exec(
			`INSERT INTO highlight_events (match_id, xuid, event_type, time_ms) VALUES (?, ?, 'kill', ?)`,
			"wk-slow", xuid, ms); err != nil {
			t.Fatalf("seed highlight_events: %v", err)
		}
	}

	client := &slowFilmClient{
		latency: 60 * time.Millisecond,
		chunks:  map[int]FilmChunkData{0: {Data: []byte{}, StartMS: 0, DurationMS: 1000}},
	}
	steps, probe, res := newFilmStepsProbe(t, player, shared, client, xuid)
	client.probe = probe

	steps.runWeaponKills(context.Background(), []string{"wk-slow"})

	if client.heldSeen.Load() {
		t.Error("RÉGRESSION : le writer RW était TENU pendant le download film (le download doit être hors lease)")
	}
	if got := client.acqSeen.Load(); got != 0 {
		t.Errorf("RÉGRESSION : %d acquisition(s) RW déjà faite(s) au moment du download, want 0", got)
	}
	if probe.acquires.Load() != 1 {
		t.Errorf("acquisitions RW = %d, want 1 (un seul burst de flush pour un lot)", probe.acquires.Load())
	}
	if !probe.seenLabel("sync_v2_postsync/weapons") {
		t.Errorf("label de télémétrie manquant sur le burst de flush, labels=%v", probe.labels)
	}

	// Parité : lignes flushées + bit posé.
	if res.WeaponKillsProcessed != 1 {
		t.Errorf("WeaponKillsProcessed = %d, want 1", res.WeaponKillsProcessed)
	}
	var nRows int
	if err := shared.QueryRow(
		`SELECT COUNT(*) FROM weapon_kills WHERE match_id = 'wk-slow' AND xuid = ?`, xuid).Scan(&nRows); err != nil {
		t.Fatalf("count weapon_kills: %v", err)
	}
	if nRows != 2 {
		t.Errorf("weapon_kills écrits = %d, want 2 (les 2 kills corrélés)", nRows)
	}
	var bits int64
	if err := shared.QueryRow(
		`SELECT COALESCE(backfill_completed, 0) FROM match_registry WHERE match_id = 'wk-slow'`).Scan(&bits); err != nil {
		t.Fatalf("read backfill_completed: %v", err)
	}
	if bits&int64(MBitWeaponKills) == 0 {
		t.Errorf("MBitWeaponKills non posé après flush (bits=%d)", bits)
	}
}

// TestPostSyncWeapons_NoFilm_MarksNoFilmBitInFlush : film absent → aucune ligne,
// bit no-film posé pendant le flush, compteur noFilm (et pas done).
func TestPostSyncWeapons_NoFilm_MarksNoFilmBitInFlush(t *testing.T) {
	shared := openBatchPathTestDB(t, migration.TargetShared)
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	const xuid = "x1"
	seedConvergenceMatch(t, shared, "wk-nofilm", xuid, true, 0)

	steps, probe, res := newFilmStepsProbe(t, player, shared, &weaponTestClient{filmPresent: false}, xuid)
	_ = probe

	steps.runWeaponKills(context.Background(), []string{"wk-nofilm"})

	if res.WeaponKillsNoFilm != 1 || res.WeaponKillsProcessed != 0 {
		t.Errorf("done=%d noFilm=%d, want done=0 noFilm=1", res.WeaponKillsProcessed, res.WeaponKillsNoFilm)
	}
	var bits int64
	if err := shared.QueryRow(
		`SELECT COALESCE(backfill_completed, 0) FROM match_registry WHERE match_id = 'wk-nofilm'`).Scan(&bits); err != nil {
		t.Fatalf("read backfill_completed: %v", err)
	}
	if bits&int64(MBitWeaponKillsNoFilm) == 0 {
		t.Errorf("MBitWeaponKillsNoFilm non posé (bits=%d)", bits)
	}
}
