package analysis

import (
	"math"
	"testing"

	"levelup/go-api/internal/games/canonical"
)

func float64PtrKPI(v float64) *float64 { return &v }

func intPtrKPI(v int) *int { return &v }

func mkRow(kills, deaths, assists, timePlayed int, outcome canonical.Outcome, accuracy, avgLife *float64) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{
		Self: canonical.MatchParticipant{
			Outcome:        outcome,
			Kills:          intPtrKPI(kills),
			Deaths:         intPtrKPI(deaths),
			Assists:        intPtrKPI(assists),
			TimePlayed:     intPtrKPI(timePlayed),
			Accuracy:       accuracy,
			AvgLifeSeconds: avgLife,
		},
	}
}

func TestComputeKPIStats_Empty(t *testing.T) {
	t.Parallel()
	got := ComputeKPIStats(nil)
	if got.MatchesCount != 0 {
		t.Errorf("empty: want 0 matches, got %d", got.MatchesCount)
	}
}

func TestComputeKPIStats_BasicAggregation(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		mkRow(10, 5, 2, 600, canonical.OutcomeWin, float64PtrKPI(45.0), float64PtrKPI(30.0)),
		mkRow(8, 7, 1, 400, canonical.OutcomeLoss, float64PtrKPI(50.0), float64PtrKPI(20.0)),
		mkRow(4, 3, 5, 500, canonical.OutcomeTie, nil, nil),
	}
	got := ComputeKPIStats(rows)

	if got.MatchesCount != 3 {
		t.Errorf("MatchesCount: want 3, got %d", got.MatchesCount)
	}
	if got.TotalPlaySeconds != 1500 {
		t.Errorf("TotalPlaySeconds: want 1500, got %d", got.TotalPlaySeconds)
	}
	if math.Abs(got.AvgMatchSeconds-500) > 1e-9 {
		t.Errorf("AvgMatchSeconds: want 500, got %v", got.AvgMatchSeconds)
	}

	// Kills total = 22, deaths = 15, assists = 8, sur 3 matchs
	if math.Abs(got.KillsPerGame-22.0/3.0) > 1e-9 {
		t.Errorf("KillsPerGame: want 22/3, got %v", got.KillsPerGame)
	}
	if math.Abs(got.DeathsPerGame-5.0) > 1e-9 {
		t.Errorf("DeathsPerGame: want 5, got %v", got.DeathsPerGame)
	}

	// PerMinute : 1500s = 25 minutes
	if math.Abs(got.KillsPerMinute-22.0/25.0) > 1e-9 {
		t.Errorf("KillsPerMinute: want 22/25, got %v", got.KillsPerMinute)
	}

	// Accuracy : moyenne sur les 2 samples (45 + 50) / 2 = 47.5
	if math.Abs(got.AvgAccuracy-47.5) > 1e-9 {
		t.Errorf("AvgAccuracy: want 47.5, got %v", got.AvgAccuracy)
	}
	// AvgLife : (30 + 20) / 2 = 25
	if math.Abs(got.AvgLifeSeconds-25.0) > 1e-9 {
		t.Errorf("AvgLifeSeconds: want 25, got %v", got.AvgLifeSeconds)
	}

	if got.Outcomes.Wins != 1 || got.Outcomes.Losses != 1 || got.Outcomes.Ties != 1 || got.Outcomes.DNF != 0 {
		t.Errorf("Outcomes: want W1/L1/T1/DNF0, got %+v", got.Outcomes)
	}
}

func TestComputeKPIStats_AllNilFieldsTolere(t *testing.T) {
	t.Parallel()
	// Row sans aucun pointer renseigne -> les agregats sont 0 mais MatchesCount=1.
	row := canonical.PlayerMatchRow{
		Self: canonical.MatchParticipant{Outcome: canonical.OutcomeWin},
	}
	got := ComputeKPIStats([]canonical.PlayerMatchRow{row})
	if got.MatchesCount != 1 {
		t.Errorf("MatchesCount: want 1, got %d", got.MatchesCount)
	}
	if got.KillsPerGame != 0 || got.AvgAccuracy != 0 {
		t.Errorf("nil fields should not contribute, got KPG=%v Acc=%v",
			got.KillsPerGame, got.AvgAccuracy)
	}
	if got.Outcomes.Wins != 1 {
		t.Errorf("Win should still be counted, got %+v", got.Outcomes)
	}
}

func TestComputeKPIStats_OutcomeBreakdown(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		mkRow(10, 5, 0, 600, canonical.OutcomeWin, nil, nil),
		mkRow(10, 5, 0, 600, canonical.OutcomeWin, nil, nil),
		mkRow(5, 10, 0, 600, canonical.OutcomeLoss, nil, nil),
		mkRow(5, 5, 0, 600, canonical.OutcomeTie, nil, nil),
		mkRow(0, 1, 0, 60, canonical.OutcomeDNF, nil, nil),
		mkRow(0, 0, 0, 0, canonical.Outcome(""), nil, nil), // outcome vide ignore
	}
	got := ComputeKPIStats(rows)
	if got.Outcomes.Wins != 2 || got.Outcomes.Losses != 1 || got.Outcomes.Ties != 1 || got.Outcomes.DNF != 1 {
		t.Errorf("outcomes: want W2/L1/T1/DNF1, got %+v", got.Outcomes)
	}
}

func TestComputeKPIStats_ZeroPlaySecondsNoPanicOnPerMin(t *testing.T) {
	t.Parallel()
	// Tous les TimePlayed = 0 -> *PerMinute restent 0 (pas de division par zero).
	rows := []canonical.PlayerMatchRow{
		mkRow(10, 5, 0, 0, canonical.OutcomeWin, nil, nil),
	}
	got := ComputeKPIStats(rows)
	if got.KillsPerMinute != 0 || got.DeathsPerMinute != 0 || got.AssistsPerMinute != 0 {
		t.Errorf("zero play seconds: per-min should be 0, got %+v",
			[]float64{got.KillsPerMinute, got.DeathsPerMinute, got.AssistsPerMinute})
	}
}
