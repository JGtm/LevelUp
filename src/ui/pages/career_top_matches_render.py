"""Page Carrière — Rendu Streamlit du Top 10 meilleurs / pires matchs."""

from __future__ import annotations

import html
import logging
import urllib.parse
from datetime import datetime

import streamlit as st

from src.ui.i18n import t
from src.ui.pages.career_top_matches_data import (
    load_top_best_matches,
    load_top_worst_matches,
)
from src.ui.pages.win_loss_table_style import map_name_cell_html

logger = logging.getLogger(__name__)

# ── Couleurs badges ─────────────────────────────────────────────────────────
_BG_DOMINATION = "#2e7d32"
_FG_DOMINATION = "#e8f5e9"
_BG_HUMILIATION = "#6a1b9a"
_FG_HUMILIATION = "#f3e5f5"


def _format_duration(seconds: int | float | None) -> str:
    """Formate une durée en mm:ss."""
    if seconds is None or seconds <= 0:
        return "—"
    m, s = divmod(int(seconds), 60)
    return f"{m}:{s:02d}"


def _format_score(my: int | None, enemy: int | None) -> str:
    """Formate le score sous forme 'my — enemy'."""
    return f"{my or 0} — {enemy or 0}"


def _match_id_link(match_id: str) -> str:
    """Lien HTML tronqué vers la page Explorer pour ce match."""
    if not match_id:
        return "—"
    href = "/?" + urllib.parse.urlencode({"page": "Explorer", "match_id": match_id})
    short = html.escape(match_id[:8] + "…")
    return (
        f"<a href='{html.escape(href)}' target='_self' "
        f"style='font-family:monospace;font-size:0.75em;"
        f"color:var(--accent,#33d6ff);text-decoration:none;'>{short}</a>"
    )


def _format_kda(row: dict) -> str:
    """Formate K/D/A compacts."""
    k = row.get("kills", 0) or 0
    d = row.get("deaths", 0) or 0
    a = row.get("assists", 0) or 0
    return f"{k}/{d}/{a}"


def _kd_ratio(row: dict) -> str:
    """Ratio K/D affiché."""
    k = row.get("kills", 0) or 0
    d = row.get("deaths", 0) or 0
    if d == 0:
        return f"{k:.1f}" if k else "—"
    return f"{k / d:.2f}"


def _kd_color(row: dict) -> str:
    """Couleur CSS selon ratio K/D."""
    k = row.get("kills", 0) or 0
    d = row.get("deaths", 0) or 0
    if d == 0:
        return "#3DFFB5" if k > 0 else ""
    ratio = k / d
    if ratio >= 1.5:
        return "#3DFFB5"
    if ratio <= 0.67:
        return "#FF4D6D"
    return ""


def _format_date(raw: object) -> str:
    """Convertit en date lisible (JJ/MM/AAAA HH:MM)."""
    if isinstance(raw, datetime):
        return raw.strftime("%d/%m/%Y %H:%M")
    if raw is not None:
        try:
            dt = datetime.fromisoformat(str(raw))
            return dt.strftime("%d/%m/%Y %H:%M")
        except (ValueError, TypeError):
            pass
    return "—"


def _badge_html(dom_flag: int, *, best: bool) -> str:
    """HTML d'un badge Domination/Humiliation ou chaîne vide."""
    if best and dom_flag == 1:
        label = html.escape(t("career_top_badge_domination"))
        return (
            f"<span style='padding:1px 6px;border-radius:3px;font-size:0.75em;"
            f"font-weight:600;background:{_BG_DOMINATION};color:{_FG_DOMINATION}'>"
            f"{label}</span>"
        )
    if not best and dom_flag == 2:
        label = html.escape(t("career_top_badge_humiliation"))
        return (
            f"<span style='padding:1px 6px;border-radius:3px;font-size:0.75em;"
            f"font-weight:600;background:{_BG_HUMILIATION};color:{_FG_HUMILIATION}'>"
            f"{label}</span>"
        )
    return ""


