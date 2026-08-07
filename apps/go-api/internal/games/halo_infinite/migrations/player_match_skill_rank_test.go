//go:build integration

// player_match_skill_rank_test.go — déplacé depuis internal/migration (Phase 1.5 b20).
// Fusionne les tests append-only (applyAppendOnlyMatchSkillRank/repairMatchSkillRankWrittenAt)
// et vue priorité CSR (applyMSRViewPriorityCSR) qui partageaient setupLegacyMatchSkillRank.
// Appellent les apply* directement (pas de RunForDB → provider non requis).
package migrations

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// setupLegacyMatchSkillRank crée une DuckDB :memory: avec l'ancien schéma (PK simple
// sur match_id) + N rows pour simuler une player DB pré-migration.
func setupLegacyMatchSkillRank(t *testing.T, rowCount int) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE match_skill_rank (
			match_id          VARCHAR PRIMARY KEY,
			rating_type       VARCHAR NOT NULL,
			rating_value      FLOAT,
			rating_deviation  FLOAT,
			tier              VARCHAR,
			tier_fr           VARCHAR,
			sub_tier          SMALLINT DEFAULT 0,
			tier_label        VARCHAR,
			rating_delta      FLOAT,
			playlist_group    VARCHAR,
			start_time        TIMESTAMP,
			created_at        TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			updated_at        TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE INDEX idx_msr_rating_type ON match_skill_rank(rating_type);
		CREATE INDEX idx_msr_playlist    ON match_skill_rank(playlist_group);
	`); err != nil {
		t.Fatalf("create legacy: %v", err)
	}
	for i := 0; i < rowCount; i++ {
		ratingType := "LUSR"
		if i%5 == 0 {
			ratingType = "CSR"
		}
		_, err := db.Exec(`
			INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, tier, playlist_group)
			VALUES (?, ?, ?, ?, ?)
		`, "m_"+strconv.Itoa(i), ratingType, float64(1000+i), "Onyx", "arena_slayer")
		if err != nil {
			t.Fatalf("seed row %d: %v", i, err)
		}
	}
	return db
}

func TestApplyAppendOnlyMatchSkillRank_BasicMigration(t *testing.T) {
	db := setupLegacyMatchSkillRank(t, 10)

	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	hasID, err := migration.ColumnExists(db, "match_skill_rank", "id")
	if err != nil {
		t.Fatal(err)
	}
	if !hasID {
		t.Error("colonne id absente après migration")
	}

	hasWritten, err := migration.ColumnExists(db, "match_skill_rank", "written_at")
	if err != nil {
		t.Fatal(err)
	}
	if !hasWritten {
		t.Error("colonne written_at absente après migration")
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 10 {
		t.Errorf("rows après migration = %d, want 10", n)
	}

	var latestN int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank_latest`).Scan(&latestN); err != nil {
		t.Errorf("query vue latest échouée: %v", err)
	}
	if latestN != 10 {
		t.Errorf("vue latest count = %d, want 10 (1 row par match_id, tous distincts)", latestN)
	}
}

func TestRepairMatchSkillRankWrittenAt(t *testing.T) {
	db := setupLegacyMatchSkillRank(t, 0)
	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Fatalf("append-only: %v", err)
	}
	if _, err := db.Exec(`ALTER TABLE match_skill_rank ALTER COLUMN written_at DROP DEFAULT`); err != nil {
		t.Logf("DROP DEFAULT non supporté (%v) — on teste l'idempotence quand même", err)
	}
	if err := repairMatchSkillRankWrittenAt(db); err != nil {
		t.Fatalf("repair: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO match_skill_rank (match_id, rating_type, rating_value, playlist_group)
		VALUES ('x', 'LUSR', 1500, 'arena_slayer')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	var nNull int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE match_id='x' AND written_at IS NULL`).Scan(&nNull); err != nil {
		t.Fatal(err)
	}
	if nNull != 0 {
		t.Errorf("written_at NULL après repair (%d) — DEFAULT UTC devrait le peupler", nNull)
	}
	if err := repairMatchSkillRankWrittenAt(db); err != nil {
		t.Errorf("repair idempotent: %v", err)
	}
	empty := setupLegacyMatchSkillRank(t, 0)
	if _, err := empty.Exec(`DROP TABLE match_skill_rank`); err != nil {
		t.Fatal(err)
	}
	if err := repairMatchSkillRankWrittenAt(empty); err != nil {
		t.Errorf("repair sans table devrait être no-op: %v", err)
	}
}

func TestApplyAppendOnlyMatchSkillRank_Idempotent(t *testing.T) {
	db := setupLegacyMatchSkillRank(t, 5)

	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Fatalf("apply #1: %v", err)
	}
	var n1 int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&n1)

	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Fatalf("apply #2 (idempotent): %v", err)
	}
	var n2 int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&n2)
	if n2 != n1 {
		t.Errorf("count change après re-apply : %d → %d (devrait être idempotent)", n1, n2)
	}
}

func TestApplyAppendOnlyMatchSkillRank_AppendOnlySemantics(t *testing.T) {
	db := setupLegacyMatchSkillRank(t, 0)

	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Fatalf("apply: %v", err)
	}

	for i := 0; i < 3; i++ {
		if _, err := db.ExecContext(context.Background(), `
			INSERT INTO match_skill_rank
				(match_id, rating_type, rating_value, playlist_group)
			VALUES (?, 'LUSR', ?, ?)
		`, "m1", float64(100+i*10), "arena_slayer"); err != nil {
			t.Fatalf("INSERT v%d: %v", i, err)
		}
	}

	var physical int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE match_id='m1'`).Scan(&physical)
	if physical != 3 {
		t.Errorf("rows physiques pour m1 = %d, want 3 (append-only)", physical)
	}

	var latestRating float64
	err := db.QueryRow(`SELECT rating_value FROM match_skill_rank_latest WHERE match_id='m1'`).Scan(&latestRating)
	if err != nil {
		t.Fatalf("query latest: %v", err)
	}
	if latestRating != 120 {
		t.Errorf("latest rating_value = %f, want 120 (dernière INSERT)", latestRating)
	}
}

