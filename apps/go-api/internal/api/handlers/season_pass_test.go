// Package handlers_test — season_pass_test.go : tests unitaires SeasonPassHandler.
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

// mockSeasonPassService implémente port.SeasonPassService pour les tests.
type mockSeasonPassService struct {
	resp domain.SeasonPassPageResponse
	err  error
}

func (m *mockSeasonPassService) GetSeasonPassPage(_ context.Context) (domain.SeasonPassPageResponse, error) {
	return m.resp, m.err
}

// Compile-time check.
var _ port.SeasonPassService = (*mockSeasonPassService)(nil)

func newSeasonPassRouter(factory handlers.SeasonPassAuthFactory) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewSeasonPassHandler(factory)
	r.Route("/players/{player_slug}/pages/palmares", func(r chi.Router) {
		r.Get("/season-pass", h.GetSeasonPass)
	})
	return r
}

func TestSeasonPassHandler_OK(t *testing.T) {
	title := "test-title"
	mock := &mockSeasonPassService{
		resp: domain.SeasonPassPageResponse{
			TitleSlug: title,
			Available: true,
			Passes:    []domain.SeasonPassTrackSummary{},
		},
	}
	factory := func(ctx context.Context, slug string) (port.SeasonPassService, context.Context, error) {
		if slug != testPlayerSlug {
			return nil, ctx, errors.New("player_not_found")
		}
		return mock, ctx, nil
	}
	r := newSeasonPassRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/palmares/season-pass", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestSeasonPassHandler_PlayerNotFound(t *testing.T) {
	factory := func(ctx context.Context, slug string) (port.SeasonPassService, context.Context, error) {
		return nil, ctx, errors.New("player_not_found")
	}
	r := newSeasonPassRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/unknown-player/pages/palmares/season-pass", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSeasonPassHandler_ServiceError(t *testing.T) {
	mock := &mockSeasonPassService{err: errors.New("db error")}
	factory := func(ctx context.Context, slug string) (port.SeasonPassService, context.Context, error) {
		return mock, ctx, nil
	}
	r := newSeasonPassRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/palmares/season-pass", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
