package main

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/platform/coalesce"
)

// drainCoalescedRecomputes attend, à l'arrêt, la fin des recalculs coalescés en vol
// (exclusion de match, liste d'amis : coalesce.Runs), au plus coalesce.MaxRunDuration, la
// borne d'un recalcul. Toute nouvelle demande est refusée dès le début de l'attente. À
// appeler après l'arrêt du serveur HTTP et avant duckdb.CloseAll : un recalcul lit la base
// partagée et écrit en base joueur (ordre tenu par TestShutdown_DrainsRecomputesBeforeCloseAll).
func drainCoalescedRecomputes() {
	ctx, cancel := context.WithTimeout(context.Background(), coalesce.MaxRunDuration)
	defer cancel()
	start := time.Now()
	if err := coalesce.Drain(ctx); err != nil {
		slog.ErrorContext(ctx, "shutdown: recalculs coalescés encore en vol à l'échéance, bases fermées quand même",
			"err", err, "max_wait", coalesce.MaxRunDuration.String())
		return
	}
	slog.InfoContext(ctx, "shutdown: recalculs coalescés terminés", "duration", time.Since(start).String())
}
