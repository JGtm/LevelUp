package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	halo_games "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/mappings"
)

// --- mock ---

type mockCareerRepo struct {
	rank     *domain.CareerRankData
	rankErr  error
	xpHist   []domain.XPHistoryPoint
	xpErr    error
	lusrHist []domain.LUSRCheckpointDTO
	lusrErr  error
	topRows  []domain.TopMatchRawRow
	topErr   error
	encRows  []domain.EncounterRawRow
	encErr   error
	// Sprint Carrière+ : extensions injectables pour les tests des 3 nouvelles
	// méthodes (highlight-matches / top-encounters / rivals).
	highlightRows           []domain.HighlightMatchIDRow
	highlightErr            error
	topEncountersRows       []domain.MatchEncounterRow
	topEncountersStats      []domain.EncounterStatsRaw
	topEncountersErr        error
	topEncountersExcludeArg []string // capture l'argument reçu (pour vérifier l'exclusion friends)
	rivalsNemeses           []domain.CareerRivalRawRow
	rivalsVictims           []domain.CareerRivalRawRow
	rivalsErr               error
}

func (m *mockCareerRepo) GetLatestRank(_ context.Context) (*domain.CareerRankData, error) {
	return m.rank, m.rankErr
}
func (m *mockCareerRepo) GetXPHistory(_ context.Context) ([]domain.XPHistoryPoint, error) {
	return m.xpHist, m.xpErr
}
func (m *mockCareerRepo) GetLUSRHistory(_ context.Context) ([]domain.LUSRCheckpointDTO, error) {
	return m.lusrHist, m.lusrErr
}
func (m *mockCareerRepo) GetTopMatches(_ context.Context) ([]domain.TopMatchRawRow, error) {
	return m.topRows, m.topErr
}
func (m *mockCareerRepo) GetEncounters(_ context.Context) ([]domain.EncounterRawRow, error) {
	return m.encRows, m.encErr
}
func (m *mockCareerRepo) GetHighlightMatchIDs(_ context.Context, _ domain.CareerHighlightFilters) ([]domain.HighlightMatchIDRow, error) {
	return m.highlightRows, m.highlightErr
}
func (m *mockCareerRepo) GetHighlightPool(_ context.Context) ([]domain.HighlightMatchPoolRow, error) {
	return nil, nil
}
func (m *mockCareerRepo) GetTopEncountersGlobal(_ context.Context, exclude []string) ([]domain.MatchEncounterRow, []domain.EncounterStatsRaw, error) {
	m.topEncountersExcludeArg = append([]string{}, exclude...)
	return m.topEncountersRows, m.topEncountersStats, m.topEncountersErr
}
func (m *mockCareerRepo) GetRivals(_ context.Context) ([]domain.CareerRivalRawRow, []domain.CareerRivalRawRow, error) {
	return m.rivalsNemeses, m.rivalsVictims, m.rivalsErr
}
func (m *mockCareerRepo) GetCSRSnapshots(_ context.Context, _ string) ([]domain.CareerPlaylistCSR, error) {
	return nil, nil
}
func (m *mockCareerRepo) AvailableCSRSeasons(_ context.Context) ([]domain.CSRSeasonOption, error) {
	return nil, nil
}
func (m *mockCareerRepo) LoadModeTranslationsFR(_ context.Context, _ []string) (map[string]string, error) {
	return nil, nil
}
func (m *mockCareerRepo) LoadPlaylistAssetTranslationsFR(_ context.Context, _ []string) (map[string]string, error) {
	return nil, nil
}

// --- tests ---

func TestCareerService_GetCareerPage_OK(t *testing.T) {
	repo := &mockCareerRepo{
		rank: &domain.CareerRankData{
			RankNumber: 50,
			CurrentXP:  1000,
			RecordedAt: time.Now(),
		},
		xpHist:   []domain.XPHistoryPoint{{Rank: 50, CurrentXP: 1000}},
		lusrHist: []domain.LUSRCheckpointDTO{{RatingValue: 1500.0}},
	}
	svc := NewCareerService(repo)

	resp, err := svc.GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Summary.RankNumber != 50 {
		t.Errorf("RankNumber = %d, want 50", resp.Summary.RankNumber)
	}
}

func TestCareerService_GetCareerPage_NilRank(t *testing.T) {
	repo := &mockCareerRepo{
		rank:     nil,
		xpHist:   []domain.XPHistoryPoint{},
		lusrHist: []domain.LUSRCheckpointDTO{},
	}
	svc := NewCareerService(repo)

	resp, err := svc.GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Summary.RankNumber != 0 {
		t.Errorf("expected zero rank for nil rank data")
	}
}

