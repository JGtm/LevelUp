//go:build integration

package sharedprovider_test

import (
	"context"
	"testing"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestProvider_RecoversFromSyncPanic_integration (T6 du plan) vérifie que
// `defer w.Release()` ramène le provider en StateRO même si la goroutine
// sync panique.
func TestProvider_RecoversFromSyncPanic_integration(t *testing.T) {
	path := setupSharedDB(t)

	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("attendait une panic dans la goroutine sync")
			}
		}()
		w, err := p.AcquireWriter(ctx)
		if err != nil {
			t.Fatalf("AcquireWriter: %v", err)
		}
		defer w.Release()

		if _, err := w.DB().ExecContext(ctx,
			"CREATE TABLE IF NOT EXISTS recovery_test (val INTEGER)"); err != nil {
			t.Fatalf("CREATE TABLE: %v", err)
		}
		panic("simulated sync panic mid-batch")
	}()

	if got := p.State(); got != sharedprovider.StateRO {
		t.Errorf("state après panic recovery = %v, attendu RO", got)
	}

	db, release, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("Get post-panic: %v", err)
	}
	defer release()

	var v string
	if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&v); err != nil {
		t.Errorf("ping post-panic: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'recovery_test'").Scan(&count); err != nil {
		t.Errorf("verify table post-panic: %v", err)
	}
	if count != 1 {
		t.Errorf("attendu table recovery_test visible post-panic, obtenu count=%d", count)
	}
}
