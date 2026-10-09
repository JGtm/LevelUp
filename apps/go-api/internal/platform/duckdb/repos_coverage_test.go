//go:build integration

package duckdb

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

// ── Enrichissements joueur (fixture partagée) ─────────────────────────────────

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
	// Append-only #23645 : convertit player_match_enrichment (id PK + stage +
	// written_at) et crée la vue player_match_enrichment_latest (lue par les repos).
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db.SQLDb()); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
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
