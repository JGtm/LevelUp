package sharedprovider

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"

	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

// Provider est le contrat exposé aux consommateurs. Une implémentation owne
// le handle DuckDB pour un chemin donné et arbitre l'accès lecture/écriture.
//
// Une instance Provider est sûre pour usage concurrent : Get peut être
// appelé depuis N goroutines en parallèle. AcquireWriter (commit 3+) est
// sérialisé via dblease.
type Provider interface {
	// Get retourne un *sql.DB en mode RO, prêt à lire. En steady state
	// (StateRO), retour immédiat. Pendant un swap RW (commits 3+), bloque
	// jusqu'au retour en RO ou jusqu'à expiration du contexte.
	//
	// Le *sql.DB retourné a la durée de vie du Provider — ne pas le fermer
	// (Provider.Close s'en charge).
	Get(ctx context.Context) (*sql.DB, error)

	// State retourne l'état courant. Lecture atomique, sans verrou.
	// Utile pour les métriques, les tests et le debug runtime.
	State() State

	// Path retourne le chemin absolu de la DB ciblée.
	Path() string

	// Close ferme le handle DuckDB sous-jacent et marque le provider comme
	// fermé. Les Get/AcquireWriter ultérieurs retournent ErrProviderClosed.
	// Idempotent : un second Close est no-op.
	Close() error
}

// providerImpl est l'implémentation par défaut.
//
// Au commit 2 (RO steady state), la struct est minimale : un handle DuckDB
// owned, un état atomique, un mutex pour Close. Les champs liés au swap
// (writerMu, drainWG, ready chan) arrivent au commit 3.
type providerImpl struct {
	path     string
	timezone string

	// state encode l'état courant en int32 pour load/store atomiques.
	// Lecture via State() sans verrou — sûr en lecture pure.
	state atomic.Int32

	// mu protège les transitions d'état et l'accès à handle pendant un
	// changement (Close, swap futur). Get lit handle sans prendre mu en
	// steady state (la valeur est stable tant que state == StateRO).
	mu     sync.Mutex
	handle *duckdbpkg.DB
}

// New ouvre une nouvelle instance Provider sur path en mode read-only.
//
// timezone (optionnel) : nom IANA passé à duckdb.OpenReadOnly ; permet
// d'appliquer SET TimeZone sur chaque conn (utile pour la cohérence
// d'affichage des TIMESTAMP WITH TIME ZONE).
//
// Retourne une erreur si l'ouverture DuckDB échoue. L'appelant DOIT appeler
// Close() une fois fini (typiquement via defer en main.go).
//
// Préférer Manager.For() en prod pour bénéficier du caching par chemin
// (utile en multi-titre, cf. commit 5).
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
		path:     path,
		timezone: tz,
		handle:   handle,
	}
	p.state.Store(int32(StateRO))
	recordStateTransition(StateRO, StateRO) // publie la gauge initiale
	return p, nil
}

// Get implémente Provider.Get.
//
// Commit 2 : pas encore de gating (state n'est jamais Draining/RW en
// pratique tant que AcquireWriter n'est pas implémenté). On vérifie juste
// Closed et on retourne la conn sous-jacente.
//
// Au commit 3, on ajoutera l'attente du retour en StateRO via le canal
// ready.
func (p *providerImpl) Get(ctx context.Context) (*sql.DB, error) {
	switch State(p.state.Load()) {
	case StateClosed:
		return nil, ErrProviderClosed
	case StateError:
		return nil, ErrSwapFailed
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Steady state RO : retour direct. handle.SQLDb() est stable sur la
	// durée de vie du provider (jusqu'à un swap RW au commit 3+).
	return p.handle.SQLDb(), nil
}

// State implémente Provider.State. Lecture atomique sans verrou.
func (p *providerImpl) State() State {
	return State(p.state.Load())
}

// Path implémente Provider.Path.
func (p *providerImpl) Path() string {
	return p.path
}

// Close implémente Provider.Close. Idempotent.
func (p *providerImpl) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	prev := State(p.state.Load())
	if prev == StateClosed {
		return nil
	}

	p.state.Store(int32(StateClosed))
	recordStateTransition(prev, StateClosed)

	if p.handle != nil {
		if err := p.handle.Close(); err != nil {
			return fmt.Errorf("sharedprovider: close %s: %w", p.path, err)
		}
		p.handle = nil
	}
	return nil
}
