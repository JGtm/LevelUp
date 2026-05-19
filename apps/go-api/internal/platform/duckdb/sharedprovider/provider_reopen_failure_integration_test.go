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
// valide le pire scénario opérationnel : la réouverture RO post-sync échoue
// → state Error → ErrSwapFailed → retry loop async recovery.
func TestProvider_ReopenROFailureDegradesGracefully_integration(t *testing.T) {
	path := setupSharedDB(t)

	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	sharedprovider.SetRetryBaseBackoffForTest(p, 50*time.Millisecond)
	sharedprovider.SetFailNextReopenForTest(p, true)

	ctx := context.Background()

	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	w.Release()

	if got := p.State(); got != sharedprovider.StateError {
		t.Errorf("state après reopen failure = %v, attendu StateError", got)
	}

	_, release, err := p.Get(ctx)
	if !errors.Is(err, sharedprovider.ErrSwapFailed) {
		t.Errorf("Get en StateError = %v, attendu ErrSwapFailed", err)
	}
	if release != nil {
		t.Error("release devrait être nil quand err != nil")
	}

	deadline := time.Now().Add(3 * time.Second)
	for p.State() == sharedprovider.StateError && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}

	if got := p.State(); got != sharedprovider.StateRO {
		t.Fatalf("après retry loop, state = %v, attendu StateRO (recovery via retry)", got)
	}

	db, release2, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("Get post-recovery: %v", err)
	}
	defer release2()

	var v string
	if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&v); err != nil {
		t.Errorf("ping post-recovery: %v", err)
	}
}
