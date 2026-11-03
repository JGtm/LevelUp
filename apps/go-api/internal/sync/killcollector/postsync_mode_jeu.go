package killcollector

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/gameproc"
)

// ReporteeEnModeJeu dit si l'étape kill source du post-sync est reportée parce que Halo tourne, et le compte.
//
// MODE JEU : cette étape DÉCODE des films dans le processus (kill-feed, positions, tirs), le travail
// lourd que la cuisson des rejeux refuse de démarrer pendant la partie. Elle est donc reportée de même :
// rien n'est perdu, le backlog se relit en base à chaque cycle et le premier cycle hors partie le
// rattrape. Les inserts courts et le réseau du reste du cycle ne sont pas touchés.
func ReporteeEnModeJeu(ctx context.Context, gamertag string, matchsInseres int) bool {
	if gameproc.Default().HeavyWorkAllowed() {
		return false
	}
	observability.IncCounter(gameproc.MetricHeavyDeferred)
	slog.InfoContext(ctx, "game_mode: kill source reportée jusqu'à la fin de la partie",
		"gamertag", gamertag, "matchs_inseres", matchsInseres)
	return true
}
