// Package analysis — squad_timeseries_test.go : tests pour les séries temporelles escouade.
package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func TestComputeSquadTimeseries_Empty(t *testing.T) {
	result := ComputeSquadTimeseries(nil, 20)
	if len(result) != 0 {
		t.Errorf("expected 0 points, got %d", len(result))
	}
}

func TestComputeSquadTimeseries_SingleMatch(t *testing.T) {
	kda := 2.5
	rows := []domain.SquadMatchRow{
		{
			MatchID:        "m1",
			StartTime:      time.Now(),
			Outcome:        OutcomeWin,
			Kills:          10,
			Deaths:         5,
			Assists:        3,
			KDA:            &kda,
			TimePlayedSecs: 600,
		},
	}
	result := ComputeSquadTimeseries(rows, 20)
	if len(result) == 0 {
		t.Error("expected at least 1 point")
	}
}

func TestGroupSquadBySession_NoSessions(t *testing.T) {
	rows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: time.Now(), Kills: 10, Deaths: 5, Outcome: OutcomeWin},
	}
	result := groupSquadBySession(rows)
	// Without session IDs, should still produce buckets (fallback to time-based)
	_ = result // just verify no panic
}
