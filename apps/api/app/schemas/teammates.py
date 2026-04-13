"""Schémas Pydantic — Escouade / Coéquipiers (Slice 6).

Contrats :
  POST /api/v1/players/{slug}/pages/teammates  → TeammatesPageResponse
"""

from __future__ import annotations

from datetime import datetime

from pydantic import BaseModel

from apps.api.app.schemas.filters import FilterContextInput

# ---------------------------------------------------------------------------
# Options de sélection de coéquipier
# ---------------------------------------------------------------------------


class TeammateOption(BaseModel):
    """Un coéquipier fréquent sélectionnable dans l'interface."""

    gamertag: str
    xuid: str | None = None
    encounter_count: int
    last_seen_at: datetime | None = None


# ---------------------------------------------------------------------------
# KPIs d'un groupe
# ---------------------------------------------------------------------------


class TeammateKPIs(BaseModel):
    """KPIs agrégés pour un groupe de matchs (avec/sans coéquipier)."""

    match_count: int = 0
    wins: int = 0
    kd_ratio: float | None = None
    win_rate: float = 0.0
    accuracy: float | None = None
    kills_per_game: float | None = None
    assists_per_game: float | None = None


# ---------------------------------------------------------------------------
# Comparaison avec / sans coéquipier
# ---------------------------------------------------------------------------


class TeammateRow(BaseModel):
    """Ligne de résultat pour un coéquipier : stats avec vs sans."""

    gamertag: str
    xuid: str | None = None
    encounter_count: int
    last_seen_at: datetime | None = None
    with_kpis: TeammateKPIs
    without_kpis: TeammateKPIs | None = None


# ---------------------------------------------------------------------------
# Requête et réponse
# ---------------------------------------------------------------------------


class TeammatesQueryRequest(BaseModel):
    """Corps de la requête pour la page Escouade."""

    selected_gamertags: list[str] = []
    filters: FilterContextInput | None = None


class TeammatesPageResponse(BaseModel):
    """Réponse de la page Escouade."""

    options: list[TeammateOption]
    teammates: list[TeammateRow]
    solo_reference: TeammateKPIs | None = None
    total_matches: int = 0