def _build_match_badge_legend_html(*, best: bool) -> str:
    """Légende inline pour le badge Domination ou Humiliation."""
    if best:
        label = html.escape(t("career_top_badge_domination"))
        text = html.escape(t("career_top_legend_domination"))
        bg, fg = _BG_DOMINATION, _FG_DOMINATION
    else:
        label = html.escape(t("career_top_badge_humiliation"))
        text = html.escape(t("career_top_legend_humiliation"))
        bg, fg = _BG_HUMILIATION, _FG_HUMILIATION
    badge = (
        f"<span style='padding:1px 6px;border-radius:3px;font-size:0.75em;"
        f"font-weight:600;background:{bg};color:{fg}'>{label}</span>"
    )
    return (
        f"<div style='font-size:0.78em;opacity:0.72;margin-top:4px;line-height:1.8;'>"
        f"{badge} {text}"
        f"</div>"
    )


def _build_top_table_html(rows: list[dict], *, best: bool) -> str:
    """Construit le tableau HTML pour le Top 10."""
    title = t("career_top_best_title") if best else t("career_top_worst_title")
    headers = [
        t("career_top_col_match_id"),
        t("career_top_col_date"),
        t("career_top_col_mode"),
        t("career_top_col_map"),
        t("career_top_col_score"),
        t("career_top_col_kda"),
        t("career_top_col_kd"),
        t("career_top_col_duration"),
        "",
    ]
    col_count = len(headers)
    head_cells = "".join(f"<th class='os-sb-th'>{html.escape(h)}</th>" for h in headers)

    body = []
    for row in rows:
        dom_flag = row.get("dominance_flag", 0) or 0
        badge = _badge_html(dom_flag, best=best)

        mid = str(row.get("match_id") or "")
        match_link = _match_id_link(mid)
        mode = html.escape(str(row.get("game_variant_name") or row.get("playlist_name") or "—"))

        score = html.escape(_format_score(row.get("my_team_score"), row.get("enemy_team_score")))

        kda = html.escape(_format_kda(row))
        kd = _kd_ratio(row)
        kd_c = _kd_color(row)
        kd_style = f" style='color:{kd_c};font-weight:700;'" if kd_c else ""
        duration = html.escape(_format_duration(row.get("time_played_seconds")))
        date_str = html.escape(_format_date(row.get("start_time")))
        map_td = map_name_cell_html(row.get("map_name")).replace("<td", "<td class='os-sb-td'", 1)

        body.append(
            f"<tr class='os-sb-row'>"
            f"<td class='os-sb-td'>{match_link}</td>"
            f"<td class='os-sb-td' style='white-space:nowrap'>{date_str}</td>"
            f"<td class='os-sb-td'>{mode}</td>"
            f"{map_td}"
            f"<td class='os-sb-td' style='font-weight:600'>{score}</td>"
            f"<td class='os-sb-td'>{kda}</td>"
            f"<td class='os-sb-td'{kd_style}>{html.escape(kd)}</td>"
            f"<td class='os-sb-td'>{duration}</td>"
            f"<td class='os-sb-td'>{badge}</td>"
            f"</tr>"
        )

    return (
        "<div class='os-table-wrap os-table-wrap--map-hover'>"
        f"<table class='os-table os-scoreboard' style='width:100%'>"
        f"<thead>"
        f"<tr><th class='os-sb-team' colspan='{col_count}'>{html.escape(title)}</th></tr>"
        f"<tr>{head_cells}</tr>"
        f"</thead>"
        f"<tbody>{''.join(body)}</tbody>"
        f"</table>"
        "</div>"
    )


def render_top_matches_section(*, db_path: str, xuid: str) -> None:
    """Rend la section Top 10 meilleurs / pires matchs dans la page Carrière."""
    st.subheader(t("career_top_matches_header"))

    best = load_top_best_matches(db_path, xuid)
    worst = load_top_worst_matches(db_path, xuid)
    logger.debug("Top matches chargés : %d best, %d worst", len(best), len(worst))

    if not best and not worst:
        st.info(t("career_top_no_data"))
        return

    tab_best, tab_worst = st.tabs([t("career_top_best_title"), t("career_top_worst_title")])
    with tab_best:
        if best:
            st.markdown(_build_top_table_html(best, best=True), unsafe_allow_html=True)
            st.markdown(_build_match_badge_legend_html(best=True), unsafe_allow_html=True)
        else:
            st.info(t("career_top_no_data"))

    with tab_worst:
        if worst:
            st.markdown(_build_top_table_html(worst, best=False), unsafe_allow_html=True)
            st.markdown(_build_match_badge_legend_html(best=False), unsafe_allow_html=True)
        else:
            st.info(t("career_top_no_data"))
