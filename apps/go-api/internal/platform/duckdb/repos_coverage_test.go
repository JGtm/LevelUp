//go:build integration

package duckdb

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// ── FanoutRepo ───────────────────────────────────────────────────────────────

func seedPlayerEnrichment(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	ddl := `CREATE TABLE IF NOT EXISTS player_match_enrichment (
		match_id VARCHAR PRIMARY KEY,
		performance_score DOUBLE DEFAULT 0,
		session_id VARCHAR DEFAULT '',
		is_with_friends BOOLEAN DEFAULT FALSE,
		is_excluded BOOLEAN DEFAULT FALSE,
		updated_at TIMESTAMPTZ DEFAULT NOW()
	)`
	if _, err := db.Exec(ctx, ddl); err != nil {
		t.Fatalf("seedPlayerEnrichment: %v", err)
	}
}

func TestFanoutRepo_InsertStubEnrichments(t *testing.T) {
	db := openMemDB(t)
	seedPlayerEnrichment(t, db)

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewFanoutRepo(pdb)

	inserted, err := repo.InsertStubEnrichments(context.Background(), "xuid001", []string{"m1", "m2", "m3"})
	if err != nil {
		t.Fatal(err)
	}
	if inserted != 3 {
		t.Fatalf("expected 3 inserted, got %d", inserted)
	}
}

func TestFanoutRepo_InsertStubEnrichments_Empty(t *testing.T) {
	db := openMemDB(t)
	seedPlayerEnrichment(t, db)

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewFanoutRepo(pdb)

	inserted, err := repo.InsertStubEnrichments(context.Background(), "xuid001", nil)
	if err != nil {
		t.Fatal(err)
	}
	if inserted != 0 {
		t.Fatalf("expected 0 inserted, got %d", inserted)
	}
}

