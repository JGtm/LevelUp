//go:build integration

package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

// applyWorldLeaderboardMigration applique les migrations du classement mondial
// (création de la table + vue _latest par batch) sur une DB de test, dans l'ordre.
// Depuis Phase 1.5 b18 ces 3 steps sont title-owned → résolus via StepsFor (+ fallback
// global ForTarget pour robustesse si l'un redevenait global).
func applyWorldLeaderboardMigration(t *testing.T, db *sql.DB) {
	t.Helper()
	wanted := []string{
		"create_world_csr_leaderboard_snapshots",
		"world_csr_leaderboard_latest_by_batch",
		"add_title_slug_to_world_csr_leaderboard",
		"add_xuid_to_world_csr_leaderboard",
		"create_world_player_season_stats",
	}
	byName := map[string]migration.Migration{}
	for _, m := range migration.ForTarget(migration.TargetShared) {
		byName[m.Name] = m
	}
	for _, m := range halomigrations.StepsFor(migration.TargetShared) {
		byName[m.Name] = m
	}
	for _, name := range wanted {
		m, ok := byName[name]
		if !ok {
			t.Fatalf("migration %s introuvable", name)
		}
		if err := m.ApplySchema(db); err != nil {
			t.Fatalf("ApplySchema(%s): %v", name, err)
		}
	}
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
	n, err := InsertWorldCSRSnapshot(ctx, db, "halo_infinite", batch1)
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
	if _, err := InsertWorldCSRSnapshot(ctx, db, "halo_infinite", batch2); err != nil {
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

	// Fix Frankenstein : la vue groupe par batch (fetched_at), pas par rang. Le
	// batch2 (1 ligne, t1) REMPLACE entièrement batch1 pour (csrseason13-2, p) :
	// la queue de batch1 (rang 2) ne doit PAS subsister dans _latest.
	var latestCount int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM world_csr_leaderboard_latest WHERE season_id='csrseason13-2' AND playlist_id='p'`,
	).Scan(&latestCount); err != nil {
		t.Fatalf("count latest après batch2: %v", err)
	}
	if latestCount != 1 {
		t.Fatalf("latest count après batch2 = %d, want 1 (batch remplacé, pas fusionné)", latestCount)
	}
}

// TestGetWorldLeaderboardCatalog valide la remontée des saisons/playlists
// distinctes présentes en base + la résolution du libellé de playlist.
func TestGetWorldLeaderboardCatalog(t *testing.T) {
	shared := openMemDB(t)
	applyWorldLeaderboardMigration(t, shared.SQLDb())
	ctx := context.Background()

	arenaID := "edfef3ac-9cbe-4fa2-b949-8f29deafd483" // Ranked Arena (NameFR "Arène classée")

	// Leaderboard scrappé : 4 saisons (dont csrseason6-1 à un chiffre pour valider le tri
	// NUMÉRIQUE, et csrseason4-1 archivée), 2 playlists.
	lbRows := []domain.LeaderboardEntry{
		{Season: "csrseason13-2", Playlist: arenaID, Rank: 1, Gamertag: "A", CSRValue: 2000, Tier: "Onyx", FetchedAt: time.Now().UTC()},
		{Season: "csrseason12-1", Playlist: arenaID, Rank: 1, Gamertag: "B", CSRValue: 1900, Tier: "Onyx", FetchedAt: time.Now().UTC()},
		{Season: "csrseason6-1", Playlist: arenaID, Rank: 1, Gamertag: "C", CSRValue: 1800, Tier: "Diamond", FetchedAt: time.Now().UTC()},
		{Season: "csrseason6-1", Playlist: "unknown-pl", Rank: 1, Gamertag: "D", CSRValue: 1750, Tier: "Diamond", FetchedAt: time.Now().UTC()},
		{Season: "csrseason4-1", Playlist: arenaID, Rank: 1, Gamertag: "Z", CSRValue: 1700, Tier: "Diamond", FetchedAt: time.Now().UTC()},
	}
	if _, err := InsertWorldCSRSnapshot(ctx, shared.SQLDb(), "halo_infinite", lbRows); err != nil {
		t.Fatalf("InsertWorldCSRSnapshot: %v", err)
	}

	// Stats enrichies pour 3 saisons ; PAS csrseason4-1 (archivée → restera Enriched=false).
	statRows := []domain.WorldPlayerSeasonStats{
		{TitleSlug: "halo_infinite", Gamertag: "A", SeasonID: "csrseason13-2", PlaylistID: arenaID, MatchCount: 10, WinCount: 6},
		{TitleSlug: "halo_infinite", Gamertag: "B", SeasonID: "csrseason12-1", PlaylistID: arenaID, MatchCount: 8, WinCount: 4},
		{TitleSlug: "halo_infinite", Gamertag: "C", SeasonID: "csrseason6-1", PlaylistID: arenaID, MatchCount: 5, WinCount: 2},
	}
	if _, err := InsertPlayerSeasonStats(ctx, shared.SQLDb(), statRows); err != nil {
		t.Fatalf("InsertPlayerSeasonStats: %v", err)
	}

	repo := NewLeaderboardRepo(&PlayerDB{Shared: shared})
	cat, err := repo.GetWorldLeaderboardCatalog(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("GetWorldLeaderboardCatalog: %v", err)
	}

	// Option C : TOUTES les saisons du leaderboard sont exposées, triées NUMÉRIQUEMENT
	// récent-d'abord (6-1 en DERNIER, pas en tête comme un tri lexicographique) ; chaque
	// saison porte Enriched (csrseason4-1 sans stats → false, affichée en classement seul).
	want := []struct {
		id       string
		enriched bool
	}{
		{"csrseason13-2", true}, {"csrseason12-1", true}, {"csrseason6-1", true}, {"csrseason4-1", false},
	}
	if len(cat.Seasons) != len(want) {
		t.Fatalf("seasons = %+v, attendu %d (toutes les saisons du leaderboard)", cat.Seasons, len(want))
	}
	for i, w := range want {
		if cat.Seasons[i].ID != w.id || cat.Seasons[i].Enriched != w.enriched {
			t.Errorf("season[%d] = {%s, enriched=%v}, attendu {%s, enriched=%v}",
				i, cat.Seasons[i].ID, cat.Seasons[i].Enriched, w.id, w.enriched)
		}
	}
	// Playlists : 2 distinctes (depuis le leaderboard) ; la playlist connue a son libellé FR.
	if len(cat.Playlists) != 2 {
		t.Fatalf("playlists = %d, attendu 2 (%+v)", len(cat.Playlists), cat.Playlists)
	}
	var arenaLabel, unknownLabel string
	for _, p := range cat.Playlists {
		switch p.ID {
		case arenaID:
			arenaLabel = p.DisplayName
		case "unknown-pl":
			unknownLabel = p.DisplayName
		}
	}
	if arenaLabel != "Arène classée" {
		t.Errorf("libellé Arena = %q, attendu \"Arène classée\"", arenaLabel)
	}
	if unknownLabel != "unknown-pl" {
		t.Errorf("libellé inconnu = %q, attendu fallback sur l'asset_id", unknownLabel)
	}
}
