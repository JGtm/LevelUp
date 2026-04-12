"""Page Match View - Affichage détaillé d'un match."""

from __future__ import annotations

import html
import logging
from typing import Any

import streamlit as st

from src.app._page_context import MatchViewParams
from src.app.helpers import normalize_map_label, normalize_mode_label
from src.config import HALO_COLORS
from src.ui import (
    translate_pair_name,
    translate_playlist_name,
)
from src.ui.formatting import format_date_fr
from src.ui.i18n import get_lang, t
from src.ui.pages.match_view_helpers import (
    map_thumb_path,
    os_card,  # noqa: F401 — re-export public (importé par test_phase6_refactoring)
)
from src.ui.pages.match_view_logic import (
    compute_perf_display,
    enrich_pm_from_row,
    resolve_outcome,
)
from src.ui.pages.match_view_rank import _build_match_rank_html
from src.ui.pages.match_view_tabs import (
    _render_citations_tab,
    _render_combat_tab,
    _render_media_tab,
    _render_summary_tab,
    _render_team_tab,
)
from src.visualization._compat import ensure_polars

logger = logging.getLogger(__name__)

_DOMINANCE_BADGE_STYLES: dict[int, tuple[str, str, str]] = {
    # flag → (i18n_key, bg_color, text_color)
    1: ("outcome_domination", "#2e7d32", "#e8f5e9"),  # vert foncé
    2: ("outcome_humiliation", "#6a1b9a", "#f3e5f5"),  # violet foncé
    3: ("career_top_badge_remontada", "#1565c0", "#e3f2fd"),  # bleu
    4: ("career_top_badge_debandade", "#bf360c", "#fbe9e7"),  # rouge-brique
    5: ("career_top_badge_contre_remontada", "#00695c", "#e0f2f1"),  # vert-canard
}


def _dominance_badge_html(flag: int | None) -> str:
    """Retourne le HTML d'un badge domination/humiliation, ou '' si non applicable."""
    if flag is None or flag not in _DOMINANCE_BADGE_STYLES:
        return ""
    i18n_key, bg, fg = _DOMINANCE_BADGE_STYLES[flag]
    label = html.escape(t(i18n_key))
    return (
        f"<span style='display:block;margin-top:1px;padding:3px 10px;"
        f"border-radius:4px;font-size:0.975em;font-weight:600;"
        f"background:{bg};color:{fg}'>{label}</span>"
    )


_KPI_TEXT_STYLE = (
    "font-family:var(--font-display);font-size:24px;font-weight:400;"
    "letter-spacing:0.02em;color:rgba(255,255,255,0.98)"
)


def _render_simple_kpi_tile(text: str) -> None:
    """Affiche une carte KPI simple avec un container natif."""
    with st.container(border=True):
        st.markdown(
            f"<div style='text-align:center;padding:9px 15px;{_KPI_TEXT_STYLE}'>"
            f"{html.escape(str(text))}</div>",
            unsafe_allow_html=True,
        )


def _render_score_kpi_tile(
    score_label: str,
    outcome_color: str,
    dominance_flag: int | None,
) -> None:
    """Affiche la carte KPI du score avec badge optionnel."""
    score_escaped = html.escape(str(score_label))
    badge_html = _dominance_badge_html(dominance_flag)
    score_color = outcome_color if str(outcome_color).strip() else "rgba(255,255,255,0.98)"

    with st.container(border=True):
        st.markdown(
            f"<div style='text-align:center;padding:9px 15px'>"
            f"<div style='font-family:var(--font-display);font-size:38px;font-weight:700;line-height:1;color:{score_color}'>"
            f"{score_escaped}</div>"
            f"{badge_html}"
            f"</div>",
            unsafe_allow_html=True,
        )


def _render_kpi_cards(  # noqa: PLR0913
    *,
    last_time: Any,
    outcome_code: int | None,
    outcome_label: str,
    outcome_color: str,
    score_label: str,
    had_bot: bool,
    dominance_flag: int | None = None,
    row: dict[str, Any],
) -> None:
    """Affiche la rangée KPI unique : Date, Résultat, Playlist, Mode sur Carte."""
    lang = get_lang()

    # Infos carte/playlist/mode
    last_playlist = row.get("playlist_name")
    last_playlist_fr = row.get("playlist_name_fr")
    last_pair = row.get("pair_name")
    map_display = _display_map(row)
    playlist_display = (
        str(last_playlist_fr)
        if last_playlist_fr
        else (translate_playlist_name(str(last_playlist), lang=lang) if last_playlist else "-")
    )
    last_mode_ui = row.get("mode_ui") or normalize_mode_label(
        str(last_pair) if last_pair else None, lang=lang
    )
    last_pair_fr = translate_pair_name(str(last_pair), lang=lang) if last_pair else None
    mode_display = last_mode_ui or last_pair_fr or row.get("game_variant_name") or "-"
    _sep = " sur " if lang == "fr" else " on "
    mode_map_display = f"{mode_display}{_sep}{map_display}"

    date_display = format_date_fr(last_time, lang=lang)
    date_col, score_col, playlist_col, mode_col = st.columns(4)

    with date_col:
        _render_simple_kpi_tile(date_display)
    with score_col:
        _render_score_kpi_tile(score_label, outcome_color, dominance_flag)
    with playlist_col:
        _render_simple_kpi_tile(playlist_display)
    with mode_col:
        _render_simple_kpi_tile(mode_map_display)


