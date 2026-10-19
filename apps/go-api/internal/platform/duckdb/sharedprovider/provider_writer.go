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

	// Début d'acquisition : aucune durée à qualifier, donc DEBUG inconditionnel
	// (cf. logSwapPhase). Le label est reporté sur les logs de fin de phase et
	// sur le WARN de drain timeout — aucune anomalie ne perd son attribution.
	label := ctxkeys.DBWriterLabel(ctx)
	slog.DebugContext(ctx, "provider: AcquireWriter démarré", "path", p.path, "label", label)
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
		// retentera au prochain cycle. Les lecteurs qui ont tenu le drain sont nommés.
		enVol := p.decrireLecteursEnVol()
		p.rollbackFromDraining()
		lease.Release()
		swapFailuresTotal.Add(failReasonDrainTimeout, 1)
		slog.WarnContext(ctx, "provider: drain timeout, rollback vers RO",
			"path", p.path, "label", label,
			"drain_ms", time.Since(drainStart).Milliseconds(), "lecteurs_en_vol", enVol, "err", err)
		if ctx.Err() == nil {
			// La borne PROPRE au drain a expiré, pas le contexte de l'appelant :
			// erreur transitoire typée (sentinelle + cause), retentable par errors.Is.
			return nil, fmt.Errorf("sharedprovider: drain inflight readers: %w: %w", ErrDrainTimeout, err)
		}
		return nil, fmt.Errorf("sharedprovider: drain inflight readers: %w", err)
	}
	drainDur := time.Since(drainStart)
	getWaitMsTotal.Add(drainDur.Milliseconds())
	p.logSwapPhase(ctx, "provider: readers drainés", drainDur,
		"path", p.path, "label", label, "drain_ms", drainDur.Milliseconds())

	rwHandle, err := p.swapToRW(ctx, swapStart)
	if err != nil {
		lease.Release()
		return nil, err
	}
	acquireDur := time.Since(swapStart)
	p.logSwapPhase(ctx, "provider: swap RO→RW terminé", acquireDur,
		"path", p.path, "label", label, "total_ms", acquireDur.Milliseconds())

	// WriterHandle construit via closure (refacto commit 8b) — la struct
	// ne référence plus *providerImpl directement, ce qui permet à
	// inMemoryProvider de construire ses propres WriterHandle avec une
	// stratégie de release no-op.
	//
	// event_id du ctx caller capturé pour les journaux du Release (RW→RO) et de
	// sa reprise éventuelle : tout le cycle du swap reste sous le même event_id.
	capturedEventID := ctxkeys.EventID(ctx)
	return &WriterHandle{
		db: rwHandle.SQLDb(),
		releaseFn: func() {
			defer func() {
				if r := recover(); r != nil {
					swapFailuresTotal.Add(failReasonPanic, 1)
					slog.ErrorContext(ctx, "sharedprovider: panic during Release",
						"panic", r, "path", p.path)
				}
				// Toujours libérer le mutex dblease, même sur panic, sinon
				// plus aucun writer ne pourra acquérir.
				lease.Release()
			}()

			// Reconstruit un ctx avec l'event_id capturé. context.Background()
			// comme parent car le ctx caller peut être annulé au moment du Release.
			releaseCtx := context.Background()
			if capturedEventID != "" {
				releaseCtx = ctxkeys.WithEventID(releaseCtx, capturedEventID)
			}
			p.releaseWriter(releaseCtx, rwHandle)
		},
	}, nil
}

