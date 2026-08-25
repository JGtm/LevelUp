// Package service — bootstrap_ownership.go : la propriété DIRECTE d'un profil.
//
// Complément de filterOwnedPlayers (bootstrap_service.go), qui répond à « ce
// profil m'est-il VISIBLE ? » (le mien, ceux de mes co-membres de groupe, tout
// le parc pour un admin). Certaines surfaces ont besoin de la question voisine,
// plus fine : « ce profil est-il LE MIEN ? » — c'est le cas du compteur
// « amis en jeu », qui compte les joueurs de mon cercle SAUF les miens.
//
// Fichier séparé de bootstrap_service.go, déjà au-delà du seuil de 500 lignes
// (dette gelée) : on ne l'agrandit pas.
package service

import (
	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/domain"
)

// OwnsPlayerDirectly indique si l'utilisateur de la session est DIRECTEMENT
// propriétaire du profil `playerXUID` — son xuid lié, et lui seul. Un profil
// accessible par appartenance à un groupe (co-membre) ou par le rôle admin
// n'est PAS possédé directement : c'est précisément la distinction que
// CanAccessPlayer ne fait pas, puisqu'elle décide de l'accès, pas de la
// propriété.
//
// Propriété NON appliquée (mode démo, auth désactivée, pas de user store) : vrai
// pour tout profil. L'instance est alors mono-utilisateur ou publique — tous les
// profils sont « les siens », et rien ne permettrait d'en désigner un comme
// appartenant à quelqu'un d'autre.
//
// Utilisateur non identifié (session anonyme, identité Halo non liée) : faux.
// Sans conséquence pour l'appelant actuel — la liste des joueurs visibles est
// alors vide (filterOwnedPlayers rejette tout), donc il n'y a rien à partitionner.
func (s *BootstrapService) OwnsPlayerDirectly(sess *domain.SessionData, playerXUID string) bool {
	if !authz.Enforced(s.cfg.DemoMode, s.cfg.AuthMode) || s.userLookup == nil {
		return true
	}
	user := authz.CurrentUser(sess, s.userLookup)
	if user == nil || user.XUID == "" || playerXUID == "" {
		return false
	}
	return user.XUID == playerXUID
}
