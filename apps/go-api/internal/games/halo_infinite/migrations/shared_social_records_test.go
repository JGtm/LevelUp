//go:build cgo

// shared_social_records_test.go — déplacé depuis internal/migration (Phase 1.5 b19).
// La famille records (create_player_records_history_append_only + previous_cols + window)
// est title-owned ; le setup câble le provider StepsFor avant RunForDB(TargetSharedSocial)
// (combine la racine globale create_notifications [player_records] + les consommateurs titre).
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

func setupRecordsTestDB(t *testing.T, withLegacyData bool) (*sql.DB, string) {
	t.Helper()
	migration.SetTitleStepsProvider(StepsFor)
	dir := t.TempDir()
	path := filepath.Join(dir, "shared_social.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if withLegacyData {
		_, err := db.Exec(`
			CREATE TABLE player_records (
				xuid              VARCHAR NOT NULL,
				metric            VARCHAR NOT NULL,
				period            VARCHAR NOT NULL DEFAULT 'all_time',
				value             DOUBLE NOT NULL,
				achieved_at       TIMESTAMP,
				achieved_match_id VARCHAR,
				updated_at        TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
				PRIMARY KEY (xuid, metric, period)
			);
		`)
		if err != nil {
			t.Fatalf("create legacy table: %v", err)
		}
		rows := [][]any{
			{"x1", "kda_best", "all_time", 3.5, time.Now().UTC(), "match-1"},
			{"x1", "perfect_kills", "all_time", 42.0, time.Now().UTC(), "match-2"},
			{"x2", "kda_best", "30d", 2.8, time.Now().UTC(), nil},
		}
		for _, r := range rows {
			if _, err := db.Exec(`
				INSERT INTO player_records (xuid, metric, period, value, achieved_at, achieved_match_id)
				VALUES (?, ?, ?, ?, ?, ?)
			`, r...); err != nil {
				t.Fatalf("insert legacy: %v", err)
			}
		}
	}

	return db, path
}

func TestMigration_RecordsAppendOnly_CreatesTableAndView(t *testing.T) {
	db, _ := setupRecordsTestDB(t, false)

	if err := migration.RunForDB(db, migration.TargetSharedSocial); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	hasHistory, err := migration.TableExists(db, "player_records_history")
	if err != nil || !hasHistory {
		t.Fatalf("player_records_history non créée: err=%v", err)
	}

	var viewName string
	err = db.QueryRow(`
		SELECT table_name FROM information_schema.tables
		WHERE table_name = 'player_records_latest' AND table_type = 'VIEW'
	`).Scan(&viewName)
	if err != nil {
		t.Fatalf("view player_records_latest non créée: %v", err)
	}
}

func TestMigration_RecordsAppendOnly_BackfillsExistingData(t *testing.T) {
	db, _ := setupRecordsTestDB(t, true)

	if err := migration.RunForDB(db, migration.TargetSharedSocial); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM player_records_history`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Errorf("backfill: count=%d, want 3", count)
	}

	var value float64
	var matchID sql.NullString
	if err := db.QueryRow(`
		SELECT value, achieved_match_id FROM player_records_history
		WHERE xuid = 'x1' AND metric = 'kda_best'
	`).Scan(&value, &matchID); err != nil {
		t.Fatalf("backfill content: %v", err)
	}
	if value != 3.5 || !matchID.Valid || matchID.String != "match-1" {
		t.Errorf("backfill data altérée: value=%v, match=%v", value, matchID)
	}
}

func TestMigration_RecordsAppendOnly_LatestViewReturnsMostRecent(t *testing.T) {
	db, _ := setupRecordsTestDB(t, false)

	if err := migration.RunForDB(db, migration.TargetSharedSocial); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	ctx := context.Background()
	t0 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	rows := []struct {
		value     float64
		writtenAt time.Time
	}{
		{1.0, t0},
		{2.0, t0.Add(1 * time.Second)},
		{3.0, t0.Add(2 * time.Second)},
	}
	for _, r := range rows {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO player_records_history (xuid, metric, period, value, written_at)
			VALUES (?, ?, ?, ?, ?)
		`, "xtest", "metric_test", "all_time", r.value, r.writtenAt); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	var latestValue float64
	if err := db.QueryRowContext(ctx, `
		SELECT value FROM player_records_latest
		WHERE xuid = 'xtest' AND metric = 'metric_test' AND period = 'all_time'
	`).Scan(&latestValue); err != nil {
		t.Fatalf("query view: %v", err)
	}
	if latestValue != 3.0 {
		t.Errorf("vue _latest: value=%v, want 3.0 (le plus récent)", latestValue)
	}
}

func TestMigration_RecordsAppendOnly_Idempotent(t *testing.T) {
	db, _ := setupRecordsTestDB(t, true)

	if err := migration.RunForDB(db, migration.TargetSharedSocial); err != nil {
		t.Fatalf("1st RunForDB: %v", err)
	}
	if err := migration.RunForDB(db, migration.TargetSharedSocial); err != nil {
		t.Fatalf("2nd RunForDB: %v", err)
	}

	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM player_records_history`).Scan(&count)
	if count != 3 {
		t.Errorf("idempotence: count=%d (want 3, double backfill ?)", count)
	}
}
