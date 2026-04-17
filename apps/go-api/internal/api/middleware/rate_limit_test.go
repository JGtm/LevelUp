// Package middleware_test — rate_limit_test.go : tests unitaires du middleware RateLimit.
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/api/middleware"
)

func TestRateLimit_AllowsRequests(t *testing.T) {
	// En mode démo la limite est 10× plus haute (1200 req/min) — les tests passent toujours.
	rl := middleware.RateLimit(true)
	handler := rl(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRateLimitMiddleware_Constants(t *testing.T) {
	if middleware.RateLimitRequests <= 0 {
		t.Fatalf("RateLimitRequests should be positive, got %d", middleware.RateLimitRequests)
	}
	if middleware.RateLimitWindow <= 0 {
		t.Fatal("RateLimitWindow should be positive")
	}
}

func TestRateLimitMiddleware_DemoMode_HigherLimit(t *testing.T) {
	// Juste vérifier que les deux modes créent bien un middleware valide.
	normalRL := middleware.RateLimit(false)
	demoRL := middleware.RateLimit(true)

	if normalRL == nil || demoRL == nil {
		t.Fatal("RateLimit middleware should not be nil")
	}
}
func TestRateLimitMiddleware_Rejects_Over_Limit(t *testing.T) {
	// mode normal = 120 req/min. Envoyer 125 requêtes, certaines doivent être rejetées.
	rl := middleware.RateLimit(false)
	handler := rl(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rejected := 0
	for i := 0; i < 125; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			rejected++
		}
	}

	if rejected == 0 {
		t.Error("expected some requests to be rejected after exceeding rate limit")
	}
}
