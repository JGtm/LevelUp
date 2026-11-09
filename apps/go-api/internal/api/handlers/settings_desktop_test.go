// Package handlers_test — settings_desktop_test.go : les réglages d'intégration au poste de
// l'application de bureau (lot L21) : défauts, refus hors capacité, état vivant.
package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/gameproc"
	"levelup/go-api/internal/platform/jobs"
	"levelup/go-api/internal/platform/schedtask"
	settings_platform "levelup/go-api/internal/platform/settings"
)

type desktopFixture struct {
	r     *chi.Mux
	store *settings_platform.Store
}

// newDesktopFixture : un routeur de réglages dont le profil porte les capacités données.
func newDesktopFixture(t *testing.T, caps config.DeploymentCapabilities) desktopFixture {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: dir, DBProfilesPath: filepath.Join(dir, "db_profiles.json")}
	cfg.Capabilities = caps
	store := settings_platform.NewStore(filepath.Join(dir, "app_settings.json"))
	h := handlers.NewSettingsHandler(cfg, store, jobs.NewStore(filepath.Join(dir, "jobs.json")))
	r := chi.NewRouter()
	h.Mount(r)
	return desktopFixture{r: r, store: store}
}

func (f desktopFixture) patch(t *testing.T, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/settings", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	f.r.ServeHTTP(w, req)
	return w
}

var desktopCaps = config.DeploymentCapabilities{
	GameProcessWatch: true, ShellIntegration: true, ScheduledRefresh: true, EditSettings: true,
}

// TestDesktopSettings_Defauts : la détection du jeu est active, les quatre autres réglages sont
// coupés tant que l'utilisateur n'a rien demandé (décision du 2026-11-02).
func TestDesktopSettings_Defauts(t *testing.T) {
	f := newDesktopFixture(t, desktopCaps)
	cfg, err := f.store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	d := cfg.DesktopSettings
	if !d.GameProcessWatch || d.NativeNotifications || d.CloseToTray || d.StartWithWindows || d.ScheduledRefresh {
		t.Fatalf("défauts = %+v ; attendu game_process_watch seul à vrai", d)
	}
}

// TestDesktopSettings_PatchEtRelecture : un PATCH écrit les clés et GET /settings les rend.
func TestDesktopSettings_PatchEtRelecture(t *testing.T) {
	f := newDesktopFixture(t, desktopCaps)
	w := f.patch(t, map[string]any{
		"game_process_watch": false, "native_notifications": true, "close_to_tray": true,
		"start_with_windows": true, "scheduled_refresh": true,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH = %d : %s", w.Code, w.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("réponse : %v", err)
	}
	want := map[string]bool{
		"game_process_watch": false, "native_notifications": true, "close_to_tray": true,
		"start_with_windows": true, "scheduled_refresh": true,
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, attendu %v", k, got[k], v)
		}
	}
	cfg, _ := f.store.Load()
	if cfg.GameProcessWatch || !cfg.NativeNotifications || !cfg.CloseToTray || !cfg.StartWithWindows || !cfg.ScheduledRefresh {
		t.Fatalf("persisté = %+v", cfg.DesktopSettings)
	}
}

