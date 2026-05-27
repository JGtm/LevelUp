package skill_v2

import (
	"math"
	"testing"
)

func TestNewGaussian_Valid(t *testing.T) {
	g, err := NewGaussian(25.0, 8.333)
	if err != nil {
		t.Fatalf("NewGaussian: %v", err)
	}
	if g.Mu != 25.0 || math.Abs(g.Sigma-8.333) > 1e-9 {
		t.Errorf("got %v, want N(25, 8.333)", g)
	}
}

func TestNewGaussian_Invalid(t *testing.T) {
	cases := []struct {
		name      string
		mu, sigma float64
	}{
		{"mu NaN", math.NaN(), 1},
		{"mu Inf", math.Inf(1), 1},
		{"sigma NaN", 0, math.NaN()},
		{"sigma negative", 0, -0.1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := NewGaussian(c.mu, c.sigma); err == nil {
				t.Errorf("expected error for (%v, %v)", c.mu, c.sigma)
			}
		})
	}
}

func TestGaussian_ConservativeRating(t *testing.T) {
	g := Gaussian{Mu: 25, Sigma: 5}
	if got := g.ConservativeRating(3); math.Abs(got-10) > 1e-9 {
		t.Errorf("ConservativeRating(3) = %v, want 10", got)
	}
	if got := g.ConservativeRating(0); got != 25 {
		t.Errorf("ConservativeRating(0) = %v, want 25", got)
	}
}

func TestDefaultPriors_Sane(t *testing.T) {
	p := DefaultPriors()
	if p.Mu0 != 25.0 {
		t.Errorf("Mu0 = %v, want 25", p.Mu0)
	}
	if math.Abs(p.Sigma0-25.0/3.0) > 1e-9 {
		t.Errorf("Sigma0 = %v, want 25/3", p.Sigma0)
	}
	if math.Abs(p.Beta-p.Sigma0/2.0) > 1e-9 {
		t.Errorf("Beta should be Sigma0/2")
	}
	if math.Abs(p.Tau-p.Sigma0/100.0) > 1e-9 {
		t.Errorf("Tau should be Sigma0/100")
	}
	if p.DrawProbability <= 0 || p.DrawProbability >= 1 {
		t.Errorf("DrawProbability = %v, expected within (0, 1)", p.DrawProbability)
	}
}

func TestPriors_NewPlayerState(t *testing.T) {
	p := DefaultPriors()
	g := p.NewPlayerState()
	if g.Mu != p.Mu0 || g.Sigma != p.Sigma0 {
		t.Errorf("NewPlayerState = %v, want N(%v, %v)", g, p.Mu0, p.Sigma0)
	}
}
