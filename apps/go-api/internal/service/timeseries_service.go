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
		SummaryTab:       buildSummaryTab(matches),
		CumulTab:         buildCumulTab(matches),
		FormTab:          buildTimeseriesFormTab(matches),
		IntensityTab:     domain.TimeseriesIntensityTab{},
		DistributionsTab: domain.TimeseriesDistributionsTab{Correlations: []domain.PlotlyFigurePayload{}},
	}

	return resp, nil
}

// ---------------------------------------------------------------------------
// Onglet Summary
// ---------------------------------------------------------------------------

func buildSummaryTab(matches []domain.StatsMatchRow) domain.TimeseriesSummaryTab {
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

func buildCumulTab(_ []domain.StatsMatchRow) domain.TimeseriesCumulTab {
	// Les charts Plotly sont construits côté frontend à partir de stats/query.
	// Le Go ne génère pas de PlotlyFigurePayload server-side.
	return domain.TimeseriesCumulTab{}
}

// ---------------------------------------------------------------------------
// Onglet Form
// ---------------------------------------------------------------------------

func buildTimeseriesFormTab(matches []domain.StatsMatchRow) domain.TimeseriesFormTab {
	regStats := computeRegressionStats(matches)
	return domain.TimeseriesFormTab{
		RegressionStats: regStats,
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