def _render_map_and_rank(  # noqa: PLR0913
    row: dict[str, Any],
    *,
    map_display: str,
    db_path: str,
    match_id: str,
    db_key: tuple[int, int] | None,
    had_bot: bool,
    outcome_code: int | None,
    perf_display: str = "-",
    perf_color: str | None = None,
) -> None:
    """Affiche la miniature de carte, le bloc performance et le rang côte à côte."""
    map_id = row.get("map_id")
    thumb = map_thumb_path(row, str(map_id) if map_id else None)

    rank_html = _build_match_rank_html(
        match_id=match_id,
        db_path=db_path,
        db_key=db_key,
        had_bot_teammate=had_bot,
        bot_outcome=outcome_code,
    )

    if not rank_html:
        _render_map_thumbnail(thumb)
        return

    map_col, perf_col, rank_col = st.columns([1.8, 0.7, 1.2])
    with map_col:
        _render_map_thumbnail(thumb)
    with perf_col:
        _render_performance_block(perf_display, perf_color)
    with rank_col:
        st.markdown(rank_html, unsafe_allow_html=True)


def _render_map_thumbnail(thumb: Any) -> None:
    """Affiche la miniature de carte avec un fallback natif si absente."""
    if thumb:
        st.image(str(thumb), width="stretch")
        return
    st.info(t("mv_thumbnail_unavailable"))


def _render_performance_block(perf_display: str, perf_color: str | None) -> None:
    """Affiche le score de performance dans un bloc compact."""
    score_color = perf_color if (perf_color and perf_display != "-") else "#888888"
    score_display = html.escape(perf_display)
    label = html.escape(t("mv_performance"))
    st.markdown(
        """<div style='text-align:center;white-space:nowrap'>
<div style='font-size:1.4em;font-weight:700;line-height:1.2;color:#dddddd'>"""
        f"{label}"
        """</div>
<div style='font-size:4.2em;font-weight:700;margin-top:4px;line-height:1;color:"""
        f"{score_color}'>"
        f"{score_display}"
        """</div>
</div>""",
        unsafe_allow_html=True,
    )


def render_match_view(
    *,
    row: dict[str, Any],
    match_id: str,
    params: MatchViewParams,
) -> None:
    """Rend la vue détaillée d'un match."""
    db_path = params.db_path
    xuid = params.xuid
    db_key = params.db_key
    settings = params.settings
    df_full = params.df_full
    if df_full is not None:
        df_full = ensure_polars(df_full)

    match_id = str(match_id or "").strip()
    if not match_id:
        st.info(t("mv_match_id_missing"))
        return

    logger.debug("render_match_view match=%s xuid=%s", match_id, xuid)

    from src.ui._cache_core import get_cached_repository_st

    repo = get_cached_repository_st(db_path, str(xuid).strip())

    outcome_code, outcome_label, outcome_color = resolve_outcome(row)
    colors = HALO_COLORS.as_dict()
    score_label = params.format_score_label_fn(
        row.get("my_team_score"), row.get("enemy_team_score")
    )
    match_url = _build_waypoint_url(params.waypoint_player, match_id)
    _had_bot, _stored_perf, _dominance_flag = repo.load_player_match_enrichment(match_id)
    _perf_score, perf_display, perf_color = compute_perf_display(
        row, df_full, _stored_perf, _had_bot
    )

    _render_match_header(
        row=row,
        outcome_code=outcome_code,
        outcome_label=outcome_label,
        outcome_color=outcome_color,
        score_label=score_label,
        perf_display=perf_display,
        perf_color=perf_color,
        had_bot=_had_bot,
        dominance_flag=_dominance_flag,
        db_path=db_path,
        match_id=match_id,
        db_key=db_key,
    )

    is_abandoned = repo.is_abandoned_match(match_id)
    if is_abandoned:
        st.warning(
            f"**{t('mv_abandoned_match')}** — {t('mv_abandoned_match_desc')}",
            icon="⚠️",
        )

    with st.spinner(t("mv_loading")):
        pm = params.load_player_match_result_fn(db_path, match_id, xuid.strip(), db_key=db_key)
        medals_last = params.load_match_medals_fn(db_path, match_id, xuid.strip(), db_key=db_key)
    if pm:
        enrich_pm_from_row(pm, row)

    _render_match_tabs(
        row=row,
        pm=pm,
        medals_last=medals_last,
        colors=colors,
        df_full=df_full,
        db_path=db_path,
        xuid=xuid,
        match_id=match_id,
        db_key=db_key,
        outcome_code=outcome_code,
        match_url=match_url,
        settings=settings,
        params=params,
        is_abandoned=is_abandoned,
    )


