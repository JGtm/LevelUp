//go:build integration

package migration

import (
	"database/sql"
	"errors"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestAddColumnIfMissing_NewColumn vérifie que la colonne est ajoutée si elle n'existe pas.
func TestAddColumnIfMissing_NewColumn(t *testing.T) {
	db := openMemDB(t)
	db.Exec("CREATE TABLE t1 (id INTEGER)")

	if err := addColumnIfMissing(db, "t1", "newcol", "VARCHAR"); err != nil {
		t.Fatal(err)
	}

	// Vérifier que la colonne existe
	exists, err := columnExists(db, "t1", "newcol")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("expected newcol to exist")
	}
}

// TestAddColumnIfMissing_AlreadyExists vérifie l'idempotence.
func TestAddColumnIfMissing_AlreadyExists(t *testing.T) {
	db := openMemDB(t)
	db.Exec("CREATE TABLE t2 (id INTEGER, existing_col VARCHAR)")

	if err := addColumnIfMissing(db, "t2", "existing_col", "VARCHAR"); err != nil {
		t.Fatal(err)
	}
}

// TestColumnExists_False vérifie le cas colonne absente.
func TestColumnExists_False(t *testing.T) {
	db := openMemDB(t)
	db.Exec("CREATE TABLE t3 (id INTEGER)")

	exists, err := columnExists(db, "t3", "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("expected column to not exist")
	}
}

// TestTableExists vérifie tableExists pour table présente et absente.
func TestTableExists_True(t *testing.T) {
	db := openMemDB(t)
	db.Exec("CREATE TABLE t4 (id INTEGER)")

	exists, err := tableExists(db, "t4")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("expected t4 to exist")
	}
}

func TestTableExists_False(t *testing.T) {
	db := openMemDB(t)

	exists, err := tableExists(db, "nonexistent_table")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("expected table to not exist")
	}
}

// TestRunForDB_WithBackfill_RequiresAPI — la branche RequiresAPI saute le backfill.
func TestRunForDB_WithBackfill_RequiresAPI(t *testing.T) {
	db := openMemDB(t)

	// Enregistrer temporairement une migration test avec RequiresAPI=true
	testTarget := TargetDB("test_target_api")
	Register(Migration{
		Name:        "test_api_migration",
		TargetDB:    testTarget,
		Description: "test migration avec RequiresAPI",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec("CREATE TABLE IF NOT EXISTS test_api_mig (id INTEGER)")
			return err
		},
		ApplyBackfill: func(db *sql.DB) error {
			return errors.New("should not be called")
		},
		RequiresAPI: true,
	})

	if err := RunForDB(db, testTarget); err != nil {
		t.Fatal(err)
	}

	// Table créée mais backfill_done=FALSE car RequiresAPI
	var backfillDone bool
	row := db.QueryRow("SELECT backfill_done FROM schema_migrations WHERE name='test_api_migration'")
	if err := row.Scan(&backfillDone); err != nil {
		t.Fatal(err)
	}
	if backfillDone {
		t.Fatal("expected backfill_done=FALSE for RequiresAPI migration")
	}
}

// TestRunForDB_WithBackfill_Success — la branche backfill sans RequiresAPI.
func TestRunForDB_WithBackfill_Success(t *testing.T) {
	db := openMemDB(t)

	testTarget := TargetDB("test_target_backfill")
	backfillCalled := false
	Register(Migration{
		Name:        "test_backfill_migration",
		TargetDB:    testTarget,
		Description: "test migration avec backfill",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec("CREATE TABLE IF NOT EXISTS test_backfill_src (id INTEGER)")
			return err
		},
		ApplyBackfill: func(db *sql.DB) error {
			backfillCalled = true
			return nil
		},
		RequiresAPI: false,
	})

	if err := RunForDB(db, testTarget); err != nil {
		t.Fatal(err)
	}

	if !backfillCalled {
		t.Fatal("expected backfill to be called")
	}

	var backfillDone bool
	row := db.QueryRow("SELECT backfill_done FROM schema_migrations WHERE name='test_backfill_migration'")
	if err := row.Scan(&backfillDone); err != nil {
		t.Fatal(err)
	}
	if !backfillDone {
		t.Fatal("expected backfill_done=TRUE")
	}
}

// TestRunForDB_WithBackfill_Error — la branche backfill qui échoue (continue).
func TestRunForDB_WithBackfill_Error(t *testing.T) {
	db := openMemDB(t)

	testTarget := TargetDB("test_target_bferr")
	Register(Migration{
		Name:        "test_backfill_error_migration",
		TargetDB:    testTarget,
		Description: "test migration backfill qui échoue",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec("CREATE TABLE IF NOT EXISTS test_bferr (id INTEGER)")
			return err
		},
		ApplyBackfill: func(db *sql.DB) error {
			return errors.New("intentional backfill error")
		},
		RequiresAPI: false,
	})

	// RunForDB ne doit PAS retourner d'erreur (le backfill continue malgré l'erreur)
	if err := RunForDB(db, testTarget); err != nil {
		t.Fatalf("expected no error (backfill errors are ignored), got: %v", err)
	}
}
