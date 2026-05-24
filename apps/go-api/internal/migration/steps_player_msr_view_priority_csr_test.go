//go:build integration

// Package migration — tests pour applyMSRViewPriorityCSR (Phase 2.E).

package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestApplyMSRViewPriorityCSR_PrefersCSRWhenBothPresent — quand un match
// a physiquement à la fois une row CSR et une row LUSR, la vue latest
// doit renvoyer le CSR (sémantique métier : CSR est l'autorité Halo).
func TestApplyMSRViewPriorityCSR_PrefersCSRWhenBothPresent(t *testing.T) {
	db := setupLegacyMatchSkillRank(t, 0)

	// Appliquer v1 (append-only) puis v2 (priorité CSR)
	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Fatalf("apply v1: %v", err)
	}
	if err := applyMSRViewPriorityCSR(db); err != nil {
		t.Fatalf("apply v2: %v", err)
	}

	// Insérer LUSR d'abord, CSR ensuite (CSR plus récent)
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

	// Vue latest : doit renvoyer la CSR (priorité)
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

	// Inverser l'ordre temporel : insérer LUSR APRÈS le CSR → CSR doit
	// quand même gagner (priorité fonctionnelle, pas temporelle).
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

// TestApplyMSRViewPriorityCSR_LatestVersionWithinType — quand plusieurs
// versions d'un même rating_type existent, la plus récente gagne.
func TestApplyMSRViewPriorityCSR_LatestVersionWithinType(t *testing.T) {
	db := setupLegacyMatchSkillRank(t, 0)
	if err := applyAppendOnlyMatchSkillRank(db); err != nil {
		t.Fatalf("apply v1: %v", err)
	}
	if err := applyMSRViewPriorityCSR(db); err != nil {
		t.Fatalf("apply v2: %v", err)
	}

	// 3 versions LUSR sur m2
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

// TestApplyMSRViewPriorityCSR_Idempotent — re-apply de la migration v2
// doit être un no-op (CREATE OR REPLACE est intrinsèquement idempotent).
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

// TestApplyMSRViewPriorityCSR_NoOpIfV1NotApplied — si la colonne `id`
// est absente (v1 pas appliquée), la migration v2 no-op silencieusement.
func TestApplyMSRViewPriorityCSR_NoOpIfV1NotApplied(t *testing.T) {
	db := setupLegacyMatchSkillRank(t, 3)
	// On NE PAS applique v1 → pas de colonne id

	if err := applyMSRViewPriorityCSR(db); err != nil {
		t.Errorf("v2 sans v1 devrait être no-op, got: %v", err)
	}

	// La vue ne doit pas avoir été créée
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

// Note : variables inutilisées importées pour éviter "imported and not used"
// si le test file devient le seul utilisateur.
var _ = sql.ErrNoRows
