// Package scheduler — auto_sync_loop.go : la boucle périodique de l'auto-sync et sa cadence.
//
// La boucle tire un cycle à chaque tick. Tant qu'au moins un joueur du cycle a un historique incomplet,
// le cycle suivant vient après catchUpCycleInterval au lieu de l'intervalle réglé : le rattrapage de
// l'historique avance par passes bornées (sync/historystate) et n'attend pas des heures entre deux.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/adminstate"
	syncv2 "levelup/go-api/internal/sync/v2"
)

// catchUpCycleInterval est l'intervalle entre deux cycles tant qu'au moins un joueur du cycle précédent
// a un historique incomplet (passe en échec, ou fin d'historique pas encore vue). Dès que tous les
// joueurs sont complets, la boucle revient à l'intervalle réglé (resolveInterval). Quand l'intervalle
// réglé est plus court, il garde la main : le rattrapage ne ralentit jamais la synchronisation.
// Le compte à rebours part de la fin du cycle, pas de son début.
const catchUpCycleInterval = 10 * time.Minute

// cycleTicker est l'horloge de la boucle de Run : *time.Ticker en production, un faux dans les tests.
type cycleTicker interface {
	C() <-chan time.Time
	Reset(d time.Duration)
	Stop()
}

// wallTicker adapte *time.Ticker à cycleTicker.
type wallTicker struct{ t *time.Ticker }

func (w wallTicker) C() <-chan time.Time   { return w.t.C }
func (w wallTicker) Reset(d time.Duration) { w.t.Reset(d) }
func (w wallTicker) Stop()                 { w.t.Stop() }

// tickState est l'état de cadence de la boucle de Run : l'intervalle réglé dans les settings, et la
// période du ticker en ce moment (celle du rattrapage pendant un rattrapage).
type tickState struct {
	configured time.Duration
	period     time.Duration
}

// clock rend l'heure murale de l'ordonnanceur.
func (s *AutoSyncScheduler) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *AutoSyncScheduler) makeTicker(d time.Duration) cycleTicker {
	if s.newTicker != nil {
		return s.newTicker(d)
	}
	return wallTicker{t: time.NewTicker(d)}
}

// NextCycleAt rend l'instant (horloge murale) du prochain cycle périodique, zéro tant que la boucle
// de Run n'a pas démarré. Un cycle demandé à la main ne le déplace pas. Il ne dit pas si l'auto-sync
// est activée : un cycle n'a lieu que si le réglage l'est à ce moment-là.
func (s *AutoSyncScheduler) NextCycleAt() time.Time {
	s.snapshotMu.RLock()
	defer s.snapshotMu.RUnlock()
	return s.nextCycleAt
}

func (s *AutoSyncScheduler) setNextCycleAt(t time.Time) {
	s.snapshotMu.Lock()
	s.nextCycleAt = t
	s.snapshotMu.Unlock()
}

// Run démarre la boucle périodique. Doit être lancé dans une goroutine.
// Se termine proprement à l'annulation de ctx.
func (s *AutoSyncScheduler) Run(ctx context.Context) {
	interval := s.CurrentInterval()
	ticker := s.makeTicker(interval)
	defer ticker.Stop()
	st := &tickState{configured: interval, period: interval}
	s.setNextCycleAt(s.clock().Add(interval))

	slog.InfoContext(ctx, "auto_sync: scheduler démarré",
		"interval", interval,
		"interval_hours", int(interval.Hours()),
		"pool_size", s.poolSizeSafe(),
	)

	for {
		select {
		case <-ticker.C():
			s.runTick(ctx, st, ticker, "")
		case pending := <-s.replanCh:
			// Un cycle demandé hors tick (clic « Synchroniser maintenant ») vient de finir : la
			// cadence suit son issue, comme après un tick.
			s.replanFromCycle(ctx, st, ticker, pending)
		case reason := <-s.catchUpCh:
			// Rattrapage demandé (RequestCatchUp) : le même tick, exécuté dans CETTE boucle, donc
			// jamais en même temps qu'un tick périodique.
			s.runTick(ctx, st, ticker, reason)
		case <-ctx.Done():
			slog.InfoContext(ctx, "auto_sync: arrêt du scheduler (contexte annulé)")
			return
		}
	}
}

// runTick exécute un tick : relit les settings (activation, intervalle), lance un cycle, journalise son
// issue puis recale le ticker (planNextCycle). catchUpReason non vide : le tick est un rattrapage
// demandé hors horloge, non le tick périodique.
func (s *AutoSyncScheduler) runTick(ctx context.Context, st *tickState, ticker cycleTicker, catchUpReason string) {
	// Sprint B1 commit 17 : event_id sur chaque tick pour tracer le
	// cycle scheduler complet à travers les modules (scheduler →
	// auth → sync → provider → pool → handlers post-sync).
	tickCtx, tickID := logging.WithEvent(ctx, "scheduler.tick")
	cfg, err := s.settings.Load()
	if err != nil {
		slog.WarnContext(tickCtx, "auto_sync: lecture settings échouée — tick ignoré", "err", err)
		s.setNextCycleAt(s.clock().Add(st.period))
		return
	}
	if !cfg.SpnkrAutoSyncEnabled {
		slog.DebugContext(tickCtx, "auto_sync: désactivé dans les settings, tick ignoré")
		s.setNextCycleAt(s.clock().Add(st.period))
		return
	}
	if newInterval := resolveInterval(cfg.SpnkrAutoSyncIntervalMinutes, cfg.SpnkrAutoSyncIntervalHours); newInterval != st.configured {
		slog.InfoContext(tickCtx, "auto_sync: intervalle mis à jour", "old", st.configured, "new", newInterval)
		st.configured = newInterval
	}
	slog.InfoContext(tickCtx, "auto_sync: tick démarré", "event", tickID, "rattrapage", catchUpReason)
	res := s.RunOnceTrigger(tickCtx, adminstate.TriggerCron)
	if res.busy {
		// Un cycle demandé à la main tourne : il recalera la cadence à sa fin.
		s.setNextCycleAt(s.clock().Add(st.period))
		return
	}
	slog.InfoContext(tickCtx, "auto_sync: cycle terminé",
		"total", res.Total,
		"synced", res.Synced,
		"skipped", res.Skipped,
		"failed", res.Failed,
		"duration", res.Duration.Round(time.Millisecond),
	)
	if res.Failed > 0 {
		slog.WarnContext(tickCtx, "auto_sync: des joueurs ont échoué — consulter le snapshot diag",
			"failed_count", res.Failed,
		)
	}
	if res.Total > 0 && res.Skipped == res.Total {
		slog.WarnContext(tickCtx, "auto_sync: aucun joueur synchronisé — vérifier le pool",
			"pool_size", s.poolSizeSafe(),
			"hint", "voir GET /api/v1/_diag/auto-sync/snapshot pour le détail",
		)
	}
	s.planNextCycle(tickCtx, st, ticker, res.historyPending)
}

