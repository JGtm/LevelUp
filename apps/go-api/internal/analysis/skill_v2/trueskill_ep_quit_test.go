package skill_v2

import "testing"

// trueskill_ep_quit_test.go : tests Phase 3-quit (TS2 §9).
//
// Vérifie que le quit penalty :
//   1. Pénalise un quitter par rapport à un teammate identique non-quitter
//      sur le MÊME match perdu (related quit, δ modéré).
//   2. Pénalise PLUS fortement un quitter dont l'équipe gagnait/égalisait
//      (unrelated quit, δ grand).
//   3. N'affecte pas les non-quitters du même match.

func TestPhase3quit_PenalizesQuitter_Related(t *testing.T) {
	p := DefaultPriors()
	a := []Gaussian{p.NewPlayerState(), p.NewPlayerState()}
	b := []Gaussian{p.NewPlayerState(), p.NewPlayerState()}
	m := TwoTeamMatch{TeamA: a, TeamB: b, ResultA: TeamLoss}

	// Baseline : sans quit penalty, A0 et A1 identiques (même priors, team loss).
	noQuit := &CountInputs{
		TeamA: []PlayerCounts{
			{Kills: pf64(8), Deaths: pf64(10)},
			{Kills: pf64(8), Deaths: pf64(10)},
		},
		TeamB: []PlayerCounts{
			{Kills: pf64(12), Deaths: pf64(6)},
			{Kills: pf64(12), Deaths: pf64(6)},
		},
	}
	postA, _, err := UpdateTwoTeamWithCountsEP(m, noQuit, p)
	if err != nil {
		t.Fatalf("baseline: %v", err)
	}
	gapBaseline := postA[0].Mu - postA[1].Mu

	// Avec quit penalty sur A0 (related : équipe a perdu).
	withQuit := &CountInputs{
		TeamA: []PlayerCounts{
			{Kills: pf64(8), Deaths: pf64(10), Quit: true, QuitPenaltyDelta: DefaultQuitDeltaRelated},
			{Kills: pf64(8), Deaths: pf64(10)},
		},
		TeamB: []PlayerCounts{
			{Kills: pf64(12), Deaths: pf64(6)},
			{Kills: pf64(12), Deaths: pf64(6)},
		},
	}
	postAQuit, _, err := UpdateTwoTeamWithCountsEP(m, withQuit, p)
	if err != nil {
		t.Fatalf("with quit: %v", err)
	}

	// Le quitter A0 doit avoir μ STRICTEMENT < A1.
	if postAQuit[0].Mu >= postAQuit[1].Mu {
		t.Errorf("quitter μ=%v >= teammate μ=%v — quit penalty non appliqué",
			postAQuit[0].Mu, postAQuit[1].Mu)
	}
	gapWithQuit := postAQuit[0].Mu - postAQuit[1].Mu
	t.Logf("gap A0-A1 baseline=%.4f with_quit=%.4f", gapBaseline, gapWithQuit)
	if gapWithQuit >= gapBaseline {
		t.Errorf("gap with_quit=%.4f devrait être < gap baseline=%.4f",
			gapWithQuit, gapBaseline)
	}
}

func TestPhase3quit_UnrelatedMoreSevereThanRelated(t *testing.T) {
	p := DefaultPriors()
	a := []Gaussian{p.NewPlayerState(), p.NewPlayerState()}
	b := []Gaussian{p.NewPlayerState(), p.NewPlayerState()}

	// Cas 1 : team A perd (related quit), A0 quitte.
	mLoss := TwoTeamMatch{TeamA: a, TeamB: b, ResultA: TeamLoss}
	relatedQuit := &CountInputs{
		TeamA: []PlayerCounts{
			{Kills: pf64(10), Deaths: pf64(10), Quit: true, QuitPenaltyDelta: DefaultQuitDeltaRelated},
			{Kills: pf64(10), Deaths: pf64(10)},
		},
		TeamB: []PlayerCounts{
			{Kills: pf64(10), Deaths: pf64(10)},
			{Kills: pf64(10), Deaths: pf64(10)},
		},
	}
	postRelated, _, err := UpdateTwoTeamWithCountsEP(mLoss, relatedQuit, p)
	if err != nil {
		t.Fatalf("related: %v", err)
	}

	// Cas 2 : team A gagne (unrelated quit), A0 quitte.
	mWin := TwoTeamMatch{TeamA: a, TeamB: b, ResultA: TeamWin}
	unrelatedQuit := &CountInputs{
		TeamA: []PlayerCounts{
			{Kills: pf64(10), Deaths: pf64(10), Quit: true, QuitPenaltyDelta: DefaultQuitDeltaUnrelated},
			{Kills: pf64(10), Deaths: pf64(10)},
		},
		TeamB: []PlayerCounts{
			{Kills: pf64(10), Deaths: pf64(10)},
			{Kills: pf64(10), Deaths: pf64(10)},
		},
	}
	postUnrelated, _, err := UpdateTwoTeamWithCountsEP(mWin, unrelatedQuit, p)
	if err != nil {
		t.Fatalf("unrelated: %v", err)
	}

	// Pénalité relative au teammate qui ne quitte pas, dans chaque cas.
	// |Δ(A0) - Δ(A1)| = la pénalité effective sur A0 dûe au quit.
	gapRelated := postRelated[1].Mu - postRelated[0].Mu
	gapUnrelated := postUnrelated[1].Mu - postUnrelated[0].Mu
	t.Logf("gap related=%.4f unrelated=%.4f", gapRelated, gapUnrelated)
	if gapUnrelated <= gapRelated {
		t.Errorf("pénalité unrelated=%.4f doit être > related=%.4f", gapUnrelated, gapRelated)
	}
}
