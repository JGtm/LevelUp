// Package handlers — stats_test.go : tests unitaires StatsHandler avec mock service.
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

type mockStatsService struct {
	resp domain.StatsPageResponse
	err  error
}

func (m *mockStatsService) GetPage(_ context.Context, _ domain.StatsQueryRequest) (domain.StatsPageResponse, error) {
	return m.resp, m.err
}

func newStatsRouter(factory handlers.ServiceFactory[port.StatsService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewStatsHandler(factory)
	r.Route("/players/{player_slug}", func(sub chi.Router) {
		h.Mount(sub)
	})
	return r
}

func TestStatsHandler_OK(t *testing.T) {
	factory := func(_ context.Context, slug string) (port.StatsService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return &mockStatsService{resp: domain.StatsPageResponse{}}, nil
	}

	body, _ := json.Marshal(domain.StatsQueryRequest{Tab: "win_loss"})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/stats/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newStatsRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStatsHandler_DefaultTab(t *testing.T) {
	var capturedReq domain.StatsQueryRequest
	svc := &struct{ mockStatsService }{} // embed, ignore
	_ = svc

	called := false
	factory := func(_ context.Context, _ string) (port.StatsService, error) {
		return &captureStatsService{onGetPage: func(req domain.StatsQueryRequest) {
			capturedReq = req
			called = true
		}}, nil
	}

	body, _ := json.Marshal(domain.StatsQueryRequest{}) // Tab vide
	req := httptest.NewRequest(http.MethodPost, "/players/p/pages/stats/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newStatsRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !called {
		t.Fatal("service.GetPage jamais appelé")
	}
	if capturedReq.Tab != "win_loss" {
		t.Errorf("default tab: got %q, want %q", capturedReq.Tab, "win_loss")
	}
}

func TestStatsHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.StatsService, error) {
		return nil, errors.New("not_found")
	}

	body, _ := json.Marshal(domain.StatsQueryRequest{Tab: "win_loss"})
	req := httptest.NewRequest(http.MethodPost, "/players/unk/pages/stats/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newStatsRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// captureStatsService permet d'inspecter la requête passée à GetPage.
type captureStatsService struct {
	onGetPage func(domain.StatsQueryRequest)
}

func (s *captureStatsService) GetPage(_ context.Context, req domain.StatsQueryRequest) (domain.StatsPageResponse, error) {
	if s.onGetPage != nil {
		s.onGetPage(req)
	}
	return domain.StatsPageResponse{}, nil
}
