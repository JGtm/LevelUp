// Package handlers_test — setup_test.go : tests SetupHandler.
package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	session_platform "levelup/go-api/internal/platform/session"
	settings_platform "levelup/go-api/internal/platform/settings"
)

// mockProfileService implémente port.ProfileService pour les tests.
type mockProfileService struct {
	playerKey string
	warnings  []string
	err       error
}

func (m *mockProfileService) CreatePlayer(_ domain.CreatePlayerProfileRequest) (string, []string, error) {
	return m.playerKey, m.warnings, m.err
}

func newSetupRouter(t *testing.T, provisionEnabled bool, profileSvc *mockProfileService) *chi.Mux {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{
		RepoRoot:        dir,
		DBProfilesPath:  filepath.Join(dir, "db_profiles.json"),
		SessionDir:      filepath.Join(dir, "sessions"),
		AppSettingsPath: filepath.Join(dir, "app_settings.json"),
	}
	sessionStore := session_platform.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32-bytesXXXXXXXXXX")
	settingsStore := settings_platform.NewStore(cfg.AppSettingsPath)
	jobStore := jobs.NewStore(filepath.Join(dir, "jobs.json"))

	// Pré-configurer can_self_provision
	appCfg, _ := settingsStore.Load()
	appCfg.CanSelfProvision = provisionEnabled
	_ = settingsStore.Save(appCfg)

	h := handlers.NewSetupHandler(cfg, sessionStore, settingsStore, jobStore, profileSvc)

	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessionStore, false))
	r.Post("/setup/players", h.CreatePlayer)
	r.Post("/setup/smoke-test", h.SmokeTest)
	return r
}

func TestSetupHandler_CreatePlayer_ProvisionDisabled(t *testing.T) {
	svc := &mockProfileService{playerKey: "test-player"}
	r := newSetupRouter(t, false, svc)

	body := `{"gamertag": "TestPlayer", "profile_mode": "manual"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/players", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when provisioning disabled, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetupHandler_CreatePlayer_InvalidBody(t *testing.T) {
	svc := &mockProfileService{playerKey: "test-player"}
	r := newSetupRouter(t, true, svc)

	req := httptest.NewRequest(http.MethodPost, "/setup/players", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSetupHandler_CreatePlayer_EmptyGamertag(t *testing.T) {
	svc := &mockProfileService{playerKey: "test-player"}
	r := newSetupRouter(t, true, svc)

	body := `{"gamertag": "", "profile_mode": "manual"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/players", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty gamertag, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetupHandler_CreatePlayer_XboxModeNoIdentity(t *testing.T) {
	svc := &mockProfileService{playerKey: "test-player"}
	r := newSetupRouter(t, true, svc)

	body := `{"gamertag": "TestPlayer", "profile_mode": "xbox"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/players", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// xbox mode sans identité Halo liée → 409
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for xbox mode without identity, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetupHandler_SmokeTest_Accepted(t *testing.T) {
	svc := &mockProfileService{}
	r := newSetupRouter(t, true, svc)

	req := httptest.NewRequest(http.MethodPost, "/setup/smoke-test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	// Attendre la fin du goroutine SmokeTest (écriture jobs.json) avant cleanup Windows.
	time.Sleep(100 * time.Millisecond)
}
