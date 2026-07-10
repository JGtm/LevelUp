// Package sync — skill_rating_test.go : tests unitaires pour les fonctions TrueSkill pures.
//
// Note : ce package importe DuckDB transitif → ne compile pas sur Windows (contrainte
// build constraint windows-amd64). Ces tests sont conçus pour tourner en CI Linux.
package skill

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
	// Valeurs issues des constantes de production : InitialMU=1500, InitialSigma=350.
	mu, sigma := InitialMU, InitialSigma
	muOpp, sigmaOpp := InitialMU, InitialSigma
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
	// Valeurs issues des constantes de production : InitialMU=1500, InitialSigma=350.
	mu, sigma := InitialMU, InitialSigma
	muOpp, sigmaOpp := InitialMU, InitialSigma
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
	// Valeurs issues des constantes de production.
	mu, sigma := InitialMU, InitialSigma
	newMu, newSigma := trueskillUpdate(mu, sigma, 1600.0, 300.0, 0.5, 1.0)
	if newSigma <= 0 {
		t.Errorf("sigma doit être > 0, got %v", newSigma)
	}
	_ = newMu
}

// ── muOpp branché : expectedScore dynamique ───────────────────────────────────

// TestTrueskillUpdate_StrongerOpponents : battre des adversaires plus forts
// (muOpp > mu) doit donner un gain supérieur à battre des égaux avec le même composite.
func TestTrueskillUpdate_StrongerOpponents_MoreGain(t *testing.T) {
	mu, sigma := 1400.0, InitialSigma // joueur Or
	composite := 0.65                 // bonne perf

	// Adversaires égaux (muOpp = mu)
	newMuEqual, _ := trueskillUpdate(mu, sigma, mu, InitialSigma, composite, 1.0)
	gainEqual := newMuEqual - mu

	// Adversaires Platine (muOpp > mu)
	newMuStrong, _ := trueskillUpdate(mu, sigma, 1700.0, InitialSigma, composite, 1.0)
	gainStrong := newMuStrong - mu

	if gainStrong <= gainEqual {
		t.Errorf("adversaires forts devraient donner plus de gain: fort=%.2f egal=%.2f", gainStrong, gainEqual)
	}
}

// TestTrueskillUpdate_WeakerOpponents_LessGain : battre des adversaires plus faibles
// avec le même composite doit donner moins de gain (ou perdre du mu si performance neutre).
func TestTrueskillUpdate_WeakerOpponents_LessGain(t *testing.T) {
	mu, sigma := 1600.0, InitialSigma // joueur Platine
	composite := 0.65

	// Adversaires égaux
	newMuEqual, _ := trueskillUpdate(mu, sigma, mu, InitialSigma, composite, 1.0)
	gainEqual := newMuEqual - mu

	// Adversaires Or (muOpp < mu)
	newMuWeak, _ := trueskillUpdate(mu, sigma, 1300.0, InitialSigma, composite, 1.0)
	gainWeak := newMuWeak - mu

	if gainWeak >= gainEqual {
		t.Errorf("adversaires faibles devraient donner moins de gain: faible=%.2f egal=%.2f", gainWeak, gainEqual)
	}
}

// TestTrueskillUpdate_EqualOpponents_NeutralPerf : avec muOpp=mu et composite=0.5
// (performance exactement attendue), le delta mu doit être nul.
func TestTrueskillUpdate_EqualOpponents_NeutralPerfZeroDelta(t *testing.T) {
	mu, sigma := InitialMU, InitialSigma
	newMu, _ := trueskillUpdate(mu, sigma, mu, sigma, 0.5, 1.0)
	delta := newMu - mu
	if delta != 0 {
		t.Errorf("perf neutre vs égaux : delta mu attendu 0, got %.4f", delta)
	}
}