// logSwapPhase journalise une phase du cycle B-swap au niveau que sa durée
// mérite : DEBUG tant qu'elle est nominale (le cycle sain tourne ~40 fois par
// minute en prod et saturait provider.log — cf. defaultSlowSwapThreshold), INFO
// dès qu'elle atteint p.slowSwapThreshold, avec sa durée et threshold_ms pour
// que l'anomalie soit lisible SANS réactiver le niveau debug.
//
// Ne couvre que le nominal : les WARN/ERROR du même cycle (drain timeout, échec
// d'open RW, reopen RO échoué, watchdog de détention) sont émis directement et
// restent inconditionnels.
func (p *providerImpl) logSwapPhase(ctx context.Context, msg string, d time.Duration, attrs ...any) {
	if threshold := p.slowSwapThreshold; threshold > 0 && d >= threshold {
		slog.InfoContext(ctx, msg, append(attrs, "threshold_ms", threshold.Milliseconds())...)
		return
	}
	slog.DebugContext(ctx, msg, attrs...)
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

// swapToRW exécute la transition Draining → RW sous p.mu : close handle RO, ouvre
// le handle RW. Sur échec de l'OpenReadWrite : transition vers StateError +
// retryReopenLoop async (refusOuvertureRWLocked). Phase 3 de AcquireWriter.
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

	rwHandle, err := duckdbpkg.OpenReadWrite(p.path, p.timezone)
	if err != nil {
		return nil, p.refusOuvertureRWLocked(ctx, err)
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

// refusOuvertureRWLocked traite le refus de l'OpenReadWrite du swap (p.mu tenu, handle RO
// déjà fermé) : StateError, réveil des Get en attente, reprise RO asynchrone. Le refus est
// journalisé en ERROR avec le nombre de références RO que le processus tient ENCORE sur le
// fichier : le provider vient de rendre la sienne, donc toute référence restante est tenue
// hors de lui et fera refuser chaque tentative suivante tant qu'elle n'est pas rendue.
func (p *providerImpl) refusOuvertureRWLocked(ctx context.Context, err error) error {
	p.state.Store(int32(StateError))
	recordStateTransition(StateDraining, StateError)
	swapFailuresTotal.Add(failReasonOpenRW, 1)
	close(p.ready)
	slog.ErrorContext(ctx, "sharedprovider: écriture refusée par DuckDB — cause habituelle : un autre handle du processus tient encore le fichier",
		"path", p.path, "label", ctxkeys.DBWriterLabel(ctx),
		"references_lecture_hors_provider", duckdbpkg.CachedReadOnlyRefs(p.path), "err", err)
	// event_id du caller propagé à la reprise asynchrone (traçabilité).
	go p.retryReopenLoop(ctxkeys.EventID(ctx))
	return fmt.Errorf("sharedprovider: open RW after RO close: %w", err)
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
//
// Sur expiration, le drain est ABANDONNÉ explicitement (abandonDrain) : rien ne
// survit à cet appel qui pourrait réveiller ou faire paniquer le drain suivant
// — c'était la cause des deux crashs du 2026-09-16 (cf. reader_drain.go).
func (p *providerImpl) waitForDrain(parentCtx context.Context) error {
	ctx, cancel := context.WithTimeout(parentCtx, p.drainTimeout)
	defer cancel()

	done := p.beginDrain()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		p.abandonDrain(done)
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
// ctx porte l'event_id capturé au moment d'AcquireWriter : le cycle de vie
// complet d'un swap RW se trace dans logs/{sync,provider,duckdb}.log.
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
		releaseDur := time.Since(swapStart)
		p.logSwapPhase(ctx, "provider: swap RW→RO terminé", releaseDur,
			"path", p.path, "label", holderLabel,
			"total_ms", releaseDur.Milliseconds())
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
// capturedEventID (event_id du caller initial du swap) est restauré dans un ctx :
// la reprise se trace sous le même event_id que le swap échoué qui l'a déclenchée.
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
			slog.InfoContext(ctx, "sharedprovider: recovered from StateError",
				"path", p.path, "attempt", attempt+1)
			return
		}
		p.mu.Unlock()
	}
	slog.ErrorContext(ctx, "sharedprovider: retry reopen RO definitively failed",
		"path", p.path, "attempts", retryMaxAttempts)
}
