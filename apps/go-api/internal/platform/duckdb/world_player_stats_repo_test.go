//go:build integration

package duckdb

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

// applyWorldPlayerStatsMigration applique create_world_player_season_stats (title-owned
// depuis Phase 1.5 b18 → résolu via StepsFor, + fallback global ForTarget).
func applyWorldPlayerStatsMigration(t *testing.T, db *sql.DB) {
	t.Helper()
	all := append(migration.ForTarget(migration.TargetShared), halomigrations.StepsFor(migration.TargetShared)...)
	for _, m := range all {
		if m.Name == "create_world_player_season_stats" {
			if err := m.ApplySchema(db); err != nil {
				t.Fatalf("ApplySchema(create_world_player_season_stats): %v", err)
			}
			return
		}
	}
	t.Fatal("migration create_world_player_season_stats introuvable")
}

// TestGetWorldPlayerSeasonStats_DerivedAndInterSeason valide, sur un dataset
// multi-saison hétérogène : (1) les ratios dérivés (win_rate, kda) ; (2) le LAG
// inter-saison qui SAUTE une saison où la playlist est absente (Alpha ne joue
// pas Arena en 13-1 → prev = 12-1) ; (3) l'isolation par playlist (le bruit
// Slayer ne pollue pas Arena) ; (4) un joueur sans saison précédente → nil.
func TestGetWorldPlayerSeasonStats_DerivedAndInterSeason(t *testing.T) {
	shared := openMemDB(t)
	applyWorldPlayerStatsMigration(t, shared.SQLDb())
	ctx := context.Background()

	const arena = "edfef3ac-9cbe-4fa2-b949-8f29deafd483"
	const slayer = "dcb2e24e-05fb-4390-8076-32a0cdb4326e"
	fixture := []domain.WorldPlayerSeasonStats{
		// Alpha/Arena : joue 12-1 et 13-2 mais PAS 13-1 → LAG doit sauter 13-1.
		{Gamertag: "Alpha", SeasonID: "csrseason12-1", PlaylistID: arena, MatchCount: 10, WinCount: 4, Kills: 100, Deaths: 100, Assists: 30, KDA: 11, Accuracy: 480, DamageDealt: 50000, DamageTaken: 49000},
		{Gamertag: "Alpha", SeasonID: "csrseason13-2", PlaylistID: arena, MatchCount: 10, WinCount: 7, Kills: 150, Deaths: 100, Assists: 30, PlaytimeSec: 6000, KDA: 16, Accuracy: 700, DamageDealt: 70000, DamageTaken: 60000},
		// Bruit : Alpha/Slayer en 13-2 (autre playlist) — ne doit PAS polluer Arena.
		{Gamertag: "Alpha", SeasonID: "csrseason13-2", PlaylistID: slayer, MatchCount: 5, WinCount: 1, Kills: 50, Deaths: 80, Assists: 10, KDA: 5},
		// Beta : nouveau en 13-2/Arena → pas de saison précédente.
		{Gamertag: "Beta", SeasonID: "csrseason13-2", PlaylistID: arena, MatchCount: 8, WinCount: 4, Kills: 80, Deaths: 80, Assists: 24, KDA: 8},
	}
	if _, err := InsertPlayerSeasonStats(ctx, shared.SQLDb(), fixture); err != nil {
		t.Fatalf("InsertPlayerSeasonStats: %v", err)
	}

	repo := NewLeaderboardRepo(&PlayerDB{Shared: shared})
	stats, err := repo.GetWorldPlayerSeasonStats(ctx, "csrseason13-2", arena)
	if err != nil {
		t.Fatalf("GetWorldPlayerSeasonStats: %v", err)
	}
	byGT := map[string]domain.WorldPlayerSeasonStats{}
	for _, s := range stats {
		byGT[s.Gamertag] = s
	}
	if len(byGT) != 2 {
		t.Fatalf("attendu 2 joueurs (Alpha, Beta) en 13-2/Arena, got %d : %+v", len(byGT), stats)
	}

	a := byGT["Alpha"]
	if a.WinRate == nil || *a.WinRate < 0.69 || *a.WinRate > 0.71 {
		t.Errorf("Alpha win_rate = %v, want ~0.70", derefF(a.WinRate))
	}
	// KDA / accuracy / dégâts : valeurs natives BRUTES sommées (pas de dérivation).
	if a.KDA < 15.9 || a.KDA > 16.1 {
		t.Errorf("Alpha kda (brut sommé) = %v, want 16", a.KDA)
	}
	if a.Accuracy < 699 || a.Accuracy > 701 {
		t.Errorf("Alpha accuracy (brut sommé) = %v, want 700", a.Accuracy)
	}
	if a.DamageDealt != 70000 || a.DamageTaken != 60000 {
		t.Errorf("Alpha dégâts = %d/%d, want 70000/60000", a.DamageDealt, a.DamageTaken)
	}
	if a.KillsPerMin == nil || *a.KillsPerMin < 1.49 || *a.KillsPerMin > 1.51 {
		t.Errorf("Alpha kills_per_min = %v, want ~1.50 (150 / (6000/60))", derefF(a.KillsPerMin))
	}
	if a.PrevSeasonID == nil || *a.PrevSeasonID != "csrseason12-1" {
		t.Errorf("Alpha prev_season = %v, want csrseason12-1 (LAG saute 13-1)", derefStr(a.PrevSeasonID))
	}
	if a.KDATrend == nil || *a.KDATrend != "up" {
		t.Errorf("Alpha kda_trend = %v, want up (kda brut 16 > 11)", derefStr(a.KDATrend))
	}
	if a.WinRateTrend == nil || *a.WinRateTrend != "up" {
		t.Errorf("Alpha win_rate_trend = %v, want up (0.70 > 0.40)", derefStr(a.WinRateTrend))
	}

	b := byGT["Beta"]
	if b.PrevSeasonID != nil {
		t.Errorf("Beta prev_season = %q, want nil (nouveau joueur)", *b.PrevSeasonID)
	}
	if b.KDATrend != nil || b.WinRateTrend != nil {
		t.Errorf("Beta trends = (%v, %v), want nil/nil", derefStr(b.KDATrend), derefStr(b.WinRateTrend))
	}
	if b.WinRate == nil || *b.WinRate < 0.49 || *b.WinRate > 0.51 {
		t.Errorf("Beta win_rate = %v, want ~0.50", derefF(b.WinRate))
	}
}

