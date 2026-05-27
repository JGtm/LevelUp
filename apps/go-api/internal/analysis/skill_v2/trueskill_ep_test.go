package skill_v2

import (
	"fmt"
	"math"
	"testing"
)

// trueskill_ep_test.go : régression test du wrapper EP vs le closed-form.
//
// Chaque scenario applique le même TwoTeamMatch via UpdateTwoTeam et
// UpdateTwoTeamEP, et vérifie que les posteriors matchent à eps près.
//
// Tolérance choisie : 1e-2 sur μ et σ. EP avec MaxIters=50 + tolerance
// 1e-4 converge typiquement bien au-delà de ce seuil ; on garde une marge
// pour éviter les flakes sur configurations bordeline.

const epRegressionTol = 1e-2

// assertGaussianClose vérifie que deux Gaussiennes sont numériquement proches.
func assertGaussianClose(t *testing.T, label string, got, want Gaussian, tol float64) {
	t.Helper()
	if math.Abs(got.Mu-want.Mu) > tol {
		t.Errorf("%s: μ %v != %v (tol %v)", label, got.Mu, want.Mu, tol)
	}
	if math.Abs(got.Sigma-want.Sigma) > tol {
		t.Errorf("%s: σ %v != %v (tol %v)", label, got.Sigma, want.Sigma, tol)
	}
}

// assertTeamsMatch compare deux slices de Gaussian.
func assertTeamsMatch(t *testing.T, label string, got, want []Gaussian, tol float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: len diff (got %d, want %d)", label, len(got), len(want))
	}
	for i := range got {
		assertGaussianClose(t, fmt.Sprintf("%s[%d]", label, i), got[i], want[i], tol)
	}
}

// runBothAndCompare applique le même match aux deux implémentations et
// retourne les deux posteriors. Échoue immédiatement si l'une produit
// une erreur.
func runBothAndCompare(t *testing.T, m TwoTeamMatch, p Priors) (cfA, cfB, epA, epB []Gaussian) {
	t.Helper()
	cfA, cfB, errCF := UpdateTwoTeam(m, p)
	if errCF != nil {
		t.Fatalf("UpdateTwoTeam: %v", errCF)
	}
	epA, epB, errEP := UpdateTwoTeamEP(m, p)
	if errEP != nil {
		t.Fatalf("UpdateTwoTeamEP: %v", errEP)
	}
	return
}

func TestEPRegression_1v1_Win(t *testing.T) {
	p := DefaultPriors()
	m := TwoTeamMatch{
		TeamA:   []Gaussian{p.NewPlayerState()},
		TeamB:   []Gaussian{p.NewPlayerState()},
		ResultA: TeamWin,
	}
	cfA, cfB, epA, epB := runBothAndCompare(t, m, p)
	assertTeamsMatch(t, "teamA", epA, cfA, epRegressionTol)
	assertTeamsMatch(t, "teamB", epB, cfB, epRegressionTol)
}

func TestEPRegression_1v1_Loss(t *testing.T) {
	p := DefaultPriors()
	m := TwoTeamMatch{
		TeamA:   []Gaussian{{Mu: 28, Sigma: 5}},
		TeamB:   []Gaussian{{Mu: 22, Sigma: 6}},
		ResultA: TeamLoss,
	}
	cfA, cfB, epA, epB := runBothAndCompare(t, m, p)
	assertTeamsMatch(t, "teamA loser", epA, cfA, epRegressionTol)
	assertTeamsMatch(t, "teamB winner", epB, cfB, epRegressionTol)
}

func TestEPRegression_1v1_Draw(t *testing.T) {
	p := DefaultPriors()
	m := TwoTeamMatch{
		TeamA:   []Gaussian{{Mu: 30, Sigma: 25.0 / 3.0}},
		TeamB:   []Gaussian{{Mu: 20, Sigma: 25.0 / 3.0}},
		ResultA: TeamDraw,
	}
	cfA, cfB, epA, epB := runBothAndCompare(t, m, p)
	assertTeamsMatch(t, "teamA draw", epA, cfA, epRegressionTol)
	assertTeamsMatch(t, "teamB draw", epB, cfB, epRegressionTol)
}

func TestEPRegression_4v4_Win(t *testing.T) {
	p := DefaultPriors()
	a := make([]Gaussian, 4)
	b := make([]Gaussian, 4)
	for i := range a {
		a[i] = p.NewPlayerState()
		b[i] = p.NewPlayerState()
	}
	m := TwoTeamMatch{TeamA: a, TeamB: b, ResultA: TeamWin}
	cfA, cfB, epA, epB := runBothAndCompare(t, m, p)
	assertTeamsMatch(t, "teamA 4v4", epA, cfA, epRegressionTol)
	assertTeamsMatch(t, "teamB 4v4", epB, cfB, epRegressionTol)
}

