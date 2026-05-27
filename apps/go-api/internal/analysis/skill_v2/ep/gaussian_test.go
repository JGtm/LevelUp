package ep

import (
	"math"
	"testing"
)

const tol = 1e-9

func TestUniformGaussian(t *testing.T) {
	g := UniformGaussian()
	if !g.IsUniform() {
		t.Error("UniformGaussian should be uniform")
	}
	if g.Pi != 0 || g.Tau != 0 {
		t.Errorf("UniformGaussian = %v, want Pi=0 Tau=0", g)
	}
}

func TestFromMeanVariance_Roundtrip(t *testing.T) {
	cases := []struct {
		mu, variance float64
	}{
		{0, 1}, {25, 8.333 * 8.333}, {-5, 0.5},
	}
	for _, c := range cases {
		g, err := FromMeanVariance(c.mu, c.variance)
		if err != nil {
			t.Fatalf("FromMeanVariance(%v, %v): %v", c.mu, c.variance, err)
		}
		if math.Abs(g.Mu()-c.mu) > tol {
			t.Errorf("Mu roundtrip : got %v, want %v", g.Mu(), c.mu)
		}
		if math.Abs(g.Variance()-c.variance) > tol {
			t.Errorf("Variance roundtrip : got %v, want %v", g.Variance(), c.variance)
		}
	}
}

func TestFromMeanVariance_Invalid(t *testing.T) {
	if _, err := FromMeanVariance(0, 0); err == nil {
		t.Error("variance=0 should error")
	}
	if _, err := FromMeanVariance(0, -1); err == nil {
		t.Error("variance<0 should error")
	}
	if _, err := FromMeanVariance(math.NaN(), 1); err == nil {
		t.Error("mu NaN should error")
	}
}

func TestGaussian_Mul_Identity(t *testing.T) {
	g, _ := FromMeanVariance(5, 2)
	out := g.Mul(UniformGaussian())
	if math.Abs(out.Mu()-g.Mu()) > tol || math.Abs(out.Variance()-g.Variance()) > tol {
		t.Errorf("g * Uniform should equal g, got %v vs %v", out, g)
	}
	// Et symétrique : Uniform * g
	out2 := UniformGaussian().Mul(g)
	if math.Abs(out2.Mu()-g.Mu()) > tol || math.Abs(out2.Variance()-g.Variance()) > tol {
		t.Errorf("Uniform * g should equal g, got %v vs %v", out2, g)
	}
}

func TestGaussian_Mul_ProductOfNonUniform(t *testing.T) {
	// Produit de deux gaussiennes : formule classique
	//   π_3 = π_1 + π_2
	//   μ_3 = (μ_1·π_1 + μ_2·π_2) / π_3
	g1, _ := FromMeanVariance(0, 1)
	g2, _ := FromMeanVariance(2, 1)
	out := g1.Mul(g2)
	if math.Abs(out.Mu()-1.0) > tol {
		t.Errorf("μ = %v, want 1.0 (centered between 0 and 2)", out.Mu())
	}
	if math.Abs(out.Variance()-0.5) > tol {
		t.Errorf("variance = %v, want 0.5 (half)", out.Variance())
	}
}

func TestGaussian_Div_Inverse(t *testing.T) {
	// (a * b) / b == a
	a, _ := FromMeanVariance(5, 3)
	b, _ := FromMeanVariance(7, 2)
	out := a.Mul(b).Div(b)
	if math.Abs(out.Mu()-a.Mu()) > tol || math.Abs(out.Variance()-a.Variance()) > tol {
		t.Errorf("(a*b)/b = %v, want %v", out, a)
	}
}

func TestGaussian_Div_NegativePrecision_FallsBackToUniform(t *testing.T) {
	// Soustraire plus que ce qu'on a → uniform (clamp)
	a, _ := FromMeanVariance(0, 10)
	b, _ := FromMeanVariance(0, 0.1) // π beaucoup plus grand
	out := a.Div(b)
	if !out.IsUniform() {
		t.Errorf("Div with larger precision divisor should fallback to uniform, got %v", out)
	}
}

func TestGaussian_AbsoluteDifference(t *testing.T) {
	g1, _ := FromMeanVariance(0, 1)
	g2, _ := FromMeanVariance(0, 1)
	if d := g1.AbsoluteDifference(g2); d > tol {
		t.Errorf("equal gaussians should have diff ≈ 0, got %v", d)
	}
	g3, _ := FromMeanVariance(1, 1)
	if d := g1.AbsoluteDifference(g3); d <= 0 {
		t.Errorf("different μ should have diff > 0, got %v", d)
	}
}

func TestGaussian_UniformQueries(t *testing.T) {
	g := UniformGaussian()
	if g.Mu() != 0 {
		t.Errorf("Mu on uniform = %v, want 0 (convention)", g.Mu())
	}
	if !math.IsInf(g.Variance(), 1) {
		t.Errorf("Variance on uniform = %v, want +Inf", g.Variance())
	}
}
