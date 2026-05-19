package sharedprovider

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/dblease"
)

// defaultReadyTimeout est le délai max d'attente d'un Get pendant un swap RW.
const defaultReadyTimeout = 30 * time.Second

// defaultRetryBaseBackoff est le délai initial du retry exponentiel quand la
// réouverture RO post-sync échoue. Backoff : 1s, 2s, 4s, 8s, 16s = 31s max
// sur 5 tentatives (aligné avec le retry metadata main.go:222-236).
const defaultRetryBaseBackoff = time.Second

// retryMaxAttempts borne le retry loop avant abandon définitif.
const retryMaxAttempts = 5

// defaultDrainTimeout est le délai max d'attente du drain des readers en vol
// avant un swap RW. Dépassé → rollback (le sync engine retentera).
//
// En pratique les Get sont brefs (queries DuckDB <100ms), drain quasi-immédiat.
// Le timeout sert de garde-fou si un consumer fait un Get + transaction longue.
const defaultDrainTimeout = 5 * time.Second

// defaultCloseDrainTimeout est le délai max d'attente du drain à Close.
// Plus court que defaultDrainTimeout — au shutdown on accepte de forcer la
// fermeture du handle plutôt que de bloquer indéfiniment.
const defaultCloseDrainTimeout = 3 * time.Second

// Provider est le contrat exposé aux consommateurs. Une implémentation owne
// le handle DuckDB pour un chemin donné et arbitre l'accès lecture/écriture.
//
// Une instance Provider est sûre pour usage concurrent : Get peut être
// appelé depuis N goroutines en parallèle. AcquireWriter est sérialisé via
// dblease.AcquireWriterCtx.
type Provider interface {
	// Get retourne un *sql.DB en mode RO + une fonction release qui DOIT
	// être appelée (typiquement via defer) une fois les lectures terminées.
	// Tant que release n'est pas appelé, le provider considère qu'un reader
	// est en vol — un AcquireWriter concurrent attendra (drain) avant de
	// fermer le handle RO.
	//
	// Si err != nil, release est nil — pas besoin de le defer.
	//
	// Erreurs possibles :
	//   - ErrProviderClosed : le provider est fermé
	//   - ErrSwapFailed     : reopen RO post-sync a échoué (StateError)
	//   - ErrSwapTimeout    : attente swap > readyTimeout
	//   - ctx.Err()         : contexte annulé/expiré
	//
	// Usage idiomatique :
	//
	//	db, release, err := provider.Get(ctx)
	//	if err != nil {
	//	    return err
	//	}
	//	defer release()
	//	var n int
	//	return db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ...").Scan(&n)
	Get(ctx context.Context) (*sql.DB, func(), error)

	// AcquireWriter prend un lease writer exclusif (via dblease), drain les
	// readers en vol, puis bascule le handle global en mode RW.
	//
	// Le WriterHandle retourné DOIT être Released (typiquement via defer).
	AcquireWriter(ctx context.Context) (*WriterHandle, error)

	// State retourne l'état courant. Lecture atomique, sans verrou.
	State() State

	// Path retourne le chemin absolu de la DB ciblée.
	Path() string

	// Close ferme le handle DuckDB sous-jacent et marque le provider comme
	// fermé. Best-effort drain des readers en vol (timeout court 3s) avant
	// fermeture forcée. Idempotent.
	Close() error

	// Subscribe enregistre un callback invoqué après chaque transition
	// d'état observable (cf. Direction). Le callback est exécuté
	// synchroniquement sans tenir le mutex interne — il est safe d'appeler
	// Get/AcquireWriter depuis le callback, mais PAS Subscribe/Unsubscribe.
	//
	// Retourne une fonction unsubscribe à appeler quand le callback n'est
	// plus nécessaire (typiquement via defer). Idempotente.
	//
	// Cas d'usage principal : le pool joueur réagit à DirectionRWToRO pour
	// purger ses conns idle qui auraient un ATTACH RO stale sur shared.
	Subscribe(fn Subscriber) (unsubscribe func())
}

