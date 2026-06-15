// Package middleware — require_active_title.go : gate du cycle de vie du titre.
//
// PMT-8 / MT-22 : protège les sous-arbres de routes title-scoped (données /
// sync) quand le titre courant n'est pas ACTIF. Le titre courant est lu depuis
// le contexte (injecté par TitleExtractor, qui résout n'importe quel titre
// CONNU — y compris coming_soon/archived). Si le titre n'est pas actif, retourne
// 503 Service Unavailable avec un body machine-readable, plutôt qu'un fallback
// silencieux vers le titre par défaut.
//
// Jumeau de RequireCapability : même code 503 (« la route existe mais le titre
// courant n'est pas servable » vs 404 = route inconnue), résolution via le
// Registry passé au constructeur (jamais un slug hardcodé).
package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
)

// messageForStatus retourne le message utilisateur selon le statut du titre.
func messageForStatus(status titlePkg.Status) string {
	switch status {
	case titlePkg.StatusComingSoon:
		return "Ce titre sera bientôt disponible."
	case titlePkg.StatusArchived:
		return "Ce titre n'est plus maintenu."
	default:
		return "Ce titre n'est pas disponible."
	}
}

// RequireActiveTitle retourne un middleware qui bloque les requêtes dont le
// titre courant n'est pas au statut actif (coming_soon, archived, ou inconnu).
//
// Réponse 503 avec body :
//
//	{
//	  "code":       "title_unavailable",
//	  "title_slug": "halo_mcc",
//	  "status":     "coming_soon",
//	  "message":    "Ce titre sera bientôt disponible.",
//	  "retryable":  false
//	}
func RequireActiveTitle(registry *titlePkg.Registry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slug := ctxkeys.TitleSlug(r.Context())
			if slug == "" {
				slug = titlePkg.DefaultSlug
			}
			desc := registry.Get(slug)
			if desc != nil && desc.IsActive() {
				next.ServeHTTP(w, r)
				return
			}

			status := titlePkg.Status("unknown")
			if desc != nil {
				status = desc.Status
			}
			slog.WarnContext(r.Context(), "title_rejected",
				"title", slug,
				"status", string(status),
				"path", r.URL.Path,
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				errKeyCode:      "title_unavailable",
				"title_slug":    slug,
				"status":        string(status),
				errKeyMessage:   messageForStatus(status),
				errKeyRetryable: false,
			})
		})
	}
}
