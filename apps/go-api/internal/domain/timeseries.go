// Package domain — timeseries.go : types pour la page Timeseries (contrat FastAPI).
//
// Sprint 33 :
//
//	POST /api/v1/players/{slug}/pages/timeseries → TimeseriesPageResponse
//
// Décision architecturale Plotly : le Go envoie les mêmes data points bruts
// que le endpoint stats/query existant (pas de Plotly server-side).
// Le frontend consomme PlotlyFigurePayload = null et construit les charts localement.
// Les onglets sont néanmoins structurés identiquement au contrat Python.
package domain

// ---------------------------------------------------------------------------
// Types communs
// ---------------------------------------------------------------------------

// PlotlyFigurePayload est un payload Plotly opaque (data+layout JSON).
// Le Go n'en génère pas server-side — ce champ est nil/nul dans les réponses.
// Le frontend construit les charts à partir des data points bruts.
type PlotlyFigurePayload struct {
	Data      interface{} `json:"data"`
	Layout    interface{} `json:"layout"`
	ConfigKey *string     `json:"config_key,omitempty"`
}

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
	KpiCards     []TimeseriesKpiCard  `json:"kpi_cards"`
	WinRateChart *PlotlyFigurePayload `json:"win_rate_chart"`
	ScoreChart   *PlotlyFigurePayload `json:"score_chart"`
	KDADistChart *PlotlyFigurePayload `json:"kda_dist_chart"`
}

// TimeseriesCumulTab est l'onglet Cumul.
type TimeseriesCumulTab struct {
	CumulNetChart  *PlotlyFigurePayload `json:"cumul_net_chart"`
	CumulKDChart   *PlotlyFigurePayload `json:"cumul_kd_chart"`
	RollingKDChart *PlotlyFigurePayload `json:"rolling_kd_chart"`
	// Data points bruts pour le frontend (Sprint 42).
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
	EWMAKDChart          *PlotlyFigurePayload      `json:"ewma_kd_chart"`
	RegressionChart      *PlotlyFigurePayload      `json:"regression_chart"`
	NetScorePerHourChart *PlotlyFigurePayload      `json:"net_score_per_hour_chart"`
	RegressionStats      TimeseriesRegressionStats `json:"regression_stats"`
	// Data points bruts pour le frontend (Sprint 42).
	EWMAKDPoints []CumulativePoint `json:"ewma_kd_points"`
}

// TimeseriesIntensityTab est l'onglet Intensité.
type TimeseriesIntensityTab struct {
	IntensityHeatmap    *PlotlyFigurePayload `json:"intensity_heatmap"`
	ScorePerMinuteChart *PlotlyFigurePayload `json:"score_per_minute_chart"`
	// Data points bruts pour le frontend (Sprint 42).
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
	KDADistribution *PlotlyFigurePayload  `json:"kda_distribution"`
	FirstKillDist   *PlotlyFigurePayload  `json:"first_kill_dist"`
	Correlations    []PlotlyFigurePayload `json:"correlations"`
	// Data points bruts pour le frontend (Sprint 42).
	KDABuckets        []DistributionBucket  `json:"kda_buckets"`
	KillsBuckets      []DistributionBucket  `json:"kills_buckets"`
	CorrelationPoints []CorrelationDataPair `json:"correlation_points"`
}

// DistributionBucket est un bucket pour un histogramme de distribution.
type DistributionBucket struct {
	BinStart float64 `json:"bin_start"`
	BinEnd   float64 `json:"bin_end"`
	Count    int     `json:"count"`
}

// CorrelationDataPair est une paire (x, y) pour un scatter plot de corrélation.
type CorrelationDataPair struct {
	Label string  `json:"label"` // ex: "kills_vs_kd"
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
}

// ---------------------------------------------------------------------------
// Réponse
// ---------------------------------------------------------------------------

// TimeseriesPageResponse est la réponse de POST /pages/timeseries.
type TimeseriesPageResponse struct {
	TotalMatches     int                        `json:"total_matches"`
	SummaryTab       TimeseriesSummaryTab       `json:"summary_tab"`
	CumulTab         TimeseriesCumulTab         `json:"cumul_tab"`
	FormTab          TimeseriesFormTab          `json:"form_tab"`
	IntensityTab     TimeseriesIntensityTab     `json:"intensity_tab"`
	DistributionsTab TimeseriesDistributionsTab `json:"distributions_tab"`
}
