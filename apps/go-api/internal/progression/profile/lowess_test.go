package profile

import (
	"math"
	"testing"
)

// lowess_test.go — comportement de ComputeMuTrend + helpers NaN.

func TestComputeMuTrend_TooShort(t *testing.T) {
	// len < 3 → Slope=0, Window=0, Metric toujours "mu".
	for _, n := range []int{0, 1, 2} {
		series := make([]float64, n)
		for i := range series {
			series[i] = float64(i)
		}
		got := ComputeMuTrend(series)
		if got.Slope != 0 {
			t.Errorf("n=%d: Slope = %v, want 0", n, got.Slope)
		}
		if got.Window != 0 {
			t.Errorf("n=%d: Window = %d, want 0", n, got.Window)
		}
		if got.Metric != "mu" {
			t.Errorf("n=%d: Metric = %q, want mu", n, got.Metric)
		}
	}
}

func TestComputeMuTrend_IncreasingSeriesPositiveSlope(t *testing.T) {
	// Série strictement croissante → tendance lissée croissante → Slope > 0.
	series := []float64{1000, 1010, 1020, 1030, 1040, 1050, 1060, 1070}
	got := ComputeMuTrend(series)
	if got.Slope <= 0 {
		t.Errorf("Slope = %v, want > 0 (série croissante)", got.Slope)
	}
	if got.Window != len(series) {
		t.Errorf("Window = %d, want %d", got.Window, len(series))
	}
	if !got.IsPositive(3) {
		t.Errorf("IsPositive(3) = false, want true (slope>0 et window>=3)")
	}
}

func TestComputeMuTrend_DecreasingSeriesNegativeSlope(t *testing.T) {
	// Série décroissante → Slope < 0, IsPositive faux.
	series := []float64{1100, 1090, 1080, 1070, 1060, 1050}
	got := ComputeMuTrend(series)
	if got.Slope >= 0 {
		t.Errorf("Slope = %v, want < 0 (série décroissante)", got.Slope)
	}
	if got.IsPositive(3) {
		t.Errorf("IsPositive(3) = true, want false (slope négatif)")
	}
}

func TestComputeMuTrend_FlatSeriesZeroSlope(t *testing.T) {
	// Série constante → lissage constant → Slope == 0.
	series := []float64{1500, 1500, 1500, 1500, 1500}
	got := ComputeMuTrend(series)
	if math.Abs(got.Slope) > 1e-9 {
		t.Errorf("Slope = %v, want ≈0 (série plate)", got.Slope)
	}
}

func TestComputeMuTrend_AllNaN(t *testing.T) {
	// Toutes les valeurs NaN → LowessSmooth renvoie NaN partout →
	// firstValid/lastValid renvoient NaN → Slope=0, Window=0.
	nan := math.NaN()
	series := []float64{nan, nan, nan, nan}
	got := ComputeMuTrend(series)
	if got.Slope != 0 {
		t.Errorf("Slope = %v, want 0 (full NaN)", got.Slope)
	}
	if got.Window != 0 {
		t.Errorf("Window = %d, want 0 (full NaN, sortie précoce)", got.Window)
	}
}

func TestFirstValid(t *testing.T) {
	nan := math.NaN()
	tests := []struct {
		name string
		in   []float64
		want float64
		nan  bool
	}{
		{"premier valide", []float64{1, 2, 3}, 1, false},
		{"saute les NaN initiaux", []float64{nan, nan, 7, 8}, 7, false},
		{"vide → NaN", []float64{}, 0, true},
		{"que des NaN → NaN", []float64{nan, nan}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstValid(tt.in)
			if tt.nan {
				if !math.IsNaN(got) {
					t.Errorf("got %v, want NaN", got)
				}
				return
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLastValid(t *testing.T) {
	nan := math.NaN()
	tests := []struct {
		name string
		in   []float64
		want float64
		nan  bool
	}{
		{"dernier valide", []float64{1, 2, 3}, 3, false},
		{"saute les NaN finaux", []float64{4, 5, nan, nan}, 5, false},
		{"vide → NaN", []float64{}, 0, true},
		{"que des NaN → NaN", []float64{nan, nan}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lastValid(tt.in)
			if tt.nan {
				if !math.IsNaN(got) {
					t.Errorf("got %v, want NaN", got)
				}
				return
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
