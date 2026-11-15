//go:build integration

// player_streak_drop_test.go — drop_streak_legacy_v1 : la table héritée `streak` disparaît des
// bases existantes une fois recopiée dans streak_history, n'est jamais créée sur une base neuve, et
// ne fait plus refuser de réparation de clé au démarrage.
package migrations

import (
	"bytes"
	"database/sql"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/migration"
)

const legacyStreakDDL = `CREATE TABLE streak (
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
CREATE INDEX idx_streak_user_title_type ON streak(user_id, title_slug, type);
CREATE INDEX idx_streak_user_status ON streak(user_id, status);`

// legacyStreakBase rend une base joueur dans l'état d'avant le retrait : migrée, mais avec une table
// `streak` portant ids (recopiés dans streak_history sauf ceux de `manquants`) et sans trace du step
// de retrait dans le registre.
func legacyStreakBase(t *testing.T, recopies, manquants []string) *sql.DB {
	t.Helper()
	db := freshProvision(t, migration.TargetPlayer)
	if err := migration.ExecScript(db, legacyStreakDDL); err != nil {
		t.Fatalf("création de streak héritée: %v", err)
	}
	for _, id := range append(append([]string{}, recopies...), manquants...) {
		if _, err := db.Exec(`INSERT INTO streak (id, user_id, title_slug, type, started_at)
			VALUES (?, 'u', 'halo_infinite', 'daily_play', TIMESTAMP '2026-01-01 00:00:00')`, id); err != nil {
			t.Fatalf("insert streak %s: %v", id, err)
		}
	}
	for _, id := range recopies {
		if _, err := db.Exec(`INSERT INTO streak_history (id, user_id, title_slug, type, started_at)
			VALUES (?, 'u', 'halo_infinite', 'daily_play', TIMESTAMP '2026-01-01 00:00:00')`, id); err != nil {
			t.Fatalf("insert streak_history %s: %v", id, err)
		}
	}
	if _, err := db.Exec(`DELETE FROM schema_migrations WHERE name = 'drop_streak_legacy_v1'`); err != nil {
		t.Fatalf("retrait du step du registre: %v", err)
	}
	return db
}

func captureWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

func countRows(t *testing.T, db *sql.DB, query string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

// Base neuve : aucune table streak (ni index), streak_history et la vue streak_latest sont là.
func TestDropStreakLegacy_BaseNeuveSansStreak(t *testing.T) {
	db := freshProvision(t, migration.TargetPlayer)
	if n := countRows(t, db, `SELECT COUNT(*) FROM duckdb_tables() WHERE table_name = 'streak'`); n != 0 {
		t.Errorf("table streak présente sur une base neuve")
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM duckdb_indexes() WHERE index_name LIKE 'idx_streak_user_%'`); n != 0 {
		t.Errorf("index de la table streak présents sur une base neuve")
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM streak_latest`); n != 0 {
		t.Errorf("streak_latest d'une base neuve = %d lignes, attendu 0", n)
	}
}

// Base existante : streak et ses lignes recopiées sont supprimées, streak_history garde tout, aucun
// refus de réparation de clé n'est journalisé, et un second passage ne change rien.
func TestDropStreakLegacy_BaseExistanteRecopieeEstSupprimee(t *testing.T) {
	db := legacyStreakBase(t, []string{"s1", "s2"}, nil)
	logs := captureWarnings(t)

	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM duckdb_tables() WHERE table_name = 'streak'`); n != 0 {
		t.Error("table streak toujours présente après le step")
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM streak_history`); n != 2 {
		t.Errorf("streak_history = %d lignes, attendu 2 (rien n'est perdu)", n)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM streak_latest`); n != 2 {
		t.Errorf("streak_latest = %d lignes, attendu 2", n)
	}
	if strings.Contains(logs.String(), "réparation refusée") {
		t.Errorf("le réparateur de clés refuse encore une table :\n%s", logs.String())
	}
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("second RunForDB: %v", err)
	}
}

// Une ligne de streak absente de streak_history n'est jamais perdue : la table reste, le journal le
// dit.
func TestDropStreakLegacy_LigneNonRecopieeConservee(t *testing.T) {
	db := legacyStreakBase(t, []string{"s1"}, []string{"s2"})
	logs := captureWarnings(t)

	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM streak`); n != 2 {
		t.Errorf("streak = %d lignes, attendu 2 (table conservée intacte)", n)
	}
	if !strings.Contains(logs.String(), "streak héritée conservée") || !strings.Contains(logs.String(), "lignes_non_recopiees=1") {
		t.Errorf("journal sans l'avertissement de conservation :\n%s", logs.String())
	}
}
