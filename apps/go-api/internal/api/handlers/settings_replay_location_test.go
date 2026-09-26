// Package handlers_test — settings_replay_location_test.go : le réglage « où se
// construit un rejeu » au PATCH /settings.
//
// LE TEST QUI COMPTE EST CELUI DU REFUS EN PRODUCTION. Un réglage accepté puis
// ignoré à l'exécution est pire que pas de réglage du tout : l'admin croirait
// avoir activé le décodage sur le VPS web, qui ne décode jamais.
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

// newSettingsRouterEnv monte le handler de réglages pour un environnement donné
// ("" = développement, "production" = prod).
func newSettingsRouterEnv(t *testing.T, environment string) *chi.Mux {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{
		Environment:    environment,
		DBProfilesPath: filepath.Join(dir, "db_profiles.json"),
	}
	h := handlers.NewSettingsHandler(cfg,
		settings_platform.NewStore(filepath.Join(dir, "app_settings.json")),
		jobs.NewStore(filepath.Join(dir, "jobs.json")))
	r := chi.NewRouter()
	h.Mount(r)
	return r
}

// patchLocation envoie un PATCH du seul réglage de lieu de construction.
func patchLocation(t *testing.T, r *chi.Mux, value string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"replay_build_location": value})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestPatchReplayBuildLocation_ValeursAcceptees : les trois valeurs, plus le vide
// (« défaut de l'instance »), passent en développement.
func TestPatchReplayBuildLocation_ValeursAcceptees(t *testing.T) {
	r := newSettingsRouterEnv(t, "")
	for _, v := range []string{"local", "worker", "off", ""} {
		if w := patchLocation(t, r, v); w.Code != http.StatusOK {
			t.Errorf("valeur %q : status = %d (attendu 200) body=%s", v, w.Code, w.Body.String())
		}
	}
}

// TestPatchReplayBuildLocation_ValeurInconnue_400 : rien d'inventé n'entre en
// configuration — sinon la valeur dormirait jusqu'au premier cycle.
func TestPatchReplayBuildLocation_ValeurInconnue_400(t *testing.T) {
	w := patchLocation(t, newSettingsRouterEnv(t, ""), "vps-de-guillaume")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d (attendu 400) body=%s", w.Code, w.Body.String())
	}
}

// TestPatchReplayBuildLocation_LocalEnProduction_400 : le refus EXPLICITE, avec
// son motif — « le VPS web ne décode jamais un film ».
func TestPatchReplayBuildLocation_LocalEnProduction_400(t *testing.T) {
	r := newSettingsRouterEnv(t, "production")
	w := patchLocation(t, r, "local")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d (attendu 400) body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("ne d")) {
		t.Errorf("le motif du refus doit être servi à l'admin, body=%s", w.Body.String())
	}
	// Et les deux autres valeurs restent ouvertes en production.
	for _, v := range []string{"worker", "off"} {
		if got := patchLocation(t, r, v); got.Code != http.StatusOK {
			t.Errorf("valeur %q en production : status = %d (attendu 200) body=%s", v, got.Code, got.Body.String())
		}
	}
}
