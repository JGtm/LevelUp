// Package middleware — player_visitor_actions.go : les écritures du VISITEUR sur le
// profil d'un ami (ADR 0029, accès famille en lecture seule et ses exceptions nommées).
//
// Une écriture sur le groupe /players/{player_slug} est normalement réservée au
// propriétaire du profil (ou à l'admin). Une action du visiteur fait exception : elle
// s'écrit au nom du VISITEUR (son profil possédé, résolu par le handler), pas au nom du
// profil consulté. Elle est donc ouverte à quiconque peut LIRE le profil.
//
// Seule exception (liste tenue par familyVisitorWrites, internal/api/family_write_routes_test.go) :
// le like d'un média (PATCH /media/likes), attribué au demandeur par
// MediaHandler.resolveLikerIdentity. La garde d'authentification des écritures
// (RequireAuthForMutations) continue de s'appliquer : la requête reste une écriture.
package middleware

import "net/http"

// mediaLikesSubPath : sous-chemin (relatif au groupe joueur) du like d'un média.
const mediaLikesSubPath = "/media/likes"

// visitorActionRoutes : méthode + sous-chemin EXACT (relatif au groupe joueur) des
// actions du visiteur.
var visitorActionRoutes = map[string]string{
	mediaLikesSubPath: http.MethodPatch,
}

// isVisitorAction dit si la requête est une action du visiteur. Un chemin ambigu n'en
// est jamais une (cf. AmbiguousPath) : la garde le lirait autrement que chi ne le
// route, l'exception ne s'applique qu'à un chemin qui se lit comme il se route.
func isVisitorAction(r *http.Request) bool {
	if AmbiguousPath(r) {
		return false
	}
	method, ok := visitorActionRoutes[playerScopedSubPath(r)]
	return ok && method == r.Method
}
