// Package domain — timeseries.go : types pour la page Timeseries (contrat FastAPI).
//
// Sprint 33 :
//
//	POST /api/v1/players/{slug}/pages/timeseries → TimeseriesPageResponse
//
// Architecture data-only : le Go envoie uniquement les data points bruts.
// Phase 3 P3.B + P3.F+G : le frontend consomme ces points via les wrappers
// ECharts (`HistogramChart`, `ScatterChart`, `Heatmap2DChart`,
// `TimeseriesLineChart`, `TimeseriesCombatYield`, `TimeseriesKdaBars`).
// Les anciens champs `*PlotlyFigurePayload` jamais populés ont été retirés
// en P3 cleanup.
package domain

import "time"

// ---------------------------------------------------------------------------
// Requête
// ---------------------------------------------------------------------------

// TimeseriesQueryRequest est le corps de POST /pages/timeseries.
type TimeseriesQueryRequest struct {
	Filters FilterContextInput `json:"filters"`
}

// ---------------------------------------------------------------------------
// Onglets
// ---------------------------------------------------------------------------

// TimeseriesKpiCard est une carte KPI dans l'onglet résumé.
type TimeseriesKpiCard struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Value string  `json:"value"`
	Delta *string `json:"delta"`
	Color *string `json:"color"`
}

// TimeseriesSummaryTab est l'onglet Résumé.
type TimeseriesSummaryTab struct {
	KpiCards []TimeseriesKpiCard `json:"kpi_cards"`
}

// TimeseriesCumulTab est l'onglet Cumul.
type TimeseriesCumulTab struct {
	CumulativeKD  []CumulativePoint `json:"cumulative_kd"`
	CumulativeNet []CumulativePoint `json:"cumulative_net"`
	RollingKD     []CumulativePoint `json:"rolling_kd"`
}

// TimeseriesRegressionStats contient les stats de régression.
type TimeseriesRegressionStats struct {
	KDSlope           *float64 `json:"kd_slope"`
	WinrateSlope      *float64 `json:"winrate_slope"`
	RSquared          *float64 `json:"r_squared"`
	HasEnoughForTrend bool     `json:"has_enough_for_trend"`
	Trend             *string  `json:"trend"` // "improving" | "declining" | "stable"
}

// TimeseriesFormTab est l'onglet Forme (EWMA, régression).
type TimeseriesFormTab struct {
	RegressionStats TimeseriesRegressionStats `json:"regression_stats"`
	EWMAKDPoints    []CumulativePoint         `json:"ewma_kd_points"`
}

// TimeseriesIntensityTab est l'onglet Intensité.
type TimeseriesIntensityTab struct {
	HeatmapData     []IntensityHeatmapPoint `json:"heatmap_data"`
	ScorePerMinData []CumulativePoint       `json:"score_per_min_data"`
}

// IntensityHeatmapPoint est un point de la heatmap jour×heure.
type IntensityHeatmapPoint struct {
	DayOfWeek int     `json:"day_of_week"` // 0=lundi, 6=dimanche
	Hour      int     `json:"hour"`        // 0-23
	Count     int     `json:"count"`
	AvgKD     float64 `json:"avg_kd"`
}

// TimeseriesDistributionsTab est l'onglet Distributions.
type TimeseriesDistributionsTab struct {
	KDABuckets         []DistributionBucket  `json:"kda_buckets"`
	KillsBuckets       []DistributionBucket  `json:"kills_buckets"`
	AccuracyBuckets    []DistributionBucket  `json:"accuracy_buckets"`
	ScorePerMinBuckets []DistributionBucket  `json:"score_per_min_buckets"`
	RollingWRBuckets   []DistributionBucket  `json:"rolling_wr_buckets"`
	CorrelationPoints  []CorrelationDataPair `json:"correlation_points"`
}

// DistributionBucket est un bucket pour un histogramme de distribution.
type DistributionBucket struct {
	BinStart float64 `json:"bin_start"`
	BinEnd   float64 `json:"bin_end"`
	Count    int     `json:"count"`
}

// CorrelationDataPair est une paire (x, y) pour un scatter plot de corrélation.
// Label exemples : "kills_vs_kd", "lifespan_vs_kills", "accuracy_vs_kda",
// "lifespan_vs_deaths", "kills_vs_deaths", "mmr_team_vs_enemy".
type CorrelationDataPair struct {
	Label   string  `json:"label"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Outcome *int    `json:"outcome"` // nil si inconnu ; 2=victoire, 3=défaite, 1=égalité
}

// TimeseriesMatchRow est une ligne de données par match pour les charts timeline.
// Fourni dans TimeseriesPageResponse.MatchRows pour permettre au frontend de
// construire les graphes K/D/A, assists, dégâts, perf score, etc.
type TimeseriesMatchRow struct {
	MatchID   string    `json:"match_id"`
	Index     int       `json:"index"`
	StartTime time.Time `json:"start_time"`
	Kills     int       `json:"kills"`
	Deaths    int       `json:"deaths"`
	Assists   int       `json:"assists"`
	// KDA et KDRatio exposés par P2.5 (revue 2026-04-29 ADR 0006).
	// Ils débloquent la suppression du recompute K/D côté front
	// (TimeseriesKdaBars.tsx:78 — voir B3).
	KDA               *float64 `json:"kda,omitempty"`
	KDRatio           *float64 `json:"kd_ratio,omitempty"`
	Accuracy          *float64 `json:"accuracy"`
	Outcome           *int     `json:"outcome"`
	PersonalScore     *int     `json:"personal_score"`
	DamageDealt       *float64 `json:"damage_dealt"`
	DamageTaken       *float64 `json:"damage_taken"`
	PerfScore         *float64 `json:"perf_score"`
	Rank              *int     `json:"rank"`
	PlaylistName      string   `json:"playlist_name"`
	TimePlayedSeconds *int     `json:"time_played_seconds"`
}

// ---------------------------------------------------------------------------
// Réponse
// ---------------------------------------------------------------------------

// TimeseriesPageResponse est la réponse de POST /pages/timeseries.
type TimeseriesPageResponse struct {
	TotalMatches     int                        `json:"total_matches"`
	MatchRows        []TimeseriesMatchRow       `json:"match_rows"`
	SummaryTab       TimeseriesSummaryTab       `json:"summary_tab"`
	CumulTab         TimeseriesCumulTab         `json:"cumul_tab"`
	FormTab          TimeseriesFormTab          `json:"form_tab"`
	IntensityTab     TimeseriesIntensityTab     `json:"intensity_tab"`
	DistributionsTab TimeseriesDistributionsTab `json:"distributions_tab"`
}
