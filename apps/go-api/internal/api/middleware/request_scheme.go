// Package middleware — request_scheme.go : détection du schéma HTTPS par requête.
//
// Source de vérité unique partagée par le cookie de session (flag Secure) et les
// en-têtes de sécurité (HSTS). On décide PAR REQUÊTE plutôt qu'avec un booléen
// figé au boot : un même binaire sert ainsi correctement en HTTP local et en
// HTTPS prod, sans reconfiguration.
//
// Pourquoi pas un calcul au démarrage : coupler « un secret de session custom est
// défini » à « servi en HTTPS » casse le dev local (cookie Secure jeté par le
// navigateur sur http://localhost → session non persistée → onboarding bloqué).
package middleware

import (
	"net/http"
	"strings"
)

// RequestIsHTTPS retourne true si la requête est servie en HTTPS, soit
// directement (TLS natif, r.TLS != nil), soit derrière un reverse proxy de
// confiance qui annonce le schéma d'origine via X-Forwarded-Proto.
//
// trustProxy DOIT être vrai uniquement si le serveur tourne derrière un proxy qui
// assainit cet en-tête (LEVELUP_TRUST_PROXY_HEADERS=true) : sinon un client
// externe peut usurper « X-Forwarded-Proto: https ». Même modèle de confiance que
// chi RealIP / LoopbackOnly (cf. server.go + loopback.go).
func RequestIsHTTPS(r *http.Request, trustProxy bool) bool {
	if r.TLS != nil {
		return true
	}
	if trustProxy {
		proto := r.Header.Get("X-Forwarded-Proto")
		// L'en-tête peut contenir une liste « https, http » (chaîne de proxies) :
		// le schéma d'origine (client → 1er proxy) est le premier élément.
		if i := strings.IndexByte(proto, ','); i >= 0 {
			proto = proto[:i]
		}
		if strings.EqualFold(strings.TrimSpace(proto), "https") {
			return true
		}
	}
	return false
}

// Modes possibles de SecureCookiePolicy.Mode.
const (
	CookieSecureAuto  = "auto"  // décision par schéma (défaut)
	CookieSecureTrue  = "true"  // force Secure (déploiement HTTPS atypique non détectable)
	CookieSecureFalse = "false" // force non-Secure (filet de secours ops)
)

// SecureCookiePolicy décide du flag Secure des cookies de session.
//
//   - Mode "auto" (défaut) : Secure suit le schéma réel de la requête.
//   - Mode "true" / "false" : override explicite (LEVELUP_COOKIE_SECURE).
//
// TrustProxy est propagé à RequestIsHTTPS pour le cas reverse proxy TLS.
type SecureCookiePolicy struct {
	Mode       string
	TrustProxy bool
}

// Secure retourne le flag Secure à poser pour cette requête selon la policy.
func (p SecureCookiePolicy) Secure(r *http.Request) bool {
	switch strings.ToLower(strings.TrimSpace(p.Mode)) {
	case CookieSecureTrue:
		return true
	case CookieSecureFalse:
		return false
	default: // "auto" et toute valeur inconnue → décision par schéma (défensif)
		return RequestIsHTTPS(r, p.TrustProxy)
	}
}
