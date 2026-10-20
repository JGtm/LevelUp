package duckdb

import (
	"database/sql"
	"errors"
	"testing"
)

func TestIsCatalogError(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	var one int
	missing := db.QueryRow(`SELECT 1 FROM table_absente LIMIT 1`).Scan(&one)
	if !IsCatalogError(missing) {
		t.Fatalf("table absente : IsCatalogError(%v) = false, attendu true", missing)
	}
	syntax := db.QueryRow(`SELEC 1`).Scan(&one)
	if syntax == nil || IsCatalogError(syntax) {
		t.Fatalf("erreur de syntaxe : IsCatalogError(%v) = true, attendu false", syntax)
	}
	if IsCatalogError(errors.New("Catalog Error: Table with name x does not exist")) {
		t.Fatal("un texte seul ne suffit pas : seul le type d'erreur du pilote compte")
	}
	if IsCatalogError(nil) {
		t.Fatal("nil n'est pas une erreur de catalogue")
	}
}
