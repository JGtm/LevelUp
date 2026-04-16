// Package analysis — squad_score.go : score collectif et grade escouade.
package analysis

import (
	"math"

	"levelup/go-api/internal/domain"
)

// =============================================================================
// Score collectif escouade — miroir de compute_squad_performance_score()
// =============================================================================

// ComputeSquadPerformanceScore calcule le score collectif d'une escouade.
//
// Paramètres :
//   - scores       : slice des performance_score individuels (0-100) — les nil sont ignorés.
//   - winRates     : slice des win_rate individuels (0-100).
//   - kdaRatios    : slice des KDA individuels.
//   - killsList    : slice des kill counts individuels pour le bonus d'équilibre.
func ComputeSquadPerformanceScore(
	scores []*float64,
	winRates []float64,
	kdaRatios []float64,
	killsList []float64,
) domain.SquadPerformanceScore {
	// Base : moyenne des scores non-nil.
	var valid []float64
	for _, s := range scores {
		if s != nil {
			valid = append(valid, *s)
		}
	}
	if len(valid) == 0 {
		return domain.SquadPerformanceScore{
			Score:      nil,
			Grade:      "N/A",
			Components: map[string]interface{}{},
		}
	}
	var sum float64
	for _, v := range valid {
		sum += v
	}
	base := sum / float64(len(valid))

	bonus := 0.0
	comps := map[string]interface{}{}

	// +5 si win_rate moyen > 60 %.
	if len(winRates) > 0 {
		var wrSum float64
		for _, wr := range winRates {
			wrSum += wr
		}
		teamWR := wrSum / float64(len(winRates))
		comps["team_win_rate"] = math.Round(teamWR*10) / 10
		if teamWR > 60.0 {
			bonus += 5.0
		}
	}

	// +5 si min(KDA) > 1.0.
	if len(kdaRatios) > 0 {
		minKDA := kdaRatios[0]
		for _, k := range kdaRatios[1:] {
			if k < minKDA {
				minKDA = k
			}
		}
		if minKDA > 1.0 {
			comps["min_kd"] = math.Round(minKDA*100) / 100
			bonus += 5.0
		}
	}

	// +3 si écart-type des kills < 3.0.
	if len(killsList) >= 2 {
		var kSum float64
		for _, k := range killsList {
			kSum += k
		}
		mean := kSum / float64(len(killsList))
		var variance float64
		for _, k := range killsList {
			diff := k - mean
			variance += diff * diff
		}
		variance /= float64(len(killsList))
		std := math.Sqrt(variance)
		comps["kills_std"] = math.Round(std*10) / 10
		if std < 3.0 {
			bonus += 3.0
		}
	}

	final := clampFloat(base+bonus, 0.0, 100.0)
	finalRounded := math.Round(final*10) / 10
	comps["base_avg"] = math.Round(base*10) / 10

	return domain.SquadPerformanceScore{
		Score:      &finalRounded,
		Grade:      resolveSquadGrade(finalRounded),
		Components: comps,
	}
}

// resolveSquadGrade retourne le grade lettre d'un score 0-100.
// Miroir de Python resolve_squad_grade().
func resolveSquadGrade(score float64) string {
	switch {
	case score >= 90:
		return "S"
	case score >= 80:
		return "A"
	case score >= 65:
		return "B"
	case score >= 50:
		return "C"
	case score >= 35:
		return "D"
	default:
		return "F"
	}
}

// =============================================================================
// Helpers internes
// =============================================================================

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
