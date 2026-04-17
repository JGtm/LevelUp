package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// buildCumulTab
// ---------------------------------------------------------------------------

func TestBuildCumulTab_Empty(t *testing.T) {
	tab := buildCumulTab(nil)
	if len(tab.CumulativeKD) != 0 {
		t.Errorf("expected empty CumulativeKD, got %d", len(tab.CumulativeKD))
	}
}

func TestBuildCumulTab_SingleMatch(t *testing.T) {
	matches := []domain.StatsMatchRow{
		{Kills: 10, Deaths: 5, StartTime: time.Now()},
	}
	tab := buildCumulTab(matches)
	if len(tab.CumulativeKD) != 1 {
		t.Fatalf("expected 1 CumulativeKD point, got %d", len(tab.CumulativeKD))
	}
	if tab.CumulativeKD[0].Value != 2.0 {
		t.Errorf("expected cumul KD 2.0, got %v", tab.CumulativeKD[0].Value)
	}
	if tab.CumulativeNet[0].Value != 5.0 {
		t.Errorf("expected cumul net 5, got %v", tab.CumulativeNet[0].Value)
	}
}

func TestBuildCumulTab_RollingKD(t *testing.T) {
	matches := make([]domain.StatsMatchRow, 25)
	now := time.Now()
	for i := range matches {
		matches[i] = domain.StatsMatchRow{
			Kills:     10,
			Deaths:    5,
			StartTime: now.Add(time.Duration(i) * time.Hour),
		}
	}
	tab := buildCumulTab(matches)
	if len(tab.RollingKD) != 25 {
		t.Fatalf("expected 25 rolling KD points, got %d", len(tab.RollingKD))
	}
	// All same stats → rolling KD should be 2.0 throughout.
	if tab.RollingKD[24].Value != 2.0 {
		t.Errorf("expected rolling KD 2.0, got %v", tab.RollingKD[24].Value)
	}
}

// ---------------------------------------------------------------------------
// buildTimeseriesFormTab
// ---------------------------------------------------------------------------

func TestBuildTimeseriesFormTab_EWMA(t *testing.T) {
	matches := []domain.StatsMatchRow{
		{Kills: 10, Deaths: 5, StartTime: time.Now()},
		{Kills: 20, Deaths: 10, StartTime: time.Now().Add(time.Hour)},
		{Kills: 5, Deaths: 10, StartTime: time.Now().Add(2 * time.Hour)},
	}
	tab := buildTimeseriesFormTab(matches)
	if len(tab.EWMAKDPoints) != 3 {
		t.Fatalf("expected 3 EWMA points, got %d", len(tab.EWMAKDPoints))
	}
	// First EWMA point = raw KD = 2.0
	if tab.EWMAKDPoints[0].Value != 2.0 {
		t.Errorf("expected first EWMA 2.0, got %v", tab.EWMAKDPoints[0].Value)
	}
}

// ---------------------------------------------------------------------------
// buildIntensityTab
// ---------------------------------------------------------------------------

func TestBuildIntensityTab_Empty(t *testing.T) {
	tab := buildIntensityTab(nil)
	if tab.HeatmapData == nil || len(tab.HeatmapData) != 0 {
		t.Errorf("expected empty HeatmapData, got %v", tab.HeatmapData)
	}
}

func TestBuildIntensityTab_SingleMatch(t *testing.T) {
	dur := 300
	score := 1000
	matches := []domain.StatsMatchRow{
		{
			Kills:             10,
			Deaths:            5,
			StartTime:         time.Date(2025, 1, 6, 14, 0, 0, 0, time.UTC), // Monday
			PersonalScore:     &score,
			TimePlayedSeconds: &dur,
		},
	}
	tab := buildIntensityTab(matches)
	if len(tab.HeatmapData) != 1 {
		t.Fatalf("expected 1 heatmap point, got %d", len(tab.HeatmapData))
	}
	p := tab.HeatmapData[0]
	if p.DayOfWeek != 0 { // Monday
		t.Errorf("expected day 0 (Monday), got %d", p.DayOfWeek)
	}
	if p.Hour != 14 {
		t.Errorf("expected hour 14, got %d", p.Hour)
	}
	if len(tab.ScorePerMinData) != 1 {
		t.Fatalf("expected 1 score/min point, got %d", len(tab.ScorePerMinData))
	}
	// 1000 / (300/60) = 200
	if tab.ScorePerMinData[0].Value != 200.0 {
		t.Errorf("expected score/min 200, got %v", tab.ScorePerMinData[0].Value)
	}
}

// ---------------------------------------------------------------------------
// buildDistributionsTab
// ---------------------------------------------------------------------------

func TestBuildDistributionsTab_Empty(t *testing.T) {
	tab := buildDistributionsTab(nil)
	if len(tab.KDABuckets) != 0 {
		t.Errorf("expected empty KDABuckets, got %d", len(tab.KDABuckets))
	}
	if len(tab.KillsBuckets) != 0 {
		t.Errorf("expected empty KillsBuckets, got %d", len(tab.KillsBuckets))
	}
}

