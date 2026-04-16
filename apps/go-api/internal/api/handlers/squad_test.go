// Package handlers — squad_test.go : tests unitaires SquadHandler avec mock service.
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

type mockSquadService struct {
	squadPage     *domain.SquadPageResponse
	synthesisPage *domain.SynthesisPageResponse
	squadErr      error
	synthesisErr  error
}

func (m *mockSquadService) GetSquadPage(_ context.Context, _, _, _ string) (*domain.SquadPageResponse, error) {
	return m.squadPage, m.squadErr
}

func (m *mockSquadService) GetSynthesisPage(_ context.Context, _ string) (*domain.SynthesisPageResponse, error) {
	return m.synthesisPage, m.synthesisErr
}

func newSquadRouter(factory handlers.ContextFactory[port.SquadService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewSquadHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Get("/pages/squad", h.GetSquadPage)
		r.Post("/pages/synthesis", h.GetSynthesisPage)
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

func TestSquadHandler_GetSynthesisPage_OK(t *testing.T) {
	expected := &domain.SynthesisPageResponse{}
	svc := &mockSquadService{synthesisPage: expected}
	r := newSquadRouter(makeSquadFactory(svc))

	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/synthesis", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp domain.SynthesisPageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}

func TestSquadHandler_GetSynthesisPage_ServiceError(t *testing.T) {
	svc := &mockSquadService{synthesisErr: errors.New("db error")}
	r := newSquadRouter(makeSquadFactory(svc))

	req := httptest.NewRequest(http.MethodPost, "/players/p/pages/synthesis", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
