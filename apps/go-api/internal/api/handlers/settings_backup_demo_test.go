package handlers_test

// settings_backup_demo_test.go — témoin HORS DÉMO de la sauvegarde restic : le cycle est
// lancé. En démo, la route est refusée avant le handler par la garde générale
// (middleware/demo_read_only.go, ratchet internal/api/demo_read_only_ratchet_test.go).

import (
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

func routeurSauvegarde(t *testing.T, sched *duckdbbackup.Scheduler) *chi.Mux {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{DBProfilesPath: filepath.Join(dir, "db_profiles.json")}
	h := handlers.NewSettingsHandlerWithIndexer(cfg,
		settings_platform.NewStore(filepath.Join(dir, "app_settings.json")),
		jobs.NewStore(filepath.Join(dir, "jobs.json")), &mockMediaIndexer{}).
		WithBackupScheduler(sched)
	r := chi.NewRouter()
	h.Mount(r)
	return r
}

func TestPostBackupRun_HorsDemo_Lance(t *testing.T) {
	var appels atomic.Int32
	r := routeurSauvegarde(t, ordonnanceurEspion(t, &appels))
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
