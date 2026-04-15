// Package migration — migration_test.go : tests d'idempotence des migrations DuckDB.
//
// Sprint 21 — tâche 5 : appliquer les migrations sur DB vierge puis sur DB existante.
//
// Ces tests requièrent CGO (driver DuckDB) et sont marqués "integration".
// Lancer avec : go test -tags=integration ./internal/migration/ -v

//go:build integration

package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openMemDB ouvre une DuckDB in-memory pour les tests.
func openMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("openMemDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// countMigrations retourne le nb de lignes dans schema_migrations.
func countMigrations(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&n); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	return n
}

// ─────────────────────────────────────────────────────────────────────────────
// Test : RunForDB(TargetMetadata) idempotent
//
// Les migrations metadata n'ont aucune dépendance externe (CREATE TABLE IF NOT
// EXISTS seulement) → elles tournent sur DB vierge sans erreur.
// ─────────────────────────────────────────────────────────────────────────────

func TestRunForDB_Metadata_IdempotentOnEmptyDB(t *testing.T) {
	db := openMemDB(t)

	metaMigs := ForTarget(TargetMetadata)
	if len(metaMigs) == 0 {
		t.Skip("aucune migration metadata enregistrée")
	}

	// Première passe — DB vierge
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("1ère passe RunForDB(Metadata): %v", err)
	}
	countAfterFirst := countMigrations(t, db)
	if countAfterFirst != len(metaMigs) {
		t.Errorf("après passe 1 : %d migrations appliquées, attendu %d",
			countAfterFirst, len(metaMigs))
	}

	// Deuxième passe — DB déjà migrée (test idempotence)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("2ème passe RunForDB(Metadata): %v", err)
	}
	countAfterSecond := countMigrations(t, db)
	if countAfterSecond != countAfterFirst {
		t.Errorf("idempotence violée : passe 2 = %d lignes, passe 1 = %d",
			countAfterSecond, countAfterFirst)
	}

	// Troisième passe — confirmation
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("3ème passe RunForDB(Metadata): %v", err)
	}
	countAfterThird := countMigrations(t, db)
	if countAfterThird != countAfterFirst {
		t.Errorf("idempotence violée passe 3 : %d vs %d", countAfterThird, countAfterFirst)
	}

	t.Logf("✅ %d migrations metadata idempotentes (3 passes)", countAfterFirst)
}

// ─────────────────────────────────────────────────────────────────────────────
// Test : schema_migrations — pas de doublon possible
// ─────────────────────────────────────────────────────────────────────────────

func TestRunForDB_Metadata_NoDuplicateRows(t *testing.T) {
	db := openMemDB(t)

	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB 2ème: %v", err)
	}

	// Vérifier pas de doublon sur name (PK)
	var cnt int
	if err := db.QueryRow(
		"SELECT COUNT(DISTINCT name) FROM schema_migrations WHERE name LIKE 'add_%' OR name LIKE 'drop_%'",
	).Scan(&cnt); err != nil {
		t.Fatalf("count distinct: %v", err)
	}
	total := countMigrations(t, db)
	if total != cnt {
		t.Errorf("doublons détectés : total=%d distinct=%d", total, cnt)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test : toutes les migrations metadata marquées schema_done=TRUE après passe
// ─────────────────────────────────────────────────────────────────────────────

func TestRunForDB_Metadata_AllSchemaDone(t *testing.T) {
	db := openMemDB(t)

	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	var notDone int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM schema_migrations WHERE schema_done = FALSE",
	).Scan(&notDone); err != nil {
		t.Fatalf("query: %v", err)
	}
	if notDone > 0 {
		t.Errorf("%d migration(s) avec schema_done=FALSE après RunForDB", notDone)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test : ForTarget filtre correctement par TargetDB
// ─────────────────────────────────────────────────────────────────────────────

func TestForTarget_ReturnsOnlyTargetMigrations(t *testing.T) {
	targets := []TargetDB{TargetMetadata, TargetPlayer, TargetShared, TargetSharedPvE}
	for _, target := range targets {
		migs := ForTarget(target)
		for _, m := range migs {
			if m.TargetDB != target {
				t.Errorf("ForTarget(%s) contient la migration %q de target %s",
					target, m.Name, m.TargetDB)
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test : le nombre total de migrations couvertes
// ─────────────────────────────────────────────────────────────────────────────

func TestMigrationCount_MinimumExpected(t *testing.T) {
	all := All()
	// Sprint 21 : au moins 36 migrations portées (player+shared+shared_pve+metadata)
	if len(all) < 36 {
		t.Errorf("seulement %d migrations enregistrées, minimum attendu: 36", len(all))
	}
	t.Logf("total migrations enregistrées: %d", len(all))

	metaCount := len(ForTarget(TargetMetadata))
	playerCount := len(ForTarget(TargetPlayer))
	sharedCount := len(ForTarget(TargetShared))
	pveCount := len(ForTarget(TargetSharedPvE))
	t.Logf("  metadata: %d, player: %d, shared: %d, pve: %d",
		metaCount, playerCount, sharedCount, pveCount)
}
