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

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// engagementMockRepo : implementation legere de port.EngagementScoreRepository
// pour ces tests. Les methodes inutilisees sont des stubs.
type engagementMockRepo struct {
	matchCtx        *port.MatchEngagementContext
	matchCtxErr     error
	allCoefs        []domain.EngagementCoefficient
	allCoefsErr     error
	historyResult   []domain.HistoricalEngagementBrut
	ratioSamples    []temporal.RatioSample
	ratioSamplesErr error
	saveCoefCalls   int
	saveCoefErr     error
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
	m.saveCoefCalls++
	return m.saveCoefErr
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
func (m *engagementMockRepo) LoadRatioSamples(_ context.Context, _, _ string, _ int) ([]temporal.RatioSample, error) {
	return m.ratioSamples, m.ratioSamplesErr
}

// newEngagementRouter cree un router test avec l'EngagementHandler branche.
func newEngagementRouter(factory handlers.ServiceFactory[*service.PlayerEngagementService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewEngagementHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Get("/matches/{match_id}/engagement", h.GetMatchEngagement)
		r.Get("/engagement_profile", h.GetEngagementProfile)
		r.Post("/engagement/recompute_coefficients", h.PostRecomputeCoefficients)
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
		return service.NewPlayerEngagementService(repo, "xuid-test", "Tester"), nil
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
		return service.NewPlayerEngagementService(repo, "xuid-test", "Tester"), nil
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
		return service.NewPlayerEngagementService(repo, "xuid-test", "Tester"), nil
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
		return service.NewPlayerEngagementService(repo, "xuid-test", "Tester"), nil
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
		return service.NewPlayerEngagementService(repo, "xuid-test", "Tester"), nil
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

// =============================================================================
// POST /engagement/recompute_coefficients
// =============================================================================

// validRatioSamples genere n samples qui passent les filtres de
// ComputeEngagementCoefficient (PaceTeam >= 1, PlayerActivity >= 3).
func validRatioSamples(n int, ratioTeam, ratioLobby float64) []temporal.RatioSample {
	const baseTeam = 10.0
	out := make([]temporal.RatioSample, n)
	for i := range out {
		out[i] = temporal.RatioSample{
			MatchID:        "m" + string(rune('0'+i%10)),
			PaceJoueur:     baseTeam * ratioTeam,
			PaceTeam:       baseTeam,
			PaceLobby:      baseTeam,
			PlayerActivity: 30,
		}
		if ratioLobby > 0 {
			out[i].PaceLobby = out[i].PaceJoueur / ratioLobby
		}
	}
	return out
}

func TestEngagementHandler_RecomputeCoefficients_OK(t *testing.T) {
	repo := &engagementMockRepo{
		ratioSamples: validRatioSamples(15, 1.25, 1.10),
	}
	factory := func(_ context.Context, _ string) (*service.PlayerEngagementService, error) {
		return service.NewPlayerEngagementService(repo, "xuid-test", "Tester"), nil
	}
	req := httptest.NewRequest(http.MethodPost,
		"/players/test-player/engagement/recompute_coefficients", nil)
	w := httptest.NewRecorder()
	newEngagementRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var report service.RecomputeReport
	if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Les 2 modes (PvP_ranked + PvP_unranked) doivent être traités —
	// le mock retourne les mêmes samples pour les 2.
	if report.NCoefsPersisted != 2 {
		t.Errorf("NCoefsPersisted want 2, got %d", report.NCoefsPersisted)
	}
	if len(report.ModesUpdated) != 2 {
		t.Errorf("ModesUpdated want 2, got %v", report.ModesUpdated)
	}
	// SaveEngagementCoefficient doit avoir été appelé 2 fois.
	if repo.saveCoefCalls != 2 {
		t.Errorf("saveCoefCalls want 2, got %d", repo.saveCoefCalls)
	}
}

func TestEngagementHandler_RecomputeCoefficients_InsufficientHistory(t *testing.T) {
	// 5 samples → sous le seuil → modes skipped, no save.
	repo := &engagementMockRepo{
		ratioSamples: validRatioSamples(5, 1.0, 1.0),
	}
	factory := func(_ context.Context, _ string) (*service.PlayerEngagementService, error) {
		return service.NewPlayerEngagementService(repo, "xuid-test", "Tester"), nil
	}
	req := httptest.NewRequest(http.MethodPost,
		"/players/test-player/engagement/recompute_coefficients", nil)
	w := httptest.NewRecorder()
	newEngagementRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (skipped, not error), got %d: %s", w.Code, w.Body.String())
	}
	var report service.RecomputeReport
	if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if report.NCoefsPersisted != 0 {
		t.Errorf("NCoefsPersisted want 0 (insufficient), got %d", report.NCoefsPersisted)
	}
	if len(report.ModesSkipped) != 2 {
		t.Errorf("ModesSkipped want 2 (both insufficient), got %v", report.ModesSkipped)
	}
	if repo.saveCoefCalls != 0 {
		t.Errorf("saveCoefCalls want 0, got %d", repo.saveCoefCalls)
	}
}

func TestEngagementHandler_RecomputeCoefficients_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (*service.PlayerEngagementService, error) {
		return nil, errors.New("player_not_found")
	}
	req := httptest.NewRequest(http.MethodPost,
		"/players/unknown/engagement/recompute_coefficients", nil)
	w := httptest.NewRecorder()
	newEngagementRouter(factory).ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestEngagementHandler_RecomputeCoefficients_Unavailable(t *testing.T) {
	repo := &engagementMockRepo{
		ratioSamplesErr: port.ErrEngagementUnavailable,
	}
	factory := func(_ context.Context, _ string) (*service.PlayerEngagementService, error) {
		return service.NewPlayerEngagementService(repo, "xuid-test", "Tester"), nil
	}
	req := httptest.NewRequest(http.MethodPost,
		"/players/test-player/engagement/recompute_coefficients", nil)
	w := httptest.NewRecorder()
	newEngagementRouter(factory).ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

// =============================================================================
// GET /pages/squad/v2/engagement (GetSquadEngagementSession)
// =============================================================================

func newSquadEngagementRouter(factory handlers.ServiceFactory[*service.PlayerEngagementService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewEngagementHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Get("/pages/squad/v2/engagement", h.GetSquadEngagementSession)
	})
	return r
}

// TestEngagementHandler_GetSquadSession_ExplicitMatchIDs : le chemin nominal
// post-fix 2026-06-10 — le front passe match_ids explicites. Le handler doit
// parser les CSV (ids + teammates zippés avec les gamertags) et retourner une
// session 200 avec main + coéquipiers dans players (même si les bundles
// échouent avec le repo stub : la structure est garantie).
func TestEngagementHandler_GetSquadSession_ExplicitMatchIDs(t *testing.T) {
	repo := &engagementMockRepo{}
	factory := func(_ context.Context, slug string) (*service.PlayerEngagementService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return service.NewPlayerEngagementService(repo, "xuid-main", "MainGT"), nil
	}

	req := httptest.NewRequest(http.MethodGet,
		"/players/test-player/pages/squad/v2/engagement?match_ids=m1,m2,m3&teammates=x1,x2&teammate_gamertags=Ami1,Ami2", nil)
	w := httptest.NewRecorder()
	newSquadEngagementRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp domain.SquadEngagementSession
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Players) != 3 {
		t.Fatalf("players = %d, want 3 (main + 2 coéquipiers)", len(resp.Players))
	}
	if resp.Players[0].Gamertag != "MainGT" {
		t.Errorf("players[0] = %q, want le main en premier", resp.Players[0].Gamertag)
	}
	if resp.Players[1].XUID != "x1" || resp.Players[1].Gamertag != "Ami1" {
		t.Errorf("teammate 1 mal zippé : %+v", resp.Players[1])
	}
}

// TestEngagementHandler_GetSquadSession_FallbackEmptyDerivation : sans
// match_ids, le handler dérive depuis GetTimeseries ; avec un historique vide,
// la dérivation produit 0 match et la réponse DOIT être une session vide 200
// (labels []), pas une erreur — le front (ChartCard) affiche son état vide.
func TestEngagementHandler_GetSquadSession_FallbackEmptyDerivation(t *testing.T) {
	repo := &engagementMockRepo{} // historyResult vide → timeseries vide
	factory := func(_ context.Context, _ string) (*service.PlayerEngagementService, error) {
		return service.NewPlayerEngagementService(repo, "xuid-main", "MainGT"), nil
	}

	req := httptest.NewRequest(http.MethodGet,
		"/players/test-player/pages/squad/v2/engagement?teammates=x1&teammate_gamertags=Ami1", nil)
	w := httptest.NewRecorder()
	newSquadEngagementRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp domain.SquadEngagementSession
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Labels) != 0 {
		t.Errorf("labels = %d, want 0 (session vide propre)", len(resp.Labels))
	}
}
