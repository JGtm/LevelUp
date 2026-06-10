//go:build integration

package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

// applyWorldPlayerStatsMigration applique la migration create_world_player_season_stats.
func applyWorldPlayerStatsMigration(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, m := range migration.ForTarget(migration.TargetShared) {
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
		{Gamertag: "Alpha", SeasonID: "csrseason12-1", PlaylistID: arena, MatchCount: 10, WinCount: 4, Kills: 100, Deaths: 100, Assists: 30},
		{Gamertag: "Alpha", SeasonID: "csrseason13-2", PlaylistID: arena, MatchCount: 10, WinCount: 7, Kills: 150, Deaths: 100, Assists: 30, PlaytimeSec: 6000},
		// Bruit : Alpha/Slayer en 13-2 (autre playlist) — ne doit PAS polluer Arena.
		{Gamertag: "Alpha", SeasonID: "csrseason13-2", PlaylistID: slayer, MatchCount: 5, WinCount: 1, Kills: 50, Deaths: 80, Assists: 10},
		// Beta : nouveau en 13-2/Arena → pas de saison précédente.
		{Gamertag: "Beta", SeasonID: "csrseason13-2", PlaylistID: arena, MatchCount: 8, WinCount: 4, Kills: 80, Deaths: 80, Assists: 24},
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
	if a.KDA == nil || *a.KDA < 1.59 || *a.KDA > 1.61 {
		t.Errorf("Alpha kda = %v, want ~1.60", derefF(a.KDA))
	}
	if a.KillsPerMin == nil || *a.KillsPerMin < 1.49 || *a.KillsPerMin > 1.51 {
		t.Errorf("Alpha kills_per_min = %v, want ~1.50 (150 / (6000/60))", derefF(a.KillsPerMin))
	}
	if a.PrevSeasonID == nil || *a.PrevSeasonID != "csrseason12-1" {
		t.Errorf("Alpha prev_season = %v, want csrseason12-1 (LAG saute 13-1)", derefStr(a.PrevSeasonID))
	}
	if a.KDATrend == nil || *a.KDATrend != "up" {
		t.Errorf("Alpha kda_trend = %v, want up (1.60 > 1.10)", derefStr(a.KDATrend))
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
