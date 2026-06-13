// Package middleware_test — rate_limit_test.go : tests unitaires du middleware RateLimit.
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/api/middleware"
)

func TestRateLimit_AllowsRequests(t *testing.T) {
	// En mode démo la limite est 10× plus haute — les tests passent toujours.
	rl := middleware.RateLimit(true, 0)
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
	normalRL := middleware.RateLimit(false, 0)
	demoRL := middleware.RateLimit(true, 0)

	if normalRL == nil || demoRL == nil {
		t.Fatal("RateLimit middleware should not be nil")
	}
}

// TestRateLimitMiddleware_DefaultWhenNonPositive vérifie que rpm <= 0 retombe sur
// le défaut RateLimitRequests : envoyer (défaut + marge) requêtes doit en rejeter.
func TestRateLimitMiddleware_DefaultWhenNonPositive(t *testing.T) {
	rl := middleware.RateLimit(false, 0)
	handler := rl(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rejected := 0
	total := middleware.RateLimitRequests + 10
	for i := 0; i < total; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.9:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			rejected++
		}
	}
	if rejected == 0 {
		t.Errorf("expected rejections past default limit %d", middleware.RateLimitRequests)
	}
}

func TestRateLimitMiddleware_Rejects_Over_Limit(t *testing.T) {
	// Limite explicite basse (50 req/min) pour un test déterministe et rapide.
	// Envoyer 60 requêtes : certaines doivent être rejetées.
	const limit = 50
	rl := middleware.RateLimit(false, limit)
	handler := rl(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rejected := 0
	for i := 0; i < limit+10; i++ {
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

// TestRateLimitMiddleware_StaticBypass vérifie que /static/* n'est jamais rate-limité.
// La home page peut émettre 100+ URLs uniques d'images statiques (maps, médailles,
// citations, ranks…) — le rate limit applicatif ne doit pas les claquer en 429.
func TestRateLimitMiddleware_StaticBypass(t *testing.T) {
	// Limite basse (50) ET 200 requêtes : si le bypass fonctionne, aucune n'est 429.
	rl := middleware.RateLimit(false, 50)
	handler := rl(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 200; i++ {
		req := httptest.NewRequest(http.MethodGet, "/static/maps/halo_infinite/Aquarius.png", nil)
		req.RemoteAddr = "10.0.0.2:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("/static request %d returned %d, expected 200 (no rate limit on static)", i, w.Code)
		}
	}
}

// TestRateLimitMiddleware_NonStaticStillLimited s'assure que les exemptions
// (/static/*, /api/v1/assets/*) ne débordent pas sur les autres préfixes.
func TestRateLimitMiddleware_NonStaticStillLimited(t *testing.T) {
	const limit = 50
	rl := middleware.RateLimit(false, limit)
	handler := rl(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rejected := 0
	for i := 0; i < limit+10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/players/test/pages/home", nil)
		req.RemoteAddr = "10.0.0.3:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			rejected++
		}
	}
	if rejected == 0 {
		t.Error("expected /api/* requests to be rate-limited")
	}
}

// TestRateLimitMiddleware_AssetsBypass vérifie que /api/v1/assets/* n'est jamais
// rate-limité. La page Season Pass tire 100+ URLs uniques (images des rewards
// par tier, par pass) servies par le resolver local-first — le rate limit ne
// doit pas les claquer en 429 (cf. bug Héritage 2026-05-06).
func TestRateLimitMiddleware_AssetsBypass(t *testing.T) {
	// Limite basse (50) ET 200 requêtes : si le bypass fonctionne, aucune n'est 429.
	rl := middleware.RateLimit(false, 50)
	handler := rl(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 200; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/battlepass/tier/progression/Inventory/Armor/Helmets/x.png", nil)
		req.RemoteAddr = "10.0.0.4:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("/api/v1/assets request %d returned %d, expected 200 (no rate limit on assets)", i, w.Code)
		}
	}
}