func TestApplyAppendOnlyMatchSkillRank_NoTable_NoOp(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Errorf("migration sur DB sans table devrait être no-op, got: %v", err)
	}
}

func TestApplyMSRViewPriorityCSR_PrefersCSRWhenBothPresent(t *testing.T) {
	db := setupLegacyMatchSkillRank(t, 0)

	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Fatalf("apply v1: %v", err)
	}
	if err := applyMSRViewPriorityCSR(db); err != nil {
		t.Fatalf("apply v2: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, playlist_group)
		VALUES ('m1', 'LUSR', 25.0, 'arena_slayer')
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, playlist_group)
		VALUES ('m1', 'CSR', 1500.0, 'arena_slayer')
	`); err != nil {
		t.Fatal(err)
	}

	var ratingType string
	var ratingValue float64
	err := db.QueryRow(`
		SELECT rating_type, rating_value FROM match_skill_rank_latest
		WHERE match_id='m1'
	`).Scan(&ratingType, &ratingValue)
	if err != nil {
		t.Fatalf("query latest: %v", err)
	}
	if ratingType != "CSR" {
		t.Errorf("latest rating_type = %q, want CSR (priorité)", ratingType)
	}
	if ratingValue != 1500.0 {
		t.Errorf("latest rating_value = %f, want 1500", ratingValue)
	}

	if _, err := db.Exec(`
		INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, playlist_group)
		VALUES ('m1', 'LUSR', 99.9, 'arena_slayer')
	`); err != nil {
		t.Fatal(err)
	}
	err = db.QueryRow(`
		SELECT rating_type FROM match_skill_rank_latest WHERE match_id='m1'
	`).Scan(&ratingType)
	if err != nil {
		t.Fatal(err)
	}
	if ratingType != "CSR" {
		t.Errorf("après LUSR post-CSR : latest = %q, want CSR (priorité absolue)", ratingType)
	}
}

func TestApplyMSRViewPriorityCSR_LatestVersionWithinType(t *testing.T) {
	db := setupLegacyMatchSkillRank(t, 0)
	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Fatalf("apply v1: %v", err)
	}
	if err := applyMSRViewPriorityCSR(db); err != nil {
		t.Fatalf("apply v2: %v", err)
	}

	for _, v := range []float64{10.0, 20.0, 30.0} {
		if _, err := db.Exec(`
			INSERT INTO match_skill_rank
				(match_id, rating_type, rating_value, playlist_group)
			VALUES ('m2', 'LUSR', ?, 'arena_slayer')
		`, v); err != nil {
			t.Fatal(err)
		}
	}

	var got float64
	err := db.QueryRow(`SELECT rating_value FROM match_skill_rank_latest WHERE match_id='m2'`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != 30.0 {
		t.Errorf("latest LUSR rating = %f, want 30 (dernière INSERT)", got)
	}
}

func TestApplyMSRViewPriorityCSR_PrefersRowWithStartTime(t *testing.T) {
	db := setupLegacyMatchSkillRank(t, 0)
	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Fatalf("v1: %v", err)
	}
	if err := applyMSRViewPriorityCSR(db); err != nil {
		t.Fatalf("v2: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, playlist_group, start_time, written_at)
		VALUES ('m3', 'LUSR', 1499.0, 'arena_slayer', NULL, TIMESTAMP '2026-05-27 15:00:00')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, playlist_group, start_time, written_at)
		VALUES ('m3', 'LUSR', 1570.0, 'arena_slayer', TIMESTAMP '2026-06-10 20:00:00', NULL)`); err != nil {
		t.Fatal(err)
	}
	var got float64
	if err := db.QueryRow(`SELECT rating_value FROM match_skill_rank_latest WHERE match_id='m3'`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != 1570.0 {
		t.Errorf("latest = %.1f, want 1570 (row avec start_time, malgré written_at NULL)", got)
	}
}

func TestApplyMSRViewPriorityCSR_Idempotent(t *testing.T) {
	db := setupLegacyMatchSkillRank(t, 5)
	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Fatalf("v1: %v", err)
	}
	if err := applyMSRViewPriorityCSR(db); err != nil {
		t.Fatalf("v2 first: %v", err)
	}
	if err := applyMSRViewPriorityCSR(db); err != nil {
		t.Errorf("v2 second (idempotent): %v", err)
	}
}

func TestApplyMSRViewPriorityCSR_NoOpIfV1NotApplied(t *testing.T) {
	db := setupLegacyMatchSkillRank(t, 3)

	if err := applyMSRViewPriorityCSR(db); err != nil {
		t.Errorf("v2 sans v1 devrait être no-op, got: %v", err)
	}

	var n int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.views WHERE table_name='match_skill_rank_latest'
	`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("vue créée sans v1 préalable, expected absente")
	}
}
