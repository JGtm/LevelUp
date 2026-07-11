package temporal

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"testing"

	"levelup/go-api/internal/analysis"
)

// =============================================================================
// Helpers de construction d'echantillons (modele lobby-anchored : seul le ratio
// pace_joueur/pace_lobby compte pour ComputeEngagementCoefficient).
// =============================================================================

// lobbySample construit un sample valide avec un ratio joueur/lobby voulu.
func lobbySample(matchID string, ratioLobby float64) RatioSample {
	const baseLobby = 10.0 // events/min/joueur — bien au-dessus du seuil 0.75
	return RatioSample{
		MatchID:        matchID,
		PaceLobby:      baseLobby,
		PaceTeam:       baseLobby, // non lu par le coef, renseigne par cohérence
		PaceJoueur:     baseLobby * ratioLobby,
		PlayerActivity: 30,
	}
}

// constantLobbySamples genere n samples avec le meme ratio lobby.
func constantLobbySamples(n int, ratioLobby float64) []RatioSample {
	out := make([]RatioSample, n)
	for i := range out {
		out[i] = lobbySample(fmt.Sprintf("m%d", i), ratioLobby)
	}
	return out
}

// =============================================================================
// Cas degeneres et seuils
// =============================================================================

func TestComputeEngagementCoefficient_EmptySample(t *testing.T) {
	t.Parallel()
	_, err := ComputeEngagementCoefficient(nil)
	if !errors.Is(err, ErrInsufficientCoefHistory) {
		t.Errorf("want ErrInsufficientCoefHistory on nil samples, got %v", err)
	}
	_, err = ComputeEngagementCoefficient([]RatioSample{})
	if !errors.Is(err, ErrInsufficientCoefHistory) {
		t.Errorf("want ErrInsufficientCoefHistory on empty samples, got %v", err)
	}
}

func TestComputeEngagementCoefficient_BelowThreshold(t *testing.T) {
	t.Parallel()
	// 9 samples valides — strictement sous le seuil de 10
	samples := constantLobbySamples(9, 1.1)
	_, err := ComputeEngagementCoefficient(samples)
	if !errors.Is(err, ErrInsufficientCoefHistory) {
		t.Errorf("want ErrInsufficientCoefHistory with 9 samples, got %v", err)
	}
}

func TestComputeEngagementCoefficient_ExactlyMinThreshold(t *testing.T) {
	t.Parallel()
	samples := constantLobbySamples(MinMatchesForCoef, 1.1)
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("want OK at MinMatchesForCoef, got %v", err)
	}
	if res.NMatches != MinMatchesForCoef {
		t.Errorf("NMatches want %d, got %d", MinMatchesForCoef, res.NMatches)
	}
}

// =============================================================================
// Mediane stable et correcte (coef lobby)
// =============================================================================

func TestComputeEngagementCoefficient_ConstantRatio(t *testing.T) {
	t.Parallel()
	samples := constantLobbySamples(50, 1.05)
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if math.Abs(res.CoefLobbyShare-1.05) > 1e-9 {
		t.Errorf("CoefLobbyShare want 1.05, got %v", res.CoefLobbyShare)
	}
	if res.NMatches != 50 {
		t.Errorf("NMatches want 50, got %d", res.NMatches)
	}
	if res.NRejected != 0 {
		t.Errorf("NRejected want 0, got %d", res.NRejected)
	}
}

func TestComputeEngagementCoefficient_OddSampleSize(t *testing.T) {
	t.Parallel()
	ratios := []float64{0.5, 0.7, 0.9, 1.0, 1.1, 1.2, 1.3, 1.4, 1.5, 1.7, 2.0}
	samples := make([]RatioSample, len(ratios))
	for i, r := range ratios {
		samples[i] = lobbySample(fmt.Sprintf("m%d", i), r)
	}
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	// Mediane de 11 valeurs triees = la 6e (index 5) = 1.2
	if math.Abs(res.CoefLobbyShare-1.2) > 1e-9 {
		t.Errorf("CoefLobbyShare want 1.2 (median of 11 ratios), got %v", res.CoefLobbyShare)
	}
}

