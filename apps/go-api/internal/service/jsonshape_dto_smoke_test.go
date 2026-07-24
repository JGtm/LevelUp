package service

import (
	"context"
	"testing"

	teammatespkg "levelup/go-api/internal/service/teammates"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/testutil"
)

// TestDTOs_NoNilSlicesOnEmptyInput exerce les principaux services avec des
// inputs vides ou minimaux et vérifie qu'aucun champ slice (sans tag
// `omitempty`) n'est nil dans la réponse — sinon le JSON marshalé contient
// `null` et le frontend non-nullable crashe sur `.filter()` / `.map()`.
//
// Garde-rail anti-régression pour la classe de bugs "slice Go nil → JSON null".
// Cf. internal/testutil/jsonshape.go et le crash 2026-05-27 sur FilterOmnibar
// (filters_service.emptyResolved oubliait Playlists/Modes/Maps).
//
// Pour ajouter un service : un nouveau sous-test qui construit le service avec
// un mock minimal et appelle `testutil.RequireNoNilSlicesWithoutOmitempty` sur
// la réponse.
func TestDTOs_NoNilSlicesOnEmptyInput(t *testing.T) {
	t.Run("FiltersService.Resolve", func(t *testing.T) {
		repo := &mockFiltersRepo{rows: []domain.FilterMatchRow{}}
		svc := NewFiltersService(repo)
		resp, err := svc.Resolve(context.Background(), domain.FilterContextInput{})
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("MatchHistoryService.GetPage", func(t *testing.T) {
		repo := &mockMatchHistoryRepo{rows: nil}
		svc := NewMatchHistoryService(repo, "Player")
		resp, err := svc.GetPage(context.Background(), domain.MatchHistoryQueryRequest{
			Pagination: domain.PaginationRequest{Page: 1, PageSize: 20},
		})
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("HomeService.GetHomePage", func(t *testing.T) {
		repo := &mockHomeRepo{
			matches:  []legacymatch.HomeMatchRow{},
			sessions: []legacymatch.HomeSessionRow{},
			media:    []domain.HomeMediaRow{},
		}
		svc := withHomeMock(NewHomeService(repo), repo)
		resp, err := svc.GetHomePage(context.Background(), "TestGT", "fr")
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("MatchViewService.GetMatchView", func(t *testing.T) {
		repo := &mockMatchViewRepo{
			meta: &domain.MatchMetaRaw{MatchID: "m1"},
		}
		svc := NewMatchViewService(repo, "Player")
		resp, err := svc.GetMatchView(context.Background(), "m1")
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("SquadService.GetSquadPage", func(t *testing.T) {
		repo := &mockSquadRepo{topRows: []domain.TopTeammateRow{}}
		svc := NewSquadService(repo).WithPlayerMatchesRepo(newSynthMockFromRows(nil, nil), "halo_infinite", "Test")
		resp, err := svc.GetSquadPage(context.Background(), "x", "gt", "")
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("SquadService.GetSynthesisPage", func(t *testing.T) {
		repo := &mockSquadRepo{synthRows: []legacymatch.SynthesisMatchRow{}}
		svc := NewSquadService(repo).WithPlayerMatchesRepo(newSynthMockFromRows(nil, nil), "halo_infinite", "Test")
		resp, err := svc.GetSynthesisPage(context.Background(), "x")
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("CareerService.GetCareerPage", func(t *testing.T) {
		repo := &mockCareerRepo{}
		svc := NewCareerService(repo)
		resp, err := svc.GetCareerPage(context.Background())
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("SessionsService.GetSessions", func(t *testing.T) {
		repo := &mockSessionsRepo{rows: []domain.SessionMatchRow{}}
		svc := NewSessionsService(repo)
		resp, err := svc.GetSessions(context.Background(), domain.SessionComputeOptions{
			GapMinutes: 60,
			Mode:       domain.SessionModeGap,
		})
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("CitationsService.GetCitationsPage", func(t *testing.T) {
		repo := &mockCitationsRepo{
			totals:   []domain.CitationTotalRow{},
			mappings: []domain.CitationMappingRow{},
		}
		svc := NewCitationsService(repo)
		resp, err := svc.GetCitationsPage(context.Background())
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("CitationsService.GetCommendationsPage", func(t *testing.T) {
		repo := &mockCitationsRepo{medals: []domain.MedalEarnedRow{}, medalMaps: []domain.MedalCitationRow{}}
		svc := NewCitationsService(repo)
		resp, err := svc.GetCommendationsPage(context.Background(), "")
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("MedalsService.GetMedalsPage", func(t *testing.T) {
		repo := &mockMedalsRepo{catalog: []domain.MedalCatalogRow{}, earned: []domain.MedalEarnedRow{}}
		svc := NewMedalsService(repo)
		resp, err := svc.GetMedalsPage(context.Background(), "")
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("LeaderboardService.GetPage", func(t *testing.T) {
		repo := &mockLeaderboardRepo{csrWorld: []domain.LeaderboardEntry{}}
		svc := NewLeaderboardService(repo)
		resp, err := svc.GetPage(context.Background(), domain.LeaderboardRequest{TitleSlug: "halo_infinite"})
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("AchievementsService.GetAchievementsPage", func(t *testing.T) {
		svc := NewAchievementsService(
			&mockAchievementsRepo{rows: []domain.PlayerAchievementRow{}},
			&mockMetadataAchievementsRepo{defs: []domain.AchievementDefinitionRow{}},
		).WithTitleSlug("halo_infinite")
		resp, err := svc.GetAchievementsPage(context.Background())
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("TeammatesService.GetPage", func(t *testing.T) {
		repo := &mockSquadRepo{topRows: []domain.TopTeammateRow{}}
		svc := teammatespkg.NewTeammatesService(repo, func(_ context.Context) []string { return nil }).
			WithPlayerMatchesRepo(newSynthMockFromRows(nil, nil), "halo_infinite", "Test")
		resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{})
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("StatsService.GetPage", func(t *testing.T) {
		repo := &mockStatsRepoForStats{matches: []legacymatch.StatsMatchRow{}}
		svc := NewStatsService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(nil, nil), "Test")
		svc.titleSlug = "halo_infinite"
		resp, err := svc.GetPage(context.Background(), domain.StatsQueryRequest{Tab: "win_loss"})
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("TimeseriesService.GetPage", func(t *testing.T) {
		svc := NewTimeseriesService(&mockTimeseriesRepo{}).
			WithPlayerMatchesRepo(newStatsMockFromRows(nil, nil), "halo_infinite", "Test")
		resp, err := svc.GetPage(context.Background(), domain.TimeseriesQueryRequest{})
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("SessionPageService.GetPage", func(t *testing.T) {
		repo := &mockSessionPageStatsRepo{matches: []legacymatch.StatsMatchRow{}}
		svc := NewSessionPageService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(nil, nil), "halo_infinite", "Test")
		resp, err := svc.GetPage(context.Background(), domain.SessionPageRequest{})
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("CompareService.GetPage", func(t *testing.T) {
		repo := &mockCompareRepoAB{a: &domain.NormalizedPlayerStats{Gamertag: "A"}, b: &domain.NormalizedPlayerStats{Gamertag: "B"}}
		provider := &mockStatsProvider{}
		svc := NewCompareService(repo, provider, "xuid-a", "halo_infinite")
		resp, err := svc.GetPage(context.Background(), domain.CompareRequest{TargetGamertag: "B"})
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("SeasonPassService.GetSeasonPassPage", func(t *testing.T) {
		repo := &mockSeasonPassRepo{tracks: []domain.SeasonPassTrackSummary{}}
		homeSvc := &mockHomeServiceForSP{}
		svc := NewSeasonPassService(repo, homeSvc, "xuid-123", "halo_infinite")
		resp, err := svc.GetSeasonPassPage(context.Background())
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("MediaService.GetMediaPage", func(t *testing.T) {
		repo := &mockMediaRepo{files: []domain.MediaFileRow{}, count: 0}
		svc := NewMediaService(repo, "")
		resp, err := svc.GetMediaPage(context.Background(), domain.MediaPageRequest{Page: 1})
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("ExplorerService.GetCommonMatches", func(t *testing.T) {
		repo := &mockExplorerRepo{xuid: "target-xuid", matches: []domain.CommonMatchRaw{}}
		svc := NewExplorerService(repo, "my-xuid")
		resp, err := svc.GetCommonMatches(context.Background(), "Target", "", 1)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	// --- Career sous-méthodes (méthodes additionnelles de CareerService) ---

	t.Run("CareerService.GetTopMatches", func(t *testing.T) {
		svc := NewCareerService(&mockCareerRepo{topRows: []domain.TopMatchRawRow{}})
		resp, err := svc.GetTopMatches(context.Background())
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("CareerService.GetEncounters", func(t *testing.T) {
		svc := NewCareerService(&mockCareerRepo{encRows: []domain.EncounterRawRow{}})
		resp, err := svc.GetEncounters(context.Background())
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("CareerService.GetRivals", func(t *testing.T) {
		svc := NewCareerService(&mockCareerRepo{})
		resp, err := svc.GetRivals(context.Background())
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("CareerService.GetCareerCSRs", func(t *testing.T) {
		svc := NewCareerService(&mockCareerRepo{})
		resp, err := svc.GetCareerCSRs(context.Background(), "")
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	// --- BootstrapService (nécessite config minimal) ---

	t.Run("BootstrapService.BuildPlayersList", func(t *testing.T) {
		repo := &mockBootRepo{matchCount: 0}
		svc := NewBootstrapService(testBootstrapConfig(), repo)
		resp, err := svc.BuildPlayersList(context.Background(), nil)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})

	t.Run("AssetService.ListMaps", func(t *testing.T) {
		repo := &mockAssetMetaRepo{maps: nil} // path "aucun résultat"
		svc := NewAssetService(repo)
		items, err := svc.ListMaps(context.Background(), "halo_infinite", "")
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		// Retour direct []canonical.AssetMeta — on enveloppe pour bénéficier du
		// helper (qui parcourt récursivement les slices comme racine).
		testutil.RequireNoNilSlicesWithoutOmitempty(t, struct {
			Items []canonical.AssetMeta `json:"items"`
		}{Items: items})
	})

	t.Run("AssetService.ListWeapons", func(t *testing.T) {
		repo := &mockAssetMetaRepo{weapons: nil}
		svc := NewAssetService(repo)
		items, err := svc.ListWeapons(context.Background(), "halo_infinite", "")
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, struct {
			Items []canonical.AssetMeta `json:"items"`
		}{Items: items})
	})

	t.Run("CareerService.GetTopEncounters", func(t *testing.T) {
		repo := &mockCareerRepo{
			topEncountersRows:  []domain.MatchEncounterRow{},
			topEncountersStats: []domain.EncounterStatsRaw{},
		}
		svc := NewCareerService(repo)
		resp, err := svc.GetTopEncounters(context.Background())
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	})
}

// testBootstrapConfig retourne un config.AppConfig minimal réutilisable par les
// smoke tests qui en ont besoin (BootstrapService).
func testBootstrapConfig() *config.AppConfig {
	return &config.AppConfig{}
}
