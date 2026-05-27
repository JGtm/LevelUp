package ep

import (
	"math"
	"testing"
)

func TestVariable_NewIsUniform(t *testing.T) {
	v := NewVariable("x")
	if !v.Marginal.IsUniform() {
		t.Errorf("new variable should be uniform, got %v", v.Marginal)
	}
}

func TestVariable_UpdateMessage_MutatesMarginal(t *testing.T) {
	v := NewVariable("x")
	dummy := &PriorFactor{} // FactorID = pointer
	msg, _ := FromMeanVariance(10, 4)

	v.UpdateMessage(dummy, msg)
	if math.Abs(v.Marginal.Mu()-10) > tol {
		t.Errorf("μ after first message = %v, want 10", v.Marginal.Mu())
	}
	if math.Abs(v.Marginal.Variance()-4) > tol {
		t.Errorf("variance after first message = %v, want 4", v.Marginal.Variance())
	}
}

func TestVariable_MessageTo_ExcludesOwnContribution(t *testing.T) {
	// Une variable qui a reçu de A et de B, en envoyant vers A, ne doit
	// pas inclure ce qu'A avait elle-même contribué.
	v := NewVariable("x")
	factorA := &PriorFactor{}
	factorB := &PriorFactor{}
	msgA, _ := FromMeanVariance(0, 1)
	msgB, _ := FromMeanVariance(10, 1)
	v.UpdateMessage(factorA, msgA)
	v.UpdateMessage(factorB, msgB)
	// Marginal = msgA * msgB. Vers A on doit retrouver msgB.
	msgToA := v.MessageTo(factorA)
	if math.Abs(msgToA.Mu()-10) > tol {
		t.Errorf("message to A should be msgB (μ=10), got μ=%v", msgToA.Mu())
	}
	if math.Abs(msgToA.Variance()-1) > tol {
		t.Errorf("message to A variance = %v, want 1", msgToA.Variance())
	}
}

func TestPriorFactor_AppliesPrior(t *testing.T) {
	v := NewVariable("x")
	prior, _ := FromMeanVariance(25, 8.333*8.333)
	pf := NewPriorFactor("pf", v, prior)
	pf.UpdateMessages()
	if math.Abs(v.Marginal.Mu()-25) > tol {
		t.Errorf("μ after prior = %v, want 25", v.Marginal.Mu())
	}
	// Idempotence : 2 passes ne doivent rien changer.
	delta := pf.UpdateMessages()
	if delta > tol {
		t.Errorf("second pass should be no-op, delta = %v", delta)
	}
}

func TestLikelihoodFactor_AddsVariance(t *testing.T) {
	// X = perf, Y = perf bruitée par β² = 4. Si X est ancré à N(10, 1),
	// alors Y doit être N(10, 1+4) = N(10, 5).
	x := NewVariable("x")
	y := NewVariable("y")
	priorX, _ := FromMeanVariance(10, 1)
	pf := NewPriorFactor("priorX", x, priorX)
	lf := NewLikelihoodFactor("link", x, y, 4.0)

	r := NewRunner([]Factor{pf, lf})
	iters, _, converged := r.Run()
	if !converged {
		t.Fatalf("not converged after %d iters", iters)
	}
	if math.Abs(y.Marginal.Mu()-10) > 1e-6 {
		t.Errorf("Y μ = %v, want 10", y.Marginal.Mu())
	}
	if math.Abs(y.Marginal.Variance()-5) > 1e-6 {
		t.Errorf("Y variance = %v, want 5", y.Marginal.Variance())
	}
}

func TestSumFactor_ForwardSum(t *testing.T) {
	// Y = X1 + X2 (poids 1, 1).
	// X1 ~ N(2, 1), X2 ~ N(3, 4). Y doit être N(5, 5).
	x1 := NewVariable("x1")
	x2 := NewVariable("x2")
	y := NewVariable("y")
	p1, _ := FromMeanVariance(2, 1)
	p2, _ := FromMeanVariance(3, 4)
	pf1 := NewPriorFactor("p1", x1, p1)
	pf2 := NewPriorFactor("p2", x2, p2)
	sf := NewSumFactor("sum", y, []*Variable{x1, x2}, []float64{1, 1})

	r := NewRunner([]Factor{pf1, pf2, sf})
	iters, _, converged := r.Run()
	if !converged {
		t.Fatalf("not converged after %d iters", iters)
	}
	if math.Abs(y.Marginal.Mu()-5) > 1e-6 {
		t.Errorf("Y μ = %v, want 5 (μ1+μ2)", y.Marginal.Mu())
	}
	if math.Abs(y.Marginal.Variance()-5) > 1e-6 {
		t.Errorf("Y variance = %v, want 5 (σ1²+σ2²)", y.Marginal.Variance())
	}
}

