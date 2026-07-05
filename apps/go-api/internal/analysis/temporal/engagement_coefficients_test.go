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
// Helpers de construction d'echantillons
// =============================================================================

// validSample construit un sample qui passe tous les filtres avec un ratio
// joueur/team voulu.
func validSample(matchID string, ratioTeam, ratioLobby float64) RatioSample {
	const baseTeam = 10.0 // events/min/joueur de team — bien au-dessus du seuil 1.0
	return RatioSample{
		MatchID:        matchID,
		PaceTeam:       baseTeam,
		PaceLobby:      baseTeam,
		PaceJoueur:     baseTeam * ratioTeam,
		PlayerActivity: 30,
	}
}

// constantRatioSamples genere n samples avec le meme ratio team (et lobby
// independant). Utile pour les tests de mediane stable.
func constantRatioSamples(n int, ratioTeam, ratioLobby float64) []RatioSample {
	const baseTeam = 10.0
	const baseLobby = 12.0
	out := make([]RatioSample, n)
	for i := range out {
		out[i] = RatioSample{
			MatchID:        fmt.Sprintf("m%d", i),
			PaceTeam:       baseTeam,
			PaceLobby:      baseLobby,
			PaceJoueur:     baseTeam*ratioTeam + 0, // ratio team applique
			PlayerActivity: 30,
		}
		// Le ratio lobby n'est pas exactement le meme : on ajuste pour que le
		// ratio observe sur baseLobby corresponde a ratioLobby voulu.
		out[i].PaceJoueur = baseLobby * ratioLobby
		// Mais on veut aussi que le ratio team soit ratioTeam. Solution : on
		// fixe PaceJoueur sur le ratio team, et on ajuste PaceLobby pour que
		// PaceJoueur/PaceLobby = ratioLobby.
		out[i].PaceJoueur = baseTeam * ratioTeam
		if ratioLobby > 0 {
			out[i].PaceLobby = out[i].PaceJoueur / ratioLobby
		}
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
	samples := constantRatioSamples(9, 1.2, 1.1)
	_, err := ComputeEngagementCoefficient(samples)
	if !errors.Is(err, ErrInsufficientCoefHistory) {
		t.Errorf("want ErrInsufficientCoefHistory with 9 samples, got %v", err)
	}
}

func TestComputeEngagementCoefficient_ExactlyMinThreshold(t *testing.T) {
	t.Parallel()
	// Pile MinMatchesForCoef samples — doit calculer
	samples := constantRatioSamples(MinMatchesForCoef, 1.2, 1.1)
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("want OK at MinMatchesForCoef, got %v", err)
	}
	if res.NMatches != MinMatchesForCoef {
		t.Errorf("NMatches want %d, got %d", MinMatchesForCoef, res.NMatches)
	}
}

// =============================================================================
// Mediane stable et correcte
// =============================================================================

