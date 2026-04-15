// Package middleware fournit les middlewares HTTP transverses.
// Sprint 4 : rate limiting par IP via go-chi/httprate.
package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

const (
	// RateLimitRequests est le nombre maximum de requêtes par fenêtre.
	RateLimitRequests = 120
	// RateLimitWindow est la durée de la fenêtre de rate limiting.
	RateLimitWindow = time.Minute
)

// RateLimit retourne un middleware qui limite à 120 req/min par IP.
// En mode démo/test, la limite est 10× plus haute pour éviter de bloquer les benchmarks.
func RateLimit(demoMode bool) func(http.Handler) http.Handler {
	limit := RateLimitRequests
	if demoMode {
		limit = RateLimitRequests * 10
	}
	return httprate.LimitByIP(limit, RateLimitWindow)
}
