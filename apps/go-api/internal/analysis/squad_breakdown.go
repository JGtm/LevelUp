// Package analysis — squad_breakdown.go : breakdown solo/escouade + synthèse + top weeks.
package analysis

import (
	"fmt"
	"math"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
)

// =============================================================================
// Breakdown solo vs escouade
// =============================================================================

// ComputeSquadBreakdown calcule les stats agrégées pour un mode (solo ou escouade).
// rows doit déjà être filtré (is_with_friends=true ou false).
func ComputeSquadBreakdown(rows []domain.SquadMatchRow) domain.SquadBreakdownStats {
	if len(rows) == 0 {
		return domain.SquadBreakdownStats{}
	}
	var wins, totalWL int
	var sumKDA, sumKills float64
	var nKDA, nKills int

	for _, r := range rows {
		if r.Outcome == domain.OutcomeWin || r.Outcome == domain.OutcomeLoss {
			totalWL++
			if r.Outcome == domain.OutcomeWin {
				wins++
			}
		}
		if r.KDA != nil {
			sumKDA += *r.KDA
			nKDA++
		}
		sumKills += float64(r.Kills)
		nKills++
	}

	var wr float64
	if totalWL > 0 {
		wr = math.Round(float64(wins)/float64(totalWL)*1000) / 10
	}
	var avgKDA float64
	if nKDA > 0 {
		avgKDA = math.Round(sumKDA/float64(nKDA)*100) / 100
	}
	var avgKills float64
	if nKills > 0 {
		avgKills = math.Round(sumKills/float64(nKills)*10) / 10
	}

	return domain.SquadBreakdownStats{
		MatchCount: len(rows),
		WinRate:    wr,
		AvgKDA:     avgKDA,
		AvgKills:   avgKills,
	}
}

// =============================================================================
// Synthèse — Heatmap + Top Weeks
// =============================================================================

// ComputeSynthesisHeatmap convertit les lignes DuckDB en cellules de heatmap.
// Calcule le win rate (%) pour chaque combinaison carte × mode.
func ComputeSynthesisHeatmap(rows []domain.SynthesisHeatmapRow) []domain.HeatmapCell {
	cells := make([]domain.HeatmapCell, 0, len(rows))
	for _, r := range rows {
		var value float64
		if r.MatchCount > 0 {
			value = math.Round(float64(r.Wins)/float64(r.MatchCount)*1000) / 10
		}
		cells = append(cells, domain.HeatmapCell{
			RowKey: r.MapName,
			ColKey: r.ModeName,
			Value:  value,
			Count:  r.MatchCount,
		})
	}
	return cells
}

// ComputeTopWeeks calcule les 5 meilleures semaines depuis les lignes squad.
// Une semaine valide a ≥ 3 matchs. Tri par win_rate décroissant.
func ComputeTopWeeks(rows []domain.SquadMatchRow) []domain.TopWeekEntry {
	if len(rows) == 0 {
		return nil
	}

	type weekAgg struct {
		label     string
		wins      int
		total     int
		sumKills  float64
		sumDeaths float64
		sumKDA    float64
		nKDA      int
		count     int
	}
	byWeek := make(map[time.Time]*weekAgg)

	for _, r := range rows {
		wd := int(r.StartTime.Weekday())
		if wd == 0 {
			wd = 7
		}
		weekStart := r.StartTime.AddDate(0, 0, -(wd - 1)).Truncate(24 * time.Hour)
		agg, ok := byWeek[weekStart]
		if !ok {
			agg = &weekAgg{label: weekStart.Format("02/01")}
			byWeek[weekStart] = agg
		}
		if r.Outcome == domain.OutcomeWin || r.Outcome == domain.OutcomeLoss {
			agg.total++
			if r.Outcome == domain.OutcomeWin {
				agg.wins++
			}
		}
		agg.sumKills += float64(r.Kills)
		agg.sumDeaths += float64(r.Deaths)
		if r.KDA != nil {
			agg.sumKDA += *r.KDA
			agg.nKDA++
		}
		agg.count++
	}

	// Filtrer semaines ≥ 3 matchs.
	type weekScore struct {
		entry domain.TopWeekEntry
		wr    float64
	}
	var candidates []weekScore //nolint:prealloc
	for _, agg := range byWeek {
		if agg.count < 3 {
			continue
		}
		var wr float64
		if agg.total > 0 {
			wr = math.Round(float64(agg.wins)/float64(agg.total)*1000) / 10
		}
		var avgKills, avgDeaths, avgKDA float64
		if agg.count > 0 {
			avgKills = math.Round(agg.sumKills/float64(agg.count)*10) / 10
			avgDeaths = math.Round(agg.sumDeaths/float64(agg.count)*10) / 10
		}
		if agg.nKDA > 0 {
			avgKDA = math.Round(agg.sumKDA/float64(agg.nKDA)*100) / 100
		}
		candidates = append(candidates, weekScore{
			wr: wr,
			entry: domain.TopWeekEntry{
				WeekLabel:  agg.label,
				WinRate:    wr,
				AvgKills:   avgKills,
				AvgDeaths:  avgDeaths,
				AvgKDA:     avgKDA,
				MatchCount: agg.count,
			},
		})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].wr > candidates[j].wr })

	const maxTopWeeks = 5
	result := make([]domain.TopWeekEntry, 0, min(len(candidates), maxTopWeeks))
	for i, c := range candidates {
		if i >= maxTopWeeks {
			break
		}
		result = append(result, c.entry)
	}
	return result
}

