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
	result   domain.FilterContextResolved
	matchIDs []string
	err      error
}

func (m *mockFiltersService) Resolve(_ context.Context, _ domain.FilterContextInput) (domain.FilterContextResolved, error) {
	return m.result, m.err
}

func (m *mockFiltersService) ResolveMatchIDs(_ context.Context, _ domain.FilterContextInput) ([]string, error) {
	return m.matchIDs, m.err
}

func newFiltersRouter(factory handlers.ServiceFactory[port.FiltersService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewFiltersHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
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

func TestFiltersHandler_MatchIDs_OK(t *testing.T) {
	factory := func(_ context.Context, slug string) (port.FiltersService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return &mockFiltersService{matchIDs: []string{"m3", "m2", "m1"}}, nil
	}

	body, _ := json.Marshal(domain.FilterContextInput{})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/filters/match-ids", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newFiltersRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp domain.FilterMatchIDsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := []string{"m3", "m2", "m1"}
	if len(resp.MatchIDs) != len(want) {
		t.Fatalf("match_ids = %v, want %v", resp.MatchIDs, want)
	}
	for i := range want {
		if resp.MatchIDs[i] != want[i] {
			t.Errorf("match_ids[%d] = %q, want %q", i, resp.MatchIDs[i], want[i])
		}
	}
}

func TestFiltersHandler_MatchIDs_EmptyIsNotNull(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.FiltersService, error) {
		return &mockFiltersService{matchIDs: nil}, nil
	}

	body, _ := json.Marshal(domain.FilterContextInput{})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/filters/match-ids", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newFiltersRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// Slice nil → JSON [] (jamais null) pour ne pas casser .length/.map côté front.
	if got := w.Body.String(); !bytes.Contains([]byte(got), []byte(`"match_ids":[]`)) {
		t.Errorf("expected match_ids:[] in body, got %s", got)
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
