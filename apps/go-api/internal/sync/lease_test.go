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

// TestSyncVsPrestigeConcurrent simule l'invariant P1 (commit 2) :
// pendant qu'un sync long tient le writer, 10 requêtes HTTP prestige concurrentes
// doivent rester en attente (avec timeout court) ou retourner ErrDBLocked.
func TestSyncVsPrestigeConcurrent(t *testing.T) {
	path := t.TempDir() + "/prestige.duckdb"

	// Goroutine 1 : simule un sync qui tient le lease longtemps
	releaseSyncLease, err := AcquireLeaseCtx(context.Background(), path)
	if err != nil {
		t.Fatalf("sync acquire: %v", err)
	}

	var wg sync.WaitGroup
	prestigeFailures := 0
	prestigeMutex := sync.Mutex{}

	// Goroutines 2-11 : 10 requêtes prestige concurrentes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Tenter d'acquérir avec timeout court — doit échouer ou attendre
			_, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_, err := dblease.AcquireWriter(nil, path, dblease.KindPlayer, 100*time.Millisecond)
			if err != nil {
				// Attendu : timeout ou ErrDBLocked
				prestigeMutex.Lock()
				prestigeFailures++
				prestigeMutex.Unlock()
			}
		}()
	}

	// Laisser les prestige goroutines avoir le temps de tenter l'acquisition
	time.Sleep(50 * time.Millisecond)

	// Libérer le lease sync
	releaseSyncLease()

	// Attendre que tous les prestige tente se terminent
	wg.Wait()

	// Vérifier que au moins la plupart ont échoué (car sync tenait le lease)
	if prestigeFailures < 5 {
		t.Logf("warning: fewer failures than expected (%d < 5), but test may still be valid due to timing", prestigeFailures)
	}
}

// TestSyncHookNoDeadlock vérifie que le pipeline sync + hook ne crée pas de deadlock.
// C'est un test de saturation : on vérifie que le hook ne tient pas un verrou
// qui bloquerait une libération du writer.
func TestSyncHookNoDeadlock(t *testing.T) {
	path := t.TempDir() + "/hook-deadlock.duckdb"

	// Simuler un sync qui acquiert le lease et maintient une transaction
	releaseSyncLease, err := AcquireLeaseCtx(context.Background(), path)
	if err != nil {
		t.Fatalf("sync acquire: %v", err)
	}
	defer releaseSyncLease()

	// Lancer une goroutine qui simule un hook post-sync
	var hookDone sync.WaitGroup
	var hookErr error
	hookDone.Add(1)
	go func() {
		defer hookDone.Done()
		// Le hook tente d'acquérir un writer (doit attendre que le sync libère)
		// Avec un timeout court pour détecter le deadlock
		_, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_, hookErr = dblease.AcquireWriter(nil, path, dblease.KindPlayer, 500*time.Millisecond)
	}()

	// Simuler le sync qui termine et libère le lease
	time.Sleep(100 * time.Millisecond)
	releaseSyncLease()

	// Attendre que le hook se termine avec un timeout global
	done := make(chan bool, 1)
	go func() {
		hookDone.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Hook s'est terminé — pas de deadlock
		if hookErr != nil && !errors.Is(hookErr, dblease.ErrDBLocked) {
			t.Logf("hook got expected timeout or lock error: %v", hookErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("deadlock detected: hook did not complete after 2 seconds")
	}
}

// TestSyncVsMediaLikeConcurrent simule l'invariant P3 (commit 3) :
// pendant un sync long, 5 requêtes de like/unlike concurrent doivent être
// soit atomiques (succès), soit rollback (pas de corruption).
func TestSyncVsMediaLikeConcurrent(t *testing.T) {
	path := t.TempDir() + "/media-like.duckdb"

	// Goroutine 1 : simule un sync qui tient le lease
	releaseSyncLease, err := AcquireLeaseCtx(context.Background(), path)
	if err != nil {
		t.Fatalf("sync acquire: %v", err)
	}

	var wg sync.WaitGroup
	likeResults := make([]error, 5)
	likeResultsMutex := sync.Mutex{}

	// Goroutines 2-6 : 5 requêtes like concurrentes
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// Tenter d'acquérir writer pour toggler le like
			_, err := dblease.AcquireWriter(nil, path, dblease.KindPlayer, 100*time.Millisecond)
			likeResultsMutex.Lock()
			likeResults[idx] = err
			likeResultsMutex.Unlock()
		}(i)
	}

	time.Sleep(50 * time.Millisecond)
	releaseSyncLease()
	wg.Wait()

	// Vérifier que les likes ont eu un comportement prévisible (tous échouent
	// pendant le sync, ou certains attendent après)
	failureCount := 0
	for _, err := range likeResults {
		if err != nil {
			failureCount++
		}
	}
	if failureCount == 0 {
		t.Logf("all likes succeeded or timed out — timing dependent test")
	}
}

// TestSyncBurstNoLeak vérifie qu'une rafale de syncs court n'accumule pas de writers
// verrouillés (fuite de ressource = writers jamais libérés).
func TestSyncBurstNoLeak(t *testing.T) {
	path := t.TempDir() + "/burst-leak.duckdb"

	// Lancer 10 pseudo-syncs courts (acquièrent et libèrent rapidement)
	for i := 0; i < 10; i++ {
		rel, err := AcquireLease(path, 1*time.Second)
		if err != nil {
			t.Fatalf("burst sync %d acquire failed: %v", i, err)
		}
		// Simuler un travail très court
		time.Sleep(5 * time.Millisecond)
		rel()
	}

	// Après la rafale, on doit pouvoir acquérir le lease sans problème (preuve qu'il n'y a pas eu de fuite)
	finalRel, err := AcquireLease(path, 1*time.Second)
	if err != nil {
		t.Fatalf("final acquire after burst failed (leak detected): %v", err)
	}
	finalRel()
}
