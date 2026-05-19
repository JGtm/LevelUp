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
// Au-delà, Get retourne ErrSwapTimeout et le handler HTTP doit mapper en 503
// Retry-After (cf. ADR-0014 à venir).
const defaultReadyTimeout = 30 * time.Second

// defaultRetryBaseBackoff est le délai initial du retry exponentiel quand la
// réouverture RO post-sync échoue. Backoff : 1s, 2s, 4s, 8s, 16s = 31s max
// sur 5 tentatives (aligné avec le retry metadata main.go:222-236).
const defaultRetryBaseBackoff = time.Second

// retryMaxAttempts borne le retry loop avant abandon définitif.
const retryMaxAttempts = 5

// Provider est le contrat exposé aux consommateurs. Une implémentation owne
// le handle DuckDB pour un chemin donné et arbitre l'accès lecture/écriture.
//
// Une instance Provider est sûre pour usage concurrent : Get peut être
// appelé depuis N goroutines en parallèle. AcquireWriter est sérialisé via
// dblease.AcquireWriterCtx.
type Provider interface {
	// Get retourne un *sql.DB en mode RO, prêt à lire. En steady state
	// (StateRO), retour immédiat. Pendant un swap RW, bloque jusqu'au
	// retour en RO, jusqu'à expiration du contexte, ou jusqu'à readyTimeout.
	//
	// Erreurs possibles :
	//   - ErrProviderClosed : le provider est fermé (shutdown)
	//   - ErrSwapFailed     : reopen RO post-sync a échoué (StateError)
	//   - ErrSwapTimeout    : l'attente du retour en RO a dépassé readyTimeout
	//   - ctx.Err()         : contexte annulé/expiré
	//
	// Le *sql.DB retourné a la durée de vie du Provider — ne pas le fermer.
	Get(ctx context.Context) (*sql.DB, error)

	// AcquireWriter prend un lease writer exclusif (via dblease) et bascule
	// le handle global en mode RW. Le sync engine appelle cette méthode au
	// début d'un RunDelta/RunBackfill, puis Release() à la fin.
	//
	// Erreurs possibles :
	//   - ErrProviderClosed : le provider est fermé
	//   - dblease.ErrDBLocked : un autre writer tient déjà le lease (ou ctx
	//     annulé avant acquisition)
	//   - erreur d'ouverture RW (catastrophique — le provider passe en
	//     StateError, un retry loop tente la récupération)
	//
	// Le WriterHandle retourné DOIT être Released (typiquement via defer).
	AcquireWriter(ctx context.Context) (*WriterHandle, error)

	// State retourne l'état courant. Lecture atomique, sans verrou.
	State() State

	// Path retourne le chemin absolu de la DB ciblée.
	Path() string

	// Close ferme le handle DuckDB sous-jacent et marque le provider comme
	// fermé. Les Get/AcquireWriter ultérieurs retournent ErrProviderClosed.
	// Idempotent.
	Close() error
}

// providerImpl est l'implémentation par défaut.
type providerImpl struct {
	path     string
	timezone string

	// state encode l'état courant en int32 pour load/store atomiques.
	state atomic.Int32

	// mu protège handle, ready, et les transitions d'état.
	mu     sync.Mutex
	handle *duckdbpkg.DB
	// ready est :
	//   - fermé quand state ∈ {RO, Error, Closed} : les Get passent immédiatement
	//   - non-fermé quand state ∈ {Draining, RW, Reopening} : les Get bloquent
	// Remplacé par un nouveau canal à chaque entrée en swap (AcquireWriter).
	ready chan struct{}

	// readyTimeout configure le délai max d'attente d'un Get pendant un swap.
	readyTimeout time.Duration
	// retryBaseBackoff configure le délai initial du retry loop reopen RO.
	retryBaseBackoff time.Duration

	// failNextReopen — hook test-only : si true, le prochain appel à
	// tryReopenROLocked retourne false sans tenter l'OpenReadOnly. Le flag
	// est consommé (CompareAndSwap) au premier essai pour simuler une
	// défaillance ponctuelle suivie d'une récupération via retry loop.
	failNextReopen atomic.Bool
}

// New ouvre une nouvelle instance Provider sur path en mode read-only.
//
// timezone (optionnel) : nom IANA passé à duckdb.OpenReadOnly.
//
// Eager open : retourne une erreur si le fichier est inaccessible.
// L'appelant DOIT appeler Close() (typiquement via defer en main.go).
//
// Préférer Manager.For() en prod pour bénéficier du caching par chemin.
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
	}
	p.state.Store(int32(StateRO))
	recordStateTransition(StateRO, StateRO)
	return p, nil
}

// newClosedChan crée un canal déjà fermé. Utilisé comme valeur initiale de
// `ready` quand le provider est en steady state RO : les Get y lisent sans
// blocage.
func newClosedChan() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// Get implémente Provider.Get.
func (p *providerImpl) Get(ctx context.Context) (*sql.DB, error) {
	deadline := time.Now().Add(p.readyTimeout)
	for {
		p.mu.Lock()
		s := State(p.state.Load())
		ready := p.ready
		handle := p.handle
		p.mu.Unlock()

		switch s {
		case StateClosed:
			return nil, ErrProviderClosed
		case StateError:
			return nil, ErrSwapFailed
		case StateRO:
			if handle == nil {
				// Edge case improbable : state RO mais handle nil. Traiter
				// comme une erreur swap.
				return nil, ErrSwapFailed
			}
			return handle.SQLDb(), nil
		}

		// state ∈ {Draining, RW, Reopening} → attendre.
		remaining := time.Until(deadline)
		if remaining <= 0 {
			getTimeoutTotal.Add(1)
			return nil, ErrSwapTimeout
		}
		timer := time.NewTimer(remaining)
		select {
		case <-ready:
			timer.Stop()
			// loop : recheck state (peut être RO, Error, ou Closed maintenant)
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
			getTimeoutTotal.Add(1)
			return nil, ErrSwapTimeout
		}
	}
}

