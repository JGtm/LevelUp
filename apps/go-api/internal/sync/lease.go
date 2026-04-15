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
// Portage du comportement du _db_lock Python avec timeout 5s.
func AcquireLease(path string, timeout time.Duration) (func(), error) {
	mu := leaseMutex(path)

	acquired := make(chan struct{}, 1)
	go func() {
		mu.Lock()
		acquired <- struct{}{}
	}()

	select {
	case <-acquired:
		return func() { mu.Unlock() }, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("write lease timeout (%v) pour %s — autre sync en cours?", timeout, path)
	}
}
