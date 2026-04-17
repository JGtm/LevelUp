// Package sync — lease.go : write lease par chemin DB.
//
// Garantit qu'une seule goroutine à la fois écrit dans une base DuckDB donnée.
// Portage du _db_lock asyncio.Lock() Python — adapté au modèle goroutines Go.
//
// Usage :
//
//	release, err := AcquireLease(dbPath, 5*time.Second)
//	if err != nil { return err }
//	defer release()
package sync

import (
	"fmt"
	"sync"
	"time"
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
// Implémentation via TryLock + polling pour éviter les fuites de goroutines :
// l'ancienne version (go func { mu.Lock() }) laissait une goroutine bloquée indéfiniment
// si le timeout survenait avant l'acquisition.
//
// Portage du comportement du _db_lock Python avec timeout 5s.
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
		time.Sleep(5 * time.Millisecond)
	}
}
