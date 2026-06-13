// Package middleware fournit les middlewares HTTP transverses.
// Sprint 4 : rate limiting par IP via go-chi/httprate.
package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/httprate"
)

const (
	// RateLimitRequests est le nombre de requêtes par fenêtre par défaut, utilisé
	// quand l'appelant ne fournit pas de plafond explicite (rpm <= 0).
	RateLimitRequests = 300
	// RateLimitWindow est la durée de la fenêtre de rate limiting.
	RateLimitWindow = time.Minute
)

// RateLimit retourne un middleware qui limite à rpm req/min par IP (défaut
// RateLimitRequests si rpm <= 0). En mode démo/test, la limite est 10× plus haute
// pour éviter de bloquer les benchmarks.
//
// IMPORTANT : httprate clé sur r.RemoteAddr. Derrière un reverse proxy, RemoteAddr
// vaut l'IP du proxy (127.0.0.1) pour tous les clients tant que chi RealIP n'a pas
// réécrit l'IP réelle (LEVELUP_TRUST_PROXY_HEADERS=true). Sans ça, tout le trafic
// partage un seul bucket et le site sature en 429 sous charge publique.
//
// Les requêtes vers /static/* et /api/v1/assets/* sont exemptées : ce sont des
// fichiers servis localement (FileServer pour /static/*, resolver local-first
// pour /api/v1/assets/*). Une page home ou Season Pass peut émettre 100+ URLs
// uniques d'images, ce qui saturait la fenêtre et finissait par claquer y compris
// les endpoints applicatifs (cf. bugs images home page 2026-05-06 et Season Pass /
// Héritage 2026-05-06). Le rate limit reste actif sur tout le reste — endpoints
// applicatifs et données.
func RateLimit(demoMode bool, rpm int) func(http.Handler) http.Handler {
	limit := rpm
	if limit <= 0 {
		limit = RateLimitRequests
	}
	if demoMode {
		limit *= 10
	}
	limiter := httprate.LimitByIP(limit, RateLimitWindow)
	return func(next http.Handler) http.Handler {
		throttled := limiter(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/static/") ||
				strings.HasPrefix(r.URL.Path, "/api/v1/assets/") {
				next.ServeHTTP(w, r)
				return
			}
			throttled.ServeHTTP(w, r)
		})
	}
}
