package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
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
		xpHist:   []domain.XPHistoryPoint{{RankNumber: 50, CurrentXP: 1000}},
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
		rows[i] = domain.TopMatchRawRow{
			MatchID:          "m" + string(rune('A'+i)),
			PerformanceScore: float64(100 - i),
			Kills:            10 + i,
			Deaths:           5,
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
