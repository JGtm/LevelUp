//go:build cgo

// Package sync — shared_rw_guard_test.go : garde-fou anti-régression du fail-fast
// "shared en read-only". Verrouille la détection qui a manqué pendant l'incident
// 31h (cf. .ai/HANDOFF_sync_combat_completion.md).
package sync

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// newGuardTestDB crée un fichier DuckDB avec match_registry (1 match
// events_loaded=FALSE) et le rouvre dans le mode demandé (RW ou RO).
func newGuardTestDB(t *testing.T, readOnly bool) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "guard_shared.duckdb")

	rw, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open RW: %v", err)
	}
	stmts := []string{
		`CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP,
			events_loaded BOOLEAN DEFAULT FALSE,
			backfill_completed BIGINT DEFAULT 0
		)`,
		`INSERT INTO match_registry (match_id, start_time, events_loaded)
			VALUES ('m-guard-1', TIMESTAMP '2026-05-30 13:00:00', FALSE)`,
	}
	for _, s := range stmts {
		if _, err := rw.ExecContext(context.Background(), s); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	if err := rw.Close(); err != nil {
		t.Fatalf("close RW: %v", err)
	}

	dsn := path
	if readOnly {
		dsn = path + "?access_mode=read_only"
	}
	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestAssertSharedWritable_RW_OK(t *testing.T) {
	db := newGuardTestDB(t, false)
	if err := assertSharedWritable(context.Background(), db); err != nil {
		t.Errorf("assertSharedWritable sur handle RW: %v, want nil", err)
	}
}

func TestAssertSharedWritable_RO_Detected(t *testing.T) {
	db := newGuardTestDB(t, true)
	err := assertSharedWritable(context.Background(), db)
	if !errors.Is(err, ErrSharedReadOnly) {
		t.Errorf("assertSharedWritable sur handle RO: %v, want ErrSharedReadOnly", err)
	}
}

func TestAssertSharedWritable_NilDB_OK(t *testing.T) {
	if err := assertSharedWritable(context.Background(), nil); err != nil {
		t.Errorf("assertSharedWritable(nil): %v, want nil", err)
	}
}

// TestHealEvents_ReadOnlyShared_FailsFast : intégration — sur un shared RO, le
// heal ne doit PAS boucler ni compter no_film, mais remonter ErrSharedReadOnly.
func TestHealEvents_ReadOnlyShared_FailsFast(t *testing.T) {
	db := newGuardTestDB(t, true)
	// client jamais appelé : le fail-fast court-circuite avant tout fetch.
	client := &mockHaloClient{}

	healed, noFilm, err := healEventsForRecentMatches(context.Background(), db, nil, client, 10)
	if !errors.Is(err, ErrSharedReadOnly) {
		t.Fatalf("err=%v, want ErrSharedReadOnly (fail-fast)", err)
	}
	if healed != 0 || noFilm != 0 {
		t.Errorf("healed=%d noFilm=%d, want 0/0 — aucune tentative ne doit être comptée", healed, noFilm)
	}
}
