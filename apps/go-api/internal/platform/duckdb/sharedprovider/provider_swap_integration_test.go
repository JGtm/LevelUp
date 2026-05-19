//go:build integration

package sharedprovider_test

import (
	"context"
	"testing"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestProvider_SwapROToRWToRO_integration (T2 du plan) valide l'invariant
// central du cycle de vie : Get → AcquireWriter → INSERT → Release → Get
// passe par les transitions RO → Draining → RW → Reopening → RO.
//
// Les transitions intermédiaires (Draining, Reopening) ne sont pas
// observables depuis l'extérieur (très brèves). On valide leur effet via
// les compteurs swapTotal et le state final.
func TestProvider_SwapROToRWToRO_integration(t *testing.T) {
	path := setupSharedDB(t)

	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()

	if got := p.State(); got != sharedprovider.StateRO {
		t.Fatalf("state initial = %v, attendu RO", got)
	}

	// Get en RO : doit fonctionner immédiatement.
	roDB, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("Get RO: %v", err)
	}
	var version string
	if err := roDB.QueryRowContext(ctx, "SELECT version()").Scan(&version); err != nil {
		t.Fatalf("RO ping: %v", err)
	}

	// AcquireWriter : transition vers RW.
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	if got := p.State(); got != sharedprovider.StateRW {
		t.Errorf("state pendant writer = %v, attendu RW", got)
	}

	// Créer une table de test + INSERT pour valider l'écriture concrète.
	// On utilise une table custom plutôt que match_registry (schéma à
	// contraintes lourdes) pour rester simple.
	if _, err := w.DB().ExecContext(ctx,
		"CREATE TABLE IF NOT EXISTS swap_test (val INTEGER)"); err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}
	if _, err := w.DB().ExecContext(ctx,
		"INSERT INTO swap_test VALUES (42)"); err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	// Release : transition vers RO.
	w.Release()

	if got := p.State(); got != sharedprovider.StateRO {
		t.Errorf("state après Release = %v, attendu RO", got)
	}

	// Idempotence : un second Release doit être no-op (pas de panic, pas de
	// double-decrement de métrique).
	w.Release()
	if got := p.State(); got != sharedprovider.StateRO {
		t.Errorf("state après double Release = %v, attendu RO", got)
	}

	// Get post-Release : la conn RO doit voir l'INSERT (visibilité cross-handle).
	roDB2, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("Get post-Release: %v", err)
	}
	var count int
	if err := roDB2.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM swap_test WHERE val = 42").Scan(&count); err != nil {
		t.Fatalf("verify INSERT: %v", err)
	}
	if count != 1 {
		t.Errorf("attendu 1 ligne insérée et visible en RO, obtenu %d", count)
	}
}
