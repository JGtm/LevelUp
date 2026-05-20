// Package middleware — csrf.go : protection CSRF par vérification d'origine.
//
// Sprint 30 : vérifie que les requêtes mutatrices (POST, PUT, PATCH, DELETE)
// proviennent d'une origine autorisée (header Origin ou Referer).
package middleware

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

// CSRF retourne un middleware qui vérifie l'en-tête Origin (ou Referer) des
// requêtes mutatrices. Seules les origines listées dans allowedOrigins sont
// acceptées. Les requêtes GET/HEAD/OPTIONS passent sans vérification.
func CSRF(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimRight(o, "/")] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = extractOrigin(r.Header.Get("Referer"))
			}

			if origin == "" || !isAllowedOrigin(origin, allowed) {
				slog.WarnContext(r.Context(), "middleware.csrf: origin non autorisée",
					"origin", origin, "path", r.URL.Path, "method", r.Method)
				http.Error(w, `{"code":"csrf_rejected","message":"origin non autorisée"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractOrigin extrait "scheme://host" depuis une URL complète (Referer).
func extractOrigin(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// isAllowedOrigin vérifie que l'origine fournie fait partie des origines autorisées.
func isAllowedOrigin(origin string, allowed map[string]struct{}) bool {
	origin = strings.TrimRight(origin, "/")
	_, ok := allowed[origin]
	return ok
}
