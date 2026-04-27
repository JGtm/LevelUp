package temporal

import (
	"math"
	"testing"
)

func TestRollingMean_Basic(t *testing.T) {
	t.Parallel()
	points := []float64{1, 2, 3, 4, 5}
	got := RollingMean(points, 3, 1)
	// i=0 -> [1]      -> mean=1
	// i=1 -> [1,2]    -> mean=1.5
	// i=2 -> [1,2,3]  -> mean=2
	// i=3 -> [2,3,4]  -> mean=3
	// i=4 -> [3,4,5]  -> mean=4
	want := []float64{1, 1.5, 2, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("want len %d got %d", len(want), len(got))
	}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Errorf("i=%d want %v got %v", i, want[i], got[i])
		}
	}
}

func TestRollingMean_MinPoints(t *testing.T) {
	t.Parallel()
	points := []float64{1, 2, 3, 4, 5}
	got := RollingMean(points, 3, 3)
	// i=0,1 fenetres trop petites -> NaN
	// i=2,3,4 -> 2, 3, 4
	if !math.IsNaN(got[0]) || !math.IsNaN(got[1]) {
		t.Errorf("i=0,1 should be NaN, got %v %v", got[0], got[1])
	}
	if got[2] != 2 || got[3] != 3 || got[4] != 4 {
		t.Errorf("i>=2 want 2,3,4 got %v %v %v", got[2], got[3], got[4])
	}
}

func TestRollingMean_AllNaN(t *testing.T) {
	t.Parallel()
	// Si minPoints > len(points), tout est NaN.
	points := []float64{1, 2, 3}
	got := RollingMean(points, 5, 5)
	for i, v := range got {
		if !math.IsNaN(v) {
			t.Errorf("i=%d expected NaN, got %v", i, v)
		}
	}
}

func TestRollingMean_Empty(t *testing.T) {
	t.Parallel()
	got := RollingMean([]float64{}, 5, 1)
	if len(got) != 0 {
		t.Errorf("want empty, got len=%d", len(got))
	}
}

func TestRollingMean_IntType(t *testing.T) {
	t.Parallel()
	// Verifie le generique sur int.
	points := []int{2, 4, 6, 8, 10}
	got := RollingMean(points, 2, 1)
	want := []float64{2, 3, 5, 7, 9}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("i=%d want %v got %v", i, want[i], got[i])
		}
	}
}

func TestRollingMean_Int64Type(t *testing.T) {
	t.Parallel()
	points := []int64{1000, 2000, 3000}
	got := RollingMean(points, 2, 1)
	want := []float64{1000, 1500, 2500}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("i=%d want %v got %v", i, want[i], got[i])
		}
	}
}

func TestRollingMean_DefensiveBounds(t *testing.T) {
	t.Parallel()
	points := []float64{1, 2, 3}
	// window=0 -> normalise a 1
	got := RollingMean(points, 0, 0)
	if got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("window=0 should normalise to 1, got %v", got)
	}
	// minPoints=0 -> normalise a 1
	got2 := RollingMean(points, 2, 0)
	if math.IsNaN(got2[0]) {
		t.Error("minPoints=0 should be normalised to 1, got NaN")
	}
}

func TestRollingMeanAdaptive_LongSeries(t *testing.T) {
	t.Parallel()
	// 100 points, pct=10 -> window=10
	points := make([]float64, 100)
	for i := range points {
		points[i] = float64(i)
	}
	got := RollingMeanAdaptive(points, 3, 10)
	// dernier point : moyenne des 10 derniers = (90+91+...+99)/10 = 94.5
	if math.Abs(got[99]-94.5) > 1e-9 {
		t.Errorf("got[99] want 94.5 got %v", got[99])
	}
}

func TestRollingMeanAdaptive_ShortSeries(t *testing.T) {
	t.Parallel()
	// 5 points, minWindow=3, pct=10 -> window = max(3, 5*10/100=0) = 3
	points := []float64{1, 2, 3, 4, 5}
	got := RollingMeanAdaptive(points, 3, 10)
	// i=2 -> [1,2,3] -> 2
	if got[2] != 2 {
		t.Errorf("i=2 want 2 got %v", got[2])
	}
	// i=0,1 trop courts (size<minWindow=3) -> NaN
	if !math.IsNaN(got[0]) || !math.IsNaN(got[1]) {
		t.Errorf("i=0,1 should be NaN, got %v %v", got[0], got[1])
	}
}

func TestRollingMeanAdaptive_Empty(t *testing.T) {
	t.Parallel()
	got := RollingMeanAdaptive([]float64{}, 3, 10)
	if len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
}

func TestRollingMeanAdaptive_DefensiveBounds(t *testing.T) {
	t.Parallel()
	points := []float64{1, 2, 3, 4, 5}
	// pct < 0 normalise a 0
	got := RollingMeanAdaptive(points, 2, -10)
	// window = max(2, 5*0/100=0) = 2
	// i=1 -> [1,2] -> 1.5
	if got[1] != 1.5 {
		t.Errorf("pct<0 should normalise to 0, got[1]=%v", got[1])
	}
}
