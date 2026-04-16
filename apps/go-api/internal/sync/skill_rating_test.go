// Package sync — skill_rating_test.go : tests unitaires pour les fonctions TrueSkill pures.
//
// Note : ce package importe DuckDB transitif → ne compile pas sur Windows (contrainte
// build constraint windows-amd64). Ces tests sont conçus pour tourner en CI Linux.
package sync

import (
	"math"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// standardNormalPDF / standardNormalCDF
// ─────────────────────────────────────────────────────────────────────────────

func TestStandardNormalPDF_Zero(t *testing.T) {
	got := standardNormalPDF(0)
	want := 1.0 / math.Sqrt(2*math.Pi)
	if math.Abs(got-want) > 1e-10 {
		t.Errorf("PDF(0): got %v, want %v", got, want)
	}
}

func TestStandardNormalCDF_Zero(t *testing.T) {
	got := standardNormalCDF(0)
	if math.Abs(got-0.5) > 1e-10 {
		t.Errorf("CDF(0): got %v, want 0.5", got)
	}
}

func TestStandardNormalCDF_Symmetry(t *testing.T) {
	// CDF(-x) = 1 - CDF(x)
	for _, x := range []float64{0.5, 1.0, 1.96, 3.0} {
		a := standardNormalCDF(x)
		b := standardNormalCDF(-x)
		if math.Abs(a+b-1.0) > 1e-10 {
			t.Errorf("CDF(%v)+CDF(-%v) = %v, want 1.0", x, x, a+b)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// vWin / wWin
// ─────────────────────────────────────────────────────────────────────────────

func TestVWin_PositiveT(t *testing.T) {
	// t > 0 → vWin doit être positif (facteur de mise à jour de la moyenne)
	v := vWin(1.0, 0.0)
	if v <= 0 {
		t.Errorf("vWin(1.0, 0.0) = %v, expected > 0", v)
	}
}

func TestWWin_InRange(t *testing.T) {
	// wWin doit être dans (0, 1)
	w := wWin(1.0, 0.0)
	if w <= 0 || w >= 1 {
		t.Errorf("wWin(1.0, 0.0) = %v, expected in (0, 1)", w)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// trueskillUpdate
// ─────────────────────────────────────────────────────────────────────────────

func TestTrueskillUpdate_Winner(t *testing.T) {
	// Un joueur qui gagne (actualScore > 0.5) doit voir sa mu augmenter.
	mu, sigma := 25.0, 8.333
	muOpp, sigmaOpp := 25.0, 8.333
	newMu, newSigma := trueskillUpdate(mu, sigma, muOpp, sigmaOpp, 1.0, 1.0)

	if newMu <= mu {
		t.Errorf("gagnant: mu devrait augmenter, got %v (was %v)", newMu, mu)
	}
	if newSigma >= sigma {
		t.Errorf("sigma devrait diminuer, got %v (was %v)", newSigma, sigma)
	}
}

func TestTrueskillUpdate_Loser(t *testing.T) {
	// Un joueur qui perd (actualScore = 0.0) doit voir sa mu diminuer.
	mu, sigma := 25.0, 8.333
	muOpp, sigmaOpp := 25.0, 8.333
	newMu, newSigma := trueskillUpdate(mu, sigma, muOpp, sigmaOpp, 0.0, 1.0)

	if newMu >= mu {
		t.Errorf("perdant: mu devrait diminuer, got %v (was %v)", newMu, mu)
	}
	if newSigma >= sigma {
		t.Errorf("sigma devrait diminuer, got %v (was %v)", newSigma, sigma)
	}
}

func TestTrueskillUpdate_SigmaPositive(t *testing.T) {
	// sigma ne doit jamais être négatif ou nul.
	mu, sigma := 25.0, 8.333
	newMu, newSigma := trueskillUpdate(mu, sigma, 30.0, 5.0, 0.5, 1.0)
	if newSigma <= 0 {
		t.Errorf("sigma doit être > 0, got %v", newSigma)
	}
	_ = newMu
}
