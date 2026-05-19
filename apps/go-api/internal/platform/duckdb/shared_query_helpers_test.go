package duckdb

import (
	"reflect"
	"testing"
)

func TestPlaceholders(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{-1, ""},
		{0, ""},
		{1, "?"},
		{2, "?, ?"},
		{3, "?, ?, ?"},
		{5, "?, ?, ?, ?, ?"},
	}
	for _, tc := range tests {
		got := Placeholders(tc.n)
		if got != tc.want {
			t.Errorf("Placeholders(%d) = %q, attendu %q", tc.n, got, tc.want)
		}
	}
}

func TestToAnySlice_Strings(t *testing.T) {
	in := []string{"a", "b", "c"}
	got := ToAnySlice(in)
	want := []any{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ToAnySlice strings = %v, attendu %v", got, want)
	}
}

func TestToAnySlice_Ints(t *testing.T) {
	in := []int{1, 2, 3}
	got := ToAnySlice(in)
	want := []any{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ToAnySlice ints = %v, attendu %v", got, want)
	}
}

func TestToAnySlice_Empty(t *testing.T) {
	if got := ToAnySlice([]string{}); got != nil {
		t.Errorf("ToAnySlice slice vide = %v, attendu nil", got)
	}
	if got := ToAnySlice[int](nil); got != nil {
		t.Errorf("ToAnySlice nil = %v, attendu nil", got)
	}
}

// TestPlaceholders_Composable vérifie l'usage typique avec un fmt.Sprintf.
func TestPlaceholders_Composable(t *testing.T) {
	ids := []string{"m1", "m2", "m3"}
	q := "SELECT x FROM t WHERE id IN (" + Placeholders(len(ids)) + ")"
	want := "SELECT x FROM t WHERE id IN (?, ?, ?)"
	if q != want {
		t.Errorf("composition = %q, attendu %q", q, want)
	}
}
