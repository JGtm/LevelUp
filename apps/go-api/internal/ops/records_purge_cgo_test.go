// Package ops — records_purge_cgo_test.go : purge des PB corrompus sur DuckDB
// temporaire (driver CGO requis). Vérifie dry-run sans mutation, apply effectif,
// sémantique de la vue _latest (retombée sur dernière version plausible), et
// intégrité (PK/vue reconstruites).
package ops

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openPurgeTestDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), name+".duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func seedPlayerRecordsHistory(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE SEQUENCE player_records_history_id_seq START 1;
		CREATE TABLE player_records_history (
			id                BIGINT PRIMARY KEY DEFAULT nextval('player_records_history_id_seq'),
			xuid              VARCHAR NOT NULL,
			metric            VARCHAR NOT NULL,
			period            VARCHAR NOT NULL DEFAULT 'all_time',
			value             DOUBLE NOT NULL,
			achieved_at       TIMESTAMP,
			achieved_match_id VARCHAR,
			written_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			previous_value    DOUBLE,
			previous_achieved_at TIMESTAMP
		);
		CREATE INDEX idx_prh_lookup ON player_records_history(xuid, metric, period, written_at DESC);
		CREATE OR REPLACE VIEW player_records_latest AS
			SELECT DISTINCT ON (xuid, metric, period)
				id, xuid, metric, period, value, achieved_at, achieved_match_id,
				previous_value, previous_achieved_at, written_at
			FROM player_records_history
			ORDER BY xuid, metric, period, written_at DESC, id DESC;
	`)
	if err != nil {
		t.Fatalf("create player_records_history: %v", err)
	}
	ins := func(xuid, metric, period string, value float64, writtenAt string) {
		if _, err := db.Exec(`INSERT INTO player_records_history (xuid, metric, period, value, written_at)
			VALUES (?, ?, ?, ?, ?)`, xuid, metric, period, value, writtenAt); err != nil {
			t.Fatalf("insert (%s,%s,%s,%v): %v", xuid, metric, period, value, err)
		}
	}
	ins("x1", "accuracy", "all_time", 0.55, "2026-05-01 10:00:00")        // plausible
	ins("x1", "accuracy", "30d", 0.60, "2026-05-01 10:00:00")             // plausible (ancienne version)
	ins("x1", "accuracy", "30d", 73.33, "2026-06-01 10:00:00")            // CORROMPU (latest) -> retirée
	ins("x2", "kda", "all_time", 107, "2026-05-10 10:00:00")              // CORROMPU (seule version) -> clé disparaît
	ins("x2", "best_kda", "all_time", 5.0, "2026-05-10 10:00:00")         // métrique hors catalogue -> retirée
	ins("x2", "performance_score", "all_time", 92, "2026-05-10 10:00:00") // plausible
}

func TestPurgePlayerRecordsHistory_DryRunThenApply(t *testing.T) {
	ctx := context.Background()
	db := openPurgeTestDB(t, "shared_social")
	seedPlayerRecordsHistory(t, db)

	// ── Dry-run : détecte 3 lignes corrompues, ne mute RIEN. ──
	dry, err := PurgePlayerRecordsHistory(ctx, db, false)
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if dry.Before != 6 || dry.Removed != 3 {
		t.Fatalf("dry: Before=%d Removed=%d (attendu 6/3)", dry.Before, dry.Removed)
	}
	if dry.Applied {
		t.Error("dry.Applied devrait être false")
	}
	var afterDry int
	if err := db.QueryRow("SELECT COUNT(*) FROM player_records_history").Scan(&afterDry); err != nil {
		t.Fatalf("count after dry: %v", err)
	}
	if afterDry != 6 {
		t.Errorf("dry-run a muté la table: %d lignes (attendu 6)", afterDry)
	}

	// ── Apply : retire les 3 lignes corrompues. ──
	res, err := PurgePlayerRecordsHistory(ctx, db, true)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Removed != 3 {
		t.Fatalf("apply: Removed=%d (attendu 3)", res.Removed)
	}
	var afterApply int
	if err := db.QueryRow("SELECT COUNT(*) FROM player_records_history").Scan(&afterApply); err != nil {
		t.Fatalf("count after apply: %v", err)
	}
	if afterApply != 3 {
		t.Fatalf("apply: %d lignes restantes (attendu 3)", afterApply)
	}

	// La vue _latest : (x1,accuracy,30d) retombe sur 0.60 (version plausible).
	var v float64
	if err := db.QueryRow(`SELECT value FROM player_records_latest
		WHERE xuid='x1' AND metric='accuracy' AND period='30d'`).Scan(&v); err != nil {
		t.Fatalf("select latest 30d: %v", err)
	}
	if v != 0.60 {
		t.Errorf("latest (x1,accuracy,30d) = %v (attendu 0.60, retombée sur version plausible)", v)
	}
	// La clé entièrement corrompue (x2,kda) a disparu.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM player_records_latest WHERE xuid='x2' AND metric='kda'`).Scan(&n); err != nil {
		t.Fatalf("count kda latest: %v", err)
	}
	if n != 0 {
		t.Errorf("la clé (x2,kda) devrait avoir disparu, %d restant(s)", n)
	}
	// La métrique hors catalogue (best_kda) a disparu.
	if err := db.QueryRow(`SELECT COUNT(*) FROM player_records_history WHERE metric='best_kda'`).Scan(&n); err != nil {
		t.Fatalf("count best_kda: %v", err)
	}
	if n != 0 {
		t.Errorf("best_kda devrait avoir disparu, %d restant(s)", n)
	}

	// Intégrité : PK + séquence fonctionnelles après reconstruction.
	if _, err := db.Exec(`INSERT INTO player_records_history (xuid, metric, period, value)
		VALUES ('x3', 'kda', 'all_time', 3.2)`); err != nil {
		t.Fatalf("insert post-purge (PK/séquence cassée ?): %v", err)
	}

	// Idempotence : re-purge = 0 retirée.
	again, err := PurgePlayerRecordsHistory(ctx, db, true)
	if err != nil {
		t.Fatalf("re-purge: %v", err)
	}
	if again.Removed != 0 {
		t.Errorf("re-purge devrait être un no-op, Removed=%d", again.Removed)
	}
}

