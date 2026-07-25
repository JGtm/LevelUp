// Package middleware — title.go : extraction du titre courant depuis la session ou le header.
//
// Sprint 44 WP4 : Le middleware TitleExtractor place le title_slug dans le contexte
// de la requête via ctxkeys.WithTitleSlug. Priorité :
//  1. Header X-LevelUp-Title (pour les clients API directs)
//  2. Session courante (CurrentTitleSlug)
//  3. Fallback : "halo_infinite"
package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
)

// TitleExtractor lit le titre courant depuis le header X-LevelUp-Title (ou la
// session) ET la locale UI depuis le header X-LevelUp-Locale, et les injecte dans
// le contexte. La locale alimente les lectures localisées (noms de commendations,
// etc.) qui résolvent via ctxkeys.Locale sans threader la locale dans chaque
// signature (utile pour les chemins canoniques ID-keyés, ex. LoadMatchSummaries).
func TitleExtractor(registry *titlePkg.Registry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slug := resolveTitleSlug(r, registry)
			ctx := ctxkeys.WithTitleSlug(r.Context(), slug)
			ctx = ctxkeys.WithLocale(ctx, resolveLocale(r))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// resolveLocale lit la locale UI depuis le header X-LevelUp-Locale (posé par le
// front sur toutes les requêtes, cf. setApiLocale). Normalise vers "fr"/"en" ;
// valeur vide ou inconnue → "fr" (cohérent avec le défaut de ctxkeys.Locale).
func resolveLocale(r *http.Request) string {
	switch strings.ToLower(strings.TrimSpace(r.Header.Get("X-LevelUp-Locale"))) {
	case "en", "en-us", "en_us":
		return "en"
	default:
		return "fr"
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
		// Header présent mais titre INCONNU du registre : cas anormal (client obsolète,
		// titre retiré/mal orthographié). On ne fuit PAS silencieusement vers un autre
		// titre — on trace AVANT de retomber sur session/défaut, pour rendre la mauvaise
		// résolution visible dans logs/ (anti-fuite : diagnostiquer une confusion de
		// titre plutôt que l'avaler — CLAUDE.md anti-pattern #10). Rare par construction
		// (le front n'émet que des slugs connus), donc pas de bruit en régime normal.
		slog.WarnContext(r.Context(), "title: header X-LevelUp-Title inconnu, ignoré (fallback session/défaut)",
			"requested_title", h)
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
