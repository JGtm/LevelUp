//go:build integration

// Package duckdb — migrated_db_test_helper_test.go : helper de test générique
// « ouvre une DB temp migrée pour un target » partagé par les tests d'intégration
// des repos duckdb-root (coach_proposal, milestones, records, streaks…).
//
// Historiquement défini dans prestige_repos_test.go ; ce dernier a suivi les repos
// Prestige dans le sous-package duckdb/prestige (extraction K3a). Le helper, lui,
// n'a rien de spécifique à Prestige et reste ici pour les tests duckdb-root — le
// sous-package prestige en garde sa propre copie (les packages de test sont
// disjoints après l'extraction ; on ne peut pas exporter un helper _test.go).
package duckdb

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// setupPrestigeDB ouvre une DB temp et applique les migrations du target.
// Retourne un *DB (avec pool) prêt pour les repos. Nettoyage via t.Cleanup.
func setupPrestigeDB(t *testing.T, target migration.TargetDB) *DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, string(target)+".duckdb")

	raw, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := migration.RunForDB(raw, target); err != nil {
		raw.Close()
		t.Fatalf("RunForDB(%s): %v", target, err)
	}
	raw.Close()

	db, err := OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.RemoveAll(dir)
	})
	return db
}
