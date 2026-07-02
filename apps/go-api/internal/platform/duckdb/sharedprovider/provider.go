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

// defaultRWHoldWatchdog est le seuil au-delà duquel un writer RW encore tenu
// déclenche un WARN + compteur (étape 0 attribution contention). 2s : les
// fenêtres saines (batch persist) sont sub-seconde, et le budget user-facing
// fail-fast est de 3s — le watchdog pré-alerte donc AVANT l'impact utilisateur.
// Fire par ACQUISITION (une fois), pas en boucle : pas de spam.
const defaultRWHoldWatchdog = 2 * time.Second

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

	// swapBlockStart marque le début de la fenêtre où les Get sont gatés
	// (gateToDraining). Lu à la réouverture RO pour mesurer la durée TOTALE de
	// blocage des lecteurs (drain + maintien RW + reopen). Protégé par p.mu —
	// un seul swap à la fois (dblease sérialise les writers).
	swapBlockStart time.Time

	// rwWindowStart marque l'instant où le handle passe RW (swapToRW). Lu au
	// releaseWriter pour mesurer la fenêtre RW STRICTE (durée pendant laquelle
	// les Get sont gatés en RW). Protégé par p.mu — un seul swap à la fois.
	rwWindowStart time.Time

	// rwHolderLabel : label du DÉTENTEUR courant du writer (ctxkeys.DBWriterLabel),
	// capturé au swapToRW, consommé au releaseWriter pour ventiler la fenêtre RW
	// par détenteur (étape 0 attribution). Protégé par p.mu.
	rwHolderLabel string
	// rwWatchdog : timer armé au passage RW — WARN + compteur si le writer est
	// tenu au-delà de rwHoldWatchdog (les lectures sont gatées pendant ce temps).
	// Désarmé au releaseWriter. Protégé par p.mu ; le callback du timer ne prend
	// AUCUN lock (valeurs capturées) — pas de deadlock possible.
	rwWatchdog *time.Timer

	// readersWG track les Get en vol. Add(1) sous p.mu quand state=RO,
	// Done() dans le release retourné. AcquireWriter Wait() avant le close
	// du handle pour éviter "database is closed" côté caller.
	readersWG sync.WaitGroup

	readyTimeout     time.Duration
	retryBaseBackoff time.Duration
	drainTimeout     time.Duration
	rwHoldWatchdog   time.Duration

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
		rwHoldWatchdog:   defaultRWHoldWatchdog,
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
	// Fenêtre d'attente d'un swap : readyTimeout par défaut (robuste aux swaps
	// légitimes côté sync). Un caller user-facing peut poser un budget plus court
	// via WithSwapWaitBudget → fail-fast (503 Retry-After) au lieu de pendre 30s
	// quand un sync tient le writer RW. On ne borne QUE l'attente du swap, pas
	// l'exécution des requêtes (le db retourné garde le ctx du caller).
	swapWait := p.readyTimeout
	if b, ok := swapWaitBudget(ctx); ok && b < swapWait {
		swapWait = b
	}
	deadline := time.Now().Add(swapWait)
	// Phase 0 — stall lecteur réel : waitStart posé au 1er passage en attente
	// (état non-RO) ; le defer ajoute la durée totale d'attente UNE fois à la
	// sortie (évite le double-comptage entre itérations de la boucle).
	var waitStart time.Time
	defer func() {
		if !waitStart.IsZero() {
			readerStallNsTotal.Add(time.Since(waitStart).Nanoseconds())
		}
	}()
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
		if waitStart.IsZero() {
			waitStart = time.Now()
			readerDelayedTotal.Add(1)
		}
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
