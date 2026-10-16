package main

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/platform/coalesce"
)

// drainCoalescedRecomputes, à l'arrêt, refuse toute nouvelle demande de recalcul coalescé
// (exclusion de match, liste d'amis : coalesce.Runs), annule les recalculs en vol et attend
// leur fin au plus coalesce.DrainWait : la suite de la séquence d'arrêt n'est jamais retardée
// au-delà. Un recalcul annulé est relancé par la prochaine demande, après le redémarrage. À
// appeler après l'arrêt du serveur HTTP et avant duckdb.CloseAll : un recalcul lit la base
// partagée et écrit en base joueur (ordre tenu par TestShutdown_DrainsRecomputesBeforeCloseAll).
func drainCoalescedRecomputes() {
	ctx, cancel := context.WithTimeout(context.Background(), coalesce.DrainWait)
	defer cancel()
	start := time.Now()
	if err := coalesce.Drain(ctx); err != nil {
		slog.ErrorContext(ctx, "shutdown: recalculs coalescés encore en vol à l'échéance, bases fermées quand même",
			"err", err, "max_wait", coalesce.DrainWait.String())
		return
	}
	slog.InfoContext(ctx, "shutdown: recalculs coalescés terminés", "duration", time.Since(start).String())
}
