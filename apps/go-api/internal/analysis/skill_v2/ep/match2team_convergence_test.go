package ep

import (
	"errors"
	"fmt"
	"math"
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

// matchSansConvergenceAvecComptes rend le match 4v6 déterministe dont le solveur ne converge pas
// avec les observations de frags et de morts, et converge sans elles : tous les joueurs à
// N(25, 25/3), poids 0,5 pour les deux derniers de l'équipe B, A gagne. Le facteur
// DrawMargin est celui d'une probabilité de nul de 10 % (Phi^-1(0,55) * sqrt(10) * beta).
func matchSansConvergenceAvecComptes(t *testing.T) Match2TeamInput {
	t.Helper()
	const beta = 25.0 / 6.0
	prior, err := FromMeanSigma(25, 25.0/3.0)
	if err != nil {
		t.Fatalf("FromMeanSigma: %v", err)
	}
	teamA := []Gaussian{prior, prior, prior, prior}
	teamB := []Gaussian{prior, prior, prior, prior, prior, prior}
	killsA, deathsA := []float64{17, 17, 15, 1}, []float64{9, 13, 10, 13}
	killsB, deathsB := []float64{18, 7, 20, 3, 0, 12}, []float64{9, 3, 11, 20, 7, 18}

	var counts []CountObservation
	for i := range teamA {
		counts = append(counts,
			CountObservation{PlayerIndex: i, Side: SideA, Type: CountKill, Value: killsA[i]},
			CountObservation{PlayerIndex: i, Side: SideA, Type: CountDeath, Value: deathsA[i]})
	}
	for i := range teamB {
		counts = append(counts,
			CountObservation{PlayerIndex: i, Side: SideB, Type: CountKill, Value: killsB[i]},
			CountObservation{PlayerIndex: i, Side: SideB, Type: CountDeath, Value: deathsB[i]})
	}
	return Match2TeamInput{
		TeamA:      teamA,
		TeamB:      teamB,
		ResultA:    TeamWin,
		Beta:       beta,
		Tau:        25.0 / 300.0,
		DrawMargin: math.Sqrt2 * math.Erfinv(2*0.55-1) * math.Sqrt(10) * beta,
		Counts:     counts,
		WeightsA:   []float64{1, 1, 1, 1},
		WeightsB:   []float64{1, 1, 1, 1, 0.5, 0.5},
	}
}

// Les observations de frags et de morts sont la cause des non-convergences mesurées : ce match
// ne converge pas avec elles, et converge sans elles, poids de temps joué conservés.
func TestUpdateMatch2Team_LesComptesFontDiverger(t *testing.T) {
	input := matchSansConvergenceAvecComptes(t)
	cfg := DefaultMatch2TeamConfig()

	_, _, err := UpdateMatch2Team(input, cfg)
	if !errors.Is(err, ErrNotConverged) {
		t.Fatalf("avec comptes : err = %v, attendu ErrNotConverged", err)
	}
	if want := fmt.Sprintf("ep: EP n'a pas convergé en %d itérations", cfg.MaxIters); err.Error() != want {
		t.Errorf("message = %q, attendu %q", err.Error(), want)
	}

	input.Counts = nil
	postA, postB, err := UpdateMatch2Team(input, cfg)
	if err != nil {
		t.Fatalf("sans comptes : err = %v, attendu une convergence", err)
	}
	if len(postA) != 4 || len(postB) != 6 {
		t.Fatalf("posteriors = %d et %d, attendu 4 et 6", len(postA), len(postB))
	}
}
