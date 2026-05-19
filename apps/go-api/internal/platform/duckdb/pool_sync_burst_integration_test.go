//go:build integration

// Test T5 burst — validation sous charge de la chaîne B-swap.
//
// Si ce test passe : le SharedDBProvider tient la charge sous burst de
// syncs concurrents, avec 0 Catalog Error pour les readers HTTP qui
// utilisent SharedReader (vs ~96% Catalog Errors avec le legacy ATTACH
// path retiré au commit 9c.4).
package duckdb_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestPool_T5BurstSharedReader_integration : variante post-8k.* du burst.
// Au lieu d'utiliser pool.Player.QueryRow (legacy ATTACH path qui souffre des
// 96% Catalog Errors pendant les swaps), les HTTP readers passent par
// SharedReader.Get() (Provider path migré commits 8k.0-8k.13).
//
// Contrat post-migration :
//   - 0 erreur sync (idem T5 original)
//   - 0 Catalog Error HTTP (la conn Provider n'a pas d'ATTACH à dropper)
//   - HTTP attend ~lockstep le swap RW (timeout configurable) mais réussit
//     toujours
//
// Cette validation matérialise le bénéfice principal de la migration : les
// repos qui ont migré vers SharedReader sont immunisés aux fenêtres de swap.
func TestPool_T5BurstSharedReader_integration(t *testing.T) {
	duckdb.CloseAll()
	t.Cleanup(duckdb.CloseAll)

	ctx := context.Background()
	dir := t.TempDir()
	sharedPath := dir + "/shared_matches_v2.duckdb"

	// Initialise le fichier shared via OpenReadWrite + schéma minimal.
	rwInit, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("OpenReadWrite shared init: %v", err)
	}
	if _, err := rwInit.Exec(ctx, `CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP)`); err != nil {
		_ = rwInit.Close()
		t.Fatalf("CREATE TABLE match_registry: %v", err)
	}
	_ = rwInit.Close()

	mgr := sharedprovider.NewManager()
	provider, err := mgr.For(sharedPath)
	if err != nil {
		t.Fatalf("Manager.For: %v", err)
	}
	t.Cleanup(func() { _ = provider.Close() })

	var (
		wg            sync.WaitGroup
		syncOK        atomic.Int64
		syncErr       atomic.Int64
		httpOK        atomic.Int64
		httpErr       atomic.Int64
		catalogErrors atomic.Int64
	)

	deadline := time.Now().Add(2 * time.Second)

	// 20 HTTP readers via SharedReader.Get() (Provider path).
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				db, release, err := provider.Get(ctx)
				if err != nil {
					httpErr.Add(1)
					continue
				}
				var count int
				err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM match_registry").Scan(&count)
				release()
				if err != nil {
					httpErr.Add(1)
					if errStr := err.Error(); len(errStr) > 0 &&
						(contains(errStr, "Catalog") || contains(errStr, "Table not found")) {
						catalogErrors.Add(1)
					}
				} else {
					httpOK.Add(1)
				}
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}

	// 5 sync goroutines : AcquireWriter + INSERT + Release.
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			seq := 0
			for time.Now().Before(deadline) {
				w, err := provider.AcquireWriter(ctx)
				if err != nil {
					syncErr.Add(1)
					continue
				}
				_, execErr := w.DB().ExecContext(ctx,
					"INSERT INTO match_registry (match_id, start_time) VALUES (?, NOW())",
					formatID(id, seq))
				w.Release()
				if execErr != nil {
					syncErr.Add(1)
				} else {
					syncOK.Add(1)
				}
				seq++
				time.Sleep(5 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("T5 burst SharedReader : sync OK=%d err=%d | HTTP OK=%d err=%d (Catalog err=%d)",
		syncOK.Load(), syncErr.Load(), httpOK.Load(), httpErr.Load(), catalogErrors.Load())

	if syncErr.Load() > 0 {
		t.Errorf("%d erreurs sync (attendu 0)", syncErr.Load())
	}
	if syncOK.Load() < 5 {
		t.Errorf("%d syncs OK (attendu ≥ 5)", syncOK.Load())
	}
	if catalogErrors.Load() > 0 {
		t.Errorf("%d Catalog Errors HTTP via SharedReader (attendu 0 — la conn Provider n'a pas d'ATTACH à dropper)",
			catalogErrors.Load())
	}
	if httpOK.Load() < int64(syncOK.Load()) {
		// HTTP OK doit largement dépasser les syncs (20 goroutines vs 5).
		t.Logf("HTTP OK=%d < syncOK=%d : faible ratio, peut indiquer un timeout Get prolongé",
			httpOK.Load(), syncOK.Load())
	}
}

// contains est un wrapper léger sur strings.Contains pour éviter l'import
// stdlib supplémentaire dans un test.
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// formatID génère un match_id unique pour les INSERTs du burst.
func formatID(goroutineID, seq int) string {
	return "burst-" + itoa(goroutineID) + "-" + itoa(seq)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
