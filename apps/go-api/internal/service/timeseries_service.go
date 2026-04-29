// Package service â€” TimeseriesService : endpoint POST /pages/timeseries (contrat FastAPI).
//
// Sprint 33 : adapte les donnÃ©es StatsService vers le contrat TimeseriesPageResponse.
//
// Architecture data-only : le Go ne gÃ©nÃ¨re pas de figures Plotly. Le frontend
// React reconstruit les visualisations via les wrappers ECharts Ã  partir des
// data points bruts fournis dans les onglets (cumulative_kd, ewma_kd_points,
// kda_buckets, correlation_points, heatmap_data, etc.).
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// TimeseriesService construit la rÃ©ponse timeseries au format FastAPI.
type TimeseriesService struct {
	statsRepo port.StatsRepository
	// dataAdapter (optionnel, Phase C+ multi-titres) : point d'extension pour
	// router LoadTimeseries via la couche canonique. Ã€ ce jour, le service
	// utilise le repo direct car canonical.MetricSeries ne couvre pas encore
	// la totalitÃ© du payload (5 onglets : win_loss/accuracy/objective/form/
	// lusr). Le hook est en place pour permettre une bascule incrÃ©mentale.
	dataAdapter games.TitleDataAdapter
	// playerMatchesRepo (P4.1, ADR 0011) : loader canonical-aware optionnel.
	// Quand fourni avec titleSlug + gamertag, GetPage charge canonical et
	// convertit via statsMatchRowFromCanonical (partagÃ© avec stats_service).
	// TODO P4.3 : retirer le converter quand les analyses timeseries
	// (buildCumulTab, computeRegressionStats, etc.) consommeront canonical.
	playerMatchesRepo port.PlayerMatchesRepository
	titleSlug         string
	gamertag          string
}

// NewTimeseriesService crÃ©e un TimeseriesService.
func NewTimeseriesService(repo port.StatsRepository) *TimeseriesService {
	return &TimeseriesService{statsRepo: repo}
}

// WithDataAdapter injecte le DataAdapter multi-titres pour activer une
// future bascule LoadTimeseries. DÃ©gradation gracieuse si nil.
func (s *TimeseriesService) WithDataAdapter(a games.TitleDataAdapter) *TimeseriesService {
	s.dataAdapter = a
	return s
}

// WithPlayerMatchesRepo (P4.1, ADR 0011) injecte le loader canonical-aware.
func (s *TimeseriesService) WithPlayerMatchesRepo(repo port.PlayerMatchesRepository, titleSlug, gamertag string) *TimeseriesService {
	s.playerMatchesRepo = repo
	s.titleSlug = titleSlug
	s.gamertag = gamertag
	return s
}

// GetPage construit la rÃ©ponse complÃ¨te avec 5 onglets.
func (s *TimeseriesService) GetPage(
	ctx context.Context,
	req domain.TimeseriesQueryRequest,
) (domain.TimeseriesPageResponse, error) {
	defer func(start time.Time) {
		observability.RecordDurationMS("timeseries_get_page", time.Since(start).Milliseconds())
	}(time.Now())
	// P4.3 finale (ADR 0011) : path canonical exclusif.
	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		return domain.TimeseriesPageResponse{}, fmt.Errorf("TimeseriesService: PlayerMatchesRepo non cÃ¢blÃ© (P4.3 finale exige le wiring DI)")
	}
	canonicalRows, err := s.playerMatchesRepo.LoadPlayerMatches(
		ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
	)
	if err != nil {
		slog.ErrorContext(ctx, "timeseries: chargement canonical", "error", err)
		return domain.TimeseriesPageResponse{}, fmt.Errorf("TimeseriesService: %w", err)
	}
	slog.DebugContext(ctx, "timeseries: loaded canonical",
		"rows", len(canonicalRows), "title_slug", s.titleSlug)
	allMatches := analysis.StatsMatchRowsFromCanonical(canonicalRows)

	matches := filterStatsMatchRows(allMatches, req.Filters)
	slog.DebugContext(ctx, "timeseries: matches chargÃ©s",
		"total", len(allMatches),
		"apres_filtres", len(matches),
	)

	resp := domain.TimeseriesPageResponse{
		TotalMatches:     len(matches),
		MatchRows:        buildMatchRows(matches),
		SummaryTab:       buildTimeseriesSummaryTab(matches),
		CumulTab:         buildCumulTab(matches),
		FormTab:          buildTimeseriesFormTab(matches),
		IntensityTab:     buildIntensityTab(matches),
		DistributionsTab: buildDistributionsTab(matches),
	}

	return resp, nil
}