func TestCareerService_GetCareerPage_RepoError(t *testing.T) {
	tests := []struct {
		name string
		repo *mockCareerRepo
	}{
		{"rank error", &mockCareerRepo{rankErr: errors.New("db fail")}},
		{"xp error", &mockCareerRepo{rank: &domain.CareerRankData{}, xpErr: errors.New("db fail")}},
		{"lusr error", &mockCareerRepo{rank: &domain.CareerRankData{}, xpHist: nil, lusrErr: errors.New("db fail")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewCareerService(tt.repo)
			_, err := svc.GetCareerPage(context.Background())
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// h5CareerStubAdapter : TitleDataAdapter h5-like qui ne sert QUE LoadCareerSnapshot
// (le reste dégrade via fakeDetailAdapter embarqué, défini dans
// match_view_canonical_test.go). Permet de tester la propagation career h5 → service
// sans dépendre du package halo_5 (évite un import croisé).
type h5CareerStubAdapter struct {
	fakeDetailAdapter
	snap *canonical.CareerSnapshot
}

func (s *h5CareerStubAdapter) LoadCareerSnapshot(_ context.Context, _ string, _ canonical.CareerOptions) (*canonical.CareerSnapshot, error) {
	return s.snap, nil
}

// TestCareerService_GetCareerPage_H5TotalRanks152 (AXE C4) : un snapshot career h5
// (RankMax=152 via l'adapter, label « SR N ») produit HeroProgress.TotalRanks=152
// — PAS le fallback HINF 272 — et le label SR n'est PAS écrasé par le catalogue
// HINF injecté au wiring (titre ≠ catalogue → catalogue neutralisé title-side).
func TestCareerService_GetCareerPage_H5TotalRanks152(t *testing.T) {
	t.Parallel()

	rankMax152 := 152
	xpMax := 50_000_000
	snap := &canonical.CareerSnapshot{
		Player:      canonical.PlayerIdentity{Gamertag: "JGtm"},
		RankNumber:  111,
		CurrentRank: &canonical.AssetReference{Kind: "spartan_rank", ID: "SR 111", DefaultLabel: "SR 111"},
		RankMax:     &rankMax152,
		XPMax:       &xpMax,
		// Libellé du rang max porté par l'adapter h5 (« SR 152 ») — le service le
		// recopie dans HeroProgress.MaxRankName* SANS passer par le catalogue HINF.
		MaxRank: &canonical.AssetReference{
			Kind: "spartan_rank", ID: "SR 152", DefaultLabel: "SR 152",
			Labels: map[string]string{"fr": "SR 152", "en": "SR152"},
		},
	}
	adapter := &h5CareerStubAdapter{snap: snap}

	// Catalogue HINF (DefaultSlug) injecté comme au wiring réel : il NE doit PAS
	// s'appliquer à un service titre halo_5. On y met un rank_id 111 avec un libellé
	// HINF distinct pour prouver l'absence d'écrasement.
	hinfCatalog := mappings.NewRankCatalog("halo_infinite", []mappings.RankEntry{
		{ID: 111, Title: map[string]string{"fr": "Cavalier HINF", "en": "HINF Rider"}},
	})

	svc := NewCareerService(&mockCareerRepo{}).
		WithTitleSlug("halo_5").
		WithDataAdapter(adapter).
		WithRankCatalog(hinfCatalog)

	resp, err := svc.GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("GetCareerPage: %v", err)
	}
	if resp.HeroProgress.TotalRanks != 152 {
		t.Errorf("TotalRanks = %d, want 152 (pas le fallback HINF 272)", resp.HeroProgress.TotalRanks)
	}
	if resp.Summary.RankNumber != 111 {
		t.Errorf("RankNumber = %d, want 111", resp.Summary.RankNumber)
	}
	// Le label SR doit survivre : le catalogue HINF (titre ≠) ne l'écrase pas.
	if resp.Summary.RankLabel != "SR 111" {
		t.Errorf("RankLabel = %q, want \"SR 111\" (catalogue HINF ne doit PAS écraser le label SR h5)", resp.Summary.RankLabel)
	}
	// Le libellé du rang max vient de la source h5 (« SR 152 »), pas du catalogue HINF.
	if resp.HeroProgress.MaxRankNameFR != "SR 152" {
		t.Errorf("MaxRankNameFR = %q, want \"SR 152\" (source h5, pas le catalogue HINF)", resp.HeroProgress.MaxRankNameFR)
	}
	if resp.HeroProgress.MaxRankNameEN != "SR152" {
		t.Errorf("MaxRankNameEN = %q, want \"SR152\"", resp.HeroProgress.MaxRankNameEN)
	}
}

// TestCareerService_GetCareerPage_MaxRankNameFromCatalog : pour un titre dont le
// catalogue s'applique (Halo Infinite), le libellé du rang MAX du gauge « path to
// max rank » provient de l'entrée SOMMET du catalogue (« Héros » / « Hero »), jamais
// d'un littéral. La source ne fournit pas MaxRankName* → résolution via catalogue.
func TestCareerService_GetCareerPage_MaxRankNameFromCatalog(t *testing.T) {
	t.Parallel()

	rankData := &domain.CareerRankData{RankNumber: 100, RecordedAt: time.Now()}
	catalog := mappings.NewRankCatalog("halo_infinite", []mappings.RankEntry{
		{ID: 100, Title: map[string]string{"fr": "Colonel", "en": "Colonel"}},
		{ID: 272, Title: map[string]string{"fr": "Héros", "en": "Hero"}},
	})
	svc := NewCareerService(&mockCareerRepo{rank: rankData}).
		WithTitleSlug("halo_infinite").
		WithRankCatalog(catalog)

	resp, err := svc.GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("GetCareerPage: %v", err)
	}
	if resp.HeroProgress.MaxRankNameFR != "Héros" {
		t.Errorf("MaxRankNameFR = %q, want \"Héros\" (entrée sommet du catalogue)", resp.HeroProgress.MaxRankNameFR)
	}
	if resp.HeroProgress.MaxRankNameEN != "Hero" {
		t.Errorf("MaxRankNameEN = %q, want \"Hero\"", resp.HeroProgress.MaxRankNameEN)
	}
}

// TestCareerService_GetCareerPage_MaxRankName_DegradesEmpty : sans catalogue
// applicable NI libellé de source, HeroProgress.MaxRankName* reste vide (dégradation
// propre — le front affiche alors un libellé générique). Aucune panique, aucun
// vocabulaire d'un autre titre.
func TestCareerService_GetCareerPage_MaxRankName_DegradesEmpty(t *testing.T) {
	t.Parallel()

	rankData := &domain.CareerRankData{RankNumber: 50, RecordedAt: time.Now()}
	svc := NewCareerService(&mockCareerRepo{rank: rankData}) // ni titleSlug ni catalogue

	resp, err := svc.GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("GetCareerPage: %v", err)
	}
	if resp.HeroProgress.MaxRankNameFR != "" || resp.HeroProgress.MaxRankNameEN != "" {
		t.Errorf("MaxRankName* = (%q, %q), want vides (dégradation)", resp.HeroProgress.MaxRankNameFR, resp.HeroProgress.MaxRankNameEN)
	}
}

// TestCareerService_GetCareerPage_HINFCatalogAppliesSameTitle : garde-fou de
// non-régression — quand le catalogue correspond au titre du service (halo_infinite),
// il s'applique normalement (RankLabel enrichi depuis le catalogue).
func TestCareerService_GetCareerPage_HINFCatalogAppliesSameTitle(t *testing.T) {
	t.Parallel()

	rankData := &domain.CareerRankData{RankNumber: 111, RecordedAt: time.Now()}
	hinfCatalog := mappings.NewRankCatalog("halo_infinite", []mappings.RankEntry{
		{ID: 111, Title: map[string]string{"fr": "Cavalier HINF", "en": "HINF Rider"}},
	})
	svc := NewCareerService(&mockCareerRepo{rank: rankData}).
		WithTitleSlug("halo_infinite").
		WithRankCatalog(hinfCatalog)

	resp, err := svc.GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("GetCareerPage: %v", err)
	}
	if resp.Summary.RankLabel != "Cavalier HINF" {
		t.Errorf("RankLabel = %q, want \"Cavalier HINF\" (catalogue même titre doit s'appliquer)", resp.Summary.RankLabel)
	}
}

