package profile

import (
	"math"
	"testing"
	"time"
)

// lowess_test.go — comportement de ComputeMuTrend + helpers NaN + buildSkillTrend.

func TestBuildSkillTrend_TooShortReturnsNil(t *testing.T) {
	base := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	for _, n := range []int{0, 1, 2} {
		pts := make([]muPoint, n)
		for i := range pts {
			pts[i] = muPoint{at: base.AddDate(0, 0, i), value: 1500 + float64(i)}
		}
		if got := buildSkillTrend(pts); got != nil {
			t.Errorf("n=%d: got %v points, want nil (< 3 → LOWESS non fiable)", n, len(got))
		}
	}
}

func TestBuildSkillTrend_SmoothesAndDatesInUTC(t *testing.T) {
	// 24 points bruités (heure > minuit UTC ; le bucket doit rester le jour UTC).
	// n=24 → window LOWESS = floor(0.3*24) = 7 : le lissage est effectif sur les
	// points intérieurs (le poids tricube est nul aux bords, il faut window ≥ 5).
	base := time.Date(2026, 7, 1, 23, 30, 0, 0, time.UTC)
	const n = 24
	pts := make([]muPoint, n)
	raw := make([]float64, n)
	for i := 0; i < n; i++ {
		v := 1500.0 + float64(i)*3 // tendance montante
		if i%2 == 0 {
			v += 120 // bruit alterné, à écraser par le lissage
		}
		raw[i] = v
		pts[i] = muPoint{at: base.AddDate(0, 0, i), value: v}
	}
	got := buildSkillTrend(pts)
	if len(got) != n {
		t.Fatalf("len = %d, want %d (1 point lissé par point d'entrée)", len(got), n)
	}
	// Le lissage LOWESS écrase le bruit : la valeur servie diffère du μ brut
	// (jamais de μ brut à l'écran, DEC-6) — au moins un point intérieur lissé.
	anySmoothed := false
	for i := range got {
		if math.Abs(got[i].Value-raw[i]) > 1e-6 {
			anySmoothed = true
		}
		if got[i].Date != pts[i].at.UTC().Format("2006-01-02") {
			t.Errorf("point %d: Date = %q, want %q", i, got[i].Date, pts[i].at.UTC().Format("2006-01-02"))
		}
	}
	if !anySmoothed {
		t.Errorf("aucun point lissé — LOWESS n'a rien changé (attendu sur série bruitée)")
	}
	// Premier jour = 2026-07-01 (23h30 UTC reste le 1er).
	if got[0].Date != "2026-07-01" {
		t.Errorf("premier jour = %q, want 2026-07-01", got[0].Date)
	}
}

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
