//go:build integration

package persist

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func importTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, filepath.Join(t.TempDir(), "src.parquet")
}

func importCount(t *testing.T, db *sql.DB, q string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q).Scan(&n); err != nil {
		t.Fatalf("%s : %v", q, err)
	}
	return n
}

// TestImportPersister_OwnerAndWhereRestrictRows : seules les lignes du xuid attendu entrent dans une table propre
// à un joueur, seules celles des matchs acceptés entrent dans une table de match.
func TestImportPersister_OwnerAndWhereRestrictRows(t *testing.T) {
	db, src := importTestDB(t)
	for _, s := range []string{
		`CREATE TABLE t (match_id VARCHAR, xuid VARCHAR)`,
		`CREATE TABLE src AS SELECT * FROM (VALUES ('m1','1'),('m1','2'),('m2','2')) v(match_id, xuid)`,
		`COPY src TO '` + strings.ReplaceAll(src, `\`, `/`) + `' (FORMAT PARQUET)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s : %v", s, err)
		}
	}
	file := strings.ReplaceAll(src, `\`, `/`)
	ctx := context.Background()
	p := NewImportPersister(db)

	if _, err := p.Persist(ctx, []ImportTable{{Name: "t", File: file, Key: []string{"match_id", "xuid"}, OwnerCol: "xuid", OwnerValue: "1"}}); err != nil {
		t.Fatal(err)
	}
	if got := importCount(t, db, `SELECT count(*) FROM t`); got != 1 {
		t.Fatalf("OwnerCol : %d ligne(s), attendu 1 (la ligne du xuid 1)", got)
	}
	if _, err := p.Persist(ctx, []ImportTable{{Name: "t", File: file, Key: []string{"match_id", "xuid"}, Where: "s.match_id = 'm1'"}}); err != nil {
		t.Fatal(err)
	}
	if got := importCount(t, db, `SELECT count(*) FROM t`); got != 2 {
		t.Fatalf("Where : %d ligne(s), attendu 2 (m1 complet : la ligne 1 existait déjà, celle du xuid 2 s'ajoute)", got)
	}
}

// TestImportPersister_LocalIDIsNotImported : une colonne id remplie par la table locale n'est pas reprise du
// fichier (la séquence d'origine n'est pas celle d'ici) ; l'ordre du fichier est conservé.
func TestImportPersister_LocalIDIsNotImported(t *testing.T) {
	db, src := importTestDB(t)
	file := strings.ReplaceAll(src, `\`, `/`)
	for _, s := range []string{
		`CREATE SEQUENCE t_seq START 1000`,
		`CREATE TABLE t (id BIGINT DEFAULT nextval('t_seq'), match_id VARCHAR NOT NULL, label VARCHAR)`,
		`CREATE TABLE src AS SELECT * FROM (VALUES (7, 'm1', 'second'), (3, 'm1', 'first')) v(id, match_id, label)`,
		`COPY src TO '` + file + `' (FORMAT PARQUET)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s : %v", s, err)
		}
	}
	if _, err := NewImportPersister(db).Persist(context.Background(), []ImportTable{{Name: "t", File: file, Key: []string{"match_id"}}}); err != nil {
		t.Fatal(err)
	}
	var first string
	var minID int64
	if err := db.QueryRow(`SELECT arg_min(label, id), min(id) FROM t`).Scan(&first, &minID); err != nil {
		t.Fatal(err)
	}
	if minID < 1000 {
		t.Errorf("id importé tel quel (%d) : la séquence locale doit le remplir", minID)
	}
	if first != "first" {
		t.Errorf("ordre du fichier perdu : la première ligne locale est %q, attendu %q (id source 3)", first, "first")
	}
}
