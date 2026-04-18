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
	"time"

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
		r.Get("/match-exclusions", h.ListExclusions)
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

func TestMatchExclusionHandler_List_OK(t *testing.T) {
	ts := time.Date(2024, 3, 15, 20, 0, 0, 0, time.UTC)
	mock := &mockMatchExclusionService{
		listed: []domain.ExcludedMatch{
			{MatchID: "match-abc", StartTime: ts, MapName: "Bazaar", ModeName: "Slayer"},
		},
	}
	factory := func(_ context.Context, slug string) (port.MatchExclusionService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newMatchExclusionRouter(factory)

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/match-exclusions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []domain.ExcludedMatch
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(resp) != 1 || resp[0].MatchID != "match-abc" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestMatchExclusionHandler_List_Empty(t *testing.T) {
	mock := &mockMatchExclusionService{listed: nil}
	factory := func(_ context.Context, slug string) (port.MatchExclusionService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newMatchExclusionRouter(factory)

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/match-exclusions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []domain.ExcludedMatch
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(resp) != 0 {
		t.Fatalf("expected empty array, got %+v", resp)
	}
}