func TestCareerService_GetTopMatches_OK(t *testing.T) {
	now := time.Now()
	rows := make([]domain.TopMatchRawRow, 20)
	for i := range rows {
		outcome := 2 // WIN pour les 10 premiers
		if i >= 10 {
			outcome = 3 // LOSS pour les 10 derniers
		}
		rows[i] = domain.TopMatchRawRow{
			MatchID:          "m" + string(rune('A'+i)),
			PerformanceScore: float64(100 - i),
			Kills:            10 + i,
			Deaths:           5,
			Outcome:          outcome,
			StartTime:        &now,
		}
	}
	repo := &mockCareerRepo{topRows: rows}
	svc := NewCareerService(repo)

	resp, err := svc.GetTopMatches(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.BestMatches) == 0 {
		t.Error("expected BestMatches to be non-empty")
	}
}

func TestCareerService_GetTopMatches_Empty(t *testing.T) {
	repo := &mockCareerRepo{topRows: []domain.TopMatchRawRow{}}
	svc := NewCareerService(repo)

	resp, err := svc.GetTopMatches(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.BestMatches) != 0 || len(resp.WorstMatches) != 0 {
		t.Error("expected empty best/worst for empty input")
	}
}

func TestCareerService_GetTopMatches_Error(t *testing.T) {
	repo := &mockCareerRepo{topErr: errors.New("fail")}
	svc := NewCareerService(repo)

	_, err := svc.GetTopMatches(context.Background())
	if err == nil {
		t.Error("expected error")
	}
}

// TestCareerService_GetTopMatches_DataAdapterParity (HIGH-C Path C) prouve que la
// bascule via TitleDataAdapter.LoadTopMatches produit STRICTEMENT le même
// CareerTopMatchesResponse que le repo legacy. Fixture avec TOUS les codes outcome
// (WIN=2/LOSS=3/TIE=1/DNF=4/unknown=0 — verrouille le code BRUT vs string canonique
// lossy + le split WIN/LOSS aval) + une map nil ET une map vide (verrouille la
// dérivation pointeur de convertTopMatches).
func TestCareerService_GetTopMatches_DataAdapterParity(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	mapA, mapEmpty := "Aquarius", ""
	kda := 2.5
	mmr := 1500.0
	rows := []domain.TopMatchRawRow{
		{MatchID: "m1", PerformanceScore: 95, Outcome: 2, Kills: 20, Deaths: 5, StartTime: &now, MapName: &mapA, KDA: &kda, TeamMMR: &mmr, DominanceFlag: 3},
		{MatchID: "m2", PerformanceScore: 90, Outcome: 3, Kills: 8, Deaths: 12, StartTime: &now, MapName: &mapEmpty}, // map vide → pointeur dérivé nil aval
		{MatchID: "m3", PerformanceScore: 70, Outcome: 1, Kills: 10, Deaths: 10, MapName: nil},                       // TIE + map nil
		{MatchID: "m4", PerformanceScore: 50, Outcome: 4, Kills: 1, Deaths: 1},                                       // DNF
		{MatchID: "m5", PerformanceScore: 40, Outcome: 0, Kills: 5, Deaths: 5},                                       // unknown
	}

	svcLegacy := NewCareerService(&mockCareerRepo{topRows: rows})
	respLegacy, err := svcLegacy.GetTopMatches(context.Background())
	if err != nil {
		t.Fatalf("legacy: %v", err)
	}

	repoAdapter := &mockCareerRepo{topRows: rows}
	dataAdapter := halo_games.NewDataAdapter(repoAdapter, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	svcAdapter := NewCareerService(repoAdapter).WithDataAdapter(dataAdapter)
	respAdapter, err := svcAdapter.GetTopMatches(context.Background())
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}

	jsonLegacy, _ := json.Marshal(respLegacy)
	jsonAdapter, _ := json.Marshal(respAdapter)
	if string(jsonLegacy) != string(jsonAdapter) {
		t.Errorf("top matches parity cassée :\nlegacy=  %s\nadapter= %s", jsonLegacy, jsonAdapter)
	}
}

