package domain

import "testing"

func TestPlacementLabel_AllerRetour(t *testing.T) {
	for _, n := range []int{0, 1, 4, 10} {
		l := PlacementLabel(n)
		if !IsPlacementLabel(l) {
			t.Fatalf("PlacementLabel(%d) = %q : pas reconnu", n, l)
		}
		if got := ParsePlacementRemaining(l); got != n {
			t.Errorf("ParsePlacementRemaining(%q) = %d, want %d", l, got, n)
		}
	}
}

func TestParsePlacementRemaining_LignesDejaEcrites(t *testing.T) {
	cases := map[string]int{
		"Placement (4 restants)":   4,
		"Placement (10 restants)":  10,
		"Placement (0 restant)":    0,
		"Placement (1 restant)":    1,
		"Placement":                PlacementDefaultRemaining,
		"":                         PlacementDefaultRemaining,
		"Placement (999 restants)": PlacementDefaultRemaining,
		"Placement (x)":            PlacementDefaultRemaining,
		"Onyx 3":                   PlacementDefaultRemaining,
		"Bronze 1":                 PlacementDefaultRemaining,
	}
	for in, want := range cases {
		if got := ParsePlacementRemaining(in); got != want {
			t.Errorf("ParsePlacementRemaining(%q) = %d, want %d", in, got, want)
		}
	}
}
