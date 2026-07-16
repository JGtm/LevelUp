// handlers_extra_test.go — tests HTTP supplémentaires (package handlers_test).
//
// Couvre : CareerHandler (GetTopMatches, GetEncounters),
// MatchHistoryHandler (Export OK/NotFound/Invalid/Error),
// ExplorerHandler (QueryMatches OK/NotFound/Error/BadBody),
// PlayersHandler (OK/Error).
package handlers_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// ---------------------------------------------------------------------------
// CareerHandler — GetTopMatches
// ---------------------------------------------------------------------------

func TestCareerHandler_GetTopMatches_OK(t *testing.T) {
	mock := &mockCareerService{topMatches: domain.CareerTopMatchesResponse{}}
	factory := func(_ context.Context, slug string) (port.CareerService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("not_found")
		}
		return mock, nil
	}
	r := newTestRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/career/top-matches", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCareerHandler_GetTopMatches_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.CareerService, error) {
		return nil, errors.New("not_found")
	}
	r := newTestRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/unknown/pages/career/top-matches", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCareerHandler_GetTopMatches_ServiceError(t *testing.T) {
	mock := &mockCareerService{topErr: errors.New("boom")}
	factory := func(_ context.Context, _ string) (port.CareerService, error) {
		return mock, nil
	}
	r := newTestRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/career/top-matches", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// CareerHandler — GetEncounters
// ---------------------------------------------------------------------------

func TestCareerHandler_GetEncounters_OK(t *testing.T) {
	mock := &mockCareerService{encounters: domain.CareerEncountersResponse{}}
	factory := func(_ context.Context, slug string) (port.CareerService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("not_found")
		}
		return mock, nil
	}
	r := newTestRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/career/encounters", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCareerHandler_GetEncounters_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.CareerService, error) {
		return nil, errors.New("not_found")
	}
	r := newTestRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/unknown/pages/career/encounters", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCareerHandler_GetEncounters_ServiceError(t *testing.T) {
	mock := &mockCareerService{encounterErr: errors.New("boom")}
	factory := func(_ context.Context, _ string) (port.CareerService, error) {
		return mock, nil
	}
	r := newTestRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/career/encounters", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// MatchHistoryHandler — Export
// ---------------------------------------------------------------------------

func makeExportToken(t *testing.T) string {
	t.Helper()
	data, err := json.Marshal(domain.MatchHistoryQueryRequest{})
	if err != nil {
		t.Fatal(err)
	}
	return base64.URLEncoding.EncodeToString(data)
}

func TestMatchHistoryHandler_Export_OK(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock := &mockMatchHistoryService{
		csvRows: []domain.MatchHistoryRow{
			{MatchID: "m1", StartTime: now, OutcomeLabel: "WIN", ScoreLabel: "25-10", MatchURL: "/m/m1"},
		},
	}
	factory := func(_ context.Context, slug string) (port.MatchHistoryService, string, string, error) {
		if slug != testPlayerSlug {
			return nil, "", "", errors.New("not_found")
		}
		return mock, testXUID1B, "Player1", nil
	}
	r := newMatchHistoryRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/match-history/export?token="+makeExportToken(t), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/csv; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/csv", ct)
	}
	if disp := w.Header().Get("Content-Disposition"); disp == "" {
		t.Error("missing Content-Disposition header")
	}
}

