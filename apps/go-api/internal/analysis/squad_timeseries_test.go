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

func TestGroupSquadBySession_WithSessions(t *testing.T) {
	sid1, sid2 := 1, 2
	lbl1, lbl2 := "S1-2025", "S2-2025"
	perf1, perf2, perf3 := 70.0, 80.0, 90.0
	rows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: time.Now(), TeamMMR: 1200, Outcome: domain.OutcomeWin, SessionID: &sid1, SessionLabel: &lbl1, PerformanceScore: &perf1},
		{MatchID: "m2", StartTime: time.Now(), TeamMMR: 1300, Outcome: domain.OutcomeLoss, SessionID: &sid1, SessionLabel: &lbl1, PerformanceScore: &perf2},
		{MatchID: "m3", StartTime: time.Now(), TeamMMR: 1400, Outcome: domain.OutcomeWin, SessionID: &sid2, SessionLabel: &lbl2, PerformanceScore: &perf3},
	}
	got := groupSquadBySession(rows)
	if len(got) != 2 {
		t.Fatalf("expected 2 session points, got %d", len(got))
	}
	if got[0].MatchCount != 2 {
		t.Errorf("expected 2 matches in session 1, got %d", got[0].MatchCount)
	}
}

func TestBucketByTime_Weekly(t *testing.T) {
	base := time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC) // Monday
	rows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: base, TeamMMR: 1200, Outcome: domain.OutcomeWin},
		{MatchID: "m2", StartTime: base.Add(24 * time.Hour), TeamMMR: 1300, Outcome: domain.OutcomeLoss},
		{MatchID: "m3", StartTime: base.Add(8 * 24 * time.Hour), TeamMMR: 1400, Outcome: domain.OutcomeWin},
	}
	got := bucketByTime(rows, "week")
	if len(got) != 2 {
		t.Fatalf("expected 2 weekly buckets, got %d", len(got))
	}
	if got[0].MatchCount != 2 {
		t.Errorf("expected 2 matches in first week, got %d", got[0].MatchCount)
	}
}

func TestBucketByTime_Monthly(t *testing.T) {
	rows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC), TeamMMR: 1200, Outcome: domain.OutcomeWin},
		{MatchID: "m2", StartTime: time.Date(2025, 1, 20, 12, 0, 0, 0, time.UTC), TeamMMR: 1300, Outcome: domain.OutcomeLoss},
		{MatchID: "m3", StartTime: time.Date(2025, 2, 5, 12, 0, 0, 0, time.UTC), TeamMMR: 1400, Outcome: domain.OutcomeWin},
	}
	got := bucketByTime(rows, "month")
	if len(got) != 2 {
		t.Fatalf("expected 2 monthly buckets, got %d", len(got))
	}
}

func TestGroupSquadByTime_FallbackToMonth(t *testing.T) {
	base := time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)
	rows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: base, TeamMMR: 1200, Outcome: domain.OutcomeWin},
		{MatchID: "m2", StartTime: base.Add(7 * 24 * time.Hour), TeamMMR: 1200, Outcome: domain.OutcomeWin},
		{MatchID: "m3", StartTime: base.Add(14 * 24 * time.Hour), TeamMMR: 1200, Outcome: domain.OutcomeWin},
		{MatchID: "m4", StartTime: base.Add(21 * 24 * time.Hour), TeamMMR: 1200, Outcome: domain.OutcomeWin},
	}
	got := groupSquadByTime(rows, 2)
	if len(got) > 2 {
		t.Errorf("expected ≤2 buckets after fallback, got %d", len(got))
	}
}
