// Package watcher — match_queue.go : file d'attente de matchs avec déduplication.
//
// Recoit des match_ids de plusieurs sources (MatchPoller de différents joueurs)
// et les délivre au sync coordinator sans doublon.
package watcher

import (
	"log/slog"
	"sync"
)

// MatchRequest est une demande de sync pour un ensemble de matchs.
type MatchRequest struct {
	Gamertag string
	XUID     string
	MatchIDs []string
	// TitleSlug porte le titre du joueur (MT-11 / PMT-3). Vide = halo_infinite.
	// Propagé au CoordinatorRequest → clé de dédup composite + ctx du moteur. Le
	// watcher ne suit aujourd'hui que halo_infinite (titleReg.MatchPresence) ; ce
	// champ est le point de câblage pour un 2e titre suivi (à remplir depuis le
	// td.Slug résolu à la détection de présence).
	TitleSlug string
}

// maxSeenEntries borne le cache de déduplication `seen` (W7, revue 2026-06-01).
// Sans borne, il accumule 1 entrée par (gamertag, match_id) jamais purgée →
// fuite mémoire lente. Au-delà, on reset : un éventuel re-traitement est
// inoffensif (dédup Coordinator par gamertag + idempotence du persister via le
// pré-check match_registry EXISTS).
const maxSeenEntries = 10000

// MatchQueue est une file d'attente de matchs avec déduplication.
type MatchQueue struct {
	ch      chan MatchRequest
	seen    map[string]bool
	seenMu  sync.Mutex
	maxSize int
}

// NewMatchQueue crée une file avec une capacité donnée.
func NewMatchQueue(maxSize int) *MatchQueue {
	return &MatchQueue{
		ch:      make(chan MatchRequest, maxSize),
		seen:    make(map[string]bool),
		maxSize: maxSize,
	}
}

// Enqueue ajoute une requête à la file après déduplication des match_ids.
// Les match_ids déjà vus sont filtrés. Un match_id n'est marqué `seen` qu'APRÈS
// un enqueue réussi : si la file est pleine (drop), le prochain poll doit pouvoir
// re-tenter ces matchs au lieu de les perdre définitivement (revue 2026-06-02).
func (q *MatchQueue) Enqueue(req MatchRequest) {
	q.seenMu.Lock()
	var newIDs []string
	for _, id := range req.MatchIDs {
		if !q.seen[req.Gamertag+":"+id] {
			newIDs = append(newIDs, id)
		}
	}
	q.seenMu.Unlock()

	if len(newIDs) == 0 {
		slog.Debug("match_queue: tous les matchs déjà en file",
			"gamertag", req.Gamertag,
			"original_count", len(req.MatchIDs),
		)
		return
	}

	filtered := MatchRequest{
		Gamertag: req.Gamertag,
		XUID:     req.XUID,
		MatchIDs: newIDs,
	}

	select {
	case q.ch <- filtered:
		// Enqueue réussi → on mémorise pour ne pas réenfiler ces matchs. Une rare
		// course (deux Enqueue concurrents passant le pré-check) est inoffensive :
		// le Coordinator dédup par gamertag + le persister est idempotent.
		q.markSeen(req.Gamertag, newIDs)
		slog.Info("match_queue: requête ajoutée",
			"gamertag", req.Gamertag,
			"match_count", len(newIDs),
		)
	default:
		// File pleine : on NE marque PAS seen → re-tenté au prochain poll.
		slog.Error("match_queue: file pleine, matchs non enfilés (re-tentés au prochain poll)",
			"gamertag", req.Gamertag,
			"match_count", len(newIDs),
		)
	}
}

// markSeen mémorise les match_ids effectivement enfilés, avec borne anti-fuite.
func (q *MatchQueue) markSeen(gamertag string, ids []string) {
	q.seenMu.Lock()
	defer q.seenMu.Unlock()
	for _, id := range ids {
		q.seen[gamertag+":"+id] = true
	}
	if len(q.seen) > maxSeenEntries {
		slog.Debug("match_queue: cache seen réinitialisé (borne atteinte)", "size", len(q.seen))
		q.seen = make(map[string]bool)
	}
}

// Dequeue retourne le channel de lecture pour consommer les requêtes.
func (q *MatchQueue) Dequeue() <-chan MatchRequest {
	return q.ch
}

// Len retourne le nombre de requêtes en attente.
func (q *MatchQueue) Len() int {
	return len(q.ch)
}

// ClearSeen réinitialise le cache de déduplication.
func (q *MatchQueue) ClearSeen() {
	q.seenMu.Lock()
	q.seen = make(map[string]bool)
	q.seenMu.Unlock()
}
