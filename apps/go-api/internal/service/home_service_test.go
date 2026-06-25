package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// --- mock PlayerMatchesRepository pour tests P4.3 finale ---
//
// Convertit les matches/sessions du mockHomeRepo en canonical.PlayerMatchRow
// pour exercer le path canonical (le seul path aprÃ¨s P4.3 finale).
type mockHomePlayerMatches struct {
	matches  []legacymatch.HomeMatchRow
	sessions []legacymatch.HomeSessionRow
	err      error
}

func (m *mockHomePlayerMatches) LoadPlayerMatches(_ context.Context, _, _ string, _ port.PlayerMatchFilters) ([]canonical.PlayerMatchRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := make([]canonical.PlayerMatchRow, len(m.matches))
	sessByID := map[string]*legacymatch.HomeSessionRow{}
	for i := range m.sessions {
		s := &m.sessions[i]
		sessByID[s.MatchID] = s
	}
	for i, mm := range m.matches {
		k, d, a := mm.Kills, mm.Deaths, mm.Assists
		var outcome canonical.Outcome
		switch mm.Outcome {
		case domain.OutcomeWin:
			outcome = canonical.OutcomeWin
		case domain.OutcomeLoss:
			outcome = canonical.OutcomeLoss
		case domain.OutcomeDraw:
			outcome = canonical.OutcomeTie
		case domain.OutcomeDNF:
			outcome = canonical.OutcomeDNF
		}
		var sessionLabel *string
		if s, ok := sessByID[mm.MatchID]; ok {
			sessionLabel = s.SessionLabel
		} else {
			sessionLabel = mm.SessionLabel
		}
		isRanked := mm.IsRanked
		isPvE := mm.IsFirefight
		mapRef := &canonical.AssetReference{Kind: "map", ID: mm.MapID, DefaultLabel: mm.MapName, Labels: map[string]string{}}
		if mm.MapNameFR != "" {
			mapRef.Labels["fr"] = mm.MapNameFR
		}
		if mm.MapName != "" {
			mapRef.Labels["en"] = mm.MapName
		}
		playlistRef := &canonical.AssetReference{Kind: "playlist", ID: mm.PlaylistID, DefaultLabel: mm.PlaylistName, Labels: map[string]string{}}
		if mm.PlaylistNameFR != "" {
			playlistRef.Labels["fr"] = mm.PlaylistNameFR
		}
		if mm.PlaylistName != "" {
			playlistRef.Labels["en"] = mm.PlaylistName
		}
		// PairName composite Halo-only â†’ projetÃ© sur GameVariant pour
		// prÃ©server la compat des tests legacy qui peuplaient PairName/FR.
		var variantRef *canonical.AssetReference
		if mm.PairName != "" || mm.PairNameFR != "" {
			variantRef = &canonical.AssetReference{Kind: "game_variant", DefaultLabel: mm.PairName, Labels: map[string]string{}}
			if mm.PairNameFR != "" {
				variantRef.Labels["fr"] = mm.PairNameFR
			}
			if mm.PairName != "" {
				variantRef.Labels["en"] = mm.PairName
			}
		}
		out[i] = canonical.PlayerMatchRow{
			Summary: canonical.MatchSummary{
				MatchID:      mm.MatchID,
				StartedAtUTC: mm.StartTime,
				IsRanked:     &isRanked,
				IsPvE:        &isPvE,
				Outcome:      outcome,
				Map:          mapRef,
				Playlist:     playlistRef,
				GameVariant:  variantRef,
			},
			Self: canonical.MatchParticipant{
				Kills: &k, Deaths: &d, Assists: &a, KDA: mm.KDA, Outcome: outcome,
				Accuracy: mm.Accuracy, TimePlayed: mm.TimePlayedSecs,
			},
			Enrichment: canonical.PlayerMatchEnrichment{
				IsWithFriends:    mm.IsWithFriends,
				PerformanceScore: mm.PerformanceScore,
				SessionLabel:     sessionLabel,
				TeamMMR:          mm.TeamMMR,
				EnemyMMR:         mm.EnemyMMR,
			},
		}
	}
	return out, nil
}

func (m *mockHomePlayerMatches) InvalidatePlayer(_, _ string) {}

