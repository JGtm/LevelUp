"""Barre KPI compacte pour le shell v7."""

from __future__ import annotations

from html import escape

import streamlit as st

from src.app.kpis import compute_kpi_stats
from src.ui.formatting import format_duration_dhm
from src.ui.i18n import get_lang, t


def _format_ratio(value: float | None) -> str:
    """Formate un ratio avec un fallback neutre."""
    if value is None:
        return "-"
    return f"{value:.2f}"


def _format_percent(value: float | None) -> str:
    """Formate un pourcentage avec un fallback neutre."""
    if value is None:
        return "-"
    return f"{value:.0f}%"


def render_kpi_bar(dff) -> None:
    """Affiche la barre KPI compacte au-dessus des hubs analytiques."""
    kpis = compute_kpi_stats(dff)
    lang = get_lang()
    win_label = "V" if lang == "fr" else "W"

    items = [
        (t("lbl_parties"), str(kpis.total_matches)),
        (t("col_total_duration"), format_duration_dhm(kpis.total_play_seconds, lang=lang)),
        ("KD", _format_ratio(kpis.global_ratio)),
        (t("col_avg_accuracy"), _format_percent(kpis.avg_accuracy)),
        (win_label, _format_percent(kpis.win_rate * 100)),
    ]

    html = ["<div class='v7-kpi-bar'>"]
    for label, value in items:
        html.append(
            f"<span class='v7-kpi-item'>{escape(label)} <strong>{escape(value)}</strong></span>"
        )
    html.append("</div>")
    st.markdown("".join(html), unsafe_allow_html=True)
