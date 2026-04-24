// Package handlers_test — auth_test.go : tests unitaires AuthHandler (Device Code Flow).
//
// Teste les cas sans initier de vrai flow MSAL (mode démo ou stubs).
package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/session"
)

// newAuthRouter crée un routeur de test avec le stub provider (pas de MSAL réel).
func newAuthRouter(t *testing.T, demoMode bool) (*chi.Mux, *session.Store) {
	return newAuthRouterWithProvider(t, demoMode, &stubTokenProvider{})
}

func newAuthRouterWithProvider(t *testing.T, demoMode bool, provider auth_platform.TokenProvider) (*chi.Mux, *session.Store) {
	t.Helper()
	dir := t.TempDir()
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	attempts := auth_platform.NewAttemptStore()
	h := handlers.NewAuthHandler(sessStore, attempts, demoMode, provider)

	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, false))
	r.Post("/auth/device-flow/start", h.StartDeviceFlow)
	r.Get("/auth/device-flow/{attempt_id}", h.GetDeviceFlowStatus)
	return r, sessStore
}

func TestAuthHandler_StartDeviceFlow_DemoMode(t *testing.T) {
	r, _ := newAuthRouter(t, true)
	req := httptest.NewRequest(http.MethodPost, "/auth/device-flow/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Mode démo → 422
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 in demo mode, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_GetDeviceFlowStatus_NotFound(t *testing.T) {
	r, _ := newAuthRouter(t, false)
	req := httptest.NewRequest(http.MethodGet, "/auth/device-flow/nonexistent-attempt", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_GetDeviceFlowStatus_Expired(t *testing.T) {
	r, _ := newAuthRouter(t, false)
	// Appeler avec un attempt_id qui n'existe pas → 404 aussi
	req := httptest.NewRequest(http.MethodGet, "/auth/device-flow/expired-attempt-id-12345", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// TestAuthHandler_StartDeviceFlow_ProviderError vérifie qu'un échec du provider
// est propagé comme HTTP 500.
func TestAuthHandler_StartDeviceFlow_ProviderError(t *testing.T) {
	provider := &stubTokenProvider{initFlowErr: errors.New("msal network error")}
	r, _ := newAuthRouterWithProvider(t, false, provider)

	req := httptest.NewRequest(http.MethodPost, "/auth/device-flow/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on provider error, got %d: %s", w.Code, w.Body.String())
	}
}

// TestAuthHandler_StartDeviceFlow_Success vérifie qu'un InitDeviceFlow réussi
// retourne 200 avec user_code et verification_url.
func TestAuthHandler_StartDeviceFlow_Success(t *testing.T) {
	// ExpiresIn=1 : le contexte du polling expirera après 1s (pas de vrai MSAL).
	flow := auth_platform.NewStubDeviceFlow("TEST42", "https://microsoft.com/devicelogin", "Entrez TEST42", 1, "msal")
	provider := &stubTokenProvider{initFlowFlow: flow}
	r, _ := newAuthRouterWithProvider(t, false, provider)

	req := httptest.NewRequest(http.MethodPost, "/auth/device-flow/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on success, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("réponse JSON invalide: %v", err)
	}
	if body["user_code"] != "TEST42" {
		t.Errorf("attendu user_code=TEST42, got %v", body["user_code"])
	}
	if body["verification_uri"] == "" {
		t.Errorf("verification_uri absent de la réponse")
	}
}
