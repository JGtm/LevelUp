// Package service — squad_service_v2_trio_charts.go : 5 timeseries multi-
// joueurs de l'onglet Contributions Squad V2 (cf. PLAN_SQUAD_GO_PORTAGE
// Phase P7, sections 17 audit).
//
//	BuildAssistsChart      : 1 trace par joueur, Y = assists par match.
//	BuildKDAChart          : 1 trace par joueur, Y = KDA par match.
//	BuildAccuracyChart     : 1 trace par joueur, Y = accuracy [0..1] par match.
//	BuildAvgLifeChart      : 1 trace par joueur, Y = avg_life_seconds par match.
//	BuildPerformanceChart  : 1 trace par joueur, Y = performance_score par match.
//
// Toutes les series sont triees chronologiquement ASC. Les rows sans valeur
// sont skippees (pas de point sur la trace) sans biaiser le voisin.
package service

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// trioExtractor extrait une valeur scalaire optionnelle d'un PlayerMatchRow.
// Retourne (valeur, true) si la valeur est dispo, (_, false) sinon.
type trioExtractor func(canonical.PlayerMatchRow) (float64, bool)

// buildTrioTimeseries factorise la construction d'une serie multi-joueurs
// type "1 trace par joueur, X = StartedAtUTC, Y = scalar". Les 5 charts
// trio partagent ce squelette et ne different que par l'extracteur.
func buildTrioTimeseries(
	rowsByPlayer map[string][]canonical.PlayerMatchRow,
	keyPrefix, labelKey string,
	extract trioExtractor,
) []domain.ChartSeries[domain.ChartPoint2D] {
	if len(rowsByPlayer) == 0 {
		return nil
	}
	gts := sortedGamertags(rowsByPlayer)

	out := make([]domain.ChartSeries[domain.ChartPoint2D], 0, len(gts))
	for _, gt := range gts {
		sorted := sortedByStartedAt(rowsByPlayer[gt])
		dps := make([]domain.ChartPoint2D, 0, len(sorted))
		for _, r := range sorted {
			v, ok := extract(r)
			if !ok {
				continue
			}
			dps = append(dps, domain.ChartPoint2D{
				X: r.Summary.StartedAtUTC,
				Y: v,
			})
		}
		if len(dps) == 0 {
			continue
		}
		out = append(out, domain.ChartSeries[domain.ChartPoint2D]{
			Key:        keyPrefix + "." + gt,
			LabelKey:   labelKey,
			Datapoints: dps,
			Meta:       map[string]any{chartMetaGamertag: gt},
		})
	}
	return out
}

// BuildAssistsChart : 1 trace par joueur, Y = assists par match.
func BuildAssistsChart(rowsByPlayer map[string][]canonical.PlayerMatchRow) []domain.ChartSeries[domain.ChartPoint2D] {
	return buildTrioTimeseries(
		rowsByPlayer,
		"squad.contrib.assists",
		"squad.contrib.assists_title",
		func(r canonical.PlayerMatchRow) (float64, bool) {
			if r.Self.Assists == nil {
				return 0, false
			}
			return float64(*r.Self.Assists), true
		},
	)
}

// BuildKDAChart : 1 trace par joueur, Y = KDA par match (champ etendu
// scoreboard, peut etre nil si non charge).
func BuildKDAChart(rowsByPlayer map[string][]canonical.PlayerMatchRow) []domain.ChartSeries[domain.ChartPoint2D] {
	return buildTrioTimeseries(
		rowsByPlayer,
		"squad.contrib.kda",
		"squad.contrib.kda_title",
		func(r canonical.PlayerMatchRow) (float64, bool) {
			if r.Self.KDA == nil {
				return 0, false
			}
			return *r.Self.KDA, true
		},
	)
}

// BuildAccuracyChart : 1 trace par joueur, Y = accuracy [0..1] par match.
func BuildAccuracyChart(rowsByPlayer map[string][]canonical.PlayerMatchRow) []domain.ChartSeries[domain.ChartPoint2D] {
	return buildTrioTimeseries(
		rowsByPlayer,
		"squad.contrib.accuracy",
		"squad.contrib.accuracy_title",
		func(r canonical.PlayerMatchRow) (float64, bool) {
			if r.Self.Accuracy == nil {
				return 0, false
			}
			return *r.Self.Accuracy, true
		},
	)
}

// BuildAvgLifeChart : 1 trace par joueur, Y = avg_life_seconds par match.
func BuildAvgLifeChart(rowsByPlayer map[string][]canonical.PlayerMatchRow) []domain.ChartSeries[domain.ChartPoint2D] {
	return buildTrioTimeseries(
		rowsByPlayer,
		"squad.contrib.avg_life",
		"squad.contrib.avg_life_title",
		func(r canonical.PlayerMatchRow) (float64, bool) {
			if r.Self.AvgLifeSeconds == nil {
				return 0, false
			}
			return *r.Self.AvgLifeSeconds, true
		},
	)
}

// BuildPerformanceChart : 1 trace par joueur, Y = performance_score par
// match (enrichment LevelUp-specific).
func BuildPerformanceChart(rowsByPlayer map[string][]canonical.PlayerMatchRow) []domain.ChartSeries[domain.ChartPoint2D] {
	return buildTrioTimeseries(
		rowsByPlayer,
		"squad.contrib.performance",
		"squad.contrib.performance_title",
		func(r canonical.PlayerMatchRow) (float64, bool) {
			if r.Enrichment.PerformanceScore == nil {
				return 0, false
			}
			return *r.Enrichment.PerformanceScore, true
		},
	)
}
