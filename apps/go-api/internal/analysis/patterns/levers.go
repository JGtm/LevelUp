package patterns

import (
	"math"
	"slices"
	"sort"
)

// levers.go — sélection et calibration des leviers d'amélioration.

// selectLevers construit la liste des leviers depuis les patterns contextuels
// et comportementaux. Retourne les 5 premiers triés par Impact décroissant.
func selectLevers(ctx []ContextualPattern, beh []BehavioralPattern, rows []MatchRow, cfg PatternConfig) []Lever {
	globalWR := globalWinRate(rows)
	var levers []Lever
	levers = append(levers, leversFromContext(ctx, globalWR)...)
	levers = append(levers, leversFromBehavior(beh, rows, cfg)...)

	// Trier par Impact décroissant
	sort.SliceStable(levers, func(i, j int) bool {
		return levers[i].Impact > levers[j].Impact
	})

	// Garder max 5, assigner le Rank (1-based)
	const maxLevers = 5
	if len(levers) > maxLevers {
		levers = levers[:maxLevers]
	}
	for i := range levers {
		levers[i].Rank = i + 1
	}
	return levers
}

// leversFromContext crée un Lever pour chaque ContextualPattern avec Signal==Weakness.
func leversFromContext(ctx []ContextualPattern, globalWR float64) []Lever {
	var out []Lever
	for _, p := range ctx {
		if p.Signal != SignalWeakness {
			continue
		}
		axis := contextAxis(p.Type)
		target := globalWR - 0.05
		if target < 0 {
			target = 0
		}
		// Horizon estimé : gap / 0.05 * 10, borné [10, 100]
		gap := math.Abs(p.Delta)
		horizon := int(gap / 0.05 * 10)
		if horizon < 10 {
			horizon = 10
		}
		if horizon > 100 {
			horizon = 100
		}
		out = append(out, Lever{
			Axis:       axis,
			ContextKey: p.Key,
			CurrentVal: p.WinRate,
			TargetVal:  target,
			Horizon:    horizon,
			Impact:     math.Abs(p.Delta),
		})
	}
	return out
}

// contextAxis mappe un ContextType vers un nom d'axe de levier.
func contextAxis(ct ContextType) string {
	switch ct {
	case ContextByMode:
		return AxisModeSelection
	case ContextByMap:
		return AxisMapAvoidance
	case ContextBySquad:
		return AxisSquadPlay
	default:
		return string(ct)
	}
}

// leversFromBehavior crée un Lever pour chaque BehavioralPattern de sévérité >= Medium.
func leversFromBehavior(beh []BehavioralPattern, rows []MatchRow, _ PatternConfig) []Lever {
	var out []Lever
	for _, p := range beh {
		if p.Severity == SeverityLow {
			continue
		}
		var lev Lever
		switch p.Type {
		case BehaviorTilt:
			lev = Lever{
				Axis:       AxisSessionManagement,
				CurrentVal: 0,
				TargetVal:  0,
				Horizon:    20,
				Impact:     0.4,
			}
		case BehaviorSessionFatigue:
			lev = Lever{
				Axis:       AxisSessionLength,
				CurrentVal: 0,
				TargetVal:  0,
				Horizon:    15,
				Impact:     0.3,
			}
		case BehaviorEngagementDrop:
			lev = Lever{
				Axis:       AxisEngagement,
				CurrentVal: 0,
				TargetVal:  0,
				Horizon:    20,
				Impact:     0.35,
			}
		case BehaviorAccuracyPlateau:
			accs := collectAccuracies(rows)
			current := meanFloat(accs)
			target, ok := calibrateTarget(accs)
			if !ok {
				target = current + 0.05
			}
			lev = Lever{
				Axis:       AxisAccuracy,
				CurrentVal: current,
				TargetVal:  target,
				Horizon:    30,
				Impact:     0.3,
			}
		case BehaviorPerfCeiling:
			lev = Lever{
				Axis:       AxisRadarAxis,
				CurrentVal: 0,
				TargetVal:  0,
				Horizon:    40,
				Impact:     0.25,
			}
		default:
			continue
		}
		out = append(out, lev)
	}
	return out
}

// calibrateTarget retourne le P60 d'une slice comme objectif cible.
// Retourne (0, false) si moins de 10 valeurs ou si P60 <= mean.
func calibrateTarget(values []float64) (float64, bool) {
	if len(values) < 10 {
		return 0, false
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	slices.Sort(sorted)
	p60 := sorted[int(0.60*float64(len(sorted)))]
	m := meanFloat(values)
	if p60 <= m {
		return 0, false
	}
	return p60, true
}

// collectAccuracies extrait les valeurs d'Accuracy de toutes les rows.
func collectAccuracies(rows []MatchRow) []float64 {
	out := make([]float64, len(rows))
	for i, r := range rows {
		out[i] = r.Accuracy
	}
	return out
}
