package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// --- mock ---

type mockFiltersRepo struct {
	rows []domain.FilterMatchRow
	err  error
}

func (m *mockFiltersRepo) LoadMatchesForFilters(_ context.Context) ([]domain.FilterMatchRow, error) {
	return m.rows, m.err
}
func (m *mockFiltersRepo) GetMatchCount(_ context.Context) (int, error) { return len(m.rows), nil }
func (m *mockFiltersRepo) GetPlayerMatchCount(_ context.Context) (int, error) {
	return len(m.rows), nil
}
func (m *mockFiltersRepo) GetAvailablePlaylists(_ context.Context) ([]domain.LabelValue, error) {
	return nil, nil
}
func (m *mockFiltersRepo) GetAvailableMaps(_ context.Context) ([]domain.LabelValue, error) {
	return nil, nil
}

// --- tests FiltersService ---

func TestFiltersService_Resolve_OK(t *testing.T) {
	now := time.Now()
	repo := &mockFiltersRepo{
		rows: []domain.FilterMatchRow{
			{MatchID: "m1", StartTime: &now, MapName: strPtr("Aquarius"), PairName: strPtr("Slayer"), PlaylistName: strPtr("Ranked Arena")},
			{MatchID: "m2", StartTime: &now, MapName: strPtr("Streets"), PairName: strPtr("CTF"), PlaylistName: strPtr("Quick Play")},
		},
	}
	svc := NewFiltersService(repo)

	resp, err := svc.Resolve(context.Background(), domain.FilterContextInput{FilterMode: "period"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Counts.TotalMatchesBeforeFilters != 2 {
		t.Errorf("TotalMatchesBeforeFilters = %d, want 2", resp.Counts.TotalMatchesBeforeFilters)
	}
}

func TestFiltersService_Resolve_Empty(t *testing.T) {
	repo := &mockFiltersRepo{rows: []domain.FilterMatchRow{}}
	svc := NewFiltersService(repo)

	resp, err := svc.Resolve(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Counts.TotalMatchesBeforeFilters != 0 {
		t.Errorf("TotalMatchesBeforeFilters = %d, want 0", resp.Counts.TotalMatchesBeforeFilters)
	}
}

func TestFiltersService_Resolve_Error(t *testing.T) {
	repo := &mockFiltersRepo{err: errors.New("fail")}
	svc := NewFiltersService(repo)

	_, err := svc.Resolve(context.Background(), domain.FilterContextInput{})
	if err == nil {
		t.Error("expected error")
	}
}
