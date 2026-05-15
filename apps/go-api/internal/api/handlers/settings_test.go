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

// newSettingsRouter retourne le routeur + le jobStore. Le jobStore est exposé
// pour permettre aux tests qui lancent des jobs background (PostMediaResetIndex,
// PostRecalculateSessions) d'attendre la fin du job via pollJobSucceeded —
// sinon la goroutine background continue à écrire `jobs.json` après que
// t.TempDir() a tenté de cleanup le dossier, ce qui faisait fail intermittent
// les tests (« RemoveAll cleanup: directory not empty »).
func newSettingsRouter(t *testing.T, demoMode bool) (*chi.Mux, *jobs.Store) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{
		DemoMode:       demoMode,
		DBProfilesPath: filepath.Join(dir, "db_profiles.json"),
	}
	settingsStore := settings_platform.NewStore(filepath.Join(dir, "app_settings.json"))
	jobStore := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	// Mock indexer pour éviter que la goroutine background de PostMediaResetIndex
	// scanne le disque réel (DirMediaIndexer par défaut) et produise des effets
	// de bord sur les tests suivants.
	h := handlers.NewSettingsHandlerWithIndexer(cfg, settingsStore, jobStore, &mockMediaIndexer{})

	r := chi.NewRouter()
	r.Get("/settings", h.GetSettings)
	r.Patch("/settings", h.PatchSettings)
	r.Post("/settings/media/reset-index", h.PostMediaResetIndex)
	r.Post("/settings/sessions/recalculate", h.PostRecalculateSessions)
	return r, jobStore
}

func TestSettingsHandler_GetSettings_OK(t *testing.T) {
	r, _ := newSettingsRouter(t, false)
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSettingsHandler_GetSettings_DemoMode(t *testing.T) {
	r, _ := newSettingsRouter(t, true)
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 in demo mode, got %d", w.Code)
	}
}

func TestSettingsHandler_PatchSettings_DemoMode_422(t *testing.T) {
	r, _ := newSettingsRouter(t, true)
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
	r, _ := newSettingsRouter(t, false)
	req := httptest.NewRequest(http.MethodPatch, "/settings", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSettingsHandler_PatchSettings_OK(t *testing.T) {
	r, _ := newSettingsRouter(t, false)
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
	r, _ := newSettingsRouter(t, false)
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
	r, _ := newSettingsRouter(t, false)
	req := httptest.NewRequest(http.MethodPost, "/settings/media/reset-index", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSettingsHandler_PostMediaReset_OK(t *testing.T) {
	r, jobStore := newSettingsRouter(t, false)
	body := `{"confirm_destructive": true}`
	req := httptest.NewRequest(http.MethodPost, "/settings/media/reset-index", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	// Attendre la fin de la goroutine background avant que t.TempDir() ne
	// tente de cleanup le dossier (sinon RemoveAll échoue intermittemment
	// car jobs.json est encore en cours d'écriture).
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if id, ok := resp["job_id"].(string); ok && id != "" {
		pollJobSucceeded(t, jobStore, id)
	}
}

// ─── Validation des nouveaux champs Analyse ───────────────────────────────────

func patch(t *testing.T, r *chi.Mux, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/settings", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSettingsHandler_Patch_SessionGapNegative_400(t *testing.T) {
	r, _ := newSettingsRouter(t, false)
	w := patch(t, r, `{"session_gap_minutes": -1}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for negative gap, got %d", w.Code)
	}
}

func TestSettingsHandler_Patch_SessionGapZero_OK(t *testing.T) {
	r, _ := newSettingsRouter(t, false)
	w := patch(t, r, `{"session_gap_minutes": 0}`)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for gap=0, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSettingsHandler_Patch_InvalidBadgeSensitivity_400(t *testing.T) {
	r, _ := newSettingsRouter(t, false)
	w := patch(t, r, `{"outcome_badge_sensitivity": "invalid"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid sensitivity, got %d", w.Code)
	}
}

func TestSettingsHandler_Patch_ValidBadgeSensitivityStrict_200(t *testing.T) {
	r, _ := newSettingsRouter(t, false)
	w := patch(t, r, `{"outcome_badge_sensitivity": "strict"}`)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for sensitivity=strict, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSettingsHandler_Patch_ValidTeamChangeModeFriends_200(t *testing.T) {
	r, _ := newSettingsRouter(t, false)
	w := patch(t, r, `{"session_team_change_mode": "friends"}`)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for team_change_mode=friends, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSettingsHandler_Patch_InvalidTeamChangeMode_400(t *testing.T) {
	r, _ := newSettingsRouter(t, false)
	w := patch(t, r, `{"session_team_change_mode": "random"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid team_change_mode, got %d", w.Code)
	}
}

func TestSettingsHandler_Patch_AnalyseRoundTrip(t *testing.T) {
	r, _ := newSettingsRouter(t, false)

	// PATCH
	patchBody := `{
		"session_gap_minutes": 90,
		"session_team_change_mode": "ignore",
		"session_split_on_ranked_change": true,
		"outcome_badge_sensitivity": "relaxed",
		"outcome_exclude_bot_matches_from_badges": false,
		"outcome_exclude_bot_matches_from_records": true
	}`
	pw := patch(t, r, patchBody)
	if pw.Code != http.StatusOK {
		t.Fatalf("PATCH failed (%d): %s", pw.Code, pw.Body.String())
	}

	// GET → vérifier que les valeurs sont persistées.
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	gw := httptest.NewRecorder()
	r.ServeHTTP(gw, req)
	if gw.Code != http.StatusOK {
		t.Fatalf("GET failed (%d): %s", gw.Code, gw.Body.String())
	}

	var got map[string]interface{}
	if err := json.Unmarshal(gw.Body.Bytes(), &got); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if got["session_gap_minutes"] != float64(90) {
		t.Errorf("session_gap_minutes = %v, want 90", got["session_gap_minutes"])
	}
	if got["session_team_change_mode"] != "ignore" {
		t.Errorf("session_team_change_mode = %v, want 'ignore'", got["session_team_change_mode"])
	}
	if got["outcome_badge_sensitivity"] != "relaxed" {
		t.Errorf("outcome_badge_sensitivity = %v, want 'relaxed'", got["outcome_badge_sensitivity"])
	}
}

func TestSettingsHandler_PostRecalculateSessions_Accepted(t *testing.T) {
	r, jobStore := newSettingsRouter(t, false)

	req := httptest.NewRequest(http.MethodPost, "/settings/sessions/recalculate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if body["job_id"] == "" || body["job_id"] == nil {
		t.Errorf("expected non-empty job_id in response, got %v", body)
	}
	if body["status"] != "queued" {
		t.Errorf("expected status=queued, got %v", body["status"])
	}
	if body["job_type"] != "sessions_recalculate" {
		t.Errorf("expected job_type=sessions_recalculate, got %v", body["job_type"])
	}

	// Attendre la fin du job background avant que t.TempDir() cleanup le dossier.
	if id, ok := body["job_id"].(string); ok && id != "" {
		pollJobSucceeded(t, jobStore, id)
	}
}
