// Package analysis â€” squad_breakdown.go : breakdown solo/escouade + synthÃ¨se + top weeks.
package analysis

import (
	"fmt"
	"math"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
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
type extKPIAcc struct {
	sumDeaths, sumAssists, sumHeadshots   float64
	sumMaxSpree, sumDmgDealt, sumDmgTaken float64
	sumPerfectKills                       float64
	nSpree, nDmgDealt, nDmgTaken          int
}

func (a *extKPIAcc) add(r canonical.PlayerMatchRow) {
	if r.Self.Deaths != nil {
		a.sumDeaths += float64(*r.Self.Deaths)
	}
	if r.Self.Assists != nil {
		a.sumAssists += float64(*r.Self.Assists)
	}
	if r.Self.HeadshotKills != nil {
		a.sumHeadshots += float64(*r.Self.HeadshotKills)
	}
	if r.Self.PerfectKills != nil {
		a.sumPerfectKills += float64(*r.Self.PerfectKills)
	}
	if r.Self.MaxKillingSpree != nil {
		a.sumMaxSpree += float64(*r.Self.MaxKillingSpree)
		a.nSpree++
	}
	if r.Self.DamageDealt != nil {
		a.sumDmgDealt += float64(*r.Self.DamageDealt)
		a.nDmgDealt++
	}
	if r.Self.DamageTaken != nil {
		a.sumDmgTaken += float64(*r.Self.DamageTaken)
		a.nDmgTaken++
	}
}

func (a *extKPIAcc) applyTo(kpis *domain.SynthesisKPIs, nMatches int, sumTimeSecs float64) {
	hpm := math.Round(a.sumHeadshots/float64(nMatches)*100) / 100
	kpis.HeadshotsPerMatch = &hpm
	pkpm := math.Round(a.sumPerfectKills/float64(nMatches)*100) / 100
	kpis.PerfectKillsPerMatch = &pkpm
	if tot := sumTimeSecs / 60.0; tot > 0 {
		dpm := math.Round(a.sumDeaths/tot*100) / 100
		kpis.DeathsPerMin = &dpm
		apm := math.Round(a.sumAssists/tot*100) / 100
		kpis.AssistsPerMin = &apm
	}
	if a.nSpree > 0 {
		v := math.Round(a.sumMaxSpree/float64(a.nSpree)*100) / 100
		kpis.AvgMaxKillingSpree = &v
	}
	if a.nDmgDealt > 0 {
		v := math.Round(a.sumDmgDealt / float64(a.nDmgDealt))
		kpis.AvgDamageDealt = &v
	}
	if a.nDmgTaken > 0 {
		v := math.Round(a.sumDmgTaken / float64(a.nDmgTaken))
		kpis.AvgDamageTaken = &v
	}
}

func ComputeSynthesisKPIsFromCanonical(rows []canonical.PlayerMatchRow, isSquad bool) domain.SynthesisKPIs {
	var kpis domain.SynthesisKPIs
	var totalWL, wins, nKDA, nAcc, nPerf, nTime int
	var sumKDA, sumAcc, sumPerf, sumKills, sumTimePlayed float64
	var ext extKPIAcc

	for _, r := range rows {
		if r.Enrichment.IsWithFriends != isSquad {
			continue
		}
		kpis.MatchCount++
		if r.Self.Outcome == canonical.OutcomeWin || r.Self.Outcome == canonical.OutcomeLoss {
			totalWL++
			if r.Self.Outcome == canonical.OutcomeWin {
				wins++
			}
		}
		if r.Self.KDA != nil {
			sumKDA += *r.Self.KDA
			nKDA++
		}
		if r.Self.Accuracy != nil {
			sumAcc += *r.Self.Accuracy
			nAcc++
		}
		if r.Enrichment.PerformanceScore != nil {
			sumPerf += *r.Enrichment.PerformanceScore
			nPerf++
		}
		if r.Self.Kills != nil {
			sumKills += float64(*r.Self.Kills)
		}
		if r.Self.TimePlayed != nil && *r.Self.TimePlayed > 0 {
			sumTimePlayed += float64(*r.Self.TimePlayed)
			nTime++
		}
		ext.add(r)
	}

	if kpis.MatchCount == 0 {
		return kpis
	}
	kpis.Wins = wins
	if totalWL > 0 {
		// Note: WinRate canonique 0..1 (ADR 0006). math.Round/1000 prÃ©serve la
		// mÃªme prÃ©cision que ComputeSynthesisKPIs (3 dÃ©cimales).
		kpis.WinRate = math.Round(WinRate(wins, totalWL)*1000) / 1000
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
		if tot := sumTimePlayed / 60.0; tot > 0 {
			kpm := math.Round(sumKills/tot*100) / 100
			kpis.KillsPerMin = &kpm
		}
	}
	kpis.TotalTimePlayedSeconds = int(sumTimePlayed)
	ext.applyTo(&kpis, kpis.MatchCount, sumTimePlayed)
	return kpis
}

// =============================================================================
// P4.3a (ADR 0011) : variantes canonical des analyses synthesis
// =============================================================================

// ComputeSynthesisTopWeeksFromCanonical est la variante canonical-aware de
// ComputeSynthesisTopWeeks. Logique strictement identique ; seule la source
// des champs change (Self.Outcome/Kills/Deaths/KDA + Summary.StartedAtUTC).
func ComputeSynthesisTopWeeksFromCanonical(rows []canonical.PlayerMatchRow) []domain.TopWeekEntry {
	if len(rows) == 0 {
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
		st := r.Summary.StartedAtUTC
		wd := int(st.Weekday())
		if wd == 0 {
			wd = 7
		}
		weekStart := st.AddDate(0, 0, -(wd - 1)).Truncate(24 * time.Hour)
		agg, ok := byWeek[weekStart]
		if !ok {
			agg = &weekAgg{label: weekStart.Format("02/01")}
			byWeek[weekStart] = agg
		}
		// Classement individuel dans le match : rank=1 = premier.
		if r.Self.RankInMatch != nil {
			agg.total++
			if *r.Self.RankInMatch == 1 {
				agg.wins++
			}
		}
		if r.Self.Kills != nil {
			agg.sumKills += float64(*r.Self.Kills)
		}
		if r.Self.Deaths != nil {
			agg.sumDeaths += float64(*r.Self.Deaths)
		}
		if r.Self.KDA != nil {
			agg.sumKDA += *r.Self.KDA
			agg.nKDA++
		}
		agg.count++
	}

	type weekScore struct {
		entry     domain.TopWeekEntry
		weekStart time.Time
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
			weekStart: ws,
			entry: domain.TopWeekEntry{
				WeekLabel:  agg.label,
				WeekStart:  ws.Format("2006-01-02"),
				WinRate:    wr,
				Wins:       agg.wins,
				AvgKills:   avgKills,
				AvgDeaths:  avgDeaths,
				AvgKDA:     avgKDA,
				MatchCount: agg.count,
			},
		})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].weekStart.Before(candidates[j].weekStart) })

	result := make([]domain.TopWeekEntry, 0, len(candidates))
	for _, c := range candidates {
		result = append(result, c.entry)
	}
	return result
}

