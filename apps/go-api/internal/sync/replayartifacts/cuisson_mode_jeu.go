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

// reporterVerrouTenu range une cuisson dont le verrou solo de decodage est tenu par un autre decodage de la machine (la
// file de regeneration des rejeux, l'action admin) : le match est REPORTE au cycle suivant, jamais compte en echec, et
// jamais un avertissement par film — une seule ligne INFO par cycle, le bilan donne le nombre.
func reporterVerrouTenu(ctx context.Context, d Deps, b *bilanCuisson, busy error) {
	if b.reportesVerrou == 0 {
		slog.InfoContext(ctx, "post-sync: cuisson des rejeux reportee — un autre decodage de film tient la machine, "+
			"reprise au cycle suivant", "gamertag", d.Gamertag, "titleSlug", d.TitleSlug, "detenteur", busy)
	}
	b.reportesVerrou++
}
