package sharedprovider

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/dblease"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

// AcquireWriter implémente Provider.AcquireWriter avec drain en 3 phases :
//   - Phase 1 : transition RO → Draining sous mu (gate les nouveaux Get)
//   - Phase 2 : drain des readers en vol (hors mu, attente bornée par drainTimeout)
//   - Phase 3 : swap RO → RW sous mu (close handle RO, open RW, set StateRW)
//
// Si phase 2 expire : rollback vers RO, lease release, erreur. Le sync
// engine peut retenter au cycle suivant.
func (p *providerImpl) AcquireWriter(ctx context.Context) (*WriterHandle, error) {
	if State(p.state.Load()) == StateClosed {
		return nil, ErrProviderClosed
	}

	slog.InfoContext(ctx, "provider: AcquireWriter démarré",
		"path", p.path, "label", ctxkeys.DBWriterLabel(ctx))
	lease, err := dblease.AcquireWriterCtx(ctx, nil, p.path, dblease.KindSharedMatches)
	if err != nil {
		swapFailuresTotal.Add(failReasonAcquireWriter, 1)
		slog.WarnContext(ctx, "provider: dblease échoué", "path", p.path, "err", err)
		return nil, err
	}

	swapStart := time.Now()

	if err := p.gateToDraining(); err != nil {
		lease.Release()
		return nil, err
	}

	drainStart := time.Now()
	if err := p.waitForDrain(ctx); err != nil {
		// Drain expiré : rollback vers RO. Les readers en vol terminent OK
		// (handle pas fermé), les nouveaux Get reprennent. Le sync engine
		// retentera au prochain cycle.
		p.rollbackFromDraining()
		lease.Release()
		swapFailuresTotal.Add(failReasonDrainTimeout, 1)
		slog.WarnContext(ctx, "provider: drain timeout, rollback vers RO",
			"path", p.path, "drain_ms", time.Since(drainStart).Milliseconds(), "err", err)
		return nil, fmt.Errorf("sharedprovider: drain inflight readers: %w", err)
	}
	drainMs := time.Since(drainStart).Milliseconds()
	getWaitMsTotal.Add(drainMs)
	slog.InfoContext(ctx, "provider: readers drainés", "path", p.path, "drain_ms", drainMs)

	rwHandle, err := p.swapToRW(ctx, swapStart)
	if err != nil {
		lease.Release()
		return nil, err
	}
	slog.InfoContext(ctx, "provider: swap RO→RW terminé",
		"path", p.path, "total_ms", time.Since(swapStart).Milliseconds())

	// WriterHandle construit via closure (refacto commit 8b) — la struct
	// ne référence plus *providerImpl directement, ce qui permet à
	// inMemoryProvider de construire ses propres WriterHandle avec une
	// stratégie de release no-op.
	//
	// Sprint B1 commit 19 : capture event_id du ctx caller pour propagation
	// au callback Subscriber lors du Release (RW→RO). Permet de tracer le
	// timeline complet d'un swap : caller (sync.RunDelta) → Provider AcquireWriter
	// → notify PreSwap (pool DETACH) → write → Release → notify RWToRO
	// (pool re-attach) — tout sous le même event_id.
	capturedEventID := ctxkeys.EventID(ctx)
	return &WriterHandle{
		db: rwHandle.SQLDb(),
		releaseFn: func() {
			defer func() {
				if r := recover(); r != nil {
					swapFailuresTotal.Add(failReasonPanic, 1)
					slog.Error("sharedprovider: panic during Release",
						"panic", r, "path", p.path)
				}
				// Toujours libérer le mutex dblease, même sur panic, sinon
				// plus aucun writer ne pourra acquérir.
				lease.Release()
			}()

			// Reconstruit un ctx avec l'event_id capturé pour propagation aux
			// Subscribers via notifyAfterSwap. context.Background() comme parent
			// car le ctx caller peut être annulé au moment du Release.
			releaseCtx := context.Background()
			if capturedEventID != "" {
				releaseCtx = ctxkeys.WithEventID(releaseCtx, capturedEventID)
			}
			p.releaseWriter(releaseCtx, rwHandle)
		},
	}, nil
}

