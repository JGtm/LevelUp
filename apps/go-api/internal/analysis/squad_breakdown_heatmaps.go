package analysis

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

func ComputeComparisonMetrics(solo, squad domain.SynthesisKPIs) []domain.ComparisonMetricItem {
	items := make([]domain.ComparisonMetricItem, 0, 18)
	items = append(items, domain.ComparisonMetricItem{
		Label: "match_count", SoloValue: float64(solo.MatchCount), SquadValue: float64(squad.MatchCount),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "ranked_match_count", SoloValue: float64(solo.RankedMatchCount), SquadValue: float64(squad.RankedMatchCount),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "time_played_seconds", SoloValue: float64(solo.TotalTimePlayedSeconds), SquadValue: float64(squad.TotalTimePlayedSeconds),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "win_rate", SoloValue: solo.WinRate, SquadValue: squad.WinRate,
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "performance_score", SoloValue: deref(solo.PerformanceScore), SquadValue: deref(squad.PerformanceScore),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "kd_ratio", SoloValue: deref(solo.KDRatio), SquadValue: deref(squad.KDRatio),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "kills_per_min", SoloValue: deref(solo.KillsPerMin), SquadValue: deref(squad.KillsPerMin),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "deaths_per_min", SoloValue: deref(solo.DeathsPerMin), SquadValue: deref(squad.DeathsPerMin),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "assists_per_min", SoloValue: deref(solo.AssistsPerMin), SquadValue: deref(squad.AssistsPerMin),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "avg_life_seconds", SoloValue: deref(solo.AvgLifeSeconds), SquadValue: deref(squad.AvgLifeSeconds),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: StatLabelAccuracy, SoloValue: deref(solo.Accuracy), SquadValue: deref(squad.Accuracy),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "headshots_per_match", SoloValue: deref(solo.HeadshotsPerMatch), SquadValue: deref(squad.HeadshotsPerMatch),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "perfect_kills_per_match", SoloValue: deref(solo.PerfectKillsPerMatch), SquadValue: deref(squad.PerfectKillsPerMatch),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "avg_max_killing_spree", SoloValue: deref(solo.AvgMaxKillingSpree), SquadValue: deref(squad.AvgMaxKillingSpree),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "avg_damage_dealt", SoloValue: deref(solo.AvgDamageDealt), SquadValue: deref(squad.AvgDamageDealt),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "offensive_conversion", SoloValue: deref(solo.AvgOffensiveConversion), SquadValue: deref(squad.AvgOffensiveConversion),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "avg_damage_taken", SoloValue: deref(solo.AvgDamageTaken), SquadValue: deref(squad.AvgDamageTaken),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "defensive_resistance", SoloValue: deref(solo.AvgDefensiveResistance), SquadValue: deref(squad.AvgDefensiveResistance),
	})
	return items
}

// ComputeTemporalHeatmap construit la heatmap jour Ã— heure depuis les matchs.
func ComputeTemporalHeatmap(rows []legacymatch.SynthesisMatchRow) []domain.TemporalHeatmapCell {
	type heatmapAgg struct {
		count int
		wins  int
	}
	cells := [7][24]heatmapAgg{}
	for _, r := range rows {
		// Go Weekday: Sunday=0, Monday=1 ... Saturday=6
		// Frontend: lundi=0 ... dimanche=6
		goDow := int(r.StartTime.Weekday())
		dow := (goDow + 6) % 7 // convertir: lundi=0
		hour := r.StartTime.Hour()
		agg := cells[dow][hour]
		agg.count++
		if r.Outcome == domain.OutcomeWin {
			agg.wins++
		}
		cells[dow][hour] = agg
	}
	result := []domain.TemporalHeatmapCell{}
	for d := 0; d < 7; d++ {
		for h := 0; h < 24; h++ {
			agg := cells[d][h]
			if agg.count > 0 {
				wr := float64(agg.wins) / float64(agg.count)
				result = append(result, domain.TemporalHeatmapCell{
					DOW:     d,
					Hour:    h,
					Count:   agg.count,
					Wins:    agg.wins,
					WinRate: wr,
				})
			}
		}
	}
	return result
}

// ComputeActivityHeatmapFromCommonMatches construit la heatmap jour × heure
// depuis la liste brute des matchs communs entre deux joueurs (Explorer mode
// Joueur). Même logique d'agrégation que ComputeTemporalHeatmap : conversion
// Go Weekday → dow 0=lundi…6=dimanche, heure UTC, comptage win sur outcome=2.
//
// Renseigne count + wins + win_rate pour réutiliser le type TemporalHeatmapCell.
// Côté frontend, c'est `count` qui pilote la coloration (intensité d'activité
// commune) ; `win_rate` reste exposé pour le tooltip.
func ComputeActivityHeatmapFromCommonMatches(rows []domain.CommonMatchRaw) []domain.TemporalHeatmapCell {
	type heatmapAgg struct {
		count int
		wins  int
	}
	cells := [7][24]heatmapAgg{}
	for _, r := range rows {
		goDow := int(r.StartTime.Weekday())
		dow := (goDow + 6) % 7 // Sunday=0…Saturday=6 → lundi=0…dimanche=6
		hour := r.StartTime.Hour()
		agg := cells[dow][hour]
		agg.count++
		if r.Player1Outcome == domain.OutcomeWin {
			agg.wins++
		}
		cells[dow][hour] = agg
	}
	result := []domain.TemporalHeatmapCell{}
	for d := 0; d < 7; d++ {
		for h := 0; h < 24; h++ {
			agg := cells[d][h]
			if agg.count > 0 {
				wr := float64(agg.wins) / float64(agg.count)
				result = append(result, domain.TemporalHeatmapCell{
					DOW:     d,
					Hour:    h,
					Count:   agg.count,
					Wins:    agg.wins,
					WinRate: wr,
				})
			}
		}
	}
	return result
}

func deref(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}
