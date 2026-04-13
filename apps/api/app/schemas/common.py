"""Schémas Pydantic transverses utilisés dans tous les endpoints.

Ces schémas représentent la couche de transport HTTP — ils ne doublonnent
pas les modèles métier de `src/`, ils les adaptent pour le wire format.
"""

from __future__ import annotations

from datetime import datetime
from typing import Any, Generic, TypeVar

from pydantic import BaseModel, Field

DataT = TypeVar("DataT")


# ---------------------------------------------------------------------------
# Méta / Erreurs
# ---------------------------------------------------------------------------


class FieldErrorSchema(BaseModel):
    """Erreur de validation sur un champ spécifique."""

    field: str
    message: str
    code: str | None = None


class ApiErrorSchema(BaseModel):
    """Enveloppe d'erreur normalisée retournée par tous les endpoints."""

    code: str
    message: str
    retryable: bool = False
    details: dict | list | None = None
    field_errors: list[FieldErrorSchema] | None = None


class ApiMeta(BaseModel):
    """Métadonnées de réponse communes à toutes les réponses."""

    request_id: str
    generated_at: datetime
    locale: str = "fr"
    app_version: str
    data_version: str | None = None


# ---------------------------------------------------------------------------
# Entités communes
# ---------------------------------------------------------------------------


class PlayerSummary(BaseModel):
    """Résumé d'un profil joueur utilisé dans bootstrap, players, etc."""

    player_slug: str
    gamertag: str
    xuid: str
    waypoint_player: str
    is_demo: bool = False


class CapabilityMap(BaseModel):
    """Capacités actives selon la configuration du serveur."""

    can_read_local_data: bool
    can_run_sync: bool
    can_use_live_halo: bool
    can_manage_settings: bool
    can_reset_media_index: bool
    can_view_media: bool
    # Capacités de provisioning (Phase 2 plan V7)
    can_self_provision: bool = False
    can_start_initial_sync: bool = False
    can_manage_instance: bool = False


class LabelValue(BaseModel):
    """Option de filtre ou de sélecteur."""

    label: str
    value: str
    disabled: bool | None = None
    count: int | None = None


class SortSpec(BaseModel):
    """Spécification de tri pour les listes paginées."""

    field: str
    direction: str = "desc"  # "asc" | "desc"


class PaginationRequest(BaseModel):
    """Paramètres de pagination envoyés par le client."""

    page: int = Field(default=1, ge=1)
    page_size: int = Field(default=50, ge=1, le=500)


# ---------------------------------------------------------------------------
# Pagination
# ---------------------------------------------------------------------------


class PaginationMeta(BaseModel):
    """Informations de pagination retournées avec les listes."""

    total: int
    page: int
    page_size: int
    has_next: bool
    has_prev: bool


class PaginatedResponse(BaseModel, Generic[DataT]):
    """Enveloppe de réponse pour les listes paginées (refresh / tri / page)."""

    items: list[DataT]
    pagination: PaginationMeta
    freshness: FreshnessInfo | None = None


# ---------------------------------------------------------------------------
# Fraîcheur des données
# ---------------------------------------------------------------------------


class FreshnessInfo(BaseModel):
    """Indicateur de fraîcheur des données retournées."""

    source: str = "cached"  # "live" | "cached" | "mixed"
    sync_status: str = "unknown"  # "fresh" | "stale" | "unknown"
    warnings: list[str] = Field(default_factory=list)


# ---------------------------------------------------------------------------
# Enveloppe de page (page-oriented endpoints)
# ---------------------------------------------------------------------------


class PageEnvelope(BaseModel, Generic[DataT]):
    """Enveloppe complète pour les endpoints page-oriented (premier chargement).

    Contient toutes les informations de contexte dont la page a besoin
    pour s'initialiser sans appels API supplémentaires.
    """

    meta: ApiMeta
    player: PlayerSummary
    filters: Any | None = None  # FilterContextInput — typé dans filters.py
    freshness: FreshnessInfo
    capabilities: CapabilityMap
    data: DataT
    partial_errors: list[dict] | None = None


# ---------------------------------------------------------------------------
# Jobs asynchrones
# ---------------------------------------------------------------------------


class AsyncJobStatus(BaseModel):
    """Statut d'un job long (smoke test, sync, backfill, reindex…)."""

    job_id: str
    job_type: str  # "setup_smoke_test" | "initial_sync" | "backfill" | "reindex_media" | "other"
    status: str  # "queued" | "running" | "succeeded" | "failed" | "cancelled"
    progress_pct: int | None = None
    current_step: str | None = None
    started_at: datetime | None = None
    finished_at: datetime | None = None
    result: dict | None = None
    error: ApiErrorSchema | None = None
    # Champs enrichis pour la sync initiale (Sprint 3)
    phase_key: str | None = None
    phase_label: str | None = None
    matches_done: int | None = None
    matches_total: int | None = None
    subtasks_done: int | None = None
    subtasks_total: int | None = None
    eta_seconds: int | None = None
    warnings: list[str] = Field(default_factory=list)


# ---------------------------------------------------------------------------
# Figures Plotly
# ---------------------------------------------------------------------------


class PlotlyFigurePayload(BaseModel):
    """Figure Plotly sérialisée en JSON pour consommation par react-plotly.js."""

    figure: dict  # fig.to_plotly_json()
    config_key: str = "clean"  # "clean" | "static"
    revision_key: str | None = None