func TestBuildDistributionsTab_CorrectBuckets(t *testing.T) {
	matches := []domain.StatsMatchRow{
		{Kills: 10, Deaths: 5, StartTime: time.Now()}, // KD = 2.0
		{Kills: 5, Deaths: 5, StartTime: time.Now()},  // KD = 1.0
		{Kills: 15, Deaths: 5, StartTime: time.Now()}, // KD = 3.0
		{Kills: 0, Deaths: 10, StartTime: time.Now()}, // KD = 0.0
	}
	tab := buildDistributionsTab(matches)
	if len(tab.KDABuckets) == 0 {
		t.Fatal("expected non-empty KDABuckets")
	}
	if len(tab.CorrelationPoints) != 4 {
		t.Errorf("expected 4 correlation points, got %d", len(tab.CorrelationPoints))
	}
}

// ---------------------------------------------------------------------------
// FanoutPlan / FanoutResult (domain types)
// ---------------------------------------------------------------------------

func TestFanoutPlan_Empty(t *testing.T) {
	plan := domain.FanoutPlan{
		SourceGamertag: "TestPlayer",
	}
	if len(plan.Targets) != 0 {
		t.Errorf("expected empty targets")
	}
}

func TestFanoutResult_NoErrors(t *testing.T) {
	result := domain.FanoutResult{
		TargetsProcessed: 3,
		MatchesEnriched:  15,
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors")
	}
	if result.TargetsProcessed != 3 {
		t.Errorf("expected 3 targets processed, got %d", result.TargetsProcessed)
	}
}

// ---------------------------------------------------------------------------
// TimeseriesService (GetPage + NewTimeseriesService)
// ---------------------------------------------------------------------------

type mockTimeseriesRepo struct {
	matches []domain.StatsMatchRow
	err     error
}

func (m *mockTimeseriesRepo) LoadStatsMatches(_ context.Context) ([]domain.StatsMatchRow, error) {
	return m.matches, m.err
}
func (m *mockTimeseriesRepo) LoadLUSRHistory(_ context.Context) ([]domain.LUSRMatchRating, error) {
	return nil, nil
}
func (m *mockTimeseriesRepo) LoadMatchParticipants(_ context.Context) ([]domain.ParticipantRow, error) {
	return nil, nil
}

func TestNewTimeseriesService(t *testing.T) {
	svc := NewTimeseriesService(&mockTimeseriesRepo{})
	if svc == nil {
		t.Fatal("expected non-nil")
	}
}

func TestTimeseriesService_GetPage_Empty(t *testing.T) {
	svc := NewTimeseriesService(&mockTimeseriesRepo{})
	resp, err := svc.GetPage(context.Background(), domain.TimeseriesQueryRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TotalMatches != 0 {
		t.Errorf("TotalMatches = %d, want 0", resp.TotalMatches)
	}
}

func TestTimeseriesService_GetPage_Error(t *testing.T) {
	svc := NewTimeseriesService(&mockTimeseriesRepo{err: errors.New("fail")})
	_, err := svc.GetPage(context.Background(), domain.TimeseriesQueryRequest{})
	if err == nil {
		t.Error("expected error")
	}
}

func TestTimeseriesService_GetPage_WithData(t *testing.T) {
	win := 2
	dur := 600
	acc := 0.5
	ps := 1500
	kda := 2.0
	matches := []domain.StatsMatchRow{
		{Kills: 10, Deaths: 5, Assists: 3, Outcome: &win, TimePlayedSeconds: &dur, Accuracy: &acc, PersonalScore: &ps, KDA: &kda},
	}
	svc := NewTimeseriesService(&mockTimeseriesRepo{matches: matches})
	resp, err := svc.GetPage(context.Background(), domain.TimeseriesQueryRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TotalMatches != 1 {
		t.Errorf("TotalMatches = %d, want 1", resp.TotalMatches)
	}
}

// ---------------------------------------------------------------------------
// buildTimeseriesSummaryTab
// ---------------------------------------------------------------------------

func TestBuildTimeseriesSummaryTab_Empty(t *testing.T) {
	tab := buildTimeseriesSummaryTab(nil)
	if len(tab.KpiCards) != 0 {
		t.Errorf("expected 0 cards, got %d", len(tab.KpiCards))
	}
}

func TestBuildTimeseriesSummaryTab_WithMatches(t *testing.T) {
	win := 2
	dur := 600
	matches := []domain.StatsMatchRow{
		{Kills: 10, Deaths: 5, Outcome: &win, TimePlayedSeconds: &dur},
		{Kills: 15, Deaths: 3, Outcome: &win, TimePlayedSeconds: &dur},
	}
	tab := buildTimeseriesSummaryTab(matches)
	if len(tab.KpiCards) == 0 {
		t.Error("expected cards")
	}
}
