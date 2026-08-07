//go:build integration

// Package sync — art_upsert_patterns_test.go : tests TDD pour évaluer les
// patterns alternatifs à `INSERT ... ON CONFLICT DO UPDATE` afin de
// contourner le bug ART DELETE-side de DuckDB.
//
// Contexte : observation prod 2026-05-23 d'un FATAL Error récurrent
// "Invalid Input Error: Failed to delete all rows from index. Only deleted
// 0 out of N rows" lors d'UPSERTs concurrents sur tables avec PK VARCHAR.
// Le bug se manifeste à la fois sur shared.match_participants (PK composite)
// et sur player.player_match_enrichment (PK simple match_id).
//
// Le bug se manifeste car DuckDB implémente `ON CONFLICT DO UPDATE` en
// DELETE+INSERT en interne, et le DELETE consulte un index ART qui peut
// avoir des entries fantômes (entries pointant vers rows inexistantes).
//
// **Question à valider** : est-ce que `INSERT OR REPLACE` a le même bug ?
// Si oui : Option A invalide, on doit utiliser SELECT-then-INSERT-or-UPDATE
// (2 round-trips, plus lent mais sûr).
//
// Ce test essaie de reproduire le bug en :memory: avec 3 patterns :
//   A. ON CONFLICT DO UPDATE (le pattern actuel buggé)
//   B. INSERT OR REPLACE (l'alternative proposée)
//   C. SELECT-then-INSERT-or-UPDATE (le pattern garanti sans DELETE)
//
// Le bug ART n'est PAS facile à reproduire en :memory: (dépend de
// combinaisons spécifiques de données + timing de DuckDB). Si le test
// passe pour les 3 patterns, on ne peut pas conclure définitivement, mais
// si l'un échoue, on a une indication forte.

package sync

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

const upsertTestRows = 200
const upsertTestConcurrency = 50

