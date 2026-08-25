// Package service — presence_service.go : « qui est en jeu, là, maintenant ».
//
// Sert GET /api/v1/presence : l'état de présence des joueurs suivis (sélecteur
// de joueur de la navigation) et le nombre d'amis en jeu. Deux sources bien
// distinctes :
//
//   - les JOUEURS suivis viennent du watcher, qui poll déjà leur présence toutes
//     les 30 s (aucun appel Xbox n'est fait ici) ;
//   - les AMIS viennent d'un appel batch Xbox à la demande, mis en cache (cf.
//     presence_friends.go) — il n'y a pas de poller dédié pour eux.
//
// Rien de tout cela n'est vital : watcher éteint, Xbox indisponible ou Réglages
// vides rendent une réponse vide en 200. Une présence manquante est un détail
// d'affichage ; une erreur 500 toutes les 30 s dans le shell n'en serait pas un.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
)

// TrackedPresence est l'état de présence d'un joueur suivi, tel que le watcher
// le connaît. Type neutre volontairement : le package service ne dépend ni du
// daemon ni du client Xbox, l'adaptation se fait au composition root.
//
// TitleSlug/TitleName sont le titre TRACKÉ sur lequel le joueur est vu (quel que
// soit son titre configuré), vides s'il n'est sur aucun titre suivi.
type TrackedPresence struct {
	Gamertag  string
	TitleSlug string
	TitleName string
}

// TrackedPresenceSource rend l'état de présence de tous les joueurs suivis.
// nil (watcher désactivé) ou tranche vide (daemon éteint) sont des cas normaux.
type TrackedPresenceSource func() []TrackedPresence

// OwnedPlayersFunc rend les joueurs du titre courant accessibles à la session
// (BootstrapService.OwnedPlayers au composition root).
type OwnedPlayersFunc func(ctx context.Context, sess *domain.SessionData) ([]domain.PlayerSummary, error)

// PresenceService construit le PresenceSnapshot de GET /api/v1/presence.
type PresenceService struct {
	ownedPlayers OwnedPlayersFunc
	tracked      TrackedPresenceSource
	friends      *FriendPresenceCounter // nil = comptage des amis désactivé
}

// NewPresenceService crée le service. Toutes les dépendances sont optionnelles :
// une dépendance absente retire sa part de la réponse (liste vide, compteur à
// zéro) sans jamais faire échouer l'endpoint.
func NewPresenceService(ownedPlayers OwnedPlayersFunc, tracked TrackedPresenceSource) *PresenceService {
	return &PresenceService{ownedPlayers: ownedPlayers, tracked: tracked}
}

// WithFriends branche le comptage des amis en jeu (cf. presence_friends.go).
// Sans lui, friends_in_game vaut toujours 0.
func (s *PresenceService) WithFriends(c *FriendPresenceCounter) *PresenceService {
	s.friends = c
	return s
}

// GetSnapshot construit la réponse de GET /api/v1/presence.
//
// La liste rendue est l'INTERSECTION des joueurs suivis par le watcher et des
// joueurs accessibles à la session : le watcher dit qui a un état de présence
// connu, l'ownership dit lesquels cet utilisateur a le droit de voir, et le
// player_slug (que le watcher ignore) vient de la configuration. Watcher éteint
// ⇒ intersection vide ⇒ liste vide, ce qui est exact : on ne sait rien.
//
// Ne retourne jamais d'erreur par conception (cf. en-tête du fichier) : chaque
// source indisponible est loggée puis remplacée par sa valeur neutre.
func (s *PresenceService) GetSnapshot(ctx context.Context, sess *domain.SessionData) *domain.PresenceSnapshot {
	snap := &domain.PresenceSnapshot{Players: []domain.PlayerPresence{}}

	byGamertag := s.trackedByGamertag()
	if len(byGamertag) > 0 {
		for _, p := range s.loadPlayers(ctx, sess) {
			t, ok := byGamertag[p.Gamertag]
			if !ok {
				continue
			}
			snap.Players = append(snap.Players, domain.PlayerPresence{
				PlayerSlug: p.PlayerSlug,
				Gamertag:   p.Gamertag,
				// « En jeu » = vu sur un titre suivi, PAS `inGame` du watcher :
				// ce dernier ne vaut vrai que sur le titre configuré du joueur,
				// et dirait « hors jeu » d'un joueur Halo 5 lançant Infinite.
				InGame:    t.TitleSlug != "",
				TitleSlug: t.TitleSlug,
				TitleName: t.TitleName,
			})
		}
	}

	if s.friends != nil {
		snap.FriendsInGame = s.friends.Count(ctx)
	}
	return snap
}

// loadPlayers rend les joueurs accessibles, ou une tranche vide si la source est
// absente ou en erreur (la présence ne justifie pas de casser le shell).
func (s *PresenceService) loadPlayers(ctx context.Context, sess *domain.SessionData) []domain.PlayerSummary {
	if s.ownedPlayers == nil {
		return nil
	}
	players, err := s.ownedPlayers(ctx, sess)
	if err != nil {
		slog.WarnContext(ctx, "presence: chargement des joueurs échoué — liste vide", "err", err)
		return nil
	}
	return players
}

// trackedByGamertag indexe l'état du watcher par gamertag. Un même gamertag peut
// être suivi sur PLUSIEURS titres (un watcher par titre) : l'entrée qui porte un
// titre courant gagne, car c'est celle qui sait où le joueur joue réellement.
func (s *PresenceService) trackedByGamertag() map[string]TrackedPresence {
	if s.tracked == nil {
		return nil
	}
	list := s.tracked()
	out := make(map[string]TrackedPresence, len(list))
	for _, t := range list {
		if t.Gamertag == "" {
			continue
		}
		if prev, ok := out[t.Gamertag]; ok && prev.TitleSlug != "" {
			continue
		}
		out[t.Gamertag] = t
	}
	return out
}
