//go:build integration

// player_perf_chain_test.go — déplacé depuis internal/migration (Phase 1.5 b15).
// Vérifie que player_match_enrichment_performance_chain_v1 (title-owned) ajoute la
// colonne performance_chain, sur DB neuve comme legacy. Câble le provider StepsFor +
// résout le step via StepsFor(TargetPlayer) (au lieu du registre global All()).
package migrations

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func mustHaveColumn(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()
	exists, err := migration.ColumnExists(db, table, column)
	if err != nil {
		t.Fatalf("ColumnExists(%q.%q): %v", table, column, err)
	}
	if !exists {
		t.Errorf("colonne %q absente sur la table %q", column, table)
	}
}

// TestRunForDB_Player_PerformanceChainColumn : performance_chain présent sur
// player_match_enrichment après RunForDB(TargetPlayer) avec provider câblé.
func TestRunForDB_Player_PerformanceChainColumn(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(Player): %v", err)
	}

	mustHaveColumn(t, db, "player_match_enrichment", "performance_chain")
}

// TestRunForDB_Player_PerformanceChain_OnLegacyDB : DB legacy sans la colonne →
// le step ciblé (résolu via StepsFor) l'ajoute, idempotent.
func TestRunForDB_Player_PerformanceChain_OnLegacyDB(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE player_match_enrichment (
			match_id          VARCHAR PRIMARY KEY,
			performance_score DOUBLE,
			session_id        VARCHAR
		);
	`); err != nil {
		t.Fatalf("seed legacy: %v", err)
	}
	if exists, err := migration.ColumnExists(db, "player_match_enrichment", "performance_chain"); err != nil {
		t.Fatalf("ColumnExists pré-migration: %v", err)
	} else if exists {
		t.Fatal("seed legacy : performance_chain ne devrait pas exister avant migration")
	}

	var mig *migration.Migration
	steps := StepsFor(migration.TargetPlayer)
	for i := range steps {
		if steps[i].Name == "player_match_enrichment_performance_chain_v1" {
			mig = &steps[i]
			break
		}
	}
	if mig == nil {
		t.Fatal("migration player_match_enrichment_performance_chain_v1 introuvable dans StepsFor(TargetPlayer)")
	}

	if err := mig.ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema performance_chain: %v", err)
	}
	mustHaveColumn(t, db, "player_match_enrichment", "performance_chain")

	if err := mig.ApplySchema(db); err != nil {
		t.Errorf("ApplySchema (2e passe, colonne déjà existante) doit être idempotent: %v", err)
	}
}
