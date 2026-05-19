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

// TestProvider_MultiTitle_Isolated_integration (T7 du plan) vérifie que
// deux providers sur des chemins distincts (cas multi-titre :
// halo_infinite + halo_5 futur) n'interfèrent pas l'un avec l'autre.
// Un AcquireWriter sur A ne gate pas les Get sur B.
//
// Aujourd'hui mono-titre, mais Manager + dblease.leaseMutex(path) sont
// clés par path donc l'isolation est native. Ce test verrouille le
// contrat avant le commit 6 où main.go pourra itérer sur les titres.
func TestProvider_MultiTitle_Isolated_integration(t *testing.T) {
	pathA := setupSharedDB(t)
	pathB := setupSharedDB(t)

	mgr := sharedprovider.NewManager()
	defer func() { _ = mgr.Close() }()

	pA, err := mgr.For(pathA)
	if err != nil {
		t.Fatalf("For A: %v", err)
	}
	pB, err := mgr.For(pathB)
	if err != nil {
		t.Fatalf("For B: %v", err)
	}
	if pA == pB {
		t.Fatal("attendu providers différents pour paths différents")
	}
	if pA.Path() != pathA {
		t.Errorf("pA.Path() = %q, attendu %q", pA.Path(), pathA)
	}
	if pB.Path() != pathB {
		t.Errorf("pB.Path() = %q, attendu %q", pB.Path(), pathB)
	}

	ctx := context.Background()

	// Acquire writer sur A — passe en StateRW.
	wA, err := pA.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter A: %v", err)
	}
	defer wA.Release()

	if got := pA.State(); got != sharedprovider.StateRW {
		t.Errorf("A state = %v, attendu RW", got)
	}
	if got := pB.State(); got != sharedprovider.StateRO {
		t.Errorf("B state = %v, attendu RO (isolation cross-titre)", got)
	}

	// Get sur B doit retourner immédiatement — pas de gating cross-titre.
	start := time.Now()
	dbB, releaseB, err := pB.Get(ctx)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Get B pendant A en RW: %v", err)
	}
	defer releaseB()

	// Tolérance large pour absorber le jitter Windows + warmup DuckDB.
	// Le vrai signal : pas plusieurs centaines de ms (gating effectif).
	if elapsed > 50*time.Millisecond {
		t.Errorf("Get B a pris %v, attendu < 50ms (pas de gating cross-titre)", elapsed)
	}

	var v string
	if err := dbB.QueryRowContext(ctx, "SELECT version()").Scan(&v); err != nil {
		t.Errorf("ping B pendant A en RW: %v", err)
	}
}

// TestManager_ConcurrentForSamePath_NoDuplicate vérifie qu'N appels
// concurrents à For(path) sur le même chemin ne créent qu'un seul
// Provider. Cas typique au boot serveur multi-titre : initialisation
// parallèle des composants qui consomment shared (bootstrap_repo, pool
// joueur, sync engine — futurs callers des commits 6-8).
func TestManager_ConcurrentForSamePath_NoDuplicate(t *testing.T) {
	path := setupSharedDB(t)

	mgr := sharedprovider.NewManager()
	defer func() { _ = mgr.Close() }()

	const n = 50
	providers := make([]sharedprovider.Provider, n)
	var wg sync.WaitGroup
	var errCount atomic.Int64

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			p, err := mgr.For(path)
			if err != nil {
				errCount.Add(1)
				return
			}
			providers[idx] = p
		}(i)
	}
	wg.Wait()

	if errCount.Load() > 0 {
		t.Fatalf("%d erreurs sur %d For() concurrents", errCount.Load(), n)
	}

	first := providers[0]
	if first == nil {
		t.Fatal("provider #0 est nil — toutes les goroutines ont échoué")
	}
	for i, p := range providers {
		if p != first {
			t.Errorf("provider #%d != provider #0 (LoadOrStore race a créé un duplicate)", i)
		}
	}
}

// TestManager_CloseAllProviders vérifie que Manager.Close ferme bien tous
// les providers gérés (utile au shutdown serveur multi-titre).
func TestManager_CloseAllProviders(t *testing.T) {
	pathA := setupSharedDB(t)
	pathB := setupSharedDB(t)

	mgr := sharedprovider.NewManager()

	pA, err := mgr.For(pathA)
	if err != nil {
		t.Fatalf("For A: %v", err)
	}
	pB, err := mgr.For(pathB)
	if err != nil {
		t.Fatalf("For B: %v", err)
	}

	if err := mgr.Close(); err != nil {
		t.Fatalf("Manager.Close: %v", err)
	}

	if got := pA.State(); got != sharedprovider.StateClosed {
		t.Errorf("après Manager.Close, A state = %v, attendu Closed", got)
	}
	if got := pB.State(); got != sharedprovider.StateClosed {
		t.Errorf("après Manager.Close, B state = %v, attendu Closed", got)
	}

	// Idempotence : Close() à nouveau doit être no-op.
	if err := mgr.Close(); err != nil {
		t.Errorf("Manager.Close #2 doit être no-op, obtenu : %v", err)
	}
}
