//go:build integration

package sync

import (
	"context"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// TestAddPlayerSnapshotBacklog_integration : sur une player DB append-only réelle,
// l'agrégat compte les pending (snapshot_ready_at NULL), les partiels (ready +
// partial_reasons non vide) et l'âge du plus vieux pending (proxy created_at).
func TestAddPlayerSnapshotBacklog_integration(t *testing.T) {
	ctx := context.Background()
	db := openSnapMemDB(t)
	snapExec(t, db, `CREATE TABLE player_match_enrichment (match_id VARCHAR)`)
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db); err != nil {
		t.Fatalf("pme append-only: %v", err)
	}

	old := time.Now().Add(-48 * time.Hour)
	recent := time.Now().Add(-1 * time.Hour)
	// 4 matchs : m1 ready clean, m2 ready partiel, m3/m4 pending (m3 plus vieux).
	seed := func(m string, created time.Time) {
		snapExec(t, db, `INSERT INTO player_match_enrichment (match_id, performance_score, created_at, stage) VALUES (?, 10.0, ?, 'live')`, m, created)
	}
	seed("m1", recent)
	seed("m2", recent)
	seed("m3", old)
	seed("m4", recent)
	now := time.Now()
	snapExec(t, db, `INSERT INTO player_match_enrichment (match_id, snapshot_ready_at, partial_reasons, stage) VALUES ('m1', ?, '[]', 'snapshot')`, now)
	snapExec(t, db, `INSERT INTO player_match_enrichment (match_id, snapshot_ready_at, partial_reasons, stage) VALUES ('m2', ?, '["no_film"]', 'snapshot')`, now)

	var rep SnapshotPendingReport
	addPlayerSnapshotBacklog(ctx, db, now, &rep)

	if rep.PendingTotal != 2 {
		t.Errorf("PendingTotal = %d, attendu 2 (m3,m4)", rep.PendingTotal)
	}
	if rep.PartialTotal != 1 {
		t.Errorf("PartialTotal = %d, attendu 1 (m2)", rep.PartialTotal)
	}
	// Plus vieux pending = m3 (~48h) → âge largement > 24h.
	if rep.OldestPendingAgeSec < 24*3600 {
		t.Errorf("OldestPendingAgeSec = %d, attendu > 86400 (m3 ~48h)", rep.OldestPendingAgeSec)
	}

	// Agrégation : un 2e joueur s'ADDITIONNE (jamais d'écrasement).
	addPlayerSnapshotBacklog(ctx, db, now, &rep)
	if rep.PendingTotal != 4 {
		t.Errorf("PendingTotal après 2e ajout = %d, attendu 4 (somme, pas overwrite)", rep.PendingTotal)
	}
}