func TestComputeEngagementCoefficient_EvenSampleSize(t *testing.T) {
	t.Parallel()
	ratios := []float64{0.5, 0.7, 0.9, 1.0, 1.1, 1.2, 1.3, 1.4, 1.5, 1.7, 2.0, 2.5}
	samples := make([]RatioSample, len(ratios))
	for i, r := range ratios {
		samples[i] = lobbySample(fmt.Sprintf("m%d", i), r)
	}
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	// Mediane de 12 valeurs = (1.2 + 1.3) / 2 = 1.25
	if math.Abs(res.CoefLobbyShare-1.25) > 1e-9 {
		t.Errorf("CoefLobbyShare want 1.25 (median of 12 ratios), got %v", res.CoefLobbyShare)
	}
}

func TestComputeEngagementCoefficient_MedianRobustToOutliers(t *testing.T) {
	t.Parallel()
	samples := make([]RatioSample, 0, 100)
	for i := 0; i < 90; i++ {
		samples = append(samples, lobbySample(fmt.Sprintf("normal%d", i), 1.0))
	}
	for i := 0; i < 10; i++ {
		samples = append(samples, RatioSample{
			MatchID:        fmt.Sprintf("outlier%d", i),
			PaceJoueur:     1000,
			PaceLobby:      10,
			PlayerActivity: 50,
		})
	}
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if math.Abs(res.CoefLobbyShare-1.0) > 1e-9 {
		t.Errorf("median should resist outliers, want ~1.0, got %v", res.CoefLobbyShare)
	}
}

// =============================================================================
// Filtres et exclusions (PlayerActivity, PaceLobby)
// =============================================================================

func TestComputeEngagementCoefficient_FilterAFKPlayerActivity(t *testing.T) {
	t.Parallel()
	samples := constantLobbySamples(15, 1.5)
	for i := 0; i < 10; i++ {
		samples = append(samples, RatioSample{
			MatchID:        fmt.Sprintf("quit%d", i),
			PaceJoueur:     5, // ratio aberrant qui DEVRAIT polluer si non filtre
			PaceLobby:      10,
			PlayerActivity: 2, // sous le seuil PlayerActivityMin=3
		})
	}
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if math.Abs(res.CoefLobbyShare-1.5) > 1e-9 {
		t.Errorf("AFK samples should be filtered, want CoefLobby=1.5, got %v", res.CoefLobbyShare)
	}
	if res.NRejected != 10 {
		t.Errorf("NRejected want 10 (AFK), got %d", res.NRejected)
	}
	if res.NMatches != 15 {
		t.Errorf("NMatches want 15, got %d", res.NMatches)
	}
}

func TestComputeEngagementCoefficient_FilterLowPaceLobby(t *testing.T) {
	t.Parallel()
	samples := constantLobbySamples(12, 1.0)
	for i := 0; i < 5; i++ {
		samples = append(samples, RatioSample{
			MatchID:        fmt.Sprintf("afkLobby%d", i),
			PaceJoueur:     5,
			PaceLobby:      0.4, // < PaceLobbyMinThreshold=0.75
			PlayerActivity: 30,
		})
	}
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if res.NMatches != 12 {
		t.Errorf("NMatches want 12 (5 lobby AFK rejected), got %d", res.NMatches)
	}
	if res.NRejected != 5 {
		t.Errorf("NRejected want 5, got %d", res.NRejected)
	}
}

func TestComputeEngagementCoefficient_FilterZeroPace(t *testing.T) {
	t.Parallel()
	samples := constantLobbySamples(15, 1.0)
	samples = append(samples, RatioSample{
		MatchID:        "zero",
		PaceJoueur:     5,
		PaceLobby:      0, // explicit zero
		PlayerActivity: 30,
	})
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if res.NRejected != 1 {
		t.Errorf("zero-pace sample should be rejected, NRejected got %d", res.NRejected)
	}
}

// =============================================================================
// Bornage (clampCoef)
// =============================================================================

func TestComputeEngagementCoefficient_ClampHighCoef(t *testing.T) {
	t.Parallel()
	samples := make([]RatioSample, 15)
	for i := range samples {
		samples[i] = RatioSample{
			MatchID:        fmt.Sprintf("m%d", i),
			PaceJoueur:     100,
			PaceLobby:      10,
			PlayerActivity: 30,
		}
	}
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if res.CoefLobbyShare != CoefMax {
		t.Errorf("clampHigh want %v, got %v", CoefMax, res.CoefLobbyShare)
	}
}

