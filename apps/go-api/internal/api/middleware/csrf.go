// Package middleware — csrf.go : protection CSRF par vérification d'origine.
//
// Sprint 30 : vérifie que les requêtes mutatrices (POST, PUT, PATCH, DELETE)
// proviennent d'une origine autorisée (header Origin ou Referer).
//
// ─── EXEMPTION CIBLÉE DE PRÉFIXES (2026-08-25, lot csrf-ouvrier) ──────────────
//
// LE DÉFAUT MESURÉ. Dry run superviseur du 2026-08-25 depuis le VPS de calcul :
// `replay-worker --once` contre https://lvelup.info/api/v1/internal répond
// `HTTP 403 {"code":"csrf_rejected","message":"origin non autorisée"}` — AVANT
// tout contrôle de jeton. Cause : ce middleware est monté TRANSVERSALEMENT sur le
// routeur racine (server.go, applyTransverseMiddlewares) et rejette toute requête
// mutatrice sans Origin ni Referer autorisé (origin == "" → 403) ; or un ouvrier
// est un client net/http qui n'envoie PAS d'Origin — par nature, pas par oubli.
// Le protocole ouvrier était donc mort à l'arrivée en production.
//
// POURQUOI EXEMPTER NE RETIRE AUCUNE PROTECTION RÉELLE. Le CSRF-par-origine
// protège les flux dont l'authentification est AMBIANTE : le navigateur joint
// tout seul le cookie de session à une requête déclenchée par un site tiers. Les
// routes du protocole ouvrier (/api/v1/internal/*) n'acceptent AUCUN cookie et ne
// lisent AUCUNE session : leur seule authentification est un jeton Bearer dédié
// (handlers.BuildWorkerHandler.RequireWorkerToken, comparaison à temps constant),
// qu'une page tierce ne peut pas fabriquer — et un en-tête Authorization
// cross-origin déclenche de toute façon un préflight CORS qui échoue. Il n'y a
// donc pas d'autorité ambiante à protéger sur ce préfixe.
//
// PORTÉE EXACTE. Seul le contrôle CSRF est levé, et seulement sous les préfixes
// passés à CSRF(). Tout le reste de la pile transverse continue de s'appliquer à
// ces routes : SecurityHeaders, RequestID, RateLimit, logs slog, compression,
// session, TitleExtractor. C'est précisément pourquoi l'exemption vit ICI et non
// dans un montage « bare » du sous-routeur /internal en dehors de la pile : sortir
// les routes de la pile leur retirerait aussi le rate-limit et les logs d'audit —
// on ne veut lever QUE le contrôle qui n'a pas de sens pour un client sans cookie.
// Le comportement par défaut (aucun préfixe passé) est strictement inchangé.
package middleware

import (
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// CSRF retourne un middleware qui vérifie l'en-tête Origin (ou Referer) des
// requêtes mutatrices. Seules les origines listées dans allowedOrigins sont
// acceptées. Les requêtes GET/HEAD/OPTIONS passent sans vérification.
//
// exemptPrefixes (variadique, vide par défaut) liste les préfixes de chemin ABSOLUS
// dont les requêtes mutatrices traversent le contrôle d'origine — réservé aux
// routes SANS authentification par cookie (cf. l'en-tête du fichier). Un préfixe
// ne matche que sur une frontière de segment : "/api/v1/internal" exempte
// "/api/v1/internal/build-queue/claim" mais PAS "/api/v1/internalise".
func CSRF(allowedOrigins []string, exemptPrefixes ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimRight(o, "/")] = struct{}{}
	}
	exempt := normalizeExemptPrefixes(exemptPrefixes)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			if isExemptPath(r.URL.Path, exempt) {
				slog.DebugContext(r.Context(), "middleware.csrf: contrôle d'origine exempté (préfixe sans cookie)",
					"path", r.URL.Path, "method", r.Method)
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = extractOrigin(r.Header.Get("Referer"))
			}

			if origin == "" || !isAllowedOrigin(origin, allowed) {
				slog.WarnContext(r.Context(), "middleware.csrf: origin non autorisée",
					"origin", origin, "path", r.URL.Path, "method", r.Method)
				http.Error(w, `{"code":"csrf_rejected","message":"origin non autorisée"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// normalizeExemptPrefixes nettoie les préfixes d'exemption et REJETTE ceux qui
// ouvriraient plus que prévu : un préfixe vide, "/" seul, ou un chemin relatif
// exempterait de fait toute l'API du contrôle CSRF. Un tel préfixe est ignoré et
// journalisé plutôt qu'appliqué en silence — une faute de câblage ne doit pas se
// solder par une protection désactivée sans trace.
func normalizeExemptPrefixes(prefixes []string) []string {
	out := make([]string, 0, len(prefixes))
	for _, p := range prefixes {
		cleaned := strings.TrimRight(strings.TrimSpace(p), "/")
		if cleaned == "" || !strings.HasPrefix(cleaned, "/") {
			slog.Warn("middleware.csrf: préfixe d'exemption ignoré (vide, racine ou relatif)", "prefix", p)
			continue
		}
		out = append(out, cleaned)
	}
	return out
}

// isExemptPath dit si le chemin demandé tombe sous un préfixe exempté.
//
// path.Clean résout "." et ".." AVANT la comparaison : sans lui,
// "/api/v1/internal/../session/context" porterait le préfixe exempté tout en
// désignant une route protégée. Le chemin nettoyé, lui, ne matche plus le
// préfixe et repasse sous le contrôle d'origine. (r.URL.Path est déjà
// pourcent-décodé par net/http : un "%2e%2e" est traité comme "..".)
func isExemptPath(reqPath string, exempt []string) bool {
	if len(exempt) == 0 {
		return false
	}
	cleaned := path.Clean(reqPath)
	for _, prefix := range exempt {
		if cleaned == prefix || strings.HasPrefix(cleaned, prefix+"/") {
			return true
		}
	}
	return false
}

// extractOrigin extrait "scheme://host" depuis une URL complète (Referer).
func extractOrigin(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// isAllowedOrigin vérifie que l'origine fournie fait partie des origines autorisées.
func isAllowedOrigin(origin string, allowed map[string]struct{}) bool {
	origin = strings.TrimRight(origin, "/")
	_, ok := allowed[origin]
	return ok
}
