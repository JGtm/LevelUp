// Package duckdb_test — vérifie que la fermeture d'une référence temporaire
// n'invalide pas une autre référence encore active sur le même fichier.
package duckdb_test

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/duckdb"
)

func TestOpenReadOnly_CloseDoesNotInvalidateOtherReferences(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "metadata.duckdb")

	rw, err := duckdb.OpenReadWrite(dbPath)
	if err != nil {
		t.Fatalf("OpenReadWrite(%s): %v", dbPath, err)
	}
	if _, err := rw.Exec(ctx, "CREATE TABLE refcount_test (value INTEGER)"); err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}
	if _, err := rw.Exec(ctx, "INSERT INTO refcount_test VALUES (42)"); err != nil {
		t.Fatalf("INSERT: %v", err)
	}
	if err := rw.Close(); err != nil {
		t.Fatalf("rw.Close(): %v", err)
	}

	first, err := duckdb.OpenReadOnly(dbPath)
	if err != nil {
		t.Fatalf("OpenReadOnly first(%s): %v", dbPath, err)
	}
	defer first.Close()

	second, err := duckdb.OpenReadOnly(dbPath)
	if err != nil {
		t.Fatalf("OpenReadOnly second(%s): %v", dbPath, err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("second.Close(): %v", err)
	}

	var value int
	if err := first.QueryRow(ctx, "SELECT value FROM refcount_test").Scan(&value); err != nil {
		t.Fatalf("SELECT après fermeture de la deuxième référence: %v", err)
	}
	if value != 42 {
		t.Fatalf("value = %d, want 42", value)
	}
}