// ---------------------------------------------------------------------------
// Onglet Summary
// ---------------------------------------------------------------------------

func buildTimeseriesSummaryTab(matches []legacymatch.StatsMatchRow) domain.TimeseriesSummaryTab {
	cards := make([]domain.TimeseriesKpiCard, 0, 6)
	n := len(matches)
	if n == 0 {
		return domain.TimeseriesSummaryTab{KpiCards: cards}
	}

	wins, totalKills, totalDeaths := 0, 0, 0
	accSum, accN := 0.0, 0
	for _, m := range matches {
		if m.Outcome != nil && *m.Outcome == analysis.OutcomeWin {
			wins++
		}
		totalKills += m.Kills
		totalDeaths += m.Deaths
		if m.Accuracy != nil {
			accSum += *m.Accuracy
			accN++
		}
	}

	// TODO P4 ADR 0006 : retirer *100 (convention API canonique 0..1).
	winRate := analysis.WinRate(wins, n) * 100
	kd := 0.0
	if totalDeaths > 0 {
		kd = float64(totalKills) / float64(totalDeaths)
	}

	cards = append(cards,
		domain.TimeseriesKpiCard{Key: "total_matches", Label: "Matchs", Value: fmt.Sprintf("%d", n)},
		domain.TimeseriesKpiCard{Key: "win_rate", Label: "Win Rate", Value: fmt.Sprintf("%.1f%%", winRate)},
		domain.TimeseriesKpiCard{Key: "kd_ratio", Label: "K/D", Value: fmt.Sprintf("%.2f", kd)},
		domain.TimeseriesKpiCard{Key: "kills_per_game", Label: "Kills/game", Value: fmt.Sprintf("%.1f", float64(totalKills)/float64(n))},
	)

	if accN > 0 {
		avgAcc := accSum / float64(accN) * 100
		cards = append(cards, domain.TimeseriesKpiCard{
			Key: "accuracy", Label: "PrÃ©cision", Value: fmt.Sprintf("%.1f%%", avgAcc),
		})
	}

	return domain.TimeseriesSummaryTab{KpiCards: cards}
}

// ---------------------------------------------------------------------------
// Onglet Cumul
// ---------------------------------------------------------------------------

func buildCumulTab(matches []legacymatch.StatsMatchRow) domain.TimeseriesCumulTab {
	n := len(matches)
	if n == 0 {
		return domain.TimeseriesCumulTab{}
	}

	cumulKD := make([]domain.CumulativePoint, 0, n)
	cumulNet := make([]domain.CumulativePoint, 0, n)
	rollingKD := make([]domain.CumulativePoint, 0, n)

	totalKills, totalDeaths, cumulNetVal := 0, 0, 0
	const rollingWindow = 5 // port Python compute_rolling_kd_polars(window=5)

	for i, m := range matches {
		totalKills += m.Kills
		totalDeaths += m.Deaths

		kd := 0.0
		if totalDeaths > 0 {
			kd = float64(totalKills) / float64(totalDeaths)
		}
		cumulKD = append(cumulKD, domain.CumulativePoint{
			Index: i, StartTime: m.StartTime, Value: math.Round(kd*100) / 100,
		})

		net := m.Kills - m.Deaths
		cumulNetVal += net
		cumulNet = append(cumulNet, domain.CumulativePoint{
			Index: i, StartTime: m.StartTime, Value: float64(cumulNetVal),
		})

		// Rolling K/D sur fenÃªtre glissante.
		start := i - rollingWindow + 1
		if start < 0 {
			start = 0
		}
		rk, rd := 0, 0
		for j := start; j <= i; j++ {
			rk += matches[j].Kills
			rd += matches[j].Deaths
		}
		rkd := 0.0
		if rd > 0 {
			rkd = float64(rk) / float64(rd)
		}
		rollingKD = append(rollingKD, domain.CumulativePoint{
			Index: i, StartTime: m.StartTime, Value: math.Round(rkd*100) / 100,
		})
	}

	return domain.TimeseriesCumulTab{
		CumulativeKD:  cumulKD,
		CumulativeNet: cumulNet,
		RollingKD:     rollingKD,
	}
}

