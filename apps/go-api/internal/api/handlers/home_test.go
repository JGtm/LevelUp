// Package handlers_test — home_test.go : tests unitaires HomeHandler.
package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
)

// mockHomeService implémente port.HomeService pour les tests.
type mockHomeService struct {
	page       *domain.HomePageResponse
	pageErr    error
	battlePass domain.BattlePassResponse
	challenges domain.ChallengesResponse
	pageLocale string
}

func (m *mockHomeService) GetHomePage(_ context.Context, _ string, locale string) (*domain.HomePageResponse, error) {
	m.pageLocale = locale
	return m.page, m.pageErr
}

func (m *mockHomeService) GetBattlePass(_ context.Context) domain.BattlePassResponse {
	return m.battlePass
}

func (m *mockHomeService) GetChallenges(_ context.Context) domain.ChallengesResponse {
	return m.challenges
}

func (m *mockHomeService) RefreshTrack(_ context.Context, _ string) {}

func newHomeRouter(factory handlers.HomeAuthFactory, settingsStore *settings_platform.Store) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewHomeHandler(factory, settingsStore)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r) // 3 routes migrées Huma (pages/home ETag, battlepass, challenges)
	})
	return r
}

func TestHomeHandler_GetHomePage_OK(t *testing.T) {
	mock := &mockHomeService{page: &domain.HomePageResponse{}}
	factory := func(ctx context.Context, slug string) (port.HomeService, context.Context, string, string, error) {
		if slug != testPlayerSlug {
			return nil, ctx, "", "", errors.New("player_not_found")
		}
		return mock, ctx, testXUID1, testGamertag, nil
	}
	r := newHomeRouter(factory, nil)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/home", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if mock.pageLocale != "fr" {
		t.Fatalf("expected default locale fr, got %q", mock.pageLocale)
	}
}

func TestHomeHandler_GetHomePage_PlayerNotFound(t *testing.T) {
	factory := func(ctx context.Context, _ string) (port.HomeService, context.Context, string, string, error) {
		return nil, ctx, "", "", errors.New("player_not_found")
	}
	r := newHomeRouter(factory, nil)
	req := httptest.NewRequest(http.MethodGet, "/players/unknown/pages/home", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHomeHandler_GetHomePage_ServiceError(t *testing.T) {
	mock := &mockHomeService{pageErr: errors.New("db_error")}
	factory := func(ctx context.Context, _ string) (port.HomeService, context.Context, string, string, error) {
		return mock, ctx, testXUID, "gt", nil
	}
	r := newHomeRouter(factory, nil)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/home", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// TestHomeHandler_GetHomePage_DBClosed_Returns503 — Phase 5 ART.
// Quand la player DB est fermée/invalidée (signature observée en prod
// 2026-05-24 20:43+ : "sql: database is closed"), le handler doit
// renvoyer 503 + Retry-After plutôt qu'un 500 sec — pour que le client
// retente après le Reopen() côté provider.
func TestHomeHandler_GetHomePage_DBClosed_Returns503(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{name: "sql database is closed", err: errors.New("PlayerMatchesRepo.Load: LoadPlayerMatchEnrichments: sql: database is closed")},
		{name: "database has been invalidated (FATAL)", err: errors.New("FATAL Error: Failed: database has been invalidated because of a previous fatal error")},
		{name: "Failed to delete all rows from index (ART)", err: errors.New("Invalid Input Error: Failed to delete all rows from index. Only deleted 0 out of 1 rows.")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockHomeService{pageErr: tc.err}
			factory := func(ctx context.Context, _ string) (port.HomeService, context.Context, string, string, error) {
				return mock, ctx, testXUID, "gt", nil
			}
			r := newHomeRouter(factory, nil)
			req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/home", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("status = %d, want 503", w.Code)
			}
			if retryAfter := w.Header().Get("Retry-After"); retryAfter == "" {
				t.Error("Retry-After header absent (attendu pour 503 recovery)")
			}
		})
	}
}

func TestHomeHandler_GetBattlePass_OK(t *testing.T) {
	mock := &mockHomeService{battlePass: domain.BattlePassResponse{}}
	factory := func(ctx context.Context, _ string) (port.HomeService, context.Context, string, string, error) {
		return mock, ctx, testXUID, "gt", nil
	}
	r := newHomeRouter(factory, nil)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/battlepass", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHomeHandler_GetHomePage_UsesSettingsLanguage(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"lang":"en"}`), 0o600); err != nil {
		t.Fatalf("write app_settings.json: %v", err)
	}
	settingsStore := settings_platform.NewStore(settingsPath)
	mock := &mockHomeService{page: &domain.HomePageResponse{}}
	factory := func(ctx context.Context, _ string) (port.HomeService, context.Context, string, string, error) {
		return mock, ctx, testXUID, "gt", nil
	}
	r := newHomeRouter(factory, settingsStore)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/home", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if mock.pageLocale != "en" {
		t.Fatalf("expected locale en from settings, got %q", mock.pageLocale)
	}
}

// TestHomeHandler_GetHomePage_HeaderOverridesSettings : le header
// X-LevelUp-Locale envoyé par le frontend prime sur app_settings.json.
// Permet au frontend de basculer la locale en runtime sans re-bootstrap.
func TestHomeHandler_GetHomePage_HeaderOverridesSettings(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "app_settings.json")
	// Settings en EN, mais le frontend demande FR via header.
	if err := os.WriteFile(settingsPath, []byte(`{"lang":"en"}`), 0o600); err != nil {
		t.Fatalf("write app_settings.json: %v", err)
	}
	settingsStore := settings_platform.NewStore(settingsPath)
	mock := &mockHomeService{page: &domain.HomePageResponse{}}
	factory := func(ctx context.Context, _ string) (port.HomeService, context.Context, string, string, error) {
		return mock, ctx, testXUID, "gt", nil
	}
	r := newHomeRouter(factory, settingsStore)
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/home", nil)
	req.Header.Set("X-LevelUp-Locale", "fr")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if mock.pageLocale != "fr" {
		t.Fatalf("expected locale fr from header (overriding settings=en), got %q", mock.pageLocale)
	}
}
