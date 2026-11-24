// Package scheduler — auto_sync_manual.go : le cycle demandé à la main (« Synchroniser maintenant »).
//
// Le clic lance, en arrière-plan, le MÊME cycle que le tick périodique (RunOnceTrigger, déclencheur
// "manual") : rattrapage de l'historique compris, auto-sync activée ou non. Jamais deux cycles à la fois :
// un clic pendant un cycle est refusé (port.ErrSyncCycleBusy), et un clic moins de manualCycleMinGap après la
// fin d'un cycle, de n'importe quelle origine, aussi (*port.SyncCooldownError). Le tick périodique et la commande
// du menu de la zone de notification ne sont pas soumis à ce délai. Le cycle appartient au cycle de vie du
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

// manualCycleMinGap est le délai minimal entre la fin d'un cycle (toute origine) et le démarrage d'un cycle
// demandé par « Synchroniser maintenant » : un compte ne peut pas enchaîner les cycles de tous les joueurs.
const manualCycleMinGap = 2 * time.Minute

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
	if wait := s.CooldownRemaining(); wait > 0 {
		s.cycleRunning.Store(false)
		return &port.SyncCooldownError{RetryAfter: wait}
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
		s.signalReplan(res.debt)
		slog.InfoContext(ctx, "auto_sync: cycle demandé terminé", "total", res.Total, "synced", res.Synced,
			"failed", res.Failed, "joueurs_en_rattrapage", len(res.debt.history),
			"films_en_retard", res.debt.films.Total(),
			"duration", res.Duration.Round(time.Millisecond))
	}()
	return nil
}

// ManualWindow rend la fin du dernier cycle connu, toute origine, et l'instant à partir duquel un cycle demandé
// est de nouveau accepté : cette fin plus manualCycleMinGap, ou zéro quand le délai est écoulé ou qu'aucun cycle
// n'est connu (port.SyncNowService). La fin d'un cycle relue du disque au démarrage compte : un redémarrage ne
// remet pas le délai à zéro.
func (s *AutoSyncScheduler) ManualWindow() (lastCycleAt, availableAt time.Time) {
	s.snapshotMu.RLock()
	last := s.lastCycleAt
	s.snapshotMu.RUnlock()
	if last.IsZero() {
		return time.Time{}, time.Time{}
	}
	if until := last.Add(manualCycleMinGap); until.Round(0).After(s.clock().Round(0)) {
		return last, until
	}
	return last, time.Time{}
}

// CooldownRemaining rend le délai avant qu'un cycle demandé soit de nouveau accepté ; zéro = accepté.
func (s *AutoSyncScheduler) CooldownRemaining() time.Duration {
	_, until := s.ManualWindow()
	if until.IsZero() {
		return 0
	}
	return until.Round(0).Sub(s.clock().Round(0))
}

// CycleRunning dit si un cycle tourne en ce moment, quelle qu'en soit l'origine (port.SyncNowService).
func (s *AutoSyncScheduler) CycleRunning() bool { return s.cycleRunning.Load() }
