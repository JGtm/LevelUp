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

// DefaultStatus fixe le code de statut de SUCCÈS de l'opération (201, 202, …) —
// modificateur composable, à passer À CÔTÉ de Op :
//
//	huma.Post(api, "/groups", h.handleCreateGroup,
//		humacore.Op("createGroup", "Crée un groupe", "groups"),
//		humacore.DefaultStatus(http.StatusCreated))
//
// SOURCE UNIQUE du statut (V721-04). Huma écrit `op.DefaultStatus` au runtime ET
// range le schéma de réponse sous ce code dans le document (huma.go, Register →
// processOutputType) — sauf si le type de sortie porte un champ `Status int`,
// auquel dernier cas la VALEUR DU CHAMP l'emporte au runtime, inconditionnellement
// (`status = int(vo.Field(outStatusIndex).Int())`).
//
// D'où la règle :
//   - statut FIXE → PAS de champ `Status` dans la struct de sortie, ce
//     modificateur seul (le code n'est écrit qu'ici, et c'est exactement ce que
//     le contrat publie) ;
//   - statut DYNAMIQUE (une même opération répond 202 OU 409 avec un corps
//     différent) → le champ `Status` reste le mécanisme runtime, et ce
//     modificateur déclare le statut NOMINAL pour le document.
//
// Sans lui, Huma documente 200 dès que la sortie a un corps : c'est le défaut qui
// obligeait le fragment manuel à supprimer le 200 dérivé et à redécrire un 20x
// que Huma sait pourtant dériver.
func DefaultStatus(status int) func(o *huma.Operation) {
	return func(o *huma.Operation) {
		o.DefaultStatus = status
	}
}

// MaxBody fixe le plafond de taille du CORPS de la requête, en octets —
// modificateur composable, à passer à côté de Op.
//
// POURQUOI IL FAUT LE POSER EXPLICITEMENT quand une route reçoit un fichier :
// Huma applique un défaut de 1 Mio (ensureMaxBodyBytes) et répond 413 dès que la
// lecture l'atteint. C'est une bonne valeur pour des corps de formulaire, et un
// refus systématique pour un artefact de rejeu (~2 Mo). Le plafond n'est donc
// jamais « désactivé » ici : il est DÉCLARÉ, fini, et lu depuis la constante du
// domaine qui le justifie.
func MaxBody(bytes int64) func(o *huma.Operation) {
	return func(o *huma.Operation) {
		o.MaxBodyBytes = bytes
	}
}