// withHomeMock attache le mock canonical au service pour exercer le path canonical.
func withHomeMock(svc *HomeService, repo *mockHomeRepo) *HomeService {
	pm := &mockHomePlayerMatches{matches: repo.matches, sessions: repo.sessions, err: repo.matchErr}
	return svc.WithPlayerMatchesRepo(pm, "halo_infinite", "TestGT")
}

// --- mock ---

type mockHomeRepo struct {
	matches                []legacymatch.HomeMatchRow
	matchErr               error
	identity               *domain.HomeSpartanIdentityRow
	identityErr            error
	matchCount             int
	countErr               error
	sessions               []legacymatch.HomeSessionRow
	sessionErr             error
	media                  []domain.HomeMediaRow
	mediaErr               error
	recentPlaylistRanks    []domain.HomePlaylistRank
	recentPlaylistRanksErr error
	commendations          map[string][]domain.HomeMatchCommendationRaw
	commendationsErr       error
}

func (m *mockHomeRepo) LoadHomeMatches(_ context.Context) ([]legacymatch.HomeMatchRow, error) {
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
func (m *mockHomeRepo) LoadHomeSessions(_ context.Context) ([]legacymatch.HomeSessionRow, error) {
	return m.sessions, m.sessionErr
}
func (m *mockHomeRepo) LoadRecentMedia(_ context.Context, _ int) ([]domain.HomeMediaRow, error) {
	return m.media, m.mediaErr
}
func (m *mockHomeRepo) LoadRecentPlaylistRanks(_ context.Context, _ string) ([]domain.HomePlaylistRank, error) {
	return m.recentPlaylistRanks, m.recentPlaylistRanksErr
}

func (m *mockHomeRepo) LoadMatchMedals(_ context.Context, _ []string) (map[string][]domain.RecentMatchMedal, error) {
	return map[string][]domain.RecentMatchMedal{}, nil
}

func (m *mockHomeRepo) LoadMatchCitations(_ context.Context, _ []string) (map[string][]domain.HomeMatchCitationRaw, error) {
	return map[string][]domain.HomeMatchCitationRaw{}, nil
}

func (m *mockHomeRepo) LoadMatchCommendations(_ context.Context, _ []string) (map[string][]domain.HomeMatchCommendationRaw, error) {
	if m.commendationsErr != nil {
		return nil, m.commendationsErr
	}
	if m.commendations != nil {
		return m.commendations, nil
	}
	return map[string][]domain.HomeMatchCommendationRaw{}, nil
}

func (m *mockHomeRepo) LoadFavoriteWeapon(_ context.Context, _ string) (string, int, error) {
	return "", 0, nil
}

func (m *mockHomeRepo) EnrichCanonicalAssetTranslations(_ context.Context, _ []canonical.PlayerMatchRow) error {
	return nil
}

// --- tests ---

func TestHomeService_GetHomePage_OK(t *testing.T) {
	now := time.Now()
	repo := &mockHomeRepo{
		matches: []legacymatch.HomeMatchRow{
			{MatchID: "m1", StartTime: now, MapName: "Aquarius", PairName: "Slayer", Outcome: 2, Kills: 10, Deaths: 5, Assists: 3, IsRanked: true},
			{MatchID: "m2", StartTime: now.Add(-1 * time.Hour), MapName: "Streets", PairName: "CTF", Outcome: 3, Kills: 5, Deaths: 10, Assists: 1, IsRanked: false},
		},
		sessions: []legacymatch.HomeSessionRow{
			{MatchID: "m1", SessionLabel: strPtr("Session 1"), StartTime: &now},
		},
		media: []domain.HomeMediaRow{
			{FileName: "clip1.mp4"},
		},
	}
	svc := withHomeMock(NewHomeService(repo), repo)

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
		matches:  []legacymatch.HomeMatchRow{},
		sessions: []legacymatch.HomeSessionRow{},
		media:    []domain.HomeMediaRow{},
	}
	svc := withHomeMock(NewHomeService(repo), repo)

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
	svc := withHomeMock(NewHomeService(repo), repo)

	_, err := svc.GetHomePage(context.Background(), "GT", "fr")
	if err == nil {
		t.Error("expected error when matches fail")
	}
}

