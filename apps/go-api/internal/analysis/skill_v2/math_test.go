package skill_v2

import (
	"math"
	"testing"
)

const tol = 1e-9

func TestStdNormalPDF_KnownValues(t *testing.T) {
	// PDF(0) = 1/sqrt(2π) ≈ 0.3989422804
	if got := stdNormalPDF(0); math.Abs(got-0.39894228040143268) > tol {
		t.Errorf("pdf(0) = %v", got)
	}
	// PDF symétrique
	if math.Abs(stdNormalPDF(-1.5)-stdNormalPDF(1.5)) > tol {
		t.Errorf("pdf non symétrique")
	}
}

func TestStdNormalCDF_KnownValues(t *testing.T) {
	if got := stdNormalCDF(0); math.Abs(got-0.5) > tol {
		t.Errorf("cdf(0) = %v, want 0.5", got)
	}
	// cdf(-∞) → 0, cdf(+∞) → 1
	if got := stdNormalCDF(-10); got > 1e-20 {
		t.Errorf("cdf(-10) = %v, expected near 0", got)
	}
	if got := stdNormalCDF(10); 1-got > 1e-20 {
		t.Errorf("cdf(10) = %v, expected near 1", got)
	}
	// Φ(1) ≈ 0.8413447
	if got := stdNormalCDF(1); math.Abs(got-0.8413447460685429) > 1e-6 {
		t.Errorf("cdf(1) = %v", got)
	}
}

func TestStdNormalInvCDF_Roundtrip(t *testing.T) {
	for _, p := range []float64{0.01, 0.1, 0.25, 0.5, 0.75, 0.9, 0.99} {
		x := stdNormalInvCDF(p)
		back := stdNormalCDF(x)
		if math.Abs(back-p) > 1e-6 {
			t.Errorf("roundtrip p=%v → x=%v → p'=%v", p, x, back)
		}
	}
}

func TestVWin_KnownValues(t *testing.T) {
	// v_win(0, 0) = pdf(0)/cdf(0) = 0.3989... / 0.5 = 0.7978845...
	if got := vWin(0, 0); math.Abs(got-0.7978845608028654) > 1e-9 {
		t.Errorf("vWin(0, 0) = %v", got)
	}
	// v_win(t, 0) > 0 toujours (winner gagne du μ)
	for _, tt := range []float64{-2, -1, 0, 1, 2} {
		if got := vWin(tt, 0); got <= 0 {
			t.Errorf("vWin(%v, 0) = %v, expected > 0", tt, got)
		}
	}
}

func TestWWin_BoundedInZeroOne(t *testing.T) {
	for _, tt := range []float64{-5, -2, 0, 2, 5} {
		for _, eps := range []float64{0, 0.1, 0.5, 1} {
			w := wWin(tt, eps)
			if w < 0 || w > 1 {
				t.Errorf("wWin(%v, %v) = %v out of [0,1]", tt, eps, w)
			}
		}
	}
}

func TestVDraw_SignFollowsT(t *testing.T) {
	// vDraw a le signe opposé à t (le joueur dont la perf était au-dessus
	// "redescend" vers la moyenne, et inversement). Cohérent avec
	// Moserware/Skills/Internal/TruncatedGaussianCorrectionFunctions.cs.
	if v := vDraw(2.0, 0.5); v >= 0 {
		t.Errorf("vDraw(2.0, 0.5) = %v, expected < 0 (positive t pulls down)", v)
	}
	if v := vDraw(-2.0, 0.5); v <= 0 {
		t.Errorf("vDraw(-2.0, 0.5) = %v, expected > 0 (negative t pulls up)", v)
	}
}

func TestDrawMargin_Monotonicity(t *testing.T) {
	beta := 25.0 / 6.0
	prev := 0.0
	for _, p := range []float64{0.01, 0.05, 0.1, 0.2, 0.4} {
		eps := DrawMargin(p, 2, 2, beta)
		if eps <= prev {
			t.Errorf("DrawMargin pas monotone : p=%v → ε=%v (prev=%v)", p, eps, prev)
		}
		prev = eps
	}
}

func TestDrawMargin_ZeroProb(t *testing.T) {
	if got := DrawMargin(0, 2, 2, 1.0); got != 0 {
		t.Errorf("DrawMargin(0,...) = %v, want 0", got)
	}
}

func TestDrawMargin_ScalesWithTeamSize(t *testing.T) {
	// ε scale en sqrt(n_a + n_b) : doubler les équipes doit multiplier ε
	// par sqrt(2) ≈ 1.414.
	eps22 := DrawMargin(0.10, 2, 2, 5.0)
	eps44 := DrawMargin(0.10, 4, 4, 5.0)
	ratio := eps44 / eps22
	if math.Abs(ratio-math.Sqrt(2.0)) > 1e-6 {
		t.Errorf("ratio ε(4,4)/ε(2,2) = %v, want sqrt(2)", ratio)
	}
}
