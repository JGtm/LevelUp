package skill_v2

import (
	"math"
	"testing"
)

// approxSumToOne vérifie que (probA, probDraw, probB) forment bien une loi de
// probabilité (somme ≈ 1, chaque terme dans [0,1]).
func assertValidProbTriple(t *testing.T, probA, probDraw, probB float64) {
	t.Helper()
	for name, p := range map[string]float64{"probA": probA, "probDraw": probDraw, "probB": probB} {
		if p < 0 || p > 1 {
			t.Errorf("%s = %v hors [0,1]", name, p)
		}
	}
	if sum := probA + probDraw + probB; math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("somme = %v, attendu ≈ 1", sum)
	}
}

func TestPredictTwoTeamWinProb_Balanced(t *testing.T) {
	p := DefaultPriors()
	// Deux équipes identiques → probA ≈ probB, et un peu de masse sur le draw.
	teamA := []Gaussian{{Mu: 25, Sigma: 3}, {Mu: 25, Sigma: 3}}
	teamB := []Gaussian{{Mu: 25, Sigma: 3}, {Mu: 25, Sigma: 3}}
	probA, probDraw, probB := PredictTwoTeamWinProb(teamA, teamB, p)
	assertValidProbTriple(t, probA, probDraw, probB)
	if math.Abs(probA-probB) > 1e-9 {
		t.Errorf("équipes identiques : probA=%v != probB=%v", probA, probB)
	}
	if probDraw <= 0 {
		t.Errorf("DrawProbability > 0 attendu mais probDraw=%v", probDraw)
	}
}

func TestPredictTwoTeamWinProb_StrongFavorite(t *testing.T) {
	p := DefaultPriors()
	// A nettement plus fort (μ=30 vs 20), faible incertitude → A > 90%.
	teamA := []Gaussian{{Mu: 30, Sigma: 1}}
	teamB := []Gaussian{{Mu: 20, Sigma: 1}}
	probA, probDraw, probB := PredictTwoTeamWinProb(teamA, teamB, p)
	assertValidProbTriple(t, probA, probDraw, probB)
	if probA <= 0.9 {
		t.Errorf("favori net : probA=%v, attendu > 0.9", probA)
	}
	if probA <= probB {
		t.Errorf("probA=%v doit dominer probB=%v", probA, probB)
	}
}

func TestPredictTwoTeamWinProb_HighSigmaPullsToFifty(t *testing.T) {
	p := DefaultPriors()
	// Même écart de μ (30 vs 20) mais grosse incertitude → proba tirée vers 0.5.
	lowSigmaA, _, _ := PredictTwoTeamWinProb(
		[]Gaussian{{Mu: 30, Sigma: 1}}, []Gaussian{{Mu: 20, Sigma: 1}}, p)
	highSigmaA, _, _ := PredictTwoTeamWinProb(
		[]Gaussian{{Mu: 30, Sigma: 8}}, []Gaussian{{Mu: 20, Sigma: 8}}, p)
	if !(highSigmaA < lowSigmaA) {
		t.Errorf("forte incertitude doit rapprocher de 0.5 : highSigma=%v >= lowSigma=%v",
			highSigmaA, lowSigmaA)
	}
	if highSigmaA <= 0.5 {
		t.Errorf("A reste favori même incertain : highSigmaA=%v, attendu > 0.5", highSigmaA)
	}
}

func TestPredictTwoTeamWinProb_DrawProbabilityRaisesDrawMass(t *testing.T) {
	balanced := func(drawProb float64) float64 {
		p := DefaultPriors()
		p.DrawProbability = drawProb
		_, probDraw, _ := PredictTwoTeamWinProb(
			[]Gaussian{{Mu: 25, Sigma: 3}}, []Gaussian{{Mu: 25, Sigma: 3}}, p)
		return probDraw
	}
	low := balanced(0.05)
	high := balanced(0.30)
	if !(high > low) {
		t.Errorf("DrawProbability plus haut doit augmenter probDraw : high=%v <= low=%v", high, low)
	}
	if low <= 0 {
		t.Errorf("DrawProbability=0.05 > 0 → probDraw attendu > 0, got %v", low)
	}
}

func TestPredictTwoTeamWinProb_EmptyTeamNeutral(t *testing.T) {
	p := DefaultPriors()
	probA, probDraw, probB := PredictTwoTeamWinProb(nil, []Gaussian{{Mu: 25, Sigma: 3}}, p)
	if probA != 0.5 || probB != 0.5 || probDraw != 0 {
		t.Errorf("équipe vide → (0.5, 0, 0.5), got (%v, %v, %v)", probA, probDraw, probB)
	}
}

// TestPredictTwoTeamWinProb_ConsistentWithLegacyHelpers vérifie que la nouvelle
// fonction reste cohérente avec PredictWinProbability (win seul) et
// PredictDrawProbability (draw seul) post-refactor matchSpread.
func TestPredictTwoTeamWinProb_ConsistentWithLegacyHelpers(t *testing.T) {
	p := DefaultPriors()
	teamA := []Gaussian{{Mu: 28, Sigma: 4}}
	teamB := []Gaussian{{Mu: 22, Sigma: 5}}
	_, probDraw, _ := PredictTwoTeamWinProb(teamA, teamB, p)
	// On vérifie la cohérence directionnelle de PredictWinProbability +
	// l'égalité exacte de probDraw avec PredictDrawProbability.
	if got := PredictDrawProbability(teamA, teamB, p); math.Abs(got-probDraw) > 1e-12 {
		t.Errorf("PredictDrawProbability=%v != probDraw=%v", got, probDraw)
	}
	if win := PredictWinProbability(teamA, teamB, p); win <= 0.5 {
		t.Errorf("A favori : PredictWinProbability=%v attendu > 0.5", win)
	}
}
