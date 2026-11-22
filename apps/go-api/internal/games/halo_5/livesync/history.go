package livesync

// history.go — état « fin de l'historique vue » du joueur Halo 5 : lecture et pose de la
// clé sync_meta history_complete_at de sa player DB, la même clé et les mêmes helpers
// qu'Infinite (syncpkg.HistoryComplete / MarkHistoryComplete). Câblés dans
// Deps.HistoryComplete et Deps.MarkHistoryComplete.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/platform/dblease"
	syncpkg "levelup/go-api/internal/sync"
)

// historyCompleteProbe rend la sonde Deps.HistoryComplete d'un joueur : vrai quand la clé est
// posée. Une player DB inouvrable compte comme « complet » (journalisé), comme la lecture en
// échec d'Infinite : un delta ordinaire vaut mieux qu'un parcours intégral à chaque passe.
func historyCompleteProbe(playerPath, gamertag string) func(ctx context.Context) bool {
	return func(ctx context.Context) bool {
		db, err := syncpkg.OpenPlayerDB(playerPath)
		if err != nil {
			slog.WarnContext(ctx, "h5 sync: player DB inouvrable, état de l'historique inconnu — delta ordinaire",
				"gamertag", gamertag, "err", err)
			return true
		}
		defer db.Close()
		return syncpkg.HistoryComplete(ctx, db.SQLDb())
	}
}

// historyCompleteMarker rend le hook Deps.MarkHistoryComplete d'un joueur : pose la clé sous
// le verrou d'écrivain de sa player DB (un seul writer par base). Un verrou ou une ouverture
// en échec est journalisé et la clé reste absente : la passe suivante rattrape encore.
func historyCompleteMarker(playerPath, gamertag string) func(ctx context.Context) {
	return func(ctx context.Context) {
		lctx, cancel := context.WithTimeout(ctx, dblease.PlayerLeaseTimeout)
		defer cancel()
		writer, err := dblease.AcquireWriterCtx(lctx, nil, playerPath, dblease.KindPlayer)
		if err != nil {
			slog.WarnContext(ctx, "h5 sync: verrou player DB indisponible — fin d'historique non rangée",
				"gamertag", gamertag, "err", err)
			return
		}
		defer writer.Release()
		db, err := syncpkg.OpenPlayerDB(playerPath)
		if err != nil {
			slog.WarnContext(ctx, "h5 sync: player DB inouvrable — fin d'historique non rangée",
				"gamertag", gamertag, "err", err)
			return
		}
		defer db.Close()
		syncpkg.MarkHistoryComplete(ctx, db.SQLDb(), gamertag)
	}
}
