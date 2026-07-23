// Package handlers_test — medals_test.go : tests du MedalsHandler.
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// mockMedalsService implémente port.MedalsService pour les tests.
type mockMedalsService struct {
	page *domain.MedalsPageResponse
	err  error
}

func (m *mockMedalsService) GetMedalsPage(_ context.Context, _ string) (*domain.MedalsPageResponse, error) {
	return m.page, m.err
}

func newMedalsRouter(factory handlers.ContextFactory[port.MedalsService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewMedalsHandler(factory)
	r.Route("/players/{player_slug}", func(sub chi.Router) {
		h.Mount(sub)
	})
	return r
}

func medalsFactory(svc port.MedalsService) handlers.ContextFactory[port.MedalsService] {
	return func(_ context.Context, slug string) (port.MedalsService, string, string, error) {
		if slug != testPlayerSlug {
			return nil, "", "", errors.New("player_not_found")
		}
		return svc, testXUID1, testGamertag, nil
	}
}

// TestMedalsHandler_GetMedals_OK vérifie 200 + contrat non-nil-slice : les champs
// medals/categories sérialisent en `[]` (jamais null) pour un consommateur typé.
func TestMedalsHandler_GetMedals_OK(t *testing.T) {
	mock := &mockMedalsService{page: &domain.MedalsPageResponse{
		Medals:     []domain.MedalSummaryItem{},
		Categories: []domain.MedalCategoryGroup{},
	}}
	r := newMedalsRouter(medalsFactory(mock))
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/medals", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Contrat : `null` interdit pour les slices sans omitempty (crash front).
	if strings.Contains(w.Body.String(), `"medals":null`) || strings.Contains(w.Body.String(), `"categories":null`) {
		t.Fatalf("slices sérialisées en null : %s", w.Body.String())
	}
	var resp domain.MedalsPageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Medals == nil || resp.Categories == nil {
		t.Fatalf("medals/categories nil après round-trip : %+v", resp)
	}
}

func TestMedalsHandler_GetMedals_PlayerNotFound(t *testing.T) {
	r := newMedalsRouter(medalsFactory(&mockMedalsService{}))
	req := httptest.NewRequest(http.MethodPost, "/players/unknown/pages/medals", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestMedalsHandler_GetMedals_ServiceError(t *testing.T) {
	mock := &mockMedalsService{err: errors.New("db_error")}
	r := newMedalsRouter(medalsFactory(mock))
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/medals", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestMedalsHandler_GetMedals_InvalidBody(t *testing.T) {
	mock := &mockMedalsService{page: &domain.MedalsPageResponse{
		Medals: []domain.MedalSummaryItem{}, Categories: []domain.MedalCategoryGroup{},
	}}
	r := newMedalsRouter(medalsFactory(mock))
	body := strings.NewReader("{invalid json")
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/medals", body)
	req.ContentLength = int64(body.Len())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
