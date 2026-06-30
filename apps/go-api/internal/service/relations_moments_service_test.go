package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/relations"
	"levelup/go-api/internal/domain"
)

func TestGetRelationsMoments_HeatmapHours(t *testing.T) {
	// 2 relations, comptes agrégés par heure (0..23). Deux lignes sur la même
	// heure (Foe à 02h ×3 et... non) : ici une ligne par (xuid, heure).
	// Foe : 02h ×3, 03h ×2, 19h ×4. Ally : 09h ×6.
	repo := &mockRelationsRepo{
		heatmapRows: []domain.RelationHeatmapRawRow{
			{XUID: "x1", Gamertag: "Foe", Hour: 2, Count: 3},
			{XUID: "x1", Gamertag: "Foe", Hour: 3, Count: 2},
			{XUID: "x1", Gamertag: "Foe", Hour: 19, Count: 4},
			{XUID: "x2", Gamertag: "Ally", Hour: 9, Count: 6},
		},
	}
	svc := NewRelationsService(repo)
	out, err := svc.GetRelationsMoments(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if repo.heatmapTopN != momentsHeatmapTopN {
		t.Fatalf("topN=%d want %d", repo.heatmapTopN, momentsHeatmapTopN)
	}
	if out.TopRelations != momentsHeatmapTopN {
		t.Fatalf("TopRelations=%d want %d", out.TopRelations, momentsHeatmapTopN)
	}

	got := map[string]int{} // "gamertag/hour" → count
	for _, c := range out.Heatmap {
		got[c.Gamertag+"/"+itoa(c.Hour)] = c.Count
	}
	if got["Foe/2"] != 3 {
		t.Fatalf("Foe 02h=%d want 3 (heatmap=%+v)", got["Foe/2"], out.Heatmap)
	}
	if got["Foe/3"] != 2 {
		t.Fatalf("Foe 03h=%d want 2", got["Foe/3"])
	}
	if got["Foe/19"] != 4 {
		t.Fatalf("Foe 19h=%d want 4", got["Foe/19"])
	}
	if got["Ally/9"] != 6 {
		t.Fatalf("Ally 09h=%d want 6", got["Ally/9"])
	}
}

func TestGetRelationsMoments_TopRivalsAndRevanche(t *testing.T) {
	rows := []domain.RelationRawRow{
		{XUID: "rivalA", Gamertag: "RivalA", EnemyCount: 20},
		{XUID: "rivalB", Gamertag: "RivalB", EnemyCount: 10},
		{XUID: "low", Gamertag: "LowFoe", EnemyCount: 2}, // sous le seuil
		{XUID: "ally", Gamertag: "Ally", EnemyCount: 0},  // pas un rival
	}
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	timelines := map[string][]domain.RelationDuelRawRow{
		"rivalA": {
			{MatchID: "m1", StartTime: now, Result: relations.ResultLoss, KillsOnRival: 1, DeathsByRival: 5},
			{MatchID: "m2", StartTime: now.Add(time.Hour), Result: relations.ResultWin, KillsOnRival: 6, DeathsByRival: 2},
		},
		"rivalB": {
			{MatchID: "m3", StartTime: now, Result: relations.ResultWin, KillsOnRival: 4, DeathsByRival: 4},
		},
	}
	repo := &mockRelationsRepo{rows: rows, timelineByXUID: timelines}
	svc := NewRelationsService(repo)
	out, err := svc.GetRelationsMoments(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if repo.timelineLimit != momentsTimelineLimit {
		t.Fatalf("timelineLimit=%d want %d", repo.timelineLimit, momentsTimelineLimit)
	}
	// 2 rivaux retenus (low sous seuil, ally non-rival), triés EnemyCount DESC.
	if len(out.Rivalries) != 2 {
		t.Fatalf("rivalries len=%d want 2 (%+v)", len(out.Rivalries), out.Rivalries)
	}
	if out.Rivalries[0].Gamertag != "RivalA" || out.Rivalries[1].Gamertag != "RivalB" {
		t.Fatalf("rival order=%v/%v want RivalA/RivalB", out.Rivalries[0].Gamertag, out.Rivalries[1].Gamertag)
	}
	a := out.Rivalries[0]
	if len(a.Duels) != 2 {
		t.Fatalf("RivalA duels=%d want 2", len(a.Duels))
	}
	// FragGap = (1-5)+(6-2) = 0 ; série en cours = 1 victoire (dernier=win).
	if a.FragGap != 0 {
		t.Fatalf("RivalA FragGap=%d want 0", a.FragGap)
	}
	if a.CurrentStreak != 1 {
		t.Fatalf("RivalA streak=%d want 1", a.CurrentStreak)
	}
	if a.Duels[0].Outcome != "loss" || a.Duels[1].Outcome != "win" {
		t.Fatalf("RivalA duel outcomes=%v/%v", a.Duels[0].Outcome, a.Duels[1].Outcome)
	}
	if a.RollingWindow != relations.RollingWinRateWindow {
		t.Fatalf("RollingWindow=%d want %d", a.RollingWindow, relations.RollingWinRateWindow)
	}
}
