"""Typage structuré du contexte de page et des paramètres match view.

Extrait de streamlit_app.py pour éviter le recours à Any dans PageContext.
"""

from __future__ import annotations

from collections.abc import Callable
from typing import Any, TypedDict


class MatchViewParams(TypedDict):
    """Paramètres communs passés aux pages de visualisation de match.

    Retourné par ``build_match_view_params()`` dans page_router.py.
    Les clés correspondent aux kwargs attendus par ``render_match_view_fn``,
    ``render_last_match_page_fn`` et ``render_match_search_page_fn``.
    """

    db_path: str
    xuid: str
    waypoint_player: str
    db_key: str | None
    settings: Any  # AppSettings (évite import circulaire au runtime)
    df_full: Any  # pl.DataFrame
    render_match_view_fn: Callable[..., Any]
    normalize_mode_label_fn: Callable[[str], str]
    format_score_label_fn: Callable[..., Any]
    score_css_color_fn: Callable[..., Any]
    format_datetime_fn: Callable[..., Any]
    load_player_match_result_fn: Callable[..., Any]
    load_match_medals_fn: Callable[..., Any]
    load_highlight_events_fn: Callable[..., Any]
    load_match_gamertags_fn: Callable[..., Any]
    load_match_rosters_fn: Callable[..., Any]
    paris_tz: Any  # ZoneInfo | pytz.timezone
