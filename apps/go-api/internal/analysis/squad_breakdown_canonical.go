package analysis

import (
	"math"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

type extKPIAcc struct {
	sumDeaths, sumAssists, sumHeadshots   float64
	sumKills                              float64
	sumMaxSpree, sumDmgDealt, sumDmgTaken float64
	sumPerfectKills                       float64
	nSpree, nDmgDealt, nDmgTaken          int
	nRanked                               int
}

func (a *extKPIAcc) add(r canonical.PlayerMatchRow) {
	if r.Self.Deaths != nil {
		a.sumDeaths += float64(*r.Self.Deaths)
	}
	if r.Self.Kills != nil {
		a.sumKills += float64(*r.Self.Kills)
	}
	if r.Self.Assists != nil {
		a.sumAssists += float64(*r.Self.Assists)
	}
	if r.Summary.IsRanked != nil && *r.Summary.IsRanked {
		a.nRanked++
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

func (a *extKPIAcc) applyTo(kpis *domain.SynthesisKPIs, nMatches int, sumTimeSecs float64, effectiveHpToKill float64) {
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
	// Rendement offensif agrégé : effectiveHpToKill × (kills + assists/3) / total_damage_dealt.
	if a.sumDmgDealt > 0 {
		oc := math.Round((effectiveHpToKill*(a.sumKills+a.sumAssists/3.0)/a.sumDmgDealt)*1000) / 1000
		kpis.AvgOffensiveConversion = &oc
	}
	// Résistance défensive agrégée : total_damage_taken / (effectiveHpToKill × deaths).
	if a.sumDeaths > 0 && a.sumDmgTaken > 0 {
		dr := math.Round((a.sumDmgTaken/(effectiveHpToKill*a.sumDeaths))*1000) / 1000
		kpis.AvgDefensiveResistance = &dr
	}
	kpis.RankedMatchCount = a.nRanked
}

func ComputeSynthesisKPIsFromCanonical(rows []canonical.PlayerMatchRow, isSquad bool, effectiveHpToKill float64) domain.SynthesisKPIs {
	var kpis domain.SynthesisKPIs
	var totalWL, wins, nKDA, nAcc, nPerf, nTime, nLife int
	var sumKDA, sumAcc, sumPerf, sumKills, sumTimePlayed, sumLife float64
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
		if r.Self.AvgLifeSeconds != nil {
			sumLife += *r.Self.AvgLifeSeconds
			nLife++
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
	// AvgLifeSeconds = moyenne de la valeur API par match (NON sumTimePlayed/n).
	if nLife > 0 {
		avgLife := math.Round(sumLife/float64(nLife)*10) / 10
		kpis.AvgLifeSeconds = &avgLife
	}
	if nTime > 0 {
		if tot := sumTimePlayed / 60.0; tot > 0 {
			kpm := math.Round(sumKills/tot*100) / 100
			kpis.KillsPerMin = &kpm
		}
	}
	kpis.TotalTimePlayedSeconds = int(sumTimePlayed)
	ext.applyTo(&kpis, kpis.MatchCount, sumTimePlayed, effectiveHpToKill)
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
