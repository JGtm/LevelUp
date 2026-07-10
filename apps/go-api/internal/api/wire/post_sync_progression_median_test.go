// Package api — post_sync_progression_median_test.go : tests Windows-friendly
// (sans dépendance duckdb) du helper medianKDA (V2 §4).
package wire

import (
	"testing"

	"levelup/go-api/internal/progression/streaks"
)

func TestMedianKDA_EmptyReturnsZero(t *testing.T) {
	if got := medianKDA(nil); got != 0 {
		t.Errorf("nil: got %.2f, want 0", got)
	}
}

func TestMedianKDA_LessThan10ReturnsZero(t *testing.T) {
	// Significativité insuffisante : retourne 0 sous 10 matchs.
	acts := make([]streaks.MatchActivity, 5)
	for i := range acts {
		acts[i] = streaks.MatchActivity{Stats: map[string]float64{"kda": 1.5}}
	}
	if got := medianKDA(acts); got != 0 {
		t.Errorf("9 matches: got %.2f, want 0", got)
	}
}

func TestMedianKDA_OddCount(t *testing.T) {
	// 11 matchs KDA = [0.5, 1.0, 1.5, 2.0, 2.5, 3.0, 3.5, 4.0, 4.5, 5.0, 5.5]
	// Médiane = 3.0 (6e élément).
	acts := []streaks.MatchActivity{
		{Stats: map[string]float64{"kda": 0.5}},
		{Stats: map[string]float64{"kda": 1.0}},
		{Stats: map[string]float64{"kda": 1.5}},
		{Stats: map[string]float64{"kda": 2.0}},
		{Stats: map[string]float64{"kda": 2.5}},
		{Stats: map[string]float64{"kda": 3.0}},
		{Stats: map[string]float64{"kda": 3.5}},
		{Stats: map[string]float64{"kda": 4.0}},
		{Stats: map[string]float64{"kda": 4.5}},
		{Stats: map[string]float64{"kda": 5.0}},
		{Stats: map[string]float64{"kda": 5.5}},
	}
	got := medianKDA(acts)
	if got != 3.0 {
		t.Errorf("odd count: got %.2f, want 3.0", got)
	}
}

func TestMedianKDA_EvenCount(t *testing.T) {
	// 10 matchs KDA = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
	// Médiane = (5 + 6) / 2 = 5.5
	acts := make([]streaks.MatchActivity, 10)
	for i := range acts {
		acts[i] = streaks.MatchActivity{Stats: map[string]float64{"kda": float64(i + 1)}}
	}
	got := medianKDA(acts)
	if got != 5.5 {
		t.Errorf("even count: got %.2f, want 5.5", got)
	}
}

func TestMedianKDA_RobustToOutliers(t *testing.T) {
	// Médiane résiste aux valeurs aberrantes. 10 matchs KDA réalistes + 1
	// match outlier à 50.0 → la médiane ne bouge pas significativement.
	acts := make([]streaks.MatchActivity, 11)
	for i := 0; i < 10; i++ {
		acts[i] = streaks.MatchActivity{Stats: map[string]float64{"kda": 1.5}}
	}
	acts[10] = streaks.MatchActivity{Stats: map[string]float64{"kda": 50.0}}
	got := medianKDA(acts)
	if got != 1.5 {
		t.Errorf("with outlier: got %.2f, want 1.5 (robust)", got)
	}
}

func TestMedianKDA_SkipsZeroAndMissing(t *testing.T) {
	// KDA=0 (Stats absent) skip ; on garde 10 valeurs valides.
	acts := []streaks.MatchActivity{
		{Stats: map[string]float64{}},           // skip
		{Stats: map[string]float64{"kda": 0.0}}, // skip
	}
	for i := 0; i < 10; i++ {
		acts = append(acts, streaks.MatchActivity{Stats: map[string]float64{"kda": float64(i + 1)}})
	}
	got := medianKDA(acts)
	// 10 valides [1..10], médiane = 5.5
	if got != 5.5 {
		t.Errorf("skipping zero/missing: got %.2f, want 5.5", got)
	}
}
