// openapi_security.go — déclaration des `securitySchemes` du contrat publié
// (reliquat V72-01, item backlog archi/contrat).
//
// Le contrat ne déclarait AUCUN mécanisme d'authentification alors que toute la
// surface gardée en repose sur un seul : le cookie de session HttpOnly posé par
// middleware.WithSession. Un consommateur du contrat n'avait donc aucun moyen de
// savoir comment s'authentifier.
//
// Ce fichier vit dans le package `api` (et non dans `humacore`, qui n'importe
// AUCUN package du projet par construction) pour lire le nom du cookie à sa
// SOURCE UNIQUE — session.CookieName — plutôt que de re-littéraliser « levelup_session ».
package api

import (
	"github.com/danielgtaylor/huma/v2"

	session_platform "levelup/go-api/internal/platform/session"
)

// SessionCookieSchemeName — clé du security scheme dans components.securitySchemes.
const SessionCookieSchemeName = "sessionCookie"

// declareSecuritySchemes pose les `securitySchemes` sur le document OpenAPI PARTAGÉ.
//
// PORTÉE VOLONTAIREMENT LIMITÉE À LA DÉCLARATION : aucune exigence `security` n'est
// posée, ni globalement ni par opération. Une exigence globale mentirait — une part
// de la surface est publique par conception (bootstrap, référentiels d'assets,
// changelog, device-flow d'onboarding : cf. publicRoutesAllowlist du ratchet
// bare_routes). L'inventaire route → garde reste porté par ce ratchet, qui est
// exécutable ; le dupliquer en annotations de contrat créerait une seconde source
// de vérité vouée à diverger.
func declareSecuritySchemes(doc *huma.OpenAPI) {
	if doc == nil || doc.Components == nil {
		return
	}
	if doc.Components.SecuritySchemes == nil {
		doc.Components.SecuritySchemes = map[string]*huma.SecurityScheme{}
	}
	doc.Components.SecuritySchemes[SessionCookieSchemeName] = &huma.SecurityScheme{
		Type: "apiKey",
		In:   "cookie",
		Name: session_platform.CookieName,
		Description: "Cookie de session HttpOnly posé par le serveur après " +
			"authentification (login local, register par invitation ou SSO Xbox). " +
			"Signé côté serveur ; les routes en écriture exigent en plus l'en-tête " +
			"CSRF Origin/Referer d'une origine autorisée.",
	}
}
