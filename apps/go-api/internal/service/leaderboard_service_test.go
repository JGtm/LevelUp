package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

// --- mock ---

type mockLeaderboardRepo struct {
	csrWorld []domain.LeaderboardEntry
	stats    []domain.LeaderboardEntry
	local    []domain.LeaderboardEntry
	err      error

	lastCategory domain.LeaderboardCategory
	lastLimit    int
}

func (m *mockLeaderboardRepo) GetLocalLeaderboard(_ context.Context, _, _, _ string) ([]domain.LeaderboardEntry, error) {
	return m.local, m.err
}

func (m *mockLeaderboardRepo) GetCSRWorldLeaderboard(_ context.Context, _, _ string, limit int) ([]domain.LeaderboardEntry, error) {
	m.lastCategory = domain.LeaderboardCSRWorld
	m.lastLimit = limit
	return m.csrWorld, m.err
}

func (m *mockLeaderboardRepo) GetStatLeaderboard(_ context.Context, category domain.LeaderboardCategory, _ string, limit int) ([]domain.LeaderboardEntry, error) {
	m.lastCategory = category
	m.lastLimit = limit
	return m.stats, m.err
}

// --- tests ---

// La catégorie par défaut route vers le classement CSR mondial.
func TestLeaderboardService_DefaultsToCSRWorld(t *testing.T) {
	entries := []domain.LeaderboardEntry{
		{Rank: 1, Gamertag: "Twissted Mindss", CSRValue: 2180},
		{Rank: 2, Gamertag: "OR81TAL", CSRValue: 2097},
	}
	repo := &mockLeaderboardRepo{csrWorld: entries}
	svc := NewLeaderboardService(repo)

	resp, err := svc.GetPage(context.Background(), domain.LeaderboardRequest{Season: "s", Playlist: "p"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastCategory != domain.LeaderboardCSRWorld {
		t.Errorf("expected csr-world routing, got %q", repo.lastCategory)
	}
	if resp.Category != string(domain.LeaderboardCSRWorld) {
		t.Errorf("expected response category csr-world, got %q", resp.Category)
	}
	if repo.lastLimit != defaultLeaderboardLimit {
		t.Errorf("expected default limit %d, got %d", defaultLeaderboardLimit, repo.lastLimit)
	}
	if len(resp.Entries) != 2 || resp.Entries[0].Gamertag != "Twissted Mindss" {
		t.Fatalf("unexpected entries: %+v", resp.Entries)
	}
}

// Une catégorie de stat route vers l'agrégation match_participants.
func TestLeaderboardService_StatCategory(t *testing.T) {
	repo := &mockLeaderboardRepo{stats: []domain.LeaderboardEntry{{Rank: 1, Gamertag: "Alice", Value: 42}}}
	svc := NewLeaderboardService(repo)

	resp, err := svc.GetPage(context.Background(), domain.LeaderboardRequest{
		Category: string(domain.LeaderboardKDA), Limit: 25,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastCategory != domain.LeaderboardKDA {
		t.Errorf("expected kda routing, got %q", repo.lastCategory)
	}
	if repo.lastLimit != 25 {
		t.Errorf("expected limit 25, got %d", repo.lastLimit)
	}
	if len(resp.Entries) != 1 || resp.Entries[0].Gamertag != "Alice" {
		t.Fatalf("unexpected entries: %+v", resp.Entries)
	}
}

func TestLeaderboardService_Empty(t *testing.T) {
	svc := NewLeaderboardService(&mockLeaderboardRepo{csrWorld: []domain.LeaderboardEntry{}})
	resp, err := svc.GetPage(context.Background(), domain.LeaderboardRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(resp.Entries))
	}
}
