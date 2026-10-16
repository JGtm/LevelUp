package sync

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Des demandes simultanées pour une même clé ne lancent jamais deux recalculs à la fois
// et sont absorbées : N demandes déposées pendant un recalcul en cours produisent UN seul
// recalcul de rattrapage, lancé après leur dépôt.
func TestRecomputeCoalescer_CoalescesConcurrentRequests(t *testing.T) {
	var c recomputeCoalescer
	var running, maxRunning, runs atomic.Int32
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})

	fn := func(context.Context) error {
		n := running.Add(1)
		for {
			m := maxRunning.Load()
			if n <= m || maxRunning.CompareAndSwap(m, n) {
				break
			}
		}
		if runs.Add(1) == 1 {
			close(firstStarted)
			<-releaseFirst
		}
		running.Add(-1)
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.do(context.Background(), "player-1", fn); err != nil {
			t.Errorf("premier appel: %v", err)
		}
	}()
	<-firstStarted

	const late = 20
	for range late {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.do(context.Background(), "player-1", fn); err != nil {
				t.Errorf("appel tardif: %v", err)
			}
		}()
	}
	// Laisser les appels tardifs se déposer avant de libérer le premier recalcul.
	time.Sleep(100 * time.Millisecond)
	close(releaseFirst)
	wg.Wait()

	if got := maxRunning.Load(); got != 1 {
		t.Fatalf("recalculs simultanés = %d, attendu 1", got)
	}
	if got := runs.Load(); got != 2 {
		t.Fatalf("recalculs = %d, attendu 2 (le premier + un rattrapage pour les %d demandes tardives)", got, late)
	}
}

// Des clés distinctes ne s'attendent pas.
func TestRecomputeCoalescer_DistinctKeysRunIndependently(t *testing.T) {
	var c recomputeCoalescer
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- c.do(context.Background(), "a", func(context.Context) error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started
	if err := c.do(context.Background(), "b", func(context.Context) error { return nil }); err != nil {
		t.Fatalf("la clé b a été bloquée par la clé a: %v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

// L'erreur du recalcul est rendue à tous les demandeurs absorbés, et l'état est purgé :
// la demande suivante relance un recalcul.
func TestRecomputeCoalescer_SharesErrorAndResets(t *testing.T) {
	var c recomputeCoalescer
	boom := errors.New("boom")
	if err := c.do(context.Background(), "k", func(context.Context) error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("err = %v, attendu boom", err)
	}
	var runs atomic.Int32
	if err := c.do(context.Background(), "k", func(context.Context) error { runs.Add(1); return nil }); err != nil {
		t.Fatal(err)
	}
	if runs.Load() != 1 {
		t.Fatalf("runs = %d", runs.Load())
	}
	c.mu.Lock()
	n := len(c.runs)
	c.mu.Unlock()
	if n != 0 {
		t.Fatalf("%d entrée(s) laissée(s) après la fin des recalculs", n)
	}
}

// Un demandeur absorbé dont le contexte est annulé cesse d'attendre sans interrompre le recalcul.
func TestRecomputeCoalescer_WaiterCancellation(t *testing.T) {
	var c recomputeCoalescer
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	var once sync.Once
	go func() {
		done <- c.do(context.Background(), "k", func(context.Context) error {
			// La demande annulée a tout de même été notée : un rattrapage suit, inoffensif.
			once.Do(func() { close(started) })
			<-release
			return nil
		})
	}()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := c.do(ctx, "k", func(context.Context) error { return nil }); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, attendu DeadlineExceeded", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("le recalcul en cours a été touché: %v", err)
	}
}

// Un recalcul en échec est retenu jusqu'au prochain succès.
func TestRecomputeCoalescer_RemembersFailedLastRun(t *testing.T) {
	var c recomputeCoalescer
	if c.failedLast("k") {
		t.Fatal("clé vierge déclarée en échec")
	}
	_ = c.do(context.Background(), "k", func(context.Context) error { return errors.New("boom") })
	if !c.failedLast("k") {
		t.Fatal("échec non retenu")
	}
	if c.failedLast("autre") {
		t.Fatal("l'échec d'une clé déborde sur une autre")
	}
	_ = c.do(context.Background(), "k", func(context.Context) error { return nil })
	if c.failedLast("k") {
		t.Fatal("échec toujours retenu après un succès")
	}
}