// setupUpsertTestTable crée une table minimale similaire à
// match_participants avec PK VARCHAR composite.
func setupUpsertTestTable(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE t (
			pk1 VARCHAR,
			pk2 VARCHAR,
			val INTEGER,
			gamertag VARCHAR,
			score DOUBLE,
			created_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			PRIMARY KEY (pk1, pk2)
		);
	`); err != nil {
		t.Fatal(err)
	}
	return db
}

// TestUpsertPattern_OnConflict_ConcurrentSameKey : pattern actuel, censé
// déclencher le bug ART sur certaines combinaisons. En :memory: il ne se
// reproduit pas systématiquement — ce test sert de baseline.
func TestUpsertPattern_OnConflict_ConcurrentSameKey(t *testing.T) {
	db := setupUpsertTestTable(t)
	ctx := context.Background()

	// Pré-seed pour que la PK existe et que les UPSERTs concurrents fassent DELETE+INSERT.
	if _, err := db.Exec(`INSERT INTO t (pk1, pk2, val, gamertag, score) VALUES ('m1', 'x1', 0, 'gt0', 0.0)`); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, upsertTestConcurrency*upsertTestRows)
	for c := 0; c < upsertTestConcurrency; c++ {
		c := c
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < upsertTestRows; i++ {
				_, err := db.ExecContext(ctx, `
					INSERT INTO t (pk1, pk2, val, gamertag, score)
					VALUES (?, ?, ?, ?, ?)
					ON CONFLICT (pk1, pk2) DO UPDATE SET
						val      = COALESCE(EXCLUDED.val, t.val),
						gamertag = COALESCE(EXCLUDED.gamertag, t.gamertag),
						score    = COALESCE(EXCLUDED.score, t.score)
				`, "m1", "x1", c*1000+i, fmt.Sprintf("gt%d", c), float64(i))
				if err != nil {
					errs <- fmt.Errorf("c=%d i=%d: %w", c, i, err)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)

	var firstErr error
	errCount := 0
	for err := range errs {
		errCount++
		if firstErr == nil {
			firstErr = err
		}
	}
	if errCount > 0 {
		t.Logf("ON CONFLICT : %d errors / %d UPSERTs. First: %v",
			errCount, upsertTestConcurrency*upsertTestRows, firstErr)
	} else {
		t.Logf("ON CONFLICT : 0 errors / %d UPSERTs. Bug ART NON reproduit en :memory:.",
			upsertTestConcurrency*upsertTestRows)
	}

	// Final state : exactement 1 row.
	var cnt int
	db.QueryRow(`SELECT COUNT(*) FROM t`).Scan(&cnt)
	if cnt != 1 {
		t.Errorf("ON CONFLICT : count = %d, want 1", cnt)
	}
}

// TestUpsertPattern_InsertOrReplace_ConcurrentSameKey : alternative
// proposée. Si le bug se reproduit ici aussi, Option A est INVALIDE.
//
// Note : DuckDB documente INSERT OR REPLACE comme syntactic sugar pour
// `ON CONFLICT DO UPDATE SET col = EXCLUDED.col` pour toutes les colonnes.
// Donc en théorie, MÊME bug. À confirmer.
func TestUpsertPattern_InsertOrReplace_ConcurrentSameKey(t *testing.T) {
	db := setupUpsertTestTable(t)
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO t (pk1, pk2, val, gamertag, score) VALUES ('m1', 'x1', 0, 'gt0', 0.0)`); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, upsertTestConcurrency*upsertTestRows)
	for c := 0; c < upsertTestConcurrency; c++ {
		c := c
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < upsertTestRows; i++ {
				_, err := db.ExecContext(ctx, `
					INSERT OR REPLACE INTO t (pk1, pk2, val, gamertag, score)
					VALUES (?, ?, ?, ?, ?)
				`, "m1", "x1", c*1000+i, fmt.Sprintf("gt%d", c), float64(i))
				if err != nil {
					errs <- fmt.Errorf("c=%d i=%d: %w", c, i, err)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)

	var firstErr error
	errCount := 0
	for err := range errs {
		errCount++
		if firstErr == nil {
			firstErr = err
		}
	}
	if errCount > 0 {
		t.Logf("INSERT OR REPLACE : %d errors / %d UPSERTs. First: %v",
			errCount, upsertTestConcurrency*upsertTestRows, firstErr)
		// Marquer comme failure si on a au moins 1 erreur — sinon "0 errors"
		// pourrait masquer un cas où INSERT OR REPLACE plante aussi.
		// On NE marque PAS failure ici car on veut juste OBSERVER.
	} else {
		t.Logf("INSERT OR REPLACE : 0 errors / %d UPSERTs. Pattern viable EN ISOLATION.",
			upsertTestConcurrency*upsertTestRows)
	}

	var cnt int
	db.QueryRow(`SELECT COUNT(*) FROM t`).Scan(&cnt)
	if cnt != 1 {
		t.Errorf("INSERT OR REPLACE : count = %d, want 1", cnt)
	}
}

// TestUpsertPattern_SelectThenUpdate_ConcurrentSameKey : pattern garanti
// sans DELETE de l'index. Plus lent (2 round-trips) mais sûr.
//
// Pattern :
//  1. SELECT 1 FROM t WHERE pk1=? AND pk2=?  -- check existence (no DELETE)
//  2. If exists  : UPDATE ... WHERE pk1=? AND pk2=?  -- in-place modify
//  3. If absent  : INSERT ...
//
// L'UPDATE in-place ne touche pas l'index ART (la PK ne change pas).
// L'INSERT ajoute une entry à l'index (jamais delete).
func TestUpsertPattern_SelectThenUpdate_ConcurrentSameKey(t *testing.T) {
	db := setupUpsertTestTable(t)
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO t (pk1, pk2, val, gamertag, score) VALUES ('m1', 'x1', 0, 'gt0', 0.0)`); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, upsertTestConcurrency*upsertTestRows)
	for c := 0; c < upsertTestConcurrency; c++ {
		c := c
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < upsertTestRows; i++ {
				// Check existence puis UPDATE ou INSERT.
				var existing int
				err := db.QueryRowContext(ctx,
					`SELECT 1 FROM t WHERE pk1 = ? AND pk2 = ?`, "m1", "x1").Scan(&existing)
				if err == sql.ErrNoRows {
					_, err = db.ExecContext(ctx, `
						INSERT INTO t (pk1, pk2, val, gamertag, score)
						VALUES (?, ?, ?, ?, ?)`,
						"m1", "x1", c*1000+i, fmt.Sprintf("gt%d", c), float64(i))
				} else if err == nil {
					_, err = db.ExecContext(ctx, `
						UPDATE t SET val = ?, gamertag = ?, score = ?
						WHERE pk1 = ? AND pk2 = ?`,
						c*1000+i, fmt.Sprintf("gt%d", c), float64(i), "m1", "x1")
				}
				if err != nil {
					errs <- fmt.Errorf("c=%d i=%d: %w", c, i, err)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)

	var firstErr error
	errCount := 0
	for err := range errs {
		errCount++
		if firstErr == nil {
			firstErr = err
		}
	}
	if errCount > 0 {
		t.Logf("SELECT-then-UPDATE : %d errors. First: %v", errCount, firstErr)
	} else {
		t.Logf("SELECT-then-UPDATE : 0 errors / %d UPSERTs. Pattern viable.",
			upsertTestConcurrency*upsertTestRows)
	}

	var cnt int
	db.QueryRow(`SELECT COUNT(*) FROM t`).Scan(&cnt)
	if cnt != 1 {
		t.Errorf("SELECT-then-UPDATE : count = %d, want 1", cnt)
	}
}
