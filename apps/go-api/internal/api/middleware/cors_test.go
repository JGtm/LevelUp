// Package middleware_test — cors_test.go : tests unitaires du middleware CORS.
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/api/middleware"
)

func TestCORS_AllowedOriginHeaderPresent(t *testing.T) {
	corsMW := middleware.CORS([]string{"http://localhost:5173"})
	handler := corsMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatal("expected CORS header on allowed origin request")
	}
}

func TestCORS_PrefightOptionsRequest(t *testing.T) {
	corsMW := middleware.CORS([]string{"http://localhost:5173"})
	handler := corsMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/data", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// OPTIONS preflight → 200 ou 204
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("expected 200 or 204 for preflight, got %d", w.Code)
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	corsMW := middleware.CORS([]string{"http://localhost:5173"})
	handler := corsMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	// Pas d'header Origin → requête same-origin, passe toujours
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for no-origin request, got %d", w.Code)
	}
}

// TestCORS_ExposeLesEntetesLusParLeClient — les en-tetes de reponse que le client lit
// (`response.headers.get`) doivent figurer dans Access-Control-Expose-Headers, sans quoi un
// front cross-origin recoit null : le garde anti-fuite cross-titre (ResolvedTitleHeader) et
// le badge admin de version de schema (ReplayLatestSchemaHeader, revue de vague 2026-09-12)
// deviendraient muets sans qu'aucun test ne le voie.
func TestCORS_ExposeLesEntetesLusParLeClient(t *testing.T) {
	corsMW := middleware.CORS([]string{"http://localhost:5173"})
	handler := corsMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// rs/cors canonise la casse des noms (X-Levelup-...) : comparaison insensible a la casse.
	exposed := strings.ToLower(w.Header().Get("Access-Control-Expose-Headers"))
	for _, h := range []string{middleware.ResolvedTitleHeader, middleware.ReplayLatestSchemaHeader} {
		if !strings.Contains(exposed, strings.ToLower(h)) {
			t.Errorf("Access-Control-Expose-Headers = %q : %s manque", exposed, h)
		}
	}
}
