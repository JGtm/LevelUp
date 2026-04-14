"""Schémas Pydantic pour les endpoints Carrière (Slice 2)."""

from __future__ import annotations

from datetime import date, datetime
from typing import Any

from pydantic import BaseModel

# ---------------------------------------------------------------------------
# Sous-schemas partagés
# ---------------------------------------------------------------------------


class PlotlyFigurePayload(BaseModel):
    """Payload JSON d'un graphe Plotly (data + layout)."""

    data: list[dict[str, Any]]
    layout: dict[str, Any]


# ---------------------------------------------------------------------------
# CareerPageResponse
# ---------------------------------------------------------------------------


class CareerSummary(BaseModel):
    """Résumé du rang carrière actuel."""

    rank_number: int
    rank_label: str
    rank_name_raw: str
    rank_tier: str
    current_xp: int
    xp_for_next_rank: int
    xp_total: int
    progress_pct: float
    is_max_rank: bool
    recorded_at: datetime | None


class HeroProgress(BaseModel):
    """Progression globale vers le rang Héros (rang 272)."""

    xp_total_required: int
    xp_remaining: int
    percentage: float
    current_rank: int


class CareerProjections(BaseModel):
    """Projections de progression XP."""

    xp_per_day_active: float
    xp_per_day_fallback: float
    estimated_hero_date: date | None
    estimated_rank_cap_date: date | None


class CareerCharts(BaseModel):
    """Graphes Plotly serialisés."""

    rank_progress_gauge: PlotlyFigurePayload | None = None
    hero_progress_gauge: PlotlyFigurePayload | None = None
    xp_history_figure: PlotlyFigurePayload | None = None
    lusr_rating_figure: PlotlyFigurePayload | None = None


class CareerHistoryPoint(BaseModel):
    """Point d'historique XP."""

    recorded_at: datetime
    rank: int
    xp_total: int


class CareerLusrCheckpoint(BaseModel):
    """Point de checkpoint LUSR/CSR."""

    recorded_at: datetime
    rating_value: float
    playlist_group: str


class CareerLusrSection(BaseModel):
    """Section LUSR (rating compétitif)."""

    current_rating: float | None
    current_tier_label: str | None
    current_playlist_group: str | None
    trend_label: str | None
    checkpoints: list[CareerLusrCheckpoint]


# ---------------------------------------------------------------------------
# Top Matches / Encounters (partagés entre page et endpoints dédiés)
# ---------------------------------------------------------------------------


class CareerTopMatch(BaseModel):
    """Informations d'un top match (meilleur ou pire)."""

    match_id: str
    start_time: datetime | None
    map_ui: str | None
    mode_ui: str | None
    playlist_label: str | None
    performance_score: float | None
    badge_type: str | None
    score_label: str | None
    outcome_label: str | None
    kills: int | None = None
    deaths: int | None = None
    assists: int | None = None
    kd_ratio: float | None = None
    variant: str | None = None  # "best" | "worst"


class CareerEncounter(BaseModel):
    """Résumé des rencontres avec un adversaire."""

    encounter_key: str
    opponent_gamertag: str
    count_matches: int
    wins: int
    losses: int
    last_seen_at: datetime | None


# ---------------------------------------------------------------------------
# Réponses
# ---------------------------------------------------------------------------


class CareerPageResponse(BaseModel):
    """Réponse complète pour la page Carrière."""

    summary: CareerSummary | None
    hero_progress: HeroProgress | None
    projections: CareerProjections | None
    charts: CareerCharts
    xp_history: list[CareerHistoryPoint]
    lusr: CareerLusrSection | None
    top_matches_preview: list[CareerTopMatch]
    encounters_preview: list[CareerEncounter]


class CareerTopMatchesResponse(BaseModel):
    """Réponse pour l'endpoint top matches."""

    items: list[CareerTopMatch]


class CareerEncountersResponse(BaseModel):
    """Réponse pour l'endpoint encounters."""

    items: list[CareerEncounter]
