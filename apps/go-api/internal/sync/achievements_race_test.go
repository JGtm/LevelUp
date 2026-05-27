//go:build integration

// Package sync — achievements_race_test.go : garde-rail anti-régression pour
// la race "TransactionContext Error: Conflict on update!" sur l'upsert
// achievement_definitions (incident 2026-05-27).
//
// CONTEXTE :
//   - `xbox_achievement_definitions` (table metadata.duckdb) est GLOBALE :
//     1 row par achievement_id, ~144 rows communs aux 4 joueurs.
//   - Le post-sync de chaque joueur appelle `SyncAchievements` qui fait
//     `INSERT ... ON CONFLICT DO UPDATE` sur cette table.
//   - Le scheduler (auto_sync) et le Coordinator (watcher) exécutent les
//     post-syncs en parallèle (`parallel_slots:2`, errgroup).
//   - Sans sérialisation applicative, 2 transactions DuckDB tentent un
//     UPDATE sur la même row → "Conflict on update!" → 1 sync failed.
//
// FIX (engine_postsync.go runAchievementsSync) : prendre un
// `dblease.AcquireWriterCtx(ctx, nil, metadataDBPath, KindMetadata)` AVANT
// l'ouverture du handle. Le 2ème caller bloque jusqu'au Release du 1er,
// AUCUNE contention DuckDB.
//
// Ce test reproduit la situation : 2 goroutines acquièrent le lease
// metadata sur le même path. La 2ème DOIT attendre (drain) jusqu'au
// Release de la 1ère.
package sync

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/platform/dblease"
)

// TestAchievementsRace_DBLeaseSerializesWriters vérifie que 2 callers
// concurrents sur dblease.AcquireWriterCtx(KindMetadata, samePath) se
// sérialisent. C'est la garantie applicative qui élimine la race observée
// dans logs/sync.log 23:58:11 :
//
//	"achievements: upsert definitions: ... TransactionContext Error: Conflict on update!"
func TestAchievementsRace_DBLeaseSerializesWriters(t *testing.T) {
	tmpDir := t.TempDir()
	metaPath := filepath.Join(tmpDir, "metadata.duckdb")

	var holdingFirst atomic.Bool
	var secondAcquiredAt atomic.Int64
	var firstReleasedAt atomic.Int64

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine A : prend le lease en premier, tient 200ms.
	go func() {
		defer wg.Done()
		lease, err := dblease.AcquireWriterCtx(context.Background(), nil, metaPath, dblease.KindMetadata)
		if err != nil {
			t.Errorf("A: AcquireWriterCtx: %v", err)
			return
		}
		holdingFirst.Store(true)
		time.Sleep(200 * time.Millisecond)
		holdingFirst.Store(false)
		firstReleasedAt.Store(time.Now().UnixNano())
		lease.Release()
	}()

	// Laisse A acquérir le lease en premier.
	time.Sleep(50 * time.Millisecond)

	// Goroutine B : essaie d'acquérir pendant que A tient. DOIT bloquer.
	go func() {
		defer wg.Done()
		lease, err := dblease.AcquireWriterCtx(context.Background(), nil, metaPath, dblease.KindMetadata)
		if err != nil {
			t.Errorf("B: AcquireWriterCtx: %v", err)
			return
		}
		secondAcquiredAt.Store(time.Now().UnixNano())

		// Pendant que B tient, A doit avoir release.
		if holdingFirst.Load() {
			t.Error("B acquired lease while A was still holding it — sérialisation cassée")
		}
		lease.Release()
	}()

	wg.Wait()

	// B doit avoir acquis APRÈS le release de A.
	released := firstReleasedAt.Load()
	acquired := secondAcquiredAt.Load()
	if released == 0 || acquired == 0 {
		t.Fatal("timestamps manquants — un des goroutines n'a pas terminé")
	}
	if acquired < released {
		t.Errorf("B a acquis le lease AVANT que A le release — sérialisation cassée (B acquired @ %d ns, A released @ %d ns)", acquired, released)
	}
}

// TestAchievementsRace_DBLeaseDifferentKindsDoNotBlock vérifie qu'un lease
// KindMetadata ne bloque PAS un lease KindPlayer (ou KindSharedMatches).
// Si on bloquait toutes les kinds, le sync d'un joueur bloquerait toute
// activité metadata d'un autre — régression de débit inacceptable.
func TestAchievementsRace_DBLeaseDifferentKindsDoNotBlock(t *testing.T) {
	tmpDir := t.TempDir()
	metaPath := filepath.Join(tmpDir, "metadata.duckdb")
	playerPath := filepath.Join(tmpDir, "player.duckdb")

	metaLease, err := dblease.AcquireWriterCtx(context.Background(), nil, metaPath, dblease.KindMetadata)
	if err != nil {
		t.Fatalf("acquire meta: %v", err)
	}
	defer metaLease.Release()

	// Acquérir un autre lease (KindPlayer sur path différent) — doit passer
	// instantanément sans attendre le metaLease.
	done := make(chan struct{})
	go func() {
		playerLease, err := dblease.AcquireWriterCtx(context.Background(), nil, playerPath, dblease.KindPlayer)
		if err != nil {
			t.Errorf("acquire player: %v", err)
			close(done)
			return
		}
		playerLease.Release()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(500 * time.Millisecond):
		t.Error("acquire player lease bloqué par metaLease — les kinds devraient être indépendantes")
	}
}

// TestAchievementsRace_ContextCanceledReleasesLease vérifie qu'un Acquire
// avec ctx canceled retourne une erreur (ne se bloque pas indéfiniment).
// Sans ça, un crash post-sync laisserait le lease orphelin → blocage
// permanent du sync metadata jusqu'au restart serveur.
func TestAchievementsRace_ContextCanceledReleasesLease(t *testing.T) {
	tmpDir := t.TempDir()
	metaPath := filepath.Join(tmpDir, "metadata.duckdb")

	// A tient le lease pendant 500ms.
	lease, err := dblease.AcquireWriterCtx(context.Background(), nil, metaPath, dblease.KindMetadata)
	if err != nil {
		t.Fatalf("A: %v", err)
	}
	go func() {
		time.Sleep(500 * time.Millisecond)
		lease.Release()
	}()

	// B essaye d'acquérir avec un ctx qui expire en 50ms.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = dblease.AcquireWriterCtx(ctx, nil, metaPath, dblease.KindMetadata)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("AcquireWriterCtx attendu en erreur (ctx expired), nil reçu")
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("AcquireWriterCtx mis %v avant retour (ctx 50ms timeout) — pas de respect du ctx", elapsed)
	}
}
