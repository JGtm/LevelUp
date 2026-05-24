package patterns

import (
	"fmt"
	"math"
	"slices"

	"levelup/go-api/internal/analysis/temporal"
)

// behavioral_engagement.go — détecteurs engagement_drop, accuracy_plateau, perf_ceiling.
//
// Séparé de behavioral.go pour respecter la limite de 500 lignes par fichier.

// detectEngagementDrop détecte une chute combinée EngageScore + ResidualBrut.
func detectEngagementDrop(rows []MatchRow, cfg PatternConfig) (BehavioralPattern, bool) {
	minRows := cfg.MinMatchesPerGroup

	var fullRows []MatchRow
	for _, r := range rows {
		if r.EngageScore != nil && r.ResidualBrut != nil {
			fullRows = append(fullRows, r)
		}
	}
	if len(fullRows) < minRows {
		return BehavioralPattern{}, false
	}

	p25Engage := percentile(collectEngageScores(fullRows), 0.25)
	p25Residual := percentile(collectResidualBrut(fullRows), 0.25)

	recent := fullRows
	if len(recent) > cfg.EngageDropWindow {
		recent = recent[:cfg.EngageDropWindow]
	}
	if len(recent) < minRows {
		return BehavioralPattern{}, false
	}

	dropCount := 0
	for _, r := range recent {
		if *r.EngageScore <= p25Engage && *r.ResidualBrut <= p25Residual {
			dropCount++
		}
	}
	if dropCount < minRows {
		return BehavioralPattern{}, false
	}

	sev := SeverityMedium
	if dropCount >= cfg.EngageDropHighThreshold {
		sev = SeverityHigh
	}

	avgEngage := meanNonNilEngageScore(recent)
	avgEngageAll := meanNonNilEngageScore(fullRows)
	engagePct := 100.0
	if avgEngageAll > 0 {
		engagePct = avgEngage / avgEngageAll * 100
	}
	avgResidual := meanNonNilResidualBrut(recent)
	avgResidualAll := meanNonNilResidualBrut(fullRows)
	residualPct := 100.0
	if avgResidualAll > 0 {
		residualPct = avgResidual / avgResidualAll * 100
	}

	return BehavioralPattern{
		Type:      BehaviorEngagementDrop,
		Trigger:   fmt.Sprintf("%d matchs récents sous les seuils habituels", dropCount),
		Evidence:  fmt.Sprintf("Engagement perso : %.0f%% vs habitude ; %.0f%% vs le lobby", engagePct, residualPct),
		Severity:  sev,
		Confirmed: false,
	}, true
}

// detectAccuracyPlateau détecte une précision stable mais basse.
func detectAccuracyPlateau(rows []MatchRow, cfg PatternConfig) (BehavioralPattern, bool) {
	const plateauWindow = 30
	window := rows
	if len(window) > plateauWindow {
		window = window[:plateauWindow]
	}
	if len(window) == 0 {
		return BehavioralPattern{}, false
	}
	accs := make([]float64, len(window))
	for i, r := range window {
		accs[i] = r.Accuracy
	}
	m := meanFloat(accs)
	std := stddev(accs, m)
	if std >= cfg.AccuracyPlateauStd || m >= cfg.AccuracyPlateauMax {
		return BehavioralPattern{}, false
	}
	sev := SeverityMedium
	if m > 0.35 {
		sev = SeverityLow
	} else if m < 0.25 {
		sev = SeverityHigh
	}
	return BehavioralPattern{
		Type:      BehaviorAccuracyPlateau,
		Trigger:   "30 matchs sans progression",
		Evidence:  fmt.Sprintf("Précision stable à %.1f%% (σ=%.3f)", m*100, std),
		Severity:  sev,
		Confirmed: false,
	}, true
}

// detectPerfCeiling détecte un plafond de performance via LOWESS.
func detectPerfCeiling(rows []MatchRow, cfg PatternConfig) (BehavioralPattern, bool) {
	var perfRows []MatchRow
	for _, r := range rows {
		if r.PerfScore != nil {
			perfRows = append(perfRows, r)
		}
	}
	if len(perfRows) < cfg.PerfCeilingMinRows {
		return BehavioralPattern{}, false
	}

	window := perfRows
	if len(window) > cfg.PerfCeilingWindow {
		window = window[:cfg.PerfCeilingWindow]
	}

	scores := make([]float64, len(window))
	for i, r := range window {
		scores[i] = *r.PerfScore
	}

	maxScore := scores[0]
	for _, s := range scores {
		if s > maxScore {
			maxScore = s
		}
	}

	sorted := make([]float64, len(scores))
	copy(sorted, scores)
	slices.Sort(sorted)
	top := sorted
	if len(top) > cfg.PerfCeilingTopN {
		top = top[len(top)-cfg.PerfCeilingTopN:]
	}
	meanTop := meanFloat(top)

	// Condition 1 : le max ne dépasse pas significativement la moyenne des tops
	if maxScore-meanTop >= 5 {
		return BehavioralPattern{}, false
	}

	// Condition 2 : pente LOWESS ≈ plate
	smoothed := temporal.LowessSmooth(scores, cfg.PerfCeilingLowessAlpha)
	slope := estimateLowessSlope(smoothed)
	if math.Abs(slope) >= cfg.PerfCeilingFlatSlopeThresh {
		return BehavioralPattern{}, false
	}

	return BehavioralPattern{
		Type:      BehaviorPerfCeiling,
		Trigger:   "Progression LOWESS plate sur 30+ matchs",
		Evidence:  fmt.Sprintf("Plafond de performance à %.0f pts (top %d moyen : %.0f)", maxScore, cfg.PerfCeilingTopN, meanTop),
		Severity:  SeverityMedium,
		Confirmed: false,
	}, true
}

// estimateLowessSlope estime la pente globale d'une série lissée.
func estimateLowessSlope(smoothed []float64) float64 {
	var valid []float64
	for _, v := range smoothed {
		if !math.IsNaN(v) {
			valid = append(valid, v)
		}
	}
	return linearSlope(valid)
}

// stddev calcule l'écart-type d'une slice.
func stddev(vals []float64, mean float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		d := v - mean
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(vals)))
}

// percentile retourne le percentile p (0..1) d'une slice.
func percentile(vals []float64, p float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	slices.Sort(sorted)
	idx := int(p * float64(len(sorted)))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// collectEngageScores extrait les EngageScore non nil.
func collectEngageScores(rows []MatchRow) []float64 {
	var out []float64
	for _, r := range rows {
		if r.EngageScore != nil {
			out = append(out, *r.EngageScore)
		}
	}
	return out
}

// collectResidualBrut extrait les ResidualBrut non nil.
func collectResidualBrut(rows []MatchRow) []float64 {
	var out []float64
	for _, r := range rows {
		if r.ResidualBrut != nil {
			out = append(out, *r.ResidualBrut)
		}
	}
	return out
}

// meanNonNilEngageScore calcule la moyenne des EngageScore non nil.
func meanNonNilEngageScore(rows []MatchRow) float64 {
	return meanFloat(collectEngageScores(rows))
}

// meanNonNilResidualBrut calcule la moyenne des ResidualBrut non nil.
func meanNonNilResidualBrut(rows []MatchRow) float64 {
	return meanFloat(collectResidualBrut(rows))
}
