//go:build integration

// Test stress concurrent Phase 3 plan stabilisation 2026-05-22 :
// 10 goroutines mutant media_files (UPDATE d'une colonne NON indexée) + 1 sync
// (INSERT match_participants) en parallèle, attendu 0 erreur SQL.
//
// La charge utile était media_files.liked jusqu'au 2026-08-04 ; cette colonne a
// été droppée (le like est par liker, en append-only). Le test porte désormais
// sur `status`, qui est la vraie colonne mutée de media_files (soft-delete) —
// ce qu'il mesure (sérialisation du pool sous writes concurrents) est inchangé.
//
// Garantit la robustesse du pool DuckDB sous charge concurrente. Si quelqu'un
// retire le `sync.Mutex` interne du `*DB` ou casse l'isolation MVCC d'un
// write, ce test détecte l'erreur immédiatement.
//
// Pattern : MVCC DuckDB sérialise les writes via single writer commit lock
// au niveau DB. Pour des micro-INSERTs/UPDATEs sur une seule conn `*sql.DB`,
// la sérialisation est ms-scale (cf. handoff §5 — "imperceptible pour
// shared_social writes").
package duckdb

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestPool_StressConcurrentLikesAndSync : 10 goroutines UPDATE media_files + 1
// goroutine INSERT match_participants, en parallèle. Attendu : 0 erreur SQL.
//
// MaxOpenConns=1 reproduit le mode prod OpenReadWrite (cf. db.go:121) qui
// sérialise les writes via le pool sql.DB. Sans ce paramètre, DuckDB
// renverrait "TransactionContext Error: Conflict on update" sur des goroutines
// tapant la même ligne simultanément (MVCC).
func TestPool_StressConcurrentLikesAndSync(t *testing.T) {
	socialDB := openMemDBForProbe(t)
	socialDB.SetMaxOpenConns(1)
	sharedDB := openMemDBForProbe(t)
	sharedDB.SetMaxOpenConns(1)
	ctx := context.Background()

	// Seed media_files (10 lignes status NULL par défaut).
	if _, err := socialDB.Exec(`
		CREATE TABLE media_files (
			id VARCHAR PRIMARY KEY,
			status VARCHAR,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if _, err := socialDB.Exec(`INSERT INTO media_files (id) VALUES (?)`,
			fmt.Sprintf("media-%d", i)); err != nil {
			t.Fatal(err)
		}
	}

	// Seed match_participants (vide — les inserts viennent du goroutine sync).
	if _, err := sharedDB.Exec(`
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			kills INTEGER,
			PRIMARY KEY (match_id, xuid)
		)
	`); err != nil {
		t.Fatal(err)
	}

	var (
		errCount  atomic.Int64
		likeOps   atomic.Int64
		insertOps atomic.Int64
		wg        sync.WaitGroup
	)
	const likeIterations = 50

	// 10 goroutines POST like.
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func(gID int) {
			defer wg.Done()
			for iter := 0; iter < likeIterations; iter++ {
				mediaID := fmt.Sprintf("media-%d", (gID+iter)%10)
				status := "active"
				if iter%2 == 0 {
					status = "deleted"
				}
				if _, err := socialDB.ExecContext(ctx,
					`UPDATE media_files SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
					status, mediaID,
				); err != nil {
					errCount.Add(1)
					t.Logf("like goroutine %d iter %d: %v", gID, iter, err)
					return
				}
				likeOps.Add(1)
			}
		}(g)
	}

	// 1 goroutine sync (INSERT match_participants).
	wg.Add(1)
	go func() {
		defer wg.Done()
		const matches = 20
		const playersPerMatch = 10
		for m := 0; m < matches; m++ {
			for p := 0; p < playersPerMatch; p++ {
				if _, err := sharedDB.ExecContext(ctx,
					`INSERT INTO match_participants (match_id, xuid, kills) VALUES (?, ?, ?)`,
					fmt.Sprintf("match-%d", m),
					fmt.Sprintf("xuid-%d-%d", m, p),
					p,
				); err != nil {
					errCount.Add(1)
					t.Logf("sync goroutine match %d player %d: %v", m, p, err)
					return
				}
				insertOps.Add(1)
			}
		}
	}()

	wg.Wait()

	// Assertions.
	if errCount.Load() > 0 {
		t.Errorf("expected 0 SQL errors under concurrent load, got %d", errCount.Load())
	}
	if got := likeOps.Load(); got != int64(10*likeIterations) {
		t.Errorf("like ops : expected %d, got %d", 10*likeIterations, got)
	}
	if got := insertOps.Load(); got != 200 {
		t.Errorf("insert ops : expected 200 (20×10), got %d", got)
	}

	// Verify final state : 10 media_files (count inchangé), 200 match_participants.
	var mediaCount, mpCount int
	if err := socialDB.QueryRow(`SELECT COUNT(*) FROM media_files`).Scan(&mediaCount); err != nil {
		t.Fatal(err)
	}
	if mediaCount != 10 {
		t.Errorf("media_files count = %d, expected 10 (les UPDATEs ne créent pas de lignes)", mediaCount)
	}
	if err := sharedDB.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&mpCount); err != nil {
		t.Fatal(err)
	}
	if mpCount != 200 {
		t.Errorf("match_participants count = %d, expected 200", mpCount)
	}

	t.Logf("stress OK : 10 goroutines × %d UPDATEs + 1 sync × 200 INSERTs = %d ops sans erreur",
		likeIterations, likeOps.Load()+insertOps.Load())
}

