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
	matches                []domain.HomeMatchRow
	matchErr               error
	identity               *domain.HomeSpartanIdentityRow
	identityErr            error
	matchCount             int
	countErr               error
	sessions               []domain.HomeSessionRow
	sessionErr             error
	media                  []domain.HomeMediaRow
	mediaErr               error
	recentPlaylistRanks    []domain.HomePlaylistRank
	recentPlaylistRanksErr error
}

func (m *mockHomeRepo) LoadHomeMatches(_ context.Context) ([]domain.HomeMatchRow, error) {
	return m.matches, m.matchErr
}
func (m *mockHomeRepo) LoadSpartanIdentity(_ context.Context) (*domain.HomeSpartanIdentityRow, error) {
	return m.identity, m.identityErr
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
func (m *mockHomeRepo) LoadRecentPlaylistRanks(_ context.Context) ([]domain.HomePlaylistRank, error) {
	return m.recentPlaylistRanks, m.recentPlaylistRanksErr
}

// --- tests ---

func TestHomeService_GetHomePage_OK(t *testing.T) {
	now := time.Now()
	repo := &mockHomeRepo{
		matches: []domain.HomeMatchRow{
			{MatchID: "m1", StartTime: now, MapName: "Aquarius", PairName: "Slayer", Outcome: 2, Kills: 10, Deaths: 5, Assists: 3, IsRanked: true},
			{MatchID: "m2", StartTime: now.Add(-1 * time.Hour), MapName: "Streets", PairName: "CTF", Outcome: 3, Kills: 5, Deaths: 10, Assists: 1, IsRanked: false},
		},
		sessions: []domain.HomeSessionRow{
			{MatchID: "m1", SessionLabel: strPtr("Session 1"), StartTime: &now},
		},
		media: []domain.HomeMediaRow{
			{FileName: "clip1.mp4"},
		},
	}
	svc := NewHomeService(repo)

	resp, err := svc.GetHomePage(context.Background(), "TestGT", "fr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if !resp.HasRankedHistory {
		t.Fatal("expected HasRankedHistory")
	}
	if !resp.HasUnrankedHistory {
		t.Fatal("expected HasUnrankedHistory")
	}
}

func TestHomeService_GetHomePage_Empty(t *testing.T) {
	repo := &mockHomeRepo{
		matches:  []domain.HomeMatchRow{},
		sessions: []domain.HomeSessionRow{},
		media:    []domain.HomeMediaRow{},
	}
	svc := NewHomeService(repo)

	resp, err := svc.GetHomePage(context.Background(), "TestGT", "fr")
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

	_, err := svc.GetHomePage(context.Background(), "GT", "fr")
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

	_, err := svc.GetHomePage(context.Background(), "GT", "fr")
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

	resp, err := svc.GetHomePage(context.Background(), "GT", "fr")
	if err != nil {
		t.Fatalf("expected graceful degradation on media error, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestHomeService_GetHomePage_RespectsLocale(t *testing.T) {
	now := time.Now()
	repo := &mockHomeRepo{
		matches: []domain.HomeMatchRow{{
			MatchID:        "m1",
			StartTime:      now,
			MapName:        "Bazaar",
			MapNameFR:      "Bazaar",
			PairName:       "Team Slayer on Bazaar",
			PairNameFR:     "Slayer en équipe sur Bazaar",
			PlaylistName:   "Quick Play",
			PlaylistNameFR: "Partie rapide",
			Outcome:        2,
		}},
	}
	svc := NewHomeService(repo)

	respFR, err := svc.GetHomePage(context.Background(), "GT", "fr")
	if err != nil {
		t.Fatalf("unexpected FR error: %v", err)
	}
	if got := *respFR.RecentMatches[0].ModeUI; got != "Slayer en équipe" {
		t.Fatalf("FR ModeUI = %q, want %q", got, "Slayer en équipe")
	}
	if got := *respFR.RecentMatches[0].PlaylistUI; got != "Partie rapide" {
		t.Fatalf("FR PlaylistUI = %q, want %q", got, "Partie rapide")
	}
	if got := respFR.RecentMatches[0].OutcomeLabel; got != "Victoire" {
		t.Fatalf("FR OutcomeLabel = %q, want %q", got, "Victoire")
	}

	respEN, err := svc.GetHomePage(context.Background(), "GT", "en")
	if err != nil {
		t.Fatalf("unexpected EN error: %v", err)
	}
	if got := *respEN.RecentMatches[0].ModeUI; got != "Team Slayer" {
		t.Fatalf("EN ModeUI = %q, want %q", got, "Team Slayer")
	}
	if got := *respEN.RecentMatches[0].PlaylistUI; got != "Quick Play" {
		t.Fatalf("EN PlaylistUI = %q, want %q", got, "Quick Play")
	}
	if got := respEN.RecentMatches[0].OutcomeLabel; got != "Victory" {
		t.Fatalf("EN OutcomeLabel = %q, want %q", got, "Victory")
	}
}

func TestHomeService_GetHomePage_IncludesSpartanIdentity(t *testing.T) {
	repo := &mockHomeRepo{
		identity: &domain.HomeSpartanIdentityRow{
			SpartanID:         strPtr("JGTM"),
			BannerImageURL:    strPtr("https://example.test/banner.png"),
			EmblemImageURL:    strPtr("https://example.test/emblem.png"),
			BackdropImageURL:  strPtr("https://example.test/backdrop.png"),
			RankNumber:        25,
			RankTitleFR:       strPtr("Caporal-chef"),
			RankTitleEN:       strPtr("Lance Corporal"),
			RankImageURL:      strPtr("https://example.test/rank.png"),
			AdornmentImageURL: strPtr("https://example.test/adornment.png"),
			CurrentXP:         5000,
			XPForNextRank:     10000,
		},
	}
	svc := NewHomeService(repo)

	resp, err := svc.GetHomePage(context.Background(), "GT", "fr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SpartanIdentity == nil || resp.SpartanIdentity.SpartanID == nil {
		t.Fatal("expected spartan_identity with spartan_id")
	}
	if got := *resp.SpartanIdentity.SpartanID; got != "JGTM" {
		t.Fatalf("spartan_id = %q, want JGTM", got)
	}
	if resp.SpartanIdentity.EmblemImageURL == nil || *resp.SpartanIdentity.EmblemImageURL != "https://example.test/emblem.png" {
		t.Fatalf("emblem_image_url = %#v, want https://example.test/emblem.png", resp.SpartanIdentity.EmblemImageURL)
	}
	if resp.SpartanIdentity.BannerImageURL == nil || *resp.SpartanIdentity.BannerImageURL != "https://example.test/banner.png" {
		t.Fatalf("banner_image_url = %#v, want https://example.test/banner.png", resp.SpartanIdentity.BannerImageURL)
	}
	if resp.SpartanIdentity.BackdropImageURL == nil || *resp.SpartanIdentity.BackdropImageURL != "https://example.test/backdrop.png" {
		t.Fatalf("backdrop_image_url = %#v, want https://example.test/backdrop.png", resp.SpartanIdentity.BackdropImageURL)
	}
	if resp.SpartanIdentity.CareerRank == nil {
		t.Fatal("expected career_rank")
	}
	if got := resp.SpartanIdentity.CareerRank.RankTitle; got != "Caporal-chef" {
		t.Fatalf("rank_title = %q, want Caporal-chef", got)
	}
	if resp.SpartanIdentity.CareerRank.RankImageURL == nil || *resp.SpartanIdentity.CareerRank.RankImageURL != "https://example.test/rank.png" {
		t.Fatalf("rank_image_url = %#v, want https://example.test/rank.png", resp.SpartanIdentity.CareerRank.RankImageURL)
	}
	if resp.SpartanIdentity.CareerRank.AdornmentImageURL == nil || *resp.SpartanIdentity.CareerRank.AdornmentImageURL != "https://example.test/adornment.png" {
		t.Fatalf("adornment_image_url = %#v, want https://example.test/adornment.png", resp.SpartanIdentity.CareerRank.AdornmentImageURL)
	}
	if got := resp.SpartanIdentity.CareerRank.ProgressPct; got != 50 {
		t.Fatalf("progress_pct = %.2f, want 50", got)
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