// ComputeSynthesisTopWeeks calcule les 5 meilleures semaines depuis les lignes SynthesisMatchRow.
// Même logique que ComputeTopWeeks mais depuis un dataset allégé (LoadSynthesisMatches).
func ComputeSynthesisTopWeeks(rows []domain.SynthesisMatchRow) []domain.TopWeekEntry {
	if len(rows) == 0 {
		return nil
	}

	type weekAgg struct {
		label     string
		wins      int
		total     int
		sumKills  float64
		sumDeaths float64
		sumKDA    float64
		nKDA      int
		count     int
	}
	byWeek := make(map[time.Time]*weekAgg)

	for _, r := range rows {
		wd := int(r.StartTime.Weekday())
		if wd == 0 {
			wd = 7
		}
		weekStart := r.StartTime.AddDate(0, 0, -(wd - 1)).Truncate(24 * time.Hour)
		agg, ok := byWeek[weekStart]
		if !ok {
			agg = &weekAgg{label: weekStart.Format("02/01")}
			byWeek[weekStart] = agg
		}
		if r.Outcome == domain.OutcomeWin || r.Outcome == domain.OutcomeLoss {
			agg.total++
			if r.Outcome == domain.OutcomeWin {
				agg.wins++
			}
		}
		agg.sumKills += float64(r.Kills)
		agg.sumDeaths += float64(r.Deaths)
		if r.KDA != nil {
			agg.sumKDA += *r.KDA
			agg.nKDA++
		}
		agg.count++
	}

	type weekScore struct {
		entry domain.TopWeekEntry
		wr    float64
	}
	var candidates []weekScore //nolint:prealloc
	for _, agg := range byWeek {
		if agg.count < 3 {
			continue
		}
		var wr, avgKills, avgDeaths, avgKDA float64
		if agg.total > 0 {
			wr = math.Round(float64(agg.wins)/float64(agg.total)*1000) / 10
		}
		if agg.count > 0 {
			avgKills = math.Round(agg.sumKills/float64(agg.count)*10) / 10
			avgDeaths = math.Round(agg.sumDeaths/float64(agg.count)*10) / 10
		}
		if agg.nKDA > 0 {
			avgKDA = math.Round(agg.sumKDA/float64(agg.nKDA)*100) / 100
		}
		candidates = append(candidates, weekScore{
			wr: wr,
			entry: domain.TopWeekEntry{
				WeekLabel:  agg.label,
				WinRate:    wr,
				AvgKills:   avgKills,
				AvgDeaths:  avgDeaths,
				AvgKDA:     avgKDA,
				MatchCount: agg.count,
			},
		})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].wr > candidates[j].wr })

	const maxTopWeeks = 5
	result := make([]domain.TopWeekEntry, 0, min(len(candidates), maxTopWeeks))
	for i, c := range candidates {
		if i >= maxTopWeeks {
			break
		}
		result = append(result, c.entry)
	}
	return result
}

// ComputeSynthesisBreakdown calcule les stats de breakdown solo ou squad.
// isSquad=true → matchs avec amis (is_with_friends=true), false → matchs solo.
func ComputeSynthesisBreakdown(rows []domain.SynthesisMatchRow, isSquad bool) domain.SquadBreakdownStats {
	var matchCount, wins, total int
	var sumKills, sumKDA float64
	var nKDA int
	for _, r := range rows {
		if r.IsWithFriends != isSquad {
			continue
		}
		matchCount++
		if r.Outcome == domain.OutcomeWin || r.Outcome == domain.OutcomeLoss {
			total++
			if r.Outcome == domain.OutcomeWin {
				wins++
			}
		}
		sumKills += float64(r.Kills)
		if r.KDA != nil {
			sumKDA += *r.KDA
			nKDA++
		}
	}
	if matchCount == 0 {
		return domain.SquadBreakdownStats{}
	}
	var winRate, avgKDA, avgKills float64
	if total > 0 {
		winRate = math.Round(float64(wins)/float64(total)*1000) / 10
	}
	avgKills = math.Round(sumKills/float64(matchCount)*10) / 10
	if nKDA > 0 {
		avgKDA = math.Round(sumKDA/float64(nKDA)*100) / 100
	}
	return domain.SquadBreakdownStats{
		MatchCount: matchCount,
		WinRate:    winRate,
		AvgKDA:     avgKDA,
		AvgKills:   avgKills,
	}
}

// =============================================================================
// Sprint 43 — Bipolaire enrichie
// =============================================================================