// ---------------------------------------------------------------------------
// Onglet Form
// ---------------------------------------------------------------------------

func buildTimeseriesFormTab(matches []legacymatch.StatsMatchRow) domain.TimeseriesFormTab {
	regStats := computeRegressionStats(matches)

	// EWMA K/D (exponentially weighted moving average).
	// alpha = 0.20 â€” aligneÌ sur Python plot_ewma_kd (alpha=0.20).
	const alpha = 0.20
	ewmaPoints := make([]domain.CumulativePoint, 0, len(matches))
	var ewma float64
	for i, m := range matches {
		kd := 0.0
		if m.Deaths > 0 {
			kd = float64(m.Kills) / float64(m.Deaths)
		}
		if i == 0 {
			ewma = kd
		} else {
			ewma = alpha*kd + (1-alpha)*ewma
		}
		ewmaPoints = append(ewmaPoints, domain.CumulativePoint{
			Index: i, StartTime: m.StartTime,
			Value: math.Round(ewma*100) / 100,
		})
	}

	return domain.TimeseriesFormTab{
		RegressionStats: regStats,
		EWMAKDPoints:    ewmaPoints,
	}
}

// computeRegressionStats calcule les statistiques de rÃ©gression.
func computeRegressionStats(matches []legacymatch.StatsMatchRow) domain.TimeseriesRegressionStats {
	const minForTrend = 20

	n := len(matches)
	if n < minForTrend {
		return domain.TimeseriesRegressionStats{HasEnoughForTrend: false}
	}

	// RÃ©gression linÃ©aire simple sur le K/D.
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	count := 0
	for i, m := range matches {
		kd := 0.0
		if m.Deaths > 0 {
			kd = float64(m.Kills) / float64(m.Deaths)
		}
		x := float64(i)
		sumX += x
		sumY += kd
		sumXY += x * kd
		sumX2 += x * x
		count++
	}

	if count < minForTrend {
		return domain.TimeseriesRegressionStats{HasEnoughForTrend: false}
	}

	fn := float64(count)
	denom := fn*sumX2 - sumX*sumX
	if denom == 0 {
		return domain.TimeseriesRegressionStats{HasEnoughForTrend: false}
	}

	slope := (fn*sumXY - sumX*sumY) / denom
	meanY := sumY / fn

	// RÂ² approximation.
	ssTot, ssRes := 0.0, 0.0
	intercept := (sumY - slope*sumX) / fn
	for i, m := range matches {
		kd := 0.0
		if m.Deaths > 0 {
			kd = float64(m.Kills) / float64(m.Deaths)
		}
		predicted := intercept + slope*float64(i)
		ssRes += (kd - predicted) * (kd - predicted)
		ssTot += (kd - meanY) * (kd - meanY)
	}

	var r2 *float64
	if ssTot > 0 {
		v := math.Max(0, 1-ssRes/ssTot)
		r2 = &v
	}

	kdSlope := math.Round(slope*1000) / 1000

	// Trend detection.
	var trend *string
	if math.Abs(slope) < 0.001 {
		t := trendLabelStable
		trend = &t
	} else if slope > 0 {
		t := "improving"
		trend = &t
	} else {
		t := "declining"
		trend = &t
	}

	return domain.TimeseriesRegressionStats{
		KDSlope:           &kdSlope,
		RSquared:          r2,
		HasEnoughForTrend: true,
		Trend:             trend,
	}
}

// ---------------------------------------------------------------------------
// Onglet IntensitÃ© (Sprint 42)
// ---------------------------------------------------------------------------

