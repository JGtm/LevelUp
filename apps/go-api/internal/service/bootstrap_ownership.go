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

// DirectOwnerFor résout l'utilisateur de la session UNE SEULE FOIS et rend le
// prédicat « ce profil est-il DIRECTEMENT le sien ? » — son xuid lié, et lui
// seul. Un profil accessible par appartenance à un groupe (co-membre) ou par le
// rôle admin n'est PAS possédé directement : c'est précisément la distinction
// que CanAccessPlayer ne fait pas, puisqu'elle décide de l'accès, pas de la
// propriété.
//
// POURQUOI UNE FABRIQUE et non un simple prédicat (sess, xuid) : l'identité est
// invariante sur toute la requête, alors que la résolution ne l'est pas —
// authz.CurrentUser passe par le user store, dont l'implémentation de production
// (platform/userstore) RELIT ET PARSE users.json à chaque appel. Appelée dans la
// boucle des joueurs en jeu, l'ancienne signature multipliait ces lectures par le
// nombre de joueurs. La fermeture rendue ici ne capture qu'un xuid.
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
// Utilisateur sans xuid propre — session anonyme, identité Halo non liée, ou
// compte ADMIN dont le xuid n'est pas renseigné : prédicat toujours faux, rien
// ne lui appartient en propre. Attention à ne PAS justifier ce faux par « sa
// liste visible est de toute façon vide » : c'est exact pour un utilisateur
// standard (CanAccessPlayer exige alors un xuid), mais FAUX pour un admin — le
// rôle accorde l'accès AVANT le test du xuid, sa liste vaut donc tout le parc.
// Conséquence assumée : un admin sans xuid compte tout le parc en jeu comme des
// amis, dans la ligne de la découverte admin déjà actée au plan (le compte est
// « ce que je vois, moins les miens », et un admin sans xuid ne possède rien).
func (s *BootstrapService) DirectOwnerFor(sess *domain.SessionData) DirectOwnerFunc {
	if !authz.Enforced(s.cfg.DemoMode, s.cfg.AuthMode) || s.userLookup == nil {
		return ownsNothing
	}
	user := authz.CurrentUser(sess, s.userLookup)
	if user == nil || user.XUID == "" {
		return ownsNothing
	}
	ownXUID := user.XUID
	return func(playerXUID string) bool {
		return playerXUID != "" && playerXUID == ownXUID
	}
}

// ownsNothing : le prédicat de l'utilisateur qui ne possède aucun profil en
// propre. Nommé plutôt que dupliqué en fermeture anonyme à chaque sortie.
func ownsNothing(string) bool { return false }
