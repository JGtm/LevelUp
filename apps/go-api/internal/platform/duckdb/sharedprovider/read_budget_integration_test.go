//go:build integration

package sharedprovider_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestProvider_SwapWaitBudget_FailFast_integration vérifie qu'un Get muni d'un
// budget d'attente court (WithSwapWaitBudget) échoue vite en ErrSwapTimeout pendant
// un swap RW long, SANS attendre le readyTimeout global du provider. C'est le
// mécanisme fail-fast des lectures user-facing (503 Retry-After) : le sync tient le
// writer plusieurs secondes, mais une page n'attend que le budget court.
func TestProvider_SwapWaitBudget_FailFast_integration(t *testing.T) {
	path := setupSharedDB(t)

	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	// readyTimeout LONG (5s) : sans budget, un Get pendant le swap attendrait ~5s.
	sharedprovider.SetReadyTimeoutForTest(p, 5*time.Second)

	ctx := context.Background()

	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	// Le writer (sync simulé) est tenu 1s — bien plus que le budget court.
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		time.Sleep(1 * time.Second)
		w.Release()
	}()

	// Budget user-facing court : 150ms. Le Get doit échouer vers ~150ms (bien avant
	// le release à 1s et bien avant le readyTimeout de 5s).
	budgetCtx := sharedprovider.WithSwapWaitBudget(ctx, 150*time.Millisecond)

	start := time.Now()
	_, release, err := p.Get(budgetCtx)
	elapsed := time.Since(start)

	if err == nil {
		if release != nil {
			release()
		}
		t.Fatal("Get avec budget court pendant swap aurait dû échouer, obtenu nil")
	}
	if !errors.Is(err, sharedprovider.ErrSwapTimeout) {
		t.Errorf("Get avec budget court = %v, attendu ErrSwapTimeout", err)
	}
	// Fail-fast : bien en-dessous du readyTimeout 5s (marge généreuse pour la CI).
	if elapsed > 1*time.Second {
		t.Errorf("Get a pris %v, attendu fail-fast < 1s (budget=150ms, readyTimeout=5s)", elapsed)
	}

	<-writerDone

	// Sans budget, un Get après le retour RO réussit normalement.
	db, release, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("Get après retour RO: %v", err)
	}
	release()
	if db == nil {
		t.Fatal("Get après retour RO: db nil")
	}
}
