//go:build integration

package sharedprovider_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	syncpkg "levelup/go-api/internal/sync"
)

// setupSharedDB crée un fichier shared_matches_v2.duckdb avec le schéma
// minimal, ferme la conn de bootstrap, et retourne le path.
func setupSharedDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "shared.duckdb")

	bootstrap, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("bootstrap OpenReadWrite: %v", err)
	}
	if err := syncpkg.EnsureSharedSchema(t.Context(), bootstrap.SQLDb()); err != nil {
		_ = bootstrap.Close()
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	if err := bootstrap.Close(); err != nil {
		t.Fatalf("bootstrap Close: %v", err)
	}
	return path
}

// TestProvider_GetInRO_ReturnsSameUnderlyingHandle vérifie l'invariant
// principal du steady state RO : Get() retourne toujours le même *sql.DB
// sous-jacent (le provider owne UN seul handle DuckDB tant qu'aucun swap
// RW n'a lieu).
func TestProvider_GetInRO_ReturnsSameUnderlyingHandle(t *testing.T) {
	path := setupSharedDB(t)

	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	if got := p.State(); got != sharedprovider.StateRO {
		t.Fatalf("State après New = %v, attendu StateRO", got)
	}

	ctx := context.Background()
	db1, release1, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("Get #1: %v", err)
	}
	defer release1()

	db2, release2, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("Get #2: %v", err)
	}
	defer release2()

	if db1 != db2 {
		t.Errorf("Get retourne 2 handles différents en steady state RO : %p vs %p", db1, db2)
	}

	var v string
	if err := db1.QueryRowContext(ctx, "SELECT version()").Scan(&v); err != nil {
		t.Errorf("ping RO: %v", err)
	}
}

// TestProvider_GetAfterClose_ReturnsProviderClosed vérifie le contrat
// d'erreur : un Get sur un Provider fermé retourne ErrProviderClosed.
func TestProvider_GetAfterClose_ReturnsProviderClosed(t *testing.T) {
	path := setupSharedDB(t)

	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got := p.State(); got != sharedprovider.StateClosed {
		t.Fatalf("State après Close = %v, attendu StateClosed", got)
	}

	_, release, err := p.Get(context.Background())
	if !errors.Is(err, sharedprovider.ErrProviderClosed) {
		t.Errorf("Get après Close = %v, attendu ErrProviderClosed", err)
	}
	if release != nil {
		t.Error("release devrait être nil quand err != nil")
	}
}

// TestProvider_CloseIsIdempotent vérifie qu'un double Close ne panique pas.
func TestProvider_CloseIsIdempotent(t *testing.T) {
	path := setupSharedDB(t)

	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := p.Close(); err != nil {
		t.Fatalf("Close #1: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Errorf("Close #2 doit être no-op, obtenu : %v", err)
	}
}

// TestManager_ForSamePath_ReturnsSameProvider vérifie la déduplication par
// chemin : deux appels For(path) renvoient le même Provider.
func TestManager_ForSamePath_ReturnsSameProvider(t *testing.T) {
	path := setupSharedDB(t)

	mgr := sharedprovider.NewManager()
	defer func() { _ = mgr.Close() }()

	p1, err := mgr.For(path)
	if err != nil {
		t.Fatalf("For #1: %v", err)
	}
	p2, err := mgr.For(path)
	if err != nil {
		t.Fatalf("For #2: %v", err)
	}
	if p1 != p2 {
		t.Errorf("Manager.For sur même path retourne 2 Provider différents")
	}
}
