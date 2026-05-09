package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

const slayerMode = "Slayer"

// --- mock ---

type mockMatchHistoryRepo struct {
	rows     []domain.MatchHistoryRawRow
	loadErr  error
	winRates map[string][2]int
	winErr   error
}

func (m *mockMatchHistoryRepo) LoadAll(_ context.Context) ([]domain.MatchHistoryRawRow, error) {
	return m.rows, m.loadErr
}
func (m *mockMatchHistoryRepo) LoadMapWinRates(_ context.Context) (map[string][2]int, error) {
	return m.winRates, m.winErr
}

// --- tests ---

func TestMatchHistoryService_GetPage_OK(t *testing.T) {
	now := time.Now()
	mapName := aquariusMap
	pairName := slayerMode
	repo := &mockMatchHistoryRepo{
		rows: []domain.MatchHistoryRawRow{
			{MatchID: "m1", StartTime: &now, MapName: &mapName, PairName: &pairName, Outcome: 2, Kills: 10, Deaths: 5},
			{MatchID: "m2", StartTime: &now, MapName: &mapName, PairName: &pairName, Outcome: 3, Kills: 5, Deaths: 10},
		},
	}
	svc := NewMatchHistoryService(repo, "TestPlayer")

	resp, err := svc.GetPage(context.Background(), domain.MatchHistoryQueryRequest{
		Pagination: domain.PaginationRequest{Page: 1, PageSize: 20},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Summary.TotalMatchesScoped != 2 {
		t.Errorf("TotalMatchesScoped = %d, want 2", resp.Summary.TotalMatchesScoped)
	}
	if len(resp.Table.Items) != 2 {
		t.Errorf("Items = %d, want 2", len(resp.Table.Items))
	}
}

func TestMatchHistoryService_GetPage_PropagatesIsWithFriendsAndExperience(t *testing.T) {
	now := time.Now()
	mapName := aquariusMap
	pairName := slayerMode
	repo := &mockMatchHistoryRepo{
		rows: []domain.MatchHistoryRawRow{
			{MatchID: "solo", StartTime: &now, MapName: &mapName, PairName: &pairName, IsWithFriends: false, IsRanked: false, Outcome: 2},
			{MatchID: "squad", StartTime: &now, MapName: &mapName, PairName: &pairName, IsWithFriends: true, IsRanked: true, Outcome: 2},
			{MatchID: "pve", StartTime: &now, MapName: &mapName, PairName: &pairName, IsWithFriends: false, IsFirefight: true, Outcome: 2},
		},
	}
	svc := NewMatchHistoryService(repo, "Player")

	resp, err := svc.GetPage(context.Background(), domain.MatchHistoryQueryRequest{
		Pagination: domain.PaginationRequest{Page: 1, PageSize: 20},
		SortField:  "match_id",
		SortDir:    "asc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	byID := map[string]domain.MatchHistoryRow{}
	for _, it := range resp.Table.Items {
		byID[it.MatchID] = it
	}
	if got := byID["solo"]; got.IsWithFriends || got.ExperienceTypeLabel != "PVP non classé" {
		t.Errorf("solo: IsWithFriends=%v ExperienceTypeLabel=%q", got.IsWithFriends, got.ExperienceTypeLabel)
	}
	if got := byID["squad"]; !got.IsWithFriends || got.ExperienceTypeLabel != "PVP classé" {
		t.Errorf("squad: IsWithFriends=%v ExperienceTypeLabel=%q", got.IsWithFriends, got.ExperienceTypeLabel)
	}
	if got := byID["pve"]; got.IsWithFriends || got.ExperienceTypeLabel != "PVE" {
		t.Errorf("pve: IsWithFriends=%v ExperienceTypeLabel=%q", got.IsWithFriends, got.ExperienceTypeLabel)
	}
}

func TestMatchHistoryService_GetPage_Empty(t *testing.T) {
	repo := &mockMatchHistoryRepo{rows: []domain.MatchHistoryRawRow{}}
	svc := NewMatchHistoryService(repo, "Player")

	resp, err := svc.GetPage(context.Background(), domain.MatchHistoryQueryRequest{
		Pagination: domain.PaginationRequest{Page: 1, PageSize: 20},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Summary.TotalMatchesScoped != 0 {
		t.Errorf("TotalMatchesScoped = %d, want 0", resp.Summary.TotalMatchesScoped)
	}
}

func TestMatchHistoryService_GetPage_Error(t *testing.T) {
	repo := &mockMatchHistoryRepo{loadErr: errors.New("fail")}
	svc := NewMatchHistoryService(repo, "Player")

	_, err := svc.GetPage(context.Background(), domain.MatchHistoryQueryRequest{})
	if err == nil {
		t.Error("expected error")
	}
}

func TestMatchHistoryService_GetPage_WithExportHint(t *testing.T) {
	now := time.Now()
	repo := &mockMatchHistoryRepo{
		rows: []domain.MatchHistoryRawRow{
			{MatchID: "m1", StartTime: &now, Outcome: 2},
		},
	}
	svc := NewMatchHistoryService(repo, "Player")

	resp, err := svc.GetPage(context.Background(), domain.MatchHistoryQueryRequest{
		Pagination:        domain.PaginationRequest{Page: 1, PageSize: 20},
		IncludeExportHint: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ExportHint == nil {
		t.Error("expected ExportHint to be set")
	}
}

func TestMatchHistoryService_ExportCSV_OK(t *testing.T) {
	now := time.Now()
	repo := &mockMatchHistoryRepo{
		rows: []domain.MatchHistoryRawRow{
			{MatchID: "m1", StartTime: &now, Outcome: 2, Kills: 10, Deaths: 3},
			{MatchID: "m2", StartTime: &now, Outcome: 3, Kills: 5, Deaths: 8},
		},
	}
	svc := NewMatchHistoryService(repo, "Player")

	items, err := svc.ExportCSV(context.Background(), domain.MatchHistoryQueryRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("len = %d, want 2", len(items))
	}
}

func TestMatchHistoryService_ExportCSV_Error(t *testing.T) {
	repo := &mockMatchHistoryRepo{loadErr: errors.New("fail")}
	svc := NewMatchHistoryService(repo, "Player")

	_, err := svc.ExportCSV(context.Background(), domain.MatchHistoryQueryRequest{})
	if err == nil {
		t.Error("expected error")
	}
}

// --- filterExcludedRows ---

func TestFilterExcludedRows(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		rows    []domain.MatchHistoryRawRow
		wantLen int
	}{
		{
			name:    "aucune exclusion",
			rows:    []domain.MatchHistoryRawRow{{MatchID: "m1", StartTime: &now, IsExcluded: false}},
			wantLen: 1,
		},
		{
			name: "une exclusion",
			rows: []domain.MatchHistoryRawRow{
				{MatchID: "m1", StartTime: &now, IsExcluded: false},
				{MatchID: "m2", StartTime: &now, IsExcluded: true},
			},
			wantLen: 1,
		},
		{
			name:    "toutes exclues",
			rows:    []domain.MatchHistoryRawRow{{MatchID: "m1", StartTime: &now, IsExcluded: true}},
			wantLen: 0,
		},
		{
			name:    "liste vide",
			rows:    []domain.MatchHistoryRawRow{},
			wantLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := filterExcludedRows(tt.rows)
			if len(out) != tt.wantLen {
				t.Errorf("filterExcludedRows(%q): got %d rows, want %d", tt.name, len(out), tt.wantLen)
			}
		})
	}
}

func TestMatchHistoryService_GetPage_ExcludedFiltered(t *testing.T) {
	now := time.Now()
	repo := &mockMatchHistoryRepo{
		rows: []domain.MatchHistoryRawRow{
			{MatchID: "m1", StartTime: &now, Outcome: 2, IsExcluded: true},
			{MatchID: "m2", StartTime: &now, Outcome: 3, IsExcluded: true},
		},
	}
	svc := NewMatchHistoryService(repo, "Player")

	resp, err := svc.GetPage(context.Background(), domain.MatchHistoryQueryRequest{
		Pagination: domain.PaginationRequest{Page: 1, PageSize: 20},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Summary.TotalMatchesScoped != 0 {
		t.Errorf("TotalMatchesScoped = %d, want 0 (all excluded)", resp.Summary.TotalMatchesScoped)
	}
	if len(resp.Table.Items) != 0 {
		t.Errorf("Items = %d, want 0 (all excluded)", len(resp.Table.Items))
	}
}
