"""Schémas Pydantic — Accueil Mission Control (Slice 5).

Contrats :
  GET  /api/v1/players/{slug}/pages/home    → HomePageResponse
  GET  /api/v1/players/{slug}/battlepass    → BattlePassResponse
  GET  /api/v1/players/{slug}/challenges    → ChallengesResponse
"""

from __future__ import annotations

from datetime import datetime

from pydantic import BaseModel

# ---------------------------------------------------------------------------
# Blocs hero
# ---------------------------------------------------------------------------


class HeroKPIs(BaseModel):
    """KPIs globaux affichés dans le hero card."""

    win_rate: float = 0.0
    global_ratio: float | None = None
    avg_accuracy: float | None = None
    total_matches: int = 0
    wins: int = 0
    losses: int = 0


class HeroTrend(BaseModel):
    """Variation par fenêtre glissante (5 derniers vs 5 précédents)."""

    ratio_delta: float | None = None
    accuracy_delta: float | None = None
    win_rate_delta: float | None = None


class HomeHeroCard(BaseModel):
    """Carte briefing principal de l'accueil."""

    player_name: str
    kpis: HeroKPIs
    trend: HeroTrend | None = None


# ---------------------------------------------------------------------------
# Signaux et highlights
# ---------------------------------------------------------------------------


class HighlightItem(BaseModel):
    """Fait saillant synthétique pour la zone signaux."""

    title: str
    value: str
    detail: str


# ---------------------------------------------------------------------------
# Matchs récents
# ---------------------------------------------------------------------------


class RecentMatchItem(BaseModel):
    """Un match récent affiché dans la timeline."""

    match_id: str
    title: str
    detail: str
    started_at: datetime | None = None
    outcome_label: str
    outcome_tone: str


# ---------------------------------------------------------------------------
# Résumé de session
# ---------------------------------------------------------------------------


class SessionSummaryItem(BaseModel):
    """Résumé d'une session (solo ou escouade) affiché dans l'accueil."""

    session_label: str
    match_count: int
    win_rate: float
    global_ratio: float | None = None
    started_at: datetime | None = None


# ---------------------------------------------------------------------------
# Médias récents
# ---------------------------------------------------------------------------


class RecentMediaItem(BaseModel):
    """Entrée compacte d'un média récent."""

    basename: str
    match_id: str | None = None
    match_start_time: datetime | None = None


# ---------------------------------------------------------------------------
# Réponse principale
# ---------------------------------------------------------------------------


class HomePageResponse(BaseModel):
    """Réponse agrégée de la page d'accueil Mission Control."""

    hero: HomeHeroCard
    highlights: list[HighlightItem]
    recent_matches: list[RecentMatchItem]
    recent_media: list[RecentMediaItem]
    solo_session: SessionSummaryItem | None = None
    squad_session: SessionSummaryItem | None = None


# ---------------------------------------------------------------------------
# Battle Pass (appel API live)
# ---------------------------------------------------------------------------


class BattlePassResponse(BaseModel):
    """Informations Battle Pass live depuis l'API Halo."""

    available: bool = False
    rank: int | None = None
    reward_track: str | None = None
    progress: int | None = None
    error_hint: str | None = None


# ---------------------------------------------------------------------------
# Défis actifs (appel API live)
# ---------------------------------------------------------------------------


class ChallengesResponse(BaseModel):
    """Résumé des défis actifs récupérés depuis l'API Halo."""

    available: bool = False
    total: int | None = None
    completed: int | None = None
    xp_available: int | None = None
    next_expiry: str | None = None
    error_hint: str | None = None
