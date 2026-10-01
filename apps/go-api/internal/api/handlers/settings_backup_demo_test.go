package handlers_test

// settings_backup_demo_test.go — la sauvegarde restic est REFUSÉE en démo (backlog
// 2026-09-26, lot B5.6 ; décision D-7).
//
// En démo, RequireAdmin est transparent (middleware/require_admin.go) : n'importe quel
// visiteur atteint POST /settings/backup/run. Le scheduler de sauvegarde est construit sur
// le PathResolver du dépôt : sans refus, une démo lancée sur un poste de dev (harnais
// visuel, LEVELUP_REPO_ROOT = le vrai checkout) sauvegardait les bases RÉELLES.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/pkg/duckdbbackup"
)

// ordonnanceurEspion : un scheduler dont l'énumération des cibles compte ses appels — un
// cycle de sauvegarde lancé l'appelle forcément.
func ordonnanceurEspion(t *testing.T, appels *atomic.Int32) *duckdbbackup.Scheduler {
	t.Helper()
	cfg := duckdbbackup.Config{Enabled: true, BackupDir: t.TempDir(), KeepDaily: 7, KeepWeekly: 4, KeepMonthly: 12}
	return duckdbbackup.New(cfg, func() ([]duckdbbackup.Target, error) {
		appels.Add(1)
		return nil, nil
	})
}

func routeurSauvegarde(t *testing.T, demo bool, sched *duckdbbackup.Scheduler) *chi.Mux {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{DemoMode: demo, DBProfilesPath: filepath.Join(dir, "db_profiles.json")}
	h := handlers.NewSettingsHandlerWithIndexer(cfg,
		settings_platform.NewStore(filepath.Join(dir, "app_settings.json")),
		jobs.NewStore(filepath.Join(dir, "jobs.json")), &mockMediaIndexer{}).
		WithBackupScheduler(sched)
	r := chi.NewRouter()
	h.Mount(r)
	return r
}

func TestPostBackupRun_DemoMode_Refuse(t *testing.T) {
	var appels atomic.Int32
	r := routeurSauvegarde(t, true, ordonnanceurEspion(t, &appels))
	req := httptest.NewRequest(http.MethodPost, "/settings/backup/run", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("démo : statut %d, attendu 403 (sauvegarde refusée) — corps %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err == nil && body["code"] != "demo_mode_forbidden" {
		t.Errorf("démo : code %v, attendu demo_mode_forbidden", body["code"])
	}
	if n := appels.Load(); n != 0 {
		t.Errorf("démo : un cycle de sauvegarde a été lancé (%d énumération(s) de cibles)", n)
	}
}

func TestPostBackupRun_HorsDemo_Lance(t *testing.T) {
	var appels atomic.Int32
	r := routeurSauvegarde(t, false, ordonnanceurEspion(t, &appels))
	req := httptest.NewRequest(http.MethodPost, "/settings/backup/run", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("hors démo : statut %d, attendu 200 — corps %s", w.Code, w.Body.String())
	}
	if appels.Load() == 0 {
		t.Error("hors démo : le cycle de sauvegarde n'a pas été lancé (comportement inchangé attendu)")
	}
}
