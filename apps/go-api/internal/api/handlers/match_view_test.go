// Package handlers — match_view_test.go : tests unitaires MatchViewHandler avec mock service.
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

type mockMatchViewService struct {
	resp domain.MatchViewResponse
	err  error
}

func (m *mockMatchViewService) GetMatchView(_ context.Context, _ string) (domain.MatchViewResponse, error) {
	return m.resp, m.err
}

func newMatchViewRouter(factory handlers.ServiceFactory[port.MatchViewService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewMatchViewHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Get("/matches/{match_id}", h.GetMatchView)
	})
	return r
}

func TestMatchViewHandler_OK(t *testing.T) {
	expected := domain.MatchViewResponse{MatchID: "abc123"}
	factory := func(_ context.Context, slug string) (port.MatchViewService, error) {
		if slug != "test-player" {
			return nil, errors.New("player_not_found")
		}
		return &mockMatchViewService{resp: expected}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/matches/abc123", nil)
	w := httptest.NewRecorder()
	newMatchViewRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp domain.MatchViewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.MatchID != expected.MatchID {
		t.Errorf("MatchID: got %q, want %q", resp.MatchID, expected.MatchID)
	}
}

func TestMatchViewHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.MatchViewService, error) {
		return nil, errors.New("player_not_found")
	}

	req := httptest.NewRequest(http.MethodGet, "/players/unknown/matches/abc123", nil)
	w := httptest.NewRecorder()
	newMatchViewRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestMatchViewHandler_ServiceError(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.MatchViewService, error) {
		return &mockMatchViewService{err: errors.New("db error")}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/players/p/matches/xyz", nil)
	w := httptest.NewRecorder()
	newMatchViewRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
