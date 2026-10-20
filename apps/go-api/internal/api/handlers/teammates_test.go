// Package handlers_test — teammates_test.go : tests unitaires TeammatesHandler.
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// mockTeammatesService implémente port.TeammatesService. pageErr fait échouer les DEUX
// routes (page et sessions) ; les champs got* retiennent l'appel de CompositionSessions.
type mockTeammatesService struct {
	page    domain.TeammatesPageResponse
	pageErr error

	sessions     []domain.CompositionSessionEntry
	latest       string
	gotXUID      string
	gotTeammates []string
	gotExact     bool
}

func (m *mockTeammatesService) GetPage(_ context.Context, _ string, _ domain.TeammatesQueryRequest) (domain.TeammatesPageResponse, error) {
	return m.page, m.pageErr
}

func (m *mockTeammatesService) CompositionSessions(_ context.Context, xuid string, teammates []string, exact bool) ([]domain.CompositionSessionEntry, string, error) {
	m.gotXUID, m.gotTeammates, m.gotExact = xuid, teammates, exact
	return m.sessions, m.latest, m.pageErr
}

func newTeammatesRouter(factory handlers.ContextFactory[port.TeammatesService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewTeammatesHandler(factory)
	r.Route("/players/{player_slug}", func(sub chi.Router) {
		h.Mount(sub)
	})
	return r
}

func TestTeammatesHandler_OK(t *testing.T) {
	mock := &mockTeammatesService{page: domain.TeammatesPageResponse{}}
	factory := func(_ context.Context, slug string) (port.TeammatesService, string, string, error) {
		if slug != testPlayerSlug {
			return nil, "", "", errors.New("player_not_found")
		}
		return mock, testXUID1, testGamertag, nil
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
		return mock, testXUID, "gt", nil
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

// TestTeammatesHandler_CapabilityAbsente_503 : un titre sans la capability rend un 503 propre
// `capability_not_supported` (MapCapabilityError), jamais le 500 de mapServiceError — comme la
// route légère des sessions (lot perf L8, 2026-09-23).
func TestTeammatesHandler_CapabilityAbsente_503(t *testing.T) {
	mock := &mockTeammatesService{pageErr: fmt.Errorf("historique: %w", games.ErrCapabilityNotSupported)}
	r := newTeammatesRouter(func(_ context.Context, _ string) (port.TeammatesService, string, string, error) {
		return mock, testXUID, testGamertag, nil
	})
	body, _ := json.Marshal(domain.TeammatesQueryRequest{})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/teammates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("statut %d, attendu 503 : %s", w.Code, w.Body.String())
	}
	if code := errorCode(t, w); code != "capability_not_supported" {
		t.Errorf("code %q, attendu capability_not_supported", code)
	}
}

func TestTeammatesHandler_InvalidBody(t *testing.T) {
	mock := &mockTeammatesService{}
	factory := func(_ context.Context, _ string) (port.TeammatesService, string, string, error) {
		return mock, testXUID, "gt", nil
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
