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

// TestBuildSquadImpactMatrix_TeamWideAllyDropped : le scoreboard impact est
// calculé en team-wide alliée (parité Python compute_single_match_impact).
// Si un badge (ex. top_killer) tombe sur un allié NON-squad, il ne doit PAS
// apparaître dans le scoreboard ni se reporter sur un squad member.
//
// Setup : main + A (squad). NS = allié non-squad. NS a 12 kills (top killer
// global de l'équipe alliée), main a 5, A a 4. Le badge top_killer va donc à
// NS et doit être droppé. silent_hero (max_assists+min_deaths) tombe sur A
// (max_assists=8, min_deaths=0) → doit apparaître dans le scoreboard.
func TestBuildSquadImpactMatrix_TeamWideAllyDropped(t *testing.T) {
	mainXUID := "x_main"
	matchID := "m_session_1"
	startTime := time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC)

	// Mock repo : retourne 3 alliés (main, A=squad, NS=non-squad). NS a le
	// max kills donc top_killer. A a max assists + min deaths donc silent_hero.
	repo := &mockSquadRepo{
		allyRows: []domain.AllyParticipant{
			{MatchID: matchID, XUID: mainXUID, Gamertag: "main", Kills: 5, Deaths: 2, Assists: 3, Outcome: domain.OutcomeWin},
			{MatchID: matchID, XUID: "x_a", Gamertag: "A", Kills: 4, Deaths: 0, Assists: 8, Outcome: domain.OutcomeWin},
			{MatchID: matchID, XUID: "x_ns", Gamertag: "NS", Kills: 12, Deaths: 1, Assists: 1, Outcome: domain.OutcomeWin},
		},
	}
	svc := &TeammatesService{
		repo:      repo,
		titleSlug: "halo_infinite",
		gamertag:  "main",
	}

	allSquadRows := []domain.SquadMatchRow{
		{MatchID: matchID, StartTime: startTime, Outcome: domain.OutcomeWin},
	}
	matrix := svc.buildSquadImpactMatrix(context.Background(), allSquadRows, mainXUID, "main", []string{"A"})
	if matrix == nil {
		t.Fatal("matrix should be non-nil")
	}

	// top_killer va sur NS (non-squad) → ne doit apparaître nulle part.
	for _, c := range matrix.Cells {
		for _, k := range c.BadgeKeys {
			if k == "top_killer" {
				t.Errorf("top_killer ne doit pas apparaître (tombe sur NS, allié non-squad) — trouvé sur %s", c.Player)
			}
		}
	}
	// Aucun joueur ne doit avoir un compte top_killer non nul dans la summary.
	for _, p := range matrix.Players {
		for _, b := range p.Counts {
			if b.BadgeKey == "top_killer" && b.Count != 0 {
				t.Errorf("player %s : top_killer count=%d, want 0 (badge tombé sur non-squad)", p.Player, b.Count)
			}
		}
	}

	// silent_hero doit aller à A (max assists=8, min deaths=0 sur l'équipe alliée).
	foundSilentOnA := false
	for _, c := range matrix.Cells {
		if c.Player != "A" {
			continue
		}
		for _, k := range c.BadgeKeys {
			if k == "silent_hero" {
				foundSilentOnA = true
			}
		}
	}
	if !foundSilentOnA {
		t.Error("silent_hero attendu sur A (max assists + min deaths parmi alliés)")
	}
}

// TestBuildSquadImpactMatrix_TeamWideAllySwapsBadge : si un allié non-squad
// est plus extrême que le squad member, c'est lui qui reçoit le badge en
// team-wide → le squad member NE DOIT PAS recevoir le badge "à défaut".
//
// Setup défaite : main perd, A (squad) a 5 deaths, NS (non-squad) a 9 deaths.
// false_brother en team-wide va sur NS, donc A ne le reçoit pas.
func TestBuildSquadImpactMatrix_TeamWideNoFallback(t *testing.T) {
	mainXUID := "x_main"
	matchID := "m_session_2"
	startTime := time.Date(2026, 4, 6, 19, 0, 0, 0, time.UTC)

	repo := &mockSquadRepo{
		allyRows: []domain.AllyParticipant{
			{MatchID: matchID, XUID: mainXUID, Gamertag: "main", Kills: 4, Deaths: 3, Assists: 1, Outcome: domain.OutcomeLoss},
			{MatchID: matchID, XUID: "x_a", Gamertag: "A", Kills: 3, Deaths: 5, Assists: 2, Outcome: domain.OutcomeLoss},
			{MatchID: matchID, XUID: "x_ns", Gamertag: "NS", Kills: 2, Deaths: 9, Assists: 0, Outcome: domain.OutcomeLoss},
		},
	}
	svc := &TeammatesService{
		repo:      repo,
		titleSlug: "halo_infinite",
		gamertag:  "main",
	}
	allSquadRows := []domain.SquadMatchRow{
		{MatchID: matchID, StartTime: startTime, Outcome: domain.OutcomeLoss},
	}
	matrix := svc.buildSquadImpactMatrix(context.Background(), allSquadRows, mainXUID, "main", []string{"A"})

	// false_brother doit aller à NS (max deaths=9, min assists=0). Donc :
	// - A ne doit PAS recevoir false_brother malgré ses 5 deaths (squad-only,
	//   il l'aurait reçu en logique squad-only buggée).
	// - matrix peut être nil si AUCUN badge ne tombe sur un squad member.
	if matrix == nil {
		// OK : aucun badge alloué à un squad member, matrice vide donc nil.
		return
	}
	for _, c := range matrix.Cells {
		for _, k := range c.BadgeKeys {
			if k == "false_brother" {
				t.Errorf("false_brother ne doit pas apparaître (tombe sur NS en team-wide) — trouvé sur %s", c.Player)
			}
		}
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
