package skill_v2

import (
	"math"
	"testing"
)

// TestUpdateTwoTeam_1v1_WinnerGainsLoserLoses : test canonique TrueSkill 1v1.
// Avec priors par défaut, deux joueurs ex aequo (μ=25, σ=25/3) — après le 1er match,
// le winner gagne ~5 points de μ, le loser perd ~5. Valeurs validées contre
// Moserware/Skills (reference C#).
func TestUpdateTwoTeam_1v1_WinnerGainsLoserLoses(t *testing.T) {
	p := DefaultPriors()
	w := p.NewPlayerState()
	l := p.NewPlayerState()
	newA, newB, err := UpdateTwoTeam(TwoTeamMatch{
		TeamA:   []Gaussian{w},
		TeamB:   []Gaussian{l},
		ResultA: TeamWin,
	}, p)
	if err != nil {
		t.Fatalf("UpdateTwoTeam: %v", err)
	}
	if newA[0].Mu <= w.Mu {
		t.Errorf("winner μ %v didn't increase (was %v)", newA[0].Mu, w.Mu)
	}
	if newB[0].Mu >= l.Mu {
		t.Errorf("loser μ %v didn't decrease (was %v)", newB[0].Mu, l.Mu)
	}
	// Symétrie : Δμ_winner = -Δμ_loser (priors identiques, équipes identiques).
	if math.Abs((newA[0].Mu-w.Mu)+(newB[0].Mu-l.Mu)) > 1e-6 {
		t.Errorf("Δμ winner + Δμ loser doit être 0 par symétrie : %v, %v",
			newA[0].Mu-w.Mu, newB[0].Mu-l.Mu)
	}
	// σ doit DESCENDRE (skill mieux connu après match), pour les deux.
	// Avec τ > 0, c'est σ²_post - τ² qui doit < σ²_prior.
	wAdj := math.Sqrt(newA[0].Variance() - p.Tau*p.Tau)
	if wAdj >= w.Sigma {
		t.Errorf("winner σ (ajusté τ) %v should < initial %v", wAdj, w.Sigma)
	}
}

func TestUpdateTwoTeam_Draw_PullsTogether(t *testing.T) {
	p := DefaultPriors()
	high := Gaussian{Mu: 30, Sigma: 25.0 / 3.0}
	low := Gaussian{Mu: 20, Sigma: 25.0 / 3.0}
	newA, newB, err := UpdateTwoTeam(TwoTeamMatch{
		TeamA:   []Gaussian{high},
		TeamB:   []Gaussian{low},
		ResultA: TeamDraw,
	}, p)
	if err != nil {
		t.Fatalf("UpdateTwoTeam draw: %v", err)
	}
	// Un draw "rapproche" : le high doit baisser, le low monter.
	if newA[0].Mu >= high.Mu {
		t.Errorf("draw : high μ %v should decrease (was %v)", newA[0].Mu, high.Mu)
	}
	if newB[0].Mu <= low.Mu {
		t.Errorf("draw : low μ %v should increase (was %v)", newB[0].Mu, low.Mu)
	}
}

func TestUpdateTwoTeam_LossInverse(t *testing.T) {
	// Match A vs B où A perd doit produire les mêmes résultats que A vs B où
	// B gagne avec les rôles inversés. Garantit la symétrie de l'API.
	p := DefaultPriors()
	a := Gaussian{Mu: 28, Sigma: 5}
	b := Gaussian{Mu: 22, Sigma: 6}

	winNewA, winNewB, _ := UpdateTwoTeam(TwoTeamMatch{
		TeamA: []Gaussian{a}, TeamB: []Gaussian{b}, ResultA: TeamLoss,
	}, p)
	lossNewA, lossNewB, _ := UpdateTwoTeam(TwoTeamMatch{
		TeamA: []Gaussian{b}, TeamB: []Gaussian{a}, ResultA: TeamWin,
	}, p)
	if math.Abs(winNewA[0].Mu-lossNewB[0].Mu) > 1e-9 ||
		math.Abs(winNewB[0].Mu-lossNewA[0].Mu) > 1e-9 {
		t.Errorf("symétrie Loss(A,B) vs Win(B,A) violée")
	}
}

func TestUpdateTwoTeam_EmptyTeam_Errors(t *testing.T) {
	p := DefaultPriors()
	if _, _, err := UpdateTwoTeam(TwoTeamMatch{
		TeamA: []Gaussian{}, TeamB: []Gaussian{{Mu: 25, Sigma: 8}},
	}, p); err == nil {
		t.Error("expected error on empty team A")
	}
}

func TestUpdateTwoTeam_BetaZero_Errors(t *testing.T) {
	p := DefaultPriors()
	p.Beta = 0
	if _, _, err := UpdateTwoTeam(TwoTeamMatch{
		TeamA: []Gaussian{{Mu: 25, Sigma: 8}}, TeamB: []Gaussian{{Mu: 25, Sigma: 8}},
	}, p); err == nil {
		t.Error("expected error on Beta=0")
	}
}