// AcquireWriter implémente Provider.AcquireWriter.
//
// Séquence :
//  1. Acquérir le mutex dblease (bloque si un autre writer tient le lease).
//  2. Transition RO → Draining → RW :
//     - state := Draining
//     - replace ready par un nouveau canal non-fermé (gate les nouveaux Get)
//     - close conn RO (libère le file lock DuckDB)
//     - open conn RW
//     - state := RW
//  3. Retourner un WriterHandle. Release() inverse l'opération.
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
	p.mu.Lock()

	prev := State(p.state.Load())
	if prev == StateClosed {
		p.mu.Unlock()
		lease.Release()
		return nil, ErrProviderClosed
	}

	// Transition vers Draining (les nouveaux Get vont bloquer).
	p.state.Store(int32(StateDraining))
	recordStateTransition(prev, StateDraining)
	p.ready = make(chan struct{})

	// Fermer le handle RO pour libérer le file lock DuckDB côté driver.
	if p.handle != nil {
		if err := p.handle.Close(); err != nil {
			slog.WarnContext(ctx, "sharedprovider: close RO failed (continuing)",
				"err", err, "path", p.path)
		}
		p.handle = nil
	}

	// Ouvrir en RW. C'est ici que peut survenir l'erreur DuckDB classique
	// si un autre composant tient encore une conn RO sur le même fichier
	// — exactement ce que le commit 6/7 va éliminer en faisant passer
	// main.go et le pool joueur par ce provider.
	rwHandle, err := duckdbpkg.OpenReadWrite(p.path, p.timezone)
	if err != nil {
		// Catastrophique : RO fermé, RW refuse. State → Error.
		p.state.Store(int32(StateError))
		recordStateTransition(StateDraining, StateError)
		swapFailuresTotal.Add(failReasonAcquireWriter, 1)
		close(p.ready) // débloquer les Get (ils verront StateError)
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

	return &WriterHandle{
		provider: p,
		rwHandle: rwHandle,
		lease:    lease,
	}, nil
}

// releaseWriter est appelé par WriterHandle.Release. Inverse AcquireWriter :
// ferme RW, rouvre RO, débloque les Get en attente.
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
		return
	}

	// Reopen RO échoué → StateError + retry loop async.
	p.state.Store(int32(StateError))
	recordStateTransition(StateReopening, StateError)
	swapFailuresTotal.Add(failReasonReopenRO, 1)
	close(p.ready) // débloquer les waiters (ils verront StateError)
	p.mu.Unlock()
	go p.retryReopenLoop()
}

// tryReopenROLocked tente d'ouvrir une nouvelle conn RO et met à jour p.handle.
// Doit être appelée avec p.mu tenu.
//
// Si le hook test failNextReopen est armé, retourne false sans tenter l'open
// (le flag est consommé via CompareAndSwap pour ne fail qu'une fois).
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

// retryReopenLoop tente de récupérer après un échec de reopen RO. Backoff
// exponentiel borné (max retryMaxAttempts tentatives). En cas de succès,
// state passe Error → RO et les futurs Get fonctionnent. En cas d'échec
// définitif, state reste Error — recovery manuelle requise (restart serveur).
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
			// Provider a été réinitialisé ou fermé entre temps.
			p.mu.Unlock()
			return
		}
		if p.tryReopenROLocked() {
			p.state.Store(int32(StateRO))
			recordStateTransition(StateError, StateRO)
			// Note : `ready` est déjà fermé (fermé lors du passage en
			// StateError). Les futurs Get verront StateRO directement.
			// Pas besoin de close à nouveau ni de recréer le canal.
			p.mu.Unlock()
			slog.Info("sharedprovider: recovered from StateError", "path", p.path, "attempt", attempt+1)
			return
		}
		p.mu.Unlock()
	}
	slog.Error("sharedprovider: retry reopen RO definitively failed",
		"path", p.path, "attempts", retryMaxAttempts)
}

// State implémente Provider.State.
func (p *providerImpl) State() State {
	return State(p.state.Load())
}

// Path implémente Provider.Path.
func (p *providerImpl) Path() string {
	return p.path
}

// Close implémente Provider.Close. Idempotent.
//
// Si un swap est en cours (state ∈ {Draining, RW, Reopening}), Close attend
// pour prendre le mutex et ferme proprement. Les Get en attente verront
// StateClosed via le ready close + relecture.
func (p *providerImpl) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	prev := State(p.state.Load())
	if prev == StateClosed {
		return nil
	}

	p.state.Store(int32(StateClosed))
	recordStateTransition(prev, StateClosed)

	// Débloquer les Get en attente : si ready n'est pas déjà fermé, le fermer.
	// Pattern non-bloquant pour close idempotent.
	select {
	case <-p.ready:
		// déjà fermé
	default:
		close(p.ready)
	}

	if p.handle != nil {
		if err := p.handle.Close(); err != nil {
			p.handle = nil
			return fmt.Errorf("sharedprovider: close %s: %w", p.path, err)
		}
		p.handle = nil
	}
	return nil
}
