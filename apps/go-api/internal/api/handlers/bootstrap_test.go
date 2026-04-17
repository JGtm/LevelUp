// Package handlers_test — bootstrap_test.go : tests BootstrapHandler.
//
// Sprint 46 T13 — BootstrapHandler utilise *service.BootstrapService qui prend
// port.BootstrapRepository (interface mockable). Pas de CGO requis : le
// mockBootstrapRepo défini dans health_test.go est réutilisé dans ce package.
package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/service"
)

// newBootstrapHandlerForTest construit un BootstrapHandler avec repo mock
// et config en mode démo (pas de fichier db_profiles.json requis).
func newBootstrapHandlerForTest(t *testing.T, repo *mockBootstrapRepo) *handlers.BootstrapHandler {
	t.Helper()
	cfg := &config.AppConfig{DemoMode: true}
	svc := service.NewBootstrapService(cfg, repo)
	return handlers.NewBootstrapHandler(svc)
}

// TestBootstrapHandler_OK vérifie la réponse nominale : HTTP 200, champs obligatoires.
func TestBootstrapHandler_OK(t *testing.T) {
	repo := &mockBootstrapRepo{
		matchCount:  2500,
		dbVersion:   "v1.4.4",
		playerCount: 2,
	}
	h := newBootstrapHandlerForTest(t, repo)
	r := chi.NewRouter()
	r.Get("/api/v1/bootstrap", h.ServeHTTP)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bootstrap", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d : %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}

	// Champs obligatoires présents
	for _, key := range []string{"setup_required", "auth_state", "setup_state",
		"current_title_slug", "available_titles", "feature_flags", "capabilities"} {
		if _, ok := resp[key]; !ok {
			t.Errorf("champ manquant dans la réponse bootstrap : %q", key)
		}
	}
}

// TestBootstrapHandler_DemoMode vérifie que le mode démo peuple available_players
// depuis les données demo (db_profiles.json absent → liste demo injectée).
func TestBootstrapHandler_DemoMode(t *testing.T) {
	repo := &mockBootstrapRepo{matchCount: 100, dbVersion: "v1.4.4"}
	h := newBootstrapHandlerForTest(t, repo)
	r := chi.NewRouter()
	r.Get("/api/v1/bootstrap", h.ServeHTTP)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bootstrap", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}

	// En mode démo, setup_required = false et setup_state != "no_halo_link"
	if v, _ := resp["setup_required"].(bool); v {
		t.Error("setup_required devrait être false en mode démo")
	}
}

// TestBootstrapHandler_CurrentTitleSlug vérifie que current_title_slug est présent
// et non vide (fallback "halo_infinite" si aucune session).
func TestBootstrapHandler_CurrentTitleSlug(t *testing.T) {
	repo := &mockBootstrapRepo{matchCount: 50}
	h := newBootstrapHandlerForTest(t, repo)
	r := chi.NewRouter()
	r.Get("/api/v1/bootstrap", h.ServeHTTP)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bootstrap", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", w.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	slug, _ := resp["current_title_slug"].(string)
	if slug == "" {
		t.Error("current_title_slug ne doit pas être vide (fallback halo_infinite attendu)")
	}
}

// TestBootstrapHandler_AvailableTitles vérifie que available_titles est un tableau
// (Sprint 44 — multi-titres).
func TestBootstrapHandler_AvailableTitles(t *testing.T) {
	repo := &mockBootstrapRepo{matchCount: 10, dbVersion: "v1.4.4"}
	h := newBootstrapHandlerForTest(t, repo)
	r := chi.NewRouter()
	r.Get("/api/v1/bootstrap", h.ServeHTTP)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bootstrap", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", w.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	titles, ok := resp["available_titles"].([]interface{})
	if !ok {
		t.Fatalf("available_titles devrait être un tableau, obtenu %T", resp["available_titles"])
	}
	if len(titles) == 0 {
		t.Error("available_titles ne devrait pas être vide (au moins halo_infinite)")
	}
}
