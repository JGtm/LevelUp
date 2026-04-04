"""Page Match View - Affichage détaillé d'un match."""

from __future__ import annotations

import contextlib
import html
import logging
from typing import Any

import streamlit as st
import streamlit.components.v1 as components

from src.app._page_context import MatchViewParams
from src.app.helpers import normalize_map_label, normalize_mode_label
from src.config import HALO_COLORS, OUTCOME_CODES
from src.ui import (
    translate_pair_name,
    translate_playlist_name,
)
from src.ui.formatting import format_date_fr
from src.ui.i18n import get_lang, t
from src.ui.pages.match_view_helpers import (
    map_thumb_path,
    os_card,
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

    outcome_class = (
        "text-win"
        if outcome_code == OUTCOME_CODES.WIN
        else ("text-loss" if outcome_code == OUTCOME_CODES.LOSS else "text-tie")
    )
    score_escaped = html.escape(str(score_label))
    dominance_badge = _dominance_badge_html(dominance_flag)
    badge_html = (
        f"<div style='margin-top:1px'>{dominance_badge}</div>" if dominance_badge else ""
    )

    _accent66 = f"{outcome_color}66" if str(outcome_color).startswith("#") else outcome_color
    _clip = (
        "clip-path:polygon(0 var(--corner-size),var(--corner-size) 0,100% 0,"
        "100% calc(100% - var(--corner-size)),calc(100% - var(--corner-size)) 100%,0 100%);"
    )
    _card_inner_style = (
        "display:flex;flex-direction:column;justify-content:center;align-items:center;"
        f"text-align:center;padding:9px 15px;background:rgba(21,50,62,0.7);height:100%;{_clip}"
    )
    _text_style = "font-family:var(--font-display);font-size:24px;font-weight:400;letter-spacing:0.02em;color:rgba(255,255,255,0.98)"
    _score_kpi = (
        f"<span class='{outcome_class} fw-bold' style='font-family:var(--font-display);font-size:38px;line-height:1'>{score_escaped}</span>"
    )

    def _simple_card(text: str) -> str:
        return (
            f"<div class='os-card' style='flex:1;margin-bottom:10px;"
            f"display:flex;flex-direction:column;justify-content:center;align-items:center;"
            f"text-align:center;padding:9px 15px;'>"
            f"<div style='{_text_style}'>{html.escape(str(text))}</div>"
            f"</div>"
        )

    date_display = format_date_fr(last_time, lang=lang)

    _score_card = (
        f"<div class='os-card' style='flex:1;margin-bottom:10px;"
        f"--card-border-color:{_accent66};"
        f"display:flex;flex-direction:column;justify-content:center;align-items:center;text-align:center;padding:9px 15px;'>"
        f"{_score_kpi}{badge_html}"
        f"</div>"
    )

    st.markdown(
        f"<div style='display:flex;gap:8px;align-items:stretch;margin-bottom:0'>"
        f"{_simple_card(date_display)}"
        f"{_score_card}"
        f"{_simple_card(playlist_display)}"
        f"{_simple_card(mode_map_display)}"
        f"</div>",
        unsafe_allow_html=True,
    )


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

    from src.ui.player_assets import file_to_data_url as _f2du

    thumb_data_url = None
    if thumb:
        with contextlib.suppress(Exception):
            thumb_data_url = _f2du(str(thumb))

    rank_html = _build_match_rank_html(
        match_id=match_id,
        db_path=db_path,
        db_key=db_key,
        had_bot_teammate=had_bot,
        bot_outcome=outcome_code,
    )

    if thumb_data_url:
        map_img_html = (
            f"<img src='{thumb_data_url}' "
            f"style='width:100%;max-width:480px;height:auto;border-radius:4px;object-fit:cover'>"
        )
    else:
        map_img_html = (
            "<div style='padding:16px;color:#888;font-style:italic'>"
            f"{t('mv_thumbnail_unavailable')}</div>"
        )

    # Bloc performance (label + score coloré)
    import html as _html

    _score_color = perf_color if (perf_color and perf_display != "-") else "#888888"
    _score_escaped = _html.escape(perf_display)
    _label_escaped = _html.escape(t("mv_performance"))
    perf_html = (
        f"<div style='text-align:center;white-space:nowrap'>"
        f"<div style='font-size:1.4em;font-weight:700;line-height:1.2;color:#dddddd'>{_label_escaped}</div>"
        f"<div style='font-size:4.2em;font-weight:700;color:{_score_color};margin-top:4px;line-height:1'>{_score_escaped}</div>"
        f"</div>"
    )

    if rank_html:
        st.markdown(
            f"""<div style='display:flex;align-items:center;gap:20px;flex-wrap:wrap;margin-bottom:1.5rem'>
  <div style='flex:1 1 45%;min-width:180px;max-width:420px'>{map_img_html}</div>
  <div style='flex:0 0 auto;min-width:90px'>{perf_html}</div>
  <div style='flex:0 0 1px;align-self:stretch;background:rgba(255,255,255,0.12);border-radius:1px'></div>
  <div style='flex:0 0 auto;width:320px;max-width:320px;overflow:hidden'>{rank_html}</div>
</div>""",
            unsafe_allow_html=True,
        )
    else:
        st.markdown(map_img_html, unsafe_allow_html=True)


def render_match_view(  # noqa: C901, PLR0912
    *,
    row: dict[str, Any],
    match_id: str,
    params: MatchViewParams,
) -> None:
    """Rend la vue détaillée d'un match."""
    db_path = params["db_path"]
    xuid = params["xuid"]
    db_key = params["db_key"]
    settings = params["settings"]
    df_full = params.get("df_full")
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
    score_label = params["format_score_label_fn"](
        row.get("my_team_score"), row.get("enemy_team_score")
    )
    match_url = _build_waypoint_url(params["waypoint_player"], match_id)
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
        pm = params["load_player_match_result_fn"](db_path, match_id, xuid.strip(), db_key=db_key)
        medals_last = params["load_match_medals_fn"](db_path, match_id, xuid.strip(), db_key=db_key)
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


def _render_match_id_badge(match_id: str) -> None:
    """Affiche un badge discret avec le match ID court et un bouton copier."""
    if not match_id:
        return
    short_id = match_id[:8] if len(match_id) >= 8 else match_id
    badge_html = f"""<!DOCTYPE html>
<html><head><style>
html,body{{margin:0;padding:0;background:transparent;overflow:hidden;}}
.badge{{display:inline-flex;align-items:center;gap:7px;padding:1px 0;}}
.mid{{font-size:11px;color:#8a8a8a;font-family:'Courier New',monospace;user-select:none;}}
.mid-val{{color:#c0c0c0;}}
.btn{{background:transparent;border:1px solid #484848;border-radius:4px;
  color:#888;cursor:pointer;font-size:10px;padding:0 6px;line-height:1.75;
  transition:all 0.15s;font-family:sans-serif;}}
.btn:hover{{border-color:#888;color:#ccc;}}
</style></head>
<body><div class="badge">
  <span class="mid">ID&thinsp;<span class="mid-val">{short_id}&hellip;</span></span>
  <button class="btn" id="b" title="Copier le match ID complet"
    onclick="navigator.clipboard.writeText('{match_id}').then(()=>{{
      var b=document.getElementById('b');
      b.textContent='&#x2713;';b.style.color='#4caf50';b.style.borderColor='#4caf50';
      setTimeout(()=>{{b.textContent='&#x1F4CB;';b.style.color='#888';b.style.borderColor='#484848';}},1500);
    }});">&#x1F4CB;</button>
</div></body></html>"""
    components.html(badge_html, height=26, scrolling=False)


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
            params["load_highlight_events_fn"],
            params["load_match_gamertags_fn"],
            colors,
            is_abandoned=is_abandoned,
        )
    with tabs[2]:
        _render_team_tab(match_id, db_path, xuid, db_key, params["load_match_gamertags_fn"])
    with tabs[3]:
        _render_citations_tab(match_id, db_path, xuid, medals_last)
    with tabs[4]:
        _render_media_tab(
            row,
            settings,
            params["format_datetime_fn"],
            params["paris_tz"],
            params["waypoint_player"],
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
