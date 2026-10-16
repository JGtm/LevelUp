// Package middleware — title_from_path.go : titre courant dérivé d'un PARAM DE
// PATH (et non du header/session).
//
// Cas d'usage : les endpoints qui agissent sur un titre CIBLE arbitraire passé
// dans l'URL (ex. activer/mettre en pause/purger un titre d'un joueur). On veut
// que la garde d'ownership (RequirePlayerOwnership, qui résout le joueur via
// ctxkeys.TitleSlug) raisonne sur le TITRE CIBLÉ, pas sur le titre du header —
// sinon un client pourrait viser un titre du path tout en présentant un header
// sur un autre titre où il n'est pas propriétaire (bypass d'ownership).
package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
)

// TitleSlugFromPath pose ctxkeys.WithTitleSlug à partir du param de path nommé
// paramName. Un titre absent du registre (quel que soit son statut, un titre
// coming_soon ou archivé reste servi) reçoit 404 title_not_found avant toute autre
// garde : le titre devient un segment de chemin disque (PathResolver) et une clé
// de db_profiles.json. Si le param est absent/vide, le contexte est laissé inchangé
// (le fallback ctxkeys.TitleSlug s'appliquera). À monter sur un groupe chi dont le
// motif CONTIENT déjà ce param (sinon chi ne l'a pas encore résolu).
func TitleSlugFromPath(registry *titlePkg.Registry, paramName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if slug := chi.URLParam(r, paramName); slug != "" {
				if !registry.Exists(slug) {
					slog.WarnContext(r.Context(), "title_from_path: titre hors registre refusé",
						"requested_title", slug, "path", r.URL.Path)
					writeTitleNotFound(w)
					return
				}
				r = r.WithContext(ctxkeys.WithTitleSlug(r.Context(), slug))
			}
			next.ServeHTTP(w, r)
		})
	}
}

// writeTitleNotFound répond 404 title_not_found (titre absent du registre).
func writeTitleNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]any{
		errKeyCode:      "title_not_found",
		errKeyMessage:   "title not found",
		errKeyRetryable: false,
	})
}