// buildIntensityTab construit la heatmap jourÃ—heure et le score/min.
func buildIntensityTab(matches []legacymatch.StatsMatchRow) domain.TimeseriesIntensityTab {
	if len(matches) == 0 {
		return domain.TimeseriesIntensityTab{
			HeatmapData:     []domain.IntensityHeatmapPoint{},
			ScorePerMinData: []domain.CumulativePoint{},
		}
	}

	// Heatmap jour Ã— heure.
	type cell struct {
		kills, deaths, count int
	}
	heatmap := make(map[[2]int]*cell) // [day, hour]

	scorePerMin := make([]domain.CumulativePoint, 0, len(matches))

	for i, m := range matches {
		day := int(m.StartTime.Weekday())
		// Convertir Sunday=0 â†’ Monday=0..Sunday=6
		day = (day + 6) % 7
		hour := m.StartTime.Hour()
		key := [2]int{day, hour}
		c, ok := heatmap[key]
		if !ok {
			c = &cell{}
			heatmap[key] = c
		}
		c.kills += m.Kills
		c.deaths += m.Deaths
		c.count++

		// Score per minute.
		spm := 0.0
		if m.PersonalScore != nil && m.TimePlayedSeconds != nil && *m.TimePlayedSeconds > 0 {
			spm = float64(*m.PersonalScore) / (float64(*m.TimePlayedSeconds) / 60.0)
		}
		scorePerMin = append(scorePerMin, domain.CumulativePoint{
			Index: i, StartTime: m.StartTime,
			Value: math.Round(spm*100) / 100,
		})
	}

	points := make([]domain.IntensityHeatmapPoint, 0, len(heatmap))
	for key, c := range heatmap {
		avgKD := 0.0
		if c.deaths > 0 {
			avgKD = float64(c.kills) / float64(c.deaths)
		}
		points = append(points, domain.IntensityHeatmapPoint{
			DayOfWeek: key[0],
			Hour:      key[1],
			Count:     c.count,
			AvgKD:     math.Round(avgKD*100) / 100,
		})
	}

	return domain.TimeseriesIntensityTab{
		HeatmapData:     points,
		ScorePerMinData: scorePerMin,
	}
}

// ---------------------------------------------------------------------------
// Onglet Distributions (Sprint 42)
// ---------------------------------------------------------------------------

// buildDistributionsTab construit les histogrammes KDA/kills et les corrÃ©lations.
func buildDistributionsTab(matches []legacymatch.StatsMatchRow) domain.TimeseriesDistributionsTab {
	if len(matches) == 0 {
		return domain.TimeseriesDistributionsTab{
			KDABuckets:         []domain.DistributionBucket{},
			KillsBuckets:       []domain.DistributionBucket{},
			AccuracyBuckets:    []domain.DistributionBucket{},
			ScorePerMinBuckets: []domain.DistributionBucket{},
			RollingWRBuckets:   []domain.DistributionBucket{},
			CorrelationPoints:  []domain.CorrelationDataPair{},
		}
	}

	return domain.TimeseriesDistributionsTab{
		KDABuckets:         buildKDABuckets(matches),
		KillsBuckets:       buildKillsBuckets(matches),
		AccuracyBuckets:    buildAccuracyBuckets(matches),
		ScorePerMinBuckets: buildScorePerMinBuckets(matches),
		RollingWRBuckets:   buildRollingWRBuckets(matches),
		CorrelationPoints:  buildCorrelationPoints(matches),
	}
}

// buildKDABuckets crÃ©e des buckets pour la distribution K/D.
func buildKDABuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const (
		binWidth = 0.25
		maxBin   = 5.0
	)
	numBins := int(maxBin / binWidth)
	counts := make([]int, numBins+1) // dernier = overflow

	for _, m := range matches {
		kd := 0.0
		if m.Deaths > 0 {
			kd = float64(m.Kills) / float64(m.Deaths)
		}
		idx := int(kd / binWidth)
		if idx >= numBins {
			idx = numBins
		}
		counts[idx]++
	}

	buckets := make([]domain.DistributionBucket, 0, numBins+1)
	for i, c := range counts {
		if c == 0 {
			continue
		}
		start := float64(i) * binWidth
		end := start + binWidth
		if i == numBins {
			end = maxBin + binWidth
		}
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: math.Round(start*100) / 100,
			BucketUpper: math.Round(end*100) / 100,
			Count:       c,
		})
	}
	return buckets
}

