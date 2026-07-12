//go:build integration

package migrations

// testhelpers_test.go — helpers de test partagés du package migrations. openEngMemDB
// (ouverture d'une DB DuckDB :memory' jetable) vivait dans player_engagement_pkfix_test.go,
// retiré avec le squash player v1 (chantier N4) ; il reste utilisé par plusieurs tests
// title-owned → relocalisé ici.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openEngMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
