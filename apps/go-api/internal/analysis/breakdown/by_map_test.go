package breakdown

import (
	"math"
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// rowFactory minimise le boilerplate de creation de fixtures.
func rowMap(mapID, label string, outcome canonical.Outcome) Row {
	return Row{Outcome: outcome, MapID: mapID, MapLabel: label}
}

func TestByMap_Empty(t *testing.T) {
	t.Parallel()
	got := ByMap(nil)
	if len(got) != 0 {
		t.Errorf("nil input: want 0, got %d", len(got))
	}
	got = ByMap([]Row{})
	if len(got) != 0 {
		t.Errorf("empty input: want 0, got %d", len(got))
	}
}

func TestByMap_SingleMap_AllOutcomes(t *testing.T) {
	t.Parallel()
	rows := []Row{
		rowMap("m1", "Bazaar", canonical.OutcomeWin),
		rowMap("m1", "Bazaar", canonical.OutcomeWin),
		rowMap("m1", "Bazaar", canonical.OutcomeLoss),
		rowMap("m1", "Bazaar", canonical.OutcomeTie),
		rowMap("m1", "Bazaar", canonical.OutcomeDNF),
	}
	got := ByMap(rows)
	if len(got) != 1 {
		t.Fatalf("want 1 aggregate, got %d", len(got))
	}
	a := got[0]
	if a.Played != 5 || a.Wins != 2 || a.Losses != 1 || a.Ties != 1 || a.DNF != 1 {
		t.Errorf("counts: %+v", a.Counts)
	}
	if math.Abs(a.WinRate-0.4) > 1e-9 {
		t.Errorf("WinRate want 0.4 got %v", a.WinRate)
	}
	if math.Abs(a.LossRate-0.2) > 1e-9 || math.Abs(a.TieRate-0.2) > 1e-9 || math.Abs(a.DNFRate-0.2) > 1e-9 {
		t.Errorf("rates: WR=%v LR=%v TR=%v DR=%v", a.WinRate, a.LossRate, a.TieRate, a.DNFRate)
	}
}

func TestByMap_AllWins(t *testing.T) {
	t.Parallel()
	rows := []Row{
		rowMap("m1", "Bazaar", canonical.OutcomeWin),
		rowMap("m1", "Bazaar", canonical.OutcomeWin),
	}
	got := ByMap(rows)
	if got[0].WinRate != 1 || got[0].LossRate != 0 {
		t.Errorf("100%% wins: WR=%v LR=%v", got[0].WinRate, got[0].LossRate)
	}
}

func TestByMap_AllLosses(t *testing.T) {
	t.Parallel()
	rows := []Row{
		rowMap("m1", "Bazaar", canonical.OutcomeLoss),
	}
	got := ByMap(rows)
	if got[0].WinRate != 0 || got[0].LossRate != 1 {
		t.Errorf("100%% losses: WR=%v LR=%v", got[0].WinRate, got[0].LossRate)
	}
}

func TestByMap_IgnoresEmptyMapID(t *testing.T) {
	t.Parallel()
	rows := []Row{
		rowMap("", "Unknown", canonical.OutcomeWin),
		rowMap("m1", "Bazaar", canonical.OutcomeWin),
	}
	got := ByMap(rows)
	if len(got) != 1 || got[0].MapID != "m1" {
		t.Errorf("empty MapID should be ignored, got %+v", got)
	}
}

func TestByMap_SortedByWinRateDesc(t *testing.T) {
	t.Parallel()
	rows := []Row{
		rowMap("loser", "Loser", canonical.OutcomeLoss),
		rowMap("loser", "Loser", canonical.OutcomeLoss),
		rowMap("winner", "Winner", canonical.OutcomeWin),
		rowMap("winner", "Winner", canonical.OutcomeWin),
		rowMap("middle", "Middle", canonical.OutcomeWin),
		rowMap("middle", "Middle", canonical.OutcomeLoss),
	}
	got := ByMap(rows)
	if len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}
	if got[0].MapID != "winner" {
		t.Errorf("first should be winner (WR=1), got %s (WR=%v)", got[0].MapID, got[0].WinRate)
	}
	if got[1].MapID != "middle" {
		t.Errorf("second should be middle (WR=0.5), got %s", got[1].MapID)
	}
	if got[2].MapID != "loser" {
		t.Errorf("third should be loser (WR=0), got %s", got[2].MapID)
	}
}

