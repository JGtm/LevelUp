// Package middleware — title.go : extraction du titre courant depuis la session ou le header.
//
// Sprint 44 WP4 : Le middleware TitleExtractor place le title_slug dans le contexte
// de la requête via ctxkeys.WithTitleSlug. Priorité :
//  1. Header X-LevelUp-Title (pour les clients API directs)
//  2. Session courante (CurrentTitleSlug)
//  3. Fallback : "halo_infinite"
package middleware

import (
	"net/http"

	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
)

// TitleExtractor lit le titre courant depuis le header X-LevelUp-Title
// ou depuis la session et l'injecte dans le contexte.
func TitleExtractor(registry *titlePkg.Registry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slug := resolveTitleSlug(r, registry)
			ctx := ctxkeys.WithTitleSlug(r.Context(), slug)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// resolveTitleSlug détermine le titre courant pour la requête.
//
// SEAM (PMT-8) : ce résolveur INJECTE le titre demandé dans le contexte dès
// qu'il EXISTE dans le registre (header > session > défaut), y compris un titre
// coming_soon/archived. Le rejet d'un titre non-actif n'est PAS fait ici (sinon
// le fallback silencieux masquerait le titre demandé) mais par le gate
// middleware.RequireActiveTitle (503 title_unavailable) monté sur les
// sous-arbres title-scoped. Un header pointant un titre INCONNU retombe sur le
// défaut.
func resolveTitleSlug(r *http.Request, registry *titlePkg.Registry) string {
	// 1. Header explicite
	if h := r.Header.Get("X-LevelUp-Title"); h != "" {
		if registry.Exists(h) {
			return h
		}
	}

	// 2. Session courante (via GetSession qui utilise la bonne context key)
	if sess := GetSession(r.Context()); sess != nil {
		if sess.CurrentTitleSlug != "" && registry.Exists(sess.CurrentTitleSlug) {
			return sess.CurrentTitleSlug
		}
	}

	// 3. Fallback
	return titlePkg.DefaultSlug
}
