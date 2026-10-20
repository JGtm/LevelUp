package duckdbkeys

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// updatedAt : nom d'horodatage préféré par RecencyColumn.
const updatedAt = "updated_at"

func openMem(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func mustExec(t *testing.T, db *sql.DB, q string) {
	t.Helper()
	if _, err := db.Exec(q); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
}

func TestTableKeys_PrimaryThenUnique(t *testing.T) {
	db := openMem(t)
	mustExec(t, db, `CREATE TABLE t (a VARCHAR, b INTEGER, c VARCHAR, UNIQUE (c), PRIMARY KEY (a, b))`)
	mustExec(t, db, `CREATE TABLE nokey (a VARCHAR)`)
	keys, err := TableKeys(context.Background(), db, "t")
	if err != nil {
		t.Fatal(err)
	}
	want := []Key{{Primary: true, Columns: []string{"a", "b"}}, {Primary: false, Columns: []string{"c"}}}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("keys = %+v, want %+v", keys, want)
	}
	none, err := TableKeys(context.Background(), db, "nokey")
	if err != nil || len(none) != 0 {
		t.Fatalf("table sans clé : keys=%+v err=%v", none, err)
	}
}

func TestRecencyColumn(t *testing.T) {
	cols := []Column{{Name: "id", Type: "VARCHAR"}, {Name: "created_at", Type: "TIMESTAMP"},
		{Name: updatedAt, Type: "TIMESTAMP WITH TIME ZONE"}}
	if got := RecencyColumn(cols); got != updatedAt {
		t.Fatalf("préféré : got %q", got)
	}
	if got := RecencyColumn([]Column{{Name: "x", Type: "DATE"}, {Name: "y", Type: "TIMESTAMP"}}); got != "x" {
		t.Fatalf("premier horodatage : got %q", got)
	}
	if got := RecencyColumn([]Column{{Name: "x", Type: "VARCHAR"}}); got != "" {
		t.Fatalf("sans horodatage : got %q", got)
	}
}

// TestCopyInto_GardeLaPlusRecenteEtCompte : doublons de clé départagés par l'horodatage, clé
// primaire NULL écartée, doublons divergents comptés.
func TestCopyInto_GardeLaPlusRecenteEtCompte(t *testing.T) {
	db := openMem(t)
	mustExec(t, db, `CREATE TABLE src (id VARCHAR, v INTEGER, updated_at TIMESTAMP)`)
	mustExec(t, db, `INSERT INTO src VALUES
		('a', 1, '2026-01-01'), ('a', 2, '2026-02-01'),
		('b', 5, '2026-01-01'), ('b', 5, '2026-01-01'),
		(NULL, 9, '2026-01-01'), ('c', 7, NULL)`)
	mustExec(t, db, `CREATE TABLE dst (id VARCHAR PRIMARY KEY, v INTEGER, updated_at TIMESTAMP)`)
	keys, err := TableKeys(context.Background(), db, "dst")
	if err != nil {
		t.Fatal(err)
	}
	cols := []string{"id", "v", "updated_at"}
	st, err := CopyInto(context.Background(), db, "dst", Ident("src"), cols, keys, updatedAt)
	if err != nil {
		t.Fatal(err)
	}
	want := CopyStats{Source: 6, Copied: 3, NullKeyDropped: 1, DuplicatesDropped: 2, DivergentKeys: 1}
	if st != want {
		t.Fatalf("stats = %+v, want %+v", st, want)
	}
	var v int
	if err := db.QueryRow(`SELECT v FROM dst WHERE id = 'a'`).Scan(&v); err != nil || v != 2 {
		t.Fatalf("la ligne la plus récente de 'a' doit être gardée : v=%d err=%v", v, err)
	}
}

// TestDedupSelect_SansHorodatageStable : sans horodatage, la ligne gardée ne dépend que du
// contenu (ordre des colonnes), pas de l'ordre d'insertion.
func TestDedupSelect_SansHorodatageStable(t *testing.T) {
	for _, order := range []string{`('k', 'z'), ('k', 'a')`, `('k', 'a'), ('k', 'z')`} {
		db := openMem(t)
		mustExec(t, db, `CREATE TABLE src (k VARCHAR, v VARCHAR)`)
		mustExec(t, db, `INSERT INTO src VALUES `+order)
		q := DedupSelect(Ident("src"), []string{"k", "v"},
			[]Key{{Primary: false, Columns: []string{"k"}}}, "")
		var k, v string
		if err := db.QueryRow(q).Scan(&k, &v); err != nil {
			t.Fatal(err)
		}
		if v != "a" {
			t.Fatalf("ordre %s : ligne gardée v=%q, want 'a'", order, v)
		}
	}
}

// TestDedupSelect_UniqueGardeLesNull : une contrainte UNIQUE ne met pas en conflit deux lignes
// dont la clé est NULL.
func TestDedupSelect_UniqueGardeLesNull(t *testing.T) {
	db := openMem(t)
	mustExec(t, db, `CREATE TABLE src (k VARCHAR, v INTEGER)`)
	mustExec(t, db, `INSERT INTO src VALUES (NULL, 1), (NULL, 2), ('x', 3), ('x', 4)`)
	var n int
	q := "SELECT count(*) FROM (" + DedupSelect(Ident("src"), []string{"k", "v"},
		[]Key{{Primary: false, Columns: []string{"k"}}}, "") + ")"
	if err := db.QueryRow(q).Scan(&n); err != nil || n != 3 {
		t.Fatalf("lignes gardées = %d (err=%v), want 3", n, err)
	}
}
