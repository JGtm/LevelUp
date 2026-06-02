// Package dblease — lease_test.go : tests unitaires du mécanisme de write lease.
package dblease

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestAcquireLease_Basic(t *testing.T) {
	path := t.TempDir() + "/test.duckdb"

	release, err := AcquireLease(path, 2*time.Second)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if release == nil {
		t.Fatal("expected release function, got nil")
	}
	release()
}

func TestAcquireLease_Timeout(t *testing.T) {
	path := t.TempDir() + "/busy.duckdb"

	release1, err := AcquireLease(path, 2*time.Second)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	defer release1()

	_, err2 := AcquireLease(path, 50*time.Millisecond)
	if err2 == nil {
		t.Fatal("expected timeout error on second acquire")
	}
}

func TestAcquireLease_Sequential(t *testing.T) {
	path := t.TempDir() + "/seq.duckdb"

	r1, err := AcquireLease(path, time.Second)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	r1()

	r2, err := AcquireLease(path, time.Second)
	if err != nil {
		t.Fatalf("second acquire after release: %v", err)
	}
	r2()
}

func TestAcquireLease_DifferentPaths(t *testing.T) {
	dir := t.TempDir()
	path1 := dir + "/db1.duckdb"
	path2 := dir + "/db2.duckdb"

	r1, err1 := AcquireLease(path1, time.Second)
	r2, err2 := AcquireLease(path2, time.Second)
	if err1 != nil || err2 != nil {
		t.Fatalf("different paths should not block: err1=%v err2=%v", err1, err2)
	}
	r1()
	r2()
}

func TestAcquireLease_Concurrent(t *testing.T) {
	path := t.TempDir() + "/concurrent.duckdb"
	var wg sync.WaitGroup
	successes := make(chan bool, 5)

	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := AcquireLease(path, 2*time.Second)
			if err != nil {
				successes <- false
				return
			}
			time.Sleep(10 * time.Millisecond)
			release()
			successes <- true
		}()
	}

	wg.Wait()
	close(successes)

	ok := 0
	for s := range successes {
		if s {
			ok++
		}
	}
	if ok == 0 {
		t.Fatal("at least one goroutine should acquire the lease")
	}
}

// TestAcquireLease_NoGoroutineLeak vérifie que le timeout ne laisse pas de goroutine orpheline.
func TestAcquireLease_NoGoroutineLeak(t *testing.T) {
	path := t.TempDir() + "/leak.duckdb"

	release, err := AcquireLease(path, time.Second)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	before := runtime.NumGoroutine()
	_, err2 := AcquireLease(path, 30*time.Millisecond)
	if err2 == nil {
		t.Fatal("expected timeout error")
	}

	time.Sleep(50 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after > before+2 {
		t.Errorf("goroutine leak: before=%d after=%d", before, after)
	}
	release()
}

// TestAcquireLeaseCtx_Basic vérifie l'acquisition simple avec contexte.
func TestAcquireLeaseCtx_Basic(t *testing.T) {
	path := t.TempDir() + "/ctx.duckdb"
	ctx := context.Background()

	release, err := AcquireLeaseCtx(ctx, path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	release()
}

// TestAcquireLeaseCtx_CancelledContext vérifie l'échec immédiat sur contexte annulé.
func TestAcquireLeaseCtx_CancelledContext(t *testing.T) {
	path := t.TempDir() + "/ctxcancel.duckdb"

	// Verrouiller d'abord pour forcer l'attente
	release, err := AcquireLease(path, time.Second)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // déjà annulé

	_, err2 := AcquireLeaseCtx(ctx, path)
	if err2 == nil {
		t.Fatal("expected error for already-cancelled context")
	}
}

// TestAcquireLeaseCtx_FreeLeaseCancelledCtx : sur un lease LIBRE mais un ctx déjà
// annulé, on REFUSE le lease (ctx.Err() avant TryLock, défense-in-depth 2026-06-02)
// au lieu de l'accorder — évite une écriture après shutdown/CloseAll (#7659).
func TestAcquireLeaseCtx_FreeLeaseCancelledCtx(t *testing.T) {
	path := t.TempDir() + "/freecancel.duckdb" // lease JAMAIS pris (libre)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // déjà annulé

	rel, err := AcquireLeaseCtx(ctx, path)
	if err == nil {
		if rel != nil {
			rel()
		}
		t.Fatal("un lease libre ne doit PAS être accordé sur un ctx déjà annulé")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err devrait wrapper context.Canceled, got %v", err)
	}
}

// TestAcquireLeaseCtx_CancelDuringWait vérifie l'annulation pendant l'attente.
func TestAcquireLeaseCtx_CancelDuringWait(t *testing.T) {
	path := t.TempDir() + "/ctxwait.duckdb"

	release, err := AcquireLease(path, time.Second)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer release()

	ctx, cancel := context.WithCancel(context.Background())
	// Annuler après 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err2 := AcquireLeaseCtx(ctx, path)
	if err2 == nil {
		t.Fatal("expected error after context cancel")
	}
}

// TestAcquireLeaseCtx_Sequential vérifie la réacquisition après release.
func TestAcquireLeaseCtx_Sequential(t *testing.T) {
	path := t.TempDir() + "/ctxseq.duckdb"
	ctx := context.Background()

	r1, err := AcquireLeaseCtx(ctx, path)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	r1()

	r2, err := AcquireLeaseCtx(ctx, path)
	if err != nil {
		t.Fatalf("second acquire: %v", err)
	}
	r2()
}
