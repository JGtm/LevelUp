//go:build integration

// prestige_telemetry_diag_repo_test.go — agrégation de prestige_telemetry par
// origine du défi (ADR 0020). Round-trip DuckDB : seed d'événements + assertions
// sur les compteurs et taux par source (dont NULL → "unknown").

package duckdb

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/migration"
)

func openTelemetryDiagPlayerDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	db, err := OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite player: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_ = migration.All()
	if err := migration.RunForDB(db.SQLDb(), migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(TargetPlayer): %v", err)
	}
	return db
}

func insertTelemetry(t *testing.T, db *DB, id, eventType string, source any) {
	t.Helper()
	_, err := db.Exec(context.Background(),
		`INSERT INTO prestige_telemetry (id, user_id, event_type, source, created_at)
		 VALUES (?, 'u1', ?, ?, CURRENT_TIMESTAMP)`,
		id, eventType, source)
	if err != nil {
		t.Fatalf("insert telemetry %s: %v", id, err)
	}
}

// TestPrestigeTelemetryDiagRepo_AggregatesBySource : ventile created/rejected/
// completed/expired/abandoned par origine, mappe source NULL → "unknown", et
// calcule les taux (sentinel -1 quand dénominateur nul).
func TestPrestigeTelemetryDiagRepo_AggregatesBySource(t *testing.T) {
	db := openTelemetryDiagPlayerDB(t)
	ctx := context.Background()

	// coach : 2 created, 1 completed, 1 rejected, 1 committed (non ventilé mais compté).
	insertTelemetry(t, db, "t1", "created", "coach")
	insertTelemetry(t, db, "t2", "created", "coach")
	insertTelemetry(t, db, "t3", "completed", "coach")
	insertTelemetry(t, db, "t4", "rejected:too_easy", "coach")
	insertTelemetry(t, db, "t5", "committed", "coach")
	// origine NULL (historique) : 1 created, 1 abandoned → bucket "unknown".
	insertTelemetry(t, db, "t6", "created", nil)
	insertTelemetry(t, db, "t7", "abandoned", nil)
	// pilot_mode : uniquement un rejected (0 created) → completion/abandon = -1.
	insertTelemetry(t, db, "t8", "rejected:too_hard", "pilot_mode")

	repo := NewPrestigeTelemetryDiagRepo(&PlayerDB{Player: db})
	diag, err := repo.GetPrestigeTelemetryDiag(ctx, "JGtm")
	if err != nil {
		t.Fatalf("GetPrestigeTelemetryDiag: %v", err)
	}
	if diag.PlayerSlug != "JGtm" {
		t.Errorf("slug: got %q want JGtm", diag.PlayerSlug)
	}
	if diag.TotalEvents != 8 {
		t.Errorf("total_events: got %d want 8", diag.TotalEvents)
	}

	by := map[string]int{}
	for i, s := range diag.BySource {
		by[s.Source] = i
	}
	for _, want := range []string{"coach", "pilot_mode", "unknown"} {
		if _, ok := by[want]; !ok {
			t.Fatalf("origine %q absente des agrégats: %+v", want, diag.BySource)
		}
	}

	coach := diag.BySource[by["coach"]]
	if coach.Created != 2 || coach.Completed != 1 || coach.Rejected != 1 {
		t.Errorf("coach: %+v", coach)
	}
	if coach.CompletionRate != 0.5 {
		t.Errorf("coach completion: got %v want 0.5", coach.CompletionRate)
	}
	if coach.AcceptanceRate < 0.66 || coach.AcceptanceRate > 0.67 {
		t.Errorf("coach acceptance: got %v want ~0.667", coach.AcceptanceRate)
	}

	unknown := diag.BySource[by["unknown"]]
	if unknown.Created != 1 || unknown.Abandoned != 1 {
		t.Errorf("unknown: %+v", unknown)
	}
	if unknown.AbandonRate != 1.0 {
		t.Errorf("unknown abandon: got %v want 1.0", unknown.AbandonRate)
	}

	pilot := diag.BySource[by["pilot_mode"]]
	if pilot.Created != 0 || pilot.Rejected != 1 {
		t.Errorf("pilot_mode: %+v", pilot)
	}
	if pilot.CompletionRate != -1 || pilot.AbandonRate != -1 {
		t.Errorf("pilot_mode sentinels: completion=%v abandon=%v want -1/-1", pilot.CompletionRate, pilot.AbandonRate)
	}
	if pilot.AcceptanceRate != 0 {
		t.Errorf("pilot_mode acceptance: got %v want 0", pilot.AcceptanceRate)
	}
}
