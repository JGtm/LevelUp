package skill_v2

import (
	"math"
	"testing"
)

// trueskill_ep_weight_test.go : TS2 — pondération temps-joué du sum-factor
// team_perf = Σ wᵢ·perfᵢ (cf. ep/sum_factor.go, branchement match2team).
//
// Un joueur ayant joué une fraction du match (wᵢ < 1) contribue moins à la
// perf d'équipe ET reçoit un Δμ plus faible (le message inverse du SumFactor a
// une variance ∝ 1/wᵢ² → moins informatif → le prior domine).

func TestTS2_TimePlayedWeight_PartialPlayerMovesLess(t *testing.T) {
	p := DefaultPriors()
	prior := p.NewPlayerState()

	// 2v2, priors identiques, Team A gagne.
	mk := func() TwoTeamMatch {
		return TwoTeamMatch{
			TeamA:   []Gaussian{p.NewPlayerState(), p.NewPlayerState()},
			TeamB:   []Gaussian{p.NewPlayerState(), p.NewPlayerState()},
			ResultA: TeamWin,
		}
	}

	// A0 plein temps (w=1), A1 a joué 20% (w=0.2). Pas de counts kills/deaths
	// pour isoler l'effet du poids.
	counts := &CountInputs{
		TeamA: []PlayerCounts{{Weight: 1.0}, {Weight: 0.2}},
		TeamB: []PlayerCounts{{Weight: 1.0}, {Weight: 1.0}},
	}
	gotA, _, err := UpdateTwoTeamWithCountsEP(mk(), counts, p)
	if err != nil {
		t.Fatalf("UpdateTwoTeamWithCountsEP: %v", err)
	}

	d0 := gotA[0].Mu - prior.Mu // gain du joueur plein temps
	d1 := gotA[1].Mu - prior.Mu // gain du joueur partiel
	if d0 <= 0 {
		t.Fatalf("A0 (plein temps) devrait gagner du μ après une victoire, Δ=%v", d0)
	}
	if d1 <= 0 {
		t.Errorf("A1 (partiel) devrait quand même gagner un peu, Δ=%v", d1)
	}
	if d1 >= d0 {
		t.Errorf("A1 (w=0.2) devrait bouger MOINS que A0 (w=1) : Δ0=%v, Δ1=%v", d0, d1)
	}

	// Sanity : poids égaux → mouvements égaux (équivalent TS classique).
	eq := &CountInputs{
		TeamA: []PlayerCounts{{Weight: 1.0}, {Weight: 1.0}},
		TeamB: []PlayerCounts{{Weight: 1.0}, {Weight: 1.0}},
	}
	gotEq, _, err := UpdateTwoTeamWithCountsEP(mk(), eq, p)
	if err != nil {
		t.Fatalf("UpdateTwoTeamWithCountsEP eq: %v", err)
	}
	if math.Abs(gotEq[0].Mu-gotEq[1].Mu) > 1e-6 {
		t.Errorf("poids égaux : A0=%v, A1=%v devraient être identiques", gotEq[0].Mu, gotEq[1].Mu)
	}
}