// ComputeSynthesisBreakdownFromCanonical est la variante canonical-aware de
// ComputeSynthesisBreakdown. Filtre par IsWithFriends comme la version legacy.
func ComputeSynthesisBreakdownFromCanonical(rows []canonical.PlayerMatchRow, isSquad bool) domain.SquadBreakdownStats {
	var matchCount, wins, total int
	var sumKills, sumKDA float64
	var nKDA int
	for _, r := range rows {
		if r.Enrichment.IsWithFriends != isSquad {
			continue
		}
		matchCount++
		if r.Self.Outcome == canonical.OutcomeWin || r.Self.Outcome == canonical.OutcomeLoss {
			total++
			if r.Self.Outcome == canonical.OutcomeWin {
				wins++
			}
		}
		if r.Self.Kills != nil {
			sumKills += float64(*r.Self.Kills)
		}
		if r.Self.KDA != nil {
			sumKDA += *r.Self.KDA
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

// ComputeTemporalHeatmapFromCanonical est la variante canonical-aware de
// ComputeTemporalHeatmap. Pas de logique mÃ©tier â€” seulement l'extraction
// du jour/heure depuis Summary.StartedAtUTC.
func ComputeTemporalHeatmapFromCanonical(rows []canonical.PlayerMatchRow) []domain.TemporalHeatmapCell {
	type heatmapAgg struct {
		count int
		wins  int
	}
	cells := [7][24]heatmapAgg{}
	for _, r := range rows {
		st := r.Summary.StartedAtUTC
		goDow := int(st.Weekday())
		dow := (goDow + 6) % 7 // convertir: lundi=0
		hour := st.Hour()
		agg := cells[dow][hour]
		agg.count++
		if r.Self.Outcome == canonical.OutcomeWin {
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

// ComputeComparisonMetrics construit les mÃ©triques bipolaires solo/escouade.
func ComputeComparisonMetrics(solo, squad domain.SynthesisKPIs) []domain.ComparisonMetricItem {
	items := make([]domain.ComparisonMetricItem, 0, 15)
	items = append(items, domain.ComparisonMetricItem{
		Label: "match_count", SoloValue: float64(solo.MatchCount), SquadValue: float64(squad.MatchCount),
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
		Label: "avg_damage_taken", SoloValue: deref(solo.AvgDamageTaken), SquadValue: deref(squad.AvgDamageTaken),
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
		if r.Outcome == 2 { // legacy domain: outcome=2 is WIN
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
		if r.Player1Outcome == 2 { // OutcomeWin (legacy code)
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

func fmtPct(v float64) string {
	return fmt.Sprintf("%.1f%%", v*100)
}
