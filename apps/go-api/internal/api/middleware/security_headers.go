// Package middleware — security_headers.go : en-têtes de sécurité HTTP transverses.
//
// Pose les en-têtes de durcissement standards sur toutes les réponses :
// anti-MIME-sniffing, anti-clickjacking, fuite de referrer, isolation du contexte
// de navigation. HSTS n'est posé que sur une requête HTTPS réelle (TLS natif ou
// reverse proxy de confiance) — l'activer sur du HTTP local forcerait le
// navigateur à basculer en HTTPS sur localhost. La décision est prise PAR REQUÊTE
// via RequestIsHTTPS (même source de vérité que le flag Secure des cookies).
//
// Content-Security-Policy y est MINIMALE (minimalCSP) : elle ferme les greffons
// (object-src), la réécriture de l'URL de base (base-uri) et l'encadrement de la
// page (frame-ancestors) sans restreindre les scripts ni les styles du SPA — une
// politique de scripts stricte casserait ECharts (styles inline) et les assets CDN
// Halo sans une stratégie de nonces coordonnée avec le frontend, à décider dans un
// ADR à part. La page /docs (hors production) pose sa propre politique, plus stricte,
// qui remplace celle-ci.
package middleware

import "net/http"

// minimalCSP : la CSP posée sur toute réponse. Aucune directive de chargement
// (default-src, script-src, style-src, img-src, connect-src) : le SPA charge ce dont
// il a besoin comme avant.
const minimalCSP = "object-src 'none'; base-uri 'self'; frame-ancestors 'none'"

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
			h.Set("Content-Security-Policy", minimalCSP)
			if RequestIsHTTPS(r, trustProxy) {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}
