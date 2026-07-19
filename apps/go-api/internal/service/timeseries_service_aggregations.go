// Package service — timeseries_service_aggregations.go : agregations (map
// breakdown, sessions, top weapons, outcomes over time, filtres canonical)
// utilisees par TimeseriesService. Decoupe de timeseries_service.go
// (god-file split, refactor 2026-05-27).
package service

import (
	"fmt"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// ---------------------------------------------------------------------------
// Map breakdown (charts teammates.02 + teammates.13 — page Stats Solo)
// ---------------------------------------------------------------------------

// filterStatsMatchRowsByContext applique uniquement le filtre match_context
// (solo/squad/all) sur des StatsMatchRow — sans cascade, period ni sessions.
// Sert à dériver la population "Historique" pour le map_breakdown.
func filterStatsMatchRowsByContext(rows []legacymatch.StatsMatchRow, ctx string) []legacymatch.StatsMatchRow {
	switch ctx {
	case domain.MatchContextSolo:
		out := make([]legacymatch.StatsMatchRow, 0, len(rows))
		for _, r := range rows {
			if !r.IsWithFriends {
				out = append(out, r)
			}
		}
		return out
	case domain.MatchContextSquad:
		out := make([]legacymatch.StatsMatchRow, 0, len(rows))
		for _, r := range rows {
			if r.IsWithFriends {
				out = append(out, r)
			}
		}
		return out
	}
	return rows
}

// buildSoloMapBreakdown agrège les stats par carte sur la session courante
// (matches filtrés) et enrichit chaque ligne avec les références historiques
// (historicalSolo : tous les matchs solo, sans cascade/period/sessions).
//
// Symétrie avec computeMapBreakdown + enrichMapBreakdownWithSquadStats du
// service teammates, adapté à StatsMatchRow + sans notion d'escouade.
//
// Clé d'agrégation : MapNameFR (priorité) puis MapName (fallback EN). Une
// carte présente dans la session sans donnée historique reste sans
// historical_* (la cellule front affiche alors "—" ou se masque).
func buildSoloMapBreakdown(current, historicalSolo []legacymatch.StatsMatchRow) []domain.MapBreakdownRow {
	if len(current) == 0 {
		return []domain.MapBreakdownRow{}
	}
	currentByMap := aggregateMapStats(current)
	historicalByMap := aggregateMapStats(historicalSolo)

	out := make([]domain.MapBreakdownRow, 0, len(currentByMap))
	for key, s := range currentByMap {
		if s.count == 0 {
			continue
		}
		row := domain.MapBreakdownRow{
			MapID:      key,
			MapUI:      s.label,
			MatchCount: s.count,
			WinRate:    round2(float64(s.wins) / float64(s.count)),
		}
		if s.perfCount > 0 {
			avg := round2(s.perfSum / float64(s.perfCount))
			row.PerformanceAvg = &avg
		}
		if h, ok := historicalByMap[key]; ok && h.count > 0 {
			hwr := round2(float64(h.wins) / float64(h.count))
			row.HistoricalWinRate = &hwr
			if h.perfCount > 0 {
				hperf := round2(h.perfSum / float64(h.perfCount))
				row.HistoricalPerformanceAvg = &hperf
			}
		}
		out = append(out, row)
	}
	// Tri stable par count desc puis label asc — confort UX (top maps en haut).
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].MatchCount != out[j].MatchCount {
			return out[i].MatchCount > out[j].MatchCount
		}
		return out[i].MapUI < out[j].MapUI
	})
	return out
}

type mapAgg struct {
	label       string
	count, wins int
	perfSum     float64
	perfCount   int
}

