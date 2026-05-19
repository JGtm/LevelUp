package sharedprovider

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
)

// FromInMemoryDB construit un Provider qui wrap une *sql.DB DuckDB déjà
// ouverte (typiquement in-memory via sql.Open("duckdb", "")).
//
// Cas d'usage principal : les tests du sync engine (internal/sync/
// engine_e2e_test.go etc.) utilisent des paires (player, shared) in-memory
// pour la rapidité. Avant le commit 8l, ces tests appellent directement
// les helpers du package sync (newInMemoryDBs, OpenSharedDB...). Au
// commit 8l, le SyncEngine accepte un Provider — les tests doivent alors
// fournir un Provider qui satisfait l'API sans nécessiter un fichier
// physique.
//
// Comportement :
//   - Get retourne toujours le même *sql.DB + release no-op
//   - AcquireWriter retourne un WriterHandle wrappant le même *sql.DB,
//     release no-op (pas de swap, pas de lease dblease)
//   - State retourne StateRO (ou StateClosed après Close)
//   - Path retourne le `path` fourni (utilisé pour étiqueter les logs/metrics)
//   - Close marque comme fermé mais NE FERME PAS le db (le caller le possède)
//   - Subscribe est fonctionnel mais aucune notification n'est jamais émise
//     (pas de transition d'état)
//
// Cette implémentation est TRÈS minimaliste et ne respecte PAS toutes les
// invariants du vrai providerImpl (notamment : pas de drain WG, pas de
// dblease writer, deux AcquireWriter concurrents retourneraient deux
// WriterHandle sur le même db). Elle est strictement réservée aux tests.
func FromInMemoryDB(db *sql.DB, path string) Provider {
	return &inMemoryProvider{db: db, path: path}
}

// inMemoryProvider implémente Provider sur un *sql.DB partagé en mémoire.
type inMemoryProvider struct {
	db   *sql.DB
	path string

	closed atomic.Bool

	subsMu    sync.Mutex
	subs      map[int64]Subscriber
	nextSubID int64
}

// Get implémente Provider.Get. Retourne immédiatement le db sous-jacent
// avec un release no-op — pas de tracking de readers en vol (le caller
// est seul à utiliser le db en test).
func (p *inMemoryProvider) Get(_ context.Context) (*sql.DB, func(), error) {
	if p.closed.Load() {
		return nil, nil, ErrProviderClosed
	}
	return p.db, func() {}, nil
}

// AcquireWriter implémente Provider.AcquireWriter. Pas de swap RW : le db
// in-memory est toujours accessible en lecture/écriture. Pas de lease
// dblease — la sérialisation des writers concurrents n'est pas garantie,
// mais les tests utilisent ce provider en single-thread typiquement.
func (p *inMemoryProvider) AcquireWriter(_ context.Context) (*WriterHandle, error) {
	if p.closed.Load() {
		return nil, ErrProviderClosed
	}
	return &WriterHandle{
		db:        p.db,
		releaseFn: func() {}, // no-op : pas de swap, pas de lease
	}, nil
}

// State retourne StateRO en steady state (ou StateClosed après Close).
// Les états transitoires (Draining/RW/Reopening) n'existent pas dans cette
// implémentation simplifiée.
func (p *inMemoryProvider) State() State {
	if p.closed.Load() {
		return StateClosed
	}
	return StateRO
}

// Path retourne le path fourni à FromInMemoryDB.
func (p *inMemoryProvider) Path() string {
	return p.path
}

// Close marque le provider comme fermé. NE FERME PAS le *sql.DB sous-jacent
// — le caller a la responsabilité de fermer le db qu'il a fourni.
// Idempotent.
func (p *inMemoryProvider) Close() error {
	p.closed.Store(true)
	return nil
}

// Subscribe enregistre un callback. Conforme à l'API publique mais aucune
// notification n'est jamais émise (le provider n'a pas de transitions
// observables).
func (p *inMemoryProvider) Subscribe(fn Subscriber) func() {
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