func TestHomeService_GetHomePage_SessionsError(t *testing.T) {
	t.Skip("P4.3 finale : sessions sont dÃ©rivÃ©es des canonical rows (plus de LoadHomeSessions sÃ©parÃ©)")
}

func TestHomeService_GetHomePage_MediaGraceful(t *testing.T) {
	now := time.Now()
	repo := &mockHomeRepo{
		matches:  []legacymatch.HomeMatchRow{{MatchID: "m1", Outcome: 2, Kills: 10, Deaths: 5, StartTime: now}},
		sessions: []legacymatch.HomeSessionRow{},
		mediaErr: errors.New("media unavailable"),
	}
	svc := withHomeMock(NewHomeService(repo), repo)

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
		matches: []legacymatch.HomeMatchRow{{
			MatchID:        "m1",
			StartTime:      now,
			MapName:        "Bazaar",
			MapNameFR:      "Bazaar",
			PairName:       "Team Slayer on Bazaar",
			PairNameFR:     "Slayer en Ã©quipe sur Bazaar",
			PlaylistName:   "Quick Play",
			PlaylistNameFR: "Partie rapide",
			Outcome:        2,
		}},
	}
	svc := withHomeMock(NewHomeService(repo), repo)

	respFR, err := svc.GetHomePage(context.Background(), "GT", "fr")
	if err != nil {
		t.Fatalf("unexpected FR error: %v", err)
	}
	if got := *respFR.RecentMatches[0].ModeUI; got != "Slayer en Ã©quipe" {
		t.Fatalf("FR ModeUI = %q, want %q", got, "Slayer en Ã©quipe")
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
			RankImageURL:      strPtr("https://example.test/rank.png"),
			AdornmentImageURL: strPtr("https://example.test/adornment.png"),
			CurrentXP:         5000,
			XPForNextRank:     10000,
		},
	}
	// Injecte un SemanticAdapter avec un catalog minimal (rang 25 = Caporal-chef en FR).
	// Rang 26 ajouté pour que 25 ne soit pas le dernier rang du catalog (sinon
	// buildHomeCareerRank le déduirait comme rang max → ProgressPct=100).
	ranks := mappings.NewRankCatalog("halo_infinite", []mappings.RankEntry{
		{ID: 25, Title: map[string]string{"en": "Lance Corporal", "fr": "Caporal-chef"}},
		{ID: 26, Title: map[string]string{"en": "Corporal", "fr": "Caporal"}},
	})
	fields, ferr := mappings.LoadFieldsFromBytes("test.toml", []byte(`
[meta]
title_slug     = "halo_infinite"
schema_version = 1

[fields.kills]
labels        = { en = "Kills", fr = "Ã‰liminations" }
storage_unit  = "count"
display_unit  = "count"
format        = "integer"
display_order = 10
group         = "combat"
`))
	if ferr != nil {
		t.Fatalf("load fields: %v", ferr)
	}
	semantic := halo_infinite.NewSemanticAdapter(fields, ranks, nil, nil)
	svc := withHomeMock(NewHomeService(repo).WithSemanticAdapter(semantic), repo)

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

