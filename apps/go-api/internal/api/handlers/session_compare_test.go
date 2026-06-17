// Package handlers_test — session_compare_test.go : tests unitaires SessionCompareHandler.
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

// mockSessionCompareService implémente port.SessionCompareService.
type mockSessionCompareService struct {
	resp domain.SessionCompareResponse
	err  error
}

func (m *mockSessionCompareService) Compare(_ context.Context, _ domain.SessionCompareRequest) (domain.SessionCompareResponse, error) {
	return m.resp, m.err
}

func newSessionCompareRouter(factory handlers.ServiceFactory[port.SessionCompareService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewSessionCompareHandler(factory)
	r.Route("/players/{player_slug}", func(sub chi.Router) {
		h.Mount(sub)
	})
	return r
}

func TestSessionCompareHandler_OK(t *testing.T) {
	mock := &mockSessionCompareService{resp: domain.SessionCompareResponse{}}
	factory := func(_ context.Context, slug string) (port.SessionCompareService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newSessionCompareRouter(factory)
	s1, s2 := "s1", "s2"
	body, _ := json.Marshal(domain.SessionCompareRequest{SessionA: &s1, SessionB: &s2})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/session-compare", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSessionCompareHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.SessionCompareService, error) {
		return nil, errors.New("player_not_found")
	}
	r := newSessionCompareRouter(factory)
	s1, s2 := "s1", "s2"
	body, _ := json.Marshal(domain.SessionCompareRequest{SessionA: &s1, SessionB: &s2})
	req := httptest.NewRequest(http.MethodPost, "/players/unknown/pages/session-compare", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSessionCompareHandler_ServiceError(t *testing.T) {
	mock := &mockSessionCompareService{err: errors.New("db_error")}
	factory := func(_ context.Context, _ string) (port.SessionCompareService, error) {
		return mock, nil
	}
	r := newSessionCompareRouter(factory)
	s1, s2 := "s1", "s2"
	body, _ := json.Marshal(domain.SessionCompareRequest{SessionA: &s1, SessionB: &s2})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/session-compare", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestSessionCompareHandler_InvalidBody(t *testing.T) {
	mock := &mockSessionCompareService{}
	factory := func(_ context.Context, _ string) (port.SessionCompareService, error) {
		return mock, nil
	}
	r := newSessionCompareRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/session-compare",
		bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
