// Package sync — coordinator.go : coordination des syncs déclenchés par le watcher.
//
// Le SyncCoordinator :
//   - Consomme les MatchRequests depuis la MatchQueue
//   - Garantit au plus N syncs simultanés via sémaphore
//   - Empêche les syncs concurrents sur le même joueur (inFlight map)
//   - Appelle le SyncEngine pour chaque joueur+matchIDs
package sync

import (
	"context"
	"log/slog"
	"sync"
)

// SyncRunner exécute le sync pour un joueur avec les match_ids donnés.
type SyncRunner interface {
	RunSync(ctx context.Context, gamertag, xuid string, matchIDs []string) error
}

// CoordinatorRequest est une demande de sync entrante.
type CoordinatorRequest struct {
	Gamertag string
	XUID     string
	MatchIDs []string
}

// Coordinator coordonne les syncs avec contrôle de concurrence.
type Coordinator struct {
	runner       SyncRunner
	maxParallel  int
	sem          chan struct{}
	inFlightMu   sync.Mutex
	inFlight     map[string]bool // gamertag → en cours
	onComplete   func(gamertag string, err error)
}

// NewCoordinator crée un coordinateur avec limite de syncs parallèles.
func NewCoordinator(runner SyncRunner, maxParallel int) *Coordinator {
	if maxParallel < 1 {
		maxParallel = 1
	}
	return &Coordinator{
		runner:      runner,
		maxParallel: maxParallel,
		sem:         make(chan struct{}, maxParallel),
		inFlight:    make(map[string]bool),
	}
}

// SetOnComplete définit un callback appelé après chaque sync (succès ou erreur).
func (c *Coordinator) SetOnComplete(fn func(gamertag string, err error)) {
	c.onComplete = fn
}

// Submit soumet une requête de sync.
// Retourne immédiatement — le sync est exécuté en goroutine.
// Retourne false si le joueur a déjà un sync en cours.
func (c *Coordinator) Submit(ctx context.Context, req CoordinatorRequest) bool {
	c.inFlightMu.Lock()
	if c.inFlight[req.Gamertag] {
		c.inFlightMu.Unlock()
		slog.Info("coordinator: sync déjà en cours, requête ignorée",
			"gamertag", req.Gamertag,
		)
		return false
	}
	c.inFlight[req.Gamertag] = true
	c.inFlightMu.Unlock()

	go c.run(ctx, req)
	return true
}

// run exécute le sync avec sémaphore.
func (c *Coordinator) run(ctx context.Context, req CoordinatorRequest) {
	// Acquérir le sémaphore
	select {
	case c.sem <- struct{}{}:
	case <-ctx.Done():
		c.releaseInFlight(req.Gamertag)
		return
	}

	defer func() {
		<-c.sem // libérer le sémaphore
		c.releaseInFlight(req.Gamertag)
	}()

	slog.Info("coordinator: démarrage sync",
		"gamertag", req.Gamertag,
		"match_count", len(req.MatchIDs),
		"parallel_slots", c.maxParallel,
	)

	err := c.runner.RunSync(ctx, req.Gamertag, req.XUID, req.MatchIDs)
	if err != nil {
		slog.Error("coordinator: sync échoué",
			"gamertag", req.Gamertag,
			"err", err,
		)
	} else {
		slog.Info("coordinator: sync terminé",
			"gamertag", req.Gamertag,
		)
	}

	if c.onComplete != nil {
		c.onComplete(req.Gamertag, err)
	}
}

// releaseInFlight retire le joueur de la map inFlight.
func (c *Coordinator) releaseInFlight(gamertag string) {
	c.inFlightMu.Lock()
	delete(c.inFlight, gamertag)
	c.inFlightMu.Unlock()
}

// IsInFlight vérifie si un joueur a un sync en cours.
func (c *Coordinator) IsInFlight(gamertag string) bool {
	c.inFlightMu.Lock()
	defer c.inFlightMu.Unlock()
	return c.inFlight[gamertag]
}

// InFlightCount retourne le nombre de syncs en cours.
func (c *Coordinator) InFlightCount() int {
	c.inFlightMu.Lock()
	defer c.inFlightMu.Unlock()
	return len(c.inFlight)
}
