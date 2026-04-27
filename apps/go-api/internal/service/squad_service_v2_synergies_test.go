package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/games/canonical"
)

func mkRowMap(matchID, mapID, mapLabel string, outcome canonical.Outcome, perf *float64) canonical.PlayerMatchRow {
	r := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID:      matchID,
			StartedAtUTC: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
			Outcome:      outcome,
			Map: &canonical.AssetReference{
				Kind:         "map",
				ID:           mapID,
				DefaultLabel: mapLabel,
			},
		},
		Self: canonical.MatchParticipant{Outcome: outcome},
	}
	if perf != nil {
		r.Enrichment.PerformanceScore = perf
	}
	return r
}

func fptrSyn(v float64) *float64 { return &v }

func TestBuildMapBreakdownLollipop_Empty(t *testing.T) {
	t.Parallel()
	got := BuildMapBreakdownLollipop(nil, 0)
	if len(got.Datapoints) != 0 {
		t.Errorf("nil input: want 0 dp, got %d", len(got.Datapoints))
	}
	if got.Key != "squad.synergies.map_breakdown" {
		t.Errorf("Key: %q", got.Key)
	}
}

func TestBuildMapBreakdownLollipop_TopByPlayed(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		// bazaar : 3 matchs (2W 1L)
		mkRowMap("m1", "bazaar", "Bazaar", canonical.OutcomeWin, nil),
		mkRowMap("m2", "bazaar", "Bazaar", canonical.OutcomeWin, nil),
		mkRowMap("m3", "bazaar", "Bazaar", canonical.OutcomeLoss, nil),
		// aquarius : 1 match
		mkRowMap("m4", "aquarius", "Aquarius", canonical.OutcomeTie, nil),
	}
	got := BuildMapBreakdownLollipop(rows, 0)
	if len(got.Datapoints) != 2 {
		t.Fatalf("want 2 maps, got %d", len(got.Datapoints))
	}
	// Bazaar (3 matchs) en premier, Aquarius (1) en second
	if got.Datapoints[0].Category != "Bazaar" {
		t.Errorf("first by Played want Bazaar, got %s", got.Datapoints[0].Category)
	}
	if got.Datapoints[0].Components["win"] != 2 || got.Datapoints[0].Components["loss"] != 1 {
		t.Errorf("Bazaar W/L want 2/1, got %v/%v",
			got.Datapoints[0].Components["win"], got.Datapoints[0].Components["loss"])
	}
}

func TestBuildMapBreakdownLollipop_LimitsApplied(t *testing.T) {
	t.Parallel()
	var rows []canonical.PlayerMatchRow
	for i := 0; i < 25; i++ {
		mapID := string(rune('a' + i%25))
		rows = append(rows, mkRowMap("m"+mapID, mapID, mapID, canonical.OutcomeWin, nil))
	}
	got := BuildMapBreakdownLollipop(rows, 5)
	if len(got.Datapoints) != 5 {
		t.Errorf("limit 5: got %d", len(got.Datapoints))
	}
}

func TestBuildBulletWinrate_SessionVsHistorical(t *testing.T) {
	t.Parallel()
	session := []canonical.PlayerMatchRow{
		mkRowMap("m1", "bazaar", "Bazaar", canonical.OutcomeWin, nil),
		mkRowMap("m2", "bazaar", "Bazaar", canonical.OutcomeLoss, nil),
	}
	historical := []canonical.PlayerMatchRow{
		mkRowMap("h1", "bazaar", "Bazaar", canonical.OutcomeWin, nil),
		mkRowMap("h2", "bazaar", "Bazaar", canonical.OutcomeWin, nil),
		mkRowMap("h3", "bazaar", "Bazaar", canonical.OutcomeWin, nil),
		mkRowMap("h4", "bazaar", "Bazaar", canonical.OutcomeLoss, nil),
	}
	got := BuildBulletWinrate(session, historical, 0)
	if len(got.Datapoints) != 1 {
		t.Fatalf("want 1 map, got %d", len(got.Datapoints))
	}
	dp := got.Datapoints[0]
	if dp.Components["session"] != 0.5 {
		t.Errorf("session winrate: want 0.5, got %v", dp.Components["session"])
	}
	if dp.Components["historical"] != 0.75 {
		t.Errorf("historical winrate: want 0.75, got %v", dp.Components["historical"])
	}
}

