package temporal

import (
	"math"
	"testing"
)

func TestLowessSmooth_Empty(t *testing.T) {
	t.Parallel()
	got := LowessSmooth(nil, 0.3)
	if got != nil {
		t.Errorf("nil input: want nil, got %v", got)
	}
}

func TestLowessSmooth_OutputSameLength(t *testing.T) {
	t.Parallel()
	points := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	got := LowessSmooth(points, 0.3)
	if len(got) != len(points) {
		t.Errorf("output length: want %d, got %d", len(points), len(got))
	}
}

func TestLowessSmooth_LinearTrend(t *testing.T) {
	t.Parallel()
	// Serie strictement lineaire -> LOWESS doit la reproduire (a tolerance pres).
	points := make([]float64, 20)
	for i := range points {
		points[i] = float64(i) * 2 // y = 2x
	}
	got := LowessSmooth(points, 0.3)
	for i := 3; i < len(got)-3; i++ {
		// Au centre de la serie, le lissage doit etre tres proche de la valeur d'origine.
		if math.Abs(got[i]-points[i]) > 0.5 {
			t.Errorf("i=%d : linear should be preserved, want ~%v, got %v",
				i, points[i], got[i])
		}
	}
}

func TestLowessSmooth_NoiseReduced(t *testing.T) {
	t.Parallel()
	// Serie bruitee autour d'une constante : le lissage doit reduire la variance.
	points := []float64{50, 60, 40, 55, 45, 52, 48, 53, 47, 51, 49, 50, 50, 50, 50}
	got := LowessSmooth(points, 0.5)

	// Variance des points lissees au centre doit etre plus faible que la variance originale.
	varOrig := variance(points[3 : len(points)-3])
	varSmoothed := variance(got[3 : len(got)-3])
	if varSmoothed >= varOrig {
		t.Errorf("LOWESS should reduce variance: orig %v vs smoothed %v",
			varOrig, varSmoothed)
	}
}

func TestLowessSmooth_AlphaClamping(t *testing.T) {
	t.Parallel()
	points := []float64{1, 2, 3, 4, 5}
	// alpha invalides clampees automatiquement
	if got := LowessSmooth(points, -0.5); len(got) != 5 {
		t.Errorf("alpha negatif : output OK attendu, got len %d", len(got))
	}
	if got := LowessSmooth(points, 5.0); len(got) != 5 {
		t.Errorf("alpha > 1 : output OK attendu, got len %d", len(got))
	}
}

func TestLowessSmooth_NanPropagated(t *testing.T) {
	t.Parallel()
	points := []float64{1, 2, math.NaN(), 4, 5, 6, 7, 8, 9, 10}
	got := LowessSmooth(points, 0.5)
	if !math.IsNaN(got[2]) {
		t.Error("NaN input doit etre preserve dans la sortie")
	}
	// Les autres points doivent etre lisses normalement.
	if math.IsNaN(got[0]) || math.IsNaN(got[5]) {
		t.Errorf("non-NaN inputs should produce non-NaN outputs, got %v %v",
			got[0], got[5])
	}
}

func TestLowessSmooth_AllNaN(t *testing.T) {
	t.Parallel()
	points := []float64{math.NaN(), math.NaN(), math.NaN()}
	got := LowessSmooth(points, 0.5)
	for i, v := range got {
		if !math.IsNaN(v) {
			t.Errorf("i=%d: all NaN input -> all NaN output, got %v", i, v)
		}
	}
}

func variance(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	mean := sum / float64(len(xs))
	var sq float64
	for _, x := range xs {
		d := x - mean
		sq += d * d
	}
	return sq / float64(len(xs))
}