// planNextCycle recale le ticker après un cycle : catchUpCycleInterval si au moins un joueur reste en
// rattrapage (sans dépasser l'intervalle réglé), l'intervalle réglé sinon. Une ligne de journal par
// cycle dit combien de joueurs rattrapent et l'intervalle retenu.
func (s *AutoSyncScheduler) planNextCycle(ctx context.Context, st *tickState, ticker cycleTicker, pending []string) {
	next := st.configured
	if len(pending) > 0 && catchUpCycleInterval < next {
		next = catchUpCycleInterval
	}
	slog.InfoContext(ctx, "auto_sync: cycle suivant planifié",
		"joueurs_en_rattrapage", len(pending),
		"gamertags", pending,
		"intervalle", next,
		"intervalle_regle", st.configured,
	)
	ticker.Reset(next)
	st.period = next
	s.setNextCycleAt(s.clock().Add(next))
}

// replanFromCycle recale la cadence après un cycle demandé hors tick. Auto-sync désactivée : rien ne
// change (aucun cycle périodique n'a lieu, et la cadence se relira à l'activation).
func (s *AutoSyncScheduler) replanFromCycle(ctx context.Context, st *tickState, ticker cycleTicker, pending []string) {
	cfg, err := s.settings.Load()
	if err != nil {
		slog.WarnContext(ctx, "auto_sync: lecture settings échouée — cadence inchangée après le cycle demandé", "err", err)
		return
	}
	if !cfg.SpnkrAutoSyncEnabled {
		return
	}
	st.configured = resolveInterval(cfg.SpnkrAutoSyncIntervalMinutes, cfg.SpnkrAutoSyncIntervalHours)
	s.planNextCycle(ctx, st, ticker, pending)
}

// signalReplan transmet à la boucle de Run les joueurs restés en rattrapage à la fin d'un cycle hors
// tick ; une issue plus récente remplace celle qui attend encore. Ne bloque jamais.
func (s *AutoSyncScheduler) signalReplan(pending []string) {
	select {
	case <-s.replanCh:
	default:
	}
	select {
	case s.replanCh <- pending:
	default:
	}
}

// historyIncomplete dit si l'historique du joueur est incomplet, lu sans écrire. Sans lecteur câblé, ou
// lecture en échec (journalisée), le joueur ne compte pas comme en rattrapage : même défaut que
// historystate.Complete, qui prend une lecture en échec pour « complet ».
func (s *AutoSyncScheduler) historyIncomplete(ctx context.Context, p domain.PlayerSummary) bool {
	if s.historyReader == nil {
		return false
	}
	complete, err := s.historyReader.HistoryComplete(ctx, resolveTitleSlug(p), p.Gamertag)
	if err != nil {
		slog.WarnContext(ctx, "auto_sync: lecture de l'état de l'historique échouée — joueur non compté en rattrapage",
			"gamertag", p.Gamertag, "title_slug", resolveTitleSlug(p), "err", err)
		return false
	}
	return !complete
}

// pendingHistoryOf liste les gamertags des joueurs dont l'historique est incomplet d'après la base,
// pour un cycle qui n'a rien pu dire d'eux (erreur globale du cycle).
func (s *AutoSyncScheduler) pendingHistoryOf(ctx context.Context, players []domain.PlayerSummary) []string {
	var out []string
	for _, p := range players {
		if s.historyIncomplete(ctx, p) {
			out = append(out, p.Gamertag)
		}
	}
	return out
}

// pendingHistoryAfterV2 liste les gamertags en rattrapage à la fin d'un cycle V2 : l'issue du joueur
// fait foi quand elle existe et que la passe n'a pas échoué ; un joueur sans issue ou en échec n'a pas
// pu dire où en est son historique, la base le dit (historyIncomplete).
func (s *AutoSyncScheduler) pendingHistoryAfterV2(ctx context.Context, players []domain.PlayerSummary, per map[string]syncv2.PlayerOutcome) []string {
	var out []string
	for _, p := range players {
		o, ok := per[p.PlayerSlug]
		var pending bool
		if ok && o.Status != "failed" {
			pending = o.HistoryPending
		} else {
			pending = s.historyIncomplete(ctx, p)
		}
		if pending {
			out = append(out, p.Gamertag)
		}
	}
	return out
}
