// Package middleware — require_capability.go : guard de capability multi-titres.
//
// P6.3 (revue 2026-04-29) : protège les sous-arbres de routes qui dépendent
// d'une capability spécifique (Firefight, Forge, Career, Media, …). Le titre
// courant est lu depuis le contexte (injecté par TitleExtractor). Si le titre
// ne déclare pas la capability, retourne 503 Service Unavailable avec un body
// explicatif machine-readable.
//
// La résolution du titre se fait via le Registry passé au constructeur — pas
// via un slug hardcodé : c'est ce qui permet à un futur titre B (sans Forge)
// de désactiver automatiquement les routes Forge sans modifier le code.
package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
)

// RequireCapability retourne un middleware qui bloque les requêtes pour
// lesquelles le titre courant ne déclare pas la capability `cap`.
//
// Réponse 503 avec body :
//
//	{
//	  "code":         "capability_unavailable",
//	  "capability":   "firefight",
//	  "title_slug":   "halo_mcc",
//	  "message":      "Cette fonctionnalité n'est pas disponible pour ce titre.",
//	  "retryable":    false
//	}
//
// Le code 503 est préféré à 404 pour signaler explicitement « la route existe
// mais le titre courant ne supporte pas cette feature » (vs 404 = route
// inconnue) — utile pour le frontend pour afficher un état dégradé clair.
func RequireCapability(registry *titlePkg.Registry, cap titlePkg.Capability) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slug := ctxkeys.TitleSlug(r.Context())
			if slug == "" {
				slug = titlePkg.DefaultSlug
			}
			desc := registry.Get(slug)
			if desc == nil || !desc.HasCapability(cap) {
				slog.WarnContext(r.Context(), "capability rejected",
					"path", r.URL.Path,
					"titleSlug", slug,
					"capability", string(cap),
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(map[string]any{
					errKeyCode:      "capability_unavailable",
					"capability":    string(cap),
					"title_slug":    slug,
					errKeyMessage:   "Cette fonctionnalité n'est pas disponible pour ce titre.",
					errKeyRetryable: false,
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
