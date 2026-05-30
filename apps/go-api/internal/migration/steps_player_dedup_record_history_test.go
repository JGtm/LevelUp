//go:build cgo

package migration

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupRecordHistoryTestDB crée une player DB avec la table record_history
// (schéma identique à create_progression_player_schema) + rows fournis.
func setupRecordHistoryTestDB(t *testing.T, rows [][]any) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := sql.Open("duckdb", filepath.Join(dir, "stats.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE record_history (
			id         VARCHAR PRIMARY KEY,
			user_id    VARCHAR NOT NULL,
			title_slug VARCHAR NOT NULL,
			metric     VARCHAR NOT NULL,
			period     VARCHAR NOT NULL,
			value      DOUBLE NOT NULL,
			achieved_at TIMESTAMP NOT NULL
		);
	`); err != nil {
		t.Fatalf("create record_history: %v", err)
	}
	for _, r := range rows {
		if _, err := db.Exec(`
			INSERT INTO record_history (id, user_id, title_slug, metric, period, value, achieved_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, r...); err != nil {
			t.Fatalf("insert row: %v", err)
		}
	}
	return db
}

func recordHistoryCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM record_history`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func TestMigration_DedupRecordHistory_RemovesDuplicates(t *testing.T) {
	at := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	// 2 lignes logiquement identiques (id différent) + 1 unique.
	db := setupRecordHistoryTestDB(t, [][]any{
		{"id-a", "u1", "halo_infinite", "kda", "all_time", 3.5, at},
		{"id-b", "u1", "halo_infinite", "kda", "all_time", 3.5, at}, // doublon de id-a
		{"id-c", "u1", "halo_infinite", "kda", "all_time", 4.0, at}, // valeur différente → conservé
	})
	if err := RunForDB(db, TargetPlayer); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}
	if n := recordHistoryCount(t, db); n != 2 {
		t.Errorf("après dédup: count=%d, want 2 (doublon retiré, valeur distincte conservée)", n)
	}
	// La ligne conservée pour la clé dupliquée = id min (id-a).
	var keptID string
	if err := db.QueryRow(`SELECT id FROM record_history WHERE value = 3.5`).Scan(&keptID); err != nil {
		t.Fatalf("query kept: %v", err)
	}
	if keptID != "id-a" {
		t.Errorf("dédup a gardé id=%s, want id-a (min)", keptID)
	}
}

func TestMigration_DedupRecordHistory_NoDups_NoOp(t *testing.T) {
	at := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	db := setupRecordHistoryTestDB(t, [][]any{
		{"id-a", "u1", "halo_infinite", "kda", "all_time", 3.5, at},
		{"id-c", "u1", "halo_infinite", "kda", "all_time", 4.0, at},
	})
	if err := RunForDB(db, TargetPlayer); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}
	if n := recordHistoryCount(t, db); n != 2 {
		t.Errorf("sans doublon: count=%d, want 2 (inchangé)", n)
	}
}

func TestMigration_DedupRecordHistory_Idempotent(t *testing.T) {
	at := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	db := setupRecordHistoryTestDB(t, [][]any{
		{"id-a", "u1", "halo_infinite", "kda", "all_time", 3.5, at},
		{"id-b", "u1", "halo_infinite", "kda", "all_time", 3.5, at},
	})
	if err := RunForDB(db, TargetPlayer); err != nil {
		t.Fatalf("1st RunForDB: %v", err)
	}
	if err := RunForDB(db, TargetPlayer); err != nil {
		t.Fatalf("2nd RunForDB: %v", err)
	}
	if n := recordHistoryCount(t, db); n != 1 {
		t.Errorf("idempotence: count=%d, want 1", n)
	}
}
