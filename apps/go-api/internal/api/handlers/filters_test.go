// Package handlers — filters_test.go : tests unitaires FiltersHandler avec mock service.
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

type mockFiltersService struct {
	result domain.FilterContextResolved
	err    error
}

func (m *mockFiltersService) Resolve(_ context.Context, _ domain.FilterContextInput) (domain.FilterContextResolved, error) {
	return m.result, m.err
}

func newFiltersRouter(factory handlers.ServiceFactory[port.FiltersService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewFiltersHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Post("/filters/resolve", h.Resolve)
	})
	return r
}

func TestFiltersHandler_Resolve_OK(t *testing.T) {
	expected := domain.FilterContextResolved{
		Counts: domain.FilterCounts{TotalMatchesAfterFilters: 42},
	}
	factory := func(_ context.Context, slug string) (port.FiltersService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return &mockFiltersService{result: expected}, nil
	}

	body, _ := json.Marshal(domain.FilterContextInput{})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/filters/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newFiltersRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp domain.FilterContextResolved
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Counts.TotalMatchesAfterFilters != expected.Counts.TotalMatchesAfterFilters {
		t.Errorf("TotalMatches: got %d, want %d", resp.Counts.TotalMatchesAfterFilters, expected.Counts.TotalMatchesAfterFilters)
	}
}

func TestFiltersHandler_Resolve_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.FiltersService, error) {
		return nil, errors.New("player_not_found")
	}

	body, _ := json.Marshal(domain.FilterContextInput{})
	req := httptest.NewRequest(http.MethodPost, "/players/unknown/filters/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newFiltersRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestFiltersHandler_Resolve_ServiceError(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.FiltersService, error) {
		return &mockFiltersService{err: errors.New("db error")}, nil
	}

	body, _ := json.Marshal(domain.FilterContextInput{})
	req := httptest.NewRequest(http.MethodPost, "/players/p/filters/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newFiltersRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
