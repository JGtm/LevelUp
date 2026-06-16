//go:build cgo

// player_streak_test.go — déplacé depuis internal/migration (Phase 1.5 b16).
// create_streak_history_append_only est title-owned : le setup câble le provider
// StepsFor avant RunForDB(TargetPlayer) (combine global progression schema + title step).
package migrations

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// setupStreakTestDB crée une player DB avec (optionnellement) la table streak legacy +
// rows, et câble le provider title-owned pour RunForDB(TargetPlayer).
func setupStreakTestDB(t *testing.T, withLegacyData bool) *sql.DB {
	t.Helper()
	migration.SetTitleStepsProvider(StepsFor)
	dir := t.TempDir()
	db, err := sql.Open("duckdb", filepath.Join(dir, "stats.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if withLegacyData {
		if _, err := db.Exec(`
			CREATE TABLE streak (
				id                VARCHAR PRIMARY KEY,
				user_id           VARCHAR NOT NULL,
				title_slug        VARCHAR NOT NULL,
				type              VARCHAR NOT NULL,
				started_at        TIMESTAMP NOT NULL,
				current_length    INTEGER NOT NULL DEFAULT 0,
				best_length       INTEGER NOT NULL DEFAULT 0,
				last_increment_at TIMESTAMP,
				threshold         DOUBLE,
				shields_used      INTEGER NOT NULL DEFAULT 0,
				shields_available INTEGER NOT NULL DEFAULT 1,
				status            VARCHAR NOT NULL DEFAULT 'active',
				broken_at         TIMESTAMP
			);
		`); err != nil {
			t.Fatalf("create legacy streak: %v", err)
		}
		rows := [][]any{
			{"s1", "u1", "halo_infinite", "daily_play", time.Now().UTC(), 5, 7, "active"},
			{"s2", "u1", "halo_infinite", "weekly_play", time.Now().UTC(), 2, 9, "broken"},
		}
		for _, r := range rows {
			if _, err := db.Exec(`
				INSERT INTO streak (id, user_id, title_slug, type, started_at, current_length, best_length, status)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, r...); err != nil {
				t.Fatalf("insert legacy streak: %v", err)
			}
		}
	}
	return db
}

func TestMigration_StreakAppendOnly_CreatesTableAndView(t *testing.T) {
	db := setupStreakTestDB(t, false)
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}
	has, err := migration.TableExists(db, "streak_history")
	if err != nil || !has {
		t.Fatalf("streak_history non créée: err=%v", err)
	}
	var viewName string
	if err := db.QueryRow(`
		SELECT table_name FROM information_schema.tables
		WHERE table_name = 'streak_latest' AND table_type = 'VIEW'
	`).Scan(&viewName); err != nil {
		t.Fatalf("vue streak_latest non créée: %v", err)
	}
}

func TestMigration_StreakAppendOnly_BackfillsExistingData(t *testing.T) {
	db := setupStreakTestDB(t, true)
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM streak_history`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("backfill: count=%d, want 2", count)
	}
	var length int
	var status string
	if err := db.QueryRow(`
		SELECT current_length, status FROM streak_latest WHERE id = 's1'
	`).Scan(&length, &status); err != nil {
		t.Fatalf("backfill content: %v", err)
	}
	if length != 5 || status != "active" {
		t.Errorf("backfill data altérée: length=%d status=%s", length, status)
	}
}

// TestMigration_StreakAppendOnly_LatestViewTieBreak : 2 versions même id + même
// written_at → la vue retourne le tech_id le plus grand (dernier insert).
func TestMigration_StreakAppendOnly_LatestViewTieBreak(t *testing.T) {
	db := setupStreakTestDB(t, false)
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}
	ctx := context.Background()
	wat := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	for _, length := range []int{1, 2} {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO streak_history (id, user_id, title_slug, type, started_at, current_length, status, written_at)
			VALUES ('sx', 'u', 'halo_infinite', 'daily_play', ?, ?, 'active', ?)
		`, wat, length, wat); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	var got int
	if err := db.QueryRowContext(ctx, `SELECT current_length FROM streak_latest WHERE id = 'sx'`).Scan(&got); err != nil {
		t.Fatalf("query view: %v", err)
	}
	if got != 2 {
		t.Errorf("tie-break: current_length=%d, want 2 (tech_id le plus grand)", got)
	}
}

func TestMigration_StreakAppendOnly_Idempotent(t *testing.T) {
	db := setupStreakTestDB(t, true)
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("1st RunForDB: %v", err)
	}
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("2nd RunForDB: %v", err)
	}
	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM streak_history`).Scan(&count)
	if count != 2 {
		t.Errorf("idempotence: count=%d (want 2, double backfill ?)", count)
	}
}