// gateToDraining transitionne l'état RO → Draining sous p.mu pour gater
// les nouveaux Get(). Retourne ErrProviderClosed si une race avec Close()
// a été détectée. Phase 1 de AcquireWriter.
func (p *providerImpl) gateToDraining() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	prev := State(p.state.Load())
	if prev == StateClosed {
		return ErrProviderClosed
	}
	p.state.Store(int32(StateDraining))
	recordStateTransition(prev, StateDraining)
	// Début de la fenêtre de blocage des lecteurs (mesurée à la réouverture RO).
	p.swapBlockStart = time.Now()
	p.ready = make(chan struct{})
	return nil
}

// swapToRW exécute la transition Draining → RW sous p.mu : close handle RO,
// notifie les Subscribers (sous mu — voir notifyAfterSwapLocked), ouvre le
// handle RW. Sur échec de l'OpenReadWrite : transition vers StateError +
// retryReopenLoop async. Phase 3 de AcquireWriter.
func (p *providerImpl) swapToRW(ctx context.Context, swapStart time.Time) (*duckdbpkg.DB, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if State(p.state.Load()) == StateClosed {
		// Race rare avec un Close concurrent. Bail.
		return nil, ErrProviderClosed
	}

	if p.handle != nil {
		if err := p.handle.Close(); err != nil {
			slog.WarnContext(ctx, "sharedprovider: close RO failed (continuing)",
				"err", err, "path", p.path)
		}
		p.handle = nil
	}

	// PHASE 3.5 (commit 8e, repositionné commit 8f) : notif PreSwapToRW
	// SYNCHRONE entre la fermeture du handle Provider et l'OpenReadWrite.
	//
	// Pourquoi ce timing précis ?
	//   - DuckDB-Go fait de l'auto-attach : si shared est ouvert quelque part
	//     dans le process, toute nouvelle conn DuckDB l'auto-attache. Si
	//     les Subscribers (pool) faisaient Reopen pendant que Provider.handle
	//     est encore ouvert, la nouvelle conn player auto-attacherait shared
	//     et bloquerait l'OpenReadWrite suivant ("Unique file handle conflict").
	//   - En émettant la notif APRÈS Close handle et AVANT OpenReadWrite,
	//     le fichier shared est totalement libéré côté Provider. Les
	//     Subscribers peuvent fermer leurs propres handles RO et Reopen leurs
	//     conns player sans auto-attach.
	//
	// Notification SYNCHRONE sous p.mu — d'où l'appel à notifyAfterSwapLocked
	// (variante explicite qui documente ce contract). Les Subscribers NE
	// DOIVENT PAS appeler Get/AcquireWriter pendant ce callback (deadlock garanti).
	p.notifyAfterSwapLocked(ctx, DirectionPreSwapToRW, StateDraining, StateDraining)

	rwHandle, err := duckdbpkg.OpenReadWrite(p.path, p.timezone)
	if err != nil {
		// Catastrophique : RO fermé, RW refuse. State → Error.
		p.state.Store(int32(StateError))
		recordStateTransition(StateDraining, StateError)
		swapFailuresTotal.Add(failReasonAcquireWriter, 1)
		close(p.ready)
		// Sprint B1 commit 19 : propage event_id du caller dans la goroutine
		// de retry async pour traçabilité de la recovery.
		go p.retryReopenLoop(ctxkeys.EventID(ctx))
		return nil, fmt.Errorf("sharedprovider: open RW after RO close: %w", err)
	}

	p.state.Store(int32(StateRW))
	recordStateTransition(StateDraining, StateRW)
	// Phase 0 — début de la fenêtre RW stricte (sous p.mu), lue au releaseWriter.
	p.rwWindowStart = time.Now()
	// Étape 0 attribution : capture du label de détenteur (posé par le caller via
	// ctxkeys.WithDBWriterLabel à l'entrée de son sous-système) + armement du
	// watchdog. Le callback du timer ne prend AUCUN lock (valeurs capturées) et
	// fire UNE fois par acquisition — il attrape aussi un writer jamais release,
	// qu'un check passif au release raterait.
	p.rwHolderLabel = ctxkeys.DBWriterLabel(ctx)
	p.armRWWatchdogLocked()
	swapTotal.Add(swapDirRoToRw, 1)
	swapDurationMsTotal.Add(swapDirRoToRw, time.Since(swapStart).Milliseconds())
	return rwHandle, nil
}

