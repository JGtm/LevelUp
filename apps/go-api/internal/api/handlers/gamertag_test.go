// Package handlers — gamertag_test.go : tests unitaires GamertagHandler avec mock service.
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

type mockGamertagSearchService struct {
	results []domain.GamertagSearchResult
	err     error
}

func (m *mockGamertagSearchService) Search(_ context.Context, _ string) ([]domain.GamertagSearchResult, error) {
	return m.results, m.err
}

func newGamertagRouter(svc port.GamertagSearchService) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewGamertagHandler(svc)
	r.Get("/directory/gamertags/search", h.Search)
	return r
}

func TestGamertagHandler_Search_OK(t *testing.T) {
	expected := []domain.GamertagSearchResult{{Gamertag: testGamertag, XUID: "123"}}
	svc := &mockGamertagSearchService{results: expected}
	r := newGamertagRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/directory/gamertags/search?q=test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp domain.GamertagSearchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Gamertag != testGamertag {
		t.Errorf("unexpected items: %+v", resp.Items)
	}
}

func TestGamertagHandler_Search_EmptyQuery(t *testing.T) {
	svc := &mockGamertagSearchService{} // ne doit pas être appelé
	r := newGamertagRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/directory/gamertags/search?q=", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp domain.GamertagSearchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected empty items, got %v", resp.Items)
	}
}

func TestGamertagHandler_Search_ServiceError(t *testing.T) {
	svc := &mockGamertagSearchService{err: errors.New("db error")}
	r := newGamertagRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/directory/gamertags/search?q=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
