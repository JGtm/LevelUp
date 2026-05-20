// Package service — squad_service_v2_timeline.go : helpers timeline + form score
// pour l'onglet Synergies de la page Squad V2 (cf. PLAN_SQUAD_GO_PORTAGE Phase P4).
//
//	Timeline performance multi-joueurs : 1 trace par joueur, X = chronologie,
//	                                     Y = performance_score, marker = outcome.
//	Form Score lisse (LOWESS)          : 1 trace, lissage de la perf du main.
package service

import (
	"sort"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// BuildTimelineMultiPlayer construit une serie par joueur avec
// X = StartedAtUTC, Y = performance_score. Les rows sans perf score sont
// skippees. Tri chronologique ASC pour permettre le rendu en line chart.
//
// Le marker outcome (W/L/T/DNF) est embarque dans le Label de chaque
// datapoint (ex. "win"). Le wrapper TimeseriesLine cote front choisit le
// symbol selon le label (cercle vert pour win, croix rouge pour loss, etc.).
func BuildTimelineMultiPlayer(rowsByPlayer map[string][]canonical.PlayerMatchRow) []domain.ChartSeries[domain.ChartPoint2D] {
	if len(rowsByPlayer) == 0 {
		return nil
	}
	gts := make([]string, 0, len(rowsByPlayer))
	for gt := range rowsByPlayer {
		gts = append(gts, gt)
	}
	sort.Strings(gts) // ordre stable

	out := make([]domain.ChartSeries[domain.ChartPoint2D], 0, len(gts))
	for _, gt := range gts {
		rows := rowsByPlayer[gt]
		// Tri chronologique ASC (sur StartedAtUTC).
		sorted := make([]canonical.PlayerMatchRow, len(rows))
		copy(sorted, rows)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Summary.StartedAtUTC.Before(sorted[j].Summary.StartedAtUTC)
		})

		dps := make([]domain.ChartPoint2D, 0, len(sorted))
		for _, r := range sorted {
			if r.Enrichment.PerformanceScore == nil {
				continue
			}
			outcome := string(r.Self.Outcome)
			dps = append(dps, domain.ChartPoint2D{
				X:     r.Summary.StartedAtUTC,
				Y:     *r.Enrichment.PerformanceScore,
				Label: &outcome,
			})
		}
		if len(dps) == 0 {
			continue
		}
		out = append(out, domain.ChartSeries[domain.ChartPoint2D]{
			Key:        "squad.synergies.timeline_perf." + gt,
			LabelKey:   "squad.synergies.timeline_player",
			Datapoints: dps,
			Meta:       map[string]any{chartMetaGamertag: gt},
		})
	}
	return out
}

// BuildFormScore lisse la performance_score du joueur principal via LOWESS.
//
// alpha : fraction du dataset utilisee comme fenetre de lissage. Default 0.3.
// Renvoie une serie unique avec X = StartedAtUTC, Y = perf_smoothed.
//
// Les rows sans perf score sont exclues du lissage (ne contribuent pas a la
// moyenne, ne produisent pas de point dans la serie).
func BuildFormScore(mainRows []canonical.PlayerMatchRow, alpha float64) domain.ChartSeries[domain.ChartPoint2D] {
	if alpha <= 0 {
		alpha = 0.3
	}
	// Tri chronologique + extraction perf score.
	sorted := make([]canonical.PlayerMatchRow, len(mainRows))
	copy(sorted, mainRows)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Summary.StartedAtUTC.Before(sorted[j].Summary.StartedAtUTC)
	})

	var validRows []canonical.PlayerMatchRow
	var values []float64
	for _, r := range sorted {
		if r.Enrichment.PerformanceScore == nil {
			continue
		}
		validRows = append(validRows, r)
		values = append(values, *r.Enrichment.PerformanceScore)
	}
	if len(values) == 0 {
		return domain.ChartSeries[domain.ChartPoint2D]{
			Key:      "squad.synergies.form_score",
			LabelKey: "squad.synergies.form_score_title",
		}
	}

	smoothed := temporal.LowessSmooth(values, alpha)
	dps := make([]domain.ChartPoint2D, 0, len(smoothed))
	for i, v := range smoothed {
		if isNaN(v) {
			continue
		}
		ts := validRows[i].Summary.StartedAtUTC
		dps = append(dps, domain.ChartPoint2D{
			X: ts,
			Y: v,
		})
	}
	return domain.ChartSeries[domain.ChartPoint2D]{
		Key:        "squad.synergies.form_score",
		LabelKey:   "squad.synergies.form_score_title",
		Datapoints: dps,
		Meta: map[string]any{
			"alpha":          alpha,
			"raw_count":      len(values),
			"smoothed_count": len(dps),
		},
	}
}

// isNaN evite l'import math juste pour ce test (Go inline parfait via float64).
func isNaN(v float64) bool {
	return v != v
}

// _ ensure time package usage in timeline (helper reserved for explicit
// timestamp formatting if needed in tests).
var _ = time.RFC3339
