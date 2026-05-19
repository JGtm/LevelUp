//go:build integration

package sharedprovider_test

import (
	"context"
	"testing"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestProvider_RecoversFromSyncPanic_integration (T6 du plan) vérifie que
// le pattern `defer w.Release()` côté caller suffit à ramener le provider
// en StateRO même si la goroutine sync panique au milieu d'une écriture.
//
// Sécurise les chemins d'écriture du sync engine où une migration SQL
// malformée, un INSERT contraint ou un bug DuckDB pourrait panic — sans ce
// contrat, le provider resterait coincé en StateRW et tous les Get HTTP
// se mettraient à timeout en chaîne.
func TestProvider_RecoversFromSyncPanic_integration(t *testing.T) {
	path := setupSharedDB(t)

	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()

	// Simule un sync qui panic après avoir acquis le writer.
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
		defer w.Release() // <-- contrat critique : doit s'exécuter sur panic

		// Travail partiel — création de table OK, puis panic avant l'INSERT
		// (situation typique : data corruption détectée mid-batch).
		if _, err := w.DB().ExecContext(ctx,
			"CREATE TABLE IF NOT EXISTS recovery_test (val INTEGER)"); err != nil {
			t.Fatalf("CREATE TABLE: %v", err)
		}
		panic("simulated sync panic mid-batch")
	}()

	// Le defer Release a tourné → state doit être revenu en RO.
	if got := p.State(); got != sharedprovider.StateRO {
		t.Errorf("state après panic recovery = %v, attendu RO", got)
	}

	// Get doit fonctionner immédiatement.
	db, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("Get post-panic: %v", err)
	}
	var v string
	if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&v); err != nil {
		t.Errorf("ping post-panic: %v", err)
	}

	// La table créée avant la panic doit être visible (commit DDL implicite
	// dans DuckDB) — preuve que le release a bien fermé proprement le RW.
	var count int
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'recovery_test'").Scan(&count); err != nil {
		t.Errorf("verify table post-panic: %v", err)
	}
	if count != 1 {
		t.Errorf("attendu table recovery_test visible post-panic, obtenu count=%d", count)
	}
}
