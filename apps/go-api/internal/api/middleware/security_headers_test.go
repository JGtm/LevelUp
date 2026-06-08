// Package middleware — security_headers_test.go : couverture du middleware
// SecurityHeaders (revue P0 2026-06-02 ; HSTS par schéma 2026-06-08).
package middleware

import (
	"crypto/tls"
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
		t.Error("HSTS ne doit pas être posé sur une requête HTTP nue")
	}
}

func TestSecurityHeaders_HSTSOnHTTPSViaTLS(t *testing.T) {
	h := SecurityHeaders(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.TLS = &tls.ConnectionState{} // simule une connexion TLS native
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Strict-Transport-Security"); got == "" {
		t.Error("HSTS doit être posé sur une requête HTTPS (TLS natif)")
	}
}

func TestSecurityHeaders_HSTSViaProxyOnlyWhenTrusted(t *testing.T) {
	// X-Forwarded-Proto: https honoré seulement si trustProxy=true.
	mkReq := func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		return req
	}

	recTrusted := httptest.NewRecorder()
	SecurityHeaders(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(recTrusted, mkReq())
	if recTrusted.Header().Get("Strict-Transport-Security") == "" {
		t.Error("HSTS attendu avec X-Forwarded-Proto=https et trustProxy=true")
	}

	recUntrusted := httptest.NewRecorder()
	SecurityHeaders(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(recUntrusted, mkReq())
	if recUntrusted.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS ne doit PAS être posé via X-Forwarded-Proto si trustProxy=false (anti-spoof)")
	}
}
