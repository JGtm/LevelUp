package duckdb

import (
	"context"
	"path/filepath"
	"testing"
)

func TestTableInspector_CountRows(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "insp.duckdb")

	// Crée la base + une table peuplée (RW), handle gardé ouvert pendant la
	// lecture (OpenReadForQuery réutilise le handle en cache).
	db, err := OpenReadWriteShared(path)
	if err != nil {
		t.Fatalf("open rw: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(ctx, `CREATE TABLE foo (id INTEGER)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO foo VALUES (1), (2), (3)`); err != nil {
		t.Fatalf("insert: %v", err)
	}

	insp := NewTableInspector()

	n, exists, err := insp.CountRows(ctx, path, "foo")
	if err != nil {
		t.Fatalf("count foo: %v", err)
	}
	if !exists || n != 3 {
		t.Fatalf("foo: exists=%v rows=%d, want true/3", exists, n)
	}

	_, exists2, err := insp.CountRows(ctx, path, "does_not_exist")
	if err != nil {
		t.Fatalf("count missing: %v", err)
	}
	if exists2 {
		t.Fatalf("does_not_exist should report exists=false")
	}
}
