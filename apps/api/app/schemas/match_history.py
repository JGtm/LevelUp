"""Schémas Pydantic — Historique des parties (Slice 3).

Contrats :
  POST /players/{slug}/pages/match-history/query  → MatchHistoryPageResponse
  POST /players/{slug}/pages/match-history/export → FileTokenResponse
"""

from __future__ import annotations

from datetime import datetime

from pydantic import BaseModel, Field

from apps.api.app.schemas.common import PaginatedResponse, PaginationRequest, SortSpec
from apps.api.app.schemas.filters import FilterContextInput

# ---------------------------------------------------------------------------
# Lignes du tableau
# ---------------------------------------------------------------------------


class MatchHistoryRow(BaseModel):
    """Une ligne dans le tableau de l'historique des parties."""

    match_id: str
    start_time: datetime
    start_time_label: str
    outcome_code: int | None = None
    outcome_label: str
    score_label: str
    map_ui: str
    mode_ui: str
    playlist_label: str
    team_mmr: float | None = None
    enemy_mmr: float | None = None
    delta_mmr: float | None = None
    win_rate_hist: float | None = None
    win_rate_hist_total: int | None = None
    performance_score_relative: int | None = None
    average_life_mmss: str
    match_url: str


# ---------------------------------------------------------------------------
# Résumé de la requête
# ---------------------------------------------------------------------------


class MatchHistoryQuerySummary(BaseModel):
    """Compteurs affichés en tête du tableau historique."""

    total_matches_scoped: int
    total_matches_unfiltered: int
    period_label: str | None = None
    active_filter_mode: str


# ---------------------------------------------------------------------------
# Hint d'export
# ---------------------------------------------------------------------------


class ExportHint(BaseModel):
    """Informations pour déclencher un export côté client."""

    file_name: str
    estimated_rows: int
    token: str | None = None


# ---------------------------------------------------------------------------
# Réponse de requête
# ---------------------------------------------------------------------------


class MatchHistoryPageResponse(BaseModel):
    """Réponse complète de l'endpoint match-history/query."""

    summary: MatchHistoryQuerySummary
    table: PaginatedResponse[MatchHistoryRow]
    available_sort_fields: list[str] = Field(
        default_factory=lambda: [
            "start_time",
            "outcome_code",
            "performance_score_relative",
            "team_mmr",
            "delta_mmr",
            "win_rate_hist",
        ]
    )
    export_hint: ExportHint | None = None


# ---------------------------------------------------------------------------
# Requête de consultation
# ---------------------------------------------------------------------------


class MatchHistoryQueryRequest(BaseModel):
    """Corps de requête pour POST match-history/query."""

    filters: FilterContextInput = Field(default_factory=FilterContextInput)
    pagination: PaginationRequest = Field(default_factory=PaginationRequest)
    columns: list[str] | None = None
    include_export_hint: bool = False


# ---------------------------------------------------------------------------
# Export CSV
# ---------------------------------------------------------------------------


class MatchHistoryExportRequest(BaseModel):
    """Corps de requête pour POST match-history/export."""

    filters: FilterContextInput = Field(default_factory=FilterContextInput)
    sort: list[SortSpec] | None = None
    columns: list[str] | None = None
    format: str = "csv"


class FileTokenResponse(BaseModel):
    """Jeton renvoyé après la génération d'un fichier d'export."""

    file_token: str
    file_name: str
    content_type: str
    download_path: str
    expires_at: datetime
    estimated_rows: int | None = None
