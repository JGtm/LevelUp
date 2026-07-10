//go:build integration

// Package halo5 — testsupport_test.go : outillage de test du sous-package.
//
// Le sous-package halo5 (extraction K3a) ne peut pas réutiliser openMemDB de
// duckdb-root (helper _test.go non exporté, packages de test disjoints) ni
// construire un *duckdb.DB (champs non-exportés). Les sources h5 ne dépendent que
// de l'interface duckdb.SharedReader (méthode unique Get -> *sql.DB) : les tests
// passent donc par un *sql.DB in-memory + un double de SharedReader, sans jamais
// matérialiser un *duckdb.DB. Aucun symbole de prod n'a besoin d'être exporté.
package halo5

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openMemSQL ouvre une DuckDB :memory: (fermée par t.Cleanup).
func openMemSQL(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open :memory:: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// memSharedReader implémente duckdb.SharedReader sur une DB in-memory : même
// contrat que le LegacySharedReader de prod (renvoie le handle + un close no-op).
type memSharedReader struct{ db *sql.DB }

func (r *memSharedReader) Get(_ context.Context) (*sql.DB, func(), error) {
	return r.db, func() {}, nil
}
