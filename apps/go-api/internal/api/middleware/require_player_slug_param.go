// Package middleware — require_player_slug_param.go : forme du slug de joueur porté par le
// chemin d'une route hors du groupe /players/{player_slug} (administration, diagnostics).
package middleware

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/domain"
)

// RequirePlayerSlugParam refuse en 400 invalid_player_slug une requête dont le paramètre de
// chemin param (slug ou gamertag d'un joueur) n'a pas la forme d'un gamertag
// (domain.IsValidPlayerSlug) : le slug devient un segment de chemin disque. Une route qui ne
// porte pas ce paramètre passe. Se monte en middleware de route (`r.With`), où chi a déjà
// résolu les paramètres, après la garde d'accès. Les routes du groupe joueur sont couvertes
// par RequirePlayerOwnership ; garde-rail `internal/api/player_slug_param_routes_test.go`.
func RequirePlayerSlugParam(param string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slug, present := routeParam(r, param)
			if present && !domain.IsValidPlayerSlug(slug) {
				slog.WarnContext(r.Context(), "authz: slug joueur mal formé refusé",
					"slug", slug, "param", param, "ip", chimiddleware.GetClientIP(r.Context()))
				writeInvalidPlayerSlug(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// routeParam rend la valeur du paramètre de chemin key et s'il est porté par la route
// résolue (une valeur vide d'un paramètre porté reste « présente »).
func routeParam(r *http.Request, key string) (string, bool) {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return "", false
	}
	for i := len(rctx.URLParams.Keys) - 1; i >= 0; i-- {
		if rctx.URLParams.Keys[i] == key {
			return rctx.URLParams.Values[i], true
		}
	}
	return "", false
}
