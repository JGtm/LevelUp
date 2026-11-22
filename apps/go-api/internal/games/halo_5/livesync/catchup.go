package livesync

// catchup.go — rattrapage de l'historique Halo 5 par la synchronisation automatique.
//
// Un delta (CollectRecentMatches) s'arrête au premier match connu : une première synchronisation
// interrompue laisserait un trou définitif derrière les matchs déjà rangés. Tant que la fin de
// l'historique n'a pas été vue pour le joueur (internal/sync/historystate), Runner.RunDelta fait donc
// une passe de rattrapage par la mécanique du backfill (RunBackfill : connus sautés, persistance page
// par page), bornée à historystate.PassBudget() matchs insérés. La fin se range seulement quand la
// passe a vu la fin de la liste sans erreur de page ni de persistance.

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/dblease"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/historystate"
)

// historyState : ce que RunDelta sait de l'historique du joueur au début du cycle.
type historyState struct {
	catchUp bool // fin jamais vue, la passe est une passe de rattrapage
	pending bool // l'historique est (ou peut être) incomplet : vrai tant que la fin n'est pas rangée
}

// readHistoryState lit l'état de l'historique. Sans hook : complet (delta ordinaire, runner testable
// sans base joueur). Base joueur illisible : delta ordinaire mais en attente, pour que l'ordonnanceur
// replanifie un cycle court (la lecture de l'état ne bloque jamais le delta).
func (r *Runner) readHistoryState(ctx context.Context) historyState {
	if r.deps.HistoryComplete == nil {
		return historyState{}
	}
	complete, err := r.deps.HistoryComplete(ctx)
	if err != nil {
		r.logger.WarnContext(ctx, "h5 sync: lecture de l'historique impossible, delta ordinaire",
			"gamertag", r.deps.Viewer.Gamertag, "err", err)
		return historyState{pending: true}
	}
	return historyState{catchUp: !complete, pending: !complete}
}

// catchUpBudget est le budget de la passe : celui de l'historique, ou le plafond plus serré que
// l'appelant demande (le tick du suivi en direct passe un petit MaxMatches).
func catchUpBudget(opts domain.SyncOptions) int {
	budget := historystate.PassBudget()
	if opts.MaxMatches > 0 && opts.MaxMatches < budget {
		return opts.MaxMatches
	}
	return budget
}

// runCatchUp exécute une passe de rattrapage et range ses compteurs dans res. known est l'ensemble
// des matchs déjà rangés, lu par RunDelta (la lecture unique de loadKnownMatchIDs) et complété au
// fil des pages. Rend les compteurs de capture de la passe et vrai quand la fin de l'historique est rangée.
func (r *Runner) runCatchUp(ctx context.Context, src halo5.CaptureSource, resolveXUID func(string) string,
	known map[string]bool, budget int, res *domain.SyncResult) (halo5.CaptureStats, bool) {
	capture := r.deps.CapturePage
	if capture == nil {
		capture = capturePageFrom(src)
	}
	deps := BackfillDeps{
		CapturePage: capture,
		Viewer:      r.deps.Viewer,
		ResolveXUID: resolveXUID,
		LoadKnown:   func(context.Context) (map[string]bool, error) { return known, nil },
		PersistPage: func(ctx context.Context, batches []*persist.MatchBatch) ([]*persist.MatchBatch, []string) {
			done, errs := r.deps.PersistAll(ctx, batches)
			res.Errors = append(res.Errors, errs...)
			for _, b := range done {
				tallyBatch(res, b)
			}
			return done, errs
		},
		Budget: budget,
	}
	stats, err := RunBackfill(ctx, deps, 0, r.logger)
	if err != nil {
		res.AddError("h5 backfill: " + err.Error())
	}
	tallySkipped(res, stats.MatchesSeen)
	res.HistoryExhausted = stats.HistoryEnd

	// Règle de la fin : fin de liste vue, aucune erreur de page, aucune erreur de persistance. Un match
	// dont la timeline ou le carnage manque n'est pas une erreur (compté par stats.EventsFailed et
	// stats.CarnageFailed : le match est rangé sans eux).
	if err != nil || !stats.HistoryEnd || stats.PersistErrors > 0 {
		r.logger.InfoContext(ctx, "h5 sync: passe de rattrapage — historique encore incomplet",
			"gamertag", r.deps.Viewer.Gamertag, "inserted", stats.Inserted, "budget", budget,
			"budget_reached", stats.BudgetReached, "history_end", stats.HistoryEnd,
			"persist_errors", stats.PersistErrors, "page_error", err != nil)
		return stats.Capture, false
	}
	if r.deps.MarkHistoryComplete == nil {
		return stats.Capture, false
	}
	return stats.Capture, r.deps.MarkHistoryComplete(ctx)
}

// playerHistoryComplete lit l'état de l'historique dans la base joueur Halo 5 (handle RW ref-compté
// de sync.OpenPlayerDB, comme les autres hooks du runner). Une lecture qui échoue est journalisée par
// historystate.Complete et compte pour « complet » ; seule l'ouverture de la base remonte en erreur.
func playerHistoryComplete(ctx context.Context, playerDBPath string) (bool, error) {
	db, err := syncpkg.OpenPlayerDB(playerDBPath)
	if err != nil {
		return false, fmt.Errorf("h5 history: open player DB: %w", err)
	}
	defer db.Close()
	return historystate.Complete(ctx, db.SQLDb()), nil
}

// markPlayerHistoryComplete range la fin d'historique dans la base joueur Halo 5, sous le bail
// d'écriture du joueur (règle ART n°4, même acquisition que le moteur V1), attendu au plus
// dblease.PlayerLeaseTimeout : le cycle ne reste pas bloqué sur un bail tenu ailleurs, la passe
// suivante réessaie. Rend vrai quand la clé est rangée.
func markPlayerHistoryComplete(ctx context.Context, playerDBPath, gamertag string) bool {
	leaseCtx, cancel := context.WithTimeout(ctx, dblease.PlayerLeaseTimeout)
	defer cancel()
	lease, err := dblease.AcquireWriterCtx(leaseCtx, nil, playerDBPath, dblease.KindPlayer)
	if err != nil {
		slog.WarnContext(ctx, "h5 sync: bail joueur indisponible — fin d'historique non rangée",
			"gamertag", gamertag, "err", err)
		return false
	}
	defer lease.Release()
	db, err := syncpkg.OpenPlayerDB(playerDBPath)
	if err != nil {
		slog.WarnContext(ctx, "h5 sync: ouverture de la base joueur échouée — fin d'historique non rangée",
			"gamertag", gamertag, "err", err)
		return false
	}
	defer db.Close()
	return historystate.MarkComplete(ctx, db.SQLDb(), gamertag)
}