func TestComputeEngagementCoefficient_ClampLowCoef(t *testing.T) {
	t.Parallel()
	samples := make([]RatioSample, 15)
	for i := range samples {
		samples[i] = RatioSample{
			MatchID:        fmt.Sprintf("m%d", i),
			PaceJoueur:     0.05,
			PaceLobby:      10,
			PlayerActivity: 5,
		}
	}
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if res.CoefLobbyShare != CoefMin {
		t.Errorf("clampLow want %v, got %v", CoefMin, res.CoefLobbyShare)
	}
}

// =============================================================================
// Determinisme et stabilite
// =============================================================================

func TestComputeEngagementCoefficient_DeterministicOrder(t *testing.T) {
	t.Parallel()
	ratios := []float64{0.9, 1.1, 1.0, 1.2, 1.3, 0.95, 1.05, 0.85, 1.15, 1.0}
	samples1 := make([]RatioSample, len(ratios))
	for i, r := range ratios {
		samples1[i] = lobbySample(fmt.Sprintf("m%d", i), r)
	}
	samples2 := make([]RatioSample, len(samples1))
	copy(samples2, samples1)
	for i, j := 0, len(samples2)-1; i < j; i, j = i+1, j-1 {
		samples2[i], samples2[j] = samples2[j], samples2[i]
	}
	r1, _ := ComputeEngagementCoefficient(samples1)
	r2, _ := ComputeEngagementCoefficient(samples2)
	if math.Abs(r1.CoefLobbyShare-r2.CoefLobbyShare) > 1e-12 {
		t.Errorf("order matters: %v vs %v", r1.CoefLobbyShare, r2.CoefLobbyShare)
	}
}

func TestComputeEngagementCoefficient_GaussianDistribution(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(42))
	const targetMedian = 1.15
	samples := make([]RatioSample, 200)
	for i := range samples {
		ratio := targetMedian + rng.NormFloat64()*0.2
		if ratio < 0.5 {
			ratio = 0.5
		}
		if ratio > 2.5 {
			ratio = 2.5
		}
		samples[i] = lobbySample(fmt.Sprintf("m%d", i), ratio)
	}
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if math.Abs(res.CoefLobbyShare-targetMedian) > 0.05 {
		t.Errorf("gaussian median should be near %v, got %v", targetMedian, res.CoefLobbyShare)
	}
}

// =============================================================================
// Mutation safety
// =============================================================================

func TestComputeEngagementCoefficient_DoesNotMutateInput(t *testing.T) {
	t.Parallel()
	samples := constantLobbySamples(20, 1.1)
	snapshot := make([]RatioSample, len(samples))
	copy(snapshot, samples)
	_, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	for i := range samples {
		if samples[i] != snapshot[i] {
			t.Errorf("sample[%d] mutated: %+v vs %+v", i, samples[i], snapshot[i])
		}
	}
}

// =============================================================================
// Sanity check helpers
// =============================================================================

func TestMedian(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   []float64
		want float64
	}{
		{"single", []float64{42}, 42},
		{"two_values", []float64{2, 4}, 3},
		{"odd_sorted", []float64{1, 2, 3, 4, 5}, 3},
		{"odd_unsorted", []float64{5, 1, 3, 4, 2}, 3},
		{"even_sorted", []float64{1, 2, 3, 4}, 2.5},
		{"with_negatives", []float64{-3, -1, 0, 1, 3}, 0},
		{"duplicates", []float64{5, 5, 5, 5, 5}, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := analysis.MedianFloat(c.in)
			if math.Abs(got-c.want) > 1e-9 {
				t.Errorf("MedianFloat(%v) want %v, got %v", c.in, c.want, got)
			}
		})
	}
}

func TestClampCoef(t *testing.T) {
	t.Parallel()
	cases := []struct{ in, want float64 }{
		{0.0, CoefMin},
		{-1.0, CoefMin},
		{0.05, CoefMin},
		{0.1, CoefMin},
		{0.5, 0.5},
		{1.0, 1.0},
		{2.5, 2.5},
		{5.0, 5.0},
		{5.1, CoefMax},
		{100.0, CoefMax},
	}
	for _, c := range cases {
		got := clampCoef(c.in)
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("clampCoef(%v) want %v, got %v", c.in, c.want, got)
		}
	}
}