func TestMatchHistoryHandler_Export_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.MatchHistoryService, string, string, error) {
		return nil, "", "", errors.New("not_found")
	}
	r := newMatchHistoryRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/unknown/pages/match-history/export?token=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestMatchHistoryHandler_Export_InvalidToken(t *testing.T) {
	mock := &mockMatchHistoryService{}
	factory := func(_ context.Context, _ string) (port.MatchHistoryService, string, string, error) {
		return mock, "x", "g", nil
	}
	r := newMatchHistoryRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/match-history/export?token=!!!invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestMatchHistoryHandler_Export_ServiceError(t *testing.T) {
	mock := &mockMatchHistoryService{csvErr: errors.New("export_fail")}
	factory := func(_ context.Context, _ string) (port.MatchHistoryService, string, string, error) {
		return mock, "x", "g", nil
	}
	r := newMatchHistoryRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/match-history/export?token="+makeExportToken(t), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// ExplorerHandler — QueryMatches
// ---------------------------------------------------------------------------

func TestExplorerHandler_QueryMatches_OK(t *testing.T) {
	mockMH := &mockMatchHistoryForExplorer{
		page: domain.MatchHistoryPageResponse{},
	}
	explorerF := func(ctx context.Context, _ string) (port.ExplorerService, context.Context, string, string, error) {
		return &mockExplorerService{}, ctx, testXUID1B, "GT", nil
	}
	matchHistF := func(_ context.Context, slug string) (port.MatchHistoryService, string, string, error) {
		if slug != testPlayerSlug {
			return nil, "", "", errors.New("not_found")
		}
		return mockMH, testXUID1B, "GT", nil
	}
	r := newExplorerRouter(explorerF, matchHistF)
	body := `{"filters":{},"pagination":{"page":1,"per_page":20}}`
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/explorer/matches-query", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExplorerHandler_QueryMatches_PlayerNotFound(t *testing.T) {
	explorerF := func(ctx context.Context, _ string) (port.ExplorerService, context.Context, string, string, error) {
		return &mockExplorerService{}, ctx, "x", "g", nil
	}
	matchHistF := func(_ context.Context, _ string) (port.MatchHistoryService, string, string, error) {
		return nil, "", "", errors.New("not_found")
	}
	r := newExplorerRouter(explorerF, matchHistF)
	body := `{"filters":{},"pagination":{"page":1,"per_page":20}}`
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/explorer/matches-query", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestExplorerHandler_QueryMatches_ServiceError(t *testing.T) {
	mockMH := &mockMatchHistoryForExplorer{pageErr: errors.New("boom")}
	explorerF := func(ctx context.Context, _ string) (port.ExplorerService, context.Context, string, string, error) {
		return &mockExplorerService{}, ctx, "x", "g", nil
	}
	matchHistF := func(_ context.Context, _ string) (port.MatchHistoryService, string, string, error) {
		return mockMH, "x", "g", nil
	}
	r := newExplorerRouter(explorerF, matchHistF)
	body := `{"filters":{},"pagination":{"page":1,"per_page":20}}`
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/explorer/matches-query", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestExplorerHandler_QueryMatches_InvalidBody(t *testing.T) {
	explorerF := func(ctx context.Context, _ string) (port.ExplorerService, context.Context, string, string, error) {
		return &mockExplorerService{}, ctx, "x", "g", nil
	}
	matchHistF := func(_ context.Context, _ string) (port.MatchHistoryService, string, string, error) {
		return &mockMatchHistoryForExplorer{}, "x", "g", nil
	}
	r := newExplorerRouter(explorerF, matchHistF)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/explorer/matches-query", bytes.NewBufferString(`{bad`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// PlayersHandler
// ---------------------------------------------------------------------------

func TestPlayersHandler_OK(t *testing.T) {
	repo := &mockBootstrapRepo{playerCount: 2, dbVersion: "v1", matchCount: 100}
	cfg := &config.AppConfig{DemoMode: true}
	svc := service.NewBootstrapService(cfg, repo)
	h := handlers.NewPlayersHandler(svc)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) { h.Mount(r) })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/players", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// CitationsHandler — GetCitations with category filter
// ---------------------------------------------------------------------------

func TestCitationsHandler_GetCitations_WithCategoryFilter(t *testing.T) {
	mock := &mockCitationsService{
		citationsPage: &domain.CitationsPageResponse{
			Citations: []domain.CitationItem{
				{NameNorm: "a", Category: "Multikill"},
				{NameNorm: "b", Category: "Style"},
			},
			Categories: []string{"Multikill", "Style"},
			TotalCount: 2,
		},
	}
	factory := func(_ context.Context, slug string) (port.CitationsService, string, string, error) {
		if slug != testPlayerSlug {
			return nil, "", "", errors.New("not_found")
		}
		return mock, testXUID1B, "GT", nil
	}
	r := newCitationsRouter(factory)
	body := `{"category":"Multikill"}`
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/citations", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCitationsHandler_GetCommendations_WithCategoryFilter(t *testing.T) {
	mock := &mockCitationsService{
		commendationsPage: &domain.CommendationsPageResponse{
			Categories: []domain.CommendationCategory{
				{Category: "Combat", Total: 10},
				{Category: "Style", Total: 5},
			},
			TotalCount: 15,
		},
	}
	factory := func(_ context.Context, slug string) (port.CitationsService, string, string, error) {
		if slug != testPlayerSlug {
			return nil, "", "", errors.New("not_found")
		}
		return mock, testXUID1B, "GT", nil
	}
	r := newCitationsRouter(factory)
	body := `{"category":"Combat"}`
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/commendations", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// SetupHandler — CreatePlayer (manual mode, covers success path)
// ---------------------------------------------------------------------------

func TestSetupHandler_CreatePlayer_ManualMode_OK(t *testing.T) {
	svc := &mockProfileService{playerKey: "test-gt"}
	r := newSetupRouter(t, true, svc)
	body := `{"gamertag":"TestGT","profile_mode":"manual"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/players", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetupHandler_CreatePlayer_ProfileServiceError(t *testing.T) {
	svc := &mockProfileService{err: errors.New("disk full")}
	r := newSetupRouter(t, true, svc)
	body := `{"gamertag":"TestGT","profile_mode":"manual"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/players", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// SyncHandler — StartInitialSync (no auth tokens)
// ---------------------------------------------------------------------------

func TestSyncHandler_InitialSync_NoTokens(t *testing.T) {
	r, _ := newSyncRouter(t, true)
	body := `{"player_slug":"test-player","max_matches":100}`
	req := httptest.NewRequest(http.MethodPost, "/sync/initial", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// No session with HaloTokens → 401
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HomeHandler — BattlePass/Challenges NotFound
// ---------------------------------------------------------------------------

func TestHomeHandler_GetBattlePass_PlayerNotFound(t *testing.T) {
	factory := func(ctx context.Context, _ string) (port.HomeService, context.Context, string, string, error) {
		return nil, ctx, "", "", errors.New("not_found")
	}
	r := newHomeRouter(factory, nil)
	req := httptest.NewRequest(http.MethodGet, "/players/unknown/battlepass", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// NB (Phase 3b) : les tests TestSquadHandler_GetSynthesisPage_* ont été retirés —
// la route POST /pages/synthesis est servie par SynthesisHandler (server.go), pas
// par SquadHandler ; sa méthode HTTP côté squad était morte (jamais montée en prod)
// et n'est plus enregistrée par squad.Mount. Couverture réelle : synthesis_handler_test.go.

// ---------------------------------------------------------------------------
// MediaHandler — PostUploadMedia guard (nil factory)
// ---------------------------------------------------------------------------

func TestMediaHandler_PostUploadMedia_NotConfigured(t *testing.T) {
	// Create handler with nil upload factory and a route for upload.
	factory := func(_ context.Context, slug string) (port.MediaService, error) {
		return &mockMediaService{}, nil
	}
	h := handlers.NewMediaHandler(factory, nil, "")
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Post("/media/upload", h.PostUploadMedia)
	})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/media/upload", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// MatchHistory.Query with IncludeExportHint
// ---------------------------------------------------------------------------

func TestMatchHistoryHandler_Query_WithExportHint(t *testing.T) {
	mock := &mockMatchHistoryService{
		page: domain.MatchHistoryPageResponse{
			ExportHint: &domain.ExportHint{},
		},
	}
	factory := func(_ context.Context, slug string) (port.MatchHistoryService, string, string, error) {
		if slug != testPlayerSlug {
			return nil, "", "", errors.New("not_found")
		}
		return mock, testXUID1B, "Player1", nil
	}
	r := newMatchHistoryRouter(factory)
	body := `{"filters":{},"pagination":{"page":1,"page_size":20},"include_export_hint":true}`
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/match-history/query", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
