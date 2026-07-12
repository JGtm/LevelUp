//go:build integration

package migration

// squash_dm5_test.go — DM-5 (équivalence ledger d'un squash, chantier N4). Prouve que
// le runner court-circuite le DDL d'une baseline squashée quand la SENTINELLE (dernier
// step squashé) est déjà appliquée (DB EXISTANTE / prod), et qu'il l'applique bien sur
// une DB VIERGE (sentinelle absente). La preuve est décisive : la baseline « poison »
// a un ApplySchema qui ÉCHOUE s'il est appelé — donc l'absence d'erreur = DDL non rejoué.

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func poisonBaseline() Migration {
	return Migration{
		Name:            "baseline_dm5_probe",
		TargetDB:        TargetPlayer,
		Description:     "baseline squashée factice (DM-5 test)",
		SupersededByAll: []string{"legacy_sentinel_dm5"},
		ApplySchema: func(*sql.DB) error {
			return fmt.Errorf("DDL de baseline rejoué alors que la sentinelle est présente (DM-5 violé)")
		},
	}
}

func legacySentinel() Migration {
	return Migration{
		Name:        "legacy_sentinel_dm5",
		TargetDB:    TargetPlayer,
		Description: "dernier step squashé (factice)",
		ApplySchema: func(*sql.DB) error { return nil },
	}
}

// TestDM5_SupersededBaselineSkipsDDLWhenSentinelApplied : DB « existante » (sentinelle
// déjà appliquée au cycle précédent) → la baseline est enregistrée SANS rejouer son DDL.
func TestDM5_SupersededBaselineSkipsDDLWhenSentinelApplied(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Cycle 1 : applique la sentinelle (simule une DB prod ayant tout l'historique squashé).
	if err := RunSteps(db, TargetPlayer, []Migration{legacySentinel()}); err != nil {
		t.Fatalf("cycle 1 (sentinelle): %v", err)
	}
	// Cycle 2 : la baseline poison ne DOIT PAS voir son ApplySchema appelé (sentinelle présente).
	if err := RunSteps(db, TargetPlayer, []Migration{legacySentinel(), poisonBaseline()}); err != nil {
		t.Fatalf("DM-5 n'a pas court-circuité le DDL de la baseline : %v", err)
	}
	// La baseline doit être enregistrée dans schema_migrations (schema_done + backfill_done).
	var schemaDone, backfillDone bool
	if err := db.QueryRow(
		`SELECT schema_done, backfill_done FROM schema_migrations WHERE name = 'baseline_dm5_probe'`,
	).Scan(&schemaDone, &backfillDone); err != nil {
		t.Fatalf("baseline non enregistrée : %v", err)
	}
	if !schemaDone || !backfillDone {
		t.Errorf("baseline enregistrée mais schema_done=%v backfill_done=%v (attendu true/true)", schemaDone, backfillDone)
	}
}

// TestDM5_BaselineDDLRunsOnVirginDB : DB VIERGE (sentinelle absente) → le DDL de la
// baseline s'exécute (le poison échoue), confirmant que DM-5 ne court-circuite QUE quand
// la sentinelle est présente (pas de faux positif masquant une vraie DB neuve).
func TestDM5_BaselineDDLRunsOnVirginDB(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	err = RunSteps(db, TargetPlayer, []Migration{poisonBaseline()})
	if err == nil {
		t.Fatalf("sur DB vierge, le DDL de la baseline aurait dû s'exécuter (poison), mais aucune erreur")
	}
}
