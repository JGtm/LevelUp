// Package middleware — loopback_preserve_test.go : non-régression de la faille
// RealIP/LoopbackOnly (revue P0 2026-06-02). PreserveRemoteAddr capture le vrai
// peer TCP avant toute réécriture de RemoteAddr, ce qui empêche un client externe
// d'usurper 127.0.0.1 via un en-tête X-Real-IP pour atteindre les endpoints /_diag.
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestLoopbackOnly_PreserveRemoteAddr_DefeatsHeaderSpoofing : un peer non-loopback
// ne doit pas contourner LoopbackOnly même si RemoteAddr a été réécrit en
// 127.0.0.1 par un middleware en amont (simulant chi RealIP sur en-tête falsifié).
func TestLoopbackOnly_PreserveRemoteAddr_DefeatsHeaderSpoofing(t *testing.T) {
	reached := false
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})

	// spoofRealIP réécrit RemoteAddr en loopback, comme le ferait chi RealIP avec
	// un en-tête « X-Real-IP: 127.0.0.1 » envoyé par un attaquant.
	spoofRealIP := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.RemoteAddr = "127.0.0.1:1234"
			next.ServeHTTP(w, r)
		})
	}
	h := PreserveRemoteAddr(spoofRealIP(LoopbackOnly(final)))

	req := httptest.NewRequest(http.MethodGet, "/_diag/x", nil)
	req.RemoteAddr = "203.0.113.7:55000" // vrai peer externe
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if reached {
		t.Fatal("LoopbackOnly a laissé passer un peer externe usurpant 127.0.0.1")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("attendu 403, obtenu %d", rec.Code)
	}
}

// TestLoopbackOnly_RealLoopbackAllowed : un vrai peer loopback passe.
func TestLoopbackOnly_RealLoopbackAllowed(t *testing.T) {
	reached := false
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	h := PreserveRemoteAddr(LoopbackOnly(final))

	req := httptest.NewRequest(http.MethodGet, "/_diag/x", nil)
	req.RemoteAddr = "127.0.0.1:4444"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !reached || rec.Code != http.StatusOK {
		t.Errorf("vrai loopback devrait passer: reached=%v code=%d", reached, rec.Code)
	}
}

// TestLoopbackOnly_NoPreserveFallsBackToRemoteAddr : sans PreserveRemoteAddr en
// amont, LoopbackOnly retombe sur r.RemoteAddr (compatibilité ascendante).
func TestLoopbackOnly_NoPreserveFallsBackToRemoteAddr(t *testing.T) {
	reached := false
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
	})
	h := LoopbackOnly(final)

	req := httptest.NewRequest(http.MethodGet, "/_diag/x", nil)
	req.RemoteAddr = "10.0.0.5:40000" // non-loopback, pas de capture amont
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if reached || rec.Code != http.StatusForbidden {
		t.Errorf("sans PreserveRemoteAddr, un peer non-loopback doit être refusé: reached=%v code=%d", reached, rec.Code)
	}
}
