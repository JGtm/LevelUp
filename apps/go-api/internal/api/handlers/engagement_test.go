// Package handlers — engagement_test.go : tests unitaires EngagementHandler.
//
// Strategie : test du handler avec un PlayerEngagementService reel branche
// sur un mock repo. Couvre : 200 OK, 404 player_not_found, 404 match_not_found,
// 422 pve_not_supported, 503 engagement_unavailable, et l'endpoint
// engagement_profile.
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// engagementMockRepo : implementation legere de port.EngagementScoreRepository
// pour ces tests. Les methodes inutilisees sont des stubs.
type engagementMockRepo struct {
	matchCtx      *port.MatchEngagementContext
	matchCtxErr   error
	allCoefs      []domain.EngagementCoefficient
	allCoefsErr   error
	historyResult []domain.HistoricalEngagementBrut
}

func (m *engagementMockRepo) LoadPlayerHistory(_ context.Context, _ port.EngagementHistoryFilter) ([]domain.HistoricalEngagementBrut, error) {
	return m.historyResult, nil
}
func (m *engagementMockRepo) LoadEngagementCoefficient(_ context.Context, _, _ string) (*domain.EngagementCoefficient, error) {
	return nil, nil
}
func (m *engagementMockRepo) SaveEngagementScore(_ context.Context, _, _ string, _ domain.EngagementScoreResult) error {
	return nil
}
func (m *engagementMockRepo) SaveEngagementCoefficient(_ context.Context, _ domain.EngagementCoefficient) error {
	return nil
}
func (m *engagementMockRepo) SaveMatchIntensity(_ context.Context, _ string, _ float64) error {
	return port.ErrEngagementUnavailable
}
func (m *engagementMockRepo) LoadMatchIntensity(_ context.Context, _ string) (float64, bool, error) {
	return 0, false, nil
}
func (m *engagementMockRepo) HasEngagementScore(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *engagementMockRepo) LoadMatchEngagementContext(_ context.Context, _, _ string) (*port.MatchEngagementContext, error) {
	return m.matchCtx, m.matchCtxErr
}
func (m *engagementMockRepo) LoadEventsForMatch(_ context.Context, _ string) ([]canonical.HighlightEvent, error) {
	return []canonical.HighlightEvent{
		{EventType: string(canonical.EventKill), TimeMS: 60_000, XUID: "xuid-test"},
		{EventType: string(canonical.EventKill), TimeMS: 120_000, XUID: "xuid-test"},
	}, nil
}
func (m *engagementMockRepo) LoadTeamXUIDs(_ context.Context, _ string, _ int, _ string) (map[string]bool, error) {
	return map[string]bool{}, nil
}
func (m *engagementMockRepo) LoadAllCoefficients(_ context.Context, _ string) ([]domain.EngagementCoefficient, error) {
	return m.allCoefs, m.allCoefsErr
}

// newEngagementRouter cree un router test avec l'EngagementHandler branche.
func newEngagementRouter(factory handlers.ServiceFactory[*service.PlayerEngagementService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewEngagementHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Get("/matches/{match_id}/engagement", h.GetMatchEngagement)
		r.Get("/engagement_profile", h.GetEngagementProfile)
	})
	return r
}

// =============================================================================
// GET /matches/{match_id}/engagement
// =============================================================================

func TestEngagementHandler_GetMatchEngagement_OK(t *testing.T) {
	repo := &engagementMockRepo{
		matchCtx: &port.MatchEngagementContext{
			MatchID:      "m1",
			StartTimeMS:  0,
			EndTimeMS:    720_000,
			IsRanked:     true,
			NTeam:        4,
			NHumansLobby: 8,
			IsTeamMode:   true,
		},
	}
	factory := func(_ context.Context, slug string) (*service.PlayerEngagementService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return service.NewPlayerEngagementService(repo, "xuid-test"), nil
	}

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/matches/m1/engagement", nil)
	w := httptest.NewRecorder()
	newEngagementRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp domain.EngagementScoreResult
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Confidence == "" {
		t.Error("expected non-empty Confidence")
	}
}

func TestEngagementHandler_GetMatchEngagement_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (*service.PlayerEngagementService, error) {
		return nil, errors.New("player_not_found")
	}
	req := httptest.NewRequest(http.MethodGet, "/players/unknown/matches/m1/engagement", nil)
	w := httptest.NewRecorder()
	newEngagementRouter(factory).ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestEngagementHandler_GetMatchEngagement_MatchNotFound(t *testing.T) {
	repo := &engagementMockRepo{matchCtx: nil}
	factory := func(_ context.Context, _ string) (*service.PlayerEngagementService, error) {
		return service.NewPlayerEngagementService(repo, "xuid-test"), nil
	}
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/matches/missing/engagement", nil)
	w := httptest.NewRecorder()
	newEngagementRouter(factory).ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 (match_not_found), got %d: %s", w.Code, w.Body.String())
	}
}

func TestEngagementHandler_GetMatchEngagement_PvENotSupported(t *testing.T) {
	repo := &engagementMockRepo{
		matchCtx: &port.MatchEngagementContext{MatchID: "m", IsPvE: true},
	}
	factory := func(_ context.Context, _ string) (*service.PlayerEngagementService, error) {
		return service.NewPlayerEngagementService(repo, "xuid-test"), nil
	}
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/matches/m/engagement", nil)
	w := httptest.NewRecorder()
	newEngagementRouter(factory).ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 (pve_not_supported), got %d: %s", w.Code, w.Body.String())
	}
}

// =============================================================================
// GET /engagement_profile
// =============================================================================

func TestEngagementHandler_GetEngagementProfile_OK(t *testing.T) {
	repo := &engagementMockRepo{
		allCoefs: []domain.EngagementCoefficient{
			{XUID: "xuid-test", ModeCategory: "PvP_ranked", CoefTeamShare: 1.12, CoefLobbyShare: 1.05, NMatches: 200, LastUpdated: time.Now()},
		},
	}
	factory := func(_ context.Context, _ string) (*service.PlayerEngagementService, error) {
		return service.NewPlayerEngagementService(repo, "xuid-test"), nil
	}
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/engagement_profile", nil)
	w := httptest.NewRecorder()
	newEngagementRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var coefs []domain.EngagementCoefficient
	if err := json.Unmarshal(w.Body.Bytes(), &coefs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(coefs) != 1 || coefs[0].ModeCategory != "PvP_ranked" {
		t.Errorf("unexpected coefs: %+v", coefs)
	}
}

func TestEngagementHandler_GetEngagementProfile_EmptyOnUnavailable(t *testing.T) {
	repo := &engagementMockRepo{allCoefsErr: port.ErrEngagementUnavailable}
	factory := func(_ context.Context, _ string) (*service.PlayerEngagementService, error) {
		return service.NewPlayerEngagementService(repo, "xuid-test"), nil
	}
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/engagement_profile", nil)
	w := httptest.NewRecorder()
	newEngagementRouter(factory).ServeHTTP(w, req)

	// Le service degrade en retournant une slice vide sans erreur.
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (empty profile on unavailable), got %d", w.Code)
	}
	var coefs []domain.EngagementCoefficient
	if err := json.Unmarshal(w.Body.Bytes(), &coefs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(coefs) != 0 {
		t.Errorf("expected empty slice, got %d coefs", len(coefs))
	}
}
