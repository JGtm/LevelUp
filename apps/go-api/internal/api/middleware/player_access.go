// Package middleware — player_access.go : la décision d'accès à un profil joueur,
// partagée par toutes les gardes qui la posent (ADR 0029, couche A).
//
// Trois appelants posent la même question — « l'utilisateur de cette session
// peut-il lire / écrire le profil de ce slug ? » : la garde de propriété du groupe
// /players/{player_slug} (RequirePlayerOwnership), la garde d'acteur des routes
// Prestige (created_by / requested_by / user_id du corps) et le choix du joueur
// courant (POST /session/context). Ils la posent ici, une seule fois : le
// résolveur slug → xuid, l'utilisateur courant et l'ensemble famille sont les
// mêmes pour les trois.
//
// Lecture et écriture ne se décident pas pareil : un co-membre de groupe LIT le
// profil d'un ami, il n'y ÉCRIT pas (authz.CanWritePlayer). L'ensemble famille
// n'est donc résolu que pour une lecture.
package middleware

import (
	"context"

	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/domain"
)

// PlayerAccess décide l'accès d'une session à un profil joueur. Construit une
// fois au câblage du routeur et partagé par ses appelants.
type PlayerAccess struct {
	enforced      bool
	resolveXUID   PlayerXUIDResolver
	users         authz.UserLookup
	resolveFamily FamilyXUIDResolver
}

// NewPlayerAccess construit la décision d'accès. La décision est ouverte quand
// l'enforcement est désactivé (démo, auth non activée) ou quand le résolveur de
// profils ou le magasin d'utilisateurs manquent. resolveFamily peut être nil :
// aucun accès famille, propriétaire et admin seulement.
func NewPlayerAccess(demoMode bool, authMode string, resolveXUID PlayerXUIDResolver,
	users authz.UserLookup, resolveFamily FamilyXUIDResolver) *PlayerAccess {
	return &PlayerAccess{
		enforced:      authz.Enforced(demoMode, authMode) && resolveXUID != nil && users != nil,
		resolveXUID:   resolveXUID,
		users:         users,
		resolveFamily: resolveFamily,
	}
}

// Enforced dit si la décision filtre réellement (faux : tout est accordé).
func (a *PlayerAccess) Enforced() bool { return a != nil && a.enforced }

// Allows dit si la session peut lire (write=false) ou écrire (write=true) le
// profil `slug` du titre courant, et si le slug désigne un profil connu.
//
//   - enforcement désactivé : accordé ;
//   - slug inconnu : accordé au seul admin (il verra le 404 du handler), refusé à
//     tout autre — un refus uniforme ne révèle pas l'existence d'un profil ;
//   - écriture : admin ou propriétaire direct (authz.CanWritePlayer) ;
//   - lecture : admin, propriétaire ou co-membre de groupe (authz.CanAccessPlayer).
//
// Une session nil ne porte aucun utilisateur : refus.
func (a *PlayerAccess) Allows(ctx context.Context, sess *domain.SessionData, slug string, write bool) (granted, known bool) {
	if !a.Enforced() {
		return true, true
	}
	user := authz.CurrentUser(sess, a.users)
	profileXUID, found := a.resolveXUID(ctx, slug)
	if !found {
		return user != nil && user.Role == domain.RoleAdmin, false
	}
	if write {
		return authz.CanWritePlayer(true, user, profileXUID), true
	}
	var family map[string]bool
	if a.resolveFamily != nil {
		family = a.resolveFamily(ctx)
	}
	return authz.CanAccessPlayer(true, user, profileXUID, family), true
}

// writeIntentKey : clé de contexte de l'intention d'écriture de la requête.
type writeIntentKey struct{}

// withWriteIntent pose dans le contexte la classification de la requête par la
// garde de propriété (IsMutatingRequest), pour les gardes en aval qui décident sur
// un autre slug que celui du chemin (acteur Prestige).
func withWriteIntent(ctx context.Context, write bool) context.Context {
	return context.WithValue(ctx, writeIntentKey{}, write)
}

// RequestWrites dit si la requête en cours écrit, tel que la garde de propriété
// l'a classée. Vrai quand la classification est absente (requête qui n'est pas
// passée par la garde) : le cas inconnu se décide comme une écriture, la règle
// la plus stricte.
func RequestWrites(ctx context.Context) bool {
	write, ok := ctx.Value(writeIntentKey{}).(bool)
	return !ok || write
}