// TestDesktopSettings_RefuseHorsCapacite : chaque clé suit SA capacité ; sans elle, 400 et rien n'est
// écrit.
func TestDesktopSettings_RefuseHorsCapacite(t *testing.T) {
	cas := []struct {
		nom  string
		caps config.DeploymentCapabilities
		body map[string]any
	}{
		{"détection du jeu", config.DeploymentCapabilities{ShellIntegration: true, ScheduledRefresh: true}, map[string]any{"game_process_watch": false}},
		{"notifications natives", config.DeploymentCapabilities{GameProcessWatch: true, ScheduledRefresh: true}, map[string]any{"native_notifications": true}},
		{"zone de notification", config.DeploymentCapabilities{GameProcessWatch: true}, map[string]any{"close_to_tray": true}},
		{"démarrage avec Windows", config.DeploymentCapabilities{GameProcessWatch: true}, map[string]any{"start_with_windows": true}},
		{"tâche planifiée", config.DeploymentCapabilities{GameProcessWatch: true, ShellIntegration: true}, map[string]any{"scheduled_refresh": true}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			f := newDesktopFixture(t, c.caps)
			w := f.patch(t, c.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("PATCH = %d, attendu 400 : %s", w.Code, w.Body.String())
			}
			var env struct {
				Code string `json:"code"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &env)
			if env.Code != "desktop_setting_not_offered" {
				t.Errorf("code = %q, attendu desktop_setting_not_offered", env.Code)
			}
			cfg, _ := f.store.Load()
			d := cfg.DesktopSettings
			if !d.GameProcessWatch || d.NativeNotifications || d.CloseToTray || d.StartWithWindows || d.ScheduledRefresh {
				t.Errorf("un PATCH refusé ne doit rien écrire : %+v", d)
			}
		})
	}
}

// TestDesktopStatus_LeJeuTourne : GET /settings/desktop rend l'état du mode jeu du processus.
func TestDesktopStatus_LeJeuTourne(t *testing.T) {
	f := newDesktopFixture(t, desktopCaps)
	m := gameproc.NewMode()
	prev := gameproc.SetDefault(m)
	t.Cleanup(func() { gameproc.SetDefault(prev) })

	get := func() bool {
		req := httptest.NewRequest(http.MethodGet, "/settings/desktop", nil)
		w := httptest.NewRecorder()
		f.r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET = %d : %s", w.Code, w.Body.String())
		}
		var body struct {
			GameRunning bool `json:"game_running"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("réponse : %v", err)
		}
		return body.GameRunning
	}
	if get() {
		t.Fatal("jeu fermé : game_running doit être faux")
	}
	w := &gameproc.Watcher{
		Lister: gameproc.ListerFunc(func() ([]string, error) { return []string{"HaloInfinite.exe"}, nil }),
		Mode:   m,
		Names:  func() []string { return []string{"HaloInfinite.exe"} },
	}
	w.Poll(t.Context())
	if !get() {
		t.Fatal("jeu lancé : game_running doit être vrai")
	}
}

// TestDesktopStatus_TachePlanifiee : l'état de la tâche planifiée n'est exposé que sous sa capacité ; avant
// toute vérification il vaut unknown, ensuite ce que la boucle du boot a observé, avec un code fixe d'erreur.
func TestDesktopStatus_TachePlanifiee(t *testing.T) {
	obs := schedtask.DefaultObserved()
	t.Cleanup(func() { obs.Set("", "") })

	get := func(f desktopFixture) map[string]any {
		req := httptest.NewRequest(http.MethodGet, "/settings/desktop", nil)
		w := httptest.NewRecorder()
		f.r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET = %d : %s", w.Code, w.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("réponse : %v", err)
		}
		return body
	}
	taskOf := func(body map[string]any) map[string]any {
		task, _ := body["scheduled_refresh"].(map[string]any)
		return task
	}

	f := newDesktopFixture(t, desktopCaps)
	obs.Set("", "")
	if task := taskOf(get(f)); task["state"] != "unknown" {
		t.Fatalf("avant toute vérification : %v, attendu unknown", task)
	}
	obs.Set(schedtask.StatePresent, "")
	if task := taskOf(get(f)); task["state"] != "present" || task["error_code"] != nil {
		t.Fatalf("tâche présente : %v", task)
	}
	obs.Set(schedtask.StateError, schedtask.ErrorCodeSchtasks)
	if task := taskOf(get(f)); task["state"] != "error" || task["error_code"] != "schtasks_failed" {
		t.Fatalf("erreur : %v, attendu le code fixe schtasks_failed", task)
	}

	noTask := newDesktopFixture(t, config.DeploymentCapabilities{GameProcessWatch: true})
	if _, present := get(noTask)["scheduled_refresh"]; present {
		t.Fatal("sans la capacité ScheduledRefresh, l'état de la tâche n'est pas exposé")
	}
}
