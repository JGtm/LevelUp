// Package service — tests pour PlayerEngagementService (Phase 8 plan).
//
// Stratégie : mock du port.EngagementScoreRepository pour tester les
// orchestrations sans dépendre de DuckDB. La logique de calcul pure
// (temporal.ComputeEngagementScore) est déjà testée séparément.
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// mockEngagementRepo implements port.EngagementScoreRepository for tests.
type mockEngagementRepo struct {
	matchCtx          *port.MatchEngagementContext
	matchCtxErr       error
	events            []canonical.HighlightEvent
	teamXUIDs         map[string]bool
	history           []domain.HistoricalEngagementBrut
	coef              *domain.EngagementCoefficient
	allCoefs          []domain.EngagementCoefficient
	loadAllCoefsErr   error
	loadCoefErr       error
	saveScoreCalls    int
	hasScoreReturnVal bool
}

func (m *mockEngagementRepo) LoadPlayerHistory(_ context.Context, _ port.EngagementHistoryFilter) ([]domain.HistoricalEngagementBrut, error) {
	return m.history, nil
}

func (m *mockEngagementRepo) LoadEngagementCoefficient(_ context.Context, _, _ string) (*domain.EngagementCoefficient, error) {
	return m.coef, m.loadCoefErr
}

func (m *mockEngagementRepo) SaveEngagementScore(_ context.Context, _, _ string, _ domain.EngagementScoreResult) error {
	m.saveScoreCalls++
	return nil
}

func (m *mockEngagementRepo) SaveEngagementCoefficient(_ context.Context, _ domain.EngagementCoefficient) error {
	return nil
}

func (m *mockEngagementRepo) SaveMatchIntensity(_ context.Context, _ string, _ float64) error {
	return port.ErrEngagementUnavailable
}

func (m *mockEngagementRepo) LoadMatchIntensity(_ context.Context, _ string) (float64, bool, error) {
	return 0, false, nil
}

func (m *mockEngagementRepo) HasEngagementScore(_ context.Context, _, _ string) (bool, error) {
	return m.hasScoreReturnVal, nil
}

func (m *mockEngagementRepo) LoadMatchEngagementContext(_ context.Context, _, _ string) (*port.MatchEngagementContext, error) {
	return m.matchCtx, m.matchCtxErr
}

func (m *mockEngagementRepo) LoadEventsForMatch(_ context.Context, _ string) ([]canonical.HighlightEvent, error) {
	return m.events, nil
}

func (m *mockEngagementRepo) LoadTeamXUIDs(_ context.Context, _ string, _ int, _ string) (map[string]bool, error) {
	return m.teamXUIDs, nil
}

func (m *mockEngagementRepo) LoadAllCoefficients(_ context.Context, _ string) ([]domain.EngagementCoefficient, error) {
	return m.allCoefs, m.loadAllCoefsErr
}

// =============================================================================
// PlayerEngagementService.GetMatchEngagement
// =============================================================================

func TestPlayerEngagement_MatchNotFound(t *testing.T) {
	svc := NewPlayerEngagementService(&mockEngagementRepo{matchCtx: nil}, "xuid-1", "Tester")
	_, err := svc.GetMatchEngagement(context.Background(), "match-x")
	if !errors.Is(err, ErrEngagementMatchNotFound) {
		t.Errorf("expected ErrEngagementMatchNotFound, got %v", err)
	}
}

func TestPlayerEngagement_PvENotSupported(t *testing.T) {
	svc := NewPlayerEngagementService(&mockEngagementRepo{
		matchCtx: &port.MatchEngagementContext{MatchID: "m", IsPvE: true},
	}, "xuid-1", "Tester")
	_, err := svc.GetMatchEngagement(context.Background(), "m")
	if !errors.Is(err, ErrEngagementPvENotSupported) {
		t.Errorf("expected ErrEngagementPvENotSupported, got %v", err)
	}
}

