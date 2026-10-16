package slots

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Au plus n titulaires à la fois ; les autres attendent leur tour.
func TestPool_NeverExceedsCapacity(t *testing.T) {
	p := New(2)
	var inFlight, peak atomic.Int32
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := p.Acquire(context.Background())
			if err != nil {
				t.Errorf("Acquire: %v", err)
				return
			}
			n := inFlight.Add(1)
			for {
				m := peak.Load()
				if n <= m || peak.CompareAndSwap(m, n) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			inFlight.Add(-1)
			release()
		}()
	}
	wg.Wait()
	if got := peak.Load(); got != 2 {
		t.Fatalf("pic de titulaires = %d, attendu 2", got)
	}
}

// Un contexte fini met fin à l'attente sans prendre de place.
func TestPool_AcquireHonoursContext(t *testing.T) {
	p := New(1)
	release, err := p.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := p.Acquire(ctx); err == nil {
		t.Fatal("Acquire doit échouer quand le pool est plein et le contexte fini")
	}
	release()
	if p.InUse() != 0 {
		t.Fatalf("places tenues = %d, attendu 0", p.InUse())
	}
	r2, err := p.Acquire(context.Background())
	if err != nil {
		t.Fatalf("la place libérée doit être reprenable: %v", err)
	}
	r2()
}

// Libérer deux fois ne rend pas deux places.
func TestPool_ReleaseIsIdempotent(t *testing.T) {
	p := New(1)
	release, _ := p.Acquire(context.Background())
	release()
	release()
	r1, _ := p.Acquire(context.Background())
	defer r1()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := p.Acquire(ctx); err == nil {
		t.Fatal("une double libération a créé une place fantôme")
	}
}

// Waiting compte les demandes bloquées faute de place, et elles seules.
func TestPool_WaitingCountsBlockedAcquirers(t *testing.T) {
	p := New(1)
	release, err := p.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if p.Waiting() != 0 {
		t.Fatalf("Waiting = %d sans demande bloquée, attendu 0", p.Waiting())
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := p.Acquire(ctx)
		done <- err
	}()
	deadline := time.Now().Add(5 * time.Second)
	for p.Waiting() != 1 {
		if time.Now().After(deadline) {
			t.Fatalf("Waiting = %d, attendu 1", p.Waiting())
		}
		runtime.Gosched()
	}
	cancel()
	if err := <-done; err == nil {
		t.Fatal("la demande annulée devait échouer")
	}
	if p.Waiting() != 0 {
		t.Fatalf("Waiting = %d après l'abandon, attendu 0", p.Waiting())
	}
	release()
}
