package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// --- mock ---

type mockHomeRepo struct {
	matches    []domain.HomeMatchRow
	matchErr   error
	matchCount int
	countErr   error
	sessions   []domain.HomeSessionRow
	sessionErr error
	media      []domain.HomeMediaRow
	mediaErr   error
}

func (m *mockHomeRepo) LoadHomeMatches(_ context.Context) ([]domain.HomeMatchRow, error) {
	return m.matches, m.matchErr
}
func (m *mockHomeRepo) CountPlayerMatches(_ context.Context) (int, error) {
	if m.matchCount > 0 {
		return m.matchCount, m.countErr
	}
	return len(m.matches), m.countErr
}
func (m *mockHomeRepo) LoadHomeSessions(_ context.Context) ([]domain.HomeSessionRow, error) {
	return m.sessions, m.sessionErr
}
func (m *mockHomeRepo) LoadRecentMedia(_ context.Context, _ int) ([]domain.HomeMediaRow, error) {
	return m.media, m.mediaErr
}

// --- tests ---

func TestHomeService_GetHomePage_OK(t *testing.T) {
	now := time.Now()
	repo := &mockHomeRepo{
		matches: []domain.HomeMatchRow{
			{MatchID: "m1", StartTime: now, MapName: "Aquarius", PairName: "Slayer", Outcome: 2, Kills: 10, Deaths: 5, Assists: 3},
			{MatchID: "m2", StartTime: now.Add(-1 * time.Hour), MapName: "Streets", PairName: "CTF", Outcome: 3, Kills: 5, Deaths: 10, Assists: 1},
		},
		sessions: []domain.HomeSessionRow{
			{MatchID: "m1", SessionLabel: strPtr("Session 1"), StartTime: &now},
		},
		media: []domain.HomeMediaRow{
			{FileName: "clip1.mp4"},
		},
	}
	svc := NewHomeService(repo)

	resp, err := svc.GetHomePage(context.Background(), "TestGT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestHomeService_GetHomePage_Empty(t *testing.T) {
	repo := &mockHomeRepo{
		matches:  []domain.HomeMatchRow{},
		sessions: []domain.HomeSessionRow{},
		media:    []domain.HomeMediaRow{},
	}
	svc := NewHomeService(repo)

	resp, err := svc.GetHomePage(context.Background(), "TestGT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response even with empty data")
	}
}

func TestHomeService_GetHomePage_MatchesError(t *testing.T) {
	repo := &mockHomeRepo{matchErr: errors.New("fail")}
	svc := NewHomeService(repo)

	_, err := svc.GetHomePage(context.Background(), "GT")
	if err == nil {
		t.Error("expected error when matches fail")
	}
}

func TestHomeService_GetHomePage_SessionsError(t *testing.T) {
	repo := &mockHomeRepo{
		matches:    []domain.HomeMatchRow{{MatchID: "m1", Outcome: 2, StartTime: time.Now()}},
		sessionErr: errors.New("fail"),
	}
	svc := NewHomeService(repo)

	_, err := svc.GetHomePage(context.Background(), "GT")
	if err == nil {
		t.Error("expected error when sessions fail")
	}
}

