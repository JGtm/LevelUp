package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/pkg/duckdbbackup"
)

// newBackupRouter builds a chi router wiring GetBackupStatus + PostBackupRun.
// backupSched may be nil to test the "scheduler not configured" paths.
func newBackupRouter(t *testing.T, backupSched *duckdbbackup.Scheduler) *chi.Mux {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{
		DBProfilesPath: filepath.Join(dir, "db_profiles.json"),
	}
	settingsStore := settings_platform.NewStore(filepath.Join(dir, "app_settings.json"))
	jobStore := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	h := handlers.NewSettingsHandlerWithIndexer(cfg, settingsStore, jobStore, &mockMediaIndexer{})
	if backupSched != nil {
		h = h.WithBackupScheduler(backupSched)
	}
	r := chi.NewRouter()
	r.Get("/settings/backup/status", h.GetBackupStatus)
	r.Post("/settings/backup/run", h.PostBackupRun)
	return r
}

// newIdleScheduler returns a Scheduler with no targets and no restic binary
// required — RunOnce will immediately return Skipped=true.
func newIdleScheduler(t *testing.T) *duckdbbackup.Scheduler {
	t.Helper()
	dir := t.TempDir()
	cfg := duckdbbackup.Config{Enabled: true, BackupDir: dir, KeepDaily: 7, KeepWeekly: 4, KeepMonthly: 12}
	return duckdbbackup.New(cfg, func() ([]duckdbbackup.Target, error) { return nil, nil })
}

// ---------------------------------------------------------------------------
// GetBackupStatus
// ---------------------------------------------------------------------------

func TestGetBackupStatus_NilScheduler(t *testing.T) {
	r := newBackupRouter(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/settings/backup/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["enabled"] != false {
		t.Errorf("expected enabled=false, got %v", body["enabled"])
	}
	if body["available"] != false {
		t.Errorf("expected available=false, got %v", body["available"])
	}
}

func TestGetBackupStatus_WithScheduler(t *testing.T) {
	r := newBackupRouter(t, newIdleScheduler(t))
	req := httptest.NewRequest(http.MethodGet, "/settings/backup/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", body["enabled"])
	}
	// config must be present and contain expected fields
	cfg, ok := body["config"].(map[string]any)
	if !ok {
		t.Fatalf("expected config object, got %T: %v", body["config"], body["config"])
	}
	if cfg["keep_daily"] == nil {
		t.Error("expected config.keep_daily to be present")
	}
}

// ---------------------------------------------------------------------------
// PostBackupRun
// ---------------------------------------------------------------------------

func TestPostBackupRun_NilScheduler(t *testing.T) {
	r := newBackupRouter(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/settings/backup/run", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
	// Axe 5 : le 503 doit suivre le shape d'erreur standard {code, message,
	// retryable} en JSON (avant : http.Error → text/plain).
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("body non JSON: %v", err)
	}
	for _, k := range []string{"code", "message", "retryable"} {
		if _, ok := body[k]; !ok {
			t.Errorf("clé %q absente du shape standard: %v", k, body)
		}
	}
	if body["code"] != "backup_scheduler_unavailable" {
		t.Errorf("code = %v, want backup_scheduler_unavailable", body["code"])
	}
}

func TestPostBackupRun_Skipped(t *testing.T) {
	// Idle scheduler (no targets) → RunOnce returns Skipped=true immediately.
	r := newBackupRouter(t, newIdleScheduler(t))
	req := httptest.NewRequest(http.MethodPost, "/settings/backup/run", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result map[string]any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["skipped"] != true {
		t.Errorf("expected skipped=true (no targets), got %v", result["skipped"])
	}
}
