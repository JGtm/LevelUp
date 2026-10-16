//go:build cgo

// security_headers_routes_test.go — la CSP minimale (object-src, base-uri, frame-ancestors)
// accompagne les réponses du routeur réel : page, 404, fichiers statiques, API, sondes
// (registre sécurité C7-2, lot L9-6).
package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const minimalCSP = "object-src 'none'; base-uri 'self'; frame-ancestors 'none'"

func TestRouter_MinimalCSPOnEveryKindOfResponse(t *testing.T) {
	router := buildTestRouter(t)
	paths := []string{
		"/",                         // page du SPA (ou repli)
		"/chemin-inexistant",        // 404
		"/static/medals/absent.png", // fichiers statiques
		"/api/v1/bootstrap",         // API
		"/api/v1/route-inexistante", // 404 de l'API
		"/health",                   // sonde
		"/healthz",
	}
	for _, path := range paths {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if got := rec.Header().Get("Content-Security-Policy"); got != minimalCSP {
			t.Errorf("GET %s (statut %d) : Content-Security-Policy = %q, attendu %q", path, rec.Code, got, minimalCSP)
		}
	}
}
