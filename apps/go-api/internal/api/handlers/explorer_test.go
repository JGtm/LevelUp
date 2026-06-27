// Package handlers_test — explorer_test.go : tests unitaires ExplorerHandler.
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

// mockExplorerService implémente port.ExplorerService.
type mockExplorerService struct {
	resp domain.ExplorerPlayerQueryResponse
	err  error
}

func (m *mockExplorerService) GetCommonMatches(_ context.Context, _, _ string, _ int) (domain.ExplorerPlayerQueryResponse, error) {
	return m.resp, m.err // gamertag/xuid/page ignorés dans le mock
}

// mockMatchHistoryForExplorer implémente port.MatchHistoryService pour l'explorer.
type mockMatchHistoryForExplorer struct {
	page    domain.MatchHistoryPageResponse
	pageErr error
}

func (m *mockMatchHistoryForExplorer) GetPage(_ context.Context, _ domain.MatchHistoryQueryRequest) (domain.MatchHistoryPageResponse, error) {
	return m.page, m.pageErr
}

func (m *mockMatchHistoryForExplorer) ExportCSV(_ context.Context, _ domain.MatchHistoryQueryRequest) ([]domain.MatchHistoryRow, error) {
	return nil, nil
}

func newExplorerRouter(
	explorerF handlers.ExplorerAuthFactory,
	matchHistF handlers.ContextFactory[port.MatchHistoryService],
) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewExplorerHandler(explorerF, matchHistF)
	r.Route("/players/{player_slug}", func(sub chi.Router) {
		h.Mount(sub)
	})
	return r
}

func TestExplorerHandler_QueryPlayer_OK(t *testing.T) {
	mock := &mockExplorerService{resp: domain.ExplorerPlayerQueryResponse{}}
	explorerF := func(ctx context.Context, slug string) (port.ExplorerService, context.Context, string, string, error) {
		if slug != testPlayerSlug {
			return nil, ctx, "", "", errors.New("not found")
		}
		return mock, ctx, testXUID, "gt", nil
	}
	matchHistF := func(_ context.Context, _ string) (port.MatchHistoryService, string, string, error) {
		return &mockMatchHistoryForExplorer{}, testXUID, "gt", nil
	}
	r := newExplorerRouter(explorerF, matchHistF)

	body, _ := json.Marshal(domain.ExplorerPlayerQueryRequest{TargetGamertag: "opponent", Page: 1})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/explorer/player-query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExplorerHandler_QueryPlayer_PlayerNotFound(t *testing.T) {
	explorerF := func(ctx context.Context, _ string) (port.ExplorerService, context.Context, string, string, error) {
		return nil, ctx, "", "", errors.New("player_not_found")
	}
	matchHistF := func(_ context.Context, _ string) (port.MatchHistoryService, string, string, error) {
		return &mockMatchHistoryForExplorer{}, testXUID, "gt", nil
	}
	r := newExplorerRouter(explorerF, matchHistF)

	body, _ := json.Marshal(domain.ExplorerPlayerQueryRequest{TargetGamertag: "opp"})
	req := httptest.NewRequest(http.MethodPost, "/players/unknown/pages/explorer/player-query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestExplorerHandler_QueryPlayer_MissingGamertag(t *testing.T) {
	mock := &mockExplorerService{}
	explorerF := func(ctx context.Context, _ string) (port.ExplorerService, context.Context, string, string, error) {
		return mock, ctx, testXUID, "gt", nil
	}
	matchHistF := func(_ context.Context, _ string) (port.MatchHistoryService, string, string, error) {
		return &mockMatchHistoryForExplorer{}, testXUID, "gt", nil
	}
	r := newExplorerRouter(explorerF, matchHistF)

	body, _ := json.Marshal(domain.ExplorerPlayerQueryRequest{TargetGamertag: ""})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/explorer/player-query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestExplorerHandler_QueryPlayer_ServiceError(t *testing.T) {
	mock := &mockExplorerService{err: errors.New("db_error")}
	explorerF := func(ctx context.Context, _ string) (port.ExplorerService, context.Context, string, string, error) {
		return mock, ctx, testXUID, "gt", nil
	}
	matchHistF := func(_ context.Context, _ string) (port.MatchHistoryService, string, string, error) {
		return &mockMatchHistoryForExplorer{}, testXUID, "gt", nil
	}
	r := newExplorerRouter(explorerF, matchHistF)

	body, _ := json.Marshal(domain.ExplorerPlayerQueryRequest{TargetGamertag: "opp"})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/explorer/player-query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
