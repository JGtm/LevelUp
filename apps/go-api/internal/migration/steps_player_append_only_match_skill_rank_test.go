//go:build integration

// Package migration — tests pour applyAppendOnlyMatchSkillRank (Phase 2.B
// du plan d'éradication ART).

package migration

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupLegacyMatchSkillRank crée une DuckDB :memory: avec l'ancien schéma
// (PK simple sur match_id) + N rows pour simuler une player DB pré-migration.
func setupLegacyMatchSkillRank(t *testing.T, rowCount int) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	// Schéma legacy reproduit (cf. steps_player.go:302)
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
			created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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

// TestApplyAppendOnlyMatchSkillRank_BasicMigration — applique sur DB
// legacy peuplée, vérifie : id ajouté, written_at ajouté, vue créée,
// rows préservées.
func TestApplyAppendOnlyMatchSkillRank_BasicMigration(t *testing.T) {
	db := setupLegacyMatchSkillRank(t, 10)

	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	// Colonne id présente
	hasID, err := columnExists(db, "match_skill_rank", "id")
	if err != nil {
		t.Fatal(err)
	}
	if !hasID {
		t.Error("colonne id absente après migration")
	}

	// Colonne written_at présente
	hasWritten, err := columnExists(db, "match_skill_rank", "written_at")
	if err != nil {
		t.Fatal(err)
	}
	if !hasWritten {
		t.Error("colonne written_at absente après migration")
	}

	// Rows préservées
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 10 {
		t.Errorf("rows après migration = %d, want 10", n)
	}

	// Vue match_skill_rank_latest accessible
	var latestN int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank_latest`).Scan(&latestN); err != nil {
		t.Errorf("query vue latest échouée: %v", err)
	}
	if latestN != 10 {
		t.Errorf("vue latest count = %d, want 10 (1 row par match_id, tous distincts)", latestN)
	}
}

// TestRepairMatchSkillRankWrittenAt — la migration de réparation (re)pose le
// DEFAULT now() sur written_at en DDL pur (jamais d'UPDATE → pas de bug ART).
// Idempotente ; no-op si la table n'existe pas.
func TestRepairMatchSkillRankWrittenAt(t *testing.T) {
	db := setupLegacyMatchSkillRank(t, 0)
	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Fatalf("append-only: %v", err)
	}
	// Simule une migration append-only partielle : DEFAULT de written_at absent.
	if _, err := db.Exec(`ALTER TABLE match_skill_rank ALTER COLUMN written_at DROP DEFAULT`); err != nil {
		t.Logf("DROP DEFAULT non supporté (%v) — on teste l'idempotence quand même", err)
	}
	if err := repairMatchSkillRankWrittenAt(db); err != nil {
		t.Fatalf("repair: %v", err)
	}
	// Le DEFAULT now() est effectif : un INSERT sans written_at le peuple.
	if _, err := db.Exec(`INSERT INTO match_skill_rank (match_id, rating_type, rating_value, playlist_group)
		VALUES ('x', 'LUSR', 1500, 'arena_slayer')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	var nNull int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE match_id='x' AND written_at IS NULL`).Scan(&nNull); err != nil {
		t.Fatal(err)
	}
	if nNull != 0 {
		t.Errorf("written_at NULL après repair (%d) — DEFAULT now() devrait le peupler", nNull)
	}
	// Idempotente.
	if err := repairMatchSkillRankWrittenAt(db); err != nil {
		t.Errorf("repair idempotent: %v", err)
	}
	// No-op si la table n'existe pas.
	empty := setupLegacyMatchSkillRank(t, 0)
	if _, err := empty.Exec(`DROP TABLE match_skill_rank`); err != nil {
		t.Fatal(err)
	}
	if err := repairMatchSkillRankWrittenAt(empty); err != nil {
		t.Errorf("repair sans table devrait être no-op: %v", err)
	}
}

// TestApplyAppendOnlyMatchSkillRank_Idempotent — re-appliquer la
// migration sur une DB déjà migrée doit être un no-op.
func TestApplyAppendOnlyMatchSkillRank_Idempotent(t *testing.T) {
	db := setupLegacyMatchSkillRank(t, 5)

	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Fatalf("apply #1: %v", err)
	}

	var n1 int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&n1)

	// 2e application doit être no-op
	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Fatalf("apply #2 (idempotent): %v", err)
	}

	var n2 int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&n2)
	if n2 != n1 {
		t.Errorf("count change après re-apply : %d → %d (devrait être idempotent)", n1, n2)
	}
}

// TestApplyAppendOnlyMatchSkillRank_AppendOnlySemantics — après migration,
// vérifie que des INSERTs répétés sur le même match_id s'accumulent
// (n'écrasent pas) et que la vue latest renvoie la version la plus récente.
func TestApplyAppendOnlyMatchSkillRank_AppendOnlySemantics(t *testing.T) {
	db := setupLegacyMatchSkillRank(t, 0) // table vide

	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// Insérer 3 versions du même match_id (id et written_at auto-générés)
	for i := 0; i < 3; i++ {
		if _, err := db.ExecContext(context.Background(), `
			INSERT INTO match_skill_rank
				(match_id, rating_type, rating_value, playlist_group)
			VALUES (?, 'LUSR', ?, ?)
		`, "m1", float64(100+i*10), "arena_slayer"); err != nil {
			t.Fatalf("INSERT v%d: %v", i, err)
		}
	}

	// Table physique : 3 rows pour m1
	var physical int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE match_id='m1'`).Scan(&physical)
	if physical != 3 {
		t.Errorf("rows physiques pour m1 = %d, want 3 (append-only)", physical)
	}

	// Vue latest : 1 row pour m1 avec rating_value = 120 (la dernière)
	var latestRating float64
	err := db.QueryRow(`SELECT rating_value FROM match_skill_rank_latest WHERE match_id='m1'`).Scan(&latestRating)
	if err != nil {
		t.Fatalf("query latest: %v", err)
	}
	if latestRating != 120 {
		t.Errorf("latest rating_value = %f, want 120 (dernière INSERT)", latestRating)
	}
}

// TestApplyAppendOnlyMatchSkillRank_NoTable_NoOp — si la table n'existe
// pas (DB neuve avant steps_player.go), la migration no-op proprement.
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
