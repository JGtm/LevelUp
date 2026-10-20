package main

import (
	"context"
	"log/slog"
	"time"
)

const (
	// shutdownBudget est la durée totale de la séquence d'arrêt, comptée depuis la demande
	// d'arrêt jusqu'à la dernière fermeture. docker-compose.yml accorde stop_grace_period
	// (45 s) : au-delà, Docker tue le process sans fermer les bases (WAL à rejouer au boot).
	// shutdown_budget_test.go lit le compose et échoue si la marge disparaît.
	shutdownBudget = 40 * time.Second

	// shutdownCloseReserve est la part du budget gardée pour la fermeture des bases
	// (duckdb.CloseAll et les fermetures qui suivent), qui ne se borne pas : aucune phase
	// qui précède n'a le droit de l'entamer.
	shutdownCloseReserve = 5 * time.Second
)

// Bornes propres à chaque phase d'attente, appliquées sous le budget restant : une phase
// n'attend jamais plus que sa borne, ni plus que ce qu'il reste du budget.
const (
	phaseWatcherCap    = 10 * time.Second // Daemon.Stop borne déjà son attente à 8 s
	phaseHTTPCap       = 15 * time.Second
	phaseSchedulerCap  = 3 * time.Second
	phaseGateCap       = 5 * time.Second
	phaseBatchDrainCap = 5 * time.Second
	phaseBatchCloseCap = 5 * time.Second
	phaseWorkerCap     = 5 * time.Second
	phaseCheckpointCap = 5 * time.Second
)

// shutdownClock suit le budget d'arrêt : un instant de départ, une durée totale, une
// réserve pour la fermeture. L'horloge est injectable (tests).
type shutdownClock struct {
	start   time.Time
	total   time.Duration
	reserve time.Duration
	now     func() time.Time
}

func newShutdownClock(total, reserve time.Duration) *shutdownClock {
	return &shutdownClock{start: time.Now(), total: total, reserve: reserve, now: time.Now}
}

// remaining est le temps dont disposent encore les phases d'attente : le budget moins ce
// qui est écoulé et moins la réserve de fermeture. Jamais négatif.
func (c *shutdownClock) remaining() time.Duration {
	left := c.total - c.reserve - c.now().Sub(c.start)
	if left < 0 {
		return 0
	}
	return left
}

// allot rend la durée accordée à une phase de borne `limit` : la plus petite de la borne et
// du budget restant.
func (c *shutdownClock) allot(limit time.Duration) time.Duration {
	return min(limit, c.remaining())
}

// shutdownPhase est une phase d'attente de la séquence d'arrêt. run reçoit un contexte
// échu à la durée accordée et rend vrai quand la phase est allée à son terme dans ce délai.
type shutdownPhase struct {
	name  string
	limit time.Duration
	run   func(ctx context.Context) bool
}

// runPhase exécute une phase sous sa durée accordée. Une phase qui dépasse (run rend faux)
// est nommée dans un Warn avec ce qu'on lui avait accordé et ce qu'il restait du budget :
// la séquence continue, l'arrêt n'attend jamais au-delà de son budget.
func (c *shutdownClock) runPhase(ctx context.Context, p shutdownPhase) bool {
	allotted := c.allot(p.limit)
	phaseCtx, cancel := context.WithTimeout(ctx, allotted)
	defer cancel()
	begin := c.now()
	if p.run(phaseCtx) {
		return true
	}
	slog.WarnContext(ctx, "shutdown: phase hors budget, la séquence continue",
		"phase", p.name,
		"allotted", allotted.String(),
		"limit", p.limit.String(),
		"waited", c.now().Sub(begin).String(),
		"budget_exhausted", c.remaining() == 0)
	return false
}

// runAll exécute les phases dans l'ordre. Une phase qui dépasse ne retient pas les suivantes :
// quand le budget est épuisé, chacune reçoit une durée nulle et rend la main aussitôt.
func (c *shutdownClock) runAll(ctx context.Context, phases []shutdownPhase) {
	for _, p := range phases {
		c.runPhase(ctx, p)
	}
}

// blocking adapte une attente qui ne prend pas de contexte (WaitGroup, Close) : elle
// tourne dans une goroutine et la phase rend la main quand le contexte échoit. La
// goroutine abandonnée se termine avec le process.
func blocking(fn func()) func(context.Context) bool {
	return func(ctx context.Context) bool {
		done := make(chan struct{})
		go func() {
			fn()
			close(done)
		}()
		select {
		case <-done:
			return true
		case <-ctx.Done():
			return false
		}
	}
}
