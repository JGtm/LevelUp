//go:build integration

package sync

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/ops"
)

// fakeLiveReader : SharedReader live de repli, retourne une DB sentinelle.
type fakeLiveReader struct {
	db    *sql.DB
	calls int
}

func (f *fakeLiveReader) Get(context.Context) (*sql.DB, func(), error) {
	f.calls++
	return f.db, func() {}, nil
}

// TestSnapshotPreferredSharedReader_FallbackLive : sans snapshot, Get sert le live.
func TestSnapshotPreferredSharedReader_FallbackLive(t *testing.T) {
	paths := title.NewPathResolver(t.TempDir(), nil)
	live := &fakeLiveReader{db: openSnapMemDB(t)}
	r := NewSnapshotPreferredSharedReader(paths, title.DefaultSlug, live)

	db, release, err := r.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	release()
	if db != live.db || live.calls != 1 {
		t.Fatalf("attendu repli live (calls=%d), got db live=%v", live.calls, db == live.db)
	}
}

// seedReaderSnapshot produit un snapshot complet (schéma shared + 1 match ready) et
// retourne le PathResolver pour le reader.
func seedReaderSnapshot(t *testing.T) *title.PathResolver {
	t.Helper()
	paths := title.NewPathResolver(t.TempDir(), nil)

	shared := openSnapMemDB(t)
	snapExec(t, shared, `CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, start_time TIMESTAMPTZ, start_time_utc TIMESTAMPTZ, is_ranked BOOLEAN, is_firefight BOOLEAN)`)
	snapExec(t, shared, `CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR, kills INTEGER)`)
	snapExec(t, shared, `CREATE TABLE medals_earned (match_id VARCHAR, xuid VARCHAR, medal_id BIGINT)`)
	snapExec(t, shared, `CREATE TABLE highlight_events (match_id VARCHAR, xuid VARCHAR, event_type VARCHAR)`)
	snapExec(t, shared, `CREATE TABLE killer_victim_pairs (match_id VARCHAR, killer_xuid VARCHAR, killer_gamertag VARCHAR, victim_xuid VARCHAR, victim_gamertag VARCHAR)`)
	snapExec(t, shared, `CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`)
	snapExec(t, shared, `CREATE TABLE weapon_kills (match_id VARCHAR, xuid VARCHAR, weapon_id BIGINT, effective_weapon_id BIGINT, kills INTEGER)`)
	snapExec(t, shared, `CREATE TABLE match_csrs (match_id VARCHAR, xuid VARCHAR, rating_value FLOAT)`)
	snapExec(t, shared, `CREATE VIEW v_weapon_kills AS SELECT * FROM weapon_kills`)
	snapExec(t, shared, `CREATE VIEW match_csrs_latest AS SELECT * FROM match_csrs`)
	now := time.Now()
	snapExec(t, shared, `INSERT INTO match_registry VALUES ('m1', ?, ?, FALSE, FALSE)`, now, now)
	snapExec(t, shared, `INSERT INTO match_participants VALUES ('m1', 'x1', 'GT1', 7)`)
	snapExec(t, shared, `INSERT INTO killer_victim_pairs VALUES ('m1', 'x1', 'GT1', 'x2', 'GT2')`)
	snapExec(t, shared, `INSERT INTO xuid_aliases VALUES ('x1', 'GT1')`)
	snapExec(t, shared, `INSERT INTO weapon_kills VALUES ('m1', 'x1', 200, 200, 7)`)

	player := openSnapMemDB(t)
	snapExec(t, player, `CREATE TABLE player_match_enrichment (match_id VARCHAR)`)
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(player); err != nil {
		t.Fatalf("pme: %v", err)
	}
	snapExec(t, player, `CREATE TABLE match_citations_latest (match_id VARCHAR)`)
	snapExec(t, player, `CREATE TABLE match_skill_rank_latest (match_id VARCHAR, rating_type VARCHAR)`)
	snapExec(t, player, `INSERT INTO player_match_enrichment (match_id, snapshot_ready_at, partial_reasons, stage) VALUES ('m1', ?, '[]', 'snapshot')`, now)

	res, err := ops.ProduceSnapshot(context.Background(), ops.SnapshotOptions{
		TitleSlug: title.DefaultSlug,
		Paths:     paths,
		Shared:    ops.SharedReadOpenerFunc(func(context.Context) (*sql.DB, func(), error) { return shared, func() {}, nil }),
		PlayerOpener: ops.PlayerReadOpenerFunc(func(_ context.Context, gt string) (*sql.DB, func(), error) {
			return player, func() {}, nil
		}),
		Players: []string{"Solo"},
	})
	if err != nil || !res.Produced {
		t.Fatalf("produce: res=%+v err=%v", res, err)
	}
	return paths
}

// TestSnapshotPreferredSharedReader_ServedAndCached : avec un snapshot, Get sert depuis
// le snapshot (pas le live) et réutilise le querier caché entre deux appels.
func TestSnapshotPreferredSharedReader_ServedAndCached(t *testing.T) {
	ctx := context.Background()
	paths := seedReaderSnapshot(t)
	live := &fakeLiveReader{db: openSnapMemDB(t)}
	r := NewSnapshotPreferredSharedReader(paths, title.DefaultSlug, live)
	defer r.Close()

	db1, rel1, err := r.Get(ctx)
	if err != nil {
		t.Fatalf("Get #1: %v", err)
	}
	rel1()
	if db1 == live.db {
		t.Fatal("Get #1 a servi le live, attendu le snapshot")
	}
	if live.calls != 0 {
		t.Fatalf("live.calls = %d, attendu 0 (servi depuis snapshot)", live.calls)
	}
	// La DB snapshot répond aux requêtes shared (m1 ready présent).
	var n int
	if err := db1.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_registry`).Scan(&n); err != nil {
		t.Fatalf("query snapshot: %v", err)
	}
	if n != 1 {
		t.Errorf("match_registry = %d, attendu 1", n)
	}
	// 2e Get : même querier caché (même *sql.DB).
	db2, rel2, _ := r.Get(ctx)
	rel2()
	if db2 != db1 {
		t.Error("2e Get a reconstruit le querier (cache versionné non réutilisé)")
	}
}
