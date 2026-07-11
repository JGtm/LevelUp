package temporal

import (
	"errors"
	"fmt"
	"math"
	"testing"
)

// binRatioSample construit un sample valide avec une intensite (paceLobby) et un
// ratio de reponse voulus.
func binRatioSample(matchID string, paceLobby, ratio float64) RatioSample {
	return RatioSample{
		MatchID:        matchID,
		PaceLobby:      paceLobby,
		PaceJoueur:     paceLobby * ratio,
		PaceTeam:       paceLobby, // non utilise par les bins
		PlayerActivity: 30,
	}
}

// makeBinBatch genere n samples a une intensite et un ratio donnes.
func makeBinBatch(prefix string, n int, paceLobby, ratio float64) []RatioSample {
	out := make([]RatioSample, n)
	for i := range out {
		out[i] = binRatioSample(fmt.Sprintf("%s_%d", prefix, i), paceLobby, ratio)
	}
	return out
}

func TestComputeEngagementResponseBins_Insufficient(t *testing.T) {
	// 9 samples valides < MinMatchesForCoef → ErrInsufficientBinHistory.
	samples := makeBinBatch("m", 9, 5.0, 1.0)
	_, err := ComputeEngagementResponseBins(samples)
	if !errors.Is(err, ErrInsufficientBinHistory) {
		t.Fatalf("want ErrInsufficientBinHistory with 9 samples, got %v", err)
	}
}

// TestComputeEngagementResponseBins_ThreeTercilesCoefs : un joueur qui repond
// bien aux matchs calmes (ratio 1.5) et mal aux matchs chaotiques (ratio 0.5)
// doit produire coef(calme) > coef(standard) > coef(chaotique).
func TestComputeEngagementResponseBins_ThreeTercilesCoefs(t *testing.T) {
	var samples []RatioSample
	samples = append(samples, makeBinBatch("calme", 15, 2.0, 1.5)...)
	samples = append(samples, makeBinBatch("standard", 15, 5.0, 1.0)...)
	samples = append(samples, makeBinBatch("chaotique", 15, 10.0, 0.5)...)

	res, err := ComputeEngagementResponseBins(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if len(res.Bins) != 3 {
		t.Fatalf("want 3 bins, got %d", len(res.Bins))
	}
	byName := map[string]float64{}
	nByName := map[string]int{}
	for _, b := range res.Bins {
		byName[b.Bin] = b.CoefLobby
		nByName[b.Bin] = b.NMatches
	}
	if math.Abs(byName[IntensityBinCalme]-1.5) > 1e-9 {
		t.Errorf("coef calme want 1.5, got %v", byName[IntensityBinCalme])
	}
	if math.Abs(byName[IntensityBinStandard]-1.0) > 1e-9 {
		t.Errorf("coef standard want 1.0, got %v", byName[IntensityBinStandard])
	}
	if math.Abs(byName[IntensityBinChaotique]-0.5) > 1e-9 {
		t.Errorf("coef chaotique want 0.5, got %v", byName[IntensityBinChaotique])
	}
	// Le trait « repond mal aux matchs intenses » : coef decroissant avec l'intensite.
	if !(byName[IntensityBinCalme] > byName[IntensityBinStandard] &&
		byName[IntensityBinStandard] > byName[IntensityBinChaotique]) {
		t.Errorf("coefs doivent decroitre avec l'intensite : %v", byName)
	}
	for _, name := range []string{IntensityBinCalme, IntensityBinStandard, IntensityBinChaotique} {
		if nByName[name] != 15 {
			t.Errorf("bin %s : n want 15, got %d", name, nByName[name])
		}
	}
}

func TestComputeEngagementResponseBins_FilterAFKAndLowPace(t *testing.T) {
	var samples []RatioSample
	samples = append(samples, makeBinBatch("valid", 12, 5.0, 1.0)...)
	// 4 AFK (activity < 3)
	for i := 0; i < 4; i++ {
		s := binRatioSample(fmt.Sprintf("afk_%d", i), 5.0, 3.0)
		s.PlayerActivity = 2
		samples = append(samples, s)
	}
	// 3 lobby quasi-inactif (paceLobby < seuil 0.75)
	for i := 0; i < 3; i++ {
		samples = append(samples, binRatioSample(fmt.Sprintf("low_%d", i), 0.5, 3.0))
	}
	res, err := ComputeEngagementResponseBins(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if res.NRejected != 7 {
		t.Errorf("NRejected want 7 (4 AFK + 3 low pace), got %d", res.NRejected)
	}
	// Les 12 valides tous a ratio 1.0 → chaque bin non vide a coef 1.0.
	for _, b := range res.Bins {
		if b.NMatches > 0 && math.Abs(b.CoefLobby-1.0) > 1e-9 {
			t.Errorf("bin %s coef want 1.0, got %v", b.Bin, b.CoefLobby)
		}
	}
}

func TestComputeEngagementResponseBins_ClampExtremeCoef(t *testing.T) {
	// Ratio 100 (super-carry irrealiste) → coef clampe a CoefMax.
	samples := makeBinBatch("m", 15, 5.0, 100.0)
	res, err := ComputeEngagementResponseBins(samples)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	for _, b := range res.Bins {
		if b.NMatches > 0 && b.CoefLobby != CoefMax {
			t.Errorf("bin %s coef want clamp %v, got %v", b.Bin, CoefMax, b.CoefLobby)
		}
	}
}