// TestCareerService_GetTopMatches_AdapterFallbackOnUnsupported : adapter sans
// CareerSource → ErrCapabilityNotSupported → fallback silencieux sur repo.
func TestCareerService_GetTopMatches_AdapterFallbackOnUnsupported(t *testing.T) {
	t.Parallel()
	rows := []domain.TopMatchRawRow{
		{MatchID: "m1", PerformanceScore: 95, Outcome: 2, Kills: 20, Deaths: 5},
	}
	repo := &mockCareerRepo{topRows: rows}
	dataAdapter := halo_games.NewDataAdapter(nil, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	svc := NewCareerService(repo).WithDataAdapter(dataAdapter)

	resp, err := svc.GetTopMatches(context.Background())
	if err != nil {
		t.Fatalf("fallback devrait être silencieux, got %v", err)
	}
	if len(resp.BestMatches) != 1 {
		t.Errorf("BestMatches via fallback = %d, want 1", len(resp.BestMatches))
	}
}

func TestCareerService_GetEncounters_OK(t *testing.T) {
	repo := &mockCareerRepo{
		encRows: []domain.EncounterRawRow{
			{Gamertag: "Ally", XUID: "x1", MatchCount: 10, AsTeammate: 8, AsEnemy: 2},
			{Gamertag: "Foe", XUID: "x2", MatchCount: 5, AsTeammate: 1, AsEnemy: 4},
		},
	}
	svc := NewCareerService(repo)

	resp, err := svc.GetEncounters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Teammates) != 1 || resp.Teammates[0].Gamertag != "Ally" {
		t.Errorf("expected Ally as teammate, got %+v", resp.Teammates)
	}
	if len(resp.Enemies) != 1 || resp.Enemies[0].Gamertag != "Foe" {
		t.Errorf("expected Foe as enemy, got %+v", resp.Enemies)
	}
	if resp.Total != 2 {
		t.Errorf("Total = %d, want 2", resp.Total)
	}
}

func TestCareerService_GetEncounters_Error(t *testing.T) {
	repo := &mockCareerRepo{encErr: errors.New("fail")}
	svc := NewCareerService(repo)

	_, err := svc.GetEncounters(context.Background())
	if err == nil {
		t.Error("expected error")
	}
}