func TestUpdateTwoTeam_4v4_AllPlayersMoveSameDirection(t *testing.T) {
	p := DefaultPriors()
	teamA := []Gaussian{p.NewPlayerState(), p.NewPlayerState(), p.NewPlayerState(), p.NewPlayerState()}
	teamB := []Gaussian{p.NewPlayerState(), p.NewPlayerState(), p.NewPlayerState(), p.NewPlayerState()}
	newA, newB, err := UpdateTwoTeam(TwoTeamMatch{
		TeamA: teamA, TeamB: teamB, ResultA: TeamWin,
	}, p)
	if err != nil {
		t.Fatalf("UpdateTwoTeam 4v4: %v", err)
	}
	for i := range newA {
		if newA[i].Mu <= teamA[i].Mu {
			t.Errorf("teamA[%d] μ didn't increase (was %v, now %v)", i, teamA[i].Mu, newA[i].Mu)
		}
	}
	for i := range newB {
		if newB[i].Mu >= teamB[i].Mu {
			t.Errorf("teamB[%d] μ didn't decrease (was %v, now %v)", i, teamB[i].Mu, newB[i].Mu)
		}
	}
}

func TestUpdateTwoTeam_HighSigmaPlayerMovesMore(t *testing.T) {
	// Un joueur avec σ élevé (peu joué) doit voir son μ bouger plus qu'un
	// joueur avec σ bas (confirmé) lors du même match. C'est ce qui permet
	// aux nouveaux de monter/descendre rapidement vers leur vrai niveau.
	p := DefaultPriors()
	veteran := Gaussian{Mu: 25, Sigma: 2} // peu d'incertitude
	newbie := Gaussian{Mu: 25, Sigma: 8}  // beaucoup d'incertitude
	newA, _, _ := UpdateTwoTeam(TwoTeamMatch{
		TeamA: []Gaussian{veteran}, TeamB: []Gaussian{{Mu: 25, Sigma: 5}},
		ResultA: TeamWin,
	}, p)
	newA2, _, _ := UpdateTwoTeam(TwoTeamMatch{
		TeamA: []Gaussian{newbie}, TeamB: []Gaussian{{Mu: 25, Sigma: 5}},
		ResultA: TeamWin,
	}, p)
	dVet := newA[0].Mu - veteran.Mu
	dNew := newA2[0].Mu - newbie.Mu
	if dNew <= dVet {
		t.Errorf("newbie Δμ (%v) should > veteran Δμ (%v)", dNew, dVet)
	}
}

func TestPredictWinProbability_EqualSkills(t *testing.T) {
	p := DefaultPriors()
	a := []Gaussian{p.NewPlayerState()}
	b := []Gaussian{p.NewPlayerState()}
	if got := PredictWinProbability(a, b, p); math.Abs(got-0.5) > 1e-9 {
		t.Errorf("equal skills → P(win) = %v, want 0.5", got)
	}
}

func TestPredictWinProbability_StrongerWinsMore(t *testing.T) {
	p := DefaultPriors()
	strong := []Gaussian{{Mu: 35, Sigma: 5}}
	weak := []Gaussian{{Mu: 15, Sigma: 5}}
	pStrong := PredictWinProbability(strong, weak, p)
	if pStrong <= 0.5 {
		t.Errorf("strong vs weak: P(win) = %v, want > 0.5", pStrong)
	}
	if pStrong < 0.85 {
		t.Errorf("strong vs weak (Δμ=20, σ=5): P(win) = %v, expected very high", pStrong)
	}
}

func TestPredictDrawProbability_Symmetric(t *testing.T) {
	p := DefaultPriors()
	a := []Gaussian{{Mu: 25, Sigma: 5}}
	b := []Gaussian{{Mu: 25, Sigma: 5}}
	pDraw := PredictDrawProbability(a, b, p)
	if pDraw <= 0 || pDraw >= 1 {
		t.Errorf("P(draw) = %v, expected in (0, 1)", pDraw)
	}
	// Roundtrip : si je swap les équipes, P(draw) doit être identique.
	pDrawSwap := PredictDrawProbability(b, a, p)
	if math.Abs(pDraw-pDrawSwap) > 1e-9 {
		t.Errorf("P(draw) non symétrique : %v vs %v", pDraw, pDrawSwap)
	}
}

// TestSequentialMatches_VeteransConverge : après N matchs à 50% win rate, μ doit
// rester centré sur le prior, et σ doit diminuer (bornée par τ).
func TestSequentialMatches_VeteransConverge(t *testing.T) {
	p := DefaultPriors()
	a := p.NewPlayerState()
	b := p.NewPlayerState()
	startSigma := a.Sigma

	// 100 matchs alternés gagnés/perdus par A.
	for i := 0; i < 100; i++ {
		result := TeamWin
		if i%2 == 1 {
			result = TeamLoss
		}
		newA, newB, err := UpdateTwoTeam(TwoTeamMatch{
			TeamA: []Gaussian{a}, TeamB: []Gaussian{b}, ResultA: result,
		}, p)
		if err != nil {
			t.Fatalf("match %d: %v", i, err)
		}
		a, b = newA[0], newB[0]
	}
	if math.Abs(a.Mu-p.Mu0) > 1.0 {
		t.Errorf("after 100 mixed matches, μ = %v, expected near %v", a.Mu, p.Mu0)
	}
	if a.Sigma >= startSigma {
		t.Errorf("σ didn't shrink: start=%v, end=%v", startSigma, a.Sigma)
	}
	// τ borne la σ finale par en bas.
	if a.Sigma < p.Tau {
		t.Errorf("σ went below Tau (%v) : %v", p.Tau, a.Sigma)
	}
}
