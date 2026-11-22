// Package service — bootstrap_ownership.go : la propriété des profils.
//
// Deux questions distinctes. « Ce profil m'est-il VISIBLE ? » (le mien, ceux de
// mes co-membres de groupe, tout le parc pour un admin) : filterOwnedPlayers,
// OwnedPlayers, BuildPlayersList. « Ce profil est-il LE MIEN ? », plus fine :
// DirectOwnerFor — c'est le cas du compteur « amis en jeu », qui compte les
// joueurs de mon cercle SAUF les miens.
package service

import (
	"context"
	"fmt"

	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/ctxkeys"
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

// defaultPlayerIndex rend l'indice, dans players (visibles et ordonnés par LoadPlayers), du
// joueur par défaut de la session : son profil en propre (DirectOwnerFor) s'il est dans la
// liste, sinon le premier. Sans cela l'ordre alphabétique mettrait en tête le profil d'un
// co-membre de groupe ou, pour un admin, n'importe quel profil du parc.
func (s *BootstrapService) defaultPlayerIndex(sess *domain.SessionData, players []domain.PlayerSummary) int {
	owns := s.DirectOwnerFor(sess)
	for i, p := range players {
		if owns(p.XUID) {
			return i
		}
	}
	return 0
}

// writablePlayerSlugs rend les slugs des profils de `players` sur lesquels la
// session peut ÉCRIRE (authz.CanWritePlayer : propriétaire direct ou admin). Sans
// propriété appliquée (démo, auth non activée, pas de user store), tous les profils
// sont inscriptibles, comme le sont alors toutes les routes. Liste jamais nil : le
// front la lit comme un ensemble.
func (s *BootstrapService) writablePlayerSlugs(sess *domain.SessionData, players []domain.PlayerSummary) []string {
	enforced := authz.Enforced(s.cfg.DemoMode, s.cfg.AuthMode) && s.userLookup != nil
	var user *domain.User
	if enforced {
		user = authz.CurrentUser(sess, s.userLookup)
	}
	out := make([]string, 0, len(players))
	for _, p := range players {
		if authz.CanWritePlayer(enforced, user, p.XUID) {
			out = append(out, p.PlayerSlug)
		}
	}
	return out
}

// filterOwnedPlayers restreint la liste aux profils accessibles par l'utilisateur
// courant (ADR 0029). Retourne la liste intacte si l'enforcement est désactivé
// (mode demo / auth non activée) ou si le user store n'est pas câblé. Un admin
// voit tout ; un utilisateur standard voit son xuid + les profils de la famille
// (familyXUIDs, #21 Phase A) pour que le sélecteur L1 liste bien la famille.
func (s *BootstrapService) filterOwnedPlayers(sess *domain.SessionData, players []domain.PlayerSummary, familyXUIDs map[string]bool) []domain.PlayerSummary {
	if !authz.Enforced(s.cfg.DemoMode, s.cfg.AuthMode) || s.userLookup == nil {
		return players
	}
	user := authz.CurrentUser(sess, s.userLookup)
	out := make([]domain.PlayerSummary, 0, len(players))
	for _, p := range players {
		if authz.CanAccessPlayer(true, user, p.XUID, familyXUIDs) {
			out = append(out, p)
		}
	}
	return out
}

// resolveCoMembers retourne l'ensemble des xuids accessibles par le user courant
// via partage de groupe (co-membres). Nil si pas de résolveur câblé, pas d'user
// lié, ou aucun groupe → CanAccessPlayer retombe sur propriétaire-only strict.
func (s *BootstrapService) resolveCoMembers(sess *domain.SessionData) map[string]bool {
	if s.coMembers == nil || s.userLookup == nil {
		return nil
	}
	user := authz.CurrentUser(sess, s.userLookup)
	if user == nil || user.XUID == "" {
		return nil
	}
	return s.coMembers(user.XUID)
}

// BuildPlayersList construit la liste des joueurs pour GET /api/v1/players.
// S4 / audit M2 (lot S) : la liste est restreinte aux profils possédés par
// l'utilisateur courant (les siens + ses co-membres de groupe), comme
// available_players dans Build. En demo/single-user, filterOwnedPlayers no-ope
// (liste complète) — l'invariant d'onboarding est préservé.
func (s *BootstrapService) BuildPlayersList(ctx context.Context, sess *domain.SessionData) (*domain.PlayersListResponse, error) {
	players, err := s.OwnedPlayers(ctx, sess)
	if err != nil {
		return nil, fmt.Errorf("BuildPlayersList: %w", err)
	}
	// defaultSlug est calculé APRÈS filtrage → ne pointe jamais sur un profil non possédé.
	var defaultSlug *string
	if len(players) > 0 {
		slug := players[s.defaultPlayerIndex(sess, players)].PlayerSlug
		defaultSlug = &slug
	}
	return &domain.PlayersListResponse{
		Items:             players,
		DefaultPlayerSlug: defaultSlug,
	}, nil
}

// OwnedPlayers retourne les joueurs du TITRE COURANT (ctx) visibles côté front
// (profils auth-only exclus) ET accessibles par la session — les siens plus ses
// co-membres de groupe (Couche A, ADR 0029). En démo / mono-utilisateur le
// filtre no-ope et la liste complète est rendue.
//
// SOURCE UNIQUE de cette combinaison « joueurs du titre + visibles + possédés » :
// consommée par GET /players (BuildPlayersList) et par la présence en jeu
// (PresenceService). Toute nouvelle surface listant des joueurs passe par ici —
// la ré-écrire ailleurs, c'est risquer d'oublier une des trois étapes.
func (s *BootstrapService) OwnedPlayers(ctx context.Context, sess *domain.SessionData) ([]domain.PlayerSummary, error) {
	players, err := s.cfg.LoadPlayers(ctxkeys.TitleSlug(ctx))
	if err != nil {
		return nil, fmt.Errorf("OwnedPlayers: %w", err)
	}
	players = excludeAuthOnly(players)
	return s.filterOwnedPlayers(sess, players, s.resolveCoMembers(sess)), nil
}

// excludeAuthOnly retire les profils auth-only (existant uniquement pour la
// gestion des tokens, sans suivi de stats) d'une liste destinée au front. Ces
// profils ne doivent jamais apparaître dans le sélecteur L1 ni les favoris de
// gamertag (Escouade/Explorer). La résolution token côté serveur continue de
// les voir via cfg.LoadPlayers (non filtré).
func excludeAuthOnly(players []domain.PlayerSummary) []domain.PlayerSummary {
	out := make([]domain.PlayerSummary, 0, len(players))
	for _, p := range players {
		if p.AuthOnly {
			continue
		}
		out = append(out, p)
	}
	return out
}
