// Package middleware — security_headers_test.go : couverture du middleware
// SecurityHeaders (revue P0 2026-06-02 ; HSTS par schéma 2026-06-08).
package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
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

// wantMinimalCSP : politique minimale (registre sécurité C7-2, lot L9-6) — ne restreint ni les
// scripts ni les styles du SPA.
const wantMinimalCSP = "object-src 'none'; base-uri 'self'; frame-ancestors 'none'"

func TestSecurityHeaders_MinimalCSPOnEveryResponse(t *testing.T) {
	r := chi.NewRouter()
	r.Use(chimiddleware.ClientIPFromRemoteAddr) // clé du limiteur, comme le routeur réel
	r.Use(SecurityHeaders(false))
	r.Use(RateLimit(false, 1))
	r.Get("/ok", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/boom", func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "boom", http.StatusInternalServerError) })

	// Une adresse par cas : le limiteur (1 requête par minute) ne doit pas les confondre.
	cases := []struct {
		name, path, addr string
		want             int
	}{
		{"200", "/ok", "203.0.113.1:1234", http.StatusOK},
		{"500", "/boom", "203.0.113.2:1234", http.StatusInternalServerError},
		{"404", "/absent", "203.0.113.3:1234", http.StatusNotFound},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		req.RemoteAddr = tc.addr
		r.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Fatalf("%s : statut %d, attendu %d", tc.name, rec.Code, tc.want)
		}
		if got := rec.Header().Get("Content-Security-Policy"); got != wantMinimalCSP {
			t.Errorf("%s : Content-Security-Policy = %q, attendu %q", tc.name, got, wantMinimalCSP)
		}
	}

	// 429 : le refus du limiteur porte l'en-tête comme toute autre réponse.
	var last *httptest.ResponseRecorder
	for i := 0; i < 3; i++ {
		last = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ok", nil)
		req.RemoteAddr = "198.51.100.9:4321"
		r.ServeHTTP(last, req)
	}
	if last.Code != http.StatusTooManyRequests {
		t.Fatalf("limiteur : statut %d, attendu 429", last.Code)
	}
	if got := last.Header().Get("Content-Security-Policy"); got != wantMinimalCSP {
		t.Errorf("429 : Content-Security-Policy = %q, attendu %q", got, wantMinimalCSP)
	}
}

func TestSecurityHeaders_CSPDoesNotRestrictScriptsOrStyles(t *testing.T) {
	for _, directive := range []string{"default-src", "script-src", "style-src", "img-src", "connect-src", "font-src"} {
		if strings.Contains(wantMinimalCSP, directive) {
			t.Fatalf("la CSP minimale ne doit pas porter %s", directive)
		}
	}
	h := SecurityHeaders(false)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	for _, directive := range []string{"default-src", "script-src", "style-src"} {
		if strings.Contains(rec.Header().Get("Content-Security-Policy"), directive) {
			t.Errorf("la CSP posée restreint %s", directive)
		}
	}
}
