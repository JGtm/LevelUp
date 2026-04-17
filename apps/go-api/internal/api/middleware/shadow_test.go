// Package middleware_test — shadow_test.go : tests unitaires du middleware Shadow.
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/api/middleware"
)

func TestShadow_EmptyPythonURL_Passthrough(t *testing.T) {
	// Sans PythonURL, le middleware passe directement au handler suivant.
	shadow := middleware.Shadow(middleware.ShadowConfig{PythonURL: ""})
	handler := shadow(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Fatalf("expected body 'ok', got %q", w.Body.String())
	}
}

func TestShadow_WithPythonURL_DoesNotBlockResponse(t *testing.T) {
	// Avec une URL Python invalide (pas de serveur), la réponse Go doit rester normale.
	// Le shadow call échoue silencieusement en goroutine.
	shadow := middleware.Shadow(middleware.ShadowConfig{PythonURL: "http://127.0.0.1:19999"})
	handler := shadow(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("main"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("shadow should not block response: got %d", w.Code)
	}
}