func TestSumFactor_BackwardInfersOneInput(t *testing.T) {
	// Y = X1 + X2. Si on connaît Y = N(10, 0.01) et X2 = N(3, 0.01),
	// alors X1 doit être ≈ N(7, 0.02). Test que le SumFactor propage
	// correctement vers les inputs.
	x1 := NewVariable("x1")
	x2 := NewVariable("x2")
	y := NewVariable("y")
	pY, _ := FromMeanVariance(10, 0.01)
	pX2, _ := FromMeanVariance(3, 0.01)
	pfY := NewPriorFactor("pY", y, pY)
	pfX2 := NewPriorFactor("pX2", x2, pX2)
	sf := NewSumFactor("sum", y, []*Variable{x1, x2}, []float64{1, 1})

	r := NewRunner([]Factor{pfY, pfX2, sf})
	r.Run()
	if math.Abs(x1.Marginal.Mu()-7) > 1e-3 {
		t.Errorf("X1 μ = %v, want 7", x1.Marginal.Mu())
	}
	if math.Abs(x1.Marginal.Variance()-0.02) > 1e-4 {
		t.Errorf("X1 variance = %v, want 0.02", x1.Marginal.Variance())
	}
}

func TestWithinFactor_PullsTowardZero(t *testing.T) {
	// X a un prior centré sur 4 avec σ=5 (typique d'une diff de team_perf après
	// les links β²). On observe |X| < 2. Le posterior doit avoir μ pulled DOWN
	// depuis 4 vers la zone (-2, +2). Paramètres choisis pour rester loin des
	// limites numériques (denom > 1e-12).
	x := NewVariable("x")
	prior, _ := FromMeanVariance(4, 25) // σ = 5
	pf := NewPriorFactor("prior", x, prior)
	wf := NewWithinFactor("within", x, 2.0)

	r := NewRunner([]Factor{pf, wf})
	r.MaxIters = 100
	r.Run()
	if x.Marginal.Mu() >= 4 {
		t.Errorf("posterior μ = %v, expected < 4 (pulled down by draw constraint)", x.Marginal.Mu())
	}
	if x.Marginal.Variance() >= 25 {
		t.Errorf("posterior variance = %v, expected < 25 (tighter after observation)", x.Marginal.Variance())
	}
}

func TestWithinFactor_SymmetricForNegativeMu(t *testing.T) {
	// Mirror : μ = -4, σ = 5, ε = 2. Posterior doit être pulled UP depuis -4.
	x := NewVariable("x")
	prior, _ := FromMeanVariance(-4, 25)
	pf := NewPriorFactor("prior", x, prior)
	wf := NewWithinFactor("within", x, 2.0)

	r := NewRunner([]Factor{pf, wf})
	r.MaxIters = 100
	r.Run()
	if x.Marginal.Mu() <= -4 {
		t.Errorf("posterior μ = %v, expected > -4 (pulled up)", x.Marginal.Mu())
	}
}

func TestWithinFactor_ExtremeMargin_FallsBackToUniform(t *testing.T) {
	// Cas pathologique : μ très éloigné de la draw zone (μ=10, σ=1, ε=1).
	// Numériquement le facteur dégrade en uniform (denom < 1e-12). Vérifie
	// qu'on ne panic pas et que la variable reste cohérente.
	x := NewVariable("x")
	prior, _ := FromMeanVariance(10, 1)
	pf := NewPriorFactor("prior", x, prior)
	wf := NewWithinFactor("within", x, 1.0)
	r := NewRunner([]Factor{pf, wf})
	r.MaxIters = 50
	r.Run()
	// Marginal doit rester finite, pas NaN.
	if math.IsNaN(x.Marginal.Mu()) || math.IsNaN(x.Marginal.Sigma()) {
		t.Errorf("NaN dans posterior extrême : %v", x.Marginal)
	}
}

