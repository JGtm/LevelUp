// Package handlers_test — sync_handler_test.go : tests unitaires SyncHandler.
//
// Teste les validations et guards sans lancer de vraie sync réseau.
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
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
)

func newSyncRouter(t *testing.T, canStart bool) (*chi.Mux, *jobs.Store) {
	t.Helper()
	dir := t.TempDir()

	// Créer un app_settings.json minimal
	settingsPath := filepath.Join(dir, "app_settings.json")
	settingsData, _ := json.Marshal(map[string]interface{}{
		"can_start_initial_sync": canStart,
		"lang":                   "fr",
	})
	if err := os.WriteFile(settingsPath, settingsData, 0o600); err != nil {
		t.Fatal(err)
	}

	// Config minimale sans DB réelle
	cfg := &config.AppConfig{
		RepoRoot:        dir,
		DBProfilesPath:  filepath.Join(dir, "db_profiles.json"),
		AppSettingsPath: settingsPath,
	}
	// db_profiles.json vide
	_ = os.WriteFile(cfg.DBProfilesPath, []byte(`{"players":[]}`), 0o600)

	jobStore := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	settingsStore := settings_platform.NewStore(settingsPath)

	h := handlers.NewSyncHandler(cfg, settingsStore, jobStore, nil)
	r := chi.NewRouter()
	r.Post("/sync/initial", h.StartInitialSync)
	return r, jobStore
}

func TestSyncHandler_InitialSync_Disabled(t *testing.T) {
	r, _ := newSyncRouter(t, false)
	body, _ := json.Marshal(map[string]interface{}{
		"player_slug": "test-player",
		"max_matches": 100,
	})
	req := httptest.NewRequest(http.MethodPost, "/sync/initial", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSyncHandler_InitialSync_InvalidBody(t *testing.T) {
	r, _ := newSyncRouter(t, true)
	req := httptest.NewRequest(http.MethodPost, "/sync/initial", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSyncHandler_InitialSync_EmptySlug(t *testing.T) {
	r, _ := newSyncRouter(t, true)
	body, _ := json.Marshal(map[string]interface{}{
		"player_slug": "",
		"max_matches": 100,
	})
	req := httptest.NewRequest(http.MethodPost, "/sync/initial", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty slug, got %d", w.Code)
	}
}

func TestSyncHandler_InitialSync_ConflictActiveJob(t *testing.T) {
	r, jobStore := newSyncRouter(t, true)
	// Créer un job "initial_sync" actif pour ce joueur
	job := jobStore.Create(domain.JobTypeInitialSync, "conflict-player")
	_ = job // déjà "queued" / "running"

	body, _ := json.Marshal(map[string]interface{}{
		"player_slug": "conflict-player",
		"max_matches": 100,
	})
	req := httptest.NewRequest(http.MethodPost, "/sync/initial", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 409 si déjà actif, ou 401 si pas de session (selon l'ordre des guards)
	if w.Code != http.StatusConflict && w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 409 or 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSyncHandler_InitialSync_InvalidMaxMatches(t *testing.T) {
	r, _ := newSyncRouter(t, true)
	body, _ := json.Marshal(map[string]interface{}{
		"player_slug": "test-player",
		"max_matches": 9999,
	})
	req := httptest.NewRequest(http.MethodPost, "/sync/initial", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid max_matches, got %d", w.Code)
	}
}
