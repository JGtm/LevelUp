// Package handlers_test — home_test.go : tests unitaires HomeHandler.
package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// mockHomeService implémente port.HomeService pour les tests.
type mockHomeService struct {
	page       *domain.HomePageResponse
	pageErr    error
	battlePass domain.BattlePassResponse
	challenges domain.ChallengesResponse
}

func (m *mockHomeService) GetHomePage(_ context.Context, _ string) (*domain.HomePageResponse, error) {
	return m.page, m.pageErr
}

func (m *mockHomeService) GetBattlePass(_ context.Context) domain.BattlePassResponse {
	return m.battlePass
}

func (m *mockHomeService) GetChallenges(_ context.Context) domain.ChallengesResponse {
	return m.challenges
}

func newHomeRouter(factory handlers.ContextFactory[port.HomeService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewHomeHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Get("/pages/home", h.GetHomePage)
		r.Get("/battlepass", h.GetBattlePass)
		r.Get("/challenges", h.GetChallenges)
	})
	return r
}

func TestHomeHandler_GetHomePage_OK(t *testing.T) {
	mock := &mockHomeService{page: &domain.HomePageResponse{}}
	factory := func(_ context.Context, slug string) (port.HomeService, string, string, error) {
		if slug != testPlayerSlug {
			return nil, "", "", errors.New("player_not_found")
		}
		return mock, "xuid-1", "TestPlayer", nil
	}
	r := newHomeRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/home", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHomeHandler_GetHomePage_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.HomeService, string, string, error) {
		return nil, "", "", errors.New("player_not_found")
	}
	r := newHomeRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/unknown/pages/home", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHomeHandler_GetHomePage_ServiceError(t *testing.T) {
	mock := &mockHomeService{pageErr: errors.New("db_error")}
	factory := func(_ context.Context, _ string) (port.HomeService, string, string, error) {
		return mock, "xuid", "gt", nil
	}
	r := newHomeRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/home", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestHomeHandler_GetBattlePass_OK(t *testing.T) {
	mock := &mockHomeService{battlePass: domain.BattlePassResponse{}}
	factory := func(_ context.Context, _ string) (port.HomeService, string, string, error) {
		return mock, "xuid", "gt", nil
	}
	r := newHomeRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/battlepass", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHomeHandler_GetChallenges_OK(t *testing.T) {
	mock := &mockHomeService{challenges: domain.ChallengesResponse{}}
	factory := func(_ context.Context, _ string) (port.HomeService, string, string, error) {
		return mock, "xuid", "gt", nil
	}
	r := newHomeRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/challenges", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