func derefF(p *float64) float64 {
	if p == nil {
		return -1
	}
	return *p
}

// TestGetCSRWorldLeaderboard_Enrichment valide le merge dans GetCSRWorldLeaderboard :
// stats enrichies fusionnées par gamertag + RankDelta (rang saison N vs N-1).
func TestGetCSRWorldLeaderboard_Enrichment(t *testing.T) {
	shared := openMemDB(t)
	applyWorldLeaderboardMigration(t, shared.SQLDb())
	applyWorldPlayerStatsMigration(t, shared.SQLDb())
	ctx := context.Background()

	const arena = "edfef3ac-9cbe-4fa2-b949-8f29deafd483"
	t0 := time.Now().UTC()
	// Saison précédente (13-1) : Alpha rang 5.
	if _, err := InsertWorldCSRSnapshot(ctx, shared.SQLDb(), "halo_infinite", []domain.LeaderboardEntry{
		{Season: "csrseason13-1", Playlist: arena, Rank: 5, Gamertag: "Alpha", CSRValue: 1800, Tier: "Diamond", FetchedAt: t0},
	}); err != nil {
		t.Fatalf("snapshot prev: %v", err)
	}
	// Saison courante (13-2) : Alpha rang 2 (a grimpé de 3).
	if _, err := InsertWorldCSRSnapshot(ctx, shared.SQLDb(), "halo_infinite", []domain.LeaderboardEntry{
		{Season: "csrseason13-2", Playlist: arena, Rank: 2, Gamertag: "Alpha", CSRValue: 2000, Tier: "Onyx", FetchedAt: t0.Add(time.Hour)},
	}); err != nil {
		t.Fatalf("snapshot cur: %v", err)
	}
	// Stats enrichies pour Alpha en 13-2/Arena (valeurs natives brutes incluses).
	if _, err := InsertPlayerSeasonStats(ctx, shared.SQLDb(), []domain.WorldPlayerSeasonStats{
		{Gamertag: "Alpha", SeasonID: "csrseason13-2", PlaylistID: arena, MatchCount: 10, WinCount: 7, Kills: 150, Deaths: 100, Assists: 30, KDA: 16, Accuracy: 700, DamageDealt: 70000, DamageTaken: 60000},
	}); err != nil {
		t.Fatalf("enriched: %v", err)
	}

	repo := NewLeaderboardRepo(&PlayerDB{Shared: shared})
	entries, err := repo.GetCSRWorldLeaderboard(ctx, "halo_infinite", "csrseason13-2", arena, 100)
	if err != nil {
		t.Fatalf("GetCSRWorldLeaderboard: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("attendu 1 entrée, got %d", len(entries))
	}
	e := entries[0]
	if e.Gamertag != "Alpha" || e.Rank != 2 {
		t.Fatalf("entrée = (%q, rang %d), want (Alpha, 2)", e.Gamertag, e.Rank)
	}
	if e.WinRate == nil || *e.WinRate < 0.69 || *e.WinRate > 0.71 {
		t.Errorf("win_rate = %v, want ~0.70", derefF(e.WinRate))
	}
	if e.KDA == nil || *e.KDA < 15.9 || *e.KDA > 16.1 {
		t.Errorf("kda (brut sommé) = %v, want 16", derefF(e.KDA))
	}
	if e.Accuracy == nil || *e.Accuracy < 699 || *e.Accuracy > 701 {
		t.Errorf("accuracy (brut sommé) = %v, want 700", derefF(e.Accuracy))
	}
	if e.DamageDealt == nil || *e.DamageDealt != 70000 {
		t.Errorf("damage_dealt = %v, want 70000", e.DamageDealt)
	}
	if e.DamageTaken == nil || *e.DamageTaken != 60000 {
		t.Errorf("damage_taken = %v, want 60000", e.DamageTaken)
	}
	if e.MatchCount == nil || *e.MatchCount != 10 {
		t.Errorf("match_count = %v, want 10", e.MatchCount)
	}
	if e.RankDelta == nil || *e.RankDelta != 3 {
		t.Errorf("rank_delta = %v, want +3 (rang 5 -> 2)", e.RankDelta)
	}
}

// TestGetCSRWorldLeaderboard_PrevSeasonCrossDigit verrouille le tri NUMÉRIQUE des
// saisons : la saison précédente de csrseason10-1 est csrseason6-1 (rang 601 < 1001),
// que l'ancien tri LEXICOGRAPHIQUE ratait (csrseason6-1 > csrseason10-1 → RankDelta nil).
// Couvre aussi le total de matchs CUMULÉ (6-1 + 10-1).
func TestGetCSRWorldLeaderboard_PrevSeasonCrossDigit(t *testing.T) {
	shared := openMemDB(t)
	applyWorldLeaderboardMigration(t, shared.SQLDb())
	applyWorldPlayerStatsMigration(t, shared.SQLDb())
	ctx := context.Background()

	const arena = "edfef3ac-9cbe-4fa2-b949-8f29deafd483"
	t0 := time.Now().UTC()
	if _, err := InsertWorldCSRSnapshot(ctx, shared.SQLDb(), "halo_infinite", []domain.LeaderboardEntry{
		{Season: "csrseason6-1", Playlist: arena, Rank: 10, Gamertag: "Alpha", CSRValue: 1700, Tier: "Diamond", FetchedAt: t0},
		{Season: "csrseason10-1", Playlist: arena, Rank: 4, Gamertag: "Alpha", CSRValue: 1950, Tier: "Onyx", FetchedAt: t0.Add(time.Hour)},
	}); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if _, err := InsertPlayerSeasonStats(ctx, shared.SQLDb(), []domain.WorldPlayerSeasonStats{
		{Gamertag: "Alpha", SeasonID: "csrseason6-1", PlaylistID: arena, MatchCount: 20, WinCount: 10, Kills: 200, Deaths: 150, Assists: 40, KDA: 30},
		{Gamertag: "Alpha", SeasonID: "csrseason10-1", PlaylistID: arena, MatchCount: 12, WinCount: 8, Kills: 150, Deaths: 90, Assists: 30, KDA: 24},
	}); err != nil {
		t.Fatalf("enriched: %v", err)
	}

	repo := NewLeaderboardRepo(&PlayerDB{Shared: shared})
	entries, err := repo.GetCSRWorldLeaderboard(ctx, "halo_infinite", "csrseason10-1", arena, 100)
	if err != nil {
		t.Fatalf("GetCSRWorldLeaderboard: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("attendu 1 entrée, got %d", len(entries))
	}
	e := entries[0]
	// Δrang : prev = csrseason6-1 (rang 10) → csrseason10-1 (rang 4) = +6.
	if e.RankDelta == nil || *e.RankDelta != 6 {
		t.Errorf("rank_delta = %v, want +6 (csrseason6-1 rang 10 -> csrseason10-1 rang 4)", e.RankDelta)
	}
	if e.MatchCount == nil || *e.MatchCount != 12 {
		t.Errorf("match_count = %v, want 12 (saison affichée)", e.MatchCount)
	}
	// Cumulé sur les saisons <= 10-1 (6-1: 20 + 10-1: 12).
	if e.CumulativeMatchCount == nil || *e.CumulativeMatchCount != 32 {
		t.Errorf("cumulative_match_count = %v, want 32 (20 + 12)", e.CumulativeMatchCount)
	}
}

// TestWorldSeasonPlayers_TopNPerPlaylist valide le cap top-N PAR playlist :
// un joueur hors du top-N d'une playlist mais DANS le top-N d'une autre reste
// inclus (sémantique par playlist = ce qu'affiche le classement) ; un joueur hors
// top-N partout est exclu ; topN <= 0 = aucun cap. Vérifie aussi que le xuid scrapé
// est remonté (dédup par gamertag via MAX(xuid), B1).
func TestWorldSeasonPlayers_TopNPerPlaylist(t *testing.T) {
	shared := openMemDB(t)
	applyWorldLeaderboardMigration(t, shared.SQLDb())
	ctx := context.Background()

	const arena = "edfef3ac-9cbe-4fa2-b949-8f29deafd483"
	const slayer = "dcb2e24e-05fb-4390-8076-32a0cdb4326e"
	t0 := time.Now().UTC()
	if _, err := InsertWorldCSRSnapshot(ctx, shared.SQLDb(), "halo_infinite", []domain.LeaderboardEntry{
		{Season: "csrseason13-2", Playlist: arena, Rank: 1, Gamertag: "Alpha", XUID: "2535000000000001", CSRValue: 2000, Tier: "Onyx", FetchedAt: t0},
		{Season: "csrseason13-2", Playlist: arena, Rank: 150, Gamertag: "Beta", XUID: "2535000000000002", CSRValue: 1200, Tier: "Diamond", FetchedAt: t0},
		{Season: "csrseason13-2", Playlist: arena, Rank: 120, Gamertag: "Charlie", XUID: "2535000000000003", CSRValue: 1300, Tier: "Diamond", FetchedAt: t0},
		// Beta est top-100 d'une AUTRE playlist (rang 5) → doit rester inclus malgré son rang 150 en arena.
		{Season: "csrseason13-2", Playlist: slayer, Rank: 5, Gamertag: "Beta", XUID: "2535000000000002", CSRValue: 1900, Tier: "Onyx", FetchedAt: t0},
	}); err != nil {
		t.Fatalf("InsertWorldCSRSnapshot: %v", err)
	}

	gts := func(players []domain.WorldPlayerRef) []string {
		out := make([]string, len(players))
		for i, p := range players {
			out[i] = p.Gamertag
		}
		return out
	}

	top100, err := WorldSeasonPlayers(ctx, shared.SQLDb(), "csrseason13-2", 100)
	if err != nil {
		t.Fatalf("WorldSeasonPlayers(top100): %v", err)
	}
	// Alpha (arena rang 1) + Beta (slayer rang 5) ; Charlie (arena rang 120, nulle part ailleurs) exclu.
	if !reflect.DeepEqual(gts(top100), []string{"Alpha", "Beta"}) {
		t.Errorf("top100 = %v, want [Alpha Beta] (Charlie >100 partout exclu, Beta inclus via slayer)", gts(top100))
	}
	// Le xuid scrapé du snapshot est remonté (alimente le court-circuit PeopleHub, B1) —
	// Beta n'apparaît qu'UNE fois malgré deux playlists (dédup GROUP BY gamertag).
	for _, p := range top100 {
		if p.XUID == "" {
			t.Errorf("xuid manquant pour %s (attendu depuis le snapshot)", p.Gamertag)
		}
	}

	all, err := WorldSeasonPlayers(ctx, shared.SQLDb(), "csrseason13-2", 0)
	if err != nil {
		t.Fatalf("WorldSeasonPlayers(all): %v", err)
	}
	if !reflect.DeepEqual(gts(all), []string{"Alpha", "Beta", "Charlie"}) {
		t.Errorf("all (topN=0) = %v, want [Alpha Beta Charlie]", gts(all))
	}
}