func TestPlayerEngagement_HappyPath_PvP(t *testing.T) {
	// Setup : match PvP 12 min avec quelques events.
	repo := &mockEngagementRepo{
		matchCtx: &port.MatchEngagementContext{
			MatchID:       "m1",
			StartTimeMS:   0,
			EndTimeMS:     720_000,
			IsRanked:      true,
			IsPvE:         false,
			TargetTeamID:  1,
			NTeam:         4,
			NHumansLobby:  8,
			IsTeamMode:    true,
			PersonalScore: 1000,
			Kills:         10,
			Assists:       3,
		},
		events: []canonical.HighlightEvent{
			{EventType: string(canonical.EventKill), TimeMS: 60_000, XUID: "xuid-1"},
			{EventType: string(canonical.EventKill), TimeMS: 120_000, XUID: "xuid-1"},
			{EventType: string(canonical.EventDeath), TimeMS: 180_000, XUID: "xuid-1"},
			{EventType: string(canonical.EventKill), TimeMS: 90_000, XUID: "ally-1"},
			{EventType: string(canonical.EventKill), TimeMS: 200_000, XUID: "ally-1"},
		},
		teamXUIDs: map[string]bool{"ally-1": true, "ally-2": true, "ally-3": true},
		history: []domain.HistoricalEngagementBrut{
			{MatchID: "h1", Brut: -1.0},
			{MatchID: "h2", Brut: -0.5},
			{MatchID: "h3", Brut: 0.0},
			{MatchID: "h4", Brut: 0.5},
			{MatchID: "h5", Brut: 1.0},
			// 5 + 5 = 10 minimum partial ; ajoutons en plus pour atteindre full
			{MatchID: "h6", Brut: -0.8},
			{MatchID: "h7", Brut: 0.2},
			{MatchID: "h8", Brut: 0.8},
			{MatchID: "h9", Brut: -0.3},
			{MatchID: "h10", Brut: 0.3},
		},
		coef: &domain.EngagementCoefficient{
			XUID:           "xuid-1",
			ModeCategory:   "PvP_ranked",
			CoefTeamShare:  1.0,
			CoefLobbyShare: 1.0,
			NMatches:       100,
			LastUpdated:    time.Now(),
		},
	}

	svc := NewPlayerEngagementService(repo, "xuid-1", "Tester")
	result, err := svc.GetMatchEngagement(context.Background(), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Confidence != "partial" && result.Confidence != "full" {
		t.Errorf("expected partial or full confidence, got %s", result.Confidence)
	}
	if len(result.EngagementCurve) == 0 {
		t.Error("expected non-empty curve")
	}
}

func TestPlayerEngagement_ColdStartUsesNeutralCoefs(t *testing.T) {
	// Aucun coef stocke -> defaut 1.0/1.0
	repo := &mockEngagementRepo{
		matchCtx: &port.MatchEngagementContext{
			MatchID:      "m",
			StartTimeMS:  0,
			EndTimeMS:    720_000,
			IsRanked:     false,
			NTeam:        4,
			NHumansLobby: 8,
			IsTeamMode:   true,
		},
		events: []canonical.HighlightEvent{
			{EventType: string(canonical.EventKill), TimeMS: 60_000, XUID: "xuid-1"},
			{EventType: string(canonical.EventKill), TimeMS: 90_000, XUID: "ally-1"},
		},
		teamXUIDs: map[string]bool{"ally-1": true},
		history:   []domain.HistoricalEngagementBrut{}, // insufficient
		coef:      nil,                                 // cold start
	}

	svc := NewPlayerEngagementService(repo, "xuid-1", "Tester")
	result, err := svc.GetMatchEngagement(context.Background(), "m")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Confidence != "insufficient_history" {
		t.Errorf("expected insufficient_history, got %s", result.Confidence)
	}
	if result.EngagementScore != nil {
		t.Errorf("expected nil score, got %v", *result.EngagementScore)
	}
	// Curve doit etre presente meme sans score
	if len(result.EngagementCurve) == 0 {
		t.Error("expected non-empty curve even without score")
	}
}

// =============================================================================
// PlayerEngagementService.GetEngagementProfile
// =============================================================================

func TestPlayerEngagement_Profile_Empty(t *testing.T) {
	svc := NewPlayerEngagementService(&mockEngagementRepo{allCoefs: []domain.EngagementCoefficient{}}, "xuid-1", "Tester")
	out, err := svc.GetEngagementProfile(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty profile, got %d entries", len(out))
	}
}

func TestPlayerEngagement_Profile_WithCoefs(t *testing.T) {
	coefs := []domain.EngagementCoefficient{
		{XUID: "xuid-1", ModeCategory: "PvP_ranked", CoefTeamShare: 1.12, CoefLobbyShare: 1.05, NMatches: 200},
		{XUID: "xuid-1", ModeCategory: "PvP_unranked", CoefTeamShare: 0.98, CoefLobbyShare: 0.95, NMatches: 150},
	}
	svc := NewPlayerEngagementService(&mockEngagementRepo{allCoefs: coefs}, "xuid-1", "Tester")
	out, err := svc.GetEngagementProfile(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("expected 2 coefs, got %d", len(out))
	}
}

func TestPlayerEngagement_Profile_UnavailableDegradesGracefully(t *testing.T) {
	svc := NewPlayerEngagementService(&mockEngagementRepo{loadAllCoefsErr: port.ErrEngagementUnavailable}, "xuid-1", "Tester")
	out, err := svc.GetEngagementProfile(context.Background())
	if err != nil {
		t.Fatalf("expected nil error on ErrEngagementUnavailable, got %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty profile on unavailable, got %d", len(out))
	}
}

func TestPlayerEngagement_Profile_EmptyXUID(t *testing.T) {
	svc := NewPlayerEngagementService(&mockEngagementRepo{}, "", "")
	_, err := svc.GetEngagementProfile(context.Background())
	if err == nil {
		t.Error("expected error on empty xuid")
	}
}

// =============================================================================
// IsCoefficientStale
// =============================================================================

func TestIsCoefficientStale(t *testing.T) {
	fresh := domain.EngagementCoefficient{LastUpdated: time.Now().Add(-7 * 24 * time.Hour)}
	if IsCoefficientStale(fresh) {
		t.Error("7-day-old coef should not be stale")
	}
	stale := domain.EngagementCoefficient{LastUpdated: time.Now().Add(-60 * 24 * time.Hour)}
	if !IsCoefficientStale(stale) {
		t.Error("60-day-old coef should be stale")
	}
}
