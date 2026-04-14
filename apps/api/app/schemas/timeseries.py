"""Schémas Pydantic — Séries temporelles (Slice 3 Phase B) + Comparaison sessions (Phase C).

Contrats :
  POST /api/v1/players/{slug}/pages/timeseries         → TimeseriesPageResponse
  POST /api/v1/players/{slug}/pages/session-compare    → SessionCompareResponse
"""

from __future__ import annotations

from pydantic import BaseModel

from apps.api.app.schemas.career import PlotlyFigurePayload
from apps.api.app.schemas.filters import FilterContextInput

# ---------------------------------------------------------------------------
# Timeseries — Slice 3 Phase B
# ---------------------------------------------------------------------------


class TimeseriesQueryRequest(BaseModel):
    """Requête pour la page Séries temporelles."""

    filters: FilterContextInput


class TimeseriesKpiCard(BaseModel):
    """KPI card affichée dans l'onglet résumé."""

    key: str
    label: str
    value: str
    delta: str | None = None
    color: str | None = None


class TimeseriesSummaryTab(BaseModel):
    """Onglet KPIs / Résumé."""

    kpi_cards: list[TimeseriesKpiCard]
    win_rate_chart: PlotlyFigurePayload | None = None
    score_chart: PlotlyFigurePayload | None = None
    kda_dist_chart: PlotlyFigurePayload | None = None


class TimeseriesCumulTab(BaseModel):
    """Onglet Cumul (net score cumulé, K/D cumulé, rolling)."""

    cumul_net_chart: PlotlyFigurePayload | None = None
    cumul_kd_chart: PlotlyFigurePayload | None = None
    rolling_kd_chart: PlotlyFigurePayload | None = None


class TimeseriesRegressionStats(BaseModel):
    """Statistiques de régression K/D exposées pour les delta cards D1."""

    kd_slope: float | None = None
    winrate_slope: float | None = None
    r_squared: float | None = None
    has_enough_for_trend: bool = False
    trend: str | None = None  # "improving" | "declining" | "stable"


class TimeseriesFormTab(BaseModel):
    """Onglet Forme récente (EWMA, régression, streaks)."""

    ewma_kd_chart: PlotlyFigurePayload | None = None
    regression_chart: PlotlyFigurePayload | None = None
    net_score_per_hour_chart: PlotlyFigurePayload | None = None
    regression_stats: TimeseriesRegressionStats = TimeseriesRegressionStats()


class TimeseriesIntensityTab(BaseModel):
    """Onglet Intensité (heatmap, score par minute)."""

    intensity_heatmap: PlotlyFigurePayload | None = None
    score_per_minute_chart: PlotlyFigurePayload | None = None


class TimeseriesDistributionsTab(BaseModel):
    """Onglet Distributions (KDA dist, first kill dist, corrélations)."""

    kda_distribution: PlotlyFigurePayload | None = None
    first_kill_dist: PlotlyFigurePayload | None = None
    correlations: list[PlotlyFigurePayload]


class TimeseriesPageResponse(BaseModel):
    """Réponse complète pour la page Séries temporelles."""

    total_matches: int
    summary_tab: TimeseriesSummaryTab
    cumul_tab: TimeseriesCumulTab
    form_tab: TimeseriesFormTab
    intensity_tab: TimeseriesIntensityTab
    distributions_tab: TimeseriesDistributionsTab


# ---------------------------------------------------------------------------
# Session Compare — Slice 3 Phase C
# ---------------------------------------------------------------------------


class SessionCompareRequest(BaseModel):
    """Requête pour la comparaison de sessions A/B."""

    filters: FilterContextInput
    session_a: str | None = None
    session_b: str | None = None


class SessionCompareEntry(BaseModel):
    """Résumé d'une session sélectionnée (A ou B)."""

    session_label: str
    start_time: str | None = None
    end_time: str | None = None
    total_matches: int
    wins: int
    losses: int
    kda: float | None = None
    performance_score: float | None = None
    with_friends: bool = False
    dominant_category: str | None = None


class SessionCompareMetricRow(BaseModel):
    """Ligne d'une comparaison metric-à-metric entre session A et B."""

    key: str
    label: str
    value_a: str
    value_b: str
    delta: str | None = None
    winner: str | None = None  # "a" | "b" | "tie" | null


class SessionCompareResponse(BaseModel):
    """Réponse complète pour la comparaison de sessions."""

    session_a: SessionCompareEntry | None = None
    session_b: SessionCompareEntry | None = None
    available_sessions: list[str]
    metrics: list[SessionCompareMetricRow]
    radar_chart: PlotlyFigurePayload | None = None
    kd_progression_chart: PlotlyFigurePayload | None = None
    outcomes_chart: PlotlyFigurePayload | None = None
    maps_table: list[dict] = []
    modes_table: list[dict] = []
