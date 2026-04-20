// Package middleware — http_cache.go : helpers Cache-Control et ETag pour les réponses JSON.
//
// Stratégie :
//   - Endpoints stables (career, home, encounters) : Cache-Control public + max-age côté CDN/proxy,
//     ETag SHA-256 tronqué pour éviter les transferts redondants (304 Not Modified).
//   - Endpoints live (battlepass, challenges) : no-store — données temps réel, jamais en cache HTTP.
//   - Endpoints mutatifs (POST, PATCH) : pas de caching.
package middleware

import (
	"crypto/sha256"
	"fmt"
	"net/http"
)

// CacheMaxAge retourne un middleware qui pose Cache-Control: max-age sur les GET réussis.
// Utiliser pour les endpoints dont les données changent peu entre deux syncs.
func CacheMaxAge(seconds int) func(http.Handler) http.Handler {
	header := fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d", seconds, seconds/2)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				w.Header().Set("Cache-Control", header)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// NoStore pose Cache-Control: no-store sur toutes les réponses (données live).
func NoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// ETagFromBytes calcule un ETag SHA-256 tronqué (16 hex chars) depuis un payload.
func ETagFromBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return fmt.Sprintf(`"%x"`, sum[:8])
}

// WriteJSONCached sérialise v, pose un ETag, et retourne 304 si le client a déjà la version.
// À utiliser dans les handlers GET pour les endpoints stables.
func WriteJSONCached(w http.ResponseWriter, r *http.Request, status int, body []byte) {
	etag := ETagFromBytes(body)
	w.Header().Set("ETag", etag)

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