func TestBuildBulletWinrate_MapAbsentFromHistorical(t *testing.T) {
	t.Parallel()
	session := []canonical.PlayerMatchRow{
		mkRowMap("m1", "newmap", "NewMap", canonical.OutcomeWin, nil),
	}
	got := BuildBulletWinrate(session, nil, 0)
	if len(got.Datapoints) != 1 {
		t.Fatalf("want 1 map, got %d", len(got.Datapoints))
	}
	if _, hasHistorical := got.Datapoints[0].Components["historical"]; hasHistorical {
		t.Error("map absente d'historical : pas de cle 'historical' dans Components")
	}
}

func TestBuildPerfVsHistorical_DeltaCorrect(t *testing.T) {
	t.Parallel()
	session := []canonical.PlayerMatchRow{
		mkRowMap("m1", "bazaar", "Bazaar", canonical.OutcomeWin, fptrSyn(80)),
		mkRowMap("m2", "bazaar", "Bazaar", canonical.OutcomeWin, fptrSyn(90)),
	}
	historical := []canonical.PlayerMatchRow{
		mkRowMap("h1", "bazaar", "Bazaar", canonical.OutcomeWin, fptrSyn(60)),
		mkRowMap("h2", "bazaar", "Bazaar", canonical.OutcomeLoss, fptrSyn(50)),
	}
	got := BuildPerfVsHistorical(session, historical, 0)
	if len(got.Datapoints) != 1 {
		t.Fatalf("want 1 dp, got %d", len(got.Datapoints))
	}
	// Session avg = 85, historical avg = 55 → delta = 30
	if got.Datapoints[0].Y != 30 {
		t.Errorf("delta want 30, got %v", got.Datapoints[0].Y)
	}
}

func TestBuildPerfVsHistorical_SkipsCardWithNoPerf(t *testing.T) {
	t.Parallel()
	session := []canonical.PlayerMatchRow{
		mkRowMap("m1", "bazaar", "Bazaar", canonical.OutcomeWin, nil), // pas de perf
	}
	historical := []canonical.PlayerMatchRow{
		mkRowMap("h1", "bazaar", "Bazaar", canonical.OutcomeWin, fptrSyn(60)),
	}
	got := BuildPerfVsHistorical(session, historical, 0)
	if len(got.Datapoints) != 0 {
		t.Errorf("session sans perf -> dp skip, got %d", len(got.Datapoints))
	}
}

func TestBuildHeatmapPlayerMap_BasicAggregation(t *testing.T) {
	t.Parallel()
	rowsByPlayer := map[string][]canonical.PlayerMatchRow{
		"main": {
			mkRowMap("m1", "bazaar", "Bazaar", canonical.OutcomeWin, fptrSyn(80)),
			mkRowMap("m2", "aquarius", "Aquarius", canonical.OutcomeWin, fptrSyn(60)),
		},
		"f1": {
			mkRowMap("m1", "bazaar", "Bazaar", canonical.OutcomeWin, fptrSyn(70)),
			mkRowMap("m2", "aquarius", "Aquarius", canonical.OutcomeLoss, fptrSyn(40)),
		},
	}
	got := BuildHeatmapPlayerMap(rowsByPlayer, 0)
	// 2 maps × 2 joueurs avec perf = 4 cellules
	if len(got.Datapoints) != 4 {
		t.Errorf("want 4 cells, got %d", len(got.Datapoints))
	}
	// Vérifier qu'on a bien main/Bazaar = 80
	found := false
	for _, dp := range got.Datapoints {
		if dp.Y == "main" && dp.X == "Bazaar" {
			found = true
			if dp.Value != 80 {
				t.Errorf("main/Bazaar: want 80, got %v", dp.Value)
			}
		}
	}
	if !found {
		t.Error("cellule main/Bazaar manquante")
	}
}

func TestBuildHeatmapPlayerMap_OmitsCellsWithoutPerf(t *testing.T) {
	t.Parallel()
	rowsByPlayer := map[string][]canonical.PlayerMatchRow{
		"main": {
			mkRowMap("m1", "bazaar", "Bazaar", canonical.OutcomeWin, fptrSyn(80)),
		},
		"f1": {
			mkRowMap("m1", "bazaar", "Bazaar", canonical.OutcomeWin, nil), // pas de perf
		},
	}
	got := BuildHeatmapPlayerMap(rowsByPlayer, 0)
	// Seul main/Bazaar doit etre present
	if len(got.Datapoints) != 1 {
		t.Errorf("cells without perf should be omitted, got %d", len(got.Datapoints))
	}
}

func TestBuildHeatmapPlayerMap_Empty(t *testing.T) {
	t.Parallel()
	got := BuildHeatmapPlayerMap(map[string][]canonical.PlayerMatchRow{}, 0)
	if len(got.Datapoints) != 0 {
		t.Errorf("empty input: want 0 dp, got %d", len(got.Datapoints))
	}
}
