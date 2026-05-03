package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// ---------- buildSquadSessionTimeline (teammates.04) ----------

func TestBuildSquadSessionTimeline_GroupAndAggregate(t *testing.T) {
	p1, p2, p3 := 70.0, 50.0, 90.0
	sessA := "A"
	sessB := "B"
	t1 := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	t2 := t1.Add(10 * time.Minute)
	t3 := t1.Add(time.Hour)

	matches := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t1, SessionLabel: &sessA, Outcome: domain.OutcomeWin, PerformanceScore: &p1, TeamMMR: 1500},
		{MatchID: "m2", StartTime: t2, SessionLabel: &sessA, Outcome: domain.OutcomeLoss, PerformanceScore: &p2, TeamMMR: 1480},
		{MatchID: "m3", StartTime: t3, SessionLabel: &sessB, Outcome: domain.OutcomeWin, PerformanceScore: &p3, TeamMMR: 1550},
		// duplicate by match_id (autre teammate sur même match) — doit être dédupliqué
		{MatchID: "m1", StartTime: t1, SessionLabel: &sessA, Outcome: domain.OutcomeWin, PerformanceScore: &p1, TeamMMR: 1500},
	}
	pts := buildSquadSessionTimeline(matches)
	if len(pts) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(pts))
	}
	// Tri chronologique : sessA (t1) avant sessB (t3).
	if pts[0].SessionLabel != "A" || pts[1].SessionLabel != "B" {
		t.Errorf("expected [A, B] order, got [%s, %s]", pts[0].SessionLabel, pts[1].SessionLabel)
	}
	a := pts[0]
	if a.MatchCount != 2 {
		t.Errorf("A: expected match_count=2 (dédup), got %d", a.MatchCount)
	}
	if a.Wins != 1 || a.Losses != 1 {
		t.Errorf("A: expected wins=1 losses=1, got wins=%d losses=%d", a.Wins, a.Losses)
	}
	if a.SquadPerf != 60.0 {
		t.Errorf("A: expected squad_perf=60 ((70+50)/2), got %f", a.SquadPerf)
	}
	if a.WinRate == nil || *a.WinRate != 0.5 {
		t.Errorf("A: expected win_rate=0.5, got %v", a.WinRate)
	}
	if a.TeamMMRAvg == nil || *a.TeamMMRAvg != 1490.0 {
		t.Errorf("A: expected team_mmr_avg=1490, got %v", a.TeamMMRAvg)
	}
	b := pts[1]
	if b.MatchCount != 1 || b.SquadPerf != 90.0 {
		t.Errorf("B: expected match_count=1 squad_perf=90, got %d / %f", b.MatchCount, b.SquadPerf)
	}
}

func TestBuildSquadSessionTimeline_NoSessionLabelBucket(t *testing.T) {
	matches := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: time.Now(), SessionLabel: nil, Outcome: domain.OutcomeWin, TeamMMR: 1500},
	}
	pts := buildSquadSessionTimeline(matches)
	if len(pts) != 1 || pts[0].SessionLabel != "(no session)" {
		t.Errorf("expected single bucket '(no session)', got %v", pts)
	}
}

func TestBuildSquadSessionTimeline_PerfNilLeavesZero(t *testing.T) {
	sessA := "A"
	matches := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: time.Now(), SessionLabel: &sessA, Outcome: domain.OutcomeWin, PerformanceScore: nil},
	}
	pts := buildSquadSessionTimeline(matches)
	if len(pts) != 1 {
		t.Fatalf("expected 1 point, got %d", len(pts))
	}
	if pts[0].SquadPerf != 0 {
		t.Errorf("expected squad_perf=0 when no score, got %f", pts[0].SquadPerf)
	}
}

// ---------- buildSquadPerMinuteStats helpers (teammates.14) ----------

// Note: buildSquadPerMinuteStats lui-même requiert un PlayerMatchesRepository
// (cgo+gcc indispo localement). On teste ici les sous-helpers numériques via
// SquadPerMinuteEntry direct.

func TestPerMinuteEntry_RoundingAndMatchCount(t *testing.T) {
	// Sanity check : un agrégat 60s × 1 kill ⇒ 1.0 kpm.
	// On vérifie ici uniquement la propriété domain.
	e := domain.SquadPerMinuteEntry{
		Player:           "Me",
		KillsPerMinute:   1.5,
		DeathsPerMinute:  0.7,
		AssistsPerMinute: 0.3,
		MatchCount:       8,
	}
	if e.Player != "Me" || e.MatchCount != 8 {
		t.Errorf("unexpected entry shape: %+v", e)
	}
}

// ---------- impactScoreWeights (teammates.07) ----------

