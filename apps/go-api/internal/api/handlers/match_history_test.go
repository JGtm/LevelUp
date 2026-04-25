// Package handlers_test — match_history_test.go : tests unitaires MatchHistoryHandler.
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

// mockMatchHistoryService implémente port.MatchHistoryService.
type mockMatchHistoryService struct {
	page    domain.MatchHistoryPageResponse
	pageErr error
	csvRows []domain.MatchHistoryRow
	csvErr  error
}

func (m *mockMatchHistoryService) GetPage(_ context.Context, _ domain.MatchHistoryQueryRequest) (domain.MatchHistoryPageResponse, error) {
	return m.page, m.pageErr
}

func (m *mockMatchHistoryService) ExportCSV(_ context.Context, _ domain.MatchHistoryQueryRequest) ([]domain.MatchHistoryRow, error) {
	return m.csvRows, m.csvErr
}

func newMatchHistoryRouter(factory handlers.ContextFactory[port.MatchHistoryService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewMatchHistoryHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Post("/pages/match-history/query", h.Query)
		r.Get("/pages/match-history/export", h.Export)
	})
	return r
}

func TestMatchHistoryHandler_Query_OK(t *testing.T) {
	mock := &mockMatchHistoryService{
		page: domain.MatchHistoryPageResponse{},
	}
	factory := func(_ context.Context, slug string) (port.MatchHistoryService, string, string, error) {
		if slug != testPlayerSlug {
			return nil, "", "", errors.New("player_not_found")
		}
		return mock, testXUID1, testGamertag, nil
	}
	r := newMatchHistoryRouter(factory)
	body, _ := json.Marshal(domain.MatchHistoryQueryRequest{Pagination: domain.PaginationRequest{Page: 1, PageSize: 20}})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/match-history/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMatchHistoryHandler_Query_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.MatchHistoryService, string, string, error) {
		return nil, "", "", errors.New("player_not_found")
	}
	r := newMatchHistoryRouter(factory)
	body, _ := json.Marshal(domain.MatchHistoryQueryRequest{})
	req := httptest.NewRequest(http.MethodPost, "/players/unknown/pages/match-history/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestMatchHistoryHandler_Query_ServiceError(t *testing.T) {
	mock := &mockMatchHistoryService{pageErr: errors.New("db_error")}
	factory := func(_ context.Context, _ string) (port.MatchHistoryService, string, string, error) {
		return mock, testXUID, "gt", nil
	}
	r := newMatchHistoryRouter(factory)
	body, _ := json.Marshal(domain.MatchHistoryQueryRequest{})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/match-history/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestMatchHistoryHandler_Export_MissingToken(t *testing.T) {
	mock := &mockMatchHistoryService{}
	factory := func(_ context.Context, slug string) (port.MatchHistoryService, string, string, error) {
		if slug != testPlayerSlug {
			return nil, "", "", errors.New("player_not_found")
		}
		return mock, testXUID, "gt", nil
	}
	r := newMatchHistoryRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/match-history/export", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
