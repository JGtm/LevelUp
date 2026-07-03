package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// --- mock ---

type mockSessionsRepo struct {
	rows []domain.SessionMatchRow
	err  error
}

func (m *mockSessionsRepo) LoadSessionMatches(_ context.Context) ([]domain.SessionMatchRow, error) {
	return m.rows, m.err
}

// --- tests SessionsService ---

func TestSessionsService_GetSessions_OK(t *testing.T) {
	now := time.Now()
	repo := &mockSessionsRepo{
		rows: []domain.SessionMatchRow{
			{MatchID: "m1", StartTime: now.Add(-3 * time.Hour)},
			{MatchID: "m2", StartTime: now.Add(-2 * time.Hour)},
			{MatchID: "m3", StartTime: now},
		},
	}
	svc := NewSessionsService(repo)

	resp, err := svc.GetSessions(context.Background(), domain.SessionComputeOptions{
		GapMinutes: 60,
		Mode:       domain.SessionModeGap,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Sessions) == 0 {
		t.Error("expected at least one session")
	}
}

func TestSessionsService_GetSessions_Empty(t *testing.T) {
	repo := &mockSessionsRepo{rows: []domain.SessionMatchRow{}}
	svc := NewSessionsService(repo)

	resp, err := svc.GetSessions(context.Background(), domain.SessionComputeOptions{
		GapMinutes: 60,
		Mode:       domain.SessionModeGap,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Sessions) != 0 {
		t.Errorf("expected 0 sessions for empty input, got %d", len(resp.Sessions))
	}
}

func TestSessionsService_GetSessions_Error(t *testing.T) {
	repo := &mockSessionsRepo{err: errors.New("fail")}
	svc := NewSessionsService(repo)

	_, err := svc.GetSessions(context.Background(), domain.SessionComputeOptions{})
	if err == nil {
		t.Error("expected error")
	}
}

func TestSessionsService_GetSessions_ContextMode(t *testing.T) {
	now := time.Now()
	repo := &mockSessionsRepo{
		rows: []domain.SessionMatchRow{
			{MatchID: "m1", StartTime: now.Add(-1 * time.Hour)},
			{MatchID: "m2", StartTime: now},
		},
	}
	svc := NewSessionsService(repo)

	resp, err := svc.GetSessions(context.Background(), domain.SessionComputeOptions{
		Mode:       domain.SessionModeContext,
		GapMinutes: 60,
		CutoffHour: 6,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Assignments) == 0 {
		t.Error("expected assignments from context mode")
	}
}

// --- tests SessionCompareService ---

type mockSessionCompareSessionsRepo struct {
	rows []domain.SessionMatchRow
	err  error
}

func (m *mockSessionCompareSessionsRepo) LoadSessionMatches(_ context.Context) ([]domain.SessionMatchRow, error) {
	return m.rows, m.err
}

type mockSessionCompareStatsRepo struct {
	matches      []legacymatch.StatsMatchRow
	matchErr     error
	lusr         []domain.LUSRMatchRating
	lusrErr      error
	participants []domain.ParticipantRow
	partErr      error
}

func (m *mockSessionCompareStatsRepo) LoadStatsMatches(_ context.Context) ([]legacymatch.StatsMatchRow, error) {
	return m.matches, m.matchErr
}
func (m *mockSessionCompareStatsRepo) LoadLUSRHistory(_ context.Context) ([]domain.LUSRMatchRating, error) {
	return m.lusr, m.lusrErr
}
func (m *mockSessionCompareStatsRepo) LoadMatchParticipants(_ context.Context) ([]domain.ParticipantRow, error) {
	return m.participants, m.partErr
}

func TestLastOrNil_Empty(t *testing.T) {
	if lastOrNil(nil, nil) != "" {
		t.Error("expected empty")
	}
}

func TestLastOrNil_WithLabels(t *testing.T) {
	labels := []string{"S1", "S2", "S3"}
	if lastOrNil(labels, nil) != "S3" {
		t.Error("expected last label")
	}
}

func TestLastOrNil_WithOverride(t *testing.T) {
	override := "custom"
	if lastOrNil([]string{"S1"}, &override) != "custom" {
		t.Error("expected override")
	}
}

func TestSecondLastOrNil_Empty(t *testing.T) {
	if secondLastOrNil(nil, nil) != "" {
		t.Error("expected empty")
	}
}

func TestSecondLastOrNil_SingleLabel(t *testing.T) {
	if secondLastOrNil([]string{"S1"}, nil) != "" {
		t.Error("expected empty with only 1 label")
	}
}

func TestSecondLastOrNil_TwoLabels(t *testing.T) {
	if secondLastOrNil([]string{"S1", "S2"}, nil) != "S1" {
		t.Error("expected S1")
	}
}

func TestFilterBySession_Empty(t *testing.T) {
	result := filterBySession(nil, "S1")
	if len(result) != 0 {
		t.Error("expected empty")
	}
}

func TestFilterBySession_EmptyLabel(t *testing.T) {
	result := filterBySession([]legacymatch.StatsMatchRow{{Kills: 10}}, "")
	if result != nil {
		t.Error("expected nil for empty label")
	}
}

func TestFilterBySession_Filters(t *testing.T) {
	s1 := "S1"
	s2 := "S2"
	matches := []legacymatch.StatsMatchRow{
		{Kills: 10, SessionLabel: &s1},
		{Kills: 20, SessionLabel: &s2},
		{Kills: 30, SessionLabel: &s1},
	}
	result := filterBySession(matches, "S1")
	if len(result) != 2 {
		t.Errorf("expected 2, got %d", len(result))
	}
}

func TestWinRate_Empty(t *testing.T) {
	if winRate(nil) != 0 {
		t.Error("expected 0")
	}
}

func TestWinRate_WithMatches(t *testing.T) {
	win := 2
	loss := 3
	matches := []legacymatch.StatsMatchRow{
		{Outcome: &win},
		{Outcome: &loss},
		{Outcome: &win},
	}
	wr := winRate(matches)
	if wr < 66 || wr > 67 {
		t.Errorf("winRate = %f, want ~66.7", wr)
	}
}

func TestAvgKD_Empty(t *testing.T) {
	if avgKD(nil) != 0 {
		t.Error("expected 0")
	}
}

func TestAvgKD_ZeroDeaths(t *testing.T) {
	matches := []legacymatch.StatsMatchRow{
		{Kills: 10, Deaths: 0},
	}
	if avgKD(matches) != 10 {
		t.Errorf("avgKD zero deaths = %f, want 10", avgKD(matches))
	}
}

func TestKillsPerGame_Empty(t *testing.T) {
	if killsPerGame(nil) != 0 {
		t.Error("expected 0")
	}
}

func TestDeathsPerGame_Empty(t *testing.T) {
	if deathsPerGame(nil) != 0 {
		t.Error("expected 0")
	}
}

func TestDetermineWinner_Tie(t *testing.T) {
	w := determineWinner(5.0, 5.0)
	if w == nil || *w != "tie" {
		t.Error("expected tie")
	}
}

func TestDetermineWinner_A(t *testing.T) {
	w := determineWinner(10.0, 5.0)
	if w == nil || *w != "a" {
		t.Error("expected a wins")
	}
}

func TestDetermineWinner_B(t *testing.T) {
	w := determineWinner(3.0, 8.0)
	if w == nil || *w != "b" {
		t.Error("expected b wins")
	}
}

func TestBuildCompareEntry_Empty(t *testing.T) {
	result := buildCompareEntry(nil, "S1", 225)
	if result != nil {
		t.Error("expected nil for empty matches")
	}
}

func TestBuildCompareEntry_EmptyLabel(t *testing.T) {
	result := buildCompareEntry([]legacymatch.StatsMatchRow{{Kills: 10}}, "", 225)
	if result != nil {
		t.Error("expected nil for empty label")
	}
}

func TestBuildCompareEntry_WithData(t *testing.T) {
	win := 2
	now := time.Now()
	matches := []legacymatch.StatsMatchRow{
		{Kills: 10, Deaths: 5, Outcome: &win, StartTime: now},
		{Kills: 8, Deaths: 3, Outcome: &win, StartTime: now.Add(time.Hour)},
	}
	result := buildCompareEntry(matches, "S1", 225)
	if result == nil {
		t.Fatal("expected non-nil")
	}
	if result.TotalMatches != 2 {
		t.Errorf("TotalMatches = %d, want 2", result.TotalMatches)
	}
	if result.Wins != 2 {
		t.Errorf("Wins = %d, want 2", result.Wins)
	}
}

func TestBuildCompareMetrics(t *testing.T) {
	win := 2
	loss := 3
	a := []legacymatch.StatsMatchRow{
		{Kills: 15, Deaths: 5, Outcome: &win},
		{Kills: 10, Deaths: 5, Outcome: &win},
	}
	b := []legacymatch.StatsMatchRow{
		{Kills: 5, Deaths: 10, Outcome: &loss},
		{Kills: 8, Deaths: 8, Outcome: &loss},
	}
	metrics := buildCompareMetrics(a, b)
	if len(metrics) < 4 {
		t.Errorf("expected at least 4 metrics, got %d", len(metrics))
	}
}

func TestCompareMetric(t *testing.T) {
	m := compareMetric("kd", "K/D", 2.0, 1.5, "%.2f")
	if m.Key != "kd" {
		t.Errorf("Key = %q, want kd", m.Key)
	}
	if m.Winner == nil || *m.Winner != "a" {
		t.Error("expected a to win with higher K/D")
	}
}

func TestCompareMetricInverse(t *testing.T) {
	m := compareMetricInverse("deaths", "Deaths", 3.0, 5.0, "%.1f")
	if m.Winner == nil || *m.Winner != "a" {
		t.Error("expected a to win with lower deaths (inverse)")
	}
}
