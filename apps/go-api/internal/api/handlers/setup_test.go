// Package handlers_test — setup_test.go : tests SetupHandler.
package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
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
	"levelup/go-api/internal/platform/userstore"
)

// mockProfileService implémente port.ProfileService pour les tests.
type mockProfileService struct {
	playerKey string
	warnings  []string
	err       error
	lastReq   domain.CreatePlayerProfileRequest // capture la dernière requête reçue
}

func (m *mockProfileService) CreatePlayer(req domain.CreatePlayerProfileRequest) (string, []string, error) {
	m.lastReq = req
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

	h := handlers.NewSetupHandler(cfg, sessionStore, settingsStore, jobStore, &mockProfileService{playerKey: "x"})
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

// Onboarding multi-titre : le title_slug du body prime sur le titre du contexte
// et initial_max_matches est propagé au ProfileService.
func TestSetupHandler_CreatePlayer_PrefersBodyTitleSlug(t *testing.T) {
	svc := &mockProfileService{playerKey: "TestPlayer"}
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

// ─── Droit de provisioning sur instance verrouillée (D3, 2026-09-15) ─────────
//
// Un invité arrivé par lien d'invitation doit pouvoir créer SON profil une fois,
// sur une instance pourtant fermée. Le droit est porté par le COMPTE
// (users.json), pas par la session : il survit à une déconnexion entre le login
// SSO et le Setup.

// lockedSetupRig monte un SetupHandler sur instance verrouillée, avec le user
// store câblé. Retourne le routeur, le store et le session store.
func lockedSetupRig(t *testing.T, svc *mockProfileService) (*chi.Mux, *userstore.Store, *session_platform.Store, *config.AppConfig) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{
		RepoRoot:        dir,
		DBProfilesPath:  filepath.Join(dir, "db_profiles.json"),
		SessionDir:      filepath.Join(dir, "sessions"),
		AppSettingsPath: filepath.Join(dir, "app_settings.json"),
		InstanceLocked:  true,
	}
	sessionStore := session_platform.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32-bytesXXXXXXXXXX")
	settingsStore := settings_platform.NewStore(cfg.AppSettingsPath)
	jobStore := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	appCfg, _ := settingsStore.Load()
	appCfg.CanSelfProvision = true
	_ = settingsStore.Save(appCfg)

	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	h := handlers.NewSetupHandler(cfg, sessionStore, settingsStore, jobStore, svc).
		WithProvisionGrant(users, users)

	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessionStore, middleware.SecureCookiePolicy{}))
	h.Mount(r)
	return r, users, sessionStore, cfg
}

// guestSession crée un compte Xbox lié (avec ou sans droit) et sa session.
func guestSession(t *testing.T, users *userstore.Store, sessions *session_platform.Store,
	gamertag, xuid, grant string,
) *http.Cookie {
	t.Helper()
	user, err := users.CreateFromXbox(gamertag, xuid)
	if err != nil {
		t.Fatalf("CreateFromXbox: %v", err)
	}
	if grant != "" {
		if err := users.SetProvisionGrant(user.Username, grant); err != nil {
			t.Fatalf("SetProvisionGrant: %v", err)
		}
	}
	sess := sessions.New()
	sess.LinkedHaloIdentity = &domain.HaloIdentity{XUID: xuid, Gamertag: gamertag}
	username := user.Username
	sess.Username = &username
	if err := sessions.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}
	return &http.Cookie{Name: session_platform.CookieName, Value: sessions.SignCookie(sess.SessionID)}
}

func postCreatePlayer(t *testing.T, r *chi.Mux, cookie *http.Cookie, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/setup/players", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// Verrouillé + droit + aucun profil → 201, puis le droit est vidé : un second
// appel retombe sur le 403 instance_locked.
func TestSetupHandler_LockedWithGrant_CreatesOnceThenRefuses(t *testing.T) {
	svc := &mockProfileService{playerKey: "Guest"}
	r, users, sessions, _ := lockedSetupRig(t, svc)
	cookie := guestSession(t, users, sessions, "Guest", "guest-x", "INVITE1")

	body := `{"gamertag": "Guest", "profile_mode": "xbox", "xuid": "guest-x"}`
	if w := postCreatePlayer(t, r, cookie, body); w.Code != http.StatusCreated {
		t.Fatalf("1er appel = %d, want 201 (corps = %s)", w.Code, w.Body.String())
	}
	user, err := users.GetByXUID("guest-x")
	if err != nil {
		t.Fatalf("GetByXUID: %v", err)
	}
	if user.ProvisionGrant != "" {
		t.Errorf("provision_grant = %q après usage, want vide (droit non rejouable)", user.ProvisionGrant)
	}

	w := postCreatePlayer(t, r, cookie, body)
	if w.Code != http.StatusForbidden {
		t.Fatalf("2e appel = %d, want 403 (corps = %s)", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("instance_locked")) {
		t.Errorf("2e appel : code attendu instance_locked, corps = %s", w.Body.String())
	}
}

// Verrouillé + compte SANS droit → 403 instance_locked (le ratchet d'origine).
func TestSetupHandler_LockedWithoutGrant_Refused(t *testing.T) {
	svc := &mockProfileService{playerKey: "Guest"}
	r, users, sessions, _ := lockedSetupRig(t, svc)
	cookie := guestSession(t, users, sessions, "Guest", "guest-x", "")

	w := postCreatePlayer(t, r, cookie, `{"gamertag": "Guest", "profile_mode": "xbox", "xuid": "guest-x"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("= %d, want 403 (corps = %s)", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("instance_locked")) {
		t.Errorf("code attendu instance_locked, corps = %s", w.Body.String())
	}
}

// Verrouillé + droit MAIS le compte a déjà un profil → 403 : le droit vaut pour
// le PREMIER profil, pas pour un second.
func TestSetupHandler_LockedWithGrant_ExistingProfileRefused(t *testing.T) {
	svc := &mockProfileService{playerKey: "Guest"}
	r, users, sessions, cfg := lockedSetupRig(t, svc)
	cookie := guestSession(t, users, sessions, "Guest", "guest-x", "INVITE1")

	profiles := `{"version":"3.0","admin":"Guest","profiles":{"halo_infinite":{"Guest":` +
		`{"db_path":"data/x.duckdb","xuid":"guest-x","waypoint_player":"Guest"}}}}`
	if err := os.WriteFile(cfg.DBProfilesPath, []byte(profiles), 0o600); err != nil {
		t.Fatalf("écriture db_profiles: %v", err)
	}

	w := postCreatePlayer(t, r, cookie, `{"gamertag": "Guest", "profile_mode": "xbox", "xuid": "guest-x"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("= %d, want 403 (corps = %s)", w.Code, w.Body.String())
	}
}

// Verrouillé + droit MAIS gamertag d'autrui → 409 : le droit ne dispense pas du
// contrôle « xuid = identité liée », qui reste la vraie barrière.
func TestSetupHandler_LockedWithGrant_ForeignGamertagRefused(t *testing.T) {
	svc := &mockProfileService{playerKey: "Other"}
	r, users, sessions, _ := lockedSetupRig(t, svc)
	cookie := guestSession(t, users, sessions, "Guest", "guest-x", "INVITE1")

	w := postCreatePlayer(t, r, cookie, `{"gamertag": "Autrui", "profile_mode": "xbox", "xuid": "autrui-x"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("= %d, want 409 identity_mismatch (corps = %s)", w.Code, w.Body.String())
	}
}
