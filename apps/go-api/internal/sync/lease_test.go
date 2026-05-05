// Package sync — lease_test.go : tests unitaires du mécanisme de write lease.
//
//go:build integration

package sync

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/platform/dblease"
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

	// Acquérir le lease puis essayer d'en obtenir un second → timeout
	release1, err := AcquireLease(path, 2*time.Second)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	defer release1()

	// Second appel avec timeout court → doit échouer
	_, err2 := AcquireLease(path, 50*time.Millisecond)
	if err2 == nil {
		t.Fatal("expected timeout error on second acquire")
	}
}

func TestAcquireLease_Sequential(t *testing.T) {
	path := t.TempDir() + "/seq.duckdb"

	// Première acquisition → libération → deuxième acquisition
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

	// Leases sur des chemins différents ne bloquent pas
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

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := AcquireLease(path, 2*time.Second)
			if err != nil {
				successes <- false
				return
			}
			// Simuler un court travail
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
// C'est la régression que le fix TryLock a corrigée.
func TestAcquireLease_NoGoroutineLeak(t *testing.T) {
	path := t.TempDir() + "/leak.duckdb"

	// Acquérir le lease (le maintenir verrouillé)
	release, err := AcquireLease(path, time.Second)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	before := runtime.NumGoroutine()

	// Tenter 10 acquisitions qui vont toutes timeout
	for i := 0; i < 10; i++ {
		_, _ = AcquireLease(path, 20*time.Millisecond)
	}

	// Attendre que d'éventuelles goroutines orphelines se terminent
	time.Sleep(100 * time.Millisecond)

	after := runtime.NumGoroutine()
	release()

	// Avec l'ancien code (go func() { mu.Lock()... }), chaque timeout laissait
	// une goroutine bloquée. Avec TryLock, zéro fuite.
	leaked := after - before
	if leaked > 2 { // marge pour le GC et le scheduler
		t.Errorf("goroutine leak detected: %d goroutines added after 10 timeouts", leaked)
	}
}

// TestCoordination_SyncLease_BlocksHTTPWriter vérifie l'invariant central du
// commit 7 db-concurrency : le lease acquis via sync.AcquireLeaseCtx (legacy
// API utilisée par le sync engine) bloque bien l'acquisition concurrente via
// dblease.AcquireWriter (nouvelle API utilisée par les handlers HTTP des
// commits 2-6). Les deux partagent le mutex `dblease.leaseMutex(path)`.
func TestCoordination_SyncLease_BlocksHTTPWriter(t *testing.T) {
	path := t.TempDir() + "/coord.duckdb"

	// Goroutine sync : acquiert via sync.AcquireLeaseCtx (legacy).
	relSync, err := AcquireLeaseCtx(context.Background(), path)
	if err != nil {
		t.Fatalf("sync acquire: %v", err)
	}
	defer relSync()

	// Goroutine HTTP : tente l'acquisition via dblease.AcquireWriter (timeout court)
	// — doit échouer avec ErrDBLocked car le sync tient le mutex partagé.
	_, err = dblease.AcquireWriter(nil, path, dblease.KindPlayer, 50*time.Millisecond)
	if err == nil {
		t.Fatal("HTTP acquire should have timed out (sync holds lease via shared mutex)")
	}
	if !errors.Is(err, dblease.ErrDBLocked) {
		t.Errorf("err should wrap dblease.ErrDBLocked, got %v", err)
	}
}

// TestCoordination_HTTPWriter_BlocksSyncLease — sens inverse : le HTTP qui
// tient le writer via dblease.AcquireWriter doit faire attendre le sync qui
// utilise sync.AcquireLeaseCtx. Confirme la symétrie de la coordination.
func TestCoordination_HTTPWriter_BlocksSyncLease(t *testing.T) {
	path := t.TempDir() + "/coord-rev.duckdb"

	// Goroutine HTTP : acquiert via la nouvelle API dblease.AcquireWriter.
	wHTTP, err := dblease.AcquireWriter(nil, path, dblease.KindPlayer, time.Second)
	if err != nil {
		t.Fatalf("HTTP acquire: %v", err)
	}
	defer wHTTP.Release()

	// Goroutine sync : tente AcquireLeaseCtx — doit échouer/attendre car le
	// HTTP tient déjà le mutex.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = AcquireLeaseCtx(ctx, path)
	if err == nil {
		t.Fatal("sync AcquireLeaseCtx should have failed (HTTP holds lease)")
	}
}
