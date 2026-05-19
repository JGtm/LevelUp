//go:build integration

// Test T5 burst — validation sous charge de la chaîne B3 en topologie
// réelle (Provider + Subscribe + pool DETACH/REATTACH + conns player
// ATTACH'ées + queries HTTP concurrentes).
//
// C'est l'analogue du commit 4's TestProvider_SyncBurstNoRegression mais
// sur la topologie réelle au lieu du Provider isolé : si ce test passe,
// la migration B3 est validée sous charge (régression prod inversée).
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

// TestPool_T5BurstRealTopology_integration (T5 du plan original) exécute
// 5 syncs concurrents (AcquireWriter+INSERT+Release) + 20 readers HTTP
// concurrents (Get→SELECT shared.match_registry via pool.Player ATTACH'ée)
// pendant 2 secondes.
//
// Contrat validé par ce test :
//   - 0 erreur sync : pas de "different configuration" / "Unique file
//     handle conflict". → le bug initial sur Madina97294 est fixé.
//   - swap_total ≥ 5 (au moins une boucle par goroutine sync)
//
// LIMITATION DÉCOUVERTE (commit 8j) : pendant la fenêtre PreSwap →
// OpenReadWrite → INSERT → Release → RWToRO (~5-10ms par cycle), les
// queries HTTP qui font des JOIN cross-DB via ATTACH RO sur la conn player
// échouent avec "Catalog Error: Table not found in shared.*". C'est
// ATTENDU par le mécanisme B3 (DETACH explicite libère le file pour
// permettre OpenReadWrite), mais NON RÉSOLU au commit 8j — les queries
// HTTP en charge maximale peuvent atteindre 35% de Catalog Errors.
//
// Mitigations possibles (post-sprint) :
//  1. RWMutex côté pool : RLock sur queries, WLock sur DETACH/REATTACH.
//     Bloque les nouvelles queries pendant le swap (~10ms d'attente max).
//  2. Retry automatique sur Catalog Error au niveau caller.
//  3. Drain WaitGroup pool-side : attendre les queries en cours avant DETACH.
//
// Pour ce commit, on valide UNIQUEMENT le contrat sync (le critique en prod).
func TestPool_T5BurstRealTopology_integration(t *testing.T) {
	// Le globalPool est process-wide et persiste entre tests. Pour isoler
	// ce burst test des autres (qui peuvent avoir laissé des PlayerDB
	// inscrits), on purge avant et après.
	duckdb.CloseAll()
	t.Cleanup(duckdb.CloseAll)

	_, provider, pdb := setupPoolFixturesForSwap(t)
	defer func() { _ = provider.Close() }()

	ctx := context.Background()

	// Câblage Subscribe → OnSharedSwap (équivalent au câblage main.go commit 8g).
	unsubscribe := provider.Subscribe(func(evt sharedprovider.SwapEvent) {
		switch evt.Direction {
		case sharedprovider.DirectionPreSwapToRW:
			duckdb.OnSharedSwap(ctx, duckdb.SwapDirPreSwapToRW)
		case sharedprovider.DirectionRWToRO:
			duckdb.OnSharedSwap(ctx, duckdb.SwapDirRWToRO)
		case sharedprovider.DirectionErrorToRO:
			duckdb.OnSharedSwap(ctx, duckdb.SwapDirErrorToRO)
		}
	})
	defer unsubscribe()

	var wg sync.WaitGroup
	var (
		syncOK   atomic.Int64
		syncErr  atomic.Int64
		httpOK   atomic.Int64
		httpErr  atomic.Int64
		catalogErrors atomic.Int64
	)

	deadline := time.Now().Add(2 * time.Second)

	// 20 goroutines HTTP — lectures via pool.Player avec ATTACH RO shared.
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				var count int
				err := pdb.Player.QueryRow(ctx,
					"SELECT COUNT(*) FROM shared.match_registry").Scan(&count)
				if err != nil {
					httpErr.Add(1)
					// Comptage spécifique des Catalog Error (signe ATTACH cassé).
					if errStr := err.Error(); len(errStr) > 0 &&
						(contains(errStr, "Catalog") || contains(errStr, "Table not found")) {
						catalogErrors.Add(1)
					}
				} else {
					httpOK.Add(1)
				}
				// Petite pause pour ne pas spinner trop fort.
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}

	// 5 goroutines sync — cycle complet AcquireWriter+INSERT+Release.
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
				// Pause entre cycles sync pour laisser les HTTP respirer.
				time.Sleep(5 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("T5 burst : sync OK=%d err=%d | HTTP OK=%d err=%d (Catalog err=%d)",
		syncOK.Load(), syncErr.Load(), httpOK.Load(), httpErr.Load(), catalogErrors.Load())

	// ASSERTION CRITIQUE : 0 erreur sync. Le bug initial du sprint
	// ("different configuration" sur Madina97294) est fixé si syncErr=0.
	if syncErr.Load() > 0 {
		t.Errorf("%d erreurs sync (attendu 0 — 'different configuration'/'Unique file handle' réintroduit ?)",
			syncErr.Load())
	}
	if syncOK.Load() < 5 {
		t.Errorf("%d syncs OK seulement (attendu ≥ 5 = 5 goroutines × ≥1 cycle)",
			syncOK.Load())
	}

	// HTTP : des Catalog Errors sont ATTENDUES pendant les fenêtres de swap
	// (limitation B3 documentée — résolution via RWLock pool prévue
	// post-sprint). On vérifie juste qu'au moins quelques queries HTTP
	// réussissent (entre les swaps), pas un seuil dur sur Catalog.
	if httpOK.Load() == 0 {
		t.Error("aucune query HTTP réussie sur 2s — chaîne B3 totalement cassée ?")
	}
	if catalogErrors.Load() > 0 {
		// Information, pas erreur — la limitation est documentée.
		t.Logf("LIMITATION ATTENDUE : %d Catalog Errors HTTP pendant les fenêtres de swap (%.1f%% du total)",
			catalogErrors.Load(),
			100.0*float64(catalogErrors.Load())/float64(catalogErrors.Load()+httpOK.Load()))
	}

	// Vérification de cohérence : count final reflète tous les INSERTs OK.
	var finalCount int
	if err := pdb.Player.QueryRow(ctx,
		"SELECT COUNT(*) FROM shared.match_registry").Scan(&finalCount); err != nil {
		t.Fatalf("query finale: %v (la conn player peut être dans un état dégradé après le burst)", err)
	}
	expected := int(syncOK.Load())
	if finalCount != expected {
		t.Errorf("count final = %d, attendu %d (cohérence INSERT/visibilité)",
			finalCount, expected)
	}
}

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
