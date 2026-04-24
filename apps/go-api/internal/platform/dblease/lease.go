// Package dblease — write lease par chemin DB, partagé entre sync et platform.
//
// Garantit qu'une seule goroutine à la fois écrit dans une base DuckDB donnée.
// Extrait de internal/sync pour permettre à PersistSink (platform/duckdb) de
// partager le même mécanisme sans créer de cycle d'import.
//
// Deux variantes :
//   - AcquireLease(path, timeout) — attente bornée (tests, chemins best-effort).
//   - AcquireLeaseCtx(ctx, path) — attente pilotée par le contexte appelant
//     (syncs longs, backfill, pipeline weapons).
//
// Usage :
//
//	release, err := AcquireLeaseCtx(ctx, dbPath)
//	if err != nil { return err }
//	defer release()
package dblease

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	// PlayerLeaseTimeout est le timeout best-effort pour les chemins player courts.
	PlayerLeaseTimeout = 5 * time.Second
	// MetadataLeaseTimeout est le timeout best-effort pour metadata.duckdb.
	MetadataLeaseTimeout = 10 * time.Second
	// SharedLeaseTimeout est le timeout best-effort pour shared_matches_v2.duckdb.
	// Utilisé pour les tests à borne dure et les backfills autonomes.
	SharedLeaseTimeout = 45 * time.Second

	// pollInterval est le délai entre deux tentatives TryLock.
	pollInterval = 5 * time.Millisecond
)

var (
	leasesMu sync.Mutex
	leases   = map[string]*sync.Mutex{}
)

// leaseMutex retourne (et crée si absent) le mutex associé à un chemin DB.
func leaseMutex(path string) *sync.Mutex {
	leasesMu.Lock()
	defer leasesMu.Unlock()
	if mu, ok := leases[path]; ok {
		return mu
	}
	mu := &sync.Mutex{}
	leases[path] = mu
	return mu
}

// AcquireLease tente d'acquérir le verrou d'écriture pour un chemin DB.
//
// Retourne une fonction release() à appeler (via defer) une fois l'écriture terminée.
// Retourne une erreur si le verrou n'est pas disponible dans le délai imparti.
//
// Implémentation via TryLock + polling pour éviter les fuites de goroutines.
// Préférer AcquireLeaseCtx pour les flux de sync longs.
func AcquireLease(path string, timeout time.Duration) (func(), error) {
	mu := leaseMutex(path)
	deadline := time.Now().Add(timeout)

	for {
		if mu.TryLock() {
			return func() { mu.Unlock() }, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("write lease timeout (%v) pour %s — autre sync en cours?", timeout, path)
		}
		time.Sleep(pollInterval)
	}
}

// AcquireLeaseCtx tente d'acquérir le verrou d'écriture pour un chemin DB.
// L'attente s'arrête si le contexte est annulé (opération longue, shutdown).
//
// Retourne une fonction release() à appeler (via defer) une fois l'écriture terminée.
// Retourne une erreur si le contexte est annulé avant l'acquisition.
func AcquireLeaseCtx(ctx context.Context, path string) (func(), error) {
	mu := leaseMutex(path)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		if mu.TryLock() {
			return func() { mu.Unlock() }, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("write lease annulée (contexte fermé) pour %s: %w", path, ctx.Err())
		case <-ticker.C:
			// nouvelle tentative
		}
	}
}