func TestHomeService_GetHomePage_MediaGraceful(t *testing.T) {
	now := time.Now()
	repo := &mockHomeRepo{
		matches:  []domain.HomeMatchRow{{MatchID: "m1", Outcome: 2, Kills: 10, Deaths: 5, StartTime: now}},
		sessions: []domain.HomeSessionRow{},
		mediaErr: errors.New("media unavailable"),
	}
	svc := NewHomeService(repo)

	resp, err := svc.GetHomePage(context.Background(), "GT")
	if err != nil {
		t.Fatalf("expected graceful degradation on media error, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestHomeService_GetBattlePass(t *testing.T) {
	repo := &mockHomeRepo{}
	svc := NewHomeService(repo)
	bp := svc.GetBattlePass(context.Background())
	// Default provider returns available=false
	if bp.Available {
		t.Error("expected Available=false from default provider")
	}
}

func TestHomeService_GetChallenges(t *testing.T) {
	repo := &mockHomeRepo{}
	svc := NewHomeService(repo)
	ch := svc.GetChallenges(context.Background())
	if ch.Available {
		t.Error("expected Available=false from default provider")
	}
}

// ---------------------------------------------------------------------------
// Tests cache-first (Phase A)
// ---------------------------------------------------------------------------

// mockBattlePassCacheRepo implémente port.BattlePassCacheRepository pour les tests.
type mockBattlePassCacheRepo struct {
	bpResp  *domain.BattlePassResponse
	bpHit   bool
	bpErr   error
	chResp  *domain.ChallengesResponse
	chHit   bool
	chErr   error
	bpCalls int
	chCalls int
}

func (m *mockBattlePassCacheRepo) LoadCachedBattlePass(_ context.Context, _ time.Duration) (*domain.BattlePassResponse, bool, error) {
	m.bpCalls++
	return m.bpResp, m.bpHit, m.bpErr
}

func (m *mockBattlePassCacheRepo) LoadCachedChallenges(_ context.Context, _ time.Duration) (*domain.ChallengesResponse, bool, error) {
	m.chCalls++
	return m.chResp, m.chHit, m.chErr
}

// stubProviderHit est un provider nul qui panique si appelé (vérifie que le cache a pris le dessus).
// (utilise la valeur par défaut de HaloProvider qui retourne available=false)

func TestHomeService_GetBattlePass_CacheHit(t *testing.T) {
	rank := 42
	track := "RewardTracks/Operations/Season1"
	progress := 100
	cacheRepo := &mockBattlePassCacheRepo{
		bpResp: &domain.BattlePassResponse{
			Available:   true,
			Rank:        &rank,
			RewardTrack: &track,
			Progress:    &progress,
			FromCache:   true,
		},
		bpHit: true,
	}

	svc := NewHomeService(&mockHomeRepo{}).WithCacheRepo(cacheRepo)
	resp := svc.GetBattlePass(context.Background())

	if !resp.Available {
		t.Error("expected Available=true from cache")
	}
	if !resp.FromCache {
		t.Error("expected FromCache=true")
	}
	if resp.Rank == nil || *resp.Rank != 42 {
		t.Errorf("expected rank 42, got %v", resp.Rank)
	}
	if cacheRepo.bpCalls != 1 {
		t.Errorf("expected 1 cache call, got %d", cacheRepo.bpCalls)
	}
}

func TestHomeService_GetBattlePass_CacheMiss(t *testing.T) {
	cacheRepo := &mockBattlePassCacheRepo{
		bpResp: nil,
		bpHit:  false,
	}

	svc := NewHomeService(&mockHomeRepo{}).WithCacheRepo(cacheRepo)
	// Le provider live retourne available=false (pas de tokens)
	resp := svc.GetBattlePass(context.Background())

	// Cache miss → live provider appelé → available=false (pas de tokens)
	if resp.FromCache {
		t.Error("expected FromCache=false on cache miss")
	}
	if cacheRepo.bpCalls != 1 {
		t.Errorf("expected 1 cache call, got %d", cacheRepo.bpCalls)
	}
}

func TestHomeService_GetChallenges_CacheHit(t *testing.T) {
	total := 10
	completed := 5
	xp := 3000
	current, target := 2, 5
	cacheRepo := &mockBattlePassCacheRepo{
		chResp: &domain.ChallengesResponse{
			Available:   true,
			Total:       &total,
			Completed:   &completed,
			XPAvailable: &xp,
			Items: []domain.ChallengeItem{{
				ChallengePath:   "Challenges/Tracking/test-1",
				Title:           "Défi test",
				ProgressCurrent: &current,
				ProgressTarget:  &target,
			}},
			FromCache: true,
		},
		chHit: true,
	}

	svc := NewHomeService(&mockHomeRepo{}).WithCacheRepo(cacheRepo)
	resp := svc.GetChallenges(context.Background())

	if !resp.Available {
		t.Error("expected Available=true from cache")
	}
	if !resp.FromCache {
		t.Error("expected FromCache=true")
	}
	if resp.Total == nil || *resp.Total != 10 {
		t.Errorf("expected total 10, got %v", resp.Total)
	}
}

func TestHomeService_GetChallenges_CacheHitWithoutItems_FallsBackLive(t *testing.T) {
	total := 9
	completed := 2
	cacheRepo := &mockBattlePassCacheRepo{
		chResp: &domain.ChallengesResponse{
			Available: true,
			Total:     &total,
			Completed: &completed,
			FromCache: true,
		},
		chHit: true,
	}

	svc := NewHomeService(&mockHomeRepo{}).WithCacheRepo(cacheRepo)
	resp := svc.GetChallenges(context.Background())

	if resp.FromCache {
		t.Error("expected FromCache=false when cache lacks active challenge items")
	}
	if cacheRepo.chCalls != 1 {
		t.Errorf("expected 1 cache call, got %d", cacheRepo.chCalls)
	}
}

func TestHomeService_GetChallenges_CacheMiss(t *testing.T) {
	cacheRepo := &mockBattlePassCacheRepo{
		chHit: false,
	}

	svc := NewHomeService(&mockHomeRepo{}).WithCacheRepo(cacheRepo)
	resp := svc.GetChallenges(context.Background())

	if resp.FromCache {
		t.Error("expected FromCache=false on cache miss")
	}
	if cacheRepo.chCalls != 1 {
		t.Errorf("expected 1 cache call, got %d", cacheRepo.chCalls)
	}
}

func TestHomeService_GetBattlePass_NoCacheRepo(t *testing.T) {
	// Sans WithCacheRepo → live direct, pas de panique
	svc := NewHomeService(&mockHomeRepo{})
	resp := svc.GetBattlePass(context.Background())
	if resp.FromCache {
		t.Error("expected FromCache=false when no cache repo")
	}
}
