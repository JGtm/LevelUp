// Package middleware — security_headers.go : en-têtes de sécurité HTTP transverses.
//
// Pose les en-têtes de durcissement standards sur toutes les réponses :
// anti-MIME-sniffing, anti-clickjacking, fuite de referrer, isolation du contexte
// de navigation. HSTS n'est posé que sur une requête HTTPS réelle (TLS natif ou
// reverse proxy de confiance) — l'activer sur du HTTP local forcerait le
// navigateur à basculer en HTTPS sur localhost. La décision est prise PAR REQUÊTE
// via RequestIsHTTPS (même source de vérité que le flag Secure des cookies).
//
// Content-Security-Policy est volontairement OMIS ici : une CSP stricte casserait
// le SPA (scripts/styles ECharts inline, assets CDN Halo) sans une stratégie de
// nonces coordonnée avec le frontend. À concevoir séparément (cf. revue P0
// 2026-06-02, dette P2).
package middleware

import "net/http"

// SecurityHeaders pose les en-têtes de sécurité HTTP sur toutes les réponses.
// HSTS (Strict-Transport-Security) n'est ajouté que si la requête est servie en
// HTTPS (cf. RequestIsHTTPS). trustProxy autorise la détection via
// X-Forwarded-Proto derrière un reverse proxy de confiance.
func SecurityHeaders(trustProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			if RequestIsHTTPS(r, trustProxy) {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}