// TestCareerService_GetEncounters_DataAdapterParity prouve que la bascule
// vers le TitleDataAdapter (Phase C+ multi-titres) produit STRICTEMENT le
// même payload JSON que la version repo legacy, sur les mêmes données.
// C'est la golden parity backend pour /api/v1/players/{slug}/pages/career/encounters.
func TestCareerService_GetEncounters_DataAdapterParity(t *testing.T) {
	t.Parallel()

	avg1, avg2 := 1.42, 0.87
	rows := []domain.EncounterRawRow{
		{Gamertag: "Ally", XUID: "x1", MatchCount: 10, AsTeammate: 8, AsEnemy: 2, AvgKDA: &avg1},
		{Gamertag: "Foe", XUID: "x2", MatchCount: 5, AsTeammate: 1, AsEnemy: 4, AvgKDA: &avg2},
		{Gamertag: "Even", XUID: "x3", MatchCount: 6, AsTeammate: 3, AsEnemy: 3, AvgKDA: nil},
	}

	// Path 1 : repo direct (legacy).
	repoLegacy := &mockCareerRepo{encRows: rows}
	svcLegacy := NewCareerService(repoLegacy)
	respLegacy, err := svcLegacy.GetEncounters(context.Background())
	if err != nil {
		t.Fatalf("legacy err: %v", err)
	}

	// Path 2 : DataAdapter HI (Phase C+).
	repoAdapter := &mockCareerRepo{encRows: rows}
	dataAdapter := halo_games.NewDataAdapter(repoAdapter, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	svcAdapter := NewCareerService(repoAdapter).WithDataAdapter(dataAdapter)
	respAdapter, err := svcAdapter.GetEncounters(context.Background())
	if err != nil {
		t.Fatalf("adapter err: %v", err)
	}

	// Parité : les deux payloads doivent sérialiser à des JSON identiques.
	jsonLegacy, err := json.Marshal(respLegacy)
	if err != nil {
		t.Fatalf("marshal legacy: %v", err)
	}
	jsonAdapter, err := json.Marshal(respAdapter)
	if err != nil {
		t.Fatalf("marshal adapter: %v", err)
	}
	if string(jsonLegacy) != string(jsonAdapter) {
		t.Errorf("golden parity cassée :\nlegacy=  %s\nadapter= %s", jsonLegacy, jsonAdapter)
	}
}

// TestCareerService_GetCareerPage_DataAdapterParity : la bascule GetLatestRank
// par DataAdapter doit produire exactement le même CareerPageResponse que la
// version repo legacy, sur les mêmes données.
func TestCareerService_GetCareerPage_DataAdapterParity(t *testing.T) {
	t.Parallel()

	rankLabel := "Diamond 3"
	rankName := "Diamant 3"
	rankTier := "DIAMOND"
	xpForNext := 1234
	xpTotal := 5_000_000
	rankData := &domain.CareerRankData{
		RankNumber:    25,
		CurrentXP:     500,
		RecordedAt:    time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
		RankLabel:     &rankLabel,
		RankName:      &rankName,
		RankTier:      &rankTier,
		XPForNextRank: &xpForNext,
		XPTotal:       &xpTotal,
		IsMaxRank:     false,
	}

	// Path 1 : repo direct.
	repoLegacy := &mockCareerRepo{rank: rankData}
	svcLegacy := NewCareerService(repoLegacy)
	respLegacy, err := svcLegacy.GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("legacy: %v", err)
	}

	// Path 2 : DataAdapter.
	repoAdapter := &mockCareerRepo{rank: rankData}
	dataAdapter := halo_games.NewDataAdapter(repoAdapter, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	svcAdapter := NewCareerService(repoAdapter).WithDataAdapter(dataAdapter)
	respAdapter, err := svcAdapter.GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}

	// Parité Summary (cœur de la page).
	jsonLegacy, _ := json.Marshal(respLegacy.Summary)
	jsonAdapter, _ := json.Marshal(respAdapter.Summary)
	if string(jsonLegacy) != string(jsonAdapter) {
		t.Errorf("Summary parity cassée :\nlegacy=  %s\nadapter= %s", jsonLegacy, jsonAdapter)
	}
}

// TestCareerService_GetXPHistory_DataAdapterParity (HIGH-C) prouve que la bascule
// de l'historique XP vers le TitleDataAdapter (LoadCareerSnapshot + IncludeHistory)
// produit STRICTEMENT le même CareerPageResponse que la version repo legacy. Le
// fixture force XPTotal != CurrentXP pour attraper toute collision des deux entiers
// (le canonique CareerHistoryEntry porte RankNumber/CurrentXP/XPTotal séparés).
func TestCareerService_GetXPHistory_DataAdapterParity(t *testing.T) {
	t.Parallel()

	rankTier := "GOLD"
	xpTotal := 5_000_000
	rankData := &domain.CareerRankData{
		RankNumber: 50, CurrentXP: 1000, RankTier: &rankTier, XPTotal: &xpTotal,
		RecordedAt: time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	}
	xpHist := []domain.XPHistoryPoint{
		{RecordedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), Rank: 48, CurrentXP: 200, XPTotal: 4_000_000},
		{RecordedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), Rank: 49, CurrentXP: 500, XPTotal: 4_500_000},
		{RecordedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC), Rank: 50, CurrentXP: 1000, XPTotal: 5_000_000},
	}

	// Path 1 : repo direct.
	svcLegacy := NewCareerService(&mockCareerRepo{rank: rankData, xpHist: xpHist})
	respLegacy, err := svcLegacy.GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("legacy: %v", err)
	}

	// Path 2 : DataAdapter (LoadCareerSnapshot IncludeHistory).
	repoAdapter := &mockCareerRepo{rank: rankData, xpHist: xpHist}
	dataAdapter := halo_games.NewDataAdapter(repoAdapter, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	svcAdapter := NewCareerService(repoAdapter).WithDataAdapter(dataAdapter)
	respAdapter, err := svcAdapter.GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}

	// Parité du CareerPageResponse COMPLET : XPHistory ET Projections (XPTotal
	// alimente les deux) doivent être byte-identiques.
	jsonLegacy, _ := json.Marshal(respLegacy)
	jsonAdapter, _ := json.Marshal(respAdapter)
	if string(jsonLegacy) != string(jsonAdapter) {
		t.Errorf("XP history parity cassée :\nlegacy=  %s\nadapter= %s", jsonLegacy, jsonAdapter)
	}
}

