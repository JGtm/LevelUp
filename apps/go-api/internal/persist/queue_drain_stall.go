package persist

// queue_drain_stall.go : l'attente du journal d'écriture, et sa variante qui rend la main quand le
// worker n'avance plus.

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// drainPollInterval : pas du sondage du dossier du journal par l'attente. Compromis entre réactivité
// (le worker termine vite) et charge (ReadDir n'est pas gratuit).
const drainPollInterval = 50 * time.Millisecond

// ErrDrainStalled est rendu (enveloppé) par DrainWhileProgressing quand le journal n'est pas vide et
// qu'aucun lot n'a été acquitté pendant le délai demandé : le worker est bloqué (base indisponible,
// lot en échec laissé à la reprise périodique). Les lots en attente restent durables dans le journal.
var ErrDrainStalled = errors.New("persist: drain sans progrès — aucun lot acquitté pendant le délai")

// DrainWhileProgressing attend, comme Drain, que le journal soit vide, mais rend ErrDrainStalled
// (enveloppée : lots en attente, délai) dès qu'aucun lot n'a été acquitté pendant stall. Un
// acquittement relance le délai : une file lente qui avance n'est jamais interrompue. stall <= 0 : Drain.
func (q *BatchQueue) DrainWhileProgressing(ctx context.Context, stall time.Duration) error {
	return q.drainLoop(ctx, stall)
}

// drainLoop sonde le journal jusqu'à ce qu'il soit vide. Arrêts anticipés : contexte fini
// (ctx.Err()), disjoncteur ouvert (ErrDrainCircuitBreaker), et, si stall > 0, aucun acquittement
// pendant stall (ErrDrainStalled).
func (q *BatchQueue) drainLoop(ctx context.Context, stall time.Duration) error {
	ticker := time.NewTicker(drainPollInterval)
	defer ticker.Stop()

	lastAcks, lastProgress := q.acks.Load(), time.Now()
	for {
		n, err := q.PendingCount()
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		// Disjoncteur : trop d'échecs consécutifs du worker alors qu'il reste des lots, il est
		// cassé — échec rapide au lieu d'attendre le délai.
		if q.consecutiveFailures.Load() >= int32(circuitBreakerThreshold) {
			return fmt.Errorf("%w (failures=%d, pending=%d)",
				ErrDrainCircuitBreaker,
				q.consecutiveFailures.Load(),
				n)
		}
		if stall > 0 {
			if acks := q.acks.Load(); acks != lastAcks {
				lastAcks, lastProgress = acks, time.Now()
			} else if time.Since(lastProgress) >= stall {
				return fmt.Errorf("%w (pending=%d, stall=%s)", ErrDrainStalled, n, stall)
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
