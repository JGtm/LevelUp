"""Schémas Pydantic — Synthèse (Slice 7).

Contrats :
  POST /api/v1/players/{slug}/pages/synthesis  → SynthesisPageResponse
"""

from __future__ import annotations

from pydantic import BaseModel

from apps.api.app.schemas.filters import FilterContextInput

_PERIOD_CHOICES = ["all", "2y", "1y", "1m", "1w"]


class SynthesisKPIs(BaseModel):
    """KPIs d'un groupe (solo ou escouade) pour la comparaison Synthèse."""

    match_count: int = 0
    wins: int = 0
    kd_ratio: float | None = None
    win_rate: float = 0.0
    accuracy: float | None = None
    kills_per_min: float | None = None
    avg_life_seconds: float | None = None
    performance_score: float | None = None


class ComparisonMetricItem(BaseModel):
    """Une métrique comparée Solo vs Escouade."""

    label: str
    solo_value: float
    squad_value: float
    solo_text: str
    squad_text: str


class SynthesisQueryRequest(BaseModel):
    """Corps de la requête pour la page Synthèse."""

    period: str = "all"
    filters: FilterContextInput | None = None


class HeatmapCell(BaseModel):
    """Cellule de la heatmap temporelle (jour × heure)."""

    dow: int  # 0 = lundi … 6 = dimanche
    hour: int  # 0 – 23
    count: int


class TopWeekItem(BaseModel):
    """Résumé d'une semaine parmi les plus actives."""

    week_label: str  # ex: "2025-W03"
    match_count: int
    win_rate: float
    kd_ratio: float | None = None


class SynthesisPageResponse(BaseModel):
    """Réponse de la page Synthèse."""

    period: str
    total_matches: int
    solo_kpis: SynthesisKPIs
    squad_kpis: SynthesisKPIs
    comparison_metrics: list[ComparisonMetricItem]
    heatmap_data: list[HeatmapCell] = []
    top_weeks: list[TopWeekItem] = []
