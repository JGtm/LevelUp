// Package humacore — operation.go : métadonnées OpenAPI stables par opération
// (chantier V72-01 / H2).
//
// Les raccourcis huma.Get/Post/... génèrent un OperationID et un Summary AUTO
// (dérivés de la méthode + du chemin) et laissent les Tags vides. Pour que le
// document OpenAPI PARTAGÉ (H1) reproduise le groupage `tags:` et les
// operationId STABLES du contrat manuel (api/openapi.yaml) — dont dépend la
// génération du client TypeScript (generated.ts, H7) —, chaque route passe le
// modificateur Op(...) en dernier argument variadique :
//
//	huma.Get(api, "/pages/career", h.handleGetCareer,
//		humacore.Op("getPlayerCareerOverview", "Page Carrière", "career"))
//
// Poser l'OperationID/Summary EXPLICITEMENT désactive la régénération auto de
// huma (y compris le PrefixModifier d'un sous-routeur, qui ne re-préfixe l'ID que
// s'il est resté celui de la convenience) : la valeur passée survit telle quelle
// dans le document, seul le chemin gagne le préfixe absolu du sous-routeur.
package humacore

import "github.com/danielgtaylor/huma/v2"

// Op retourne un modificateur d'opération (dernier argument variadique de
// huma.Get/Post/Put/Patch/Delete) qui fixe les métadonnées OpenAPI stables :
// OperationID, Summary et Tags. Ces valeurs reproduisent le contrat manuel
// (parité avec api/openapi.yaml) ou, pour les routes non encore documentées,
// suivent la même convention de nommage. Passer AU MOINS un tag (garde-rail H2 :
// toute opération du document doit porter operationID + summary + >= 1 tag).
func Op(operationID, summary string, tags ...string) func(o *huma.Operation) {
	return func(o *huma.Operation) {
		o.OperationID = operationID
		o.Summary = summary
		o.Tags = tags
	}
}
