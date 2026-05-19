//go:build integration

// Package sync — engine_provider_backfill_e2e_test.go : E2E qui couvre
// RunBackfill en mode Provider (commit 11b). Ce test échouait en deadlock
// avant le fix B3 (double-dblease) — il valide que RunBackfill peut
// désormais s'exécuter avec WithSharedProvider sans bloquer.
//
// Couvre également le scénario "HTTP sync + auto_sync concurrents" :
// RunDelta et RunBackfill lancés en parallèle sur le même user partagent
// le même Provider, dblease arbitre, readers HTTP non-impactés.
package sync

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestE2E_RunBackfill_WithProvider_NoDeadlock_integration : test de régression
// du fix B3 (commit 11b). Avant ce fix, RunBackfill prenait `dblease shared`
// puis `OpenSharedDB` direct. Si on lui wirait `WithSharedProvider`,
// `acquireSharedWriter` tenterait de re-acquérir le même dblease →
// auto-deadlock jusqu'à expiration du ctx.
func TestE2E_RunBackfill_WithProvider_NoDeadlock_integration(t *testing.T) {
	env := newE2EEnv(t)
	// Pas de seed mock : RunBackfill détecte les matchs sans données et
	// retourne la liste — pas d'appel réseau requis.
	env.mock.history = nil

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Si le bug B3 était présent, ce call deadlockait jusqu'à expiration
	// du ctx 15s, puis retournait ErrDBLocked. Désormais il retourne
	// quasi-instantanément avec une liste vide (DBs vierges).
	missing, err := env.engine.RunBackfill(ctx, &SyncScope{})
	if err != nil {
		t.Fatalf("RunBackfill avec Provider : %v (était deadlock avant fix B3)", err)
	}
	if missing != nil {
		t.Logf("RunBackfill returned %d missing matches (expected 0 sur DB vide)", len(missing))
	}

	// Provider doit être revenu à StateRO après RunBackfill.
	if state := env.provider.State(); state != sharedprovider.StateRO {
		t.Errorf("Provider.State post-RunBackfill = %v (attendu StateRO)", state)
	}

	// Sanity : un second RunBackfill consécutif marche aussi (pas de leak
	// de lease entre les runs).
	if _, err := env.engine.RunBackfill(ctx, &SyncScope{}); err != nil {
		t.Errorf("RunBackfill 2nd call : %v", err)
	}
}

// TestE2E_RunDeltaAndBackfill_Concurrent_NoFriction_integration : scénario
// le plus rapproché du flow prod — RunDelta (auto_sync) et RunBackfill
// (HTTP-triggered) en parallèle sur le même user, sous trafic readers.
// dblease arbitre la sérialisation, Provider isole les readers des swaps.
//
// Ce test prouve que les fixes B2+B3 s'intègrent correctement :
//   - RunDelta + RunBackfill peuvent coexister sans deadlock
//   - Les readers continuent OK
//   - 0 Catalog Error / different configuration
func TestE2E_RunDeltaAndBackfill_Concurrent_NoFriction_integration(t *testing.T) {
	env := newE2EEnv(t)
	env.seedMockMatches("concurrent", 5)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var (
		runDeltaErr     error
		runBackfillErr  error
		readerOK        atomic.Int64
		readerErr       atomic.Int64
		readerCatalog   atomic.Int64
		readerDiffCfg   atomic.Int64
		stopReaders     atomic.Bool
		readersWG       sync.WaitGroup
	)

	// 5 readers HTTP-like en boucle.
	for i := 0; i < 5; i++ {
		readersWG.Add(1)
		go func() {
			defer readersWG.Done()
			for !stopReaders.Load() {
				db, release, err := env.pool.SharedReadDB().Get(ctx)
				if err != nil {
					if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
						return
					}
					readerErr.Add(1)
					classifyErr(err, &readerCatalog, &readerDiffCfg)
					continue
				}
				var n int
				qErr := db.QueryRowContext(ctx,
					"SELECT COUNT(*) FROM match_registry").Scan(&n)
				release()
				if qErr != nil {
					readerErr.Add(1)
					classifyErr(qErr, &readerCatalog, &readerDiffCfg)
				} else {
					readerOK.Add(1)
				}
				time.Sleep(time.Millisecond)
			}
		}()
	}

	// 2 writers en parallèle : RunDelta (sync nominal) + RunBackfill (HTTP-triggered).
	// dblease serialize les writes, mais on vérifie qu'aucun ne deadlock.
	var writersWG sync.WaitGroup
	writersWG.Add(2)
	go func() {
		defer writersWG.Done()
		_, runDeltaErr = env.engine.RunDelta(ctx, domain.SyncOptions{
			MatchType:         "matchmaking",
			MaxMatches:        5,
			WithParticipants:  true,
			WithMedals:        true,
			RequestsPerSecond: 100,
		})
	}()
	go func() {
		defer writersWG.Done()
		// Petite pause pour offset l'AcquireWriter sur RunBackfill — sinon
		// les 2 essaient de prendre le lease en même temps et un attend.
		time.Sleep(10 * time.Millisecond)
		_, runBackfillErr = env.engine.RunBackfill(ctx, &SyncScope{})
	}()
	writersWG.Wait()

	time.Sleep(150 * time.Millisecond)
	stopReaders.Store(true)
	readersWG.Wait()

	t.Logf("RunDelta+RunBackfill concurrent : readers OK=%d err=%d (Catalog=%d, DiffCfg=%d)",
		readerOK.Load(), readerErr.Load(),
		readerCatalog.Load(), readerDiffCfg.Load())

	if runDeltaErr != nil {
		t.Errorf("RunDelta : %v (attendu nil)", runDeltaErr)
	}
	if runBackfillErr != nil {
		t.Errorf("RunBackfill : %v (attendu nil — fix B3)", runBackfillErr)
	}
	if readerCatalog.Load() > 0 {
		t.Errorf("%d Catalog Error reader pendant writers concurrents (attendu 0)",
			readerCatalog.Load())
	}
	if readerDiffCfg.Load() > 0 {
		t.Errorf("%d \"different configuration\" reader (attendu 0 — bug initial éteint)",
			readerDiffCfg.Load())
	}
	if readerOK.Load() < 50 {
		t.Errorf("readerOK=%d trop faible (attendu ≥ 50)", readerOK.Load())
	}
	if state := env.provider.State(); state != sharedprovider.StateRO {
		t.Errorf("Provider.State post-concurrent = %v (attendu StateRO)", state)
	}
}
