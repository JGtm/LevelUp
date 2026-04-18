package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// --- mocks ---

type mockCompareRepo struct {
	stats    *domain.NormalizedPlayerStats
	statsErr error
	xuid     string
	xuidErr  error
}

func (m *mockCompareRepo) GetLocalStats(_ context.Context, _, _ string) (*domain.NormalizedPlayerStats, error) {
	return m.stats, m.statsErr
}

func (m *mockCompareRepo) ResolveXUID(_ context.Context, _ string) (string, error) {
	return m.xuid, m.xuidErr
}

type mockStatsProvider struct {
	stats    *domain.NormalizedPlayerStats
	statsErr error
}

func (m *mockStatsProvider) FetchRemoteStats(_ context.Context, _, _ string) (*domain.NormalizedPlayerStats, error) {
	return m.stats, m.statsErr
}

// --- tests ---

func TestCompareService_BothLocal(t *testing.T) {
	statsA := &domain.NormalizedPlayerStats{
		Gamertag: "PlayerA",
		Matches:  100,
		WinRate:  0.60,
		KDR:      1.5,
	}
	statsB := &domain.NormalizedPlayerStats{
		Gamertag: "PlayerB",
		Matches:  80,
		WinRate:  0.50,
		KDR:      1.2,
	}

	callCount := 0
	repo := &mockCompareRepo{
		xuid: "xuid-b",
	}
	// GetLocalStats retourne A pour xuid "xuid-a", B pour xuid "xuid-b"
	repo.stats = statsA

	provider := &mockStatsProvider{statsErr: errors.New("not needed")}

	svc := NewCompareService(&mockCompareRepoAB{a: statsA, b: statsB}, provider, "xuid-a", "hi")
	_ = callCount

	resp, err := svc.GetPage(context.Background(), domain.CompareRequest{TargetGamertag: "PlayerB"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PlayerA.Gamertag != "PlayerA" {
		t.Errorf("expected PlayerA, got %s", resp.PlayerA.Gamertag)
	}
	if len(resp.Metrics) == 0 {
		t.Error("expected metrics to be populated")
	}
}

func TestCompareService_PlayerBNotFound(t *testing.T) {
	statsA := &domain.NormalizedPlayerStats{
		Gamertag: "PlayerA",
		Matches:  50,
	}

	repo := &mockCompareRepoAB{
		a:       statsA,
		bErr:    errors.New("not found"),
		xuidErr: errors.New("not found"),
	}
	provider := &mockStatsProvider{statsErr: errors.New("not found")}

	svc := NewCompareService(repo, provider, "xuid-a", "hi")
	_, err := svc.GetPage(context.Background(), domain.CompareRequest{TargetGamertag: "UnknownPlayer"})
	if err == nil {
		t.Error("expected error when player B not found, got nil")
	}
}

// mockCompareRepoAB — retourne stats différentes selon le xuid demandé.
type mockCompareRepoAB struct {
	a       *domain.NormalizedPlayerStats
	b       *domain.NormalizedPlayerStats
	bErr    error
	xuid    string
	xuidErr error
	calls   int
}

func (m *mockCompareRepoAB) GetLocalStats(_ context.Context, xuid, _ string) (*domain.NormalizedPlayerStats, error) {
	m.calls++
	if xuid == "xuid-b" || (m.a != nil && xuid != "xuid-a") {
		if m.bErr != nil {
			return nil, m.bErr
		}
		return m.b, nil
	}
	return m.a, nil
}

func (m *mockCompareRepoAB) ResolveXUID(_ context.Context, _ string) (string, error) {
	if m.xuidErr != nil {
		return "", m.xuidErr
	}
	if m.xuid != "" {
		return m.xuid, nil
	}
	return "xuid-b", nil
}