func seedRecordHistory(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TABLE record_history (
			id          VARCHAR PRIMARY KEY,
			user_id     VARCHAR NOT NULL,
			title_slug  VARCHAR NOT NULL,
			metric      VARCHAR NOT NULL,
			period      VARCHAR NOT NULL,
			value       DOUBLE NOT NULL,
			achieved_at TIMESTAMP NOT NULL
		);
		CREATE INDEX idx_rec_hist_user_title_metric ON record_history(user_id, title_slug, metric);
		CREATE INDEX idx_rec_hist_achieved_desc ON record_history(user_id, achieved_at DESC);
	`)
	if err != nil {
		t.Fatalf("create record_history: %v", err)
	}
	ins := func(id, metric string, value float64) {
		if _, err := db.Exec(`INSERT INTO record_history (id, user_id, title_slug, metric, period, value, achieved_at)
			VALUES (?, 'x1', 'halo_infinite', ?, 'all_time', ?, '2026-05-01 10:00:00')`, id, metric, value); err != nil {
			t.Fatalf("insert %s: %v", id, err)
		}
	}
	ins("h1", "accuracy", 0.55)  // plausible
	ins("h2", "accuracy", 73.33) // corrompu
	ins("h3", "best_kda", 5.0)   // hors catalogue
	ins("h4", "kda", 4.2)        // plausible
}

func TestPurgeRecordHistory_Apply(t *testing.T) {
	ctx := context.Background()
	db := openPurgeTestDB(t, "stats")
	seedRecordHistory(t, db)

	dry, err := PurgeRecordHistory(ctx, db, false)
	if err != nil {
		t.Fatalf("dry: %v", err)
	}
	if dry.Removed != 2 {
		t.Fatalf("dry Removed=%d (attendu 2)", dry.Removed)
	}

	res, err := PurgeRecordHistory(ctx, db, true)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Removed != 2 {
		t.Fatalf("apply Removed=%d (attendu 2)", res.Removed)
	}
	var ids []string
	rows, err := db.Query("SELECT id FROM record_history ORDER BY id")
	if err != nil {
		t.Fatalf("select ids: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, id)
	}
	if len(ids) != 2 || ids[0] != "h1" || ids[1] != "h4" {
		t.Errorf("lignes restantes = %v (attendu [h1 h4])", ids)
	}
	// PK toujours en place : réinsérer h1 doit échouer (contrainte unique).
	if _, err := db.Exec(`INSERT INTO record_history (id, user_id, title_slug, metric, period, value, achieved_at)
		VALUES ('h1', 'x1', 'halo_infinite', 'kda', 'all_time', 1.0, '2026-05-01 10:00:00')`); err == nil {
		t.Error("PK non reconstruite : doublon h1 accepté")
	}
}

func TestPurgeRecordsTable_MissingTableNoOp(t *testing.T) {
	ctx := context.Background()
	db := openPurgeTestDB(t, "empty")
	res, err := PurgeRecordHistory(ctx, db, true)
	if err != nil {
		t.Fatalf("purge sur table absente: %v", err)
	}
	if res.Removed != 0 || res.Before != 0 {
		t.Errorf("table absente devrait être un no-op, got %+v", res)
	}
}
