//go:build integration

// Tests pour l'adaptateur FromInMemoryDB (commit 8b).
//
// Valide que FromInMemoryDB satisfait l'interface Provider et permet aux
// tests du sync engine (futur commit 8l) d'utiliser un Provider sans
// fichier physique.
package sharedprovider_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// openInMemoryDB ouvre un *sql.DB DuckDB anonyme in-memory pour les tests
// de l'adaptateur.
func openInMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("sql.Open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestFromInMemoryDB_GetReturnsSameDB valide que Get retourne toujours
// exactement le *sql.DB fourni, sans wrapper.
func TestFromInMemoryDB_GetReturnsSameDB(t *testing.T) {
	db := openInMemoryDB(t)
	p := sharedprovider.FromInMemoryDB(db, "memory://test")

	if got := p.Path(); got != "memory://test" {
		t.Errorf("Path = %q, attendu %q", got, "memory://test")
	}
	if got := p.State(); got != sharedprovider.StateRO {
		t.Errorf("State = %v, attendu StateRO", got)
	}

	ctx := context.Background()
	got, release, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer release()

	if got != db {
		t.Errorf("Get retourne *sql.DB différent du db fourni : %p vs %p", got, db)
	}

	// Sanity ping pour confirmer le db est utilisable.
	var v string
	if err := got.QueryRowContext(ctx, "SELECT version()").Scan(&v); err != nil {
		t.Errorf("ping: %v", err)
	}
}

// TestFromInMemoryDB_AcquireWriter valide que AcquireWriter retourne un
// WriterHandle sur le même db, et que Release ne ferme PAS le db.
func TestFromInMemoryDB_AcquireWriter(t *testing.T) {
	db := openInMemoryDB(t)
	p := sharedprovider.FromInMemoryDB(db, "memory://test")

	ctx := context.Background()
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}

	if w.DB() != db {
		t.Errorf("WriterHandle.DB() différent du db fourni : %p vs %p", w.DB(), db)
	}

	// On peut écrire via le writer (le db est in-memory, pas read-only).
	if _, err := w.DB().ExecContext(ctx,
		"CREATE TABLE IF NOT EXISTS imdb_test (val INT)"); err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}
	if _, err := w.DB().ExecContext(ctx, "INSERT INTO imdb_test VALUES (1)"); err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	w.Release()
	// Idempotence
	w.Release()

	// Après Release, le db est toujours utilisable (pas fermé).
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM imdb_test").Scan(&count); err != nil {
		t.Fatalf("verify post-release: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, attendu 1", count)
	}
}

// TestFromInMemoryDB_CloseDoesNotCloseUnderlyingDB valide que Close marque
// le provider comme fermé mais ne ferme PAS le *sql.DB sous-jacent — le
// caller garde la responsabilité.
func TestFromInMemoryDB_CloseDoesNotCloseUnderlyingDB(t *testing.T) {
	db := openInMemoryDB(t)
	p := sharedprovider.FromInMemoryDB(db, "memory://test")

	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got := p.State(); got != sharedprovider.StateClosed {
		t.Errorf("State après Close = %v, attendu StateClosed", got)
	}

	// Get après Close retourne ErrProviderClosed.
	_, release, err := p.Get(context.Background())
	if !errors.Is(err, sharedprovider.ErrProviderClosed) {
		t.Errorf("Get après Close = %v, attendu ErrProviderClosed", err)
	}
	if release != nil {
		t.Error("release devrait être nil quand err != nil")
	}

	// AcquireWriter après Close retourne aussi ErrProviderClosed.
	_, err = p.AcquireWriter(context.Background())
	if !errors.Is(err, sharedprovider.ErrProviderClosed) {
		t.Errorf("AcquireWriter après Close = %v, attendu ErrProviderClosed", err)
	}

	// Le *sql.DB sous-jacent reste utilisable directement (le caller
	// l'a fourni, il garde la propriété).
	var v string
	if err := db.QueryRowContext(context.Background(), "SELECT version()").Scan(&v); err != nil {
		t.Errorf("db sous-jacent inutilisable après Close du Provider : %v", err)
	}

	// Idempotence : Close à nouveau.
	if err := p.Close(); err != nil {
		t.Errorf("Close #2 doit être no-op : %v", err)
	}
}

// TestFromInMemoryDB_SubscribeNoEvents vérifie que Subscribe est fonctionnel
// (pas de panic, unsubscribe idempotent) mais qu'aucun event n'est jamais
// émis — l'adaptateur n'a pas de transitions d'état observables.
func TestFromInMemoryDB_SubscribeNoEvents(t *testing.T) {
	db := openInMemoryDB(t)
	p := sharedprovider.FromInMemoryDB(db, "memory://test")
	defer func() { _ = p.Close() }()

	called := false
	unsubscribe := p.Subscribe(func(_ sharedprovider.SwapEvent) {
		called = true
	})

	// Faire un cycle AcquireWriter/Release — devrait être no-op et NE PAS
	// déclencher de Subscribers (l'adaptateur n'émet jamais).
	ctx := context.Background()
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	w.Release()

	if called {
		t.Error("Subscriber appelé alors qu'aucun event n'est attendu pour FromInMemoryDB")
	}

	// Unsubscribe + idempotence.
	unsubscribe()
	unsubscribe()
}