// TestCareerService_GetXPHistory_EmptySerializesAsArray garantit que l'historique
// XP vide sérialise en `[]` (pas `null`) via les deux chemins — l'adapter retourne
// nil (comme le repo), que GetCareerPage ré-initialise en [].
func TestCareerService_GetXPHistory_EmptySerializesAsArray(t *testing.T) {
	t.Parallel()
	repoAdapter := &mockCareerRepo{} // rank nil, xpHist nil
	dataAdapter := halo_games.NewDataAdapter(repoAdapter, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	svc := NewCareerService(repoAdapter).WithDataAdapter(dataAdapter)
	resp, err := svc.GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	b, _ := json.Marshal(resp)
	if strings.Contains(string(b), `"xp_history":null`) {
		t.Errorf("xp_history sérialisé en null (attendu []) : %s", b)
	}
}

// TestCareerService_GetXPHistory_AdapterFallbackOnUnsupported : adapter sans
// CareerSource → ErrCapabilityNotSupported → fallback silencieux sur repo.GetXPHistory.
func TestCareerService_GetXPHistory_AdapterFallbackOnUnsupported(t *testing.T) {
	t.Parallel()
	xpHist := []domain.XPHistoryPoint{
		{RecordedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), Rank: 48, CurrentXP: 200, XPTotal: 4_000_000},
		{RecordedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), Rank: 49, CurrentXP: 500, XPTotal: 4_500_000},
	}
	repo := &mockCareerRepo{xpHist: xpHist}
	dataAdapter := halo_games.NewDataAdapter(nil, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	svc := NewCareerService(repo).WithDataAdapter(dataAdapter)

	resp, err := svc.GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("fallback devrait être silencieux, got %v", err)
	}
	if len(resp.XPHistory) != 2 {
		t.Errorf("XPHistory via fallback repo = %d, want 2", len(resp.XPHistory))
	}
}

// TestCareerService_GetLUSRHistory_DataAdapterParity (HIGH-C Path B) prouve que la
// bascule de l'historique LUSR vers TitleDataAdapter.LoadLUSRHistory produit
// STRICTEMENT le même CareerPageResponse que la version repo legacy. Fixture
// hétérogène (champs nullables set ET nil, delta omitempty).
func TestCareerService_GetLUSRHistory_DataAdapterParity(t *testing.T) {
	t.Parallel()

	tier := "Diamant 2"
	grp := "ranked"
	t1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	delta := 35.5
	badge := "https://img/badge.png"
	lusr := []domain.LUSRCheckpointDTO{
		{MatchID: "m1", RatingType: "lusr", RatingValue: 1850.0, TierLabel: &tier, PlaylistGroup: &grp, PlaylistName: "Arène", PlaylistID: "p1", RecordedAt: &t1, RatingDelta: nil, BadgeImageURL: &badge},
		{MatchID: "m2", RatingType: "lusr", RatingValue: 1885.5, TierLabel: &tier, PlaylistGroup: &grp, PlaylistName: "Arène", PlaylistID: "p1", RecordedAt: &t2, RatingDelta: &delta, BadgeImageURL: nil},
	}
	// Rang non-nil pour isoler la parité LUSR (un GetLatestRank (nil,nil) du mock
	// — irréaliste en prod où il retourne ErrNoRows/une row — diverge sur le bloc
	// summary, hors scope de ce test).
	rankTier := "GOLD"
	rank := &domain.CareerRankData{RankNumber: 50, CurrentXP: 1000, RankTier: &rankTier, RecordedAt: t1}

	svcLegacy := NewCareerService(&mockCareerRepo{rank: rank, lusrHist: lusr})
	respLegacy, err := svcLegacy.GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("legacy: %v", err)
	}

	repoAdapter := &mockCareerRepo{rank: rank, lusrHist: lusr}
	dataAdapter := halo_games.NewDataAdapter(repoAdapter, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	svcAdapter := NewCareerService(repoAdapter).WithDataAdapter(dataAdapter)
	respAdapter, err := svcAdapter.GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}

	jsonLegacy, _ := json.Marshal(respLegacy)
	jsonAdapter, _ := json.Marshal(respAdapter)
	if string(jsonLegacy) != string(jsonAdapter) {
		t.Errorf("LUSR history parity cassée :\nlegacy=  %s\nadapter= %s", jsonLegacy, jsonAdapter)
	}
}

// TestCareerService_GetLUSRHistory_AdapterFallbackOnUnsupported : adapter sans
// CareerSource → ErrCapabilityNotSupported → fallback silencieux sur repo.
func TestCareerService_GetLUSRHistory_AdapterFallbackOnUnsupported(t *testing.T) {
	t.Parallel()
	tier := "Or 1"
	lusr := []domain.LUSRCheckpointDTO{
		{MatchID: "m1", RatingType: "lusr", RatingValue: 1450, TierLabel: &tier, PlaylistName: "Arène", PlaylistID: "p1"},
	}
	repo := &mockCareerRepo{lusrHist: lusr}
	dataAdapter := halo_games.NewDataAdapter(nil, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	svc := NewCareerService(repo).WithDataAdapter(dataAdapter)

	resp, err := svc.GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("fallback devrait être silencieux, got %v", err)
	}
	if len(resp.LUSR.Checkpoints) != 1 {
		t.Errorf("LUSR checkpoints via fallback = %d, want 1", len(resp.LUSR.Checkpoints))
	}
}

