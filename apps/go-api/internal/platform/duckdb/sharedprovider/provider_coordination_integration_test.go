//go:build integration

package sharedprovider_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestProvider_HTTPReadersWaitDuringSync_integration (T3 du plan) vérifie
// le contrat utilisateur principal du B-swap : pendant qu'un sync writer
// tient le handle RW, les Get HTTP attendent (gating via ready chan) et
// reprennent dès le release.
//
// Assertions clés :
//  1. Aucune erreur HTTP — tous les Get finissent par réussir
//  2. Au moins un Get a attendu > 100ms — preuve du gating effectif
//  3. Aucun Get qui RÉUSSIT n'a vu StateRW au moment du SELECT — preuve
//     que le drain WaitGroup a empêché la race "handle closed while reading"
func TestProvider_HTTPReadersWaitDuringSync_integration(t *testing.T) {
	path := setupSharedDB(t)

	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()

	var wg sync.WaitGroup
	var (
		maxWaitNs atomic.Int64
		seenRW    atomic.Int64
		httpOK    atomic.Int64
		httpErr   atomic.Int64
	)

	deadline := time.Now().Add(500 * time.Millisecond)

	// 10 readers HTTP en boucle.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				start := time.Now()
				db, release, err := p.Get(ctx)
				waited := time.Since(start).Nanoseconds()

				// Track max wait observed across goroutines.
				for {
					cur := maxWaitNs.Load()
					if waited <= cur {
						break
					}
					if maxWaitNs.CompareAndSwap(cur, waited) {
						break
					}
				}

				if err != nil {
					httpErr.Add(1)
					continue
				}

				// Si on tient un release valide, le state ne doit JAMAIS être
				// RW (preuve que le drain WaitGroup nous protège).
				if p.State() == sharedprovider.StateRW {
					seenRW.Add(1)
				}

				var v string
				queryErr := db.QueryRowContext(ctx, "SELECT version()").Scan(&v)
				release()
				if queryErr != nil {
					httpErr.Add(1)
				} else {
					httpOK.Add(1)
				}
			}
		}()
	}

	// Laisser les readers démarrer un peu, puis le sync writer entre en scène.
	time.Sleep(50 * time.Millisecond)
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	w.Release()

	wg.Wait()

	if httpErr.Load() > 0 {
		t.Errorf("%d erreurs HTTP, attendu 0 (preuve : drain protège des handle.Close)", httpErr.Load())
	}
	if httpOK.Load() == 0 {
		t.Error("aucune query HTTP réussie")
	}
	if seenRW.Load() > 0 {
		t.Errorf("%d Get ont vu StateRW (race drain non bloquante), attendu 0", seenRW.Load())
	}
	maxWait := time.Duration(maxWaitNs.Load())
	if maxWait < 100*time.Millisecond {
		t.Errorf("max wait observée = %v, attendu > 100ms (preuve du gating effectif pendant les 200ms RW)",
			maxWait)
	}
	t.Logf("coordination stats : %d HTTP OK, %d HTTP err, max wait %v",
		httpOK.Load(), httpErr.Load(), maxWait)
}
