//go:build integration

// Package persist — post_sync_lusr_persister_test.go : tests TDD pour
// PostSyncLUSRPersister (Phase 4 du refactor Collect→Persist).

package persist

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openLUSRTestDB ouvre une DuckDB :memory: avec le schéma match_skill_rank.
func openLUSRTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
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
			created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

// ─── Test 1 : Persist batch nominal ───────────────────────────────────────

func TestPostSyncLUSRPersister_Persist_InsertsBatch(t *testing.T) {
	db := openLUSRTestDB(t)
	p := NewPostSyncLUSRPersister(db)

	strPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }

	rows := []LUSRRatingInsert{
		{MatchID: "m1", RatingValue: 25.5, RatingDeviation: 3.2,
			Tier: strPtr("Diamond"), SubTier: intPtr(2), TierLabel: strPtr("Diamond 2"),
			PlaylistGroup: "arena_slayer"},
		{MatchID: "m2", RatingValue: 27.1, RatingDeviation: 2.8,
			Tier: strPtr("Onyx"), SubTier: intPtr(0), TierLabel: strPtr("Onyx 27"),
			PlaylistGroup: "arena_slayer"},
	}
	if err := p.Persist(context.Background(), rows); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type='LUSR'`).Scan(&n)
	if n != 2 {
		t.Errorf("LUSR rows = %d, want 2", n)
	}
}

// ─── Test 2 : DELETE WHERE rating_type='LUSR' préserve les CSR ────────────

func TestPostSyncLUSRPersister_Persist_PreservesCSRRows(t *testing.T) {
	db := openLUSRTestDB(t)

	// Pré-seed 1 row CSR (qui doit survivre).
	if _, err := db.Exec(`
		INSERT INTO match_skill_rank (match_id, rating_type, rating_value, tier, playlist_group)
		VALUES ('csr_match', 'CSR', 1450, 'Onyx', 'ranked_arena')
	`); err != nil {
		t.Fatal(err)
	}

	// Pré-seed 2 rows LUSR (qui doivent être remplacées).
	for _, mid := range []string{"old_lusr_1", "old_lusr_2"} {
		_, _ = db.Exec(`
			INSERT INTO match_skill_rank (match_id, rating_type, rating_value, playlist_group)
			VALUES (?, 'LUSR', 20.0, 'arena_slayer')
		`, mid)
	}

	p := NewPostSyncLUSRPersister(db)
	rows := []LUSRRatingInsert{
		{MatchID: "new_lusr_1", RatingValue: 25.0, PlaylistGroup: "arena_slayer"},
	}
	if err := p.Persist(context.Background(), rows); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	// CSR row préservée
	var csrCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type='CSR'`).Scan(&csrCount)
	if csrCount != 1 {
		t.Errorf("CSR rows after persist = %d, want 1 (préservées)", csrCount)
	}

	// 2 anciennes LUSR supprimées, 1 nouvelle insérée
	var lusrCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type='LUSR'`).Scan(&lusrCount)
	if lusrCount != 1 {
		t.Errorf("LUSR rows after persist = %d, want 1 (anciennes deleted, 1 nouvelle insérée)", lusrCount)
	}

	// Vérifier que la nouvelle row est là
	var newMid string
	_ = db.QueryRow(`SELECT match_id FROM match_skill_rank WHERE rating_type='LUSR'`).Scan(&newMid)
	if newMid != "new_lusr_1" {
		t.Errorf("nouvelle LUSR match_id = %q, want new_lusr_1", newMid)
	}
}

// ─── Test 3 : Empty batch → no-op (préserve l'existant) ───────────────────

func TestPostSyncLUSRPersister_Persist_EmptyBatch_NoOp(t *testing.T) {
	db := openLUSRTestDB(t)

	// Pré-seed 1 LUSR
	_, _ = db.Exec(`
		INSERT INTO match_skill_rank (match_id, rating_type, rating_value, playlist_group)
		VALUES ('keep_me', 'LUSR', 22.0, 'arena_slayer')
	`)

	p := NewPostSyncLUSRPersister(db)
	if err := p.Persist(context.Background(), nil); err != nil {
		t.Fatalf("Persist nil: %v", err)
	}
	if err := p.Persist(context.Background(), []LUSRRatingInsert{}); err != nil {
		t.Fatalf("Persist empty: %v", err)
	}

	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&n)
	if n != 1 {
		t.Errorf("rows = %d, want 1 (empty batch ne touche pas l'existant)", n)
	}
}

// ─── Test 4 : Atomicité — INSERT échoue → DELETE rollback ─────────────────

func TestPostSyncLUSRPersister_Persist_AtomicityRollback(t *testing.T) {
	db := openLUSRTestDB(t)

	// Pré-seed 2 LUSR (qui doivent survivre si la TX rollback).
	for _, mid := range []string{"survive_1", "survive_2"} {
		_, _ = db.Exec(`
			INSERT INTO match_skill_rank (match_id, rating_type, rating_value, playlist_group)
			VALUES (?, 'LUSR', 20.0, 'arena_slayer')
		`, mid)
	}

	p := NewPostSyncLUSRPersister(db)

	// Batch invalide : 2 rows avec le même match_id → PK conflict au 2e INSERT
	rows := []LUSRRatingInsert{
		{MatchID: "dup", RatingValue: 25.0, PlaylistGroup: "arena_slayer"},
		{MatchID: "dup", RatingValue: 26.0, PlaylistGroup: "arena_slayer"}, // PK violation
	}
	if err := p.Persist(context.Background(), rows); err == nil {
		t.Error("Persist devrait échouer sur PK duplicate")
	}

	// Vérifier que les 2 LUSR pré-seed ont survécu (TX rollback)
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type='LUSR'`).Scan(&n)
	if n != 2 {
		t.Errorf("LUSR rows post-rollback = %d, want 2 (TX devrait avoir rollback)", n)
	}
}