// providerImpl est l'implémentation par défaut.
type providerImpl struct {
	path     string
	timezone string

	state atomic.Int32

	// mu protège handle, ready, et les transitions d'état.
	mu     sync.Mutex
	handle *duckdbpkg.DB
	ready  chan struct{}

	// readersWG track les Get en vol. Add(1) sous p.mu quand state=RO,
	// Done() dans le release retourné. AcquireWriter Wait() avant le close
	// du handle pour éviter "database is closed" côté caller.
	readersWG sync.WaitGroup

	readyTimeout     time.Duration
	retryBaseBackoff time.Duration
	drainTimeout     time.Duration

	failNextReopen atomic.Bool

	// subsMu protège les subscribers Subscribe/Unsubscribe.
	// Distinct de p.mu pour ne pas bloquer un Subscribe pendant un swap.
	subsMu    sync.Mutex
	subs      map[int64]Subscriber
	nextSubID int64
}

// New ouvre une nouvelle instance Provider sur path en mode read-only.
func New(path string, timezone ...string) (Provider, error) {
	tz := ""
	if len(timezone) > 0 {
		tz = timezone[0]
	}

	handle, err := duckdbpkg.OpenReadOnly(path, tz)
	if err != nil {
		return nil, fmt.Errorf("sharedprovider: open RO %s: %w", path, err)
	}

	p := &providerImpl{
		path:             path,
		timezone:         tz,
		handle:           handle,
		ready:            newClosedChan(),
		readyTimeout:     defaultReadyTimeout,
		retryBaseBackoff: defaultRetryBaseBackoff,
		drainTimeout:     defaultDrainTimeout,
	}
	p.state.Store(int32(StateRO))
	recordStateTransition(StateRO, StateRO)
	return p, nil
}

