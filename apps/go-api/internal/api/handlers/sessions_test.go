// Package handlers_test — sessions_test.go : tests unitaires SessionsHandler.
package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// mockSessionsService implémente port.SessionsService pour les tests.
type mockSessionsService struct {
	resp domain.SessionsResponse
	err  error
	rule domain.SessionComputeOptions
	got  domain.SessionComputeOptions // options reçues par GetSessions
}

func (m *mockSessionsService) SessionOptions(_ context.Context) domain.SessionComputeOptions {
	return m.rule
}

func (m *mockSessionsService) GetSessions(_ context.Context, opts domain.SessionComputeOptions) (domain.SessionsResponse, error) {
	m.got = opts
	return m.resp, m.err
}

// TestSessionsHandler_UsesPlayerRuleWithoutParams : sans paramètre, la page découpe avec la
// règle du joueur (celle des écrivains de session_id) ; un paramètre la surcharge.
func TestSessionsHandler_UsesPlayerRuleWithoutParams(t *testing.T) {
	rule := domain.SessionComputeOptions{
		GapMinutes: 75, TeamChangeMode: domain.TeamChangeModeGroup,
		FriendsXUIDs: []string{"xa"}, Mode: domain.SessionModeContext,
	}
	mock := &mockSessionsService{rule: rule}
	r := newSessionsRouter(func(context.Context, string) (port.SessionsService, error) { return mock, nil })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/players/test-player/pages/sessions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if mock.got.GapMinutes != 75 || mock.got.TeamChangeMode != domain.TeamChangeModeGroup || len(mock.got.FriendsXUIDs) != 1 {
		t.Fatalf("options transmises %+v, attendu la règle du joueur %+v", mock.got, rule)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/players/test-player/pages/sessions?gap_minutes=30", nil))
	if mock.got.GapMinutes != 30 || mock.got.TeamChangeMode != domain.TeamChangeModeGroup {
		t.Fatalf("surcharge gap_minutes : %+v, attendu gap=30 sur la règle du joueur", mock.got)
	}
}

func newSessionsRouter(factory handlers.ServiceFactory[port.SessionsService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewSessionsHandler(factory)
	r.Route("/players/{player_slug}", func(sub chi.Router) {
		h.Mount(sub)
	})
	return r
}

func TestSessionsHandler_OK(t *testing.T) {
	mock := &mockSessionsService{
		resp: domain.SessionsResponse{Sessions: []domain.SessionGroup{{SessionID: 1}}},
	}
	factory := func(_ context.Context, slug string) (port.SessionsService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newSessionsRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/sessions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSessionsHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.SessionsService, error) {
		return nil, errors.New("unknown player")
	}
	r := newSessionsRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/unknown/pages/sessions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSessionsHandler_ServiceError(t *testing.T) {
	mock := &mockSessionsService{err: errors.New("db_error")}
	factory := func(_ context.Context, _ string) (port.SessionsService, error) {
		return mock, nil
	}
	r := newSessionsRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/sessions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestSessionsHandler_WithGapParam(t *testing.T) {
	mock := &mockSessionsService{resp: domain.SessionsResponse{}}
	factory := func(_ context.Context, _ string) (port.SessionsService, error) {
		return mock, nil
	}
	r := newSessionsRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/sessions?gap_minutes=30&mode=daily&split_ranked=true", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