// buildKillsBuckets crÃ©e des buckets pour la distribution des kills par match.
func buildKillsBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const binWidth = 5.0
	maxKills := 0
	for _, m := range matches {
		if m.Kills > maxKills {
			maxKills = m.Kills
		}
	}
	numBins := maxKills/int(binWidth) + 1
	counts := make([]int, numBins+1)

	for _, m := range matches {
		idx := m.Kills / int(binWidth)
		if idx > numBins {
			idx = numBins
		}
		counts[idx]++
	}

	buckets := make([]domain.DistributionBucket, 0, numBins+1)
	for i, c := range counts {
		if c == 0 {
			continue
		}
		start := float64(i) * binWidth
		end := start + binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start,
			BucketUpper: end,
			Count:       c,
		})
	}
	return buckets
}

// buildCorrelationPoints construit les paires de corrÃ©lation pour 5 types de scatter.
func buildCorrelationPoints(matches []legacymatch.StatsMatchRow) []domain.CorrelationDataPair {
	points := make([]domain.CorrelationDataPair, 0, len(matches)*5)
	for _, m := range matches {
		kd := 0.0
		if m.Deaths > 0 {
			kd = float64(m.Kills) / float64(m.Deaths)
		}
		// DurÃ©e de vie approchÃ©e : time_played / (deaths+1)
		lifespan := 0.0
		if m.TimePlayedSeconds != nil && *m.TimePlayedSeconds > 0 {
			lifespan = float64(*m.TimePlayedSeconds) / float64(m.Deaths+1)
		}

		// P7.1 (revue 2026-04-29) : Label composite ("kills_vs_kd") séparé en
		// MetricXKey/MetricYKey ; X/Y → XValue/YValue.
		points = append(points, domain.CorrelationDataPair{
			MetricXKey: "kills", MetricYKey: "kd_ratio",
			XValue: float64(m.Kills), YValue: math.Round(kd*100) / 100, Outcome: m.Outcome,
		})
		points = append(points, domain.CorrelationDataPair{
			MetricXKey: "lifespan", MetricYKey: "kills",
			XValue: math.Round(lifespan*10) / 10, YValue: float64(m.Kills), Outcome: m.Outcome,
		})
		if m.Accuracy != nil && m.KDA != nil {
			points = append(points, domain.CorrelationDataPair{
				MetricXKey: "accuracy", MetricYKey: "kda",
				XValue: math.Round(*m.Accuracy*1000) / 10, YValue: math.Round(*m.KDA*100) / 100, Outcome: m.Outcome,
			})
		}
		points = append(points, domain.CorrelationDataPair{
			MetricXKey: "lifespan", MetricYKey: "deaths",
			XValue: math.Round(lifespan*10) / 10, YValue: float64(m.Deaths), Outcome: m.Outcome,
		})
		points = append(points, domain.CorrelationDataPair{
			MetricXKey: "kills", MetricYKey: "deaths",
			XValue: float64(m.Kills), YValue: float64(m.Deaths), Outcome: m.Outcome,
		})
		if m.TeamMMR != nil && m.EnemyMMR != nil {
			points = append(points, domain.CorrelationDataPair{
				MetricXKey: "mmr_team", MetricYKey: "mmr_enemy",
				XValue: math.Round(*m.TeamMMR*100) / 100, YValue: math.Round(*m.EnemyMMR*100) / 100, Outcome: m.Outcome,
			})
		}
	}
	return points
}

// ---------------------------------------------------------------------------
// Lignes match brutes (pour les charts timeline React)
// ---------------------------------------------------------------------------

