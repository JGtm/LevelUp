package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

// --- mock ---

type mockLeaderboardRepo struct {
	entries []domain.LeaderboardEntry
	err     error
}

func (m *mockLeaderboardRepo) GetLocalLeaderboard(_ context.Context, _, _, _ string) ([]domain.LeaderboardEntry, error) {
	return m.entries, m.err
}

// --- tests ---

func TestLeaderboardService_LocalFirst(t *testing.T) {
	entries := []domain.LeaderboardEntry{
		{Gamertag: "Alice", CSR: 1800, IsLocal: true},
		{Gamertag: "Bob", CSR: 1700, IsLocal: true},
		{Gamertag: "Charlie", CSR: 1600, IsLocal: false},
	}

	svc := NewLeaderboardService(&mockLeaderboardRepo{entries: entries})
	resp, err := svc.GetPage(context.Background(), domain.LeaderboardRequest{TitleSlug: "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(resp.Entries))
	}

	// Les rangs doivent être renumérotés à partir de 1.
	for i, e := range resp.Entries {
		if e.Rank != i+1 {
			t.Errorf("entry %d: expected rank %d, got %d", i, i+1, e.Rank)
		}
	}

	// Local en premier.
	if !resp.Entries[0].IsLocal {
		t.Error("expected first entry to be local")
	}
}

func TestLeaderboardService_Empty(t *testing.T) {
	svc := NewLeaderboardService(&mockLeaderboardRepo{entries: []domain.LeaderboardEntry{}})
	resp, err := svc.GetPage(context.Background(), domain.LeaderboardRequest{TitleSlug: "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(resp.Entries))
	}
}
