package auth

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestXBLUserAuthSlot_BoundsConcurrency vérifie que le sémaphore XBL user-token
// borne la concurrence à xblUserAuthMaxConcurrent, quel que soit le nombre
// d'appelants (refresher pool + user-facing) en parallèle.
func TestXBLUserAuthSlot_BoundsConcurrency(t *testing.T) {
	const callers = 24

	var current int32
	var maxObserved int32
	var wg sync.WaitGroup

	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := acquireXBLUserAuthSlot(context.Background())
			if err != nil {
				t.Errorf("acquireXBLUserAuthSlot: %v", err)
				return
			}
			defer release()

			c := atomic.AddInt32(&current, 1)
			for {
				m := atomic.LoadInt32(&maxObserved)
				if c <= m || atomic.CompareAndSwapInt32(&maxObserved, m, c) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond) // tenir le slot pour forcer le contention
			atomic.AddInt32(&current, -1)
		}()
	}
	wg.Wait()

	if maxObserved > xblUserAuthMaxConcurrent {
		t.Errorf("concurrence XBL observée = %d, borne = %d", maxObserved, xblUserAuthMaxConcurrent)
	}
	if maxObserved == 0 {
		t.Error("aucune acquisition observée — test cassé")
	}
}

// TestXBLUserAuthSlot_RespectsContextCancel vérifie qu'une acquisition rend la main
// proprement si le contexte est annulé pendant l'attente (pas de blocage infini).
func TestXBLUserAuthSlot_RespectsContextCancel(t *testing.T) {
	// Saturer le sémaphore.
	releases := make([]func(), 0, xblUserAuthMaxConcurrent)
	for i := 0; i < xblUserAuthMaxConcurrent; i++ {
		release, err := acquireXBLUserAuthSlot(context.Background())
		if err != nil {
			t.Fatalf("acquire slot %d: %v", i, err)
		}
		releases = append(releases, release)
	}
	defer func() {
		for _, r := range releases {
			r()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := acquireXBLUserAuthSlot(ctx)
	if err == nil {
		t.Fatal("acquire aurait dû échouer (sémaphore saturé + ctx expiré)")
	}
}
