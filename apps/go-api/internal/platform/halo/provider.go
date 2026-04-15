// Package halo — provider.go : client HTTP pour l'API Halo Infinite.
//
// Ce fichier contient le squelette du provider Halo Infinite avec :
//   - Rate limiting (max 60 req/min) via token bucket.
//   - Retry exponentiel (max 3 tentatives).
//   - HTTP client configuré (timeout 15s).
//
// Les appels live (Battle Pass, Challenges) nécessitent une session Halo active.
// Ils seront implémentés en Sprint 15 (Device Code Flow + MSAL Go).
// Pour l'instant, ces endpoints retournent available=false, error_hint="auth_required".
package halo

import (
	"context"
	"net/http"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// Rate limiter (token bucket)
// ---------------------------------------------------------------------------

// rateLimiter implémente un token bucket pour limiter les requêtes.
// Capacité : maxTokens, rechargement : refillRate tokens/seconde.
type rateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens par seconde
	lastRefill time.Time
}

// newRateLimiter crée un limiter avec une capacité de maxPerMinute req/min.
func newRateLimiter(maxPerMinute int) *rateLimiter {
	return &rateLimiter{
		tokens:     float64(maxPerMinute),
		maxTokens:  float64(maxPerMinute),
		refillRate: float64(maxPerMinute) / 60.0,
		lastRefill: time.Now(),
	}
}

// Consume tente de consommer un token. Retourne false si le bucket est vide.
// Ne bloque pas — l'appelant doit boucler avec Wait.
func (r *rateLimiter) tryConsume() (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens += elapsed * r.refillRate
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}
	r.lastRefill = now

	if r.tokens >= 1.0 {
		r.tokens--
		return true, 0
	}
	wait := time.Duration((1.0-r.tokens)/r.refillRate*1000) * time.Millisecond
	return false, wait
}

// Wait bloque jusqu'à ce qu'un token soit disponible ou que ctx soit annulé.
func (r *rateLimiter) Wait(ctx context.Context) error {
	for {
		ok, wait := r.tryConsume()
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

// ---------------------------------------------------------------------------
// HaloProvider
// ---------------------------------------------------------------------------

// HaloProvider est le client HTTP pour l'API Halo Infinite (343 Industries).
// Thread-safe : une seule instance par processus.
type HaloProvider struct {
	client     *http.Client
	limiter    *rateLimiter
	maxRetries int
}

// DefaultHaloProvider est l'instance globale du provider (60 req/min, 3 retries).
var DefaultHaloProvider = NewHaloProvider()

// NewHaloProvider crée un provider Halo avec les paramètres par défaut.
// Rate limit : 60 req/min. Timeout HTTP : 15s. Max retries : 3.
func NewHaloProvider() *HaloProvider {
	return &HaloProvider{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		limiter:    newRateLimiter(60),
		maxRetries: 3,
	}
}

// ---------------------------------------------------------------------------
// Endpoints best-effort (Sprint 15 : implémentation live)
// ---------------------------------------------------------------------------

// GetBattlePass tente de récupérer les infos Battle Pass.
// Retourne available=false, error_hint="auth_required" tant que l'auth n'est pas portée.
// TODO Sprint 15 : implémenter l'appel live avec spartan_token.
func (p *HaloProvider) GetBattlePass(_ context.Context) domain.BattlePassResponse {
	hint := "auth_required"
	return domain.BattlePassResponse{Available: false, ErrorHint: &hint}
}

// GetChallenges tente de récupérer les défis actifs.
// Retourne available=false, error_hint="auth_required" tant que l'auth n'est pas portée.
// TODO Sprint 15 : implémenter l'appel live avec spartan_token.
func (p *HaloProvider) GetChallenges(_ context.Context) domain.ChallengesResponse {
	hint := "auth_required"
	return domain.ChallengesResponse{Available: false, ErrorHint: &hint}
}

// ---------------------------------------------------------------------------
// doRequest — appel HTTP avec retry exponentiel (usage interne Sprint 15)
// ---------------------------------------------------------------------------

// doRequest effectue un appel HTTP avec rate limiting et retry exponentiel.
// Politiques de retry : 429 Too Many Requests, 503 Service Unavailable.
// Exponentiel : 1s, 2s, 4s.
func (p *HaloProvider) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	if err := p.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	var lastErr error
	delay := time.Second
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
		}

		resp, err := p.client.Do(req.WithContext(ctx))
		if err != nil {
			lastErr = err
			continue
		}
		// Retry sur 429 et 503.
		if resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode == http.StatusServiceUnavailable {
			resp.Body.Close()
			lastErr = &httpStatusError{Code: resp.StatusCode}
			continue
		}
		return resp, nil
	}
	return nil, lastErr
}

// httpStatusError est une erreur HTTP avec son code de statut.
type httpStatusError struct{ Code int }

func (e *httpStatusError) Error() string {
	return http.StatusText(e.Code)
}
