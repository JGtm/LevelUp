// Package handlers_test — settings_test.go : tests SettingsHandler.
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
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
)

func newSettingsRouter(t *testing.T, demoMode bool) *chi.Mux {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{DemoMode: demoMode}
	settingsStore := settings_platform.NewStore(filepath.Join(dir, "app_settings.json"))
	jobStore := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	h := handlers.NewSettingsHandler(cfg, settingsStore, jobStore)

	r := chi.NewRouter()
	r.Get("/settings", h.GetSettings)
	r.Patch("/settings", h.PatchSettings)
	r.Post("/settings/media/reset-index", h.PostMediaResetIndex)
	return r
}

func TestSettingsHandler_GetSettings_OK(t *testing.T) {
	r := newSettingsRouter(t, false)
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSettingsHandler_GetSettings_DemoMode(t *testing.T) {
	r := newSettingsRouter(t, true)
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 in demo mode, got %d", w.Code)
	}
}

func TestSettingsHandler_PatchSettings_DemoMode_422(t *testing.T) {
	r := newSettingsRouter(t, true)
	body := `{"lang": "en"}`
	req := httptest.NewRequest(http.MethodPatch, "/settings", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSettingsHandler_PatchSettings_InvalidBody(t *testing.T) {
	r := newSettingsRouter(t, false)
	req := httptest.NewRequest(http.MethodPatch, "/settings", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSettingsHandler_PatchSettings_OK(t *testing.T) {
	r := newSettingsRouter(t, false)
	body, _ := json.Marshal(map[string]string{"lang": "en"})
	req := httptest.NewRequest(http.MethodPatch, "/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSettingsHandler_PostMediaReset_NoConfirm(t *testing.T) {
	r := newSettingsRouter(t, false)
	body := `{"confirm_destructive": false}`
	req := httptest.NewRequest(http.MethodPost, "/settings/media/reset-index", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSettingsHandler_PostMediaReset_InvalidBody(t *testing.T) {
	r := newSettingsRouter(t, false)
	req := httptest.NewRequest(http.MethodPost, "/settings/media/reset-index", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSettingsHandler_PostMediaReset_OK(t *testing.T) {
	r := newSettingsRouter(t, false)
	body := `{"confirm_destructive": true}`
	req := httptest.NewRequest(http.MethodPost, "/settings/media/reset-index", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
}
