package replayartifacts

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/platform/gameproc"
)

// reporterEnModeJeu dit si la cuisson de ce match est reportée parce que Halo tourne, et le compte.
//
// MODE JEU : tant que Halo tourne, aucune cuisson ne DÉMARRE (le décodage d'un film pèse sur le
// processeur et la mémoire que la partie utilise). Le film est déjà persisté, le rattrapage
// construira l'artefact après la partie. Jamais un kill : la porte se consulte avant chaque
// lancement, une cuisson commencée va à son terme. Sans cuisson câblée, il n'y a rien à reporter.
func reporterEnModeJeu(ctx context.Context, d Deps, b *bilanCuisson) bool {
	if d.BuildOne == nil || gameproc.Default().HeavyWorkAllowed() {
		return false
	}
	if b.reportesModeJeu == 0 {
		slog.InfoContext(ctx, "game_mode: cuisson des rejeux différée jusqu'à la fin de la partie",
			"gamertag", d.Gamertag, "titleSlug", d.TitleSlug)
	}
	b.reportesModeJeu++
	return true
}
