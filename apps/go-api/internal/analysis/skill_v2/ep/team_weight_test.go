package ep

import "testing"

func TestResolveTeamWeight(t *testing.T) {
	w := []float64{0.5, 0.0, -1.0, 0.02, 1.5}
	cases := []struct {
		name string
		idx  int
		want float64
	}{
		{"valeur normale", 0, 0.5},
		{"zéro → 1 (non spécifié)", 1, 1.0},
		{"négatif → 1", 2, 1.0},
		{"sous le plancher → plancher", 3, teamWeightFloor},
		{"> 1 → 1", 4, 1.0},
		{"index hors borne → 1", 99, 1.0},
		{"slice nil → 1", 0, 1.0}, // testé séparément ci-dessous
	}
	for _, tc := range cases[:6] {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveTeamWeight(w, tc.idx); got != tc.want {
				t.Errorf("resolveTeamWeight(idx=%d) = %v, want %v", tc.idx, got, tc.want)
			}
		})
	}
	if got := resolveTeamWeight(nil, 0); got != 1.0 {
		t.Errorf("resolveTeamWeight(nil) = %v, want 1.0", got)
	}
}
