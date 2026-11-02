package scheduler

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/platform/gameproc"
)

// runHeavy lance un cycle de tâche lourde (audit, balayage, recalcul) si le jeu ne tourne pas ; si
// le jeu tourne, le cycle attend la fin de la partie (porte [gameproc.Mode.WaitHeavyAllowed]).
// Une tâche déjà lancée n'est jamais interrompue : la porte se consulte AVANT le lancement, jamais
// pendant. Rend sans lancer si ctx est annulé pendant l'attente.
//
// Garde-rail : heavy_gate_guard_test.go impose que les quatre boucles lourdes passent par ici.
func runHeavy(ctx context.Context, task string, cycle func(context.Context)) {
	err := gameproc.Default().WaitHeavyAllowed(ctx, func() {
		slog.InfoContext(ctx, "game_mode: tâche lourde différée jusqu'à la fin de la partie", "tache", task)
	})
	if err != nil {
		return
	}
	cycle(ctx)
}
