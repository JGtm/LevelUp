"""Schémas Pydantic — Explorer (Slice 4).

Contrats :
  GET  /api/v1/directory/gamertags/search                             → GamertagSearchResponse
  POST /api/v1/players/{slug}/pages/explorer/matches-query            → ExplorerMatchesQueryResponse
  POST /api/v1/players/{slug}/pages/explorer/player-query             → ExplorerPlayerQueryResponse
"""

from __future__ import annotations

from datetime import date, datetime

from pydantic import BaseModel, Field

from apps.api.app.schemas.common import PaginatedResponse, PaginationRequest
from apps.api.app.schemas.filters import FilterContextInput

# ---------------------------------------------------------------------------
# Recherche gamertags
# ---------------------------------------------------------------------------


class GamertagSuggestion(BaseModel):
    """Une suggestion de gamertag dans les résultats de recherche."""

    gamertag: str
    xuid: str | None = None
    score: float = 1.0
    exact_match: bool = False


class GamertagSearchResponse(BaseModel):
    """Réponse de la recherche de gamertags."""

    query: str
    items: list[GamertagSuggestion]


# ---------------------------------------------------------------------------
# Lignes des tableaux Explorer
# ---------------------------------------------------------------------------


class ExplorerMatchRow(BaseModel):
    """Une ligne dans le tableau de résultats de l'Explorer."""

    match_id: str
    start_time: datetime
    start_time_label: str
    map_ui: str
    mode_ui: str
    playlist_label: str
    outcome_label: str
    score_label: str
    is_with_friends: bool = False
    experience_type_label: str = "Non classé"


class ExplorerEncounterRow(BaseModel):
    """Résumé d'un encounter avec un joueur (alliés ou adversaires)."""

    gamertag: str
    xuid: str | None = None
    count_matches: int
    wins: int
    losses: int
    last_seen_at: datetime | None = None
    same_team: bool | None = None


# ---------------------------------------------------------------------------
# Résumés
# ---------------------------------------------------------------------------


class ExplorerMatchesQuerySummary(BaseModel):
    """Compteurs de la requête matches."""

    total_matches: int
    selected_match_id: str | None = None


class ExplorerPlayerTarget(BaseModel):
    """Identité du joueur cible de la recherche."""

    gamertag: str
    xuid: str | None = None


class ExplorerPlayerSummary(BaseModel):
    """Bilan global de l'encounter avec le joueur cible."""

    matches_together: int
    wins_together: int
    losses_together: int
    last_seen_at: datetime | None = None


# ---------------------------------------------------------------------------
# Filtres locaux de la vue Explorer
# ---------------------------------------------------------------------------


class ExplorerMatchFilters(BaseModel):
    """Filtres locaux de la vue Explorer (non liés aux filtres globaux)."""

    selected_date: date | None = None
    squad_scope: str = "all"  # "all" | "solo" | "squad"
    experience_type: str | None = None
    playlist: str | None = None
    mode: str | None = None
    map: str | None = None
    selected_match_id: str | None = None


# ---------------------------------------------------------------------------
# Requêtes
# ---------------------------------------------------------------------------


class ExplorerMatchesQueryRequest(BaseModel):
    """Corps de requête pour POST explorer/matches-query."""

    filters: FilterContextInput = Field(default_factory=FilterContextInput)
    match_filters: ExplorerMatchFilters = Field(default_factory=ExplorerMatchFilters)
    pagination: PaginationRequest = Field(default_factory=PaginationRequest)


class ExplorerPlayerQueryRequest(BaseModel):
    """Corps de requête pour POST explorer/player-query."""

    target_gamertag: str
    filters: FilterContextInput | None = None


# ---------------------------------------------------------------------------
# Réponses
# ---------------------------------------------------------------------------


class ExplorerMatchesQueryResponse(BaseModel):
    """Réponse de l'endpoint explorer/matches-query."""

    summary: ExplorerMatchesQuerySummary
    table: PaginatedResponse[ExplorerMatchRow]


class ExplorerPlayerQueryResponse(BaseModel):
    """Réponse de l'endpoint explorer/player-query."""

    target: ExplorerPlayerTarget
    summary: ExplorerPlayerSummary
    allies_table: list[ExplorerEncounterRow]
    enemies_table: list[ExplorerEncounterRow]
    common_matches: list[ExplorerMatchRow]
