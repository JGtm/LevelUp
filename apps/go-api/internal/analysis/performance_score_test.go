// Package analysis — performance_score_test.go : tests des utilitaires restants de
// performance_score.go (l'algorithme de performance score relatif a été retiré le
// 2026-09-10, cf. en-tête de performance_score.go).
package analysis

import "testing"

func TestClampF(t *testing.T) {
	tests := []struct {
		v, min, max, want float64
	}{
		{5, 0, 10, 5},
		{-1, 0, 10, 0},
		{15, 0, 10, 10},
		{0, 0, 10, 0},
		{10, 0, 10, 10},
	}
	for _, tt := range tests {
		got := clampF(tt.v, tt.min, tt.max)
		if got != tt.want {
			t.Errorf("clampF(%f, %f, %f) = %f, want %f", tt.v, tt.min, tt.max, got, tt.want)
		}
	}
}

// intPtr est un helper partagé par les tests du paquet (cf. medal_exploit_test.go).
func intPtr(v int) *int { return &v }