// ─── Test 6 : Upsert ne touche QUE les match_id du batch ──────────────────

func TestPostSyncLUSRPersister_Upsert_PreservesOtherLUSRRows(t *testing.T) {
	db := openLUSRTestDB(t)

	// Pré-seed 3 LUSR rows (lusr_A, lusr_B, lusr_C)
	for _, mid := range []string{"lusr_A", "lusr_B", "lusr_C"} {
		_, _ = db.Exec(`
			INSERT INTO match_skill_rank (match_id, rating_type, rating_value, playlist_group)
			VALUES (?, 'LUSR', 20.0, 'arena_slayer')
		`, mid)
	}

	p := NewPostSyncLUSRPersister(db)
	// Upsert seulement lusr_A et lusr_B (lusr_C doit être préservé intact)
	rows := []LUSRRatingInsert{
		{MatchID: "lusr_A", RatingValue: 25.0, PlaylistGroup: "arena_slayer"},
		{MatchID: "lusr_B", RatingValue: 26.0, PlaylistGroup: "arena_slayer"},
	}
	if err := p.Upsert(context.Background(), rows); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	// 3 LUSR rows toujours présentes
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type='LUSR'`).Scan(&n)
	if n != 3 {
		t.Errorf("LUSR rows = %d, want 3 (lusr_C préservé + lusr_A/B upserted)", n)
	}

	// lusr_A et lusr_B mis à jour (rating_value=25/26)
	var ratingA, ratingB, ratingC float64
	_ = db.QueryRow(`SELECT rating_value FROM match_skill_rank WHERE match_id='lusr_A'`).Scan(&ratingA)
	_ = db.QueryRow(`SELECT rating_value FROM match_skill_rank WHERE match_id='lusr_B'`).Scan(&ratingB)
	_ = db.QueryRow(`SELECT rating_value FROM match_skill_rank WHERE match_id='lusr_C'`).Scan(&ratingC)

	if ratingA != 25.0 {
		t.Errorf("lusr_A rating = %f, want 25.0 (upserted)", ratingA)
	}
	if ratingB != 26.0 {
		t.Errorf("lusr_B rating = %f, want 26.0 (upserted)", ratingB)
	}
	if ratingC != 20.0 {
		t.Errorf("lusr_C rating = %f, want 20.0 (préservé, non touché)", ratingC)
	}
}

// ─── Test 7 : Upsert préserve CSR existant ─────────────────────────────────

func TestPostSyncLUSRPersister_Upsert_PreservesCSRForSameMatchID(t *testing.T) {
	db := openLUSRTestDB(t)

	// Pré-seed 1 CSR sur match_id 'ranked_match'
	_, _ = db.Exec(`
		INSERT INTO match_skill_rank (match_id, rating_type, rating_value, tier, playlist_group)
		VALUES ('ranked_match', 'CSR', 1450, 'Onyx', 'ranked_arena')
	`)

	p := NewPostSyncLUSRPersister(db)
	// Tenter d'upsert un LUSR sur le même match_id que le CSR
	// → PK conflict (CSR a déjà la row, et le DELETE filtre LUSR only)
	rows := []LUSRRatingInsert{
		{MatchID: "ranked_match", RatingValue: 99.9, PlaylistGroup: "arena_slayer"},
	}
	err := p.Upsert(context.Background(), rows)
	if err == nil {
		t.Error("Upsert devrait échouer (PK conflict avec CSR existant)")
	}

	// Le CSR doit être intact
	var rating float64
	_ = db.QueryRow(`SELECT rating_value FROM match_skill_rank WHERE match_id='ranked_match'`).Scan(&rating)
	if rating != 1450.0 {
		t.Errorf("CSR rating = %f, want 1450.0 (intact)", rating)
	}
}

// ─── Test 5 : INSERT-only property — aucun UPDATE émis ─────────────────────
//
// Vérifie indirectement que les rating_value des CSR ne sont jamais
// touchées par le Persister (qui ne fait que DELETE WHERE LUSR + INSERT LUSR).

func TestPostSyncLUSRPersister_Persist_NeverUpdatesCSRRow(t *testing.T) {
	db := openLUSRTestDB(t)

	_, _ = db.Exec(`
		INSERT INTO match_skill_rank (match_id, rating_type, rating_value, tier, playlist_group)
		VALUES ('csr_immutable', 'CSR', 1450.0, 'Onyx', 'ranked_arena')
	`)

	p := NewPostSyncLUSRPersister(db)
	// Tenter d'ajouter un LUSR sur le MÊME match_id que le CSR — devrait
	// échouer car PK conflict (LUSR ne peut pas écraser CSR par construction).
	rows := []LUSRRatingInsert{
		{MatchID: "csr_immutable", RatingValue: 99.9, PlaylistGroup: "arena_slayer"},
	}
	err := p.Persist(context.Background(), rows)
	if err == nil {
		t.Error("Persist devrait échouer (LUSR ne peut pas avoir même match_id qu'un CSR)")
	}

	// Le CSR initial doit être intact (rating_value=1450, pas 99.9)
	var rating float64
	_ = db.QueryRow(`SELECT rating_value FROM match_skill_rank WHERE match_id='csr_immutable'`).Scan(&rating)
	if rating != 1450.0 {
		t.Errorf("CSR rating_value = %f, want 1450.0 (jamais touché)", rating)
	}
}
