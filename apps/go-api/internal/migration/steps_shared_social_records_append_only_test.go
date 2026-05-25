//go:build cgo

package migration

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupRecordsTestDB crée une DB shared_social avec player_records (ancien
// schéma) + N rows de test, puis applique TOUTES les migrations TargetSharedSocial.
func setupRecordsTestDB(t *testing.T, withLegacyData bool) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "shared_social.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if withLegacyData {
		// Pré-créer la table player_records (ancien schéma) avec data.
		_, err := db.Exec(`
			CREATE TABLE player_records (
				xuid              VARCHAR NOT NULL,
				metric            VARCHAR NOT NULL,
				period            VARCHAR NOT NULL DEFAULT 'all_time',
				value             DOUBLE NOT NULL,
				achieved_at       TIMESTAMP,
				achieved_match_id VARCHAR,
				updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (xuid, metric, period)
			);
		`)
		if err != nil {
			t.Fatalf("create legacy table: %v", err)
		}
		// 3 rows test.
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

// TestMigration_RecordsAppendOnly_CreatesTableAndView vérifie que la
// migration crée bien la table + vue + index.
func TestMigration_RecordsAppendOnly_CreatesTableAndView(t *testing.T) {
	db, _ := setupRecordsTestDB(t, false) // pas de data legacy

	if err := RunForDB(db, TargetSharedSocial); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	// Vérifier que player_records_history existe.
	hasHistory, err := tableExists(db, "player_records_history")
	if err != nil || !hasHistory {
		t.Fatalf("player_records_history non créée: err=%v", err)
	}

	// Vérifier que la vue _latest existe.
	var viewName string
	err = db.QueryRow(`
		SELECT table_name FROM information_schema.tables
		WHERE table_name = 'player_records_latest' AND table_type = 'VIEW'
	`).Scan(&viewName)
	if err != nil {
		t.Fatalf("view player_records_latest non créée: %v", err)
	}
}

// TestMigration_RecordsAppendOnly_BackfillsExistingData vérifie que les
// données existantes dans `player_records` sont backfillées dans `_history`.
func TestMigration_RecordsAppendOnly_BackfillsExistingData(t *testing.T) {
	db, _ := setupRecordsTestDB(t, true) // 3 rows legacy

	if err := RunForDB(db, TargetSharedSocial); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM player_records_history`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Errorf("backfill: count=%d, want 3", count)
	}

	// Vérifier valeurs (1 ligne complète).
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

// TestMigration_RecordsAppendOnly_LatestViewReturnsMostRecent vérifie que la
// vue _latest retourne bien la dernière value (par written_at).
func TestMigration_RecordsAppendOnly_LatestViewReturnsMostRecent(t *testing.T) {
	db, _ := setupRecordsTestDB(t, false)

	if err := RunForDB(db, TargetSharedSocial); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	ctx := context.Background()
	// 3 INSERTs sur le MÊME (xuid, metric, period) à 1s d'intervalle simulé.
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

	// La vue _latest doit retourner la 3e valeur (la plus récente).
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

// TestMigration_RecordsAppendOnly_Idempotent vérifie que re-run la migration
// est un no-op (pas de double backfill, pas d'erreur).
func TestMigration_RecordsAppendOnly_Idempotent(t *testing.T) {
	db, _ := setupRecordsTestDB(t, true)

	if err := RunForDB(db, TargetSharedSocial); err != nil {
		t.Fatalf("1st RunForDB: %v", err)
	}
	if err := RunForDB(db, TargetSharedSocial); err != nil {
		t.Fatalf("2nd RunForDB: %v", err)
	}

	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM player_records_history`).Scan(&count)
	if count != 3 {
		t.Errorf("idempotence: count=%d (want 3, double backfill ?)", count)
	}
}