func newClosedChan() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// Get implémente Provider.Get.
func (p *providerImpl) Get(ctx context.Context) (*sql.DB, func(), error) {
	deadline := time.Now().Add(p.readyTimeout)
	for {
		p.mu.Lock()
		s := State(p.state.Load())
		switch s {
		case StateClosed:
			p.mu.Unlock()
			return nil, nil, ErrProviderClosed
		case StateError:
			p.mu.Unlock()
			return nil, nil, ErrSwapFailed
		case StateRO:
			handle := p.handle
			if handle == nil {
				p.mu.Unlock()
				return nil, nil, ErrSwapFailed
			}
			// Track le reader AVANT de relâcher mu — sinon race possible
			// avec un swap qui drain trop tôt (sans nous attendre).
			p.readersWG.Add(1)
			readersInUse.Add(1)
			p.mu.Unlock()

			db := handle.SQLDb()
			release := sync.OnceFunc(func() {
				p.readersWG.Done()
				readersInUse.Add(-1)
			})
			return db, release, nil
		}

		// state ∈ {Draining, RW, Reopening} → attendre.
		ready := p.ready
		p.mu.Unlock()

		remaining := time.Until(deadline)
		if remaining <= 0 {
			getTimeoutTotal.Add(1)
			return nil, nil, ErrSwapTimeout
		}
		timer := time.NewTimer(remaining)
		select {
		case <-ready:
			timer.Stop()
		case <-ctx.Done():
			timer.Stop()
			return nil, nil, ctx.Err()
		case <-timer.C:
			getTimeoutTotal.Add(1)
			return nil, nil, ErrSwapTimeout
		}
	}
}

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

	// PHASE 1 : transition vers Draining (gate nouveaux Get).
	p.mu.Lock()
	prev := State(p.state.Load())
	if prev == StateClosed {
		p.mu.Unlock()
		lease.Release()
		return nil, ErrProviderClosed
	}
	p.state.Store(int32(StateDraining))
	recordStateTransition(prev, StateDraining)
	p.ready = make(chan struct{})
	p.mu.Unlock()

	// PHASE 2 : drain inflight readers (sans tenir mu pour ne pas deadlock
	// les release() qui n'ont pas besoin de mu mais bloqueraient si pris).
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

	// PHASE 3 : swap RO → RW sous mu (drain confirmé, plus de readers actifs).
	p.mu.Lock()
	if State(p.state.Load()) == StateClosed {
		// Race rare avec un Close concurrent. Bail.
		p.mu.Unlock()
		lease.Release()
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
	// Notification SYNCHRONE sous p.mu — les Subscribers NE DOIVENT PAS
	// appeler Get/AcquireWriter pendant ce callback (deadlock garanti).
	// Le doc de DirectionPreSwapToRW le précise.
	p.notifyAfterSwap(DirectionPreSwapToRW, StateDraining, StateDraining)

	rwHandle, err := duckdbpkg.OpenReadWrite(p.path, p.timezone)
	if err != nil {
		// Catastrophique : RO fermé, RW refuse. State → Error.
		p.state.Store(int32(StateError))
		recordStateTransition(StateDraining, StateError)
		swapFailuresTotal.Add(failReasonAcquireWriter, 1)
		close(p.ready)
		p.mu.Unlock()
		lease.Release()
		go p.retryReopenLoop()
		return nil, fmt.Errorf("sharedprovider: open RW after RO close: %w", err)
	}

	p.state.Store(int32(StateRW))
	recordStateTransition(StateDraining, StateRW)
	swapTotal.Add(swapDirRoToRw, 1)
	swapDurationMsTotal.Add(swapDirRoToRw, time.Since(swapStart).Milliseconds())
	p.mu.Unlock()

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

// Subscribe enregistre un callback notifié après chaque transition vers RO.
// Retourne une fonction unsubscribe idempotente (sync.OnceFunc).
//
// Le callback est invoqué synchroniquement par la goroutine qui termine le
// swap, après que p.mu soit relâché — il est donc safe d'appeler
// Get/AcquireWriter depuis le callback (mais PAS Subscribe/Unsubscribe).
func (p *providerImpl) Subscribe(fn Subscriber) func() {
	p.subsMu.Lock()
	defer p.subsMu.Unlock()
	if p.subs == nil {
		p.subs = make(map[int64]Subscriber)
	}
	id := p.nextSubID
	p.nextSubID++
	p.subs[id] = fn

	return sync.OnceFunc(func() {
		p.subsMu.Lock()
		defer p.subsMu.Unlock()
		delete(p.subs, id)
	})
}

// notifyAfterSwap émet un SwapEvent à tous les Subscribers actifs.
//
// IMPORTANT : doit être appelé SANS tenir p.mu, sinon un Subscriber qui
// appelle Get/AcquireWriter (qui prennent p.mu) deadlocke.
// On capture le snapshot de subs sous subsMu pour éviter une race avec
// un Unsubscribe concurrent.
func (p *providerImpl) notifyAfterSwap(direction Direction, from, to State) {
	p.subsMu.Lock()
	if len(p.subs) == 0 {
		p.subsMu.Unlock()
		return
	}
	subs := make([]Subscriber, 0, len(p.subs))
	for _, fn := range p.subs {
		subs = append(subs, fn)
	}
	p.subsMu.Unlock()

	evt := SwapEvent{Direction: direction, From: from, To: to, Path: p.path}
	for _, fn := range subs {
		// Pas de recover ici : un Subscriber qui panic indique un bug applicatif
		// — laisser remonter pour ne pas masquer le problème. Le swap est
		// déjà terminé côté Provider, le state est cohérent.
		fn(evt)
	}
}

// State implémente Provider.State.
func (p *providerImpl) State() State {
	return State(p.state.Load())
}

// Path implémente Provider.Path.
func (p *providerImpl) Path() string {
	return p.path
}

// Close implémente Provider.Close. Best-effort drain (3s timeout) puis force
// la fermeture du handle. Idempotent.
func (p *providerImpl) Close() error {
	p.mu.Lock()
	prev := State(p.state.Load())
	if prev == StateClosed {
		p.mu.Unlock()
		return nil
	}
	p.state.Store(int32(StateClosed))
	recordStateTransition(prev, StateClosed)
	select {
	case <-p.ready:
	default:
		close(p.ready)
	}
	p.mu.Unlock()

	// Best-effort drain — ne pas bloquer le shutdown sur des readers stuck.
	drainDone := make(chan struct{})
	go func() {
		p.readersWG.Wait()
		close(drainDone)
	}()
	select {
	case <-drainDone:
	case <-time.After(defaultCloseDrainTimeout):
		slog.Warn("sharedprovider: Close drain timeout, forcing handle close",
			"path", p.path)
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.handle != nil {
		err := p.handle.Close()
		p.handle = nil
		if err != nil {
			return fmt.Errorf("sharedprovider: close %s: %w", p.path, err)
		}
	}
	return nil
}
