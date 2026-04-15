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
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}