// armRWWatchdogLocked arme le timer de surveillance de détention du writer.
// Doit être appelé avec p.mu tenu, juste après la pose de rwWindowStart et
// rwHolderLabel. No-op si le seuil est <= 0 (désactivation explicite).
func (p *providerImpl) armRWWatchdogLocked() {
	if p.rwHoldWatchdog <= 0 {
		return
	}
	label, path, threshold, start := p.rwHolderLabel, p.path, p.rwHoldWatchdog, p.rwWindowStart
	p.rwWatchdog = time.AfterFunc(threshold, func() {
		watchdogFiredTotal.Add(1)
		recordHolderWatchdog(label)
		slog.Warn("sharedprovider: writer RW tenu au-delà du seuil — lectures gatées pendant ce temps",
			"label", label, "path", path,
			"threshold_ms", threshold.Milliseconds(),
			"held_ms", time.Since(start).Milliseconds())
	})
}

// disarmRWWatchdogLocked stoppe le watchdog et efface le label du détenteur.
// Doit être appelé avec p.mu tenu (releaseWriter, y compris le chemin Close).
// Retourne le label qui était actif (pour l'attribution/les logs du release).
func (p *providerImpl) disarmRWWatchdogLocked() string {
	label := p.rwHolderLabel
	if p.rwWatchdog != nil {
		p.rwWatchdog.Stop()
		p.rwWatchdog = nil
	}
	p.rwHolderLabel = ""
	return label
}

