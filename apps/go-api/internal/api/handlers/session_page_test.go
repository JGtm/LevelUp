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

type mockSessionPageService struct {
	resp domain.SessionPageResponse
	err  error
}

func (m *mockSessionPageService) GetPage(_ context.Context, _ domain.SessionPageRequest) (domain.SessionPageResponse, error) {
	return m.resp, m.err
}

func newSessionPageRouter(factory handlers.ServiceFactory[port.SessionPageService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewSessionPageHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Post("/pages/sessions/detail", h.GetPage)
	})
	return r
}

func TestSessionPageHandler_OK(t *testing.T) {
	mock := &mockSessionPageService{resp: domain.SessionPageResponse{AvailableSessions: []string{"S1"}}}
	factory := func(_ context.Context, slug string) (port.SessionPageService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newSessionPageRouter(factory)
	body, _ := json.Marshal(domain.SessionPageRequest{EnableCompare: true})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/sessions/detail", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSessionPageHandler_OKWithResolvedSession(t *testing.T) {
	sessionLabel := "S1"
	mock := &mockSessionPageService{resp: domain.SessionPageResponse{
		CurrentSession:    &domain.SessionCompareEntry{SessionLabel: sessionLabel},
		AvailableSessions: []string{sessionLabel},
	}}
	factory := func(_ context.Context, _ string) (port.SessionPageService, error) {
		return mock, nil
	}
	r := newSessionPageRouter(factory)
	body, _ := json.Marshal(domain.SessionPageRequest{SessionLabel: &sessionLabel})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/sessions/detail", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSessionPageHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.SessionPageService, error) {
		return nil, errors.New("player_not_found")
	}
	r := newSessionPageRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/unknown/pages/sessions/detail", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSessionPageHandler_ServiceError(t *testing.T) {
	mock := &mockSessionPageService{err: errors.New("db_error")}
	factory := func(_ context.Context, _ string) (port.SessionPageService, error) {
		return mock, nil
	}
	r := newSessionPageRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/sessions/detail", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestSessionPageHandler_InvalidBody(t *testing.T) {
	mock := &mockSessionPageService{}
	factory := func(_ context.Context, _ string) (port.SessionPageService, error) {
		return mock, nil
	}
	r := newSessionPageRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/sessions/detail", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSessionPageHandler_InvalidRequest(t *testing.T) {
	mock := &mockSessionPageService{}
	factory := func(_ context.Context, _ string) (port.SessionPageService, error) {
		return mock, nil
	}
	r := newSessionPageRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/sessions/detail", bytes.NewReader([]byte(`{"session_label":""}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
