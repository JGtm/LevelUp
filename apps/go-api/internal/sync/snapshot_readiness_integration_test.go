//go:build integration

package sync

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func openSnapMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func snapExec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}

// TestEvaluateSnapshotReadiness_integration : dataset hétérogène (complet /
// transitoire perf-NULL / FFA) sur DuckDB :memory:. Prouve l'orchestration,
// l'intégration du prédicat, la propagation de snapshot_ready_at via la vue
// player_match_enrichment_latest (migration append-only RÉELLE), l'INSERT pur
// append-only, et l'idempotence (2e passe = 0).
func TestEvaluateSnapshotReadiness_integration(t *testing.T) {
	ctx := context.Background()
	player := openSnapMemDB(t)
	shared := openSnapMemDB(t)

	// ── Player DB : PME append-only (vue _latest avec snapshot_ready_at) + EXISTS-tables.
	snapExec(t, player, `CREATE TABLE player_match_enrichment (match_id VARCHAR)`)
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(player); err != nil {
		t.Fatalf("pme append-only: %v", err)
	}
	snapExec(t, player, `CREATE TABLE match_citations_latest (match_id VARCHAR)`)
	snapExec(t, player, `CREATE TABLE match_skill_rank_latest (match_id VARCHAR, rating_type VARCHAR)`)

	seedPME := func(matchID string, perf sql.NullFloat64) {
		snapExec(t, player, `INSERT INTO player_match_enrichment
			(match_id, performance_score, dominance_flag, psa_checked_at, stage)
			VALUES (?, ?, 0, ?, 'live')`, matchID, perf, time.Now())
	}
	seedPME("m_complete", sql.NullFloat64{Float64: 12, Valid: true})
	seedPME("m_transient", sql.NullFloat64{}) // perf NULL → bloqué
	seedPME("m_ffa", sql.NullFloat64{Float64: 8, Valid: true})

	for _, m := range []string{"m_complete", "m_transient", "m_ffa"} {
		snapExec(t, player, `INSERT INTO match_citations_latest VALUES (?)`, m)
	}
	// LUSR présent pour complete & transient ; m_ffa inéligible (3 teams) → pas besoin.
	snapExec(t, player, `INSERT INTO match_skill_rank_latest VALUES ('m_complete','LUSR')`)
	snapExec(t, player, `INSERT INTO match_skill_rank_latest VALUES ('m_transient','LUSR')`)

	// ── Shared DB.
	snapExec(t, shared, `CREATE TABLE match_registry (
		match_id VARCHAR PRIMARY KEY, start_time TIMESTAMPTZ, start_time_utc TIMESTAMPTZ,
		events_loaded BOOLEAN, backfill_completed BIGINT, is_ranked BOOLEAN,
		is_firefight BOOLEAN, duration_seconds INTEGER)`)
	snapExec(t, shared, `CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, team_id INTEGER)`)

	recent := time.Now().Add(-time.Hour) // dans la fenêtre de grâce
	wk := int64(MBitWeaponKills)
	seedReg := func(matchID string) {
		snapExec(t, shared, `INSERT INTO match_registry VALUES (?, ?, NULL, TRUE, ?, FALSE, FALSE, 600)`,
			matchID, recent, wk)
	}
	seedPart := func(matchID string, teams int) {
		snapExec(t, shared, `INSERT INTO match_participants VALUES (?, 'owner', 0)`, matchID)
		snapExec(t, shared, `INSERT INTO match_participants VALUES (?, 'opp', 1)`, matchID)
		if teams >= 3 {
			snapExec(t, shared, `INSERT INTO match_participants VALUES (?, 'third', 2)`, matchID)
		}
	}
	for _, m := range []string{"m_complete", "m_transient", "m_ffa"} {
		seedReg(m)
	}
	seedPart("m_complete", 2)
	seedPart("m_transient", 2)
	seedPart("m_ffa", 3)

	// ── 1re passe.
	n, err := evaluateSnapshotReadiness(ctx, player, shared, "owner", "halo_infinite")
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if n != 2 {
		t.Fatalf("marqués = %d, attendu 2 (complete + ffa ; transient exclu)", n)
	}

	readyOf := func(matchID string) (sql.NullTime, string) {
		var ts sql.NullTime
		var reasons sql.NullString
		if err := player.QueryRowContext(ctx,
			`SELECT snapshot_ready_at, partial_reasons FROM player_match_enrichment_latest WHERE match_id = ?`,
			matchID).Scan(&ts, &reasons); err != nil {
			t.Fatalf("read latest %s: %v", matchID, err)
		}
		return ts, reasons.String
	}

	if ts, reasons := readyOf("m_complete"); !ts.Valid || reasons != "[]" {
		t.Errorf("m_complete: ready=%v reasons=%q, attendu ready + []", ts.Valid, reasons)
	}
	if ts, _ := readyOf("m_transient"); ts.Valid {
		t.Error("m_transient (perf NULL, dans la grâce) ne doit PAS être ready")
	}
	if ts, reasons := readyOf("m_ffa"); !ts.Valid || !strings.Contains(reasons, "lusr_ineligible") {
		t.Errorf("m_ffa: ready=%v reasons=%q, attendu ready + lusr_ineligible", ts.Valid, reasons)
	}

	// ── 2e passe : idempotent (complete & ffa déjà ready, transient toujours bloqué).
	n2, err := evaluateSnapshotReadiness(ctx, player, shared, "owner", "halo_infinite")
	if err != nil {
		t.Fatalf("evaluate #2: %v", err)
	}
	if n2 != 0 {
		t.Errorf("2e passe = %d, attendu 0 (idempotent)", n2)
	}
}