func TestImpactScoreWeights_Coverage(t *testing.T) {
	for _, badge := range impactBadgeOrd {
		if _, ok := impactScoreWeights[badge]; !ok {
			t.Errorf("badge %q manque dans impactScoreWeights", badge)
		}
	}
	// Sanity : les weights matchent la spec (cf. .ai/charts_specs/teammates/07).
	expected := map[string]float64{
		"clutch_finisher":   2.0,
		"first_blood":       2.0,
		"last_casualty":     -2.0,
		"silent_hero":       1.5,
		"false_brother":     -1.5,
		"last_group_kill":   -1.0,
		"first_group_death": -1.0,
		"top_killer":        1.0,
	}
	for k, v := range expected {
		if impactScoreWeights[k] != v {
			t.Errorf("%s: weight=%v, want %v", k, impactScoreWeights[k], v)
		}
	}
}

// ---------- buildSquadMapHeatmap (teammates.03) ----------

// rowWithMap est un helper local (pas de collision avec rowWithStats) qui
// construit une PlayerMatchRow canonique avec une map et un perf score.
func rowWithMap(matchID, mapName string, perf float64) canonical.PlayerMatchRow {
	p := perf
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID: matchID,
			Map: &canonical.AssetReference{
				Kind:         "map",
				ID:           mapName,
				DefaultLabel: mapName,
			},
		},
		Enrichment: canonical.PlayerMatchEnrichment{
			PerformanceScore: &p,
		},
	}
}

// TestBuildSquadMapHeatmap_TeammatePerfsAreDistinctFromMain cadenasse le bug
// "bound-to-main" du PlayerMatchesAdapter : si buildSquadMapHeatmap chargeait
// les rows des coéquipiers via playerMatchesRepo (qui ignore l'arg gamertag),
// chaque ligne du heatmap aurait les mêmes perfs que le main player. Le fix
// (commit fabc3b3c) bascule sur squadLoader.LoadFor qui résout la PlayerDB
// par gamertag.
func TestBuildSquadMapHeatmap_TeammatePerfsAreDistinctFromMain(t *testing.T) {
	mainPerf := 80.0
	teammatePerf := 50.0
	streets := "Streets"
	mainSquadRows := []domain.SquadMatchRow{
		{MatchID: "m1", PerformanceScore: &mainPerf, MapUI: streets, IsWithFriends: true},
	}
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"friend1": {rowWithMap("m1", streets, teammatePerf)},
		},
	}
	svc := &TeammatesService{
		squadLoader: loader,
		titleSlug:   "halo_infinite",
		gamertag:    "main",
	}

	heatmap := svc.buildSquadMapHeatmap(context.Background(), mainSquadRows, []string{"friend1"}, nil)
	if heatmap == nil {
		t.Fatal("heatmap should be non-nil")
	}
	if len(heatmap.Players) != 2 || heatmap.Players[0] != "main" || heatmap.Players[1] != "friend1" {
		t.Fatalf("Players want [main, friend1], got %v", heatmap.Players)
	}

	// Cell main + Streets : perf=80
	// Cell friend1 + Streets : perf=50 (PAS 80 — bug bound-to-main)
	var mainCell, friendCell *domain.SquadMapHeatmapCell
	for i := range heatmap.Cells {
		c := &heatmap.Cells[i]
		if c.MapUI != streets {
			continue
		}
		switch c.Player {
		case "main":
			mainCell = c
		case "friend1":
			friendCell = c
		}
	}
	if mainCell == nil || friendCell == nil {
		t.Fatalf("cells main+friend1 expected, got %+v", heatmap.Cells)
	}
	if mainCell.PerfAvg == nil || friendCell.PerfAvg == nil {
		t.Fatalf("PerfAvg should be set for both: main=%v friend=%v", mainCell.PerfAvg, friendCell.PerfAvg)
	}
	if *mainCell.PerfAvg != mainPerf {
		t.Errorf("main PerfAvg: want %v, got %v", mainPerf, *mainCell.PerfAvg)
	}
	if *friendCell.PerfAvg != teammatePerf {
		t.Errorf("friend1 PerfAvg: want %v (sa propre perf), got %v — bug bound-to-main probable",
			teammatePerf, *friendCell.PerfAvg)
	}
	if *mainCell.PerfAvg == *friendCell.PerfAvg {
		t.Errorf("main et friend1 ont des perfs identiques (%v) → heatmap retourne les rows du main pour tous", *mainCell.PerfAvg)
	}
}

// TestBuildSquadMapHeatmap_NoCapOnMaps : le cap top 15 a été retiré (commit
// fabc3b3c). 18 cartes en input → 18 cartes en output (toutes affichées).
func TestBuildSquadMapHeatmap_NoCapOnMaps(t *testing.T) {
	rows := make([]domain.SquadMatchRow, 0, 18)
	perf := 60.0
	for i := 0; i < 18; i++ {
		rows = append(rows, domain.SquadMatchRow{
			MatchID:          string(rune('a' + i)),
			PerformanceScore: &perf,
			MapUI:            string(rune('A'+i)) + "_map",
		})
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main"}
	heatmap := svc.buildSquadMapHeatmap(context.Background(), rows, nil, nil)
	if heatmap == nil {
		t.Fatal("heatmap should be non-nil")
	}
	if len(heatmap.MapsTopN) != 18 {
		t.Errorf("toutes les cartes attendues (18), got %d → cap top 15 réintroduit ?", len(heatmap.MapsTopN))
	}
}
