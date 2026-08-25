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
	"time"

	"levelup/go-api/internal/domain"
)

// friendsCountBudget borne l'attente du comptage d'amis DANS la réponse.
//
// Les joueurs suivis viennent du watcher (déjà en mémoire, aucune sortie) ; les
// amis, eux, peuvent exiger un aller-retour Xbox. Sans borne, /presence hérite
// de la latence de Xbox — jusqu'à la vingtaine de secondes d'un incident — sur
// une requête que le shell rejoue toutes les 30 s et dont l'utilisateur attend
// la liste des JOUEURS. 3 s : large pour un lot nominal (~200 ms), assez court
// pour qu'un incident ne se voie pas.
//
// Dépassement = zéro, pas d'erreur : la pastille d'amis disparaît, la liste des
// joueurs est servie. Le contexte annulé fait AVORTER le lot en cours : rien
// n'alimente le cache, et l'échec arme le backoff d'échec du compteur (~30 s).
// Dans un régime où Xbox dépasse durablement le budget, le compteur reste donc
// à zéro — dégradation assumée (consignée au plan, lot F ronde 2, constat 1).
const friendsCountBudget = 3 * time.Second

// presenceFreshnessWindow : au-delà, le titre courant d'un joueur n'est plus
// servi comme un fait présent.
//
// POURQUOI CETTE BORNE EXISTE, ET POURQUOI ELLE EST LICITE ICI. Le titre courant
// est mémorisé par le handler de présence du daemon ; il n'est jamais effacé par
// un minuteur. Si le poll d'un joueur s'arrête (XSTS mort, 429 prolongé, réseau),
// la dernière valeur reste en mémoire indéfiniment : le shell afficherait une
// manette « en jeu » sur un joueur déconnecté depuis des heures.
//
// La borne suppose que le témoin de vivacité avance TOUT SEUL tant que le poll
// vit — vérifié sur pièces : `RESTPoller.tickOnce` (watcher/rest_poller.go)
// appelle son handler à CHAQUE poll réussi, sans filtre de changement d'état, à
// `restPollInterval` = 10 s ; et le handler du daemon pose `pw.RecordEvent(...)`
// AVANT tout filtrage (watcher/daemon.go). Un joueur immobile fait donc avancer
// LastEventAt toutes les 10 s. Si un jour les events ne partaient qu'aux
// CHANGEMENTS d'état, cette borne effacerait le titre d'un joueur bel et bien en
// partie : elle serait à retirer, pas à rallonger.
//
// 3 minutes = 18 polls nominaux. Les backoffs d'erreur du poller (30 s réseau /
// transitoire) tiennent largement dedans ; seuls le backoff rate-limit (5 min) et
// un arrêt réel dépassent.
const presenceFreshnessWindow = 3 * time.Minute

// TrackedPresence est l'état de présence d'un joueur suivi, tel que le watcher
// le connaît. Type neutre volontairement : le package service ne dépend ni du
// daemon ni du client Xbox, l'adaptation se fait au composition root.
//
// TitleSlug/TitleName sont le titre TRACKÉ sur lequel le joueur est vu (quel que
// soit son titre configuré), vides s'il n'est sur aucun titre suivi.
//
// LastEventAt est l'instant du dernier event de présence reçu pour ce joueur —
// le témoin de VIVACITÉ du poll, pas d'activité en jeu. Zéro = aucun event reçu.
type TrackedPresence struct {
	Gamertag    string
	TitleSlug   string
	TitleName   string
	LastEventAt time.Time
}

// fresh rend l'entrée telle quelle si son témoin de vivacité est récent, et
// BLANCHIE (aucun titre) sinon. Appliquée à l'INGESTION, avant l'arbitrage entre
// watchers d'un même gamertag : sans cela une entrée périmée porteuse d'un titre
// l'emporterait sur une entrée fraîche qui dit « hors jeu ».
func (t TrackedPresence) fresh(now time.Time) TrackedPresence {
	if t.TitleSlug == "" && t.TitleName == "" {
		return t
	}
	if t.LastEventAt.IsZero() || now.Sub(t.LastEventAt) > presenceFreshnessWindow {
		t.TitleSlug, t.TitleName = "", ""
	}
	return t
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
	// friendsBudget : friendsCountBudget en production ; les tests l'abaissent
	// pour ne pas attendre 3 s (même procédé que RESTPoller.WithInterval).
	friendsBudget time.Duration
}

// NewPresenceService crée le service. Toutes les dépendances sont optionnelles :
// une dépendance absente retire sa part de la réponse (liste vide, compteur à
// zéro) sans jamais faire échouer l'endpoint.
func NewPresenceService(ownedPlayers OwnedPlayersFunc, tracked TrackedPresenceSource) *PresenceService {
	return &PresenceService{
		ownedPlayers:  ownedPlayers,
		tracked:       tracked,
		friendsBudget: friendsCountBudget,
	}
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
		snap.FriendsInGame = s.countFriendsWithinBudget(ctx)
	}
	return snap
}

// countFriendsWithinBudget rend le compteur d'amis, ou zéro si le budget expire.
//
// Le comptage tourne dans une goroutine et la réponse ne l'attend que le temps du
// budget : sans cela, un `Count` bloqué (Xbox lent, ou attente du singleflight
// d'un appelant précédent) tiendrait la requête HTTP ouverte d'autant. La
// goroutine n'est pas orpheline — son contexte est annulé au retour, ce qui coupe
// l'appel sortant, et le canal est tamponné pour qu'elle se termine même si plus
// personne ne lit.
func (s *PresenceService) countFriendsWithinBudget(ctx context.Context) int {
	budget := s.friendsBudget
	if budget <= 0 {
		budget = friendsCountBudget
	}
	cctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()

	done := make(chan int, 1)
	go func() { done <- s.friends.Count(cctx) }()

	select {
	case n := <-done:
		return n
	case <-cctx.Done():
		slog.DebugContext(ctx, "presence: comptage des amis hors budget — compteur à zéro",
			"budget", budget, "err", cctx.Err())
		return 0
	}
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
//
// Chaque entrée passe d'abord par `fresh` : un titre dont le témoin de vivacité
// est périmé n'est plus un titre (cf. presenceFreshnessWindow).
func (s *PresenceService) trackedByGamertag() map[string]TrackedPresence {
	if s.tracked == nil {
		return nil
	}
	list := s.tracked()
	now := time.Now()
	out := make(map[string]TrackedPresence, len(list))
	for _, raw := range list {
		if raw.Gamertag == "" {
			continue
		}
		t := raw.fresh(now)
		if prev, ok := out[t.Gamertag]; ok && prev.TitleSlug != "" {
			continue
		}
		out[t.Gamertag] = t
	}
	return out
}
