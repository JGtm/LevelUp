//go:build integration

package sharedprovider_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestProvider_ReopenROFailureDegradesGracefully_integration (T11 du plan)
// valide le pire scénario opérationnel : la réouverture RO post-sync
// échoue (file lock OS orphelin, FS plein, perm denied…) → le provider
// doit passer proprement en StateError, retourner ErrSwapFailed aux Get,
// puis tenter une récupération via retry loop borné.
//
// Aligné sur le pattern retry metadata main.go:222-236 : 5 tentatives
// avec backoff exponentiel. Si recovery réussit avant les 5 tentatives,
// state repasse Error → RO automatiquement.
func TestProvider_ReopenROFailureDegradesGracefully_integration(t *testing.T) {
	path := setupSharedDB(t)

	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	// Backoff initial réduit pour que le retry loop ne prenne pas 1s+.
	sharedprovider.SetRetryBaseBackoffForTest(p, 50*time.Millisecond)

	// Armer le hook : le PROCHAIN reopen RO échouera (consommé en 1 appel).
	sharedprovider.SetFailNextReopenForTest(p, true)

	ctx := context.Background()

	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	w.Release()

	// Immédiatement après Release : le reopen a échoué (hook), state = Error.
	if got := p.State(); got != sharedprovider.StateError {
		t.Errorf("state après reopen failure = %v, attendu StateError", got)
	}

	// Get doit retourner ErrSwapFailed (et non un timeout ou une autre erreur).
	_, err = p.Get(ctx)
	if !errors.Is(err, sharedprovider.ErrSwapFailed) {
		t.Errorf("Get en StateError = %v, attendu ErrSwapFailed", err)
	}

	// Le retry loop async démarre avec backoff=50ms. Le hook a été consommé
	// au 1er essai, donc le 2e essai (dans le retry loop, ~50ms après) doit
	// réussir et state passe Error → RO.
	deadline := time.Now().Add(3 * time.Second)
	for p.State() == sharedprovider.StateError && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}

	if got := p.State(); got != sharedprovider.StateRO {
		t.Fatalf("après retry loop, state = %v, attendu StateRO (recovery via retry)", got)
	}

	// Get doit re-fonctionner.
	db, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("Get post-recovery: %v", err)
	}
	var v string
	if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&v); err != nil {
		t.Errorf("ping post-recovery: %v", err)
	}
}
