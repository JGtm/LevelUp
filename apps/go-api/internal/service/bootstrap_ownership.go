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
// DEUX RÉGIMES, décision produit du 2026-08-25 :
//
//   - propriété APPLIQUÉE (auth password/xbox + user store câblé) : possédé =
//     « le xuid lié de l'utilisateur de la session ». Le compteur d'amis vaut
//     alors « mon cercle visible MOINS mes propres profils » ;
//   - propriété NON APPLIQUÉE (LEVELUP_AUTH_MODE=none — la valeur par défaut —,
//     mode démo, ou aucun user store) : FAUX pour tout profil. Sans identités,
//     il n'existe aucun « possédé en propre » à retrancher : l'instance ENTIÈRE
//     est le cercle de son opérateur, et le compteur vaut « tous les joueurs
//     visibles en jeu ». Rendre vrai ici (comportement d'origine) mettait la
//     pastille à zéro en permanence sur la configuration par défaut, soit une
//     fonctionnalité livrée éteinte (règle n°11 du dépôt).
//
// Utilisateur non identifié (session anonyme, identité Halo non liée) : faux.
func (s *BootstrapService) OwnsPlayerDirectly(sess *domain.SessionData, playerXUID string) bool {
	if !authz.Enforced(s.cfg.DemoMode, s.cfg.AuthMode) || s.userLookup == nil {
		return false
	}
	user := authz.CurrentUser(sess, s.userLookup)
	if user == nil || user.XUID == "" || playerXUID == "" {
		return false
	}
	return user.XUID == playerXUID
}
