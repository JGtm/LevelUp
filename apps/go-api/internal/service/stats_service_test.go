package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// --- mock (reuse interface from last_match_service_test.go) ---

type mockStatsRepoForStats struct {
	matches  []domain.StatsMatchRow
	matchErr error
	lusr     []domain.LUSRMatchRating
	lusrErr  error
	parts    []domain.ParticipantRow
	partErr  error
}

func (m *mockStatsRepoForStats) LoadStatsMatches(_ context.Context) ([]domain.StatsMatchRow, error) {
	return m.matches, m.matchErr
}
func (m *mockStatsRepoForStats) LoadLUSRHistory(_ context.Context) ([]domain.LUSRMatchRating, error) {
	return m.lusr, m.lusrErr
}
func (m *mockStatsRepoForStats) LoadMatchParticipants(_ context.Context) ([]domain.ParticipantRow, error) {
	return m.parts, m.partErr
}

// --- tests ---

func TestStatsService_GetPage_WinLoss(t *testing.T) {
	now := time.Now()
	win, loss := 2, 3
	repo := &mockStatsRepoForStats{
		matches: []domain.StatsMatchRow{
			{MatchID: "m1", StartTime: now.Add(-2 * time.Hour), Outcome: &win, Kills: 15, Deaths: 5, Assists: 3, Accuracy: float64Ptr(0.6)},
			{MatchID: "m2", StartTime: now.Add(-1 * time.Hour), Outcome: &loss, Kills: 5, Deaths: 10, Assists: 1, Accuracy: float64Ptr(0.375)},
			{MatchID: "m3", StartTime: now, Outcome: &win, Kills: 20, Deaths: 8, Assists: 5, Accuracy: float64Ptr(0.6)},
		},
	}
	svc := NewStatsService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, nil), "Test")
	svc.titleSlug = "halo_infinite"

	resp, err := svc.GetPage(context.Background(), domain.StatsQueryRequest{Tab: "win_loss"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.WinLoss == nil {
		t.Fatal("expected WinLoss tab to be populated")
	}
	if resp.TotalMatches != 3 {
		t.Errorf("TotalMatches = %d, want 3", resp.TotalMatches)
	}
}

func TestStatsService_GetPage_Accuracy(t *testing.T) {
	now := time.Now()
	repo := &mockStatsRepoForStats{
		matches: []domain.StatsMatchRow{
			{MatchID: "m1", StartTime: now, Kills: 10, Deaths: 3, Accuracy: float64Ptr(0.6), TimePlayedSeconds: intPtr(600)},
		},
	}
	svc := NewStatsService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, nil), "Test")
	svc.titleSlug = "halo_infinite"

	resp, err := svc.GetPage(context.Background(), domain.StatsQueryRequest{Tab: "accuracy"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Accuracy == nil {
		t.Fatal("expected Accuracy tab to be populated")
	}
}

func TestStatsService_GetPage_Objective(t *testing.T) {
	now := time.Now()
	repo := &mockStatsRepoForStats{
		matches: []domain.StatsMatchRow{
			{MatchID: "m1", StartTime: now, Kills: 10, Deaths: 3, PersonalScore: intPtr(500)},
		},
	}
	svc := NewStatsService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, nil), "Test")
	svc.titleSlug = "halo_infinite"

	resp, err := svc.GetPage(context.Background(), domain.StatsQueryRequest{Tab: "objective"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Objective == nil {
		t.Fatal("expected Objective tab to be populated")
	}
}

func TestStatsService_GetPage_Form(t *testing.T) {
	now := time.Now()
	repo := &mockStatsRepoForStats{
		matches: []domain.StatsMatchRow{
			{MatchID: "m1", StartTime: now, Kills: 10, Deaths: 3, PerfScoreComputed: float64Ptr(1.2)},
		},
	}
	svc := NewStatsService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, nil), "Test")
	svc.titleSlug = "halo_infinite"

	resp, err := svc.GetPage(context.Background(), domain.StatsQueryRequest{Tab: "form"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Form == nil {
		t.Fatal("expected Form tab to be populated")
	}
}

func TestStatsService_GetPage_LUSR(t *testing.T) {
	now := time.Now()
	repo := &mockStatsRepoForStats{
		matches: []domain.StatsMatchRow{
			{MatchID: "m1", StartTime: now, Kills: 10, Deaths: 3},
		},
		lusr: []domain.LUSRMatchRating{
			{MatchID: "m1", RatingValue: 25.0, RatingDeviation: 8.0},
		},
	}
	svc := NewStatsService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, nil), "Test")
	svc.titleSlug = "halo_infinite"

	resp, err := svc.GetPage(context.Background(), domain.StatsQueryRequest{Tab: "lusr"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.LUSR == nil {
		t.Fatal("expected LUSR tab to be populated")
	}
}

func TestStatsService_GetPage_Empty(t *testing.T) {
	repo := &mockStatsRepoForStats{matches: []domain.StatsMatchRow{}}
	svc := NewStatsService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, nil), "Test")
	svc.titleSlug = "halo_infinite"

	resp, err := svc.GetPage(context.Background(), domain.StatsQueryRequest{Tab: "win_loss"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalMatches != 0 {
		t.Errorf("TotalMatches = %d, want 0", resp.TotalMatches)
	}
	if resp.WinLoss != nil {
		t.Error("expected WinLoss to be nil for empty matches")
	}
}

func TestStatsService_GetPage_Error(t *testing.T) {
	repo := &mockStatsRepoForStats{matchErr: errors.New("fail")}
	svc := NewStatsService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(nil, errors.New("canonical fail")), "Test")
	svc.titleSlug = "halo_infinite"

	_, err := svc.GetPage(context.Background(), domain.StatsQueryRequest{})
	if err == nil {
		t.Error("expected error")
	}
}

func TestStatsService_GetPage_All(t *testing.T) {
	now := time.Now()
	win := 2
	repo := &mockStatsRepoForStats{
		matches: []domain.StatsMatchRow{
			{MatchID: "m1", StartTime: now, Outcome: &win, Kills: 10, Deaths: 3, Accuracy: float64Ptr(0.5), PersonalScore: intPtr(500), PerfScoreComputed: float64Ptr(1.1), TimePlayedSeconds: intPtr(600)},
		},
		lusr: []domain.LUSRMatchRating{
			{MatchID: "m1", RatingValue: 25.0, RatingDeviation: 8.0},
		},
	}
	svc := NewStatsService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, nil), "Test")
	svc.titleSlug = "halo_infinite"

	resp, err := svc.GetPage(context.Background(), domain.StatsQueryRequest{Tab: "all"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.WinLoss == nil || resp.Accuracy == nil || resp.Objective == nil || resp.Form == nil || resp.LUSR == nil {
		t.Error("expected all tabs to be populated for tab=all")
	}
}
