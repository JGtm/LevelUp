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

// TestProvider_SyncBurstNoRegression_integration (T5 du plan) reproduit la
// charge qui plantait en prod : N syncs concurrents + M HTTP readers
// pendant une fenêtre soutenue. Doit passer zéro erreur DuckDB
// "different configuration", zéro panic, ≥ swap_total expected.
//
// Variante augmentée de internal/sync/burst_integration_test.go:117
// adaptée au sharedprovider — au commit 8 on connectera vraiment le sync
// engine, ici on simule directement via AcquireWriter.
func TestProvider_SyncBurstNoRegression_integration(t *testing.T) {
	path := setupSharedDB(t)

	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()

	var wg sync.WaitGroup
	var (
		httpOK  atomic.Int64
		httpErr atomic.Int64
		syncOK  atomic.Int64
		syncErr atomic.Int64
	)

	deadline := time.Now().Add(2 * time.Second)

	// 20 goroutines HTTP — lecteurs concurrents.
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				db, release, err := p.Get(ctx)
				if err != nil {
					httpErr.Add(1)
					continue
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

	// 5 goroutines sync writers — chacun cycle : Acquire → 50ms work → Release.
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				w, err := p.AcquireWriter(ctx)
				if err != nil {
					syncErr.Add(1)
					continue
				}
				time.Sleep(50 * time.Millisecond)
				w.Release()
				syncOK.Add(1)
			}
		}()
	}

	wg.Wait()

	if httpErr.Load() > 0 {
		t.Errorf("%d erreurs HTTP, attendu 0", httpErr.Load())
	}
	if syncErr.Load() > 0 {
		t.Errorf("%d erreurs sync, attendu 0 (preuve : zéro \"different configuration\")", syncErr.Load())
	}
	if syncOK.Load() < 5 {
		t.Errorf("%d syncs OK seulement, attendu ≥ 5 (5 goroutines × ≥1 cycle)", syncOK.Load())
	}
	if httpOK.Load() == 0 {
		t.Error("aucune query HTTP réussie sur 2s")
	}

	t.Logf("burst stats : %d HTTP OK, %d HTTP err, %d syncs OK, %d syncs err",
		httpOK.Load(), httpErr.Load(), syncOK.Load(), syncErr.Load())
}