func TestComputeEngagementCoefficient_ConstantRatio(t *testing.T) {
	t.Parallel()
	samples := constantRatioSamples(50, 1.2, 1.05)
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if math.Abs(res.CoefTeamShare-1.2) > 1e-9 {
		t.Errorf("CoefTeamShare want 1.2, got %v", res.CoefTeamShare)
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
	// 11 samples avec ratios distincts — la mediane est la 6e valeur triee
	ratios := []float64{0.5, 0.7, 0.9, 1.0, 1.1, 1.2, 1.3, 1.4, 1.5, 1.7, 2.0}
	samples := make([]RatioSample, len(ratios))
	for i, r := range ratios {
		samples[i] = validSample(fmt.Sprintf("m%d", i), r, r)
	}
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	// Mediane de 11 valeurs triees = la 6e (index 5) = 1.2
	if math.Abs(res.CoefTeamShare-1.2) > 1e-9 {
		t.Errorf("CoefTeamShare want 1.2 (median of 11 ratios), got %v", res.CoefTeamShare)
	}
}

func TestComputeEngagementCoefficient_EvenSampleSize(t *testing.T) {
	t.Parallel()
	// 12 samples — mediane = (6e + 7e) / 2
	ratios := []float64{0.5, 0.7, 0.9, 1.0, 1.1, 1.2, 1.3, 1.4, 1.5, 1.7, 2.0, 2.5}
	samples := make([]RatioSample, len(ratios))
	for i, r := range ratios {
		samples[i] = validSample(fmt.Sprintf("m%d", i), r, r)
	}
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	// Mediane de 12 valeurs = (1.2 + 1.3) / 2 = 1.25
	if math.Abs(res.CoefTeamShare-1.25) > 1e-9 {
		t.Errorf("CoefTeamShare want 1.25 (median of 12 ratios), got %v", res.CoefTeamShare)
	}
}

func TestComputeEngagementCoefficient_MedianRobustToOutliers(t *testing.T) {
	t.Parallel()
	// 90 ratios constants 1.0 + 10 ratios extremes 100.0 → mediane reste 1.0
	samples := make([]RatioSample, 0, 100)
	for i := 0; i < 90; i++ {
		samples = append(samples, validSample(fmt.Sprintf("normal%d", i), 1.0, 1.0))
	}
	for i := 0; i < 10; i++ {
		// Ratio extreme : pace_joueur enorme, sera clamp par CoefMax mais
		// l'important est que la mediane n'est pas tiree par les outliers.
		samples = append(samples, RatioSample{
			MatchID:        fmt.Sprintf("outlier%d", i),
			PaceJoueur:     1000,
			PaceTeam:       10,
			PaceLobby:      10,
			PlayerActivity: 50,
		})
	}
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	// Mediane avec 90% normaux → 1.0
	if math.Abs(res.CoefTeamShare-1.0) > 1e-9 {
		t.Errorf("median should resist outliers, want ~1.0, got %v", res.CoefTeamShare)
	}
}

// =============================================================================
// Filtres et exclusions (PlayerActivity, PaceTeam, PaceLobby)
// =============================================================================

func TestComputeEngagementCoefficient_FilterAFKPlayerActivity(t *testing.T) {
	t.Parallel()
	// 15 samples valides + 10 samples "quitter" (activity=2) → seuls les 15
	// valides comptent
	samples := constantRatioSamples(15, 1.5, 1.5)
	for i := 0; i < 10; i++ {
		samples = append(samples, RatioSample{
			MatchID:        fmt.Sprintf("quit%d", i),
			PaceJoueur:     0.5, // ratio aberrant qui DEVRAIT polluer si non filtre
			PaceTeam:       10,
			PaceLobby:      10,
			PlayerActivity: 2, // sous le seuil PlayerActivityMin=3
		})
	}
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if math.Abs(res.CoefTeamShare-1.5) > 1e-9 {
		t.Errorf("AFK samples should be filtered, want CoefTeam=1.5, got %v", res.CoefTeamShare)
	}
	if res.NRejected != 10 {
		t.Errorf("NRejected want 10 (AFK), got %d", res.NRejected)
	}
	if res.NMatches != 15 {
		t.Errorf("NMatches want 15, got %d", res.NMatches)
	}
}

func TestComputeEngagementCoefficient_FilterLowPaceTeam(t *testing.T) {
	t.Parallel()
	// 12 samples normaux + 5 samples "lobby AFK" (pace_team < 1.0) qui doivent
	// etre exclus du ratio team.
	samples := constantRatioSamples(12, 1.0, 1.0)
	for i := 0; i < 5; i++ {
		samples = append(samples, RatioSample{
			MatchID:        fmt.Sprintf("afkLobby%d", i),
			PaceJoueur:     5,
			PaceTeam:       0.5, // < PaceTeamMinThreshold=1.0
			PaceLobby:      0.4, // < PaceLobbyMinThreshold=1.0
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
	// Sample avec PaceTeam=0 (division par 0 a eviter)
	samples := constantRatioSamples(15, 1.0, 1.0)
	samples = append(samples, RatioSample{
		MatchID:        "zero",
		PaceJoueur:     5,
		PaceTeam:       0, // explicit zero
		PaceLobby:      0,
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
// Coef lobby independant du coef team
// =============================================================================

func TestComputeEngagementCoefficient_LobbyIndependent(t *testing.T) {
	t.Parallel()
	// 15 samples avec ratio team=1.2 et ratio lobby=0.8 — les deux coefs
	// doivent refleter ces deux mediane independantes.
	samples := constantRatioSamples(15, 1.2, 0.8)
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if math.Abs(res.CoefTeamShare-1.2) > 1e-9 {
		t.Errorf("CoefTeamShare want 1.2, got %v", res.CoefTeamShare)
	}
	if math.Abs(res.CoefLobbyShare-0.8) > 1e-9 {
		t.Errorf("CoefLobbyShare want 0.8, got %v", res.CoefLobbyShare)
	}
}

func TestComputeEngagementCoefficient_LobbyFallbackWhenInsufficient(t *testing.T) {
	t.Parallel()
	// 15 samples avec PaceLobby valide + 0 sample lobby → coefLobby fallback 1.0
	// On simule cela avec PaceLobby<seuil pour tous mais PaceTeam OK.
	samples := make([]RatioSample, 15)
	for i := range samples {
		samples[i] = RatioSample{
			MatchID:        fmt.Sprintf("m%d", i),
			PaceJoueur:     12,
			PaceTeam:       10,
			PaceLobby:      0.5, // sous seuil PaceLobbyMinThreshold=1.0
			PlayerActivity: 30,
		}
	}
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if math.Abs(res.CoefTeamShare-1.2) > 1e-9 {
		t.Errorf("CoefTeamShare want 1.2, got %v", res.CoefTeamShare)
	}
	// Lobby fallback : pas assez de samples lobby valides → 1.0 neutre
	if res.CoefLobbyShare != 1.0 {
		t.Errorf("CoefLobbyShare fallback want 1.0, got %v", res.CoefLobbyShare)
	}
}

// =============================================================================
// Bornage (clampCoef)
// =============================================================================

func TestComputeEngagementCoefficient_ClampHighCoef(t *testing.T) {
	t.Parallel()
	// Tous les samples avec ratio=10 (super-carry irrealiste) → clamp a CoefMax=5.0
	samples := make([]RatioSample, 15)
	for i := range samples {
		samples[i] = RatioSample{
			MatchID:        fmt.Sprintf("m%d", i),
			PaceJoueur:     100,
			PaceTeam:       10,
			PaceLobby:      10,
			PlayerActivity: 30,
		}
	}
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if res.CoefTeamShare != CoefMax {
		t.Errorf("clampHigh want %v, got %v", CoefMax, res.CoefTeamShare)
	}
	if res.CoefLobbyShare != CoefMax {
		t.Errorf("clampHigh lobby want %v, got %v", CoefMax, res.CoefLobbyShare)
	}
}

func TestComputeEngagementCoefficient_ClampLowCoef(t *testing.T) {
	t.Parallel()
	// Tous les samples avec ratio=0.01 (joueur quasi-inactif mais activity OK)
	samples := make([]RatioSample, 15)
	for i := range samples {
		samples[i] = RatioSample{
			MatchID:        fmt.Sprintf("m%d", i),
			PaceJoueur:     0.05,
			PaceTeam:       10,
			PaceLobby:      10,
			PlayerActivity: 5,
		}
	}
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if res.CoefTeamShare != CoefMin {
		t.Errorf("clampLow want %v, got %v", CoefMin, res.CoefTeamShare)
	}
}

// =============================================================================
// Determinisme et stabilite
// =============================================================================

func TestComputeEngagementCoefficient_DeterministicOrder(t *testing.T) {
	t.Parallel()
	// Ordre des samples ne doit pas affecter le resultat (mediane = invariant
	// permutation).
	samples1 := []RatioSample{
		validSample("a", 0.8, 0.9),
		validSample("b", 1.0, 1.1),
		validSample("c", 1.2, 1.0),
		validSample("d", 1.4, 1.2),
		validSample("e", 1.5, 1.3),
		validSample("f", 1.0, 0.95),
		validSample("g", 1.1, 1.05),
		validSample("h", 0.9, 0.85),
		validSample("i", 1.3, 1.15),
		validSample("j", 1.05, 1.0),
	}
	samples2 := make([]RatioSample, len(samples1))
	copy(samples2, samples1)
	// Inverse l'ordre
	for i, j := 0, len(samples2)-1; i < j; i, j = i+1, j-1 {
		samples2[i], samples2[j] = samples2[j], samples2[i]
	}
	r1, _ := ComputeEngagementCoefficient(samples1)
	r2, _ := ComputeEngagementCoefficient(samples2)
	if math.Abs(r1.CoefTeamShare-r2.CoefTeamShare) > 1e-12 {
		t.Errorf("order matters: %v vs %v", r1.CoefTeamShare, r2.CoefTeamShare)
	}
	if math.Abs(r1.CoefLobbyShare-r2.CoefLobbyShare) > 1e-12 {
		t.Errorf("lobby order matters: %v vs %v", r1.CoefLobbyShare, r2.CoefLobbyShare)
	}
}

func TestComputeEngagementCoefficient_GaussianDistribution(t *testing.T) {
	t.Parallel()
	// 200 samples gaussiens centres sur 1.15, sigma 0.2 → mediane ≈ 1.15
	rng := rand.New(rand.NewSource(42))
	const targetMedian = 1.15
	samples := make([]RatioSample, 200)
	for i := range samples {
		ratio := targetMedian + rng.NormFloat64()*0.2
		// Borner pour rester dans une plage realiste (et ne pas etre clamp)
		if ratio < 0.5 {
			ratio = 0.5
		}
		if ratio > 2.5 {
			ratio = 2.5
		}
		samples[i] = validSample(fmt.Sprintf("m%d", i), ratio, ratio)
	}
	res, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	// Mediane d'un echantillon gaussien doit etre proche de la mediane theorique.
	// Tolerance 5% sur 200 samples — large pour eviter flakiness.
	if math.Abs(res.CoefTeamShare-targetMedian) > 0.05 {
		t.Errorf("gaussian median should be near %v, got %v", targetMedian, res.CoefTeamShare)
	}
}

// =============================================================================
// Mutation safety
// =============================================================================

func TestComputeEngagementCoefficient_DoesNotMutateInput(t *testing.T) {
	t.Parallel()
	samples := constantRatioSamples(20, 1.3, 1.1)
	// Snapshot avant
	snapshot := make([]RatioSample, len(samples))
	copy(snapshot, samples)
	_, err := ComputeEngagementCoefficient(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	// Verifie qu'aucun sample n'a ete modifie
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
