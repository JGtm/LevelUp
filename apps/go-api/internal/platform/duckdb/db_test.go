//go:build integration

// Package duckdb_test — db_test.go : tests lifecycle connexions DuckDB.
//
// Sprint 47 T21 — tester open/close, ping, et l'isolation read-only.
// CGO requis : github.com/duckdb/duckdb-go/v2.
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -v
package duckdb_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	ddb "levelup/go-api/internal/platform/duckdb"
)

// TestOpenReadWrite_MemDB vérifie qu'on peut ouvrir, écrire et lire une DB in-memory.
func TestOpenReadWrite_MemDB(t *testing.T) {
	ctx := context.Background()
	db, err := ddb.OpenReadWrite(":memory:")
	if err != nil {
		t.Fatalf("OpenReadWrite(:memory:): %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(ctx, "CREATE TABLE test_rw (id INTEGER, val TEXT)"); err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}
	if _, err := db.Exec(ctx, "INSERT INTO test_rw VALUES (1, 'hello')"); err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	var val string
	if err := db.QueryRow(ctx, "SELECT val FROM test_rw WHERE id = 1").Scan(&val); err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	if val != "hello" {
		t.Errorf("attendu 'hello', obtenu %q", val)
	}
}

// TestOpenReadOnly_FileDB vérifie open read-only sur un fichier temporaire.
func TestOpenReadOnly_FileDB(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.duckdb")

	// Créer la DB avec une table
	rw, err := ddb.OpenReadWrite(dbPath)
	if err != nil {
		t.Fatalf("OpenReadWrite(%s): %v", dbPath, err)
	}
	if _, err := rw.Exec(ctx, "CREATE TABLE ro_test (n INTEGER)"); err != nil {
		t.Fatalf("CREATE: %v", err)
	}
	if _, err := rw.Exec(ctx, "INSERT INTO ro_test VALUES (42)"); err != nil {
		t.Fatalf("INSERT: %v", err)
	}
	rw.Close()

	// Ouvrir en read-only
	ro, err := ddb.OpenReadOnly(dbPath)
	if err != nil {
		t.Fatalf("OpenReadOnly(%s): %v", dbPath, err)
	}
	defer ro.Close()

	var n int
	if err := ro.QueryRow(ctx, "SELECT n FROM ro_test").Scan(&n); err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	if n != 42 {
		t.Errorf("attendu 42, obtenu %d", n)
	}

	// Écriture interdite sur read-only (doit retourner une erreur)
	_, errWrite := ro.Exec(ctx, "INSERT INTO ro_test VALUES (99)")
	if errWrite == nil {
		t.Error("INSERT sur DB read-only aurait dû échouer")
	}
}

// TestOpenReadWrite_ConcurrentReads vérifie la sécurité concurrente en lecture.
func TestOpenReadWrite_ConcurrentReads(t *testing.T) {
	ctx := context.Background()
	db, err := ddb.OpenReadWrite(":memory:")
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(ctx, "CREATE TABLE concurrent (id INTEGER)"); err != nil {
		t.Fatalf("CREATE: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := db.Exec(ctx, "INSERT INTO concurrent VALUES (?)", i); err != nil {
			t.Fatalf("INSERT %d: %v", i, err)
		}
	}

	var wg sync.WaitGroup
	errs := make([]error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var cnt int
			errs[idx] = db.QueryRow(ctx, "SELECT COUNT(*) FROM concurrent").Scan(&cnt)
		}(i)
	}
	wg.Wait()

	for i, e := range errs {
		if e != nil {
			t.Errorf("goroutine %d: %v", i, e)
		}
	}
}

// TestOpenReadWrite_FileNotFound vérifie qu'on peut créer une nouvelle DB sur disque.
func TestOpenReadWrite_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	newDBPath := filepath.Join(dir, "brand_new.duckdb")

	if _, err := os.Stat(newDBPath); !os.IsNotExist(err) {
		t.Fatalf("le fichier ne devrait pas exister avant le test")
	}

	db, err := ddb.OpenReadWrite(newDBPath)
	if err != nil {
		t.Fatalf("OpenReadWrite sur nouveau chemin: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(newDBPath); err != nil {
		t.Errorf("le fichier DuckDB devrait avoir été créé : %v", err)
	}
}