// TestPool_StressConcurrentSameDB : variante avec un seul *sql.DB partagé
// (case réel : media_files + sync share shared_social.duckdb). Verify que
// le single writer commit lock de DuckDB sérialise correctement.
func TestPool_StressConcurrentSameDB(t *testing.T) {
	db := openMemDBForProbe(t)
	db.SetMaxOpenConns(1)
	ctx := context.Background()

	if _, err := db.Exec(`
		CREATE TABLE counter_a (k VARCHAR PRIMARY KEY, n INTEGER DEFAULT 0);
		CREATE TABLE counter_b (k VARCHAR PRIMARY KEY, n INTEGER DEFAULT 0);
		INSERT INTO counter_a VALUES ('k1', 0);
		INSERT INTO counter_b VALUES ('k1', 0);
	`); err != nil {
		t.Fatal(err)
	}

	var (
		errCount atomic.Int64
		wg       sync.WaitGroup
	)
	const goroutines = 10
	const iterations = 100

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		// 5 goroutines incrémentent counter_a, 5 incrémentent counter_b.
		table := "counter_a"
		if g%2 == 1 {
			table = "counter_b"
		}
		go func(tbl string) {
			defer wg.Done()
			for iter := 0; iter < iterations; iter++ {
				if _, err := db.ExecContext(ctx,
					"UPDATE "+tbl+" SET n = n + 1 WHERE k = 'k1'",
				); err != nil {
					errCount.Add(1)
					return
				}
			}
		}(table)
	}
	wg.Wait()

	if errCount.Load() > 0 {
		t.Errorf("expected 0 errors, got %d", errCount.Load())
	}

	// Vérifier les compteurs finals (5 goroutines × 100 iterations × 2 tables).
	var a, b int
	_ = db.QueryRow(`SELECT n FROM counter_a WHERE k = 'k1'`).Scan(&a)
	_ = db.QueryRow(`SELECT n FROM counter_b WHERE k = 'k1'`).Scan(&b)
	if a != 5*iterations {
		t.Errorf("counter_a = %d, expected %d (5 goroutines × %d iter)", a, 5*iterations, iterations)
	}
	if b != 5*iterations {
		t.Errorf("counter_b = %d, expected %d", b, 5*iterations)
	}
}
