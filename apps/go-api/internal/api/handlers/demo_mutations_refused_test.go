package handlers_test

// demo_mutations_refused_test.go — les trois routes de mutation relevées par la revue
// adversariale du lot B5 (backlog 2026-09-26, constats C1 et C2, lot B-C1) sont REFUSÉES
// en démo, sur le modèle de POST /settings/backup/run (B5.6) : 403 demo_mode_forbidden.
//
// Pourquoi : RequireAdmin est transparent en démo, donc un visiteur les atteint.
//   - DELETE /profiles/{p}/titles/{t}/data : ProfileService est enraciné sur le dépôt ; sur
//     un vrai checkout, la purge effaçait <dépôt>/data/titles/<t>/players/<p>.
//   - POST /setup/players et PATCH /watcher/subscriptions : db_profiles.json et
//     app_settings.json visent la fixture, montée en écriture dans le conteneur de
//     production ; un visiteur anonyme y persistait ses écritures.
// Hors démo, le comportement est inchangé (chaque refus a son témoin « hors démo »).

import (
	"bytes"
	"encoding/json"
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

// exigerRefusDemo vérifie le contrat du refus : 403 et code demo_mode_forbidden.
func exigerRefusDemo(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	if w.Code != http.StatusForbidden {
		t.Errorf("démo : statut %d, attendu 403 — corps %s", w.Code, w.Body.String())
		return
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body["code"] != "demo_mode_forbidden" {
		t.Errorf("démo : code %v (err %v), attendu demo_mode_forbidden", body["code"], err)
	}
}

// ─── POST /setup/players ─────────────────────────────────────────────────────

func routeurCreationProfil(t *testing.T, demo bool, directory *mockDirectory) (*chi.Mux, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{
		DemoMode:        demo,
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

func TestSetupPlayers_DemoMode_Refuse(t *testing.T) {
	dir := &mockDirectory{playerKey: "Visiteur"}
	r, _ := routeurCreationProfil(t, true, dir)
	exigerRefusDemo(t, postCreationProfil(r))
	if dir.lastReq.Gamertag != "" {
		t.Errorf("démo : l'annuaire a reçu une création de profil (%+v)", dir.lastReq)
	}
}

func TestSetupPlayers_HorsDemo_Cree(t *testing.T) {
	dir := &mockDirectory{playerKey: "Visiteur"}
	r, _ := routeurCreationProfil(t, false, dir)
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
func fixturePurge(t *testing.T, demo bool) (*chi.Mux, string, string) {
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
	h := handlers.NewTitleSyncHandler(service.NewProfileService(profilesPath, root), demo)
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

func TestPurgeTitleData_DemoMode_Refuse(t *testing.T) {
	r, playerDir, profilesPath := fixturePurge(t, true)
	exigerRefusDemo(t, deletePurge(r))
	if _, err := os.Stat(filepath.Join(playerDir, "stats.duckdb")); err != nil {
		t.Errorf("démo : le dossier du joueur a été effacé (%v)", err)
	}
	data, _ := os.ReadFile(profilesPath)
	if !bytes.Contains(data, []byte(`"halo_5"`)) || !bytes.Contains(data, []byte(`"y"`)) {
		t.Errorf("démo : l'entrée de profil halo_5 a été retirée : %s", data)
	}
}

func TestPurgeTitleData_HorsDemo_Purge(t *testing.T) {
	r, playerDir, _ := fixturePurge(t, false)
	w := deletePurge(r)
	if w.Code != http.StatusOK {
		t.Fatalf("hors démo : statut %d, attendu 200 — corps %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(playerDir); !os.IsNotExist(err) {
		t.Errorf("hors démo : le dossier du joueur devait être supprimé (stat err = %v)", err)
	}
}

// ─── PATCH /watcher/subscriptions ────────────────────────────────────────────

func routeurAbonnements(t *testing.T, demo bool) (*chi.Mux, *mockDaemon, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{DemoMode: demo, RepoRoot: dir}
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

func TestWatcherSubscriptions_DemoMode_Refuse(t *testing.T) {
	r, daemon, settingsPath := routeurAbonnements(t, true)
	exigerRefusDemo(t, patchAbonnements(r))
	if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
		t.Errorf("démo : app_settings.json a été écrit (stat err = %v)", err)
	}
	if len(daemon.subscriptions) != 0 {
		t.Errorf("démo : les abonnements du watcher ont été modifiés (%v)", daemon.subscriptions)
	}
}

func TestWatcherSubscriptions_HorsDemo_Enregistre(t *testing.T) {
	r, daemon, settingsPath := routeurAbonnements(t, false)
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
