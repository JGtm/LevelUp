//go:build integration

package duckdb

import (
	"context"
	"testing"
)

// TestWriteOAuthRefreshToken_NoPKConstraint est le test de non-régression du bug
// « The specified columns as conflict target are not referenced by a
// UNIQUE/PRIMARY KEY CONSTRAINT » : la player DB legacy de Chocoboflor avait un
// sync_meta SANS PRIMARY KEY sur `key`, ce qui faisait échouer l'ancien
// `INSERT ... ON CONFLICT(key)`. Le pattern SELECT-then-UPDATE-or-INSERT doit
// fonctionner aussi bien avec qu'avec PK, en INSERT comme en UPDATE.
func TestWriteOAuthRefreshToken_NoPKConstraint(t *testing.T) {
	cases := []struct {
		name string
		ddl  string
	}{
		{"avec_pk", "CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR)"},
		{"legacy_sans_pk", "CREATE TABLE sync_meta (key VARCHAR, value VARCHAR, updated_at TIMESTAMP)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := openMemDB(t)
			ctx := context.Background()
			if _, err := db.Exec(ctx, tc.ddl); err != nil {
				t.Fatalf("create sync_meta: %v", err)
			}

			// 1er write : pas de ligne existante → INSERT.
			if err := WriteOAuthRefreshToken(ctx, db, "rt_v1"); err != nil {
				t.Fatalf("write (insert): %v", err)
			}
			if got := readOAuth(t, db); got != "rt_v1" {
				t.Fatalf("après insert = %q, want rt_v1", got)
			}

			// 2e write : ligne existante → UPDATE in-place (pas de doublon).
			if err := WriteOAuthRefreshToken(ctx, db, "rt_v2"); err != nil {
				t.Fatalf("write (update): %v", err)
			}
			if got := readOAuth(t, db); got != "rt_v2" {
				t.Fatalf("après update = %q, want rt_v2", got)
			}
			var n int
			if err := db.QueryRow(ctx,
				"SELECT COUNT(*) FROM sync_meta WHERE key = 'oauth_refresh_token'").Scan(&n); err != nil {
				t.Fatalf("count: %v", err)
			}
			if n != 1 {
				t.Fatalf("nombre de lignes oauth_refresh_token = %d, want 1", n)
			}
		})
	}
}

// TestWriteOAuthRefreshToken_EmptyTokenNoop : token vide = no-op silencieux.
func TestWriteOAuthRefreshToken_EmptyTokenNoop(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()
	if _, err := db.Exec(ctx, "CREATE TABLE sync_meta (key VARCHAR, value VARCHAR)"); err != nil {
		t.Fatal(err)
	}
	if err := WriteOAuthRefreshToken(ctx, db, ""); err != nil {
		t.Fatalf("write empty: %v", err)
	}
	var n int
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM sync_meta").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("token vide a écrit %d lignes, want 0", n)
	}
}

func readOAuth(t *testing.T, db *DB) string {
	t.Helper()
	got, err := ReadOAuthRefreshToken(context.Background(), db)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return got
}