def _render_match_header(  # noqa: PLR0913
    *,
    row: dict,
    outcome_code: int | None,
    outcome_label: str,
    outcome_color: str,
    score_label: str,
    perf_display: str,
    perf_color: str | None,
    had_bot: bool,
    dominance_flag: int | None,
    db_path: str,
    match_id: str,
    db_key: Any,
) -> None:
    """Affiche KPI fusionnées (1 rangée) et miniature carte + rang."""
    _render_kpi_cards(
        last_time=row.get("start_time"),
        outcome_code=outcome_code,
        outcome_label=outcome_label,
        outcome_color=outcome_color,
        score_label=score_label,
        had_bot=had_bot,
        dominance_flag=dominance_flag,
        row=row,
    )
    _render_map_and_rank(
        row,
        map_display=_display_map(row),
        db_path=db_path,
        match_id=match_id,
        db_key=db_key,
        had_bot=had_bot,
        outcome_code=outcome_code,
        perf_display=perf_display,
        perf_color=perf_color,
    )


def _short_match_id(match_id: str) -> str:
    """Retourne une version courte du match_id pour l'UI."""
    return match_id[:8] if len(match_id) >= 8 else match_id


def _render_match_id_badge(match_id: str) -> None:
    """Affiche un accès discret au match ID complet sans iframe HTML."""
    if not match_id:
        return
    short_id = _short_match_id(match_id)
    with st.popover(t("mv_match_id_popover", short_id=short_id)):
        st.caption(t("mv_match_id_copy_hint"))
        st.code(match_id, language=None)


def _render_match_tabs(  # noqa: PLR0913
    *,
    row: dict,
    pm: Any,
    medals_last: Any,
    colors: dict,
    df_full: Any,
    db_path: str,
    xuid: str,
    match_id: str,
    db_key: Any,
    outcome_code: int | None,
    match_url: str | None,
    settings: Any,
    params: MatchViewParams,
    is_abandoned: bool = False,
) -> None:
    """Affiche les 5 onglets du match."""
    _render_match_id_badge(match_id)
    tabs = st.tabs(
        [
            t("mv_tab_summary"),
            t("mv_tab_combat"),
            t("mv_tab_team"),
            t("mv_tab_citations_medals"),
            t("mv_tab_media"),
        ]
    )
    with tabs[0]:
        _render_summary_tab(
            pm, row, colors, df_full, db_path, xuid, match_id, db_key, is_abandoned=is_abandoned
        )
    with tabs[1]:
        _render_combat_tab(
            match_id,
            db_path,
            xuid,
            db_key,
            outcome_code,
            row,
            params.load_highlight_events_fn,
            params.load_match_gamertags_fn,
            colors,
            is_abandoned=is_abandoned,
        )
    with tabs[2]:
        _render_team_tab(match_id, db_path, xuid, db_key, params.load_match_gamertags_fn)
    with tabs[3]:
        _render_citations_tab(match_id, db_path, xuid, medals_last)
    with tabs[4]:
        _render_media_tab(
            row,
            settings,
            params.format_datetime_fn,
            params.paris_tz,
            params.waypoint_player,
            db_path,
            xuid,
            match_url,
        )


def _build_waypoint_url(waypoint_player: str | None, match_id: str) -> str | None:
    """Construit l'URL Waypoint pour un match."""
    wp = str(waypoint_player or "").strip()
    if wp and match_id and match_id.strip() != "-":
        return f"https://www.halowaypoint.com/halo-infinite/players/{wp}/matches/{match_id.strip()}"
    return None


def _display_map(row: dict[str, Any]) -> str:
    """Retourne le label carte normalisé (FR en priorité, fallback EN normalisé)."""
    map_fr = row.get("map_name_fr")
    if map_fr:
        return str(map_fr)
    last_map = row.get("map_name")
    return normalize_map_label(last_map) if last_map else "-"


__all__ = ["render_match_view"]
