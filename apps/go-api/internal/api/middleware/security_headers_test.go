// Package middleware — security_headers_test.go : couverture du middleware
// SecurityHeaders (revue P0 2026-06-02).
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders_SetsBaseHeaders(t *testing.T) {
	h := SecurityHeaders(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	want := map[string]string{
		"X-Content-Type-Options":     "nosniff",
		"X-Frame-Options":            "DENY",
		"Referrer-Policy":            "strict-origin-when-cross-origin",
		"Cross-Origin-Opener-Policy": "same-origin",
	}
	for k, v := range want {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("%s = %q, attendu %q", k, got, v)
		}
	}
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS ne doit pas être posé hors production (hsts=false)")
	}
}

func TestSecurityHeaders_HSTSOnlyInProduction(t *testing.T) {
	h := SecurityHeaders(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := rec.Header().Get("Strict-Transport-Security"); got == "" {
		t.Error("HSTS doit être posé quand hsts=true (production/TLS)")
	}
}
