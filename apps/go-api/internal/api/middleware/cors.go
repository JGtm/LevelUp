// Package middleware fournit les middlewares HTTP transverses.
// Sprint 4 : CORS configurable depuis AppConfig.
package middleware

import (
	"net/http"

	"github.com/go-chi/cors"
)

// ReplayLatestSchemaHeader porte la version COURANTE du producteur de rejeu 2D (lot A,
// 2026-09-11 ; handlers/replay.go, tag `header:` sur replayOutput — un tag de struct ne peut
// pas citer une constante, le test du handler verrouille l'egalite). Il doit etre EXPOSE au
// client cross-origin comme ResolvedTitleHeader, sans quoi le badge admin ne lit jamais la
// version courante et ne dit jamais « a recuire » (revue de vague, 2026-09-12).
const ReplayLatestSchemaHeader = "X-Replay-Latest-Schema-Version"

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
		ExposedHeaders:   []string{"X-Request-ID", ResolvedTitleHeader, ReplayLatestSchemaHeader},
		AllowCredentials: true,
		MaxAge:           300,
	})
}
