// Package service — TimeseriesService : endpoint POST /pages/timeseries (contrat FastAPI).
//
// Sprint 33 : adapte les données StatsService vers le contrat TimeseriesPageResponse.
//
// Décision architecturale Plotly : le Go envoie PlotlyFigurePayload = null pour
// tous les charts. Le frontend React reconstruit les visualisations à partir
// des data points fournis dans les onglets existants du StatsService (win_loss,
// accuracy, objective, form, lusr). Cette approche évite la génération Plotly
// server-side en Go tout en restant compatible avec le contrat TypeScript.
package service

import (
	"context"
	"fmt"
	"math"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// TimeseriesService construit la réponse timeseries au format FastAPI.
type TimeseriesService struct {
	statsRepo port.StatsRepository
}

// NewTimeseriesService crée un TimeseriesService.
func NewTimeseriesService(repo port.StatsRepository) *TimeseriesService {
	return &TimeseriesService{statsRepo: repo}
}

// GetPage construit la réponse complète avec 5 onglets.
func (s *TimeseriesService) GetPage(
	ctx context.Context,
	_ domain.TimeseriesQueryRequest,
) (domain.TimeseriesPageResponse, error) {
	matches, err := s.statsRepo.LoadStatsMatches(ctx)
	if err != nil {
		return domain.TimeseriesPageResponse{}, fmt.Errorf("TimeseriesService: %w", err)
	}

	resp := domain.TimeseriesPageResponse{
		TotalMatches:     len(matches),
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

func buildTimeseriesSummaryTab(matches []domain.StatsMatchRow) domain.TimeseriesSummaryTab {
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

	winRate := float64(wins) / float64(n) * 100
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
			Key: "accuracy", Label: "Précision", Value: fmt.Sprintf("%.1f%%", avgAcc),
		})
	}

	return domain.TimeseriesSummaryTab{KpiCards: cards}
}

// ---------------------------------------------------------------------------
// Onglet Cumul
// ---------------------------------------------------------------------------

func buildCumulTab(matches []domain.StatsMatchRow) domain.TimeseriesCumulTab {
	n := len(matches)
	if n == 0 {
		return domain.TimeseriesCumulTab{}
	}

	cumulKD := make([]domain.CumulativePoint, 0, n)
	cumulNet := make([]domain.CumulativePoint, 0, n)
	rollingKD := make([]domain.CumulativePoint, 0, n)

	totalKills, totalDeaths, cumulNetVal := 0, 0, 0
	const rollingWindow = 20

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

		// Rolling K/D sur fenêtre glissante.
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

func buildTimeseriesFormTab(matches []domain.StatsMatchRow) domain.TimeseriesFormTab {
	regStats := computeRegressionStats(matches)

	// EWMA K/D (exponentially weighted moving average).
	const alpha = 0.1
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

// computeRegressionStats calcule les statistiques de régression.
func computeRegressionStats(matches []domain.StatsMatchRow) domain.TimeseriesRegressionStats {
	const minForTrend = 20

	n := len(matches)
	if n < minForTrend {
		return domain.TimeseriesRegressionStats{HasEnoughForTrend: false}
	}

	// Régression linéaire simple sur le K/D.
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

	// R² approximation.
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
		t := "stable"
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
// Onglet Intensité (Sprint 42)
// ---------------------------------------------------------------------------

// buildIntensityTab construit la heatmap jour×heure et le score/min.
func buildIntensityTab(matches []domain.StatsMatchRow) domain.TimeseriesIntensityTab {
	if len(matches) == 0 {
		return domain.TimeseriesIntensityTab{
			HeatmapData:     []domain.IntensityHeatmapPoint{},
			ScorePerMinData: []domain.CumulativePoint{},
		}
	}

	// Heatmap jour × heure.
	type cell struct {
		kills, deaths, count int
	}
	heatmap := make(map[[2]int]*cell) // [day, hour]

	scorePerMin := make([]domain.CumulativePoint, 0, len(matches))

	for i, m := range matches {
		day := int(m.StartTime.Weekday())
		// Convertir Sunday=0 → Monday=0..Sunday=6
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

// buildDistributionsTab construit les histogrammes KDA/kills et les corrélations.
func buildDistributionsTab(matches []domain.StatsMatchRow) domain.TimeseriesDistributionsTab {
	if len(matches) == 0 {
		return domain.TimeseriesDistributionsTab{
			KDABuckets:        []domain.DistributionBucket{},
			KillsBuckets:      []domain.DistributionBucket{},
			CorrelationPoints: []domain.CorrelationDataPair{},
			Correlations:      []domain.PlotlyFigurePayload{},
		}
	}

	kdaBuckets := buildKDABuckets(matches)
	killsBuckets := buildKillsBuckets(matches)
	correlations := buildCorrelationPoints(matches)

	return domain.TimeseriesDistributionsTab{
		KDABuckets:        kdaBuckets,
		KillsBuckets:      killsBuckets,
		CorrelationPoints: correlations,
		Correlations:      []domain.PlotlyFigurePayload{},
	}
}

// buildKDABuckets crée des buckets pour la distribution K/D.
func buildKDABuckets(matches []domain.StatsMatchRow) []domain.DistributionBucket {
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
			BinStart: math.Round(start*100) / 100,
			BinEnd:   math.Round(end*100) / 100,
			Count:    c,
		})
	}
	return buckets
}

// buildKillsBuckets crée des buckets pour la distribution des kills par match.
func buildKillsBuckets(matches []domain.StatsMatchRow) []domain.DistributionBucket {
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
			BinStart: start,
			BinEnd:   end,
			Count:    c,
		})
	}
	return buckets
}

// buildCorrelationPoints construit les paires de corrélation kills↔K/D.
func buildCorrelationPoints(matches []domain.StatsMatchRow) []domain.CorrelationDataPair {
	points := make([]domain.CorrelationDataPair, 0, len(matches))
	for _, m := range matches {
		kd := 0.0
		if m.Deaths > 0 {
			kd = float64(m.Kills) / float64(m.Deaths)
		}
		points = append(points, domain.CorrelationDataPair{
			Label: "kills_vs_kd",
			X:     float64(m.Kills),
			Y:     math.Round(kd*100) / 100,
		})
	}
	return points
}
