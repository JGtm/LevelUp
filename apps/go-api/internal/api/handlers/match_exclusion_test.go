// Package handlers_test — match_exclusion_test.go : tests unitaires MatchExclusionHandler.
package handlers_test

import (
	"bytes"
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

// mockMatchExclusionService implémente port.MatchExclusionService.
type mockMatchExclusionService struct {
	setErr  error
	listed  []domain.ExcludedMatch
	listErr error
}

func (m *mockMatchExclusionService) SetExclusion(_ context.Context, _ string, _ bool) error {
	return m.setErr
}

func (m *mockMatchExclusionService) ListExcluded(_ context.Context) ([]domain.ExcludedMatch, error) {
	return m.listed, m.listErr
}

func newMatchExclusionRouter(factory handlers.ServiceFactory[port.MatchExclusionService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewMatchExclusionHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Patch("/matches/{match_id}/exclusion", h.SetExclusion)
	})
	return r
}

func TestMatchExclusionHandler_Set_OK(t *testing.T) {
	mock := &mockMatchExclusionService{}
	factory := func(_ context.Context, slug string) (port.MatchExclusionService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newMatchExclusionRouter(factory)

	body, _ := json.Marshal(domain.SetMatchExclusionRequest{Excluded: true})
	req := httptest.NewRequest(http.MethodPatch, "/players/test-player/matches/match-abc/exclusion", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMatchExclusionHandler_Set_DBError(t *testing.T) {
	mock := &mockMatchExclusionService{setErr: errors.New("db error")}
	factory := func(_ context.Context, slug string) (port.MatchExclusionService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newMatchExclusionRouter(factory)

	body, _ := json.Marshal(domain.SetMatchExclusionRequest{Excluded: true})
	req := httptest.NewRequest(http.MethodPatch, "/players/test-player/matches/match-abc/exclusion", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMatchExclusionHandler_Set_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.MatchExclusionService, error) {
		return nil, errors.New("player_not_found")
	}
	r := newMatchExclusionRouter(factory)

	body, _ := json.Marshal(domain.SetMatchExclusionRequest{Excluded: true})
	req := httptest.NewRequest(http.MethodPatch, "/players/unknown/matches/match-abc/exclusion", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestMatchExclusionHandler_List_* supprimés en revue 2026-04-29 P0.2 Q6
// (endpoint GET /match-exclusions retiré, voir match_exclusion.go).