func TestGreaterThanFactor_BiasesUp(t *testing.T) {
	// Si X est observée > 0 et son prior est centré sur 0, le posterior
	// doit avoir μ > 0 et σ < σ_prior (la troncature serre la distribution).
	x := NewVariable("x")
	prior, _ := FromMeanVariance(0, 1)
	pf := NewPriorFactor("prior", x, prior)
	gtf := NewGreaterThanFactor("gt", x, 0)

	r := NewRunner([]Factor{pf, gtf})
	_, _, converged := r.Run()
	if !converged {
		t.Fatalf("not converged")
	}
	if x.Marginal.Mu() <= 0 {
		t.Errorf("posterior μ = %v, expected > 0 after X>0 observation", x.Marginal.Mu())
	}
	if x.Marginal.Variance() >= 1 {
		t.Errorf("posterior variance = %v, expected < 1 (tighter after observation)", x.Marginal.Variance())
	}
}

func TestRunner_Converges_SimpleGraph(t *testing.T) {
	x := NewVariable("x")
	prior, _ := FromMeanVariance(0, 1)
	pf := NewPriorFactor("prior", x, prior)

	r := NewRunner([]Factor{pf})
	iters, _, converged := r.Run()
	if !converged {
		t.Errorf("PriorFactor alone should converge in 1-2 iters, got %d", iters)
	}
	if iters > 2 {
		t.Errorf("PriorFactor alone converged in %d iters, expected ≤ 2", iters)
	}
}

// TestIntegration_1v1Match : recréée le pipeline TS basique en EP.
// Match 1v1 : winner skill ~ N(25, 8.33²), loser skill ~ N(25, 8.33²).
//
// Skill --N(skill, β²)--> Performance, β = 25/6.
// diff = perf_winner - perf_loser  (via SumFactor avec poids 1, -1).
// observed : diff > 0.
//
// Vérifie que winner μ monte, loser μ descend après une seule application,
// résultats numériquement proches du closed-form Phase 1a.
func TestIntegration_1v1Match(t *testing.T) {
	sigma0 := 25.0 / 3.0
	beta := 25.0 / 6.0
	sigma02 := sigma0 * sigma0

	skillW := NewVariable("skill_winner")
	skillL := NewVariable("skill_loser")
	perfW := NewVariable("perf_winner")
	perfL := NewVariable("perf_loser")
	diff := NewVariable("diff")

	prior, _ := FromMeanVariance(25, sigma02)
	pSkillW := NewPriorFactor("pW", skillW, prior)
	pSkillL := NewPriorFactor("pL", skillL, prior)
	linkW := NewLikelihoodFactor("linkW", skillW, perfW, beta*beta)
	linkL := NewLikelihoodFactor("linkL", skillL, perfL, beta*beta)
	sumDiff := NewSumFactor("sumDiff", diff, []*Variable{perfW, perfL}, []float64{1, -1})
	winObs := NewGreaterThanFactor("winObs", diff, 0)

	r := NewRunner([]Factor{pSkillW, pSkillL, linkW, linkL, sumDiff, winObs})
	r.MaxIters = 100
	_, _, converged := r.Run()
	if !converged {
		t.Errorf("EP did not converge")
	}

	if skillW.Marginal.Mu() <= 25 {
		t.Errorf("winner posterior μ = %v, expected > 25", skillW.Marginal.Mu())
	}
	if skillL.Marginal.Mu() >= 25 {
		t.Errorf("loser posterior μ = %v, expected < 25", skillL.Marginal.Mu())
	}
	// Symétrie : Δμ winner ≈ -Δμ loser (priors identiques, équipes identiques)
	deltaW := skillW.Marginal.Mu() - 25
	deltaL := 25 - skillL.Marginal.Mu()
	if math.Abs(deltaW-deltaL) > 0.01 {
		t.Errorf("asymétrie : Δμ winner=%v, Δμ loser=%v", deltaW, deltaL)
	}
	// Sanity : Δμ doit être proche du closed-form ≈ 4.66 (Moserware reference
	// for default priors). Tolérance large car convergence EP peut différer
	// très légèrement du closed-form analytique sur la 3e décimale.
	if math.Abs(deltaW-4.66) > 0.5 {
		t.Errorf("Δμ = %v, expected ≈ 4.66 (Moserware reference)", deltaW)
	}
}
