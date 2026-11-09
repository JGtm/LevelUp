package ep

import (
	"errors"
	"testing"
)

// Le solveur arrêté avant convergence rend l'erreur typée, reconnue par errors.Is et portant le
// nombre d'itérations.
func TestUpdateMatch2Team_NonConvergenceTypee(t *testing.T) {
	prior, err := FromMeanVariance(25, 70)
	if err != nil {
		t.Fatalf("FromMeanVariance: %v", err)
	}
	input := Match2TeamInput{
		TeamA:   []Gaussian{prior, prior},
		TeamB:   []Gaussian{prior, prior},
		ResultA: TeamWin,
		Beta:    4,
	}

	_, _, err = UpdateMatch2Team(input, Match2TeamConfig{Tolerance: 1e-12, MaxIters: 1})

	if !errors.Is(err, ErrNotConverged) {
		t.Fatalf("err = %v, attendu ErrNotConverged", err)
	}
	if want := "ep: EP n'a pas convergé en 1 itérations"; err.Error() != want {
		t.Errorf("message = %q, attendu %q", err.Error(), want)
	}
}
