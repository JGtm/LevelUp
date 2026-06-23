// Package handlers_test — commendation_totals_test.go : tests CommendationTotalsHandler.
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

type mockCommendationTotalsService struct {
	resp *domain.NativeCommendationsTotalsResponse
	err  error
}

func (m *mockCommendationTotalsService) GetTotals(_ context.Context) (*domain.NativeCommendationsTotalsResponse, error) {
	return m.resp, m.err
}

func newCommendationTotalsRouter(factory handlers.ContextFactory[port.CommendationTotalsService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewCommendationTotalsHandler(factory)
	r.Route("/players/{player_slug}", func(sub chi.Router) {
		h.Mount(sub)
	})
	return r
}

func TestCommendationTotalsHandler_OK(t *testing.T) {
	mock := &mockCommendationTotalsService{resp: &domain.NativeCommendationsTotalsResponse{
		Categories: []domain.NativeCommendationCategoryGroup{}, TotalCount: 0,
	}}
	factory := func(_ context.Context, slug string) (port.CommendationTotalsService, string, string, error) {
		if slug != testPlayerSlug {
			return nil, "", "", errors.New("player_not_found")
		}
		return mock, testXUID1, testGamertag, nil
	}
	r := newCommendationTotalsRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/commendations/totals", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCommendationTotalsHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.CommendationTotalsService, string, string, error) {
		return nil, "", "", errors.New("player_not_found")
	}
	r := newCommendationTotalsRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/unknown/commendations/totals", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCommendationTotalsHandler_ServiceError(t *testing.T) {
	mock := &mockCommendationTotalsService{err: errors.New("db_error")}
	factory := func(_ context.Context, _ string) (port.CommendationTotalsService, string, string, error) {
		return mock, testXUID, "gt", nil
	}
	r := newCommendationTotalsRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/commendations/totals", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
