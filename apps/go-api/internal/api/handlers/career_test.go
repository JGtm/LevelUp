// Package handlers — career_test.go : test unitaire avec mock service (Sprint 37).
//
// Démontre que le pattern DI permet de tester les handlers sans DB.
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

// mockCareerService implémente port.CareerService pour les tests.
type mockCareerService struct {
	careerPage   domain.CareerPageResponse
	topMatches   domain.CareerTopMatchesResponse
	encounters   domain.CareerEncountersResponse
	careerErr    error
	topErr       error
	encounterErr error
}

func (m *mockCareerService) GetCareerPage(_ context.Context) (domain.CareerPageResponse, error) {
	return m.careerPage, m.careerErr
}

func (m *mockCareerService) GetTopMatches(_ context.Context) (domain.CareerTopMatchesResponse, error) {
	return m.topMatches, m.topErr
}

func (m *mockCareerService) GetEncounters(_ context.Context) (domain.CareerEncountersResponse, error) {
	return m.encounters, m.encounterErr
}

// newTestRouter construit un routeur chi avec le handler career câblé.
func newTestRouter(factory handlers.ServiceFactory[port.CareerService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewCareerHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Get("/pages/career", h.GetCareer)
		r.Get("/pages/career/top-matches", h.GetTopMatches)
		r.Get("/pages/career/encounters", h.GetEncounters)
	})
	return r
}

func TestCareerHandler_GetCareer_OK(t *testing.T) {
	mock := &mockCareerService{
		careerPage: domain.CareerPageResponse{
			Summary: domain.CareerRankSummary{CurrentRankName: "Diamond 1"},
		},
	}
	factory := func(_ context.Context, slug string) (port.CareerService, error) {
		if slug != "test-player" {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}

	r := newTestRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/career", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp domain.CareerPageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Summary.CurrentRankName != "Diamond 1" {
		t.Errorf("expected rank 'Diamond 1', got %q", resp.Summary.CurrentRankName)
	}
}

func TestCareerHandler_GetCareer_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.CareerService, error) {
		return nil, errors.New("unknown player slug")
	}

	r := newTestRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/unknown/pages/career", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCareerHandler_GetCareer_ServiceError(t *testing.T) {
	mock := &mockCareerService{
		careerErr: errors.New("db timeout"),
	}
	factory := func(_ context.Context, _ string) (port.CareerService, error) {
		return mock, nil
	}

	r := newTestRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/p1/pages/career", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}