// buildMatchRows convertit StatsMatchRow en TimeseriesMatchRow (1 ligne = 1 match).
//
// KDA et KDRatio sont calcules par P2.5 (revue 2026-04-29 ADR 0006) â€” debloque
// la suppression du recompute K/D cote front (TimeseriesKdaBars.tsx:78, B3).
func buildMatchRows(matches []legacymatch.StatsMatchRow) []domain.TimeseriesMatchRow {
	rows := make([]domain.TimeseriesMatchRow, 0, len(matches))
	for i, m := range matches {
		// KDR canonique calcule a partir des compteurs (analysis.KDR).
		// Distinct du KDA pre-calcule cote sync (m.KDA inclut les assists).
		kdr := analysis.KDR(m.Kills, m.Deaths)
		rows = append(rows, domain.TimeseriesMatchRow{
			MatchID:           m.MatchID,
			Index:             i,
			StartTime:         m.StartTime,
			Kills:             m.Kills,
			Deaths:            m.Deaths,
			Assists:           m.Assists,
			KDA:               m.KDA,
			KDRatio:           &kdr,
			Accuracy:          m.Accuracy,
			Outcome:           m.Outcome,
			PersonalScore:     m.PersonalScore,
			DamageDealt:       m.DamageDealt,
			DamageTaken:       m.DamageTaken,
			PerfScore:         m.PerfScoreComputed,
			Rank:              m.Rank,
			PlaylistName:      m.PlaylistName,
			TimePlayedSeconds: m.TimePlayedSeconds,
		})
	}
	return rows
}

// ---------------------------------------------------------------------------
// Histogrammes supplÃ©mentaires (PrÃ©cision, Score/min, Win Rate glissant)
// ---------------------------------------------------------------------------

// buildAccuracyBuckets crÃ©e des buckets de 5 % pour la distribution de prÃ©cision.
func buildAccuracyBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const binWidth = 5.0      // 5 % par bin (valeurs [0, 1] â†’ converties en %)
	counts := make([]int, 21) // bins 0-5, 5-10, â€¦, 95-100, 100+

	for _, m := range matches {
		if m.Accuracy == nil {
			continue
		}
		pct := *m.Accuracy * 100
		idx := int(pct / binWidth)
		if idx >= len(counts) {
			idx = len(counts) - 1
		}
		counts[idx]++
	}

	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for i, c := range counts {
		if c == 0 {
			continue
		}
		start := float64(i) * binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start, BucketUpper: start + binWidth, Count: c,
		})
	}
	return buckets
}

// buildScorePerMinBuckets crÃ©e des buckets de 10 pts/min pour la distribution score/min.
func buildScorePerMinBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const binWidth = 10.0
	counts := make(map[int]int)

	for _, m := range matches {
		if m.PersonalScore == nil || m.TimePlayedSeconds == nil || *m.TimePlayedSeconds == 0 {
			continue
		}
		spm := float64(*m.PersonalScore) / (float64(*m.TimePlayedSeconds) / 60.0)
		idx := int(spm / binWidth)
		counts[idx]++
	}

	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for idx, c := range counts {
		start := float64(idx) * binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start, BucketUpper: start + binWidth, Count: c,
		})
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].BucketLower < buckets[j].BucketLower })
	return buckets
}

// buildRollingWRBuckets crÃ©e des buckets de 5 % pour la distribution du win-rate glissant (fenÃªtre 14).
func buildRollingWRBuckets(matches []legacymatch.StatsMatchRow) []domain.DistributionBucket {
	const (
		window   = 14
		binWidth = 5.0
	)
	counts := make([]int, 21) // bins 0-5, 5-10, â€¦, 95-100

	for i := range matches {
		start := i - window + 1
		if start < 0 {
			start = 0
		}
		wins := 0
		for j := start; j <= i; j++ {
			if matches[j].Outcome != nil && *matches[j].Outcome == analysis.OutcomeWin {
				wins++
			}
		}
		total := i - start + 1
		// TODO P4 ADR 0006 : retirer *100 (convention API canonique 0..1).
		wr := analysis.WinRate(wins, total) * 100
		idx := int(wr / binWidth)
		if idx >= len(counts) {
			idx = len(counts) - 1
		}
		counts[idx]++
	}

	buckets := make([]domain.DistributionBucket, 0, len(counts))
	for i, c := range counts {
		if c == 0 {
			continue
		}
		start := float64(i) * binWidth
		buckets = append(buckets, domain.DistributionBucket{
			BucketLower: start, BucketUpper: start + binWidth, Count: c,
		})
	}
	return buckets
}