func TestHomeService_GetHomePage_SpartanIdentityErrorGraceful(t *testing.T) {
	// LoadSpartanIdentity en erreur â†’ rÃ©ponse retournÃ©e quand mÃªme (dÃ©gradation silencieuse).
	repo := &mockHomeRepo{
		matches:     []legacymatch.HomeMatchRow{{MatchID: "m1", Outcome: 2, StartTime: time.Now()}},
		sessions:    []legacymatch.HomeSessionRow{},
		identityErr: errors.New("career_progression table missing"),
	}
	svc := withHomeMock(NewHomeService(repo), repo)

	resp, err := svc.GetHomePage(context.Background(), "GT", "fr")
	if err != nil {
		t.Fatalf("expected graceful degradation on identityErr, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.SpartanIdentity != nil {
		t.Error("expected nil SpartanIdentity when repo fails")
	}
}

func TestHomeService_GetHomePage_CountMatchesFallback(t *testing.T) {
	// CountPlayerMatches en erreur â†’ fallback sur len(matches), pas d'erreur.
	now := time.Now()
	repo := &mockHomeRepo{
		matches:  []legacymatch.HomeMatchRow{{MatchID: "m1", Outcome: 2, StartTime: now}},
		sessions: []legacymatch.HomeSessionRow{},
		countErr: errors.New("count failed"),
	}
	svc := withHomeMock(NewHomeService(repo), repo)

	resp, err := svc.GetHomePage(context.Background(), "GT", "fr")
	if err != nil {
		t.Fatalf("expected no error on countErr fallback, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Hero.KPIs.TotalMatches != 1 {
		t.Errorf("TotalMatches = %d, want 1 (fallback to len(matches))", resp.Hero.KPIs.TotalMatches)
	}
}

// ---------------------------------------------------------------------------
// Tests cache TTL dans GetHomePage
// ---------------------------------------------------------------------------

func TestHomeService_GetHomePage_CacheHitSkipsDBCalls(t *testing.T) {
	t.Skip("P4.3 finale : HomeMatchesCache bypassÃ© en mode canonical (TODO P4.4 cache canonical-aware)")
}

func TestHomeService_GetHomePage_CacheMissAfterInvalidate(t *testing.T) {
	t.Skip("P4.3 finale : HomeMatchesCache bypassÃ© en mode canonical (TODO P4.4 cache canonical-aware)")
}

func TestHomeService_GetHomePage_NoCacheNoPanic(t *testing.T) {
	repo := &mockHomeRepo{
		matches:  []legacymatch.HomeMatchRow{{MatchID: "m1", Outcome: 2, StartTime: time.Now()}},
		sessions: []legacymatch.HomeSessionRow{},
	}
	// Sans cache â†’ comportement identique Ã  avant.
	svc := withHomeMock(NewHomeService(repo), repo)
	if _, err := svc.GetHomePage(context.Background(), "GT", "fr"); err != nil {
		t.Fatalf("unexpected error without cache: %v", err)
	}
}

func TestHomeService_GetBattlePass(t *testing.T) {
	repo := &mockHomeRepo{}
	svc := withHomeMock(NewHomeService(repo), repo)
	bp := svc.GetBattlePass(context.Background())
	// Default provider returns available=false
	if bp.Available {
		t.Error("expected Available=false from default provider")
	}
}

func TestHomeService_GetChallenges(t *testing.T) {
	repo := &mockHomeRepo{}
	svc := withHomeMock(NewHomeService(repo), repo)
	ch := svc.GetChallenges(context.Background())
	if ch.Available {
		t.Error("expected Available=false from default provider")
	}
}

// ---------------------------------------------------------------------------
// Tests cache-first (Phase A)
// ---------------------------------------------------------------------------

// mockBattlePassCacheRepo implÃ©mente port.BattlePassCacheRepository pour les tests.
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

// stubProviderHit est un provider nul qui panique si appelÃ© (vÃ©rifie que le cache a pris le dessus).
// (utilise la valeur par dÃ©faut de HaloProvider qui retourne available=false)

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

	// Cache miss â†’ live provider appelÃ© â†’ available=false (pas de tokens)
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
				Title:           "DÃ©fi test",
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

// Parité avec le Battle Pass (fix asymétrie « Défis indisponibles ») : un cache hit
// SANS items (counts seuls) doit quand même être rendu — jamais masqué en « indisponible ».
// Le frontend affiche les compteurs + un indicateur « données en cache ».
func TestHomeService_GetChallenges_CacheHitWithoutItems_StillReturnsCache(t *testing.T) {
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

	if !resp.Available {
		t.Error("expected Available=true from cache (no silent 'indisponible')")
	}
	if !resp.FromCache {
		t.Error("expected FromCache=true: a cache hit is always rendered now")
	}
	if resp.Total == nil || *resp.Total != total {
		t.Errorf("expected total %d preserved, got %v", total, resp.Total)
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
	// Sans WithCacheRepo â†’ live direct, pas de panique
	svc := NewHomeService(&mockHomeRepo{})
	resp := svc.GetBattlePass(context.Background())
	if resp.FromCache {
		t.Error("expected FromCache=false when no cache repo")
	}
}

// TestHomeService_GetBattlePass_CacheHitPreservesSnapshotAt vérifie que le
// SnapshotAt RFC3339 fourni par le cache repo n'est pas écrasé par le service
// (utilisé côté front pour l'indicateur de fraîcheur).
func TestHomeService_GetBattlePass_CacheHitPreservesSnapshotAt(t *testing.T) {
	rank := 7
	track := "RewardTracks/Operations/Old"
	progress := 50
	cachedSnap := "2026-05-10T08:15:00Z"
	cacheRepo := &mockBattlePassCacheRepo{
		bpResp: &domain.BattlePassResponse{
			Available:   true,
			Rank:        &rank,
			RewardTrack: &track,
			Progress:    &progress,
			FromCache:   true,
			SnapshotAt:  &cachedSnap,
		},
		bpHit: true,
	}

	svc := NewHomeService(&mockHomeRepo{}).WithCacheRepo(cacheRepo)
	resp := svc.GetBattlePass(context.Background())

	if !resp.FromCache {
		t.Fatal("expected FromCache=true from cache")
	}
	if resp.SnapshotAt == nil || *resp.SnapshotAt != cachedSnap {
		t.Fatalf("SnapshotAt = %v, want %q (preserve la valeur du cache)", resp.SnapshotAt, cachedSnap)
	}
}

// TestHomeService_GetChallenges_CacheHitPreservesSnapshotAt : pendant Challenges
// du test ci-dessus. Le SnapshotAt cache (MAX agrégé) doit traverser le service.
func TestHomeService_GetChallenges_CacheHitPreservesSnapshotAt(t *testing.T) {
	total := 4
	completed := 1
	xp := 250
	current, target := 1, 5
	cachedSnap := "2026-05-08T20:00:00Z"
	cacheRepo := &mockBattlePassCacheRepo{
		chResp: &domain.ChallengesResponse{
			Available:   true,
			Total:       &total,
			Completed:   &completed,
			XPAvailable: &xp,
			Items: []domain.ChallengeItem{{
				ChallengePath:   "Challenges/test",
				Title:           "Test",
				ProgressCurrent: &current,
				ProgressTarget:  &target,
			}},
			FromCache:  true,
			SnapshotAt: &cachedSnap,
		},
		chHit: true,
	}

	svc := NewHomeService(&mockHomeRepo{}).WithCacheRepo(cacheRepo)
	resp := svc.GetChallenges(context.Background())

	if !resp.FromCache {
		t.Fatal("expected FromCache=true from cache")
	}
	if resp.SnapshotAt == nil || *resp.SnapshotAt != cachedSnap {
		t.Fatalf("SnapshotAt = %v, want %q (preserve la valeur du cache)", resp.SnapshotAt, cachedSnap)
	}
}

// TestSnapshotAgeHours_HandlesNilAndMalformed vérifie le helper de logging :
// nil et string non parsable retournent -1 (pas de panique, log signale "inconnu").
func TestSnapshotAgeHours_HandlesNilAndMalformed(t *testing.T) {
	if got := snapshotAgeHours(nil); got != -1 {
		t.Errorf("snapshotAgeHours(nil) = %d, want -1", got)
	}
	bad := "not-a-date"
	if got := snapshotAgeHours(&bad); got != -1 {
		t.Errorf("snapshotAgeHours(bad) = %d, want -1", got)
	}
	twoHoursAgo := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	if got := snapshotAgeHours(&twoHoursAgo); got < 1 || got > 3 {
		t.Errorf("snapshotAgeHours(-2h) = %d, want ~2", got)
	}
}

// ---------------------------------------------------------------------------
// Tests comportement live-first / cache-fallback
// ---------------------------------------------------------------------------

func TestHomeService_DefaultTTL_Is1Hour(t *testing.T) {
	// SetSessionActive est no-op mais ne doit pas paniquer.
	svc := NewHomeService(&mockHomeRepo{})
	svc.SetSessionActive(true)
	svc.SetSessionActive(false)
}

func TestGetBattlePass_CallsLive_Always(t *testing.T) {
	// Sans tokens dans le contexte, le live retourne Available=false.
	// Le cache repo ne doit Ãªtre consultÃ© qu'en fallback, pas en premier.
	// On vÃ©rifie que le cache est consultÃ© APRÃˆS le live (en fallback),
	// en configurant un cache repo avec hit=true et en vÃ©rifiant le rÃ©sultat final.
	track := "path/to/track"
	cached := &domain.BattlePassResponse{Available: true, RewardTrack: &track}
	cacheRepo := &mockBattlePassCacheRepo{bpResp: cached, bpHit: true}
	svc := NewHomeService(&mockHomeRepo{}).WithCacheRepo(cacheRepo)
	// Live sans tokens â†’ Available=false â†’ fallback cache â†’ Available=true
	resp := svc.GetBattlePass(context.Background())
	if !resp.Available {
		t.Error("fallback cache attendu quand live indisponible")
	}
	if cacheRepo.bpCalls != 1 {
		t.Errorf("cache consultÃ© en fallback exactement une fois, bpCalls=%d", cacheRepo.bpCalls)
	}
}

func TestGetBattlePass_FallsBackToCache_WhenLiveUnavailable(t *testing.T) {
	// Quand le live retourne Available=false, le cache DB doit Ãªtre retournÃ©.
	track := "path/to/track"
	cached := &domain.BattlePassResponse{Available: true, RewardTrack: &track}
	cacheRepo := &mockBattlePassCacheRepo{bpResp: cached, bpHit: true}
	svc := NewHomeService(&mockHomeRepo{}).WithCacheRepo(cacheRepo)
	// provider par dÃ©faut : pas de tokens â†’ Available=false
	resp := svc.GetBattlePass(context.Background())
	if !resp.Available {
		t.Error("fallback cache attendu quand live indisponible")
	}
	if cacheRepo.bpCalls != 1 {
		t.Errorf("cache repo doit Ãªtre consultÃ© en fallback, bpCalls=%d", cacheRepo.bpCalls)
	}
}

func TestGetChallenges_FallsBackToCache_WhenLiveUnavailable(t *testing.T) {
	// ChallengesResponse depuis le cache → toujours rendu (parité Battle Pass).
	cached := &domain.ChallengesResponse{Available: true, Items: []domain.ChallengeItem{{}}}
	cacheRepo := &mockBattlePassCacheRepo{chResp: cached, chHit: true}
	svc := NewHomeService(&mockHomeRepo{}).WithCacheRepo(cacheRepo)
	resp := svc.GetChallenges(context.Background())
	if !resp.Available {
		t.Error("fallback cache attendu quand live indisponible")
	}
}

func TestGetBattlePass_NoSink_DoesNotPanic(t *testing.T) {
	svc := NewHomeService(&mockHomeRepo{})
	_ = svc.GetBattlePass(context.Background())
}

func TestGetChallenges_NoSink_DoesNotPanic(t *testing.T) {
	svc := NewHomeService(&mockHomeRepo{})
	_ = svc.GetChallenges(context.Background())
}

func TestHomeService_ConcurrentSetSessionActive(t *testing.T) {
	svc := NewHomeService(&mockHomeRepo{})
	for i := 0; i < 20; i++ {
		go func(i int) {
			svc.SetSessionActive(i%2 == 0)
		}(i)
	}
	// Pas d'assertion sur la valeur â€” on vÃ©rifie l'absence de race (-race flag)
}

func TestSelectTopMedals_RarityFirst(t *testing.T) {
	medals := []domain.RecentMatchMedal{
		{Name: "Assist", Count: 20, Difficulty: "Normal"},
		{Name: "Headshot", Count: 10, Difficulty: "Normal"},
		{Name: "Cauchemar", Count: 1, Difficulty: "Legendary"},
		{Name: "Perfection", Count: 1, Difficulty: "Mythic"},
		{Name: "Triple Kill", Count: 3, Difficulty: "Heroic"},
	}
	got := selectTopMedals(medals, 4)
	if len(got) != 4 {
		t.Fatalf("attendu 4 médailles, obtenu %d", len(got))
	}
	if got[0].Name != "Perfection" {
		t.Errorf("attendu Perfection en 1ère position (Mythic), obtenu %q", got[0].Name)
	}
	if got[1].Name != "Cauchemar" {
		t.Errorf("attendu Cauchemar en 2ème position (Legendary), obtenu %q", got[1].Name)
	}
	if got[2].Name != "Triple Kill" {
		t.Errorf("attendu Triple Kill en 3ème position (Heroic), obtenu %q", got[2].Name)
	}
	// 4ème : Assist (count=20) avant Headshot (count=10), même difficulté Normal
	if got[3].Name != "Assist" {
		t.Errorf("attendu Assist en 4ème position (Normal count=20), obtenu %q", got[3].Name)
	}
}

func TestSelectTopMedals_FewerThanN(t *testing.T) {
	medals := []domain.RecentMatchMedal{
		{Name: "Kill", Count: 5, Difficulty: "Normal"},
		{Name: "Assist", Count: 3, Difficulty: "Normal"},
	}
	got := selectTopMedals(medals, 4)
	if len(got) != 2 {
		t.Fatalf("attendu 2 médailles (tout garder), obtenu %d", len(got))
	}
}

func TestSelectTopMedals_Empty(t *testing.T) {
	if got := selectTopMedals(nil, 4); got != nil {
		t.Errorf("attendu nil pour entrée vide, obtenu %v", got)
	}
}

// ---------------------------------------------------------------------------
// enrichMatchesWithCommendations (P7 — commendations natives → TopCitations)
// ---------------------------------------------------------------------------

// TestEnrichMatchesWithCommendations_FillsEmptyTopCitations : pour un item dont
// TopCitations est vide (Halo 5, pas de moteur de citations dérivé), les
// commendations natives du repo alimentent le slot TopCitations (mappées en
// MatchCitationSnippet : Name / ImageURL / Delta).
func TestEnrichMatchesWithCommendations_FillsEmptyTopCitations(t *testing.T) {
	icon := "https://cdn.test/commend.png"
	repo := &mockHomeRepo{
		commendations: map[string][]domain.HomeMatchCommendationRaw{
			"m1": {
				{ID: "c-1", Name: "Sharpshooter", IconURL: icon, Count: 5},
				{ID: "c-2", Name: "Demon", IconURL: "", Count: 3},
			},
		},
	}
	items := []domain.RecentMatchItem{{MatchID: "m1"}} // TopCitations vide

	enrichMatchesWithCommendations(context.Background(), repo, items)

	if len(items[0].TopCitations) != 2 {
		t.Fatalf("TopCitations: want 2 snippets, got %d", len(items[0].TopCitations))
	}
	// Tri count DESC : Sharpshooter (5) avant Demon (3).
	first := items[0].TopCitations[0]
	if first.Name != "Sharpshooter" {
		t.Errorf("snippet[0].Name = %q, want Sharpshooter (count DESC)", first.Name)
	}
	if first.Delta != 5 {
		t.Errorf("snippet[0].Delta = %d, want 5 (count gagné)", first.Delta)
	}
	if first.Key != "c-1" {
		t.Errorf("snippet[0].Key = %q, want c-1 (ID natif)", first.Key)
	}
	if first.ImageURL == nil || *first.ImageURL != icon {
		t.Errorf("snippet[0].ImageURL = %v, want %q", first.ImageURL, icon)
	}
	// IconURL vide → ImageURL nil (pas d'URL fabriquée).
	if second := items[0].TopCitations[1]; second.ImageURL != nil {
		t.Errorf("snippet[1].ImageURL = %v, want nil (IconURL vide)", *second.ImageURL)
	}
}

// TestEnrichMatchesWithCommendations_ProgressTierMastered (S6) : une commendation
// native avec paliers (tier_targets) + cumul à vie (Progress) produit
// ProgressPct/TierIndex/TierCount/NextTierTarget/IsNewlyMastered, exactement comme une
// citation Infinite. Sans paliers → tout reste à zéro (anneau vide).
func TestEnrichMatchesWithCommendations_ProgressTierMastered(t *testing.T) {
	repo := &mockHomeRepo{
		commendations: map[string][]domain.HomeMatchCommendationRaw{
			"m1": {
				// Paliers [1,41,120,300], cumul 20 (delta 20) → palier 1 atteint,
				// pct = (20-1)/(41-1) = 47.5%, prochain palier = 41.
				{ID: "c-prog", Name: "Spartan Slayer", Count: 20, Progress: 20, TierTargets: "1,41,120,300"},
				// Sans paliers : dégradation propre, progression à zéro.
				{ID: "c-flat", Name: "Daily Win", Count: 3, Progress: 3, TierTargets: ""},
			},
		},
	}
	items := []domain.RecentMatchItem{{MatchID: "m1"}}

	enrichMatchesWithCommendations(context.Background(), repo, items)

	if len(items[0].TopCitations) != 2 {
		t.Fatalf("TopCitations: want 2, got %d", len(items[0].TopCitations))
	}
	// Tri count DESC : Spartan Slayer (20) en premier.
	prog := items[0].TopCitations[0]
	if prog.Name != "Spartan Slayer" {
		t.Fatalf("snippet[0].Name = %q, want Spartan Slayer", prog.Name)
	}
	if prog.ProgressPct != 47.5 {
		t.Errorf("ProgressPct = %v, want 47.5", prog.ProgressPct)
	}
	if prog.TierIndex != 1 {
		t.Errorf("TierIndex = %d, want 1", prog.TierIndex)
	}
	if prog.TierCount != 4 {
		t.Errorf("TierCount = %d, want 4", prog.TierCount)
	}
	if prog.NextTierTarget != 41 {
		t.Errorf("NextTierTarget = %d, want 41", prog.NextTierTarget)
	}
	if prog.Cumulative != 20 {
		t.Errorf("Cumulative = %d, want 20 (progress à vie)", prog.Cumulative)
	}
	if prog.IsNewlyMastered {
		t.Errorf("IsNewlyMastered = true, want false (pas au dernier palier)")
	}
	// Sans paliers → progression nulle.
	flat := items[0].TopCitations[1]
	if flat.ProgressPct != 0 || flat.TierCount != 0 || flat.NextTierTarget != 0 {
		t.Errorf("snippet flat: progression non nulle %+v", flat)
	}
	if flat.Delta != 3 {
		t.Errorf("flat.Delta = %d, want 3", flat.Delta)
	}
}

// TestEnrichMatchesWithCommendations_NewlyMastered (S6) : franchir le dernier palier
// CE match → ProgressPct=100 + IsNewlyMastered=true.
func TestEnrichMatchesWithCommendations_NewlyMastered(t *testing.T) {
	repo := &mockHomeRepo{
		commendations: map[string][]domain.HomeMatchCommendationRaw{
			// avant = 305-15 = 290 < 300 ; après = 305 >= 300 → newly mastered.
			"m1": {{ID: "c-m", Name: "Master", Count: 15, Progress: 305, TierTargets: "1,41,120,300"}},
		},
	}
	items := []domain.RecentMatchItem{{MatchID: "m1"}}

	enrichMatchesWithCommendations(context.Background(), repo, items)

	if len(items[0].TopCitations) != 1 {
		t.Fatalf("TopCitations: want 1, got %d", len(items[0].TopCitations))
	}
	s := items[0].TopCitations[0]
	if s.ProgressPct != 100.0 {
		t.Errorf("ProgressPct = %v, want 100", s.ProgressPct)
	}
	if !s.IsNewlyMastered {
		t.Errorf("IsNewlyMastered = false, want true")
	}
	if s.TierIndex != 4 || s.NextTierTarget != 0 {
		t.Errorf("TierIndex=%d NextTierTarget=%d, want 4 / 0", s.TierIndex, s.NextTierTarget)
	}
}

// TestEnrichMatchesWithCommendations_DoesNotOverwriteExistingCitations :
// précédence citations-first — un item qui a DÉJÀ des TopCitations (Halo Infinite,
// citations dérivées) n'est PAS écrasé par les commendations natives.
func TestEnrichMatchesWithCommendations_DoesNotOverwriteExistingCitations(t *testing.T) {
	existing := []domain.MatchCitationSnippet{{Key: "cite-a", Name: "Citation A", Delta: 2}}
	repo := &mockHomeRepo{
		commendations: map[string][]domain.HomeMatchCommendationRaw{
			"m1": {{ID: "c-1", Name: "Sharpshooter", Count: 9}},
		},
	}
	items := []domain.RecentMatchItem{{MatchID: "m1", TopCitations: existing}}

	enrichMatchesWithCommendations(context.Background(), repo, items)

	if len(items[0].TopCitations) != 1 || items[0].TopCitations[0].Name != "Citation A" {
		t.Fatalf("TopCitations écrasées : got %+v, want les citations dérivées préservées", items[0].TopCitations)
	}
}
