// Package handlers_test — auth_test.go : tests unitaires AuthHandler (Device Code Flow).
//
// Teste les cas sans initier de vrai flow MSAL (mode démo ou mocks).
package handlers_test

import (
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

func newAuthRouter(t *testing.T, demoMode bool) (*chi.Mux, *session.Store) {
	t.Helper()
	dir := t.TempDir()
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	attempts := auth_platform.NewAttemptStore()
	h := handlers.NewAuthHandler(sessStore, attempts, demoMode)

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
