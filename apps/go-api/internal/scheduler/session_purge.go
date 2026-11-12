// Package scheduler — session_purge.go : purge récurrente des sessions HTTP expirées.
package scheduler

import (
	"context"
	"log/slog"
	"time"
)

// SessionPurgePeriod : intervalle entre deux purges des sessions expirées.
const SessionPurgePeriod = 6 * time.Hour

// SessionPurger supprime les sessions dont le TTL est dépassé et rend leur nombre.
// Satisfait par *session.Store.
type SessionPurger interface {
	PurgeExpired() int
}

// RunSessionPurgeLoop purge tout de suite (rattrape le retard d'un arrêt long) puis toutes les
// SessionPurgePeriod, jusqu'à l'annulation de ctx. Bloquante : l'appelant la lance dans sa
// goroutine. Ne touche qu'aux fichiers de session, pas aux bases.
func RunSessionPurgeLoop(ctx context.Context, purger SessionPurger) {
	runSessionPurgeLoop(ctx, purger, SessionPurgePeriod)
}

func runSessionPurgeLoop(ctx context.Context, purger SessionPurger, period time.Duration) {
	if n := purger.PurgeExpired(); n > 0 {
		slog.InfoContext(ctx, "session: purge initiale sessions expirées", "removed", n)
	}
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n := purger.PurgeExpired(); n > 0 {
				slog.InfoContext(ctx, "session: purge sessions expirées", "removed", n)
			}
		}
	}
}
