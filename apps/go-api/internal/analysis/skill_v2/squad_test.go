package skill_v2

import (
	"math"
	"testing"
)

func TestComputeSquadOffset_EmptyHistory(t *testing.T) {
	if got := ComputeSquadOffset(nil, 6.0); got != 0 {
		t.Errorf("0 match → offset 0, got %v", got)
	}
	if got := ComputeSquadOffset([]SquadCoMatch{}, 6.0); got != 0 {
		t.Errorf("slice vide → offset 0, got %v", got)
	}
}

func TestComputeSquadOffset_SystematicOverperformance(t *testing.T) {
	// 10 matchs où la paire gagne (Won=1) alors que le solo prédisait 0.75 :
	// résidu moyen = 0.25. Avec gain 6.0 → offset 1.5.
	matches := make([]SquadCoMatch, 10)
	for i := range matches {
		matches[i] = SquadCoMatch{Won: 1.0, SoloWinProb: 0.75}
	}
	got := ComputeSquadOffset(matches, 6.0)
	if math.Abs(got-1.5) > 1e-9 {
		t.Errorf("offset = %v, want 1.5 (résidu 0.25 × gain 6.0)", got)
	}
}

func TestComputeSquadOffset_ClampedAtCap(t *testing.T) {
	// 5 matchs avec un gros résidu (+0.5) et gain 10 → 5.0 brut, clampé à 2.0.
	matches := make([]SquadCoMatch, 5)
	for i := range matches {
		matches[i] = SquadCoMatch{Won: 1.0, SoloWinProb: 0.5}
	}
	got := ComputeSquadOffset(matches, 10.0)
	if got != SquadOffsetCap {
		t.Errorf("offset = %v, want clamp à %v", got, SquadOffsetCap)
	}
}

func TestComputeSquadOffset_NegativeSynergyClamped(t *testing.T) {
	// Paire qui SOUS-performe (perd alors que solo prédisait une victoire).
	matches := []SquadCoMatch{
		{Won: 0.0, SoloWinProb: 0.8},
		{Won: 0.0, SoloWinProb: 0.9},
	}
	got := ComputeSquadOffset(matches, 100.0) // gain énorme → clamp
	if got != -SquadOffsetCap {
		t.Errorf("offset = %v, want clamp à %v", got, -SquadOffsetCap)
	}
}

func TestClampSquadOffset(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{0, 0}, {1.5, 1.5}, {2.0, 2.0}, {2.5, 2.0}, {-2.5, -2.0}, {-1.0, -1.0},
	}
	for _, c := range cases {
		if got := ClampSquadOffset(c.in); got != c.want {
			t.Errorf("ClampSquadOffset(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestApplySquadOffset_ShiftsMuKeepsSigma(t *testing.T) {
	g := Gaussian{Mu: 25, Sigma: 4}
	out := ApplySquadOffset(g, 1.5)
	if out.Mu != 26.5 {
		t.Errorf("μ = %v, want 26.5", out.Mu)
	}
	if out.Sigma != 4 {
		t.Errorf("σ = %v, want 4 (inchangé)", out.Sigma)
	}
}
