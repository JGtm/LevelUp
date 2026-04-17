// Package handlers_test — teammates_test.go : tests unitaires TeammatesHandler.
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

// mockTeammatesService implémente port.TeammatesService.
type mockTeammatesService struct {
	page    domain.TeammatesPageResponse
	pageErr error
}

func (m *mockTeammatesService) GetPage(_ context.Context, _ string, _ domain.TeammatesQueryRequest) (domain.TeammatesPageResponse, error) {
	return m.page, m.pageErr
}

func newTeammatesRouter(factory handlers.ContextFactory[port.TeammatesService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewTeammatesHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Post("/pages/teammates", h.GetPage)
	})
	return r
}

func TestTeammatesHandler_OK(t *testing.T) {
	mock := &mockTeammatesService{page: domain.TeammatesPageResponse{}}
	factory := func(_ context.Context, slug string) (port.TeammatesService, string, string, error) {
		if slug != testPlayerSlug {
			return nil, "", "", errors.New("player_not_found")
		}
		return mock, "xuid-1", "TestPlayer", nil
	}
	r := newTeammatesRouter(factory)
	body, _ := json.Marshal(domain.TeammatesQueryRequest{})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/teammates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTeammatesHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.TeammatesService, string, string, error) {
		return nil, "", "", errors.New("player_not_found")
	}
	r := newTeammatesRouter(factory)
	body, _ := json.Marshal(domain.TeammatesQueryRequest{})
	req := httptest.NewRequest(http.MethodPost, "/players/unknown/pages/teammates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestTeammatesHandler_ServiceError(t *testing.T) {
	mock := &mockTeammatesService{pageErr: errors.New("db_error")}
	factory := func(_ context.Context, _ string) (port.TeammatesService, string, string, error) {
		return mock, "xuid", "gt", nil
	}
	r := newTeammatesRouter(factory)
	body, _ := json.Marshal(domain.TeammatesQueryRequest{})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/teammates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTeammatesHandler_InvalidBody(t *testing.T) {
	mock := &mockTeammatesService{}
	factory := func(_ context.Context, _ string) (port.TeammatesService, string, string, error) {
		return mock, "xuid", "gt", nil
	}
	r := newTeammatesRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/teammates",
		bytes.NewReader([]byte("{bad json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
