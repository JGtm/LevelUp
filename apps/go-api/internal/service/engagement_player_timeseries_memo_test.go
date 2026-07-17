// Package service — engagement_player_timeseries_memo_test.go : garde-rail E1
// (revue 2026-07). GetTimeseries recompute le score par match ; coef lobby +
// bins de reponse ne dependent que de (xuid, mode_category). Ils doivent etre
// memoises par mode_category : une serie de 200 matchs ne doit relire coef+bins
// qu'au plus une fois par mode (≤ 4 lectures), pas une fois par match (~400).
package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// memoCountingEngagementRepo compte les lectures coef+bins et sert 200 matchs
// (moitie ranked / moitie unranked) via le fast path ListRecentPvPMatchIDs.
type memoCountingEngagementRepo struct {
	coefCalls int
	binsCalls int
	ids       []string
}

func (m *memoCountingEngagementRepo) LoadEngagementCoefficient(_ context.Context, _, _ string) (*domain.EngagementCoefficient, error) {
	m.coefCalls++
	return nil, nil
}

func (m *memoCountingEngagementRepo) LoadResponseBins(_ context.Context, _, _ string) (*domain.EngagementResponseBins, error) {
	m.binsCalls++
	return nil, nil
}

func (m *memoCountingEngagementRepo) LoadPlayerHistory(_ context.Context, _ port.EngagementHistoryFilter) ([]domain.HistoricalEngagementBrut, error) {
	return nil, nil
}

func (m *memoCountingEngagementRepo) LoadMatchEngagementContext(_ context.Context, matchID, _ string) (*port.MatchEngagementContext, error) {
	return &port.MatchEngagementContext{
		MatchID:      matchID,
		StartTimeMS:  0,
		EndTimeMS:    720_000,
		IsRanked:     strings.HasPrefix(matchID, "ranked"),
		IsPvE:        false,
		TargetTeamID: 1,
		NTeam:        4,
		NHumansLobby: 8,
		IsTeamMode:   true,
	}, nil
}

func (m *memoCountingEngagementRepo) LoadEventsForMatch(_ context.Context, _ string) ([]canonical.HighlightEvent, error) {
	return []canonical.HighlightEvent{
		{EventType: string(canonical.EventKill), TimeMS: 60_000, XUID: "xuid-test"},
		{EventType: string(canonical.EventKill), TimeMS: 120_000, XUID: "xuid-test"},
	}, nil
}

func (m *memoCountingEngagementRepo) LoadTeamXUIDs(_ context.Context, _ string, _ int, _ string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

// ListRecentPvPMatchIDs satisfait l'interface optionnelle `lister` du service
// (fast path SQL). Retourne les 200 ids (l'ordre est inverse cote service).
func (m *memoCountingEngagementRepo) ListRecentPvPMatchIDs(_ context.Context, _ string, _ int) ([]string, error) {
	return m.ids, nil
}

// --- stubs (non exerces par GetTimeseries) ---

func (m *memoCountingEngagementRepo) SaveResponseBins(context.Context, domain.EngagementResponseBins) error {
	return nil
}
func (m *memoCountingEngagementRepo) SaveEngagementScore(context.Context, string, string, domain.EngagementScoreResult) error {
	return nil
}
func (m *memoCountingEngagementRepo) SaveEngagementCoefficient(context.Context, domain.EngagementCoefficient) error {
	return nil
}
func (m *memoCountingEngagementRepo) SaveMatchIntensity(context.Context, string, float64) error {
	return nil
}
func (m *memoCountingEngagementRepo) LoadMatchIntensity(context.Context, string) (float64, bool, error) {
	return 0, false, nil
}
func (m *memoCountingEngagementRepo) HasEngagementScore(context.Context, string, string) (bool, error) {
	return false, nil
}
func (m *memoCountingEngagementRepo) LoadAllCoefficients(context.Context, string) ([]domain.EngagementCoefficient, error) {
	return nil, nil
}
func (m *memoCountingEngagementRepo) LoadRatioSamples(context.Context, string, string, int) ([]temporal.RatioSample, error) {
	return nil, nil
}

// TestGetTimeseries_MemoizesCoefAndBinsPerMode : sur 200 matchs couvrant 2
// mode_category, coef et bins ne sont lus qu'une fois par mode (E1). Avant le
// fix : une lecture coef + deux lectures bins (dont scan information_schema) PAR
// match, soit ~600 requetes pour 200 matchs.
func TestGetTimeseries_MemoizesCoefAndBinsPerMode(t *testing.T) {
	ids := make([]string, 0, 200)
	for i := 0; i < 100; i++ {
		ids = append(ids, fmt.Sprintf("ranked-%d", i))
		ids = append(ids, fmt.Sprintf("unranked-%d", i))
	}
	repo := &memoCountingEngagementRepo{ids: ids}
	svc := NewPlayerEngagementService(repo, "xuid-test", "GT-test")

	resp, err := svc.GetTimeseries(context.Background(), domain.FilterContextInput{}, 50)
	if err != nil {
		t.Fatalf("GetTimeseries: %v", err)
	}
	if resp == nil {
		t.Fatal("reponse nil")
	}
	if resp.TotalMatches != 200 {
		t.Fatalf("total_matches = %d, attendu 200 (le fast path doit servir les 200 ids)", resp.TotalMatches)
	}

	// 2 mode_category distinctes → exactement 1 lecture coef + 1 lecture bins par
	// mode (memoise), donc 2 + 2 = 4 AU PLUS quel que soit le nombre de matchs.
	if repo.coefCalls != 2 {
		t.Errorf("LoadEngagementCoefficient appele %d fois (attendu 2 = 1 par mode_category memoise)", repo.coefCalls)
	}
	if repo.binsCalls != 2 {
		t.Errorf("LoadResponseBins appele %d fois (attendu 2 = 1 par mode_category memoise)", repo.binsCalls)
	}
	if repo.coefCalls+repo.binsCalls > 4 {
		t.Errorf("total lectures coef+bins = %d pour 200 matchs (attendu ≤ 4)", repo.coefCalls+repo.binsCalls)
	}
}
