"""Schémas Pydantic — Match View (Slice 4 Phase B) + Last Match (Slice 4 Phase C).

Contrats :
  GET  /api/v1/players/{slug}/matches/{match_id}           → MatchViewResponse
  POST /api/v1/players/{slug}/pages/last-match/resolve     → LastMatchResolveResponse
"""

from __future__ import annotations

from datetime import datetime

from pydantic import BaseModel

from apps.api.app.schemas.career import PlotlyFigurePayload
from apps.api.app.schemas.filters import FilterContextInput

# ---------------------------------------------------------------------------
# Éléments du header
# ---------------------------------------------------------------------------


class MatchViewHeader(BaseModel):
    """En-tête du match : identité, résultat, contexte."""

    match_id: str
    start_time: datetime | None = None
    start_time_label: str = ""
    outcome_code: int | None = None
    outcome_label: str = "-"
    outcome_color: str = "#94a3b8"
    score_label: str = ""
    dominance_flag: bool = False
    had_bot_teammate: bool = False
    map_ui: str = ""
    map_id: str | None = None
    mode_ui: str = ""
    playlist_label: str = ""
    performance_display: str = "-"
    performance_color: str | None = None


class MatchViewRank(BaseModel):
    """Informations de rank (CSR ou LUSR) pour ce match."""

    rating_type: str = "none"  # "CSR" | "LUSR" | "none"
    tier_label: str | None = None
    numeric_value: float | None = None
    delta_value: float | None = None
    icon_url: str | None = None


# ---------------------------------------------------------------------------
# Onglet résumé
# ---------------------------------------------------------------------------


class MatchMedal(BaseModel):
    """Une médaille gagnée dans le match."""

    medal_name_id: int
    name: str
    count: int
    description: str | None = None


class MatchCitation(BaseModel):
    """Un badge de citation (commendation Halo 5) associé au match."""

    key: str
    label: str
    color: str | None = None
    value: float | None = None


class MatchSummaryKpis(BaseModel):
    """KPIs personnels du résumé de match."""

    kills: int | None = None
    deaths: int | None = None
    assists: int | None = None
    kda: float | None = None
    damage_dealt: float | None = None
    average_life: str | None = None


class MatchPersonalResult(BaseModel):
    """Résultat personnel du joueur sur ce match."""

    outcome_label: str = "-"
    outcome_color: str = "#94a3b8"
    score: int | None = None
    rank_in_team: int | None = None


class MatchExpectedStats(BaseModel):
    """Comparaison réel vs attendu (CSR/LUSR) — C3 NATIVE_COMPONENTS."""

    has_expected_data: bool = False
    expected_kills: float | None = None
    expected_deaths: float | None = None
    expected_assists: float | None = None


class MatchSummaryTab(BaseModel):
    """Contenu de l'onglet Résumé."""

    kpis: MatchSummaryKpis
    personal_result: MatchPersonalResult
    medals: list[MatchMedal]
    citations: list[MatchCitation]
    expected_stats: MatchExpectedStats = MatchExpectedStats()


# ---------------------------------------------------------------------------
# Onglet combat
# ---------------------------------------------------------------------------


class MatchWeaponKill(BaseModel):
    """Kills par arme pour le joueur sur ce match."""

    weapon_id: int
    weapon_label: str
    effective_weapon_id: int | None = None
    kill_count: int


class MatchHighlightEvent(BaseModel):
    """Événement de highlight (medal, kill, spike) horodaté."""

    event_time_ms: int | None = None
    event_type: str
    actor_xuid: str | None = None
    target_xuid: str | None = None
    weapon_id: int | None = None


class MatchCombatTab(BaseModel):
    """Contenu de l'onglet Combat."""

    weapon_kills: list[MatchWeaponKill]
    highlight_events: list[MatchHighlightEvent]
    charts: list[PlotlyFigurePayload]


# ---------------------------------------------------------------------------
# Onglet équipe
# ---------------------------------------------------------------------------


class MatchRosterRow(BaseModel):
    """Ligne du roster d'équipe (participation simple)."""

    xuid: str
    gamertag: str
    team_side: str | None = None
    is_me: bool = False
    is_bot: bool = False
    kills: int | None = None
    deaths: int | None = None
    assists: int | None = None
    kda: float | None = None
    damage_dealt: float | None = None
    damage_taken: float | None = None


class MatchScoreboardRow(BaseModel):
    """Ligne du scoreboard complet (19 colonnes)."""

    xuid: str
    gamertag: str
    team_side: str | None = None
    is_me: bool = False
    rank: int | None = None
    kills: int | None = None
    deaths: int | None = None
    assists: int | None = None
    betrayals: int | None = None
    suicides: int | None = None
    shots_fired: int | None = None
    shots_hit: int | None = None
    shots_accuracy: float | None = None
    damage_dealt: float | None = None
    damage_taken: float | None = None
    damage_efficiency: float | None = None
    average_life: str | None = None
    objectives_stolen: int | None = None
    headshot_kills: int | None = None
    max_killing_spree: int | None = None
    perfect_kills: int | None = None
    power_weapon_kills: int | None = None
    melee_kills: int | None = None
    outcome_label: str = "-"


class MatchNemesisRow(BaseModel):
    """Adversaire fréquent (kills reçus de lui)."""

    xuid: str
    gamertag: str
    killed_me: int
    i_killed: int


class MatchEncounterRow(BaseModel):
    """Récurrence d'encounter avec un joueur croisé."""

    xuid: str
    gamertag: str
    count_together: int
    is_ally: bool


class MatchTeamTab(BaseModel):
    """Contenu de l'onglet Équipe."""

    roster: list[MatchRosterRow]
    scoreboard: list[MatchScoreboardRow]
    nemesis: list[MatchNemesisRow]
    encounters: list[MatchEncounterRow]


# ---------------------------------------------------------------------------
# Onglet médias
# ---------------------------------------------------------------------------


class AssociatedMediaItem(BaseModel):
    """Fichier média associé au match."""

    file_id: str
    file_name: str
    file_path: str
    thumbnail_url: str | None = None
    duration_seconds: float | None = None
    capture_time: datetime | None = None
    liked: bool = False


class MatchMediaTab(BaseModel):
    """Contenu de l'onglet Médias."""

    media_items: list[AssociatedMediaItem]


# ---------------------------------------------------------------------------
# Onglet citations (Commendations + Médailles contextuelles)
# ---------------------------------------------------------------------------


class MatchCitationsTab(BaseModel):
    """Contenu de l'onglet Citations."""

    commendations: list[MatchCitation]
    medals: list[MatchMedal]


# ---------------------------------------------------------------------------
# Réponse complète Match View
# ---------------------------------------------------------------------------


class MatchViewResponse(BaseModel):
    """Réponse complète pour la vue détail d'un match."""

    header: MatchViewHeader
    rank: MatchViewRank
    summary_tab: MatchSummaryTab
    combat_tab: MatchCombatTab
    team_tab: MatchTeamTab
    media_tab: MatchMediaTab
    citations_tab: MatchCitationsTab


# ---------------------------------------------------------------------------
# Last Match — resolve
# ---------------------------------------------------------------------------


class LastMatchResolveRequest(BaseModel):
    """Requête de résolution du dernier match (ou match courant de la navigation)."""

    filters: FilterContextInput
    current_index: int | None = None


class LastMatchResolveResponse(BaseModel):
    """Réponse de résolution du dernier match."""

    current_match_id: str
    total_matches_in_scope: int
    current_index: int
    previous_match_id: str | None = None
    next_match_id: str | None = None
    session_tracking_key: str
