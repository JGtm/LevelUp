package sync

// engine_reconcile_test.go — couvre reconcileInsertedAgainstRegistry (réconciliation
// post-Drain en mode async dégradé). CGO requis (DuckDB :memory:).

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/domain"

	_ "github.com/duckdb/duckdb-go/v2"
)

func newRegistryMemDB(t *testing.T, ids ...string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open :memory: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	for _, id := range ids {
		if _, err := db.Exec(`INSERT INTO match_registry VALUES (?)`, id); err != nil {
			t.Fatalf("insert %s: %v", id, err)
		}
	}
	return db
}

func TestReconcileInsertedAgainstRegistry_DropsAbsent(t *testing.T) {
	db := newRegistryMemDB(t, "m1", "m3") // m2 absent du registry
	res := &domain.SyncResult{InsertedMatchIDs: []string{"m1", "m2", "m3"}, MatchesInserted: 3}

	reconcileInsertedAgainstRegistry(context.Background(), db, res, "TestGT")

	if len(res.InsertedMatchIDs) != 2 || res.InsertedMatchIDs[0] != "m1" || res.InsertedMatchIDs[1] != "m3" {
		t.Errorf("InsertedMatchIDs = %v, want [m1 m3]", res.InsertedMatchIDs)
	}
	if res.MatchesInserted != 2 {
		t.Errorf("MatchesInserted = %d, want 2", res.MatchesInserted)
	}
	if res.MatchesSkipped != 1 {
		t.Errorf("MatchesSkipped = %d, want 1", res.MatchesSkipped)
	}
}

func TestReconcileInsertedAgainstRegistry_AllConfirmed_NoOp(t *testing.T) {
	db := newRegistryMemDB(t, "m1", "m2")
	res := &domain.SyncResult{InsertedMatchIDs: []string{"m1", "m2"}, MatchesInserted: 2}

	reconcileInsertedAgainstRegistry(context.Background(), db, res, "TestGT")

	if len(res.InsertedMatchIDs) != 2 || res.MatchesInserted != 2 || res.MatchesSkipped != 0 {
		t.Errorf("tous présents → aucun drop attendu, got %+v", res)
	}
}

func TestReconcileInsertedAgainstRegistry_NilDB_NoOp(t *testing.T) {
	res := &domain.SyncResult{InsertedMatchIDs: []string{"m1"}, MatchesInserted: 1}
	reconcileInsertedAgainstRegistry(context.Background(), nil, res, "GT")
	if len(res.InsertedMatchIDs) != 1 || res.MatchesInserted != 1 || res.MatchesSkipped != 0 {
		t.Errorf("sharedDB nil → no-op attendu, got %+v", res)
	}
}

// Bénéfice du doute : si la requête échoue (table match_registry absente), les
// matchs sont CONSERVÉS — une erreur transitoire ne doit pas priver un match
// réellement persisté de son post-sync.
func TestReconcileInsertedAgainstRegistry_QueryError_KeepsMatches(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:") // pas de table match_registry → la requête échoue
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	res := &domain.SyncResult{InsertedMatchIDs: []string{"m1", "m2"}, MatchesInserted: 2}

	reconcileInsertedAgainstRegistry(context.Background(), db, res, "GT")

	if len(res.InsertedMatchIDs) != 2 || res.MatchesInserted != 2 || res.MatchesSkipped != 0 {
		t.Errorf("erreur de requête → bénéfice du doute, aucun drop attendu, got %+v", res)
	}
}
