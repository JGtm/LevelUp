"""Typage structuré du contexte de page et des paramètres match view.

Extrait de streamlit_app.py pour éviter le recours à Any dans PageContext.
"""

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from datetime import date
from typing import Any


@dataclass(frozen=True)
class MatchViewParams:
    """Paramètres communs passés aux pages de visualisation de match.

    Retourné par ``build_match_view_params()`` dans page_router.py.
    Les champs correspondent aux kwargs attendus par ``render_match_view_fn``
    et ``render_last_match_page_fn``.
    """

    db_path: str
    xuid: str
    waypoint_player: str
    db_key: str | None
    settings: Any  # AppSettings (évite import circulaire au runtime)
    df_full: Any  # pl.DataFrame
    render_match_view_fn: Callable[..., Any]
    format_score_label_fn: Callable[..., Any]
    score_css_color_fn: Callable[..., Any]
    format_datetime_fn: Callable[..., Any]
    load_player_match_result_fn: Callable[..., Any]
    load_match_medals_fn: Callable[..., Any]
    load_highlight_events_fn: Callable[..., Any]
    load_match_gamertags_fn: Callable[..., Any]
    load_match_rosters_fn: Callable[..., Any]
    paris_tz: Any  # ZoneInfo | pytz.timezone


@dataclass(frozen=True)
class FilterSidebarCallbacks:
    """Callbacks injectées dans render_filters_sidebar."""

    date_range_fn: Callable[..., tuple[date, date]]
    clean_asset_label_fn: Callable[[str], str]
    build_friends_opts_map_fn: Callable[..., Any]


@dataclass(frozen=True)
class TeammateContext:
    """Contexte de données pour les vues coéquipiers."""

    me_name: str
    xuid: str
    db_path: str
    db_key: tuple[int, int] | None
    picked_xuids: list[str]
    aliases_key: int | None
    picked_session_labels: list[str] | None
    include_firefight: bool
    waypoint_player: str


@dataclass(frozen=True)
class TeammateFilterOptions:
    """Options de filtrage pour les vues coéquipiers."""

    apply_current_filters: bool
    same_team_only: bool
    show_smooth: bool


@dataclass(frozen=True)
class TeammateCallbacks:
    """Callbacks injectées dans les vues coéquipiers."""

    assign_player_colors_fn: Callable[..., Any]
    plot_multi_metric_bars_fn: Callable[..., Any]
    top_medals_fn: Callable[..., Any]
    load_teammate_stats_fn: Callable[..., Any]
    enrich_series_fn: Callable[..., Any]
