// Package middleware fournit les middlewares HTTP transverses.
// Sprint 4 : rate limiting par IP via go-chi/httprate.
package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/httprate"

	"levelup/go-api/internal/observability"
)

const (
	// RateLimitRequests est le nombre de requêtes par fenêtre par défaut, utilisé
	// quand l'appelant ne fournit pas de plafond explicite (rpm <= 0).
	RateLimitRequests = 300
	// RateLimitWindow est la durée de la fenêtre de rate limiting.
	RateLimitWindow = time.Minute

	// rateLimit429Key : clé d'anti-flood ET nom du compteur expvar. Un seul
	// appel à observability.AllowThrottledLog produit les deux :
	// le compteur cumulatif `levelup.rate_limit_429_total` (visible sur
	// /debug/vars, JAMAIS remis à zéro) et la décision d'émettre le log.
	rateLimit429Key = "rate_limit_429"
	// RateLimit429Counter est le nom du compteur expvar des réponses 429
	// (exporté pour les tests et le panneau monitoring).
	RateLimit429Counter = rateLimit429Key + "_total"
	// rateLimit429LogWindow : au plus UN log WARN par fenêtre, avec le nombre
	// d'occurrences agrégées (`throttled_since_last`). Un bucket épuisé produit
	// des centaines de 429/s : un log par hit noierait les fichiers (c'est
	// exactement le volume qu'on vient de borner par la rotation).
	rateLimit429LogWindow = 30 * time.Second
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
// Exemptions (cf. rateLimitExempt) : fichiers statiques servis localement. Une
// page home ou Season Pass peut émettre 100+ URLs uniques d'images, ce qui
// saturait la fenêtre et finissait par claquer y compris les endpoints
// applicatifs (bugs images home page 2026-05-06 et Season Pass / Héritage
// 2026-05-06). Le rate limit reste actif sur tout le reste — endpoints
// applicatifs et données.
//
// Décision multi-titre (MT-25, PMT-13) : le rate limit est une protection
// transverse PAR IP, INVARIANTE au titre — il n'a (et ne doit avoir) aucune
// dimension slug. Les exemptions sont définies par préfixe/extension d'URL, donc
// title-agnostiques par construction. Aucun changement requis.
func RateLimit(demoMode bool, rpm int) func(http.Handler) http.Handler {
	limit := rpm
	if limit <= 0 {
		limit = RateLimitRequests
	}
	if demoMode {
		limit *= 10
	}
	limiter := httprate.Limit(limit, RateLimitWindow,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitHandler(newRateLimitedHandler(limit)),
	)
	return func(next http.Handler) http.Handler {
		throttled := limiter(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rateLimitExempt(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			throttled.ServeHTTP(w, r)
		})
	}
}

// rateLimitExempt indique qu'un chemin ne consomme PAS le bucket httprate.
//
// Sont exemptés : les fichiers servis localement par le serveur — /static/*
// (FileServer), /api/v1/assets/* (resolver local-first) et tout chemin à
// extension statique servi par le catch-all SPA depuis le dist Vite (icônes,
// /logo.png, /titles/**, polices... issus de apps/web/public/). Ces derniers
// consommaient le bucket jusqu'au 2026-07-26 : même classe de bug que les
// images 2026-05-06, mais sur les fichiers de public/ servis à la racine.
//
// L'exemption ne couvre JAMAIS /api/* (hors /api/v1/assets/ ci-dessus) : un
// endpoint applicatif à extension `.json` dans son chemin resterait limité.
func rateLimitExempt(urlPath string) bool {
	if strings.HasPrefix(urlPath, "/static/") || strings.HasPrefix(urlPath, "/api/v1/assets/") {
		return true
	}
	if strings.HasPrefix(urlPath, "/api/") {
		return false
	}
	return IsStaticAssetPath(urlPath)
}

// newRateLimitedHandler construit la réponse 429 : même corps que le défaut
// httprate, PLUS l'observabilité qui manquait (le 429 était totalement muet —
// ni log ni compteur, donc un bucket épuisé restait invisible en prod).
func newRateLimitedHandler(limit int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if allow, since := observability.AllowThrottledLog(rateLimit429Key, rateLimit429LogWindow); allow {
			slog.WarnContext(r.Context(), "rate limit atteint: 429",
				"path", r.URL.Path,
				"method", r.Method,
				"remote_addr", r.RemoteAddr,
				"limit_per_window", limit,
				"window", RateLimitWindow.String(),
				"throttled_since_last", since)
		}
		http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
	}
}
