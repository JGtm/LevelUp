// Package handlers_test — setup_test.go : tests SetupHandler.
package handlers_test

import (
	"bytes"
	"context"
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

// mockDirectory implémente port.PlayerDirectory pour les tests du SetupHandler.
// Seul Onboard porte un comportement : le handler ne consomme que lui (ADR 0035
// D4) — les trois méthodes de lecture sont là pour satisfaire le port.
type mockDirectory struct {
	playerKey string
	warnings  []string
	err       error
	lastReq   domain.OnboardRequest // capture la dernière demande reçue
}

func (m *mockDirectory) Onboard(_ context.Context, req domain.OnboardRequest) (domain.OnboardResult, error) {
	m.lastReq = req
	if m.err != nil {
		return domain.OnboardResult{}, m.err
	}
	return domain.OnboardResult{PlayerKey: m.playerKey, Warnings: m.warnings}, nil
}

func (m *mockDirectory) List(context.Context) (domain.AdminIdentitiesResponse, error) {
	return domain.AdminIdentitiesResponse{}, nil
}

func (m *mockDirectory) Get(context.Context, string) (domain.IdentityRecord, error) {
	return domain.IdentityRecord{}, nil
}

func (m *mockDirectory) HasTrackedProfile(context.Context, string, string) (bool, error) {
	return false, nil
}

func (m *mockDirectory) Purge(context.Context, string, domain.PurgeOptions) (domain.PurgeReport, error) {
	return domain.PurgeReport{}, nil
}

func newSetupRouter(t *testing.T, provisionEnabled bool, directory *mockDirectory) *chi.Mux {
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

	h := handlers.NewSetupHandler(cfg, sessionStore, settingsStore, jobStore).WithDirectory(directory)

	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessionStore, middleware.SecureCookiePolicy{}))
	h.Mount(r)
	return r
}

// TestSetupHandler_CreatePlayer_InstanceLocked : instance fermée → 403, même si
// can_self_provision=true (le verrou est cumulatif).
func TestSetupHandler_CreatePlayer_InstanceLocked(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.AppConfig{
		RepoRoot:        dir,
		DBProfilesPath:  filepath.Join(dir, "db_profiles.json"),
		SessionDir:      filepath.Join(dir, "sessions"),
		AppSettingsPath: filepath.Join(dir, "app_settings.json"),
		InstanceLocked:  true, // verrou forcé (env)
	}
	sessionStore := session_platform.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32-bytesXXXXXXXXXX")
	settingsStore := settings_platform.NewStore(cfg.AppSettingsPath)
	jobStore := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	appCfg, _ := settingsStore.Load()
	appCfg.CanSelfProvision = true // provisioning activé mais instance verrouillée
	_ = settingsStore.Save(appCfg)

	h := handlers.NewSetupHandler(cfg, sessionStore, settingsStore, jobStore).
		WithDirectory(&mockDirectory{playerKey: "x"}).
		WithInstanceLock(func() bool { return true }) // résolveur injecté (ADR 0035 D5)
	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessionStore, middleware.SecureCookiePolicy{}))
	h.Mount(r)

	body := `{"gamertag": "TestPlayer", "profile_mode": "manual"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/players", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when instance locked, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("instance_locked")) {
		t.Errorf("expected code instance_locked, body: %s", w.Body.String())
	}
}

func TestSetupHandler_CreatePlayer_ProvisionDisabled(t *testing.T) {
	svc := &mockDirectory{playerKey: "test-player"}
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
	svc := &mockDirectory{playerKey: "test-player"}
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
	svc := &mockDirectory{playerKey: "test-player"}
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

// Onboarding multi-titre : le title_slug du body prime sur le titre du contexte
// et initial_max_matches est propagé à l'annuaire.
func TestSetupHandler_CreatePlayer_PrefersBodyTitleSlug(t *testing.T) {
	svc := &mockDirectory{playerKey: "TestPlayer"}
	r := newSetupRouter(t, true, svc)

	body := `{"gamertag":"TestPlayer","profile_mode":"manual","title_slug":"halo_5","initial_max_matches":42}`
	req := httptest.NewRequest(http.MethodPost, "/setup/players", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if svc.lastReq.TitleSlug != "halo_5" {
		t.Fatalf("title_slug du body devrait primer, reçu %q", svc.lastReq.TitleSlug)
	}
	if svc.lastReq.InitialMaxMatches != 42 {
		t.Fatalf("initial_max_matches devrait être propagé, reçu %d", svc.lastReq.InitialMaxMatches)
	}
}

// POST /setup/players passe par l'annuaire (ADR 0035 D4) : le contrat de réponse
// est inchangé — la clé de profil et les warnings viennent d'Onboard.
func TestSetupHandler_CreatePlayer_ViaAnnuaire(t *testing.T) {
	svc := &mockDirectory{playerKey: "TestPlayer", warnings: []string{"Dossier joueur non cree"}}
	r := newSetupRouter(t, true, svc)

	body := `{"gamertag":"TestPlayer","profile_mode":"manual","xuid":"123"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/players", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("201 attendu, reçu %d : %s", w.Code, w.Body.String())
	}
	if svc.lastReq.Gamertag != "TestPlayer" || svc.lastReq.XUID != "123" {
		t.Errorf("la demande transmise à l'annuaire est incomplète : %+v", svc.lastReq)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Dossier joueur non cree")) {
		t.Errorf("les warnings d'Onboard doivent être rendus : %s", w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"player_slug":"TestPlayer"`)) {
		t.Errorf("la clé de profil rendue par Onboard doit être dans la réponse : %s", w.Body.String())
	}
}

// Annuaire non câblé : 503 typé, jamais un 201 qui ferait croire à un profil créé.
func TestSetupHandler_CreatePlayer_SansAnnuaire(t *testing.T) {
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
	appCfg, _ := settingsStore.Load()
	appCfg.CanSelfProvision = true
	_ = settingsStore.Save(appCfg)

	h := handlers.NewSetupHandler(cfg, sessionStore, settingsStore, jobStore)
	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessionStore, middleware.SecureCookiePolicy{}))
	h.Mount(r)

	body := `{"gamertag":"TestPlayer","profile_mode":"manual"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/players", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("503 attendu sans annuaire, reçu %d : %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("directory_unavailable")) {
		t.Errorf("code directory_unavailable attendu : %s", w.Body.String())
	}
}

func TestSetupHandler_CreatePlayer_XboxModeNoIdentity(t *testing.T) {
	svc := &mockDirectory{playerKey: "test-player"}
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
	svc := &mockDirectory{}
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
