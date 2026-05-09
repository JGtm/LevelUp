package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	halo_games "levelup/go-api/internal/games/halo_infinite"
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
