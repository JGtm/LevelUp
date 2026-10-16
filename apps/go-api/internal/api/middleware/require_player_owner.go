// Package middleware — require_player_owner.go : garde des données PRIVÉES d'un profil
// joueur (ADR 0029).
//
// La garde de groupe (RequirePlayerOwnership) laisse un co-membre de groupe LIRE le
// profil d'un ami. Certaines données ne se lisent pas entre amis : elles appartiennent au
// seul propriétaire du profil. RequirePlayerOwner est montée en garde de ROUTE (r.With),
// après le routage : elle ne s'applique qu'aux routes qui la portent, et décide chacune
// avec la règle d'écriture (propriétaire direct ou admin), quelle que soit la méthode.
//
// Routes qui la portent : les notifications (/players/{player_slug}/notifications...).
// Garde-rail : internal/api/family_write_routes_test.go (TestOwnerOnlyRoutesCarryOwnerGuard).
package middleware

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// RequirePlayerOwner refuse (403 player_forbidden) toute requête, lecture comprise, sur
// le profil `player_slug` dont la session n'est ni le propriétaire direct ni l'admin.
// Transparente quand la propriété n'est pas appliquée (démo, auth non activée). Une
// session absente passe, comme pour la garde de groupe qui la précède.
func RequirePlayerOwner(access *PlayerAccess) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if !access.Enforced() {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := GetSession(r.Context())
			if sess == nil {
				next.ServeHTTP(w, r)
				return
			}
			slug := chi.URLParam(r, "player_slug")
			if granted, _ := access.Allows(r.Context(), sess, slug, true); granted {
				next.ServeHTTP(w, r)
				return
			}
			slog.WarnContext(r.Context(), "authz: donnée privée du profil refusée hors propriétaire",
				"slug", slug, "path", r.URL.Path, "method", r.Method,
				"ip", chimiddleware.GetClientIP(r.Context()))
			writePlayerForbidden(w)
		})
	}
}
