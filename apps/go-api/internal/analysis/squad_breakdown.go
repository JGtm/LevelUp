// Package analysis â€” squad_breakdown.go : breakdown solo/escouade + synthÃ¨se + top weeks.
package analysis

import (
	"math"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// =============================================================================
// Breakdown solo vs escouade
// =============================================================================

// ComputeSquadBreakdown calcule les stats agrÃ©gÃ©es pour un mode (solo ou escouade).
// rows doit dÃ©jÃ  Ãªtre filtrÃ© (is_with_friends=true ou false).
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
		// TODO P4 ADR 0006 : convertir vers WinRate canonique (0..1) + format front.
		// Conserve l'unitÃ© 0..100.0 historique avec arrondi 1 dÃ©cimale.
		wr = math.Round(WinRate(wins, totalWL)*1000) / 10
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
// SynthÃ¨se â€” Heatmap + Top Weeks
// =============================================================================

// ComputeSynthesisHeatmap convertit les lignes DuckDB en cellules de heatmap.
// Calcule le win rate (%) pour chaque combinaison carte Ã— mode.
func ComputeSynthesisHeatmap(rows []domain.SynthesisHeatmapRow) []domain.HeatmapCell {
	cells := make([]domain.HeatmapCell, 0, len(rows))
	for _, r := range rows {
		var value float64
		if r.MatchCount > 0 {
			value = math.Round(float64(r.Wins)/float64(r.MatchCount)*1000) / 10
		}
		cells = append(cells, domain.HeatmapCell{
			MapName:  r.MapName,
			ModeName: r.ModeName,
			Value:    value,
			Count:    r.MatchCount,
		})
	}
	return cells
}

// ComputeTopWeeks calcule les 5 meilleures semaines depuis les lignes squad.
// Une semaine valide a >= 3 matchs. Tri par win_rate decroissant.
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

	// Filtrer semaines â‰¥ 3 matchs.
	type weekScore struct {
		entry domain.TopWeekEntry
		wr    float64
	}
	var candidates []weekScore //nolint:prealloc
	for ws, agg := range byWeek {
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
				WeekStart:  ws.Format("2006-01-02"),
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
// Meme logique que ComputeTopWeeks mais depuis un dataset allege (LoadSynthesisMatches).
func ComputeSynthesisTopWeeks(rows []legacymatch.SynthesisMatchRow) []domain.TopWeekEntry {
	if len(rows) == 0 {
		// Init à [] plutôt que nil : un slice nil sérialise en JSON `null` et
		// crashe le front. Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
		return []domain.TopWeekEntry{}
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
	for ws, agg := range byWeek {
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
				WeekStart:  ws.Format("2006-01-02"),
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
// isSquad=true â†’ matchs avec amis (is_with_friends=true), false â†’ matchs solo.
func ComputeSynthesisBreakdown(rows []legacymatch.SynthesisMatchRow, isSquad bool) domain.SquadBreakdownStats {
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
// Sprint 43 â€” Bipolaire enrichie
// =============================================================================

// ComputeSynthesisKPIs calcule les KPIs dÃ©taillÃ©s pour un sous-ensemble solo ou squad.
func ComputeSynthesisKPIs(rows []legacymatch.SynthesisMatchRow, isSquad bool) domain.SynthesisKPIs {
	var kpis domain.SynthesisKPIs
	var totalWL, wins int
	var sumKDA, sumAcc, sumPerf float64
	var nKDA, nAcc, nPerf int
	var sumKills, sumTimePlayed, sumLife float64
	var nTime, nLife int

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
		// AvgLifeSeconds : moyenne des valeurs API par match (jamais dérivé de
		// time_played). Mirror exact de ComputeSynthesisKPIsFromCanonical.
		if r.AvgLifeSeconds != nil {
			sumLife += *r.AvgLifeSeconds
			nLife++
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
	// AvgLifeSeconds = moyenne de la valeur API par match (NON sumTimePlayed/n).
	// avg_life_seconds est désormais chargé dans SynthesisMatchRow (Q33b).
	// Identique à ComputeSynthesisKPIsFromCanonical (garde-fou parité).
	if nLife > 0 {
		avgLife := math.Round(sumLife/float64(nLife)*10) / 10
		kpis.AvgLifeSeconds = &avgLife
	}
	if nTime > 0 {
		totalMinutes := sumTimePlayed / 60.0
		if totalMinutes > 0 {
			kpm := math.Round(sumKills/totalMinutes*100) / 100
			kpis.KillsPerMin = &kpm
		}
	}
	return kpis
}

// ComputeSynthesisKPIsFromCanonical est la variante canonical-aware de
// ComputeSynthesisKPIs (P4 pilote synthesis, ADR 0011). Consomme directement
// `[]canonical.PlayerMatchRow` sans intermÃ©diaire `legacymatch.SynthesisMatchRow`.
//
// Comportement strictement Ã©quivalent Ã  ComputeSynthesisKPIs ; la seule
// diffÃ©rence est la source des champs (Self.Kills, Self.KDA, Self.Outcome
// au lieu de SynthesisMatchRow.{Kills,KDA,Outcome}).
//
// Migration progressive : ComputeSynthesisKPIs reste pour les autres callers
// (Squad, Teammates) qui n'ont pas encore migrÃ©. Sera supprimÃ© en P4.3 quand
// tous les callers seront sur canonical.
// extKPIAcc accumule les compteurs des KPIs etendus du bipolaire Solo/Escouade.
