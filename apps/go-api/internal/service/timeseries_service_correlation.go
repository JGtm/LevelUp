// Package service — timeseries_service_correlation.go : builder des paires de
// corrélation (onglet Distributions, scatter "X vs Y"). Extrait de
// timeseries_service_tabs.go (god-file split, V721-14a D-09 2026-07-25) —
// isolé dans son propre fichier pour garder timeseries_service_tabs.go sous
// le seuil de 500 L du CLAUDE.md après l'ajout de la télémétrie de repli.
package service

import (
	"context"
	"log/slog"
	"math"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// buildCorrelationPoints construit les paires de corrélation pour 5 types de scatter.
//
// Durée de vie (lifespan) : même helper que buildLifeBuckets (matchAvgLifeSeconds,
// B1 2026-07-25 ; D-09 V721-14a) — priorité à AvgLifeSeconds (valeur RÉELLE de
// l'API), repli sur le proxy time_played/(morts+1) sinon. Avant ce correctif, ce
// builder recalculait le proxy en dur : l'histogramme (buildLifeBuckets) et le
// nuage de points racontaient deux histoires différentes de la même métrique sur
// la même page. ctx ne sert qu'à la télémétrie du repli (Debug, jamais de
// dégradation muette).
func buildCorrelationPoints(ctx context.Context, matches []legacymatch.StatsMatchRow) []domain.CorrelationDataPair {
	points := make([]domain.CorrelationDataPair, 0, len(matches)*5)
	fallbacks, used := 0, 0
	for _, m := range matches {
		kd := 0.0
		if m.Deaths > 0 {
			kd = float64(m.Kills) / float64(m.Deaths)
		}
		lifespan, isReal, ok := matchAvgLifeSeconds(m)
		if ok {
			used++
			if !isReal {
				fallbacks++
			}
		}

		// P7.1 (revue 2026-04-29) : Label composite ("kills_vs_kd") séparé en
		// MetricXKey/MetricYKey ; X/Y → XValue/YValue.
		points = append(points, domain.CorrelationDataPair{
			MetricXKey: tsMetricKeyKills, MetricYKey: "kd_ratio",
			XValue: float64(m.Kills), YValue: math.Round(kd*100) / 100, Outcome: m.Outcome,
		})
		points = append(points, domain.CorrelationDataPair{
			MetricXKey: "lifespan", MetricYKey: tsMetricKeyKills,
			XValue: math.Round(lifespan*10) / 10, YValue: float64(m.Kills), Outcome: m.Outcome,
		})
		if m.Accuracy != nil && m.KDA != nil {
			// Accuracy déjà en % 0..100 (sync/transforms.go:315) — round à 1 décimale
			// sans re-multiplier (bug historique du *100 qui sortait du domaine).
			points = append(points, domain.CorrelationDataPair{
				MetricXKey: tsMetricKeyAccuracy, MetricYKey: "kda",
				XValue: math.Round(*m.Accuracy*10) / 10, YValue: math.Round(*m.KDA*100) / 100, Outcome: m.Outcome,
			})
		}
		points = append(points, domain.CorrelationDataPair{
			MetricXKey: "lifespan", MetricYKey: "deaths",
			XValue: math.Round(lifespan*10) / 10, YValue: float64(m.Deaths), Outcome: m.Outcome,
		})
		points = append(points, domain.CorrelationDataPair{
			MetricXKey: tsMetricKeyKills, MetricYKey: "deaths",
			XValue: float64(m.Kills), YValue: float64(m.Deaths), Outcome: m.Outcome,
		})
		if m.TeamMMR != nil && m.EnemyMMR != nil {
			points = append(points, domain.CorrelationDataPair{
				MetricXKey: "mmr_team", MetricYKey: "mmr_enemy",
				XValue: math.Round(*m.TeamMMR*100) / 100, YValue: math.Round(*m.EnemyMMR*100) / 100, Outcome: m.Outcome,
			})
		}
	}
	if fallbacks > 0 {
		slog.DebugContext(ctx, "timeseries: durée de vie estimée par repli (proxy temps_joué/(morts+1)) — nuage de corrélation",
			"matches_fallback", fallbacks, "matches_used", used, "matches_total", len(matches))
	}
	return points
}