func TestEPRegression_AsymmetricSkills_Win(t *testing.T) {
	p := DefaultPriors()
	// Veteran (faible σ) vs newbie (large σ) avec écart μ marqué.
	veteran := Gaussian{Mu: 30, Sigma: 2}
	newbie := Gaussian{Mu: 20, Sigma: 8}
	m := TwoTeamMatch{
		TeamA:   []Gaussian{veteran},
		TeamB:   []Gaussian{newbie},
		ResultA: TeamWin,
	}
	cfA, cfB, epA, epB := runBothAndCompare(t, m, p)
	assertTeamsMatch(t, "veteran", epA, cfA, epRegressionTol)
	assertTeamsMatch(t, "newbie", epB, cfB, epRegressionTol)
}

func TestEPRegression_TeamSize_1v3(t *testing.T) {
	p := DefaultPriors()
	a := []Gaussian{p.NewPlayerState()}
	b := []Gaussian{p.NewPlayerState(), p.NewPlayerState(), p.NewPlayerState()}
	m := TwoTeamMatch{TeamA: a, TeamB: b, ResultA: TeamWin}
	cfA, cfB, epA, epB := runBothAndCompare(t, m, p)
	// 1 player solo qui gagne contre 3 : sa σ chute fortement.
	assertTeamsMatch(t, "solo winner", epA, cfA, epRegressionTol)
	assertTeamsMatch(t, "team of 3 losers", epB, cfB, epRegressionTol)
}

func TestEPRegression_MixedSkillTeams(t *testing.T) {
	p := DefaultPriors()
	// Team A : 1 top (30, 3), 1 average (25, 6), 1 newbie (20, 8)
	// Team B : 3 average (25, 6)
	a := []Gaussian{
		{Mu: 30, Sigma: 3}, {Mu: 25, Sigma: 6}, {Mu: 20, Sigma: 8},
	}
	b := []Gaussian{
		{Mu: 25, Sigma: 6}, {Mu: 25, Sigma: 6}, {Mu: 25, Sigma: 6},
	}
	m := TwoTeamMatch{TeamA: a, TeamB: b, ResultA: TeamWin}
	cfA, cfB, epA, epB := runBothAndCompare(t, m, p)
	assertTeamsMatch(t, "mixed teamA", epA, cfA, epRegressionTol)
	assertTeamsMatch(t, "uniform teamB", epB, cfB, epRegressionTol)
}

// TestEPRegression_SequentialMatches : enchaînement de matchs, état
// alimenté par les sorties. Vérifie que les deux implémentations restent
// numériquement alignées sur la durée (pas de dérive cumulée).
func TestEPRegression_SequentialMatches(t *testing.T) {
	p := DefaultPriors()
	cf := []Gaussian{p.NewPlayerState(), p.NewPlayerState()}
	ep := []Gaussian{p.NewPlayerState(), p.NewPlayerState()}

	results := []TeamResult{TeamWin, TeamLoss, TeamDraw, TeamWin, TeamLoss}
	for k, r := range results {
		mCF := TwoTeamMatch{TeamA: []Gaussian{cf[0]}, TeamB: []Gaussian{cf[1]}, ResultA: r}
		newCF_A, newCF_B, err := UpdateTwoTeam(mCF, p)
		if err != nil {
			t.Fatalf("match %d closed-form: %v", k, err)
		}
		cf = []Gaussian{newCF_A[0], newCF_B[0]}

		mEP := TwoTeamMatch{TeamA: []Gaussian{ep[0]}, TeamB: []Gaussian{ep[1]}, ResultA: r}
		newEP_A, newEP_B, err := UpdateTwoTeamEP(mEP, p)
		if err != nil {
			t.Fatalf("match %d EP: %v", k, err)
		}
		ep = []Gaussian{newEP_A[0], newEP_B[0]}

		assertGaussianClose(t, fmt.Sprintf("match %d player A", k), ep[0], cf[0], epRegressionTol)
		assertGaussianClose(t, fmt.Sprintf("match %d player B", k), ep[1], cf[1], epRegressionTol)
	}
}

func TestUpdateTwoTeamEP_EmptyTeam_Errors(t *testing.T) {
	p := DefaultPriors()
	if _, _, err := UpdateTwoTeamEP(TwoTeamMatch{
		TeamA: []Gaussian{}, TeamB: []Gaussian{{Mu: 25, Sigma: 8}},
	}, p); err == nil {
		t.Error("expected error on empty teamA")
	}
}

func TestUpdateTwoTeamEP_BetaZero_Errors(t *testing.T) {
	p := DefaultPriors()
	p.Beta = 0
	if _, _, err := UpdateTwoTeamEP(TwoTeamMatch{
		TeamA: []Gaussian{{Mu: 25, Sigma: 8}}, TeamB: []Gaussian{{Mu: 25, Sigma: 8}},
	}, p); err == nil {
		t.Error("expected error on Beta=0")
	}
}
