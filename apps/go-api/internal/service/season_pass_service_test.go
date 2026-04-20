package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// --- mocks ---

type mockSeasonPassRepo struct {
	tracks []domain.SeasonPassTrackSummary
	err    error
	calls  int
}

func (m *mockSeasonPassRepo) LoadSeasonPassTracks(_ context.Context, _, _ string) ([]domain.SeasonPassTrackSummary, error) {
	m.calls++
	return m.tracks, m.err
}

type mockHomeServiceForSP struct {
	challenges domain.ChallengesResponse
}

func (m *mockHomeServiceForSP) GetHomePage(_ context.Context, _ string) (*domain.HomePageResponse, error) {
	return nil, nil
}
func (m *mockHomeServiceForSP) GetBattlePass(_ context.Context) domain.BattlePassResponse {
	return domain.BattlePassResponse{}
}
func (m *mockHomeServiceForSP) GetChallenges(_ context.Context) domain.ChallengesResponse {
	return m.challenges
}

// --- tests ---

func TestSeasonPassService_GetSeasonPassPage_OK(t *testing.T) {
	activeTrack := "RewardTracks/Operations/OpFoo.json"
	repo := &mockSeasonPassRepo{
		tracks: []domain.SeasonPassTrackSummary{
			{RewardTrackPath: activeTrack, Name: "Op Foo", IsActive: true, CurrentRank: 15},
			{RewardTrackPath: "RewardTracks/Operations/OpBar.json", Name: "Op Bar", IsActive: false},
		},
	}
	total10, completed5 := 10, 5
	homeSvc := &mockHomeServiceForSP{
		challenges: domain.ChallengesResponse{Available: true, Total: &total10, Completed: &completed5},
	}
	svc := NewSeasonPassService(repo, homeSvc, "xuid-123", "HaloInfinite")

	resp, err := svc.GetSeasonPassPage(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Available {
		t.Error("expected Available=true")
	}
	if resp.TitleSlug != "HaloInfinite" {
		t.Errorf("TitleSlug = %q, want HaloInfinite", resp.TitleSlug)
	}
	if len(resp.Passes) != 2 {
		t.Errorf("Passes len = %d, want 2", len(resp.Passes))
	}
	if resp.ActiveTrackPath == nil || *resp.ActiveTrackPath != activeTrack {
		t.Errorf("ActiveTrackPath = %v, want %q", resp.ActiveTrackPath, activeTrack)
	}
	if resp.Challenges.Total == nil || *resp.Challenges.Total != total10 {
		t.Error("challenges not propagated")
	}
	if repo.calls != 1 {
		t.Errorf("repo.calls = %d, want 1", repo.calls)
	}
}

func TestSeasonPassService_GetSeasonPassPage_NoActiveTrack(t *testing.T) {
	repo := &mockSeasonPassRepo{
		tracks: []domain.SeasonPassTrackSummary{
			{RewardTrackPath: "RewardTracks/Operations/OpFoo.json", Name: "Op Foo", IsActive: false},
		},
	}
	svc := NewSeasonPassService(repo, &mockHomeServiceForSP{}, "xuid-123", "HaloInfinite")

	resp, err := svc.GetSeasonPassPage(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ActiveTrackPath != nil {
		t.Errorf("expected nil ActiveTrackPath, got %v", resp.ActiveTrackPath)
	}
	if !resp.Available {
		t.Error("expected Available=true (1 track present)")
	}
}

func TestSeasonPassService_GetSeasonPassPage_EmptyTracks(t *testing.T) {
	repo := &mockSeasonPassRepo{tracks: []domain.SeasonPassTrackSummary{}}
	svc := NewSeasonPassService(repo, &mockHomeServiceForSP{}, "xuid-123", "HaloInfinite")

	resp, err := svc.GetSeasonPassPage(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Available {
		t.Error("expected Available=false when no tracks")
	}
	if resp.ErrorHint == nil {
		t.Error("expected ErrorHint when no tracks")
	}
}

func TestSeasonPassService_GetSeasonPassPage_RepoError(t *testing.T) {
	repo := &mockSeasonPassRepo{err: errors.New("db unavailable")}
	svc := NewSeasonPassService(repo, &mockHomeServiceForSP{}, "xuid-123", "HaloInfinite")

	resp, err := svc.GetSeasonPassPage(context.Background())
	// l'erreur repo est absorbée en dégradation gracieuse
	if err != nil {
		t.Fatalf("unexpected error returned: %v", err)
	}
	if resp.Available {
		t.Error("expected Available=false on repo error")
	}
	if resp.ErrorHint == nil {
		t.Error("expected ErrorHint on repo error")
	}
	if resp.TitleSlug != "HaloInfinite" {
		t.Errorf("TitleSlug = %q", resp.TitleSlug)
	}
}

func TestSeasonPassService_ChallengesAlwaysPresent(t *testing.T) {
	repo := &mockSeasonPassRepo{err: errors.New("db error")}
	total7 := 7
	homeSvc := &mockHomeServiceForSP{
		challenges: domain.ChallengesResponse{Available: true, Total: &total7},
	}
	svc := NewSeasonPassService(repo, homeSvc, "xuid-123", "HaloInfinite")

	resp, _ := svc.GetSeasonPassPage(context.Background())
	if resp.Challenges.Total == nil || *resp.Challenges.Total != total7 {
		t.Error("challenges should be propagated even on repo error")
	}
}