// waitForDrain attend que tous les readers en vol fassent leur release().
// Borné par p.drainTimeout ou ctx.Done(), la première échéance gagne.
func (p *providerImpl) waitForDrain(parentCtx context.Context) error {
	ctx, cancel := context.WithTimeout(parentCtx, p.drainTimeout)
	defer cancel()

	done := make(chan struct{})
	go func() {
		p.readersWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// rollbackFromDraining ré-établit StateRO après un drain expiré. Le handle
// RO est resté ouvert (on n'a fermé que dans la phase 3), les readers en
// vol continuent leurs queries. Les Get bloqués sur ready sont débloqués.
func (p *providerImpl) rollbackFromDraining() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if State(p.state.Load()) != StateDraining {
		// Quelqu'un d'autre a déjà transitionné (Close ?). Bail.
		return
	}
	p.state.Store(int32(StateRO))
	recordStateTransition(StateDraining, StateRO)
	select {
	case <-p.ready:
	default:
		close(p.ready)
	}
}

// releaseWriter est appelé par WriterHandle.Release.
//
// Sprint B1 commit 19 : ctx contient l'event_id capturé au moment d'AcquireWriter
// pour propagation aux Subscribers via notifyAfterSwap. Permet de tracer le
// cycle de vie complet d'un swap RW dans logs/{sync,provider,duckdb}.log.
func (p *providerImpl) releaseWriter(ctx context.Context, rwHandle *duckdbpkg.DB) {
	swapStart := time.Now()
	p.mu.Lock()

	prev := State(p.state.Load())
	if prev == StateClosed {
		// Désarmer le watchdog même sur ce chemin — sinon il firerait un WARN
		// fantôme après la fermeture du provider.
		p.disarmRWWatchdogLocked()
		p.mu.Unlock()
		if rwHandle != nil {
			_ = rwHandle.Close()
		}
		return
	}

	p.state.Store(int32(StateReopening))
	recordStateTransition(prev, StateReopening)
	// Phase 0 — fenêtre RW stricte : durée pendant laquelle le handle était RW
	// (Get gatés), enregistrée à la sortie de l'état RW quel que soit le reopen.
	// Étape 0 attribution : la même fenêtre est ventilée PAR DÉTENTEUR (label
	// capturé au swapToRW) et le watchdog est désarmé.
	holderLabel := p.disarmRWWatchdogLocked()
	if !p.rwWindowStart.IsZero() {
		heldMs := time.Since(p.rwWindowStart).Milliseconds()
		observability.RecordDurationMS("shared_provider_rw_window_ms", heldMs)
		recordHolderWindow(holderLabel, heldMs)
		p.rwWindowStart = time.Time{}
	}

	if rwHandle != nil {
		if err := rwHandle.Close(); err != nil {
			slog.WarnContext(ctx, "sharedprovider: close RW failed (continuing)",
				"err", err, "path", p.path)
		}
	}

	if p.tryReopenROLocked() {
		p.state.Store(int32(StateRO))
		recordStateTransition(StateReopening, StateRO)
		swapTotal.Add(swapDirRwToRo, 1)
		swapDurationMsTotal.Add(swapDirRwToRo, time.Since(swapStart).Milliseconds())
		// Fenêtre totale de blocage des lecteurs (drain + maintien RW + reopen) :
		// le nombre le plus représentatif du « stall » ressenti par swap.
		if !p.swapBlockStart.IsZero() {
			observability.RecordDurationMS("shared_provider_blocked_window_ms",
				time.Since(p.swapBlockStart).Milliseconds())
		}
		close(p.ready)
		p.mu.Unlock()
		slog.InfoContext(ctx, "provider: swap RW→RO terminé",
			"path", p.path, "label", holderLabel,
			"total_ms", time.Since(swapStart).Milliseconds())
		// IMPORTANT : notify HORS du lock — les Subscribers peuvent appeler
		// Get/AcquireWriter (qui prennent p.mu) sans deadlock.
		//
		// Race documentée (constatée commit 12a) : entre l'unlock ci-dessus
		// et l'exécution du callback Subscriber, un Get() concurrent peut
		// succeed (state == StateRO). Le Subscriber est alors en train de
		// reopen son propre handle RO en parallèle. Aucun conflit : DuckDB-Go
		// tolère N ouvertures RO simultanées sur le même fichier (mode RO
		// uniforme, pas de file lock exclusif). Cette tolérance est testée
		// par TestProvider_HTTPReadersWaitDuringSync_integration.
		// Si une future version de DuckDB durcissait ce contrat, il faudrait
		// notifier sous mu (cf. notifyAfterSwapLocked) — moyennant un
		// refactor du callback pool (pool_swap_hook.go).
		p.notifyAfterSwap(ctx, DirectionRWToRO, StateRW, StateRO)
		return
	}

	p.state.Store(int32(StateError))
	recordStateTransition(StateReopening, StateError)
	swapFailuresTotal.Add(failReasonReopenRO, 1)
	close(p.ready)
	p.mu.Unlock()
	// Sprint B1 commit 19 : propage l'event_id via ctxkeys.WithEventID dans
	// la goroutine async — context.Background() pour découpler du ctx caller
	// (qui peut être annulé), mais préserve l'event_id pour traçabilité.
	go p.retryReopenLoop(ctxkeys.EventID(ctx))
}

// tryReopenROLocked tente d'ouvrir une nouvelle conn RO. Doit être appelée
// avec p.mu tenu. Hook test failNextReopen consommé via CompareAndSwap.
func (p *providerImpl) tryReopenROLocked() bool {
	if p.failNextReopen.CompareAndSwap(true, false) {
		slog.Warn("sharedprovider: tryReopenRO failed by test hook", "path", p.path)
		return false
	}
	handle, err := duckdbpkg.OpenReadOnly(p.path, p.timezone)
	if err != nil {
		slog.Error("sharedprovider: reopen RO failed", "err", err, "path", p.path)
		return false
	}
	p.handle = handle
	return true
}

// retryReopenLoop tente de récupérer après un échec de reopen RO.
//
// Sprint B1 commit 19 : capturedEventID (event_id du caller initial du swap)
// est restauré dans un ctx pour propagation aux Subscribers via notifyAfterSwap.
// Permet de tracer la recovery DirectionErrorToRO sous le même event_id que
// le swap échoué qui l'a déclenchée.
func (p *providerImpl) retryReopenLoop(capturedEventID string) {
	ctx := context.Background()
	if capturedEventID != "" {
		ctx = ctxkeys.WithEventID(ctx, capturedEventID)
	}
	backoff := p.retryBaseBackoff
	for attempt := 0; attempt < retryMaxAttempts; attempt++ {
		time.Sleep(backoff)
		backoff *= 2

		if State(p.state.Load()) == StateClosed {
			return
		}

		p.mu.Lock()
		if State(p.state.Load()) != StateError {
			p.mu.Unlock()
			return
		}
		if p.tryReopenROLocked() {
			p.state.Store(int32(StateRO))
			recordStateTransition(StateError, StateRO)
			p.mu.Unlock()
			p.notifyAfterSwap(ctx, DirectionErrorToRO, StateError, StateRO)
			slog.InfoContext(ctx, "sharedprovider: recovered from StateError",
				"path", p.path, "attempt", attempt+1)
			return
		}
		p.mu.Unlock()
	}
	slog.Error("sharedprovider: retry reopen RO definitively failed",
		"path", p.path, "attempts", retryMaxAttempts)
}