func aggregateMapStats(rows []legacymatch.StatsMatchRow) map[string]*mapAgg {
	m := make(map[string]*mapAgg, 16)
	for _, r := range rows {
		label := r.MapNameFR
		if label == "" {
			label = r.MapName
		}
		if label == "" {
			label = tsLabelUnknown
		}
		// Clé = label affiché (StatsMatchRow n'expose pas de map_id).
		// Cohérent avec le fallback de computeMapBreakdown côté squad.
		key := label
		if _, ok := m[key]; !ok {
			m[key] = &mapAgg{label: label}
		}
		m[key].count++
		if r.Outcome != nil && *r.Outcome == analysis.OutcomeWin {
			m[key].wins++
		}
		if r.PerfScoreComputed != nil {
			m[key].perfSum += *r.PerfScoreComputed
			m[key].perfCount++
		}
	}
	return m
}

// ---------------------------------------------------------------------------
// Solo session performance (Synthèse — agrégat par session sur tout l'historique)
// ---------------------------------------------------------------------------

// buildSoloSessionPerf agrège les matchs solo (sans filtre period/session/
// cascade) par session_label, puis re-agrège en semaines/mois si la densité
// devient illisible.
//
// Granularité adaptative :
//   - ≤ 30 sessions : session par session (label original)
//   - 31..30 semaines : groupe par semaine ISO (label "2026-S12")
//   - sinon : groupe par mois (label "Jan 26")
func buildSoloSessionPerf(rows []legacymatch.StatsMatchRow) *domain.SoloSessionPerfBlock {
	if len(rows) == 0 {
		return nil
	}
	const sessionCap = 30
	const weekCap = 30

	sessionPts := aggregateSessions(rows)
	if len(sessionPts) == 0 {
		return nil
	}
	if len(sessionPts) <= sessionCap {
		return &domain.SoloSessionPerfBlock{Granularity: "session", Points: sessionPts}
	}
	weekPts := rollupSessionPoints(sessionPts, "week")
	if len(weekPts) <= weekCap {
		return &domain.SoloSessionPerfBlock{Granularity: "week", Points: weekPts}
	}
	monthPts := rollupSessionPoints(sessionPts, "month")
	return &domain.SoloSessionPerfBlock{Granularity: "month", Points: monthPts}
}

// aggregateSessions agrège les rows par session_label (granularité finest).
func aggregateSessions(rows []legacymatch.StatsMatchRow) []domain.SoloSessionPerfPoint {
	type acc struct {
		label                            string
		earliest                         time.Time
		count, wins, perfCount, mmrCount int
		perfSum, mmrSum                  float64
	}
	bySession := make(map[string]*acc)
	for _, r := range rows {
		if r.SessionLabel == nil || *r.SessionLabel == "" {
			continue
		}
		lbl := *r.SessionLabel
		a, ok := bySession[lbl]
		if !ok {
			a = &acc{label: lbl, earliest: r.StartTime}
			bySession[lbl] = a
		}
		a.count++
		if r.Outcome != nil && *r.Outcome == domain.OutcomeWin {
			a.wins++
		}
		if r.PerfScoreComputed != nil {
			a.perfSum += *r.PerfScoreComputed
			a.perfCount++
		}
		if r.TeamMMR != nil {
			a.mmrSum += *r.TeamMMR
			a.mmrCount++
		}
		if r.StartTime.Before(a.earliest) {
			a.earliest = r.StartTime
		}
	}
	out := make([]domain.SoloSessionPerfPoint, 0, len(bySession))
	for _, a := range bySession {
		pt := domain.SoloSessionPerfPoint{
			SessionLabel: a.label,
			StartedAtUTC: a.earliest.Format(time.RFC3339),
			MatchCount:   a.count,
			Wins:         a.wins,
			WinRate:      round2(float64(a.wins) / float64(a.count)),
		}
		if a.perfCount > 0 {
			v := round2(a.perfSum / float64(a.perfCount))
			pt.PerfAvg = &v
		}
		if a.mmrCount > 0 {
			v := round2(a.mmrSum / float64(a.mmrCount))
			pt.TeamMMRAvg = &v
		}
		out = append(out, pt)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].StartedAtUTC < out[j].StartedAtUTC
	})
	return out
}

