//go:build integration

package ops

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
)

// snapRemSetup monte un shared minimal (5 tables de base) + 1 player DB, et retourne
// les openers fakes + les handles pour re-marquer des matchs ready entre deux cuts.
func snapRemSetup(t *testing.T) (*title.PathResolver, *sql.DB, *sql.DB, SnapshotOptions) {
	t.Helper()
	paths := title.NewPathResolver(t.TempDir(), nil)
	shared := snapOpenMem(t)
	seedSharedSchema(t, shared)
	player := seedPlayerDB(t, nil, nil) // PME append-only + tables _latest, aucun ready au départ
	opts := SnapshotOptions{
		TitleSlug:     title.DefaultSlug,
		Paths:         paths,
		Shared:        fakeSharedOpener{db: shared},
		PlayerOpener:  fakePlayerOpener{byGT: map[string]*sql.DB{"Solo": player}},
		Players:       []string{"Solo"},
		RetentionKeep: 2,
	}
	return paths, shared, player, opts
}

// markReady seede un match shared + une row PME stage='snapshot' (snapshot_ready_at).
func markReady(t *testing.T, shared, player *sql.DB, matchID string, readyAt time.Time) {
	t.Helper()
	seedSharedMatch(t, shared, matchID, time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	snapExecT(t, player, `INSERT INTO player_match_enrichment (match_id, snapshot_ready_at, partial_reasons, stage) VALUES (?, ?, '[]', 'snapshot')`, matchID, readyAt)
}

// TestProduceSnapshot_ChangeGate_ReCutOnWatermark : compte ready identique mais
// watermark plus récent (un match re-marqué ready plus tard) → le change-gate NE doit
// PAS dire "unchanged" : un nouveau cut est produit (sinon backlog figé périmé).
func TestProduceSnapshot_ChangeGate_ReCutOnWatermark(t *testing.T) {
	ctx := context.Background()
	_, shared, player, opts := snapRemSetup(t)

	t0 := time.Now().Add(-2 * time.Hour)
	markReady(t, shared, player, "m1", t0)
	opts.Now = time.Now()
	res, err := ProduceSnapshot(ctx, opts)
	if err != nil || !res.Produced || res.Version != 1 {
		t.Fatalf("cut #1 = %+v err=%v, attendu v1 produit", res, err)
	}

	// Re-marque m1 ready PLUS TARD (compte toujours 1, watermark avance).
	snapExecT(t, player, `INSERT INTO player_match_enrichment (match_id, snapshot_ready_at, partial_reasons, stage) VALUES ('m1', ?, '[]', 'snapshot')`, time.Now())
	res2, err := ProduceSnapshot(ctx, opts)
	if err != nil {
		t.Fatalf("cut #2: %v", err)
	}
	if !res2.Produced || res2.Version != 2 {
		t.Fatalf("cut #2 = %+v, attendu v2 produit (watermark avancé, PAS unchanged)", res2)
	}
}

// TestProduceSnapshot_Retention_Purges : RetentionKeep=2 → après 4 versions, seules les
// 2 plus récentes (+ la courante) subsistent ; les anciennes sont purgées.
func TestProduceSnapshot_Retention_Purges(t *testing.T) {
	ctx := context.Background()
	paths, shared, player, opts := snapRemSetup(t)

	for i := 1; i <= 4; i++ {
		markReady(t, shared, player, "m"+string(rune('0'+i)), time.Now().Add(time.Duration(i)*time.Minute))
		opts.Now = time.Now().Add(time.Duration(i) * time.Minute)
		res, err := ProduceSnapshot(ctx, opts)
		if err != nil || !res.Produced {
			t.Fatalf("cut #%d = %+v err=%v", i, res, err)
		}
	}

	// keep=2 → v3,v4 gardées (v4 = courante). v1,v2 purgées.
	exists := func(v int64) bool {
		_, err := os.Stat(paths.SnapshotVersionDir(title.DefaultSlug, v))
		return err == nil
	}
	if exists(1) || exists(2) {
		t.Errorf("v1/v2 devraient être purgées (keep=2)")
	}
	if !exists(3) || !exists(4) {
		t.Errorf("v3/v4 devraient subsister (keep=2 + courante)")
	}
	if v, _ := readCurrent(paths.SnapshotCurrentManifestPath(title.DefaultSlug)); v != 4 {
		t.Errorf("CURRENT = %d, attendu 4", v)
	}
}
