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
// IMPORTANT : le release() du Get DOIT être appelé avant AcquireWriter,
// sinon le drain de la phase 2 attend ce reader et le swap timeout.
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
	roDB, releaseRO, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("Get RO: %v", err)
	}
	var version string
	if err := roDB.QueryRowContext(ctx, "SELECT version()").Scan(&version); err != nil {
		releaseRO()
		t.Fatalf("RO ping: %v", err)
	}
	// CRITIQUE : release explicite AVANT AcquireWriter pour ne pas bloquer
	// la phase de drain.
	releaseRO()

	// AcquireWriter : transition vers RW.
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	if got := p.State(); got != sharedprovider.StateRW {
		t.Errorf("state pendant writer = %v, attendu RW", got)
	}

	if _, err := w.DB().ExecContext(ctx,
		"CREATE TABLE IF NOT EXISTS swap_test (val INTEGER)"); err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}
	if _, err := w.DB().ExecContext(ctx,
		"INSERT INTO swap_test VALUES (42)"); err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	w.Release()

	if got := p.State(); got != sharedprovider.StateRO {
		t.Errorf("state après Release = %v, attendu RO", got)
	}

	// Idempotence du Release.
	w.Release()
	if got := p.State(); got != sharedprovider.StateRO {
		t.Errorf("state après double Release = %v, attendu RO", got)
	}

	// Get post-Release : la conn RO doit voir l'INSERT (visibilité cross-handle).
	roDB2, releaseRO2, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("Get post-Release: %v", err)
	}
	defer releaseRO2()

	var count int
	if err := roDB2.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM swap_test WHERE val = 42").Scan(&count); err != nil {
		t.Fatalf("verify INSERT: %v", err)
	}
	if count != 1 {
		t.Errorf("attendu 1 ligne insérée et visible en RO, obtenu %d", count)
	}
}
