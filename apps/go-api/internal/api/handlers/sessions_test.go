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
}

func (m *mockSessionsService) GetSessions(_ context.Context, _ domain.SessionComputeOptions) (domain.SessionsResponse, error) {
	return m.resp, m.err
}

func newSessionsRouter(factory handlers.ServiceFactory[port.SessionsService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewSessionsHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Get("/pages/sessions", h.GetSessions)
	})
	return r
}

func TestSessionsHandler_OK(t *testing.T) {
	mock := &mockSessionsService{
		resp: domain.SessionsResponse{Sessions: []domain.Session{{SessionID: "s1"}}},
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
