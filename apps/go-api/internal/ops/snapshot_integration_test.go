//go:build integration

package ops

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
)

// ── Fakes openers (in-memory) ──────────────────────────────────────────────

type fakeSharedOpener struct{ db *sql.DB }

func (f fakeSharedOpener) OpenSharedRO(context.Context) (*sql.DB, func(), error) {
	return f.db, func() {}, nil
}

type fakePlayerOpener struct{ byGT map[string]*sql.DB }

func (f fakePlayerOpener) OpenPlayerRO(_ context.Context, gt string) (*sql.DB, func(), error) {
	db, ok := f.byGT[gt]
	if !ok {
		return nil, nil, os.ErrNotExist
	}
	return db, func() {}, nil
}

func snapOpenMem(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func snapExecT(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}

// seedSharedFacts crée les 5 faits shared + seed un match (registry + 1 participant +
// 1 médaille + 1 event + 1 paire killer/victim).
func seedSharedSchema(t *testing.T, shared *sql.DB) {
	t.Helper()
	snapExecT(t, shared, `CREATE TABLE match_registry (
		match_id VARCHAR PRIMARY KEY, start_time TIMESTAMPTZ, start_time_utc TIMESTAMPTZ,
		is_ranked BOOLEAN, is_firefight BOOLEAN)`)
	snapExecT(t, shared, `CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, kills INTEGER)`)
	snapExecT(t, shared, `CREATE TABLE medals_earned (match_id VARCHAR, xuid VARCHAR, medal_id BIGINT)`)
	snapExecT(t, shared, `CREATE TABLE highlight_events (match_id VARCHAR, xuid VARCHAR, event_type VARCHAR)`)
	snapExecT(t, shared, `CREATE TABLE killer_victim_pairs (match_id VARCHAR, killer_xuid VARCHAR, victim_xuid VARCHAR)`)
}

func seedSharedMatch(t *testing.T, shared *sql.DB, matchID string, when time.Time) {
	t.Helper()
	snapExecT(t, shared, `INSERT INTO match_registry VALUES (?, ?, ?, FALSE, FALSE)`, matchID, when, when)
	snapExecT(t, shared, `INSERT INTO match_participants VALUES (?, 'xa', 10)`, matchID)
	snapExecT(t, shared, `INSERT INTO medals_earned VALUES (?, 'xa', 42)`, matchID)
	snapExecT(t, shared, `INSERT INTO highlight_events VALUES (?, 'xa', 'kill')`, matchID)
	snapExecT(t, shared, `INSERT INTO killer_victim_pairs VALUES (?, 'xa', 'xb')`, matchID)
}

// seedPlayerDB monte une player DB : PME append-only (vue _latest avec
// snapshot_ready_at) + match_skill_rank_latest + match_citations_latest, et marque
// `ready` les matchs donnés (INSERT stage='snapshot').
func seedPlayerDB(t *testing.T, ready []string, notReady []string) *sql.DB {
	t.Helper()
	db := snapOpenMem(t)
	snapExecT(t, db, `CREATE TABLE player_match_enrichment (match_id VARCHAR)`)
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db); err != nil {
		t.Fatalf("pme append-only: %v", err)
	}
	snapExecT(t, db, `CREATE TABLE match_skill_rank_latest (match_id VARCHAR, rating_type VARCHAR, rating_value DOUBLE)`)
	snapExecT(t, db, `CREATE TABLE match_citations_latest (match_id VARCHAR, citation VARCHAR)`)
	all := append(append([]string{}, ready...), notReady...)
	for _, m := range all {
		snapExecT(t, db, `INSERT INTO player_match_enrichment (match_id, performance_score, stage) VALUES (?, 12.0, 'live')`, m)
		snapExecT(t, db, `INSERT INTO match_skill_rank_latest VALUES (?, 'LUSR', 1500.0)`, m)
		snapExecT(t, db, `INSERT INTO match_citations_latest VALUES (?, 'Carnage')`, m)
	}
	now := time.Now()
	for _, m := range ready {
		snapExecT(t, db, `INSERT INTO player_match_enrichment (match_id, snapshot_ready_at, partial_reasons, stage) VALUES (?, ?, '[]', 'snapshot')`, m, now)
	}
	return db
}

// readParquetCount compte les lignes d'un parquet via une DuckDB :memory: fraîche.
func readParquetCount(t *testing.T, file string) int {
	t.Helper()
	q := snapOpenMem(t)
	var n int
	if err := q.QueryRow(`SELECT COUNT(*) FROM read_parquet(?)`, filepath.ToSlash(file)).Scan(&n); err != nil {
		t.Fatalf("read_parquet count %s: %v", file, err)
	}
	return n
}

func readParquetHasMatch(t *testing.T, file, matchID string) bool {
	t.Helper()
	q := snapOpenMem(t)
	var n int
	if err := q.QueryRow(`SELECT COUNT(*) FROM read_parquet(?) WHERE match_id = ?`, filepath.ToSlash(file), matchID).Scan(&n); err != nil {
		t.Fatalf("read_parquet match %s: %v", file, err)
	}
	return n > 0
}

