package campaign

import (
	"math"
	"testing"
)

// Tests unitaires du test de Mann-Whitney U.

func TestMannWhitneyU_IdenticalDistributions(t *testing.T) {
	a := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	b := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	_, p := MannWhitneyU(a, b)
	if p < 0.5 {
		t.Errorf("identical distros: p=%.3f, want > 0.5", p)
	}
}

func TestMannWhitneyU_ClearlyDifferent(t *testing.T) {
	// Distribution a centrée sur 1-10, b centrée sur 11-20 → différence évidente.
	a := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	b := []float64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	_, p := MannWhitneyU(a, b)
	if p >= 0.05 {
		t.Errorf("clearly different distros: p=%.3f, want < 0.05", p)
	}
}

func TestMannWhitneyU_SmallSample_ReturnsOne(t *testing.T) {
	a := []float64{1, 2}
	b := []float64{3, 4, 5}
	_, p := MannWhitneyU(a, b)
	if p != 1.0 {
		t.Errorf("small sample: p=%.3f, want 1.0 (conservative)", p)
	}
}

func TestMannWhitneyU_BelowCLT_ReturnsOne(t *testing.T) {
	// 5+5=10 < 20 → l'approximation normale n'est pas appliquée.
	a := []float64{1, 2, 3, 4, 5}
	b := []float64{6, 7, 8, 9, 10}
	_, p := MannWhitneyU(a, b)
	if p != 1.0 {
		t.Errorf("below CLT: p=%.3f, want 1.0", p)
	}
}

func TestMannWhitneyU_TiesHandled(t *testing.T) {
	// Beaucoup d'ex-aequo → ne crash pas, retourne un p raisonnable.
	a := []float64{5, 5, 5, 5, 5, 5, 5, 5, 5, 5}
	b := []float64{5, 5, 5, 5, 5, 5, 5, 5, 5, 5}
	_, p := MannWhitneyU(a, b)
	if math.IsNaN(p) {
		t.Errorf("ties: got NaN")
	}
	if p < 0.5 {
		t.Errorf("identical ties: p=%.3f, want close to 1", p)
	}
}

func TestNormalCDF_Boundaries(t *testing.T) {
	if got := normalCDF(0); math.Abs(got-0.5) > 1e-9 {
		t.Errorf("normalCDF(0) = %.6f, want 0.5", got)
	}
	if got := normalCDF(10); math.Abs(got-1.0) > 1e-9 {
		t.Errorf("normalCDF(10) ≈ %.6f, want 1.0", got)
	}
	if got := normalCDF(-10); math.Abs(got) > 1e-9 {
		t.Errorf("normalCDF(-10) ≈ %.6f, want 0.0", got)
	}
}
