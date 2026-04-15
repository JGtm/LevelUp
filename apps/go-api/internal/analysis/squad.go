// Package analysis — squad.go : algorithmes purs pour la page Escouade et Synthèse.
//
// Miroir Go de :
//   src/analysis/_performance_squad.py   → ComputeSquadPerformanceScore, ComputeSquadTimeseries
//   src/analysis/squad_records.py        → ComputeParticipationProfile, ComputeSquadRecords
//   src/data/services/teammates_service.py (impact) → ComputeImpactSummary
//
// Règle architecture : 0 accès DB, 0 import Streamlit — entrée domain.*, sortie domain.*.
package analysis

import (
	"fmt"
	"math"
	"sort"
	"time"

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
// Profil de participation radar
// =============================================================================

// ComputeParticipationProfile calcule le profil radar à 6 axes d'un joueur
// depuis ses matchs communs en escouade.
//
// Axes : kills, deaths, assists, accuracy, kills_per_min, kda.
// Chaque valeur est la moyenne sur les matchs non-nuls.
// Miroir de TeammateStats.ParticipationProfile dans teammates_service.py.
func ComputeParticipationProfile(rows []domain.SquadMatchRow, name, color string) domain.ParticipationProfile {
	if len(rows) == 0 {
		return domain.ParticipationProfile{Name: name, Color: color, Values: map[string]float64{}}
	}

	var (
		sumKills, sumDeaths, sumAssists, sumAccuracy, sumKPM, sumKDA float64
		nKills, nDeaths, nAssists, nAccuracy, nKPM, nKDA             int
	)
	for _, r := range rows {
		sumKills += float64(r.Kills)
		nKills++
		sumDeaths += float64(r.Deaths)
		nDeaths++
		sumAssists += float64(r.Assists)
		nAssists++
		if r.Accuracy != nil {
			sumAccuracy += *r.Accuracy
			nAccuracy++
		}
		if r.TimePlayedSecs > 0 {
			kpm := float64(r.Kills) * 60.0 / float64(r.TimePlayedSecs)
			sumKPM += kpm
			nKPM++
		}
		if r.KDA != nil {
			sumKDA += *r.KDA
			nKDA++
		}
	}

	avg := func(s float64, n int) float64 {
		if n == 0 {
			return 0
		}
		return math.Round((s/float64(n))*100) / 100
	}

	return domain.ParticipationProfile{
		Name:  name,
		Color: color,
		Values: map[string]float64{
			"kills":        avg(sumKills, nKills),
			"deaths":       avg(sumDeaths, nDeaths),
			"assists":      avg(sumAssists, nAssists),
			"accuracy":     avg(sumAccuracy, nAccuracy),
			"kills_per_min": avg(sumKPM, nKPM),
			"kda":          avg(sumKDA, nKDA),
		},
	}
}

// ComputeTeammateProfile calcule le profil radar depuis les lignes TeammateMatchRow.
func ComputeTeammateProfile(rows []domain.TeammateMatchRow, name, color string) domain.ParticipationProfile {
	if len(rows) == 0 {
		return domain.ParticipationProfile{Name: name, Color: color, Values: map[string]float64{}}
	}

	var (
		sumKills, sumDeaths, sumAssists, sumAccuracy, sumKPM, sumKDA float64
		nKills, nDeaths, nAssists, nAccuracy, nKPM, nKDA             int
	)
	for _, r := range rows {
		sumKills += float64(r.Kills)
		nKills++
		sumDeaths += float64(r.Deaths)
		nDeaths++
		sumAssists += float64(r.Assists)
		nAssists++
		if r.Accuracy != nil {
			sumAccuracy += *r.Accuracy
			nAccuracy++
		}
		if r.TimePlayedSecs > 0 {
			kpm := float64(r.Kills) * 60.0 / float64(r.TimePlayedSecs)
			sumKPM += kpm
			nKPM++
		}
		if r.Ratio != nil {
			sumKDA += *r.Ratio
			nKDA++
		}
	}

	avg := func(s float64, n int) float64 {
		if n == 0 {
			return 0
		}
		return math.Round((s/float64(n))*100) / 100
	}

	return domain.ParticipationProfile{
		Name:  name,
		Color: color,
		Values: map[string]float64{
			"kills":         avg(sumKills, nKills),
			"deaths":        avg(sumDeaths, nDeaths),
			"assists":       avg(sumAssists, nAssists),
			"accuracy":      avg(sumAccuracy, nAccuracy),
			"kills_per_min": avg(sumKPM, nKPM),
			"kda":           avg(sumKDA, nKDA),
		},
	}
}

// =============================================================================
// Records escouade — miroir de squad_records.py
// =============================================================================

// squadMetrics définit les métriques de record et leur sens (false=max, true=min).
var squadMetrics = []struct {
	name    string
	fromRow func(domain.SquadMatchRow) *float64
	isMin   bool
}{
	{"kills", func(r domain.SquadMatchRow) *float64 { v := float64(r.Kills); return &v }, false},
	{"deaths", func(r domain.SquadMatchRow) *float64 { v := float64(r.Deaths); return &v }, true},
	{"assists", func(r domain.SquadMatchRow) *float64 { v := float64(r.Assists); return &v }, false},
	{"kda", func(r domain.SquadMatchRow) *float64 { return r.KDA }, false},
	{"accuracy", func(r domain.SquadMatchRow) *float64 { return r.Accuracy }, false},
}

// ComputeSquadRecords calcule les records individuels pour un joueur
// sur un ensemble de matchs communs.
// Retourne un map metric → valeur record (nil si aucune donnée).
func ComputeSquadRecords(rows []domain.SquadMatchRow) map[string]*float64 {
	result := make(map[string]*float64, len(squadMetrics))
	for _, m := range squadMetrics {
		result[m.name] = nil
	}
	if len(rows) == 0 {
		return result
	}
	for _, m := range squadMetrics {
		for _, row := range rows {
			v := m.fromRow(row)
			if v == nil {
				continue
			}
			cur := result[m.name]
			if cur == nil {
				cp := *v
				result[m.name] = &cp
				continue
			}
			if m.isMin && *v < *cur {
				cp := *v
				result[m.name] = &cp
			} else if !m.isMin && *v > *cur {
				cp := *v
				result[m.name] = &cp
			}
		}
	}
	return result
}

// ComputeTeammateRecords calcule les records d'un coéquipier (TeammateMatchRow).
func ComputeTeammateRecords(rows []domain.TeammateMatchRow) map[string]*float64 {
	type metricDef struct {
		name    string
		fromRow func(domain.TeammateMatchRow) *float64
		isMin   bool
	}
	metrics := []metricDef{
		{"kills", func(r domain.TeammateMatchRow) *float64 { v := float64(r.Kills); return &v }, false},
		{"deaths", func(r domain.TeammateMatchRow) *float64 { v := float64(r.Deaths); return &v }, true},
		{"assists", func(r domain.TeammateMatchRow) *float64 { v := float64(r.Assists); return &v }, false},
		{"kda", func(r domain.TeammateMatchRow) *float64 { return r.Ratio }, false},
		{"accuracy", func(r domain.TeammateMatchRow) *float64 { return r.Accuracy }, false},
	}
	result := make(map[string]*float64, len(metrics))
	for _, m := range metrics {
		result[m.name] = nil
	}
	for _, m := range metrics {
		for _, row := range rows {
			v := m.fromRow(row)
			if v == nil {
				continue
			}
			cur := result[m.name]
			if cur == nil {
				cp := *v
				result[m.name] = &cp
				continue
			}
			if m.isMin && *v < *cur {
				cp := *v
				result[m.name] = &cp
			} else if !m.isMin && *v > *cur {
				cp := *v
				result[m.name] = &cp
			}
		}
	}
	return result
}

// =============================================================================
// Analyse d'impact — miroir de friends_impact.py (simplifié)
// =============================================================================

// ComputeImpactSummary analyse les événements highlight pour identifier
// first bloods, clutch kills, last kills, first deaths d'un joueur et son coéquipier.
//
// myXUID et friendXUID sont utilisés pour distinguer « me » de « teammate ».
func ComputeImpactSummary(
	events []domain.ImpactEventRow,
	myXUID, friendXUID string,
) domain.SquadImpact {
	if len(events) == 0 {
		return domain.SquadImpact{Available: false}
	}

	// Grouper par match_id.
	type matchEvents struct {
		kills  []domain.ImpactEventRow
		deaths []domain.ImpactEventRow
	}
	byMatch := make(map[string]*matchEvents)
	for _, e := range events {
		if byMatch[e.MatchID] == nil {
			byMatch[e.MatchID] = &matchEvents{}
		}
		switch e.EventType {
		case "kill":
			byMatch[e.MatchID].kills = append(byMatch[e.MatchID].kills, e)
		case "death":
			byMatch[e.MatchID].deaths = append(byMatch[e.MatchID].deaths, e)
		}
	}

	var firstBloodsMe, firstBloodsTm, clutchMe, clutchTm, lastKillsMe, lastKillsTm, firstDeathsMe, firstDeathsTm int

	for _, me := range byMatch {
		// First blood (premier kill du match) :
		kills := me.kills
		sort.Slice(kills, func(i, j int) bool { return kills[i].TimeMS < kills[j].TimeMS })
		if len(kills) > 0 {
			switch kills[0].XUID {
			case myXUID:
				firstBloodsMe++
			case friendXUID:
				firstBloodsTm++
			}
		}
		// Last kill (dernier kill du match) :
		if len(kills) > 0 {
			last := kills[len(kills)-1]
			switch last.XUID {
			case myXUID:
				lastKillsMe++
			case friendXUID:
				lastKillsTm++
			}
		}
		// Clutch : kill dans les 30s finales du match (approximation via position dans la liste).
		// On prend les kills dans le dernier tiers de la liste.
		if len(kills) >= 3 {
			cutoff := kills[len(kills)*2/3].TimeMS
			for _, k := range kills {
				if k.TimeMS >= cutoff {
					switch k.XUID {
					case myXUID:
						clutchMe++
					case friendXUID:
						clutchTm++
					}
				}
			}
		}
		// First death (première mort du match) :
		deaths := me.deaths
		sort.Slice(deaths, func(i, j int) bool { return deaths[i].TimeMS < deaths[j].TimeMS })
		if len(deaths) > 0 {
			switch deaths[0].XUID {
			case myXUID:
				firstDeathsMe++
			case friendXUID:
				firstDeathsTm++
			}
		}
	}

	available := firstBloodsMe+firstBloodsTm+clutchMe+clutchTm+lastKillsMe+lastKillsTm > 0

	return domain.SquadImpact{
		FirstBloods: domain.ImpactEventSummary{Me: firstBloodsMe, Teammate: firstBloodsTm},
		ClutchKills: domain.ImpactEventSummary{Me: clutchMe, Teammate: clutchTm},
		LastKills:   domain.ImpactEventSummary{Me: lastKillsMe, Teammate: lastKillsTm},
		FirstDeaths: domain.ImpactEventSummary{Me: firstDeathsMe, Teammate: firstDeathsTm},
		Available:   available,
	}
}

// =============================================================================
// Série temporelle escouade — miroir de compute_squad_timeseries()
// =============================================================================

// ComputeSquadTimeseries calcule la série temporelle des performances escouade.
//
// Groupe par session_id si disponible, sinon par période temporelle (semaine → mois).
// maxBuckets limite le nombre de points retournés.
// Miroir de Python compute_squad_timeseries() dans _performance_squad.py.
func ComputeSquadTimeseries(rows []domain.SquadMatchRow, maxBuckets int) []domain.SquadTimeseriesPoint {
	if len(rows) == 0 {
		return nil
	}

	// Essayer le groupement par session_id.
	bySession := groupSquadBySession(rows)
	if len(bySession) > 0 {
		return bySession
	}

	// Fallback : groupement temporel.
	return groupSquadByTime(rows, maxBuckets)
}

// groupSquadBySession regroupe les matchs par session_id.
func groupSquadBySession(rows []domain.SquadMatchRow) []domain.SquadTimeseriesPoint {
	// Vérifier que des sessions existent.
	hasSession := false
	for _, r := range rows {
		if r.SessionID != nil {
			hasSession = true
			break
		}
	}
	if !hasSession {
		return nil
	}

	type sessionAgg struct {
		label      string
		sortKey    int
		totalPerf  float64
		totalMMR   float64
		wins, losses int
		count      int
		countPerf  int
	}
	bySession := make(map[int]*sessionAgg)

	for _, r := range rows {
		if r.SessionID == nil {
			continue
		}
		sid := *r.SessionID
		agg, ok := bySession[sid]
		if !ok {
			label := fmt.Sprintf("%d", sid)
			if r.SessionLabel != nil && len(*r.SessionLabel) >= 5 {
				label = (*r.SessionLabel)[:5]
			}
			agg = &sessionAgg{label: label, sortKey: sid}
			bySession[sid] = agg
		}
		if r.PerformanceScore != nil {
			agg.totalPerf += *r.PerformanceScore
			agg.countPerf++
		}
		agg.totalMMR += r.TeamMMR
		agg.count++
		if r.Outcome == 2 {
			agg.wins++
		} else if r.Outcome == 3 {
			agg.losses++
		}
	}

	// Trier par sortKey.
	ids := make([]int, 0, len(bySession))
	for id := range bySession {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	points := make([]domain.SquadTimeseriesPoint, 0, len(ids))
	for _, id := range ids {
		agg := bySession[id]
		var perfPtr *float64
		if agg.countPerf > 0 {
			v := math.Round((agg.totalPerf/float64(agg.countPerf))*10) / 10
			perfPtr = &v
		}
		mmrAvg := agg.totalMMR / float64(agg.count)
		p := domain.SquadTimeseriesPoint{
			BucketLabel: agg.label,
			SquadPerf:   perfPtr,
			TeamMMRAvg:  &mmrAvg,
			MatchCount:  agg.count,
		}
		total := agg.wins + agg.losses
		if total > 0 {
			wr := math.Round(float64(agg.wins)/float64(total)*1000) / 10
			p.WinRate = &wr
		}
		points = append(points, p)
	}
	return points
}

// groupSquadByTime regroupe les matchs par période temporelle (semaine → mois).
func groupSquadByTime(rows []domain.SquadMatchRow, maxBuckets int) []domain.SquadTimeseriesPoint {
	// Tenter week, puis month.
	for _, trunc := range []string{"week", "month"} {
		points := bucketByTime(rows, trunc)
		if len(points) <= maxBuckets {
			return points
		}
	}
	return bucketByTime(rows, "month")
}

// bucketByTime groupe les rows par période (week ou month).
func bucketByTime(rows []domain.SquadMatchRow, period string) []domain.SquadTimeseriesPoint {
	type bucketAgg struct {
		bucketTime time.Time
		label      string
		totalPerf  float64
		totalMMR   float64
		wins, losses int
		count      int
		countPerf  int
	}
	byBucket := make(map[time.Time]*bucketAgg)

	for _, r := range rows {
		var bucket time.Time
		var label string
		if period == "week" {
			// Tronquer au lundi de la semaine.
			wd := int(r.StartTime.Weekday())
			if wd == 0 {
				wd = 7
			}
			bucket = r.StartTime.AddDate(0, 0, -(wd - 1)).Truncate(24 * time.Hour)
			label = bucket.Format("02/01")
		} else {
			bucket = time.Date(r.StartTime.Year(), r.StartTime.Month(), 1, 0, 0, 0, 0, time.UTC)
			label = bucket.Format("01/2006")
		}

		agg, ok := byBucket[bucket]
		if !ok {
			agg = &bucketAgg{bucketTime: bucket, label: label}
			byBucket[bucket] = agg
		}
		if r.PerformanceScore != nil {
			agg.totalPerf += *r.PerformanceScore
			agg.countPerf++
		}
		agg.totalMMR += r.TeamMMR
		agg.count++
		if r.Outcome == 2 {
			agg.wins++
		} else if r.Outcome == 3 {
			agg.losses++
		}
	}

	// Trier par date.
	buckets := make([]time.Time, 0, len(byBucket))
	for b := range byBucket {
		buckets = append(buckets, b)
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].Before(buckets[j]) })

	points := make([]domain.SquadTimeseriesPoint, 0, len(buckets))
	for _, b := range buckets {
		agg := byBucket[b]
		var perfPtr *float64
		if agg.countPerf > 0 {
			v := math.Round((agg.totalPerf/float64(agg.countPerf))*10) / 10
			perfPtr = &v
		}
		mmrAvg := agg.totalMMR / float64(agg.count)
		p := domain.SquadTimeseriesPoint{
			BucketLabel: agg.label,
			SquadPerf:   perfPtr,
			TeamMMRAvg:  &mmrAvg,
			MatchCount:  agg.count,
		}
		total := agg.wins + agg.losses
		if total > 0 {
			wr := math.Round(float64(agg.wins)/float64(total)*1000) / 10
			p.WinRate = &wr
		}
		points = append(points, p)
	}
	return points
}

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
		if r.Outcome == 2 || r.Outcome == 3 {
			totalWL++
			if r.Outcome == 2 {
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
		label    string
		wins     int
		total    int
		sumKills float64
		count    int
	}
	byWeek := make(map[time.Time]*weekAgg)

	for _, r := range rows {
		wd := int(r.StartTime.Weekday())
		if wd == 0 {
			wd = 7
		}
		weekStart := r.StartTime.AddDate(0, 0, -(wd-1)).Truncate(24 * time.Hour)
		agg, ok := byWeek[weekStart]
		if !ok {
			agg = &weekAgg{label: weekStart.Format("02/01")}
			byWeek[weekStart] = agg
		}
		if r.Outcome == 2 || r.Outcome == 3 {
			agg.total++
			if r.Outcome == 2 {
				agg.wins++
			}
		}
		agg.sumKills += float64(r.Kills)
		agg.count++
	}

	// Filtrer semaines ≥ 3 matchs.
	type weekScore struct {
		entry domain.TopWeekEntry
		wr    float64
	}
	var candidates []weekScore
	for _, agg := range byWeek {
		if agg.count < 3 {
			continue
		}
		var wr float64
		if agg.total > 0 {
			wr = math.Round(float64(agg.wins)/float64(agg.total)*1000) / 10
		}
		var avgKills float64
		if agg.count > 0 {
			avgKills = math.Round(agg.sumKills/float64(agg.count)*10) / 10
		}
		candidates = append(candidates, weekScore{
			wr: wr,
			entry: domain.TopWeekEntry{
				WeekLabel:  agg.label,
				WinRate:    wr,
				AvgKills:   avgKills,
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

// =============================================================================
// Helpers
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