// TestProduceSnapshot_integration : 2 joueurs, dataset hétérogène (ready / not-ready
// / partagé). Prouve : version 1 produite, CURRENT flippé, faits shared filtrés au set
// ready (m3 exclu), dérivés par joueur, manifest cohérent, et idempotence (2e cut =
// unchanged no-op, pas de v2).
func TestProduceSnapshot_integration(t *testing.T) {
	ctx := context.Background()
	repoRoot := t.TempDir()
	paths := title.NewPathResolver(repoRoot, nil)
	slug := title.DefaultSlug

	shared := snapOpenMem(t)
	seedSharedSchema(t, shared)
	when := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	for _, m := range []string{"m1", "m2", "m3", "m4"} {
		seedSharedMatch(t, shared, m, when)
	}

	// Alpha : m1,m2 ready ; m3 not ready. Bravo : m2 (partagé),m4 ready.
	alpha := seedPlayerDB(t, []string{"m1", "m2"}, []string{"m3"})
	bravo := seedPlayerDB(t, []string{"m2", "m4"}, nil)

	opts := SnapshotOptions{
		TitleSlug:     slug,
		Paths:         paths,
		Shared:        fakeSharedOpener{db: shared},
		PlayerOpener:  fakePlayerOpener{byGT: map[string]*sql.DB{"Alpha": alpha, "Bravo": bravo}},
		Players:       []string{"Alpha", "Bravo"},
		Now:           time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC),
		RetentionKeep: 3,
	}

	res, err := ProduceSnapshot(ctx, opts)
	if err != nil {
		t.Fatalf("ProduceSnapshot: %v", err)
	}
	if !res.Produced || res.Version != 1 {
		t.Fatalf("résultat = %+v, attendu Produced v1", res)
	}
	if res.ReadyMatchCount != 3 { // union {m1,m2,m4}
		t.Fatalf("ReadyMatchCount = %d, attendu 3 (m1,m2,m4 ; m3 exclu)", res.ReadyMatchCount)
	}

	// CURRENT pointe sur v1.
	if v, _ := readCurrent(paths.SnapshotCurrentManifestPath(slug)); v != 1 {
		t.Fatalf("CURRENT = %d, attendu 1", v)
	}
	vdir := paths.SnapshotVersionDir(slug, 1)

	// Faits shared : m3 (not ready) exclu, m1/m2/m4 présents.
	reg := filepath.Join(vdir, "shared", "match_registry.parquet")
	if got := readParquetCount(t, reg); got != 3 {
		t.Errorf("match_registry rows = %d, attendu 3", got)
	}
	if readParquetHasMatch(t, reg, "m3") {
		t.Error("m3 (not ready) ne doit PAS être dans le snapshot shared")
	}
	for _, m := range []string{"m1", "m2", "m4"} {
		if !readParquetHasMatch(t, reg, m) {
			t.Errorf("%s (ready) doit être dans le snapshot shared", m)
		}
	}

	// Le snapshot ne contient QUE des faits shared (pas de dérivé player : la lecture
	// scoped MatchView lit le dérivé sur la player DB live). Aucun répertoire derived/.
	if _, err := os.Stat(filepath.Join(vdir, "derived")); !os.IsNotExist(err) {
		t.Errorf("répertoire derived/ ne doit pas exister (export dérivé retiré)")
	}

	// Manifest cohérent.
	data, err := os.ReadFile(filepath.Join(vdir, "manifest.json"))
	if err != nil {
		t.Fatalf("lecture manifest: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("manifest vide")
	}

	// 2e cut : rien n'a changé → unchanged no-op, pas de v2.
	res2, err := ProduceSnapshot(ctx, opts)
	if err != nil {
		t.Fatalf("ProduceSnapshot #2: %v", err)
	}
	if res2.Produced || res2.NoopReason != "unchanged" {
		t.Fatalf("2e cut = %+v, attendu no-op unchanged", res2)
	}
	if _, err := os.Stat(paths.SnapshotVersionDir(slug, 2)); !os.IsNotExist(err) {
		t.Error("v2 ne doit pas exister (rien n'a changé)")
	}
}

// TestProduceSnapshot_NoReady_integration : aucun match ready → no-op propre, pas de
// version, pas de CURRENT.
func TestProduceSnapshot_NoReady_integration(t *testing.T) {
	ctx := context.Background()
	repoRoot := t.TempDir()
	paths := title.NewPathResolver(repoRoot, nil)
	slug := title.DefaultSlug

	shared := snapOpenMem(t)
	seedSharedSchema(t, shared)
	empty := seedPlayerDB(t, nil, []string{"mX"}) // mX présent mais pas ready

	res, err := ProduceSnapshot(ctx, SnapshotOptions{
		TitleSlug: slug, Paths: paths,
		Shared:       fakeSharedOpener{db: shared},
		PlayerOpener: fakePlayerOpener{byGT: map[string]*sql.DB{"Solo": empty}},
		Players:      []string{"Solo"},
	})
	if err != nil {
		t.Fatalf("ProduceSnapshot: %v", err)
	}
	if res.Produced || res.NoopReason != "no_ready_matches" {
		t.Fatalf("résultat = %+v, attendu no-op no_ready_matches", res)
	}
	if v, _ := readCurrent(paths.SnapshotCurrentManifestPath(slug)); v != 0 {
		t.Errorf("CURRENT = %d, attendu 0 (aucune version)", v)
	}
}
