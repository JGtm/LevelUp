// Package scheduler — auto_sync_loop.go : la boucle périodique de l'auto-sync et sa cadence.
//
// La boucle tire un cycle à chaque tick. Tant qu'au moins un joueur du cycle a un historique incomplet,
// OU qu'un arriéré de films reste à traiter sur un titre du cycle (source des frags, artefacts de
// rejeu), le cycle suivant vient après catchUpCycleInterval au lieu de l'intervalle réglé : le
// rattrapage avance par passes bornées (sync/historystate pour l'historique, budget de temps par
// étape pour les films) et n'attend pas des heures entre deux.
package scheduler

import (
	"context"
	"log/slog"
	gosync "sync"
	"sync/atomic"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/adminstate"
	"levelup/go-api/internal/port"
	syncv2 "levelup/go-api/internal/sync/v2"
)

// catchUpCycleInterval est l'intervalle entre deux cycles tant que le cycle précédent laisse une dette
// (cycleDebt) : au moins un joueur à l'historique incomplet (passe en échec, ou fin d'historique pas
// encore vue), ou un arriéré de films sur un titre du cycle. Dès que tout est à jour, la boucle revient
// à l'intervalle réglé (resolveInterval). Quand l'intervalle réglé est plus court, il garde la main : le
// rattrapage ne ralentit jamais la synchronisation. Le compte à rebours part de la fin du cycle, pas de
// son début.
const catchUpCycleInterval = 10 * time.Minute

// cycleDebt est ce qu'un cycle laisse à rattraper : de quoi décider si le cycle suivant vient vite.
type cycleDebt struct {
	// history : gamertags des joueurs dont l'historique est incomplet en fin de cycle.
	history []string
	// films : arriéré de films des titres du cycle (somme des titres), tel que les étapes film du
	// post-sync l'ont publié à la fin de leur dernière passe.
	films port.FilmBacklog
}

// pending dit s'il reste quelque chose à rattraper.
func (d cycleDebt) pending() bool { return len(d.history) > 0 || d.films.Pending() }

// cycleState regroupe l'état de la boucle de Run et des cycles hors tick ; AutoSyncScheduler l'embarque.
// Les champs protégés par snapshotMu le sont par celui de l'ordonnanceur.
type cycleState struct {
	// historyReader dit si l'historique d'un joueur est complet quand sa passe a échoué avant de pouvoir le
	// dire (WithHistoryReader). Nil : un joueur en échec ne compte pas comme en rattrapage.
	historyReader port.HistoryCompletionReader
	// filmBacklog dit l'arriéré de films d'un titre (WithFilmBacklogReader) : la cadence de rattrapage
	// tient tant qu'il n'est pas vide. Nil : aucun arriéré connu, la cadence ne suit que l'historique.
	filmBacklog port.FilmBacklogReader

	// newTicker et now sont les horloges de la boucle de Run ; nil = horloge réelle (tests : injectées).
	newTicker func(time.Duration) cycleTicker
	now       func() time.Time
	// nextCycleAt : instant (horloge murale) du prochain cycle périodique, mis à jour par la boucle de
	// Run ; zéro tant que la boucle n'a pas démarré. Protégé par snapshotMu.
	nextCycleAt time.Time

	// cycleRunning : un cycle tourne (RunOnceTrigger). Source de « jamais deux cycles à la fois » et du
	// refus d'un clic « Synchroniser maintenant » pendant un cycle.
	cycleRunning atomic.Bool
	// manualCtx et manualWG portent les cycles demandés à la main (BindManualCycles) : annulés à l'arrêt,
	// attendus avant la fermeture des bases. replanCh porte l'issue d'un cycle hors tick jusqu'à la boucle
	// de Run.
	manualCtx context.Context
	manualWG  *gosync.WaitGroup
	replanCh  chan cycleDebt

	// lastPasses : dernière passe par (titre, joueur) depuis le démarrage. Protégé par snapshotMu.
	lastPasses map[string]port.SyncPass
}

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
// de Run n'a pas démarré. La fin d'un cycle, périodique ou demandé à la main, le recale. Il ne dit pas
// si l'auto-sync est activée : un cycle n'a lieu que si le réglage l'est à ce moment-là.
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
		case debt := <-s.replanCh:
			// Un cycle demandé hors tick (clic « Synchroniser maintenant ») vient de finir : la
			// cadence suit son issue, comme après un tick.
			s.replanFromCycle(ctx, st, ticker, debt)
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
	s.planNextCycle(tickCtx, st, ticker, res.debt)
}