// TestCareerService_GetEncounters_AdapterFallbackOnUnsupported prouve que si
// le DataAdapter retourne ErrCapabilityNotSupported, le service retombe sur
// le repo sans propager l'erreur (dégradation gracieuse).
func TestCareerService_GetEncounters_AdapterFallbackOnUnsupported(t *testing.T) {
	t.Parallel()

	rows := []domain.EncounterRawRow{
		{Gamertag: "Ally", XUID: "x1", MatchCount: 1, AsTeammate: 1, AsEnemy: 0},
	}
	repo := &mockCareerRepo{encRows: rows}

	// DataAdapter sans CareerSource → LoadEncounters retourne ErrCapabilityNotSupported.
	dataAdapter := halo_games.NewDataAdapter(nil, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	svc := NewCareerService(repo).WithDataAdapter(dataAdapter)

	resp, err := svc.GetEncounters(context.Background())
	if err != nil {
		t.Fatalf("fallback devrait être silencieux, got %v", err)
	}
	if resp.Total != 1 || len(resp.Teammates) != 1 {
		t.Errorf("payload via fallback repo incorrect : %+v", resp)
	}
}

// ---------------------------------------------------------------------------
// Tests Sprint Carrière+ : highlight-matches, top-encounters, rivals
// ---------------------------------------------------------------------------

func TestCareerService_GetHighlightMatchIDs_PassesThroughFromRepo(t *testing.T) {
	rows := []domain.HighlightMatchIDRow{
		{MatchID: "m1", Outcome: 2, Section: 1},
		{MatchID: "m2", Outcome: 3, Section: 2},
	}
	repo := &mockCareerRepo{highlightRows: rows}
	svc := NewCareerService(repo)

	got, err := svc.GetHighlightMatchIDs(context.Background(), domain.HighlightFilterInput{})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(got.Rows) != 2 || got.Rows[0].MatchID != "m1" || got.Rows[1].Section != 2 {
		t.Errorf("rows mal projetées : %+v", got.Rows)
	}
}

func TestCareerService_GetHighlightMatchIDs_RepoError(t *testing.T) {
	repo := &mockCareerRepo{highlightErr: errors.New("db down")}
	svc := NewCareerService(repo)
	if _, err := svc.GetHighlightMatchIDs(context.Background(), domain.HighlightFilterInput{}); err == nil {
		t.Error("expected error")
	}
}

func TestCareerService_GetRivals_RatioComputation(t *testing.T) {
	repo := &mockCareerRepo{
		rivalsNemeses: []domain.CareerRivalRawRow{
			{Gamertag: "Killer1", Frags: 5, Deaths: 10, MatchCount: 3}, // ratio = 0.5
			{Gamertag: "Killer2", Frags: 0, Deaths: 4, MatchCount: 2},  // ratio = 0
		},
		rivalsVictims: []domain.CareerRivalRawRow{
			{Gamertag: "Victim1", Frags: 8, Deaths: 0, MatchCount: 2},  // div par zéro → ratio = Frags
			{Gamertag: "Victim2", Frags: 12, Deaths: 4, MatchCount: 5}, // ratio = 3
		},
	}
	svc := NewCareerService(repo)

	resp, err := svc.GetRivals(context.Background())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(resp.Nemeses) != 2 || len(resp.Victims) != 2 {
		t.Fatalf("wrong counts : %+v", resp)
	}
	if resp.Nemeses[0].Ratio != 0.5 {
		t.Errorf("Killer1 ratio = %v, want 0.5", resp.Nemeses[0].Ratio)
	}
	if resp.Nemeses[1].Ratio != 0 {
		t.Errorf("Killer2 (frags=0) ratio = %v, want 0", resp.Nemeses[1].Ratio)
	}
	// Victim1 a 0 deaths → ratio = float64(Frags) (sentinelle "infini" approximé).
	if resp.Victims[0].Ratio != 8 {
		t.Errorf("Victim1 (deaths=0) ratio = %v, want 8 (frags fallback)", resp.Victims[0].Ratio)
	}
	if resp.Victims[1].Ratio != 3 {
		t.Errorf("Victim2 ratio = %v, want 3", resp.Victims[1].Ratio)
	}
	if resp.Victims[1].MatchCount != 5 {
		t.Errorf("MatchCount projection cassée : %+v", resp.Victims[1])
	}
}

func TestCareerService_GetRivals_RepoError(t *testing.T) {
	repo := &mockCareerRepo{rivalsErr: errors.New("kvp scan failed")}
	svc := NewCareerService(repo)
	if _, err := svc.GetRivals(context.Background()); err == nil {
		t.Error("expected error")
	}
}

func TestCareerService_GetTopEncounters_AppliesFriendExclusion(t *testing.T) {
	repo := &mockCareerRepo{
		topEncountersRows: []domain.MatchEncounterRow{
			{XUID: "x42", Gamertag: "Stranger", CountTogether: 5},
		},
		topEncountersStats: []domain.EncounterStatsRaw{
			{XUID: "x42", AllyCount: 0, EnemyCount: 5},
		},
	}
	svc := NewCareerService(repo).
		WithFriendGamertagsResolver(func(_ context.Context) []string { return []string{"BestFriend", "OtherFriend"} }).
		WithFriendXUIDResolver(func(_ context.Context, gt string) (string, error) {
			switch gt {
			case "BestFriend":
				return "xuid-friend-1", nil
			case "OtherFriend":
				return "xuid-friend-2", nil
			}
			return "", errors.New("not found")
		})

	_, err := svc.GetTopEncounters(context.Background())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(repo.topEncountersExcludeArg) != 2 {
		t.Fatalf("exclude xuid count = %d, want 2 (xuid-friend-1, xuid-friend-2)", len(repo.topEncountersExcludeArg))
	}
	want := map[string]bool{"xuid-friend-1": true, "xuid-friend-2": true}
	for _, x := range repo.topEncountersExcludeArg {
		if !want[x] {
			t.Errorf("XUID inattendu dans exclude : %q", x)
		}
	}
}

func TestCareerService_GetTopEncounters_NoFriendsResolverGracefulDegradation(t *testing.T) {
	repo := &mockCareerRepo{
		topEncountersRows:  []domain.MatchEncounterRow{{XUID: "x1", Gamertag: "Player", CountTogether: 3}},
		topEncountersStats: []domain.EncounterStatsRaw{{XUID: "x1", AllyCount: 1, EnemyCount: 2}},
	}
	// Pas de WithFriendGamertagsResolver → exclusion silencieusement désactivée.
	svc := NewCareerService(repo)

	resp, err := svc.GetTopEncounters(context.Background())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(resp.Items) != 1 {
		t.Errorf("Items = %d, want 1 (pas d'exclusion sans resolver)", len(resp.Items))
	}
	// L'argument exclude doit avoir été vide (resolveFriendXUIDs retourne nil).
	if len(repo.topEncountersExcludeArg) != 0 {
		t.Errorf("exclude arg = %v, want empty", repo.topEncountersExcludeArg)
	}
}

func TestCareerService_GetTopEncounters_AppliesNarrativeBadges(t *testing.T) {
	// Encounter avec 5 enemy_count + 10 deaths_suffered + 0 kills_dealt
	// → tough_enemy badge (KillsDealt=0 + DeathsSuffered>=3).
	// Plus ordinal badge automatique car CountTogether>0.
	repo := &mockCareerRepo{
		topEncountersRows: []domain.MatchEncounterRow{
			{XUID: "x1", Gamertag: "ToughOne", CountTogether: 5},
		},
		topEncountersStats: []domain.EncounterStatsRaw{
			{XUID: "x1", AllyCount: 0, EnemyCount: 5, KillsDealt: 0, DeathsSuffered: 10},
		},
	}
	svc := NewCareerService(repo)

	resp, err := svc.GetTopEncounters(context.Background())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("Items = %d, want 1", len(resp.Items))
	}
	got := resp.Items[0]
	if len(got.Badges) == 0 {
		t.Fatalf("expected at least 1 badge (ordinal + tough_enemy), got 0")
	}
	hasOrdinal := false
	hasTough := false
	for _, b := range got.Badges {
		if b.Kind == "ordinal" {
			hasOrdinal = true
		}
		if b.Kind == "tough_enemy" {
			hasTough = true
		}
	}
	if !hasOrdinal {
		t.Error("missing ordinal badge")
	}
	if !hasTough {
		t.Error("missing tough_enemy badge (KillsDealt=0 + DeathsSuffered>=3 doit déclencher)")
	}
}

