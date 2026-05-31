//go:build cgo

// Package sync — citations_phase4_test.go : Phase 4, recompute citations après
// arrivée des events. isEventsLoaded décide si un match à 0 citation peut
// recevoir le sentinel "_processed" (events présents → oui) ou doit rester
// candidat (events pas encore chargés → non), évitant le piège "citations vides
// en permanence" (le sentinel sortait définitivement le match du pool de
// selectMatchesForCitations).
package sync

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openPhase4SharedDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE match_registry (
		match_id VARCHAR PRIMARY KEY,
		events_loaded BOOLEAN DEFAULT FALSE
	)`); err != nil {
		t.Fatalf("create match_registry: %v", err)
	}
	return db
}

func TestIsEventsLoaded(t *testing.T) {
	ctx := context.Background()
	db := openPhase4SharedDB(t)

	_, _ = db.Exec(`INSERT INTO match_registry (match_id, events_loaded) VALUES ('m-true', TRUE)`)
	_, _ = db.Exec(`INSERT INTO match_registry (match_id, events_loaded) VALUES ('m-false', FALSE)`)
	_, _ = db.Exec(`INSERT INTO match_registry (match_id, events_loaded) VALUES ('m-null', NULL)`)

	if !isEventsLoaded(ctx, db, "m-true") {
		t.Error("events_loaded=TRUE → isEventsLoaded doit retourner true")
	}
	if isEventsLoaded(ctx, db, "m-false") {
		t.Error("events_loaded=FALSE → isEventsLoaded doit retourner false (match reste candidat)")
	}
	if isEventsLoaded(ctx, db, "m-null") {
		t.Error("events_loaded=NULL → isEventsLoaded doit retourner false (events pas confirmés)")
	}
	// Match inconnu : best-effort → true (ne bloque pas le pipeline, comportement legacy).
	if !isEventsLoaded(ctx, db, "m-unknown") {
		t.Error("match inconnu → isEventsLoaded doit retourner true (best-effort)")
	}
	// sharedDB nil → true (best-effort).
	if !isEventsLoaded(ctx, nil, "m-true") {
		t.Error("sharedDB nil → isEventsLoaded doit retourner true (best-effort)")
	}
}
