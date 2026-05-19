package sharedprovider

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/dblease"
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

	lease, err := dblease.AcquireWriterCtx(ctx, nil, p.path, dblease.KindSharedMatches)
	if err != nil {
		swapFailuresTotal.Add(failReasonAcquireWriter, 1)
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
		swapFailuresTotal.Add(failReasonAcquireWriter, 1)
		return nil, fmt.Errorf("sharedprovider: drain inflight readers: %w", err)
	}
	getWaitMsTotal.Add(time.Since(drainStart).Milliseconds())

	rwHandle, err := p.swapToRW(ctx, swapStart)
	if err != nil {
		lease.Release()
		return nil, err
	}

	// WriterHandle construit via closure (refacto commit 8b) — la struct
	// ne référence plus *providerImpl directement, ce qui permet à
	// inMemoryProvider de construire ses propres WriterHandle avec une
	// stratégie de release no-op.
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

			p.releaseWriter(rwHandle)
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
	p.notifyAfterSwapLocked(DirectionPreSwapToRW, StateDraining, StateDraining)

	rwHandle, err := duckdbpkg.OpenReadWrite(p.path, p.timezone)
	if err != nil {
		// Catastrophique : RO fermé, RW refuse. State → Error.
		p.state.Store(int32(StateError))
		recordStateTransition(StateDraining, StateError)
		swapFailuresTotal.Add(failReasonAcquireWriter, 1)
		close(p.ready)
		go p.retryReopenLoop()
		return nil, fmt.Errorf("sharedprovider: open RW after RO close: %w", err)
	}

	p.state.Store(int32(StateRW))
	recordStateTransition(StateDraining, StateRW)
	swapTotal.Add(swapDirRoToRw, 1)
	swapDurationMsTotal.Add(swapDirRoToRw, time.Since(swapStart).Milliseconds())
	return rwHandle, nil
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
func (p *providerImpl) releaseWriter(rwHandle *duckdbpkg.DB) {
	swapStart := time.Now()
	p.mu.Lock()

	prev := State(p.state.Load())
	if prev == StateClosed {
		p.mu.Unlock()
		if rwHandle != nil {
			_ = rwHandle.Close()
		}
		return
	}

	p.state.Store(int32(StateReopening))
	recordStateTransition(prev, StateReopening)

	if rwHandle != nil {
		if err := rwHandle.Close(); err != nil {
			slog.Warn("sharedprovider: close RW failed (continuing)",
				"err", err, "path", p.path)
		}
	}

	if p.tryReopenROLocked() {
		p.state.Store(int32(StateRO))
		recordStateTransition(StateReopening, StateRO)
		swapTotal.Add(swapDirRwToRo, 1)
		swapDurationMsTotal.Add(swapDirRwToRo, time.Since(swapStart).Milliseconds())
		close(p.ready)
		p.mu.Unlock()
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
		p.notifyAfterSwap(DirectionRWToRO, StateRW, StateRO)
		return
	}

	p.state.Store(int32(StateError))
	recordStateTransition(StateReopening, StateError)
	swapFailuresTotal.Add(failReasonReopenRO, 1)
	close(p.ready)
	p.mu.Unlock()
	go p.retryReopenLoop()
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
func (p *providerImpl) retryReopenLoop() {
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
			p.notifyAfterSwap(DirectionErrorToRO, StateError, StateRO)
			slog.Info("sharedprovider: recovered from StateError",
				"path", p.path, "attempt", attempt+1)
			return
		}
		p.mu.Unlock()
	}
	slog.Error("sharedprovider: retry reopen RO definitively failed",
		"path", p.path, "attempts", retryMaxAttempts)
}
