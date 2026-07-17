package main

// main_test.go — ME1 (revue 2026-07). escapeLikeLiteral échappe les métacaractères
// LIKE d'un préfixe étranger ; le prédicat `LIKE ... ESCAPE '\'` doit alors matcher
// STRICTEMENT (comme le HasPrefix de l'indexeur), pas traiter `_`/`%` en wildcards.

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestEscapeLikeLiteral(t *testing.T) {
	cases := map[string]string{
		"Halo_5_Guardians-": `Halo\_5\_Guardians-`,
		"a%b":               `a\%b`,
		`c\d`:               `c\\d`,
		"plain-":            "plain-",
		`_%\`:               `\_\%\\`,
	}
	for in, want := range cases {
		if got := escapeLikeLiteral(in); got != want {
			t.Errorf("escapeLikeLiteral(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestForeignPurgeLikeMatchesLiteralPrefix — le prédicat réel (escapeLikeLiteral +
// ESCAPE '\') ne matche QUE le préfixe littéral : "Halo_5_Guardians-" ne doit PAS
// attraper "HaloX5YGuardians-" (où `_` aurait matché n'importe quel caractère avant le
// fix). Preuve que la purge --foreign-only reste bornée aux fichiers revendiqués.
func TestForeignPurgeLikeMatchesLiteralPrefix(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `CREATE TABLE media_files (file_name VARCHAR)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	names := []string{
		"Halo_5_Guardians-clip1.mp4", // préfixe littéral → doit matcher
		"HaloX5YGuardians-evil.mp4",  // `_`→X/Y : ne DOIT PAS matcher (bug d'origine)
		"Infinite-clip.mp4",          // sans rapport → ne matche pas
	}
	for _, n := range names {
		if _, err := db.ExecContext(ctx, `INSERT INTO media_files VALUES (?)`, n); err != nil {
			t.Fatalf("insert %s: %v", n, err)
		}
	}

	const prefix = "Halo_5_Guardians-"
	var got []string
	rows, err := db.QueryContext(ctx,
		`SELECT file_name FROM media_files WHERE lower(file_name) LIKE lower(?) || '%' ESCAPE '\'`,
		escapeLikeLiteral(prefix))
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, n)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}

	if len(got) != 1 || got[0] != "Halo_5_Guardians-clip1.mp4" {
		t.Errorf("prédicat foreign = %v, attendu uniquement [Halo_5_Guardians-clip1.mp4] "+
			"(le `_` ne doit pas matcher X/Y)", got)
	}
}
