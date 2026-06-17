// Package handlers — achievements_test.go : test handler avec mock service.
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

// mockAchievementsService implémente port.AchievementsService.
type mockAchievementsService struct {
	resp domain.AchievementsPageResponse
	err  error
}

func (m *mockAchievementsService) GetAchievementsPage(_ context.Context) (domain.AchievementsPageResponse, error) {
	return m.resp, m.err
}

func newAchievementsRouter(factory handlers.ServiceFactory[port.AchievementsService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewAchievementsHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})
	return r
}

func TestAchievementsHandler_OK(t *testing.T) {
	mock := &mockAchievementsService{
		resp: domain.AchievementsPageResponse{
			Summary: domain.AchievementsSummary{
				TotalCount:       50,
				UnlockedCount:    20,
				TotalGamerscore:  1000,
				EarnedGamerscore: 400,
				CompletionPct:    40.0,
			},
			Achievements: []domain.AchievementEntry{
				{AchievementID: "a", NameEN: "First", NameFR: "Premier", Gamerscore: 10, Unlocked: true},
				{AchievementID: "b", NameEN: "Second", NameFR: "Second", Gamerscore: 20, Unlocked: false},
			},
		},
	}
	factory := func(_ context.Context, slug string) (port.AchievementsService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}

	r := newAchievementsRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/achievements", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", w.Code, w.Body.String())
	}

	var resp domain.AchievementsPageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Summary.TotalCount != 50 || resp.Summary.UnlockedCount != 20 {
		t.Errorf("summary count incorrect: %+v", resp.Summary)
	}
	if len(resp.Achievements) != 2 {
		t.Errorf("attendu 2 achievements, obtenu %d", len(resp.Achievements))
	}
}

func TestAchievementsHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.AchievementsService, error) {
		return nil, errors.New("unknown player slug")
	}

	r := newAchievementsRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/unknown/pages/achievements", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("attendu 404, obtenu %d: %s", w.Code, w.Body.String())
	}
}

func TestAchievementsHandler_ServiceError(t *testing.T) {
	mock := &mockAchievementsService{err: errors.New("boom")}
	factory := func(_ context.Context, _ string) (port.AchievementsService, error) {
		return mock, nil
	}

	r := newAchievementsRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/achievements", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("attendu 500, obtenu %d: %s", w.Code, w.Body.String())
	}
}