// rollupSessionPoints regroupe des SoloSessionPerfPoint par bucket temporel
// (week|month). Pondère perf/mmr par match_count pour préserver la
// représentativité des sessions denses.
func rollupSessionPoints(points []domain.SoloSessionPerfPoint, mode string) []domain.SoloSessionPerfPoint {
	type acc struct {
		label, startISO                          string
		earliest                                 time.Time
		count, wins, perfWeightedN, mmrWeightedN int
		perfWeightedSum, mmrWeightedSum          float64
	}
	keyFn := func(t time.Time) (string, string, time.Time) {
		switch mode {
		case "month":
			d := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			return d.Format("2006-01"), d.Format(time.RFC3339), d
		default: // "week"
			y, w := t.ISOWeek()
			jan4 := time.Date(y, 1, 4, 0, 0, 0, 0, time.UTC)
			start := jan4.AddDate(0, 0, -int(jan4.Weekday()-time.Monday)+(w-1)*7)
			return fmt.Sprintf("%d-S%02d", y, w), start.Format(time.RFC3339), start
		}
	}
	buckets := make(map[string]*acc)
	for _, p := range points {
		t, err := time.Parse(time.RFC3339, p.StartedAtUTC)
		if err != nil {
			continue
		}
		key, startISO, start := keyFn(t)
		a, ok := buckets[key]
		if !ok {
			a = &acc{label: key, startISO: startISO, earliest: start}
			buckets[key] = a
		}
		a.count += p.MatchCount
		a.wins += p.Wins
		if p.PerfAvg != nil && p.MatchCount > 0 {
			a.perfWeightedSum += *p.PerfAvg * float64(p.MatchCount)
			a.perfWeightedN += p.MatchCount
		}
		if p.TeamMMRAvg != nil && p.MatchCount > 0 {
			a.mmrWeightedSum += *p.TeamMMRAvg * float64(p.MatchCount)
			a.mmrWeightedN += p.MatchCount
		}
	}
	out := make([]domain.SoloSessionPerfPoint, 0, len(buckets))
	for _, a := range buckets {
		pt := domain.SoloSessionPerfPoint{
			SessionLabel: a.label,
			StartedAtUTC: a.startISO,
			MatchCount:   a.count,
			Wins:         a.wins,
			WinRate:      round2(float64(a.wins) / float64(a.count)),
		}
		if a.perfWeightedN > 0 {
			v := round2(a.perfWeightedSum / float64(a.perfWeightedN))
			pt.PerfAvg = &v
		}
		if a.mmrWeightedN > 0 {
			v := round2(a.mmrWeightedSum / float64(a.mmrWeightedN))
			pt.TeamMMRAvg = &v
		}
		out = append(out, pt)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].StartedAtUTC < out[j].StartedAtUTC
	})
	return out
}

// ---------------------------------------------------------------------------
// Top weapons (chart .04)
// ---------------------------------------------------------------------------

