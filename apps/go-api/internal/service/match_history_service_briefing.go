// Package service — match_history_service_briefing.go : construction du bandeau
// de briefing de l'Explorer (mode Matchs), servi au-dessus du tableau des matchs
// filtrés quand la requête pose include_briefing=true.
//
// Le socle (KPIs personnels du scope) réutilise les canonical KPIs déjà calculés
// (kpisFromScoped). Les modules baseline / dimensions / tendance sont bâtis sur
// les MatchHistoryRawRow : celles-ci portent déjà les libellés FR résolus
// (MapNameFR / PairNameFR / PlaylistName coalescé) — les canonical rows chargées
// par LoadPlayerMatches ne sont PAS enrichies FR, les utiliser afficherait des
// libellés anglais sous locale FR. Les raw rows « full history » sont en outre
// déjà post-exclusions manuelles (= baseline DEC-3). Le module classé réutilise
// le RankDelta canonique (delta CSR) + les probabilités pré-match des raw rows.
//
// Contrat produit : PLAN_EXPLORER_BRIEFING_CARDS_2026-07.md (DEC-1..8).
package service

import (
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/breakdown"
	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// Seuils nommés du briefing (DEC-4 : pas de magic numbers).
const (
	// MinBriefingModulesMatches est le seuil sous lequel le briefing ne sert que
	// le socle (KPIs + frise + période) avec LowSample=true ; baseline / dimensions
	// / tendance / classé sont omis. Aligné sur analysis.MinMatchesForRelative
	// (activation du score relatif).
	MinBriefingModulesMatches = analysis.MinMatchesForRelative
	// MinDimensionGroupMatches est le seuil sous lequel un groupe (carte/mode/
	// playlist) n'a pas de note et n'apparaît pas en top/flop.
	MinDimensionGroupMatches = 10
	// minTrendMatches / minTrendSpanDays : activation du module tendance (DEC-4) —
	// assez de matchs ET d'étalement temporel pour une courbe lisible.
	minTrendMatches  = 20
	minTrendSpanDays = 14
	// maxOutcomeSequencePoints borne la frise des résultats aux N derniers matchs.
	maxOutcomeSequencePoints = 60
	// dimensionTopFlopCount : nombre d'entrées top ET flop par dimension (DEC-8).
	dimensionTopFlopCount = 3
)

// buildExplorerBriefing assemble le bandeau de briefing du scope filtré.
//
//   - filtered : sous-ensemble filtré (scope) en raw rows (libellés FR + start_time).
//   - allRaw   : historique complet post-exclusions (baseline DEC-3) en raw rows.
//   - scopedKPIs : KPIs canoniques du scope (socle + source du RankDelta classé).
//
// Retourne nil si le scope est vide. LowSample (scope < seuil) → socle seul.
func (s *MatchHistoryService) buildExplorerBriefing(
	filtered, allRaw []domain.MatchHistoryRawRow,
	scopedKPIs *domain.KPIStats,
) *domain.ExplorerBriefing {
	if len(filtered) == 0 {
		return nil
	}
	b := &domain.ExplorerBriefing{KPIs: scopedKPIs}
	b.PeriodStart, b.PeriodEnd = scopePeriod(filtered)
	b.OutcomeSequence = buildOutcomeSequence(filtered)
	b.LowSample = len(filtered) < MinBriefingModulesMatches
	if b.LowSample {
		return b
	}
	b.Baseline = buildBriefingBaseline(filtered, allRaw)
	b.Dimensions = buildBriefingDimensions(filtered, allRaw)
	b.Trend = buildBriefingTrend(filtered, b.PeriodStart, b.PeriodEnd)
	if s.rankedCapable {
		b.Ranked = buildBriefingRanked(filtered, scopedKPIs)
	}
	return b
}

// scopePeriod retourne les bornes temporelles (min/max start_time non nil) du scope.
func scopePeriod(rows []domain.MatchHistoryRawRow) (start, end *time.Time) {
	for _, r := range rows {
		if r.StartTime == nil {
			continue
		}
		t := *r.StartTime
		if start == nil || t.Before(*start) {
			v := t
			start = &v
		}
		if end == nil || t.After(*end) {
			v := t
			end = &v
		}
	}
	return start, end
}

// buildOutcomeSequence construit la frise des résultats : N derniers matchs du
// scope (cap maxOutcomeSequencePoints), tri chronologique ascendant. Les rows
// sans start_time sont écartées (non ordonnables).
func buildOutcomeSequence(rows []domain.MatchHistoryRawRow) []domain.ExplorerBriefingOutcome {
	seq := make([]domain.ExplorerBriefingOutcome, 0, len(rows))
	for _, r := range rows {
		if r.StartTime == nil {
			continue
		}
		seq = append(seq, domain.ExplorerBriefingOutcome{
			MatchID:     r.MatchID,
			OutcomeCode: r.Outcome,
			StartTime:   *r.StartTime,
		})
	}
	sort.SliceStable(seq, func(i, j int) bool { return seq[i].StartTime.Before(seq[j].StartTime) })
	if len(seq) > maxOutcomeSequencePoints {
		seq = seq[len(seq)-maxOutcomeSequencePoints:]
	}
	if len(seq) == 0 {
		return nil
	}
	return seq
}

// buildBriefingBaseline compare le scope à l'historique complet (post-exclusions).
// Deltas signés = valeur(scope) − valeur(baseline). nil si l'un des deux est vide.
func buildBriefingBaseline(scope, all []domain.MatchHistoryRawRow) *domain.ExplorerBriefingBaseline {
	if len(scope) == 0 || len(all) == 0 {
		return nil
	}
	baseWins, baseTotal, baseKDA, basePerf := aggregateRawStats(all)
	scopeWins, scopeTotal, scopeKDA, scopePerf := aggregateRawStats(scope)
	baseWR := analysis.WinRate(baseWins, baseTotal)
	scopeWR := analysis.WinRate(scopeWins, scopeTotal)
	out := &domain.ExplorerBriefingBaseline{
		Matches:      baseTotal,
		WinRate:      baseWR,
		KDA:          baseKDA,
		AvgPerf:      basePerf,
		DeltaWinRate: scopeWR - baseWR,
		DeltaKDA:     scopeKDA - baseKDA,
	}
	if scopePerf != nil && basePerf != nil {
		d := *scopePerf - *basePerf
		out.DeltaPerf = &d
	}
	return out
}

// aggregateRawStats agrège wins, total, KDA agrégat ADR 0006 et perf moyenne d'un
// ensemble de raw rows. perf est nil si aucune row ne porte de performance_score.
func aggregateRawStats(rows []domain.MatchHistoryRawRow) (wins, total int, kda float64, perf *float64) {
	var k, a, d int
	var perfSum float64
	var perfN int
	for _, r := range rows {
		total++
		k += r.Kills
		a += r.Assists
		d += r.Deaths
		if r.Outcome == domain.OutcomeWin {
			wins++
		}
		if r.PerformanceScore != nil {
			perfSum += *r.PerformanceScore
			perfN++
		}
	}
	kda = analysis.AggregateKDA(k, a, d, total)
	if perfN > 0 {
		avg := perfSum / float64(perfN)
		perf = &avg
	}
	return wins, total, kda, perf
}

// buildBriefingDimensions émet une carte par dimension LIBRE (≥ 2 valeurs
// distinctes dans le scope, DEC-8) parmi carte / mode / playlist. Par dimension :
// top 3 + flop 3 des groupes qualifiés (≥ MinDimensionGroupMatches), triés par
// delta winrate vs baseline.
func buildBriefingDimensions(scope, all []domain.MatchHistoryRawRow) []domain.ExplorerBriefingDimension {
	if len(scope) == 0 {
		return nil
	}
	scopeRows := rawRowsToBreakdownRows(scope)
	allRows := rawRowsToBreakdownRows(all)
	out := make([]domain.ExplorerBriefingDimension, 0, 3)
	if d := buildDimension("map", mapKeyed(breakdown.ByMap(scopeRows)), mapKeyed(breakdown.ByMap(allRows))); d != nil {
		out = append(out, *d)
	}
	if d := buildDimension("mode", modeKeyed(breakdown.ByMode(scopeRows)), modeKeyed(breakdown.ByMode(allRows))); d != nil {
		out = append(out, *d)
	}
	if d := buildDimension("playlist", playlistKeyed(breakdown.ByPlaylist(scopeRows)), playlistKeyed(breakdown.ByPlaylist(allRows))); d != nil {
		out = append(out, *d)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// buildDimension construit une dimension à partir de ses agrégats scope / historique
// génériques par clé. Retourne nil si la dimension n'est pas libre (< 2 valeurs
// distinctes) ou si aucune entrée qualifiée.
func buildDimension(dim string, sessionKA, histKA []breakdown.KeyedAggregate) *domain.ExplorerBriefingDimension {
	distinct := 0
	for _, a := range sessionKA {
		if a.Played > 0 {
			distinct++
		}
	}
	if distinct < 2 {
		return nil
	}
	perfByKey := make(map[string]*float64, len(sessionKA))
	for _, a := range sessionKA {
		perfByKey[a.Key] = a.AvgPerformanceScore
	}
	deltas := breakdown.CompareByKey(sessionKA, histKA)
	qualified := make([]breakdown.KeyedDelta, 0, len(deltas))
	for _, d := range deltas {
		if d.Session.Played >= MinDimensionGroupMatches {
			qualified = append(qualified, d)
		}
	}
	qualified = selectTopFlop(qualified, dimensionTopFlopCount)
	entries := make([]domain.ExplorerBriefingDimensionEntry, 0, len(qualified))
	for _, d := range qualified {
		avgPerf := perfByKey[d.Key]
		e := domain.ExplorerBriefingDimensionEntry{
			Label:        d.Label,
			Matches:      d.Session.Played,
			WinRate:      d.Session.WinRate,
			DeltaWinRate: d.WinRateDelta,
			AvgPerf:      avgPerf,
		}
		if avgPerf != nil {
			tier := int(analysis.PerfTier(*avgPerf))
			e.NoteTier = &tier
		}
		entries = append(entries, e)
	}
	if len(entries) == 0 {
		return nil
	}
	return &domain.ExplorerBriefingDimension{Dimension: dim, Entries: entries}
}

// selectTopFlop retourne les k premiers + k derniers éléments d'une liste triée.
// Si la liste tient en 2*k, elle est retournée telle quelle (pas de recouvrement).
func selectTopFlop[T any](items []T, k int) []T {
	if len(items) <= 2*k {
		return items
	}
	out := make([]T, 0, 2*k)
	out = append(out, items[:k]...)
	out = append(out, items[len(items)-k:]...)
	return out
}

// buildBriefingTrend bucketise le scope si les seuils DEC-4 sont atteints (assez
// de matchs ET d'étalement temporel). Granularité résolue via temporal.ResolveAdaptive
// à partir de l'étendue. Aucun lissage serveur (série brute par bucket).
func buildBriefingTrend(scope []domain.MatchHistoryRawRow, start, end *time.Time) *domain.ExplorerBriefingTrend {
	if len(scope) < minTrendMatches || start == nil || end == nil {
		return nil
	}
	if end.Sub(*start) < time.Duration(minTrendSpanDays)*24*time.Hour {
		return nil
	}
	spanDays := int(end.Sub(*start).Hours() / 24)
	var period temporal.Period
	switch {
	case spanDays <= 31:
		period = temporal.Period1M // → 1d
	case spanDays <= 366:
		period = temporal.Period1Y // → 1w
	default:
		period = temporal.PeriodAll // → 1m
	}
	gran := temporal.ResolveAdaptive(period)
	rows := make([]trendRow, 0, len(scope))
	for _, r := range scope {
		if r.StartTime == nil {
			continue
		}
		rows = append(rows, trendRow{start: *r.StartTime, outcome: r.Outcome, perf: r.PerformanceScore})
	}
	buckets := temporal.BucketByGranularity(rows, gran, period)
	points := make([]domain.ExplorerBriefingTrendPoint, 0, len(buckets))
	for _, bk := range buckets {
		var wins, perfN int
		var perfSum float64
		for _, it := range bk.Items {
			if it.outcome == domain.OutcomeWin {
				wins++
			}
			if it.perf != nil {
				perfSum += *it.perf
				perfN++
			}
		}
		pt := domain.ExplorerBriefingTrendPoint{
			BucketStart: bk.Start,
			Matches:     len(bk.Items),
			WinRate:     analysis.WinRate(wins, len(bk.Items)),
		}
		if perfN > 0 {
			avg := perfSum / float64(perfN)
			pt.AvgPerf = &avg
		}
		points = append(points, pt)
	}
	if len(points) == 0 {
		return nil
	}
	return &domain.ExplorerBriefingTrend{Granularity: string(gran), Points: points}
}

// trendRow adapte une raw row au contrat temporal.HasStartTime pour le bucketing.
type trendRow struct {
	start   time.Time
	outcome int
	perf    *float64
}

func (t trendRow) GetStartTime() time.Time { return t.start }

// buildBriefingRanked émet le module classé si le scope est majoritairement CSR
// (RankDelta.Kind == "csr", DEC-7). DeltaSum réutilise le RankDelta canonique ;
// attendu vs réel calculés sur les matchs CSR du scope (probas pré-match des raw
// rows). nil hors contexte classé.
func buildBriefingRanked(scope []domain.MatchHistoryRawRow, scopedKPIs *domain.KPIStats) *domain.ExplorerBriefingRanked {
	if scopedKPIs == nil || scopedKPIs.RankDelta == nil {
		return nil
	}
	rd := scopedKPIs.RankDelta
	if !strings.EqualFold(rd.Kind, "csr") {
		return nil
	}
	var probs []float64
	var wins, total int
	for _, r := range scope {
		if r.SkillRatingType == nil || !strings.EqualFold(*r.SkillRatingType, "csr") {
			continue
		}
		total++
		if r.Outcome == domain.OutcomeWin {
			wins++
		}
		if r.SkillExpectedWinProb != nil {
			probs = append(probs, *r.SkillExpectedWinProb)
		}
	}
	if total == 0 {
		return nil
	}
	expected, actual := analysis.ExpectedVsActual(probs, wins, total)
	return &domain.ExplorerBriefingRanked{
		RatingKind:            rd.Kind,
		DeltaSum:              rd.Value,
		ExpectedWinRate:       expected,
		ActualWinRate:         actual,
		MatchesWithPrediction: len(probs),
	}
}

// ─── conversion raw row → breakdown.Row (libellés FR) ────────────────────────

// rawRowsToBreakdownRows projette les raw rows vers breakdown.Row en résolvant
// les libellés FR (MapNameFR / PairNameFR normalisé / PlaylistName coalescé).
func rawRowsToBreakdownRows(rows []domain.MatchHistoryRawRow) []breakdown.Row {
	out := make([]breakdown.Row, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		br := breakdown.Row{Outcome: rawOutcomeToCanonical(r.Outcome)}
		if r.PerformanceScore != nil {
			v := *r.PerformanceScore
			br.PerformanceScore = &v
		}
		if r.MapID != nil {
			br.MapID = *r.MapID
		}
		br.MapLabel = coalesceStr(r.MapNameFR, r.MapName)
		if mode := analysis.NormalizeModeLabel(coalesceStr(r.PairNameFR, r.PairName)); mode != "" {
			br.ModeName = mode
		}
		if r.PlaylistName != nil {
			br.PlaylistName = *r.PlaylistName
		}
		out = append(out, br)
	}
	return out
}

// rawOutcomeToCanonical mappe le code outcome int (domain.Outcome*) vers l'enum
// canonical.Outcome consommé par le package breakdown. "" si code inconnu.
func rawOutcomeToCanonical(code int) canonical.Outcome {
	switch code {
	case domain.OutcomeWin:
		return canonical.OutcomeWin
	case domain.OutcomeLoss:
		return canonical.OutcomeLoss
	case domain.OutcomeDraw:
		return canonical.OutcomeTie
	case domain.OutcomeDNF:
		return canonical.OutcomeDNF
	}
	return ""
}

// coalesceStr retourne le premier pointeur non nil / non vide, "" sinon.
func coalesceStr(ptrs ...*string) string {
	for _, p := range ptrs {
		if p != nil && *p != "" {
			return *p
		}
	}
	return ""
}

// mapKeyed / modeKeyed / playlistKeyed projettent les agrégats de dimension vers
// la forme pivot KeyedAggregate (clé stable + libellé) pour breakdown.CompareByKey.
func mapKeyed(aggs []breakdown.MapAggregate) []breakdown.KeyedAggregate {
	out := make([]breakdown.KeyedAggregate, 0, len(aggs))
	for _, a := range aggs {
		label := a.MapLabel
		if label == "" {
			label = a.MapID
		}
		out = append(out, breakdown.KeyedAggregate{
			Key: a.MapID, Label: label, Counts: a.Counts, AvgPerformanceScore: a.AvgPerformanceScore,
		})
	}
	return out
}

func modeKeyed(aggs []breakdown.ModeAggregate) []breakdown.KeyedAggregate {
	out := make([]breakdown.KeyedAggregate, 0, len(aggs))
	for _, a := range aggs {
		out = append(out, breakdown.KeyedAggregate{
			Key: a.ModeName, Label: a.ModeName, Counts: a.Counts, AvgPerformanceScore: a.AvgPerformanceScore,
		})
	}
	return out
}

func playlistKeyed(aggs []breakdown.PlaylistAggregate) []breakdown.KeyedAggregate {
	out := make([]breakdown.KeyedAggregate, 0, len(aggs))
	for _, a := range aggs {
		out = append(out, breakdown.KeyedAggregate{
			Key: a.PlaylistName, Label: a.PlaylistName, Counts: a.Counts, AvgPerformanceScore: a.AvgPerformanceScore,
		})
	}
	return out
}