// planNextCycle recale le ticker après un cycle : catchUpCycleInterval s'il reste une dette (joueur en
// rattrapage d'historique ou arriéré de films), sans dépasser l'intervalle réglé ; l'intervalle réglé
// sinon. Une ligne de journal par cycle dit combien de joueurs rattrapent, combien de films sont en
// retard et l'intervalle retenu.
func (s *AutoSyncScheduler) planNextCycle(ctx context.Context, st *tickState, ticker cycleTicker, debt cycleDebt) {
	next := st.configured
	if debt.pending() && catchUpCycleInterval < next {
		next = catchUpCycleInterval
	}
	slog.InfoContext(ctx, "auto_sync: cycle suivant planifié",
		"joueurs_en_rattrapage", len(debt.history),
		"gamertags", debt.history,
		"films_en_retard", debt.films.Total(),
		"films_frags", debt.films.Frags,
		"films_rejeux", debt.films.Rejeux,
		"intervalle", next,
		"intervalle_regle", st.configured,
	)
	ticker.Reset(next)
	st.period = next
	s.setNextCycleAt(s.clock().Add(next))
}

// replanFromCycle recale la cadence après un cycle demandé hors tick. Auto-sync désactivée : rien ne
// change (aucun cycle périodique n'a lieu, et la cadence se relira à l'activation).
func (s *AutoSyncScheduler) replanFromCycle(ctx context.Context, st *tickState, ticker cycleTicker, debt cycleDebt) {
	cfg, err := s.settings.Load()
	if err != nil {
		slog.WarnContext(ctx, "auto_sync: lecture settings échouée — cadence inchangée après le cycle demandé", "err", err)
		return
	}
	if !cfg.SpnkrAutoSyncEnabled {
		return
	}
	st.configured = resolveInterval(cfg.SpnkrAutoSyncIntervalMinutes, cfg.SpnkrAutoSyncIntervalHours)
	s.planNextCycle(ctx, st, ticker, debt)
}

// signalReplan transmet à la boucle de Run la dette laissée par un cycle hors tick ; une issue plus
// récente remplace celle qui attend encore. Ne bloque jamais.
func (s *AutoSyncScheduler) signalReplan(debt cycleDebt) {
	select {
	case <-s.replanCh:
	default:
	}
	select {
	case s.replanCh <- debt:
	default:
	}
}

// filmBacklogOf rend l'arriéré de films des titres des joueurs d'un cycle (somme des titres, chaque
// titre compté une fois), lu après le post-sync : c'est la fin de la dernière passe des étapes film
// qui l'a publié. Sans lecteur câblé, l'arriéré est vide.
func (s *AutoSyncScheduler) filmBacklogOf(players []domain.PlayerSummary) port.FilmBacklog {
	var total port.FilmBacklog
	if s.filmBacklog == nil {
		return total
	}
	seen := make(map[string]bool, 2)
	for _, p := range players {
		slug := resolveTitleSlug(p)
		if seen[slug] {
			continue
		}
		seen[slug] = true
		b := s.filmBacklog.FilmBacklog(slug)
		total.Frags += b.Frags
		total.Rejeux += b.Rejeux
	}
	return total
}

// FilmsPending dit si le titre a un arriéré de films (port.SyncActivity).
func (s *AutoSyncScheduler) FilmsPending(titleSlug string) bool {
	return s.filmBacklog != nil && s.filmBacklog.FilmBacklog(titleSlug).Pending()
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
		if ok && o.Status != statusFailed {
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

// WithFilmBacklogReader branche la lecture de l'arriéré de films par titre, que la cadence de rattrapage
// et l'écran d'état de l'historique consultent.
func (s *AutoSyncScheduler) WithFilmBacklogReader(r port.FilmBacklogReader) *AutoSyncScheduler {
	s.filmBacklog = r
	return s
}

// WithHistoryReader branche la lecture de l'état de l'historique (clé de fin d'historique de la base
// joueur) dont l'ordonnanceur se sert pour un joueur dont la passe a échoué avant de pouvoir le dire.
func (s *AutoSyncScheduler) WithHistoryReader(r port.HistoryCompletionReader) *AutoSyncScheduler {
	s.historyReader = r
	return s
}
