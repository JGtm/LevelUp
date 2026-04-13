"""Schemas Pydantic pour les filtres — contrat FilterContextInput / FilterContextResolved.

Ces schemas remplacent les shadow keys, session_state et GAP_MINUTES_FIXED
de ``filters_render.py`` / ``filter_state.py`` sans dépendance Streamlit.
"""

from __future__ import annotations

from datetime import date

from pydantic import BaseModel, Field

# ---------------------------------------------------------------------------
# Input
# ---------------------------------------------------------------------------


class PeriodInput(BaseModel):
    """Plage de dates (filtre mode Période)."""

    start_date: date | None = None
    end_date: date | None = None


class SessionsInput(BaseModel):
    """Sélecteurs de sessions (filtre mode Sessions)."""

    picked_session_label: str | None = None
    picked_solo_session_label: str | None = None
    picked_squad_session_label: str | None = None
    picked_sessions: list[str] = Field(default_factory=list)
    gap_minutes: int = Field(
        default=120,
        ge=1,
        le=1440,
        description="Invariant: toujours 120 dans le code actuel.",
    )


class CascadeInput(BaseModel):
    """Filtres cascade Expérience → Playlist → Mode → Carte.

    Listes vides = « tout coché » (pas de filtre restrictif).
    """

    experience_types: list[str] = Field(default_factory=list)
    playlists: list[str] = Field(default_factory=list)
    modes: list[str] = Field(default_factory=list)
    maps: list[str] = Field(default_factory=list)


class FilterContextInput(BaseModel):
    """Contexte de filtres complet envoyé par le frontend.

    Exemple minimal (plage de dates, pas de cascade) :
    ```json
    {
        "filter_mode": "period",
        "period": {"start_date": null, "end_date": null},
        "cascade": {}
    }
    ```
    """

    filter_mode: str = Field(
        default="period",
        pattern="^(period|sessions)$",
    )
    period: PeriodInput = Field(default_factory=PeriodInput)
    sessions: SessionsInput = Field(default_factory=SessionsInput)
    cascade: CascadeInput = Field(default_factory=CascadeInput)


# ---------------------------------------------------------------------------
# Output
# ---------------------------------------------------------------------------


class LabelValue(BaseModel):
    """Option affichable dans un multi-select."""

    label: str
    value: str


class AvailableOptions(BaseModel):
    """Options disponibles pour les filtres cascade, après application du mode/expérience."""

    experience_types: list[LabelValue] = Field(default_factory=list)
    playlists: list[LabelValue] = Field(default_factory=list)
    modes: list[LabelValue] = Field(default_factory=list)
    maps: list[LabelValue] = Field(default_factory=list)


class SessionOption(BaseModel):
    """Session disponible avec sa classification solo/squad."""

    label: str
    """Label affiché (ex: '2025-01-01 — Session 1')."""
    session_id: str
    match_count: int
    is_squad: bool
    """True si la session contient au moins un match joué avec des amis."""


class SessionOptions(BaseModel):
    """Sessions disponibles par catégorie."""

    all_sessions: list[SessionOption] = Field(default_factory=list)
    solo_labels: list[str] = Field(default_factory=list)
    squad_labels: list[str] = Field(default_factory=list)


class FilterCounts(BaseModel):
    """Compteurs avant/après application des filtres."""

    total_matches_before_filters: int
    total_matches_after_filters: int


class FilterContextResolved(BaseModel):
    """Réponse complète du endpoint ``POST /filters/resolve``.

    ``effective`` est le ``FilterContextInput`` normalisé (dates defaults,
    nouvelles options auto-cochées en mode exclude, etc.).
    """

    effective: FilterContextInput
    available_options: AvailableOptions
    session_options: SessionOptions
    counts: FilterCounts
