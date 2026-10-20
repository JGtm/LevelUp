package handlers_test

// demo_mutations_refused_test.go — TÉMOINS HORS DÉMO de trois écritures que la démo refuse :
// création de profil, purge des données d'un titre, abonnements du watcher.
//
// Le REFUS lui-même n'est plus porté par ces handlers : la garde générale « démo en lecture
// seule » (middleware/demo_read_only.go) refuse toute écriture avant le handler, et
// internal/api/demo_read_only_ratchet_test.go la vérifie sur le routeur assemblé. Ces témoins
// gardent la preuve que, hors démo, chaque écriture a toujours lieu.

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/jobs"
	session_platform "levelup/go-api/internal/platform/session"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/service"
)

// ─── POST /setup/players ─────────────────────────────────────────────────────

func routeurCreationProfil(t *testing.T, directory *mockDirectory) (*chi.Mux, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{
		RepoRoot:        dir,
		DBProfilesPath:  filepath.Join(dir, "db_profiles.json"),
		SessionDir:      filepath.Join(dir, "sessions"),
		AppSettingsPath: filepath.Join(dir, "app_settings.json"),
	}
	sessionStore := session_platform.NewStore(cfg.SessionDir, time.Hour, "test-secret-32-bytesXXXXXXXXXX")
	settingsStore := settings_platform.NewStore(cfg.AppSettingsPath)
	appCfg, _ := settingsStore.Load()
	appCfg.CanSelfProvision = true
	if err := settingsStore.Save(appCfg); err != nil {
		t.Fatalf("app_settings : %v", err)
	}
	h := handlers.NewSetupHandler(cfg, sessionStore, settingsStore, jobs.NewStore(filepath.Join(dir, "jobs.json"))).
		WithDirectory(directory)
	r := chi.NewRouter()
	h.Mount(r)
	return r, cfg.AppSettingsPath
}

func postCreationProfil(r *chi.Mux) *httptest.ResponseRecorder {
	body := `{"gamertag":"Visiteur","profile_mode":"manual","xuid":"123"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/players", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSetupPlayers_HorsDemo_Cree(t *testing.T) {
	dir := &mockDirectory{playerKey: "Visiteur"}
	r, _ := routeurCreationProfil(t, dir)
	w := postCreationProfil(r)
	if w.Code != http.StatusCreated {
		t.Fatalf("hors démo : statut %d, attendu 201 — corps %s", w.Code, w.Body.String())
	}
	if dir.lastReq.Gamertag != "Visiteur" {
		t.Errorf("hors démo : l'annuaire n'a pas reçu la création (%+v)", dir.lastReq)
	}
}

// ─── DELETE /profiles/{p}/titles/{t}/data ────────────────────────────────────

// fixturePurge : un joueur suivi sur deux titres (le store refuse de purger le dernier
// titre actif) et un dossier de joueur RÉEL sous la racine, avec un fichier sentinelle.
func fixturePurge(t *testing.T) (*chi.Mux, string, string) {
	t.Helper()
	root := t.TempDir()
	profilesPath := filepath.Join(root, "db_profiles.json")
	if err := os.WriteFile(profilesPath, []byte(`{
  "version": "3.0",
  "profiles": {
    "halo_infinite": {"Spartan": {"db_path": "x", "xuid": "111"}},
    "halo_5": {"Spartan": {"db_path": "y", "xuid": "111"}}
  }
}`), 0o644); err != nil {
		t.Fatalf("db_profiles : %v", err)
	}
	playerDir := title.NewPathResolver(root).PlayerDir("halo_5", "Spartan")
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		t.Fatalf("dossier joueur : %v", err)
	}
	if err := os.WriteFile(filepath.Join(playerDir, "stats.duckdb"), []byte("sentinelle"), 0o644); err != nil {
		t.Fatalf("sentinelle : %v", err)
	}
	h := handlers.NewTitleSyncHandler(service.NewProfileService(profilesPath, root))
	r := chi.NewRouter()
	r.Route("/profiles/{player_slug}/titles/{slug}", func(r chi.Router) { h.Mount(r) })
	return r, playerDir, profilesPath
}

func deletePurge(r *chi.Mux) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, "/profiles/Spartan/titles/halo_5/data", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestPurgeTitleData_HorsDemo_Purge(t *testing.T) {
	r, playerDir, _ := fixturePurge(t)
	w := deletePurge(r)
	if w.Code != http.StatusOK {
		t.Fatalf("hors démo : statut %d, attendu 200 — corps %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(playerDir); !os.IsNotExist(err) {
		t.Errorf("hors démo : le dossier du joueur devait être supprimé (stat err = %v)", err)
	}
}

// ─── PATCH /watcher/subscriptions ────────────────────────────────────────────

func routeurAbonnements(t *testing.T) (*chi.Mux, *mockDaemon, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: dir}
	settingsPath := filepath.Join(dir, "app_settings.json")
	daemon := &mockDaemon{running: true}
	h := handlers.NewWatcherHandler(cfg, settings_platform.NewStore(settingsPath), daemon,
		&stubTokenProvider{}, nil)
	r := chi.NewRouter()
	r.Route("/watcher", func(r chi.Router) { h.Mount(r) })
	return r, daemon, settingsPath
}

func patchAbonnements(r *chi.Mux) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPatch, "/watcher/subscriptions",
		strings.NewReader(`{"subscribed_players":["Visiteur"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestWatcherSubscriptions_HorsDemo_Enregistre(t *testing.T) {
	r, daemon, settingsPath := routeurAbonnements(t)
	w := patchAbonnements(r)
	if w.Code != http.StatusOK {
		t.Fatalf("hors démo : statut %d, attendu 200 — corps %s", w.Code, w.Body.String())
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil || !bytes.Contains(data, []byte("Visiteur")) {
		t.Errorf("hors démo : app_settings.json non écrit (err %v) : %s", err, data)
	}
	if len(daemon.subscriptions) != 1 || daemon.subscriptions[0] != "Visiteur" {
		t.Errorf("hors démo : abonnements du watcher %v, attendu [Visiteur]", daemon.subscriptions)
	}
}