func TestByMap_TieBreakerByMapIDAsc(t *testing.T) {
	t.Parallel()
	rows := []Row{
		rowMap("zebra", "Zebra", canonical.OutcomeWin),
		rowMap("alpha", "Alpha", canonical.OutcomeWin),
		rowMap("middle", "Middle", canonical.OutcomeWin),
	}
	got := ByMap(rows)
	if got[0].MapID != "alpha" || got[1].MapID != "middle" || got[2].MapID != "zebra" {
		t.Errorf("alphabetic tie-breaker failed: %v / %v / %v", got[0].MapID, got[1].MapID, got[2].MapID)
	}
}

func TestByMap_PreservesFirstNonEmptyLabel(t *testing.T) {
	t.Parallel()
	rows := []Row{
		rowMap("m1", "", canonical.OutcomeWin),
		rowMap("m1", "Bazaar", canonical.OutcomeWin),
		rowMap("m1", "Bazaar Renamed", canonical.OutcomeLoss),
	}
	got := ByMap(rows)
	if got[0].MapLabel != "Bazaar" {
		t.Errorf("first non-empty label should win, got %q", got[0].MapLabel)
	}
}

func TestByMap_AvgPerformance(t *testing.T) {
	t.Parallel()
	score := func(v float64) *float64 { return &v }
	rows := []Row{
		{MapID: "m1", MapLabel: "Bazaar", Outcome: canonical.OutcomeWin, PerformanceScore: score(80)},
		{MapID: "m1", MapLabel: "Bazaar", Outcome: canonical.OutcomeWin, PerformanceScore: score(60)},
		{MapID: "m1", MapLabel: "Bazaar", Outcome: canonical.OutcomeLoss}, // pas de score
	}
	got := ByMap(rows)
	if got[0].AvgPerformanceScore == nil || *got[0].AvgPerformanceScore != 70 {
		t.Errorf("avg should be 70 (mean of 80+60), got %v", got[0].AvgPerformanceScore)
	}
}

func TestByMap_AvgPerformanceAllNil(t *testing.T) {
	t.Parallel()
	rows := []Row{
		rowMap("m1", "Bazaar", canonical.OutcomeWin),
		rowMap("m1", "Bazaar", canonical.OutcomeLoss),
	}
	got := ByMap(rows)
	if got[0].AvgPerformanceScore != nil {
		t.Errorf("nil expected when no row has score, got %v", *got[0].AvgPerformanceScore)
	}
}

func TestByMap_UnknownOutcomeNotCounted(t *testing.T) {
	t.Parallel()
	rows := []Row{
		rowMap("m1", "Bazaar", canonical.OutcomeWin),
		rowMap("m1", "Bazaar", canonical.Outcome("unknown")),
		rowMap("m1", "Bazaar", canonical.Outcome("")),
	}
	got := ByMap(rows)
	if got[0].Played != 3 {
		t.Errorf("Played should count unknown rows, got %d", got[0].Played)
	}
	if got[0].Wins != 1 || got[0].Losses != 0 {
		t.Errorf("only 1 win expected, got W=%d L=%d", got[0].Wins, got[0].Losses)
	}
	// WinRate = 1/3 (rate sur Played, pas sur outcomes connus)
	if math.Abs(got[0].WinRate-1.0/3.0) > 1e-9 {
		t.Errorf("WinRate want 1/3 got %v", got[0].WinRate)
	}
}
