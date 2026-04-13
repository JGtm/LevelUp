"""Schémas Pydantic — Citations (Slice 2 Phase B).

Contrat :
  POST /api/v1/players/{slug}/pages/citations   → CitationsPageResponse
"""

from __future__ import annotations

from pydantic import BaseModel

from apps.api.app.schemas.career import PlotlyFigurePayload
from apps.api.app.schemas.filters import FilterContextInput


class CitationsQueryRequest(BaseModel):
    """Requête pour la page Citations."""

    filters: FilterContextInput


class CommendationSummary(BaseModel):
    """Résumé d'une commendation (Halo 5 : Guardians)."""

    key: str
    label: str
    category: str | None = None
    current_value: int
    color: str | None = None
    icon_path: str | None = None
    tier_label: str | None = None
    mastery_pct: float | None = None


class MedalSummary(BaseModel):
    """Résumé d'une médaille (Halo Infinite)."""

    medal_name_id: int
    name: str
    count_filtered: int
    count_total: int
    description: str | None = None


class CitationsDeltas(BaseModel):
    """Compteurs globaux filtrés vs total."""

    filtered_total: int
    unfiltered_total: int
    delta_count: int


class CitationsPageResponse(BaseModel):
    """Réponse complète pour la page Citations."""

    commendations: list[CommendationSummary]
    medals_summary: list[MedalSummary]
    deltas: CitationsDeltas
    distribution_chart: PlotlyFigurePayload | None = None
