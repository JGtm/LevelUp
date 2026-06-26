// Package migration — tests unitaires des helpers PURS (parsing SQL, tokenisation,
// tri par ordre). Zéro DB : les migrations DDL sont exercées par les *_test.go
// tag integration. Cible les pièges documentés (ADR 0026 : `;` en commentaire SQL).
package migration

import (
	"reflect"
	"testing"
)

// TestSplitSQL_SemicolonInComment : un `;` à l'intérieur d'un commentaire `--`
// ne doit PAS scinder l'instruction (piège ADR 0026 : sinon une note de doc casse
// le statement en deux moitiés dont l'une devient du SQL invalide). Le commentaire
// placé sur sa PROPRE ligne avant l'INSERT reste collé à ce statement (commentaire
// de tête toléré par DuckDB) — l'invariant clé est qu'il n'y a PAS de 3e fragment
// né du `;` interne au commentaire.
func TestSplitSQL_SemicolonInComment(t *testing.T) {
	script := `CREATE TABLE t (a INT);
-- note avec ; au milieu
INSERT INTO t VALUES (1);`
	got := splitSQL(script)
	if len(got) != 2 {
		t.Fatalf("splitSQL = %d statements, want 2 (le `;` du commentaire ne scinde pas)\n%#v", len(got), got)
	}
	if got[0] != "CREATE TABLE t (a INT)" {
		t.Errorf("stmt[0] = %q", got[0])
	}
	// stmt[1] porte le commentaire de tête + l'INSERT, sur une seule instruction.
	if want := "-- note avec ; au milieu\nINSERT INTO t VALUES (1)"; got[1] != want {
		t.Errorf("stmt[1] = %q, want %q", got[1], want)
	}
}

// TestSplitSQL_TrailingCommentOnly : un fragment qui ne contient que des
// commentaires après le dernier `;` est ignoré (DuckDB rejette une "empty query").
func TestSplitSQL_TrailingCommentOnly(t *testing.T) {
	script := `CREATE TABLE t (a INT);
-- ceci est juste une note finale, pas une instruction
`
	got := splitSQL(script)
	if len(got) != 1 {
		t.Fatalf("splitSQL = %d, want 1 (le commentaire-seul est ignoré)\n%#v", len(got), got)
	}
	if got[0] != "CREATE TABLE t (a INT)" {
		t.Errorf("stmt[0] = %q", got[0])
	}
}

// TestSplitSQL_MultipleAndEmpty : plusieurs statements + un `;;` (fragment vide
// entre deux) → seuls les non-vides ressortent.
func TestSplitSQL_MultipleAndEmpty(t *testing.T) {
	got := splitSQL(`A;;B;`)
	if !reflect.DeepEqual(got, []string{"A", "B"}) {
		t.Errorf("splitSQL(`A;;B;`) = %#v, want [A B]", got)
	}
}

// TestIsCommentOnly : true ssi le fragment ne contient que des lignes `--` et/ou
// vides ; false dès qu'une ligne porte du SQL exécutable.
func TestIsCommentOnly(t *testing.T) {
	cases := []struct {
		name     string
		fragment string
		want     bool
	}{
		{"empty", "", true},
		{"only comment", "-- juste une note", true},
		{"comment + blank lines", "\n-- a\n\n-- b\n", true},
		{"sql present", "-- note\nSELECT 1", false},
		{"sql then comment", "SELECT 1 -- inline", false},
	}
	for _, tc := range cases {
		if got := isCommentOnly(tc.fragment); got != tc.want {
			t.Errorf("isCommentOnly(%q) = %v, want %v", tc.fragment, got, tc.want)
		}
	}
}

// TestFirstWords : extrait les n premiers mots (séparateurs espace/tab/newline),
// utilisé pour préfixer les logs de purge. Couvre n=0, n > nb de mots, espaces
// multiples et leading/trailing.
func TestFirstWords(t *testing.T) {
	cases := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{"two words", "DROP TABLE foo", 2, "DROP TABLE"},
		{"n exceeds", "alpha beta", 5, "alpha beta"},
		{"n zero", "alpha beta", 0, ""},
		{"collapse spaces", "  a    b   c ", 2, "a b"},
		{"tabs and newlines", "a\tb\nc", 2, "a b"},
		{"empty", "", 3, ""},
	}
	for _, tc := range cases {
		if got := firstWords(tc.s, tc.n); got != tc.want {
			t.Errorf("%s: firstWords(%q,%d) = %q, want %q", tc.name, tc.s, tc.n, got, tc.want)
		}
	}
}

// TestSortByOrder : réordonne selon l'ordre fourni ; les noms inconnus vont en
// fin en préservant leur ordre relatif d'entrée (tri STABLE — PMT-9, un titre
// non-défaut impose son propre ordre).
func TestSortByOrder(t *testing.T) {
	order := []string{"first", "second", "third"}
	ms := []Migration{
		{Name: "unknown_a"},
		{Name: "third"},
		{Name: "first"},
		{Name: "unknown_b"},
		{Name: "second"},
	}
	sortByOrder(order, ms)
	gotNames := make([]string, len(ms))
	for i, m := range ms {
		gotNames[i] = m.Name
	}
	want := []string{"first", "second", "third", "unknown_a", "unknown_b"}
	if !reflect.DeepEqual(gotNames, want) {
		t.Errorf("sortByOrder = %v, want %v (inconnus en fin, ordre relatif stable)", gotNames, want)
	}
}
