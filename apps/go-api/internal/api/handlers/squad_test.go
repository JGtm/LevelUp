// Package handlers — squad_test.go : tests unitaires SquadHandler avec mock service.
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

type mockSquadService struct {
	squadPage *domain.SquadPageResponse
	squadErr  error
}

func (m *mockSquadService) GetSquadPage(_ context.Context, _, _, _ string) (*domain.SquadPageResponse, error) {
	return m.squadPage, m.squadErr
}

// GetSynthesisPage satisfait port.SquadService (la route POST /pages/synthesis
// est servie par SynthesisHandler depuis Sprint 55 D1 — ce mock n'a plus à la
// piloter, mais l'interface l'exige toujours).
func (m *mockSquadService) GetSynthesisPage(_ context.Context, _ string) (*domain.SynthesisPageResponse, error) {
	return nil, nil
}

func newSquadRouter(factory handlers.ContextFactory[port.SquadService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewSquadHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})
	return r
}

func makeSquadFactory(svc port.SquadService) handlers.ContextFactory[port.SquadService] {
	return func(_ context.Context, slug string) (port.SquadService, string, string, error) {
		if slug == "unknown" {
			return nil, "", "", errors.New("player_not_found")
		}
		return svc, "xuid-test", slug, nil
	}
}

func TestSquadHandler_GetSquadPage_OK(t *testing.T) {
	expected := &domain.SquadPageResponse{}
	svc := &mockSquadService{squadPage: expected}
	r := newSquadRouter(makeSquadFactory(svc))

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/squad", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSquadHandler_GetSquadPage_PlayerNotFound(t *testing.T) {
	r := newSquadRouter(makeSquadFactory(&mockSquadService{}))

	req := httptest.NewRequest(http.MethodGet, "/players/unknown/pages/squad", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