// TestCareerService_GetTopEncounters_AppliesRelationSolidBadges : parité avec le hub
// Communauté > Relations après unification (helper partagé relations.ComputeBadges) —
// la page Carrière attribue désormais les badges « solid » (ici duo_gagnant), plus
// seulement les 4 badges narratifs historiques.
func TestCareerService_GetTopEncounters_AppliesRelationSolidBadges(t *testing.T) {
	repo := &mockCareerRepo{
		topEncountersRows: []domain.MatchEncounterRow{
			{XUID: "x_duo", Gamertag: "Duo", CountTogether: 12},
		},
		topEncountersStats: []domain.EncounterStatsRaw{
			{XUID: "x_duo", AllyCount: 12, WinsAsAlly: 8, LossesAsAlly: 4}, // WR allié 0.667 >= 0.60 sur 12 matchs
		},
	}
	resp, err := NewCareerService(repo).GetTopEncounters(context.Background())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("Items = %d, want 1", len(resp.Items))
	}
	found := false
	for _, b := range resp.Items[0].Badges {
		if b.Kind == "duo_gagnant" {
			found = true
		}
	}
	if !found {
		t.Errorf("duo_gagnant attendu (parité hub Relations), got %+v", resp.Items[0].Badges)
	}
}

// TestCareerService_GetTopEncounters_FirstSeenPowersRecrue : first_seen_at (ajouté à
// Q26CareerTopEncountersTpl) alimente le badge temporel recrue (relation récente
// < 30 j ET >= 4 matchs).
func TestCareerService_GetTopEncounters_FirstSeenPowersRecrue(t *testing.T) {
	repo := &mockCareerRepo{
		topEncountersRows: []domain.MatchEncounterRow{
			{XUID: "x_new", Gamertag: "Rookie", CountTogether: 5},
		},
		topEncountersStats: []domain.EncounterStatsRaw{
			{XUID: "x_new", AllyCount: 5, FirstSeen: time.Now().AddDate(0, 0, -10)}, // il y a 10 j (< 30)
		},
	}
	resp, err := NewCareerService(repo).GetTopEncounters(context.Background())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	found := false
	for _, b := range resp.Items[0].Badges {
		if b.Kind == "recrue" {
			found = true
		}
	}
	if !found {
		t.Errorf("recrue attendu (first_seen il y a 10 j, 5 matchs), got %+v", resp.Items[0].Badges)
	}
}

func TestCareerService_GetTopEncounters_RepoError(t *testing.T) {
	repo := &mockCareerRepo{topEncountersErr: errors.New("query timeout")}
	svc := NewCareerService(repo)
	if _, err := svc.GetTopEncounters(context.Background()); err == nil {
		t.Error("expected error")
	}
}