// buildTopWeapons trie par kills desc et retourne le top N.
func buildTopWeapons(rows []port.WeaponKillRow, topN int) []domain.TimeseriesWeaponKill {
	if len(rows) == 0 {
		return []domain.TimeseriesWeaponKill{}
	}
	type agg struct {
		label string
		class string
		kills int
	}
	byID := make(map[int64]*agg, len(rows))
	for _, r := range rows {
		if r.IsGrenadeMelee {
			continue
		}
		a, ok := byID[r.WeaponID]
		if !ok {
			a = &agg{label: r.Label, class: r.Class}
			byID[r.WeaponID] = a
		}
		if a.label == "" && r.Label != "" {
			a.label = r.Label
		}
		// Class porté depuis le registre (ResolveRoles) pour recolorer le bar chart
		// par classe (cohérence sunburst v2). Une arme = une classe → 1re valeur non vide.
		if a.class == "" && r.Class != "" {
			a.class = r.Class
		}
		a.kills += r.Kills
	}
	out := make([]domain.TimeseriesWeaponKill, 0, len(byID))
	for id, a := range byID {
		out = append(out, domain.TimeseriesWeaponKill{
			WeaponID: id,
			Label:    a.label,
			Kills:    a.kills,
			Class:    a.class,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kills != out[j].Kills {
			return out[i].Kills > out[j].Kills
		}
		return out[i].WeaponID < out[j].WeaponID
	})
	if len(out) > topN {
		out = out[:topN]
	}
	return out
}

// ---------------------------------------------------------------------------
// Outcomes over time (chart .05)
// ---------------------------------------------------------------------------

// buildOutcomesOverTime agrège les outcomes par période. La granularité est
// choisie selon la durée totale du scope : <=14j → day, <=120j → week, sinon
// month. Les périodes vides ne sont pas émises.
func buildOutcomesOverTime(matches []legacymatch.StatsMatchRow) []domain.OutcomesPeriodPoint {
	if len(matches) == 0 {
		return []domain.OutcomesPeriodPoint{}
	}
	first := matches[0].StartTime
	last := matches[len(matches)-1].StartTime
	if last.Before(first) {
		first, last = last, first
	}
	totalDays := int(last.Sub(first).Hours()/24) + 1
	type bucket struct {
		startDate  time.Time
		label      string
		w, l, t, d int
	}
	buckets := make(map[string]*bucket)
	keyFn := func(t time.Time) (string, time.Time, string) {
		switch {
		case totalDays <= 14:
			d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			return d.Format("2006-01-02"), d, d.Format("02 Jan")
		case totalDays <= 120:
			y, w := t.ISOWeek()
			// Lundi de la semaine ISO
			jan4 := time.Date(y, 1, 4, 0, 0, 0, 0, time.UTC)
			startISO := jan4.AddDate(0, 0, -int(jan4.Weekday()-time.Monday)+(w-1)*7)
			label := fmt.Sprintf("%d-W%02d", y, w)
			return label, startISO, label
		default:
			d := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			return d.Format("2006-01"), d, d.Format("Jan 06")
		}
	}
	for _, m := range matches {
		key, start, label := keyFn(m.StartTime)
		b, ok := buckets[key]
		if !ok {
			b = &bucket{startDate: start, label: label}
			buckets[key] = b
		}
		if m.Outcome == nil {
			b.d++
			continue
		}
		switch *m.Outcome {
		case domain.OutcomeWin:
			b.w++
		case domain.OutcomeLoss:
			b.l++
		case domain.OutcomeDraw:
			b.t++
		default:
			b.d++
		}
	}
	out := make([]domain.OutcomesPeriodPoint, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, domain.OutcomesPeriodPoint{
			PeriodLabel: b.label,
			StartDate:   b.startDate.Format("2006-01-02"),
			Wins:        b.w,
			Losses:      b.l,
			Ties:        b.t,
			DNF:         b.d,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].StartDate < out[j].StartDate
	})
	return out
}

// filterCanonicalByMatchIDs ne garde que les canonical rows dont le match_id
// figure dans le slice de StatsMatchRow filtre. Sert de pont entre la pipeline
// legacy (filterStatsMatchRows operant sur StatsMatchRow) et ComputeKPIStats
// (qui consomme du canonical.PlayerMatchRow).
func filterCanonicalByMatchIDs(canonicalRows []canonical.PlayerMatchRow, matches []legacymatch.StatsMatchRow) []canonical.PlayerMatchRow {
	if len(matches) == 0 || len(canonicalRows) == 0 {
		return nil
	}
	keep := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		keep[m.MatchID] = struct{}{}
	}
	out := make([]canonical.PlayerMatchRow, 0, len(matches))
	for _, r := range canonicalRows {
		if _, ok := keep[r.Summary.MatchID]; ok {
			out = append(out, r)
		}
	}
	return out
}