// ComputeSynthesisKPIs calcule les KPIs détaillés pour un sous-ensemble solo ou squad.
func ComputeSynthesisKPIs(rows []domain.SynthesisMatchRow, isSquad bool) domain.SynthesisKPIs {
	var kpis domain.SynthesisKPIs
	var totalWL, wins int
	var sumKDA, sumAcc, sumPerf float64
	var nKDA, nAcc, nPerf int
	var sumKills, sumTimePlayed float64
	var nTime int

	for _, r := range rows {
		if r.IsWithFriends != isSquad {
			continue
		}
		kpis.MatchCount++
		if r.Outcome == domain.OutcomeWin || r.Outcome == domain.OutcomeLoss {
			totalWL++
			if r.Outcome == domain.OutcomeWin {
				wins++
			}
		}
		if r.KDA != nil {
			sumKDA += *r.KDA
			nKDA++
		}
		if r.Accuracy != nil {
			sumAcc += *r.Accuracy
			nAcc++
		}
		if r.PerformanceScore != nil {
			sumPerf += *r.PerformanceScore
			nPerf++
		}
		sumKills += float64(r.Kills)
		if r.TimePlayedSecs != nil && *r.TimePlayedSecs > 0 {
			sumTimePlayed += float64(*r.TimePlayedSecs)
			nTime++
		}
	}

	if kpis.MatchCount == 0 {
		return kpis
	}
	kpis.Wins = wins
	if totalWL > 0 {
		kpis.WinRate = math.Round(float64(wins)/float64(totalWL)*1000) / 1000
	}
	if nKDA > 0 {
		v := math.Round(sumKDA/float64(nKDA)*100) / 100
		kpis.KDRatio = &v
	}
	if nAcc > 0 {
		v := math.Round(sumAcc/float64(nAcc)*1000) / 1000
		kpis.Accuracy = &v
	}
	if nPerf > 0 {
		v := math.Round(sumPerf / float64(nPerf))
		kpis.PerformanceScore = &v
	}
	if nTime > 0 {
		avgLife := math.Round(sumTimePlayed/float64(nTime)*10) / 10
		kpis.AvgLifeSeconds = &avgLife
		totalMinutes := sumTimePlayed / 60.0
		if totalMinutes > 0 {
			kpm := math.Round(sumKills/totalMinutes*100) / 100
			kpis.KillsPerMin = &kpm
		}
	}
	return kpis
}

// ComputeComparisonMetrics construit les métriques bipolaires solo/escouade.
func ComputeComparisonMetrics(solo, squad domain.SynthesisKPIs) []domain.ComparisonMetricItem {
	items := make([]domain.ComparisonMetricItem, 0, 5)

	items = append(items, domain.ComparisonMetricItem{
		Label: "Win Rate", SoloValue: solo.WinRate, SquadValue: squad.WinRate,
		SoloText: fmtPct(solo.WinRate), SquadText: fmtPct(squad.WinRate),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "K/D", SoloValue: deref(solo.KDRatio), SquadValue: deref(squad.KDRatio),
		SoloText: fmtFloat2(solo.KDRatio), SquadText: fmtFloat2(squad.KDRatio),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "Précision", SoloValue: deref(solo.Accuracy), SquadValue: deref(squad.Accuracy),
		SoloText: fmtPct(deref(solo.Accuracy)), SquadText: fmtPct(deref(squad.Accuracy)),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "Kills/min", SoloValue: deref(solo.KillsPerMin), SquadValue: deref(squad.KillsPerMin),
		SoloText: fmtFloat2(solo.KillsPerMin), SquadText: fmtFloat2(squad.KillsPerMin),
	})
	items = append(items, domain.ComparisonMetricItem{
		Label: "Perf. Score", SoloValue: deref(solo.PerformanceScore), SquadValue: deref(squad.PerformanceScore),
		SoloText: fmtFloat0(solo.PerformanceScore), SquadText: fmtFloat0(squad.PerformanceScore),
	})
	return items
}

// ComputeTemporalHeatmap construit la heatmap jour × heure depuis les matchs.
func ComputeTemporalHeatmap(rows []domain.SynthesisMatchRow) []domain.TemporalHeatmapCell {
	counts := [7][24]int{}
	for _, r := range rows {
		// Go Weekday: Sunday=0, Monday=1 … Saturday=6
		// Frontend: lundi=0 … dimanche=6
		goDow := int(r.StartTime.Weekday())
		dow := (goDow + 6) % 7 // convertir: lundi=0
		hour := r.StartTime.Hour()
		counts[dow][hour]++
	}
	var cells []domain.TemporalHeatmapCell
	for d := 0; d < 7; d++ {
		for h := 0; h < 24; h++ {
			if counts[d][h] > 0 {
				cells = append(cells, domain.TemporalHeatmapCell{DOW: d, Hour: h, Count: counts[d][h]})
			}
		}
	}
	return cells
}

func deref(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func fmtPct(v float64) string {
	return fmt.Sprintf("%.1f%%", v*100)
}

func fmtFloat2(p *float64) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%.2f", *p)
}

func fmtFloat0(p *float64) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%.0f", *p)
}
