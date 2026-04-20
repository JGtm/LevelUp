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
}

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
// Les match_ids déjà vus sont filtrés.
func (q *MatchQueue) Enqueue(req MatchRequest) {
	q.seenMu.Lock()
	var newIDs []string
	for _, id := range req.MatchIDs {
		key := req.Gamertag + ":" + id
		if !q.seen[key] {
			q.seen[key] = true
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
		slog.Info("match_queue: requête ajoutée",
			"gamertag", req.Gamertag,
			"match_count", len(newIDs),
		)
	default:
		slog.Warn("match_queue: file pleine, requête ignorée",
			"gamertag", req.Gamertag,
			"match_count", len(newIDs),
		)
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
