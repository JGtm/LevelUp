// Package handlers_test — settings_lock_forced_test.go : le verrou forcé par
// l'environnement (LEVELUP_INSTANCE_LOCKED) ne se lève pas depuis les réglages
// (revue adversariale du 2026-09-16 : sinon le fichier passait à false, l'instance
// restait fermée, et l'interrupteur admin se recochait seul sans message).
package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
)

func newEnvLockedSettingsRouter(t *testing.T) (*chi.Mux, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{
		InstanceLocked: true, // LEVELUP_INSTANCE_LOCKED
		DBProfilesPath: filepath.Join(dir, "db_profiles.json"),
	}
	settingsPath := filepath.Join(dir, "app_settings.json")
	settingsStore := settings_platform.NewStore(settingsPath)
	jobStore := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	h := handlers.NewSettingsHandlerWithIndexer(cfg, settingsStore, jobStore, &mockMediaIndexer{})
	r := chi.NewRouter()
	h.Mount(r)
	return r, settingsPath
}

func patchInstanceLock(t *testing.T, r *chi.Mux, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/settings", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSettingsHandler_PatchInstanceLocked_EnvForced_Refuse409(t *testing.T) {
	r, settingsPath := newEnvLockedSettingsRouter(t)

	w := patchInstanceLock(t, r, `{"instance_locked": false}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("corps: %v", err)
	}
	if resp.Code != "instance_lock_forced" {
		t.Fatalf("code = %q, attendu instance_lock_forced", resp.Code)
	}
	// Un refus ne modifie rien : le fichier n'existe pas encore, ou ne porte pas
	// la valeur explicite `false`.
	if raw, err := os.ReadFile(settingsPath); err == nil && bytes.Contains(raw, []byte(`"instance_locked": false`)) {
		t.Fatalf("le refus ne doit rien persister, fichier = %s", raw)
	}

	// Poser `true` reste accepté (no-op cohérent avec l'environnement).
	w = patchInstanceLock(t, r, `{"instance_locked": true}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 pour instance_locked=true, got %d: %s", w.Code, w.Body.String())
	}
}
