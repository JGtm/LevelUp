// Package middleware fournit les middlewares HTTP transverses.
// Sprint 4 : CORS configurable depuis AppConfig.
package middleware

import (
	"net/http"

	"github.com/go-chi/cors"
)

// CORS retourne un middleware CORS configuré avec les origines autorisées.
// En développement, les origines par défaut sont localhost:5173 (Vite dev server).
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		// ResolvedTitleHeader doit être LISIBLE par le client cross-origin (fetch ne
		// voit un header de réponse que s'il est exposé) : le garde anti-fuite
		// cross-titre du client API le lit pour rejeter les réponses d'un autre titre.
		ExposedHeaders:   []string{"X-Request-ID", ResolvedTitleHeader},
		AllowCredentials: true,
		MaxAge:           300,
	})
}
