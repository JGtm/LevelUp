// Package scheduler — auto_sync_manual.go : le cycle demandé à la main (« Synchroniser maintenant »).
//
// Le clic lance, en arrière-plan, le MÊME cycle que le tick périodique (RunOnceTrigger, déclencheur
// "manual") : rattrapage de l'historique compris, auto-sync activée ou non. Jamais deux cycles à la fois :
// un clic pendant un cycle est refusé (port.ErrSyncCycleBusy). Le cycle appartient au cycle de vie du
// serveur : il s'arrête avec son contexte et l'arrêt l'attend avant la fermeture des bases.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	gosync "sync"
	"time"

	"levelup/go-api/internal/platform/adminstate"
	"levelup/go-api/internal/port"
)

var _ port.SyncNowService = (*AutoSyncScheduler)(nil)

// BindManualCycles donne aux cycles demandés à la main le contexte du serveur (annulé à l'arrêt) et le
// groupe d'attente que l'arrêt draine avant la fermeture des bases. Sans appel, StartSyncNow refuse.
func (s *AutoSyncScheduler) BindManualCycles(ctx context.Context, wg *gosync.WaitGroup) *AutoSyncScheduler {
	s.manualCtx, s.manualWG = ctx, wg
	return s
}

// StartSyncNow démarre un cycle de synchronisation en arrière-plan (port.SyncNowService).
func (s *AutoSyncScheduler) StartSyncNow() error {
	ctx, wg := s.manualCtx, s.manualWG
	if ctx == nil || wg == nil || ctx.Err() != nil {
		return port.ErrSyncCycleUnavailable
	}
	if !s.cycleRunning.CompareAndSwap(false, true) {
		return port.ErrSyncCycleBusy
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer s.cycleRunning.Store(false)
		defer func() {
			if rec := recover(); rec != nil {
				slog.ErrorContext(ctx, "auto_sync: panic pendant le cycle demandé", "recover", fmt.Sprint(rec))
			}
		}()
		res := s.runCycle(ctx, adminstate.TriggerManual)
		s.signalReplan(res.historyPending)
		slog.InfoContext(ctx, "auto_sync: cycle demandé terminé", "total", res.Total, "synced", res.Synced,
			"failed", res.Failed, "joueurs_en_rattrapage", len(res.historyPending),
			"duration", res.Duration.Round(time.Millisecond))
	}()
	return nil
}

// CycleRunning dit si un cycle tourne en ce moment, quelle qu'en soit l'origine (port.SyncNowService).
func (s *AutoSyncScheduler) CycleRunning() bool { return s.cycleRunning.Load() }
