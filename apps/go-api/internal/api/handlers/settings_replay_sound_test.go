// Package handlers_test — settings_replay_sound_test.go : bornes des réglages « Sons du
// rejeu 2D » et persistance par aller-retour PATCH puis GET.
package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// patchSettings joue un PATCH et rend le code de statut.
func patchSettings(t *testing.T, r http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/settings", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// Une valeur hors bornes est REFUSÉE, pas ramenée en silence : un curseur qui affiche 150
// alors que le serveur a retenu 100 ment à l'opérateur.
func TestSettingsHandler_Patch_ReplaySoundHorsBornes_400(t *testing.T) {
	r, _ := newSettingsRouter(t, false)
	for _, corps := range []string{
		`{"replay_sound_variation_percent": 150}`,
		`{"replay_sound_variation_percent": -1}`,
		`{"replay_sound_distance_percent": 101}`,
		`{"replay_sound_distance_percent": -20}`,
	} {
		if w := patchSettings(t, r, corps); w.Code != http.StatusBadRequest {
			t.Errorf("PATCH %s : statut %d, attendu 400 (%s)", corps, w.Code, w.Body.String())
		}
	}
}

// Les deux bornes exactes sont valides : 0 = effet coupé, 100 = effet plein.
func TestSettingsHandler_Patch_ReplaySoundBornesValides(t *testing.T) {
	r, _ := newSettingsRouter(t, false)
	for _, corps := range []string{
		`{"replay_sound_variation_percent": 0}`,
		`{"replay_sound_variation_percent": 100}`,
		`{"replay_sound_distance_percent": 0}`,
		`{"replay_sound_distance_percent": 100}`,
	} {
		if w := patchSettings(t, r, corps); w.Code != http.StatusOK {
			t.Errorf("PATCH %s : statut %d, attendu 200 (%s)", corps, w.Code, w.Body.String())
		}
	}
}

// Aller-retour complet : ce qui est écrit est relu. Le défaut au départ vaut 100/0.
func TestSettingsHandler_Patch_ReplaySoundRoundTrip(t *testing.T) {
	r, _ := newSettingsRouter(t, false)

	depart := getSettingsBody(t, r)
	if depart["replay_sound_variation_percent"] != float64(100) {
		t.Errorf("variation au départ = %v, attendu 100", depart["replay_sound_variation_percent"])
	}
	if depart["replay_sound_distance_percent"] != float64(0) {
		t.Errorf("distance au départ = %v, attendu 0", depart["replay_sound_distance_percent"])
	}

	if w := patchSettings(t, r,
		`{"replay_sound_variation_percent": 35, "replay_sound_distance_percent": 80}`); w.Code != http.StatusOK {
		t.Fatalf("PATCH : statut %d (%s)", w.Code, w.Body.String())
	}

	apres := getSettingsBody(t, r)
	if apres["replay_sound_variation_percent"] != float64(35) {
		t.Errorf("variation relue = %v, attendu 35", apres["replay_sound_variation_percent"])
	}
	if apres["replay_sound_distance_percent"] != float64(80) {
		t.Errorf("distance relue = %v, attendu 80", apres["replay_sound_distance_percent"])
	}
}

func getSettingsBody(t *testing.T, r http.Handler) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /settings : statut %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("GET /settings : corps illisible : %v", err)
	}
	return body
}
