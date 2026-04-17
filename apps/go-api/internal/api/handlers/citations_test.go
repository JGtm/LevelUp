// Package handlers_test — citations_test.go : tests unitaires CitationsHandler.
package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// mockCitationsService implémente port.CitationsService pour les tests.
type mockCitationsService struct {
	citationsPage     *domain.CitationsPageResponse
	citationsErr      error
	commendationsPage *domain.CommendationsPageResponse
	commendationsErr  error
}

func (m *mockCitationsService) GetCitationsPage(_ context.Context) (*domain.CitationsPageResponse, error) {
	return m.citationsPage, m.citationsErr
}

func (m *mockCitationsService) GetCommendationsPage(_ context.Context, _ string) (*domain.CommendationsPageResponse, error) {
	return m.commendationsPage, m.commendationsErr
}

func newCitationsRouter(factory handlers.ContextFactory[port.CitationsService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewCitationsHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Post("/pages/citations", h.GetCitations)
		r.Post("/pages/commendations", h.GetCommendations)
	})
	return r
}

func TestCitationsHandler_GetCitations_OK(t *testing.T) {
	mock := &mockCitationsService{citationsPage: &domain.CitationsPageResponse{}}
	factory := func(_ context.Context, slug string) (port.CitationsService, string, string, error) {
		if slug != testPlayerSlug {
			return nil, "", "", errors.New("player_not_found")
		}
		return mock, "xuid-1", "TestPlayer", nil
	}
	r := newCitationsRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/citations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCitationsHandler_GetCitations_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.CitationsService, string, string, error) {
		return nil, "", "", errors.New("player_not_found")
	}
	r := newCitationsRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/unknown/pages/citations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCitationsHandler_GetCitations_ServiceError(t *testing.T) {
	mock := &mockCitationsService{citationsErr: errors.New("db_error")}
	factory := func(_ context.Context, _ string) (port.CitationsService, string, string, error) {
		return mock, "xuid", "gt", nil
	}
	r := newCitationsRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/citations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestCitationsHandler_GetCitations_InvalidBody(t *testing.T) {
	mock := &mockCitationsService{citationsPage: &domain.CitationsPageResponse{}}
	factory := func(_ context.Context, _ string) (port.CitationsService, string, string, error) {
		return mock, "xuid", "gt", nil
	}
	r := newCitationsRouter(factory)
	body := strings.NewReader("{invalid json")
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/citations", body)
	req.ContentLength = int64(body.Len())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCitationsHandler_GetCommendations_OK(t *testing.T) {
	mock := &mockCitationsService{commendationsPage: &domain.CommendationsPageResponse{}}
	factory := func(_ context.Context, _ string) (port.CitationsService, string, string, error) {
		return mock, "xuid-1", "TestPlayer", nil
	}
	r := newCitationsRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/commendations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCitationsHandler_GetCommendations_ServiceError(t *testing.T) {
	mock := &mockCitationsService{commendationsErr: errors.New("db_error")}
	factory := func(_ context.Context, _ string) (port.CitationsService, string, string, error) {
		return mock, "xuid", "gt", nil
	}
	r := newCitationsRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/commendations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
