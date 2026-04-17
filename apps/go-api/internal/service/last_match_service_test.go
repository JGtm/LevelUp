package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// --- mock ---

type mockStatsRepo struct {
	matches      []domain.StatsMatchRow
	matchErr     error
	lusr         []domain.LUSRMatchRating
	lusrErr      error
	participants []domain.ParticipantRow
	partErr      error
}

func (m *mockStatsRepo) LoadStatsMatches(_ context.Context) ([]domain.StatsMatchRow, error) {
	return m.matches, m.matchErr
}
func (m *mockStatsRepo) LoadLUSRHistory(_ context.Context) ([]domain.LUSRMatchRating, error) {
	return m.lusr, m.lusrErr
}
func (m *mockStatsRepo) LoadMatchParticipants(_ context.Context) ([]domain.ParticipantRow, error) {
	return m.participants, m.partErr
}

// --- tests LastMatchService ---

func TestLastMatchService_Resolve_OK(t *testing.T) {
	now := time.Now()
	label := "S1"
	repo := &mockStatsRepo{
		matches: []domain.StatsMatchRow{
			{MatchID: "m1", StartTime: now.Add(-2 * time.Hour)},
			{MatchID: "m2", StartTime: now.Add(-1 * time.Hour), SessionLabel: &label},
			{MatchID: "m3", StartTime: now},
		},
	}
	svc := NewLastMatchService(repo)

	resp, err := svc.Resolve(context.Background(), domain.LastMatchResolveRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Default index = 0, sorted DESC → m3 is first (most recent)
	if resp.CurrentMatchID != "m3" {
		t.Errorf("CurrentMatchID = %q, want m3", resp.CurrentMatchID)
	}
	if resp.TotalMatchesInScope != 3 {
		t.Errorf("Total = %d, want 3", resp.TotalMatchesInScope)
	}
	if resp.PreviousMatchID != nil {
		t.Errorf("PreviousMatchID = %v, want nil (first)", resp.PreviousMatchID)
	}
	if resp.NextMatchID == nil || *resp.NextMatchID != "m2" {
		t.Errorf("NextMatchID = %v, want m2", resp.NextMatchID)
	}
}

func TestLastMatchService_Resolve_WithIndex(t *testing.T) {
	now := time.Now()
	repo := &mockStatsRepo{
		matches: []domain.StatsMatchRow{
			{MatchID: "m1", StartTime: now.Add(-2 * time.Hour)},
			{MatchID: "m2", StartTime: now.Add(-1 * time.Hour)},
			{MatchID: "m3", StartTime: now},
		},
	}
	svc := NewLastMatchService(repo)
	idx := 1

	resp, err := svc.Resolve(context.Background(), domain.LastMatchResolveRequest{
		CurrentIndex: &idx,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Sorted DESC: [m3, m2, m1], index 1 → m2
	if resp.CurrentMatchID != "m2" {
		t.Errorf("CurrentMatchID = %q, want m2", resp.CurrentMatchID)
	}
	if resp.PreviousMatchID == nil || *resp.PreviousMatchID != "m3" {
		t.Errorf("PreviousMatchID = %v, want m3", resp.PreviousMatchID)
	}
	if resp.NextMatchID == nil || *resp.NextMatchID != "m1" {
		t.Errorf("NextMatchID = %v, want m1", resp.NextMatchID)
	}
}

func TestLastMatchService_Resolve_SessionLabel(t *testing.T) {
	now := time.Now()
	label := "Session #42"
	repo := &mockStatsRepo{
		matches: []domain.StatsMatchRow{
			{MatchID: "m1", StartTime: now, SessionLabel: &label},
		},
	}
	svc := NewLastMatchService(repo)

	resp, err := svc.Resolve(context.Background(), domain.LastMatchResolveRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SessionTrackingKey != "Session #42" {
		t.Errorf("SessionTrackingKey = %q, want %q", resp.SessionTrackingKey, "Session #42")
	}
}

func TestLastMatchService_Resolve_EmptyMatches(t *testing.T) {
	repo := &mockStatsRepo{matches: []domain.StatsMatchRow{}}
	svc := NewLastMatchService(repo)

	_, err := svc.Resolve(context.Background(), domain.LastMatchResolveRequest{})
	if err == nil {
		t.Error("expected error for empty matches")
	}
}

func TestLastMatchService_Resolve_Error(t *testing.T) {
	repo := &mockStatsRepo{matchErr: errors.New("fail")}
	svc := NewLastMatchService(repo)

	_, err := svc.Resolve(context.Background(), domain.LastMatchResolveRequest{})
	if err == nil {
		t.Error("expected error")
	}
}
