"""Chips de filtres actives pour le shell v7."""

from __future__ import annotations

from datetime import date
from html import escape
from typing import Any

import streamlit as st

from src.app.session_keys import SK
from src.ui.i18n import t


def _summarize_values(values: Any, max_items: int = 2) -> str | None:
    """Résume une sélection de filtres en une chaîne compacte."""
    if values is None:
        return None
    if isinstance(values, str):
        cleaned = values.strip()
        return cleaned or None
    if not isinstance(values, list | tuple):
        cleaned = str(values).strip()
        return cleaned or None

    items = [str(value).strip() for value in values if str(value).strip()]
    if not items:
        return None
    if len(items) <= max_items:
        return ", ".join(items)
    remaining = len(items) - max_items
    return f"{', '.join(items[:max_items])} +{remaining}"


def _format_period(start_value: Any, end_value: Any) -> str | None:
    """Formate une période en texte court."""
    if not start_value and not end_value:
        return None

    def _fmt(value: Any) -> str | None:
        if value is None:
            return None
        if isinstance(value, date):
            return value.strftime("%d/%m/%Y")
        text = str(value).strip()
        return text or None

    start_txt = _fmt(start_value)
    end_txt = _fmt(end_value)
    if start_txt and end_txt:
        return f"{start_txt} -> {end_txt}"
    return start_txt or end_txt


def get_active_filter_chips() -> list[tuple[str, str]]:
    """Retourne les filtres actifs sous forme de couples (label, valeur)."""
    chips: list[tuple[str, str]] = []
    mappings = [
        (SK.FILTER_PLAYLISTS, t("v7_chip_playlists")),
        (SK.FILTER_MODES, t("v7_chip_modes")),
        (SK.FILTER_MAPS, t("v7_chip_maps")),
        (SK.PICKED_SESSIONS, t("v7_chip_sessions")),
    ]

    for state_key, label in mappings:
        summary = _summarize_values(st.session_state.get(state_key))
        if summary:
            chips.append((label, summary))

    period = _format_period(
        st.session_state.get(SK.START_DATE),
        st.session_state.get(SK.END_DATE),
    )
    if period:
        chips.append((t("v7_chip_period"), period))

    scope = _summarize_values(st.session_state.get(SK.FILTER_MODE))
    if scope:
        chips.append((t("v7_chip_scope"), scope))

    return chips


def render_filter_chips() -> int:
    """Affiche les chips des filtres actifs.

    Returns:
        Nombre de chips rendues.
    """
    chips = get_active_filter_chips()
    if not chips:
        st.markdown(
            f"<div class='v7-inline-note'>{escape(t('v7_filters_none'))}</div>",
            unsafe_allow_html=True,
        )
        return 0

    html = ["<div class='v7-filter-chips'>"]
    for label, value in chips:
        html.append(
            "<span class='v7-chip'>"
            f"<strong>{escape(label)}</strong>"
            f"<span>{escape(value)}</span>"
            "</span>"
        )
    html.append("</div>")
    st.markdown("".join(html), unsafe_allow_html=True)
    return len(chips)
