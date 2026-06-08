//go:build integration

package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

// applyWorldLeaderboardMigration applique la migration de création de
// world_csr_leaderboard_snapshots sur une DB de test.
func applyWorldLeaderboardMigration(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, m := range migration.ForTarget(migration.TargetShared) {
		if m.Name == "create_world_csr_leaderboard_snapshots" {
			if err := m.ApplySchema(db); err != nil {
				t.Fatalf("ApplySchema: %v", err)
			}
			return
		}
	}
	t.Fatal("migration create_world_csr_leaderboard_snapshots introuvable")
}

// TestInsertWorldCSRSnapshot_AppendOnlyAndLatestView valide que l'insertion est
// append-only et que la vue _latest retourne le dernier snapshot par rang.
func TestInsertWorldCSRSnapshot_AppendOnlyAndLatestView(t *testing.T) {
	db := openMemDB(t).SQLDb()
	applyWorldLeaderboardMigration(t, db)
	ctx := context.Background()

	t0 := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	batch1 := []domain.LeaderboardEntry{
		{Season: "csrseason13-2", Playlist: "p", Rank: 1, Gamertag: "Twissted Mindss", CSRValue: 2180, Tier: "Onyx", FetchedAt: t0},
		{Season: "csrseason13-2", Playlist: "p", Rank: 2, Gamertag: "OR81TAL", CSRValue: 2097, Tier: "Onyx", FetchedAt: t0},
	}
	n, err := InsertWorldCSRSnapshot(ctx, db, batch1)
	if err != nil {
		t.Fatalf("InsertWorldCSRSnapshot batch1: %v", err)
	}
	if n != 2 {
		t.Fatalf("inserted = %d, want 2", n)
	}

	// Vue _latest : 2 lignes, rang 1 = Twissted Mindss.
	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM world_csr_leaderboard_latest WHERE season_id='csrseason13-2' AND playlist_id='p'`,
	).Scan(&count); err != nil {
		t.Fatalf("count latest: %v", err)
	}
	if count != 2 {
		t.Fatalf("latest count = %d, want 2", count)
	}

	// Nouveau snapshot plus récent : le rang 1 change d'occupant.
	t1 := t0.Add(24 * time.Hour)
	batch2 := []domain.LeaderboardEntry{
		{Season: "csrseason13-2", Playlist: "p", Rank: 1, Gamertag: "NewKing", CSRValue: 2222, Tier: "Onyx", FetchedAt: t1},
	}
	if _, err := InsertWorldCSRSnapshot(ctx, db, batch2); err != nil {
		t.Fatalf("InsertWorldCSRSnapshot batch2: %v", err)
	}

	// Append-only : la table garde les 3 lignes (2 + 1).
	var raw int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM world_csr_leaderboard_snapshots`).Scan(&raw); err != nil {
		t.Fatalf("count raw: %v", err)
	}
	if raw != 3 {
		t.Fatalf("raw rows = %d, want 3 (append-only)", raw)
	}

	// La vue _latest retourne le nouvel occupant du rang 1.
	var gt string
	var csr int
	if err := db.QueryRowContext(ctx,
		`SELECT gamertag, csr_value FROM world_csr_leaderboard_latest WHERE season_id='csrseason13-2' AND playlist_id='p' AND rank=1`,
	).Scan(&gt, &csr); err != nil {
		t.Fatalf("query rank 1: %v", err)
	}
	if gt != "NewKing" || csr != 2222 {
		t.Fatalf("rank 1 latest = (%q, %d), want (NewKing, 2222)", gt, csr)
	}
}