func TestFanoutRepo_InsertStubEnrichments_Idempotent(t *testing.T) {
	db := openMemDB(t)
	seedPlayerEnrichment(t, db)

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewFanoutRepo(pdb)
	ctx := context.Background()

	// 1er appel : m1, m2 → 2 rows insérées.
	first, err := repo.InsertStubEnrichments(ctx, "xuid001", []string{"m1", "m2"})
	if err != nil {
		t.Fatalf("1er insert: %v", err)
	}
	if first != 2 {
		t.Errorf("1er insert count = %d, want 2", first)
	}

	// 2e appel : m1, m2 (déjà présents), m3 → INSERT OR IGNORE n'erreure pas
	// sur les duplicates, donc le compteur retourne 3 (toutes les iterations
	// du loop "réussissent" côté SQL).
	second, err := repo.InsertStubEnrichments(ctx, "xuid001", []string{"m1", "m2", "m3"})
	if err != nil {
		t.Fatalf("2e insert: %v", err)
	}
	if second != 3 {
		t.Errorf("2e insert count = %d, want 3 (loop count, pas affected rows)", second)
	}

	// Vraie vérification de l'idempotence : 3 rows distinctes au total
	// dans la DB (m1, m2, m3), pas 5 (qui prouverait que INSERT OR IGNORE
	// crée des doublons). C'est l'invariance critique du nom du test.
	var count int
	if err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM player_match_enrichment WHERE match_id IN ('m1','m2','m3')`).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 3 {
		t.Errorf("rows distinctes = %d, want 3 (idempotence cassée)", count)
	}
}

func TestFanoutRepo_LoadExistingEnrichments_Subset(t *testing.T) {
	db := openMemDB(t)
	seedPlayerEnrichment(t, db)
	ctx := context.Background()

	if _, err := db.Exec(ctx, `INSERT INTO player_match_enrichment (match_id) VALUES ('m1'), ('m3')`); err != nil {
		t.Fatal(err)
	}

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewFanoutRepo(pdb)

	existing, err := repo.LoadExistingEnrichments(ctx, []string{"m1", "m2", "m3"})
	if err != nil {
		t.Fatal(err)
	}
	if !existing["m1"] || existing["m2"] || !existing["m3"] {
		t.Fatalf("unexpected existing: %v", existing)
	}
}

func TestFanoutRepo_LoadExistingEnrichments_EmptyInput(t *testing.T) {
	db := openMemDB(t)
	seedPlayerEnrichment(t, db)

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewFanoutRepo(pdb)

	existing, err := repo.LoadExistingEnrichments(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(existing) != 0 {
		t.Fatalf("expected empty, got %v", existing)
	}
}

func TestFanoutRepo_CountCommonMatchesForXUID(t *testing.T) {
	db := openMemDB(t)
	seedShared(t, db)

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewFanoutRepo(pdb)

	count, err := repo.CountCommonMatchesForXUID(context.Background(), "xuid001", []string{"m1", "m2", "m3"})
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 { // xuid001 is in m1, m2
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestFanoutRepo_CountCommonMatchesForXUID_Empty(t *testing.T) {
	db := openMemDB(t)

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewFanoutRepo(pdb)

	count, err := repo.CountCommonMatchesForXUID(context.Background(), "xuid001", nil)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}

// ── LeaderboardRepo ──────────────────────────────────────────────────────────
//
// 2 tests retirés au commit 12c : ils étaient `t.Skip`d depuis longtemps avec
// commentaire "pre-existing SQL bug: ambiguous gamertag reference in GROUP BY",
// mais la query actuelle de GetLocalLeaderboard (leaderboard_repo.go:34) n'a
// ni GROUP BY ni colonne gamertag — elle lit `match_skill_rank` (player DB) +
// `shared.match_registry`, pas `shared.match_participants` comme seedait le
// test. Le code a évolué, le test était mort. Couverture future via un test
// fresh aligné sur la query réelle.
//
// ── MatchExclusionRepo ───────────────────────────────────────────────────────

func seedSharedForExclusion(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	ddl := []string{
		`CREATE SCHEMA IF NOT EXISTS shared`,
		// start_time TIMESTAMP (naïf, convention mixte) + start_time_utc
		// TIMESTAMPTZ (UTC garanti). Les queries de prod lisent toujours
		// COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC').
		`CREATE TABLE IF NOT EXISTS shared.match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP,
			start_time_utc TIMESTAMPTZ,
			map_name VARCHAR DEFAULT '',
			pair_name VARCHAR DEFAULT '',
			is_ranked BOOLEAN DEFAULT FALSE,
			is_firefight BOOLEAN DEFAULT FALSE
		)`,
		// Vue root-level pour le pipeline split (P7-4) : match_registry sans
		// préfixe `shared.` lu via SharedReader.
		`CREATE VIEW IF NOT EXISTS match_registry AS SELECT * FROM shared.match_registry`,
	}
	for _, q := range ddl {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("seedSharedForExclusion: %v\nSQL: %s", err, q)
		}
	}
	_, err := db.Exec(ctx, `INSERT INTO shared.match_registry VALUES
		('m1', TIMESTAMP '2025-01-10 14:00:00', TIMESTAMPTZ '2025-01-10 14:00:00+00', 'Recharge', 'Slayer', FALSE, FALSE),
		('m2', TIMESTAMP '2025-01-11 18:00:00', TIMESTAMPTZ '2025-01-11 18:00:00+00', 'Streets', 'CTF', FALSE, FALSE),
		('m_ranked', TIMESTAMP '2025-01-12 20:00:00', TIMESTAMPTZ '2025-01-12 20:00:00+00', 'Live Fire', 'Ranked Slayer', TRUE, FALSE),
		('m_ff', TIMESTAMP '2025-01-13 22:00:00', TIMESTAMPTZ '2025-01-13 22:00:00+00', 'Outpost Tremonios', 'Firefight', FALSE, TRUE)`)
	if err != nil {
		t.Fatalf("seedSharedForExclusion INSERT: %v", err)
	}
}

func TestMatchExclusionRepo_ListExcluded_NoExcluded(t *testing.T) {
	db := openMemDB(t)
	seedSharedForExclusion(t, db)
	seedPlayerEnrichment(t, db)

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewMatchExclusionRepo(pdb)

	results, err := repo.ListExcluded(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0, got %d", len(results))
	}
}

func TestMatchExclusionRepo_ListExcluded_WithData(t *testing.T) {
	db := openMemDB(t)
	seedSharedForExclusion(t, db)
	seedPlayerEnrichment(t, db)
	ctx := context.Background()

	_, err := db.Exec(ctx, `INSERT INTO player_match_enrichment (match_id, is_excluded) VALUES ('m1', TRUE), ('m2', FALSE)`)
	if err != nil {
		t.Fatal(err)
	}

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewMatchExclusionRepo(pdb)

	results, err := repo.ListExcluded(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 excluded, got %d", len(results))
	}
	if results[0].MatchID != "m1" {
		t.Fatalf("expected m1, got %s", results[0].MatchID)
	}
}

func TestMatchExclusionRepo_GetMatchRegistryInfo_Social(t *testing.T) {
	db := openMemDB(t)
	seedSharedForExclusion(t, db)
	seedPlayerEnrichment(t, db)

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewMatchExclusionRepo(pdb)

	info, err := repo.GetMatchRegistryInfo(context.Background(), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.MatchID != "m1" {
		t.Fatalf("expected match_id m1, got %s", info.MatchID)
	}
	if info.IsRanked {
		t.Error("m1 should not be ranked")
	}
	if info.IsFirefight {
		t.Error("m1 should not be firefight")
	}
	if info.PairName != "Slayer" {
		t.Errorf("expected pair_name 'Slayer', got %q", info.PairName)
	}
	if info.StartTime.IsZero() {
		t.Error("start_time should not be zero")
	}
}

func TestMatchExclusionRepo_GetMatchRegistryInfo_Ranked(t *testing.T) {
	db := openMemDB(t)
	seedSharedForExclusion(t, db)
	seedPlayerEnrichment(t, db)

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewMatchExclusionRepo(pdb)

	info, err := repo.GetMatchRegistryInfo(context.Background(), "m_ranked")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.IsRanked {
		t.Error("m_ranked should be flagged is_ranked=true")
	}
	if info.PairName != "Ranked Slayer" {
		t.Errorf("expected pair_name 'Ranked Slayer', got %q", info.PairName)
	}
}

func TestMatchExclusionRepo_GetMatchRegistryInfo_Firefight(t *testing.T) {
	db := openMemDB(t)
	seedSharedForExclusion(t, db)
	seedPlayerEnrichment(t, db)

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewMatchExclusionRepo(pdb)

	info, err := repo.GetMatchRegistryInfo(context.Background(), "m_ff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.IsRanked {
		t.Error("m_ff should not be ranked")
	}
	if !info.IsFirefight {
		t.Error("m_ff should be flagged is_firefight=true")
	}
}

func TestMatchExclusionRepo_GetMatchRegistryInfo_NotFound(t *testing.T) {
	db := openMemDB(t)
	seedSharedForExclusion(t, db)
	seedPlayerEnrichment(t, db)

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewMatchExclusionRepo(pdb)

	_, err := repo.GetMatchRegistryInfo(context.Background(), "ghost-id")
	if !errors.Is(err, domain.ErrMatchNotFound) {
		t.Fatalf("expected domain.ErrMatchNotFound, got %v", err)
	}
}
