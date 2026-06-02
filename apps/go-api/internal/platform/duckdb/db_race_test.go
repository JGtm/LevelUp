package duckdb

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestDB_AtomicSwap_NoRaceUnderConcurrentReads valide le fix H-C1 (revue P0
// 2026-06-02) : le handle sqlDB peut être remplacé (Store) pendant que des
// lecteurs l'utilisent (loadSQL via Query/SQLDb), sans data race. Avant la
// migration vers atomic.Pointer, ce scénario était une lecture/écriture
// concurrente non synchronisée d'un pointeur partagé → `go test -race` échouait.
//
// Lancer avec -race pour que le test soit significatif.
func TestDB_AtomicSwap_NoRaceUnderConcurrentReads(t *testing.T) {
	sqlDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open :memory: %v", err)
	}
	defer sqlDB.Close()
	db := newTestDB(sqlDB, ":memory:")

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Lecteurs concurrents : Load() via SQLDb() et QueryRow().
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_ = db.SQLDb()
					_ = db.QueryRow(context.Background(), "SELECT 1").Scan(new(int))
				}
			}
		}()
	}

	// Swaps concurrents du handle (re-Store du même pointeur : exerce le chemin
	// atomic Store en concurrence avec les Load des lecteurs).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			db.sqlDB.Store(sqlDB)
		}
		close(stop)
	}()

	wg.Wait()
}
