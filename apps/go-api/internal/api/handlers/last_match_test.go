// Package handlers_test — last_match_test.go : tests unitaires LastMatchHandler.
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

// mockLastMatchService implémente port.LastMatchService.
type mockLastMatchService struct {
	resp domain.LastMatchResolveResponse
	err  error
}

func (m *mockLastMatchService) Resolve(_ context.Context, _ domain.LastMatchResolveRequest) (domain.LastMatchResolveResponse, error) {
	return m.resp, m.err
}

func newLastMatchRouter(factory handlers.ServiceFactory[port.LastMatchService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewLastMatchHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Post("/pages/last-match/resolve", h.Resolve)
	})
	return r
}

func TestLastMatchHandler_OK(t *testing.T) {
	mock := &mockLastMatchService{resp: domain.LastMatchResolveResponse{MatchID: "m1"}}
	factory := func(_ context.Context, slug string) (port.LastMatchService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newLastMatchRouter(factory)
	body, _ := json.Marshal(domain.LastMatchResolveRequest{})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/last-match/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLastMatchHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.LastMatchService, error) {
		return nil, errors.New("player_not_found")
	}
	r := newLastMatchRouter(factory)
	body, _ := json.Marshal(domain.LastMatchResolveRequest{})
	req := httptest.NewRequest(http.MethodPost, "/players/unknown/pages/last-match/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestLastMatchHandler_ServiceError(t *testing.T) {
	mock := &mockLastMatchService{err: errors.New("db_error")}
	factory := func(_ context.Context, _ string) (port.LastMatchService, error) {
		return mock, nil
	}
	r := newLastMatchRouter(factory)
	body, _ := json.Marshal(domain.LastMatchResolveRequest{})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/last-match/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestLastMatchHandler_InvalidBody(t *testing.T) {
	mock := &mockLastMatchService{}
	factory := func(_ context.Context, _ string) (port.LastMatchService, error) {
		return mock, nil
	}
	r := newLastMatchRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/last-match/resolve",
		bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
