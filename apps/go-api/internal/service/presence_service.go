// Package service — presence_service.go : « qui est en jeu, là, maintenant ».
//
// Sert GET /api/v1/presence : l'état de présence des joueurs suivis (sélecteur
// de joueur de la navigation) et le nombre d'amis en jeu. UNE SEULE source pour
// les deux, le watcher, qui poll déjà la présence des joueurs suivis : AUCUN
// appel Xbox n'est fait ici, ni pour les joueurs, ni pour les amis.
//
// « MES AMIS » = LES JOUEURS INSCRITS DANS L'APP QUI NE SONT PAS LES MIENS,
// dans la limite de mon cercle (ADR 0029) — décision produit du 2026-08-25.
// Concrètement : les joueurs que l'utilisateur voit déjà dans son sélecteur (les
// siens plus ceux de ses co-membres de groupe), PRIVÉS de ceux dont il est
// directement propriétaire, et actuellement en jeu. Le compte est donc PERSONNEL
// à chaque utilisateur : deux utilisateurs d'une même instance, sur le même état
// de watcher, obtiennent deux valeurs différentes, et un utilisateur étranger à
// mon groupe n'entre jamais dans mon compte (il n'est pas dans ma liste visible).
// La liste `friend_gamertags` des Réglages n'a plus AUCUN rôle dans la présence.
//
// Ce que la réponse expose de ces amis : un ENTIER, rien d'autre. Les identités
// servies restent celles de la liste `players`, filtrée par visibilité comme
// avant.
//
// Rien de tout cela n'est vital : watcher éteint ou joueurs illisibles rendent
// une réponse vide en 200. Une présence manquante est un détail d'affichage ;
// une erreur 500 toutes les 30 s dans le shell n'en serait pas un.
package service

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
)

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

// DirectOwnerFunc dit si l'utilisateur de la session est DIRECTEMENT
// propriétaire du profil joueur donné — son xuid lié — par opposition à
// « visible via un groupe » (BootstrapService.OwnsPlayerDirectly au composition
// root).
//
// C'est la SEULE chose que ce service a besoin de savoir en plus de la liste
// qu'il sert déjà : qui est visible, et pourquoi (groupe, famille, rôle admin,
// mode démo), est décidé par OwnedPlayersFunc et n'est jamais redécidé ici.
type DirectOwnerFunc func(sess *domain.SessionData, playerXUID string) bool

// PresenceService construit le PresenceSnapshot de GET /api/v1/presence.
type PresenceService struct {
	ownedPlayers OwnedPlayersFunc
	tracked      TrackedPresenceSource
	// directOwner : nil = comptage des amis désactivé (compteur à zéro). Sans
	// prédicat de propriété, on ne peut pas dire d'un joueur visible qu'il n'est
	// pas le sien — et compter ses propres joueurs comme des amis serait faux.
	directOwner DirectOwnerFunc
}

// NewPresenceService crée le service. Toutes les dépendances sont optionnelles :
// une dépendance absente retire sa part de la réponse (liste vide, compteur à
// zéro) sans jamais faire échouer l'endpoint.
func NewPresenceService(ownedPlayers OwnedPlayersFunc, tracked TrackedPresenceSource) *PresenceService {
	return &PresenceService{ownedPlayers: ownedPlayers, tracked: tracked}
}

// WithFriends branche le comptage des amis en jeu en fournissant le prédicat de
// propriété directe. Sans lui, friends_in_game vaut toujours 0.
func (s *PresenceService) WithFriends(directOwner DirectOwnerFunc) *PresenceService {
	s.directOwner = directOwner
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
// LE COMPTEUR D'AMIS SORT DE CETTE MÊME BOUCLE, et c'est délibéré : il compte
// les joueurs VISIBLES (donc de mon cercle) que je ne possède pas, avec le même
// prédicat « en jeu » et la même borne de fraîcheur que la manette servie à côté
// — un seul chemin, pas deux définitions de « en jeu » à faire diverger.
//
// Les profils démo et auth-only ne sont pas re-filtrés ici : ils n'ont aucun état
// de watcher (le daemon ne suit que domain.SyncablePlayers, cf.
// cmd/server/main.go) et les auth-only sont déjà retirés en amont par
// BootstrapService.OwnedPlayers — ils tombent donc d'eux-mêmes à l'intersection.
//
// Ne retourne jamais d'erreur par conception (cf. en-tête du fichier) : chaque
// source indisponible est loggée puis remplacée par sa valeur neutre.
func (s *PresenceService) GetSnapshot(ctx context.Context, sess *domain.SessionData) *domain.PresenceSnapshot {
	snap := &domain.PresenceSnapshot{Players: []domain.PlayerPresence{}}

	byGamertag := s.trackedByGamertag()
	if len(byGamertag) == 0 {
		return snap
	}
	for _, p := range s.loadPlayers(ctx, sess) {
		t, ok := byGamertag[p.Gamertag]
		if !ok {
			continue
		}
		// « En jeu » = vu sur un titre suivi, PAS `inGame` du watcher : ce
		// dernier ne vaut vrai que sur le titre configuré du joueur, et dirait
		// « hors jeu » d'un joueur Halo 5 lançant Infinite.
		inGame := t.TitleSlug != ""
		snap.Players = append(snap.Players, domain.PlayerPresence{
			PlayerSlug: p.PlayerSlug,
			Gamertag:   p.Gamertag,
			InGame:     inGame,
			TitleSlug:  t.TitleSlug,
			TitleName:  t.TitleName,
		})
		if inGame && s.isFriend(sess, p) {
			snap.FriendsInGame++
		}
	}
	return snap
}

// isFriend : ce joueur visible est-il « un ami » — un joueur de mon cercle dont
// je ne suis pas le propriétaire ?
//
// Sans prédicat de propriété câblé, personne n'est un ami : mieux vaut un
// compteur à zéro que compter ses propres joueurs. Un profil sans xuid n'est
// jamais compté non plus — il ne peut être attribué à personne, et la propriété
// ne se déduit pas d'un gamertag.
func (s *PresenceService) isFriend(sess *domain.SessionData, p domain.PlayerSummary) bool {
	if s.directOwner == nil || p.XUID == "" {
		return false
	}
	return !s.directOwner(sess, p.XUID)
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
