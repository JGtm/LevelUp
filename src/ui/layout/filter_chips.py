"""Chips de filtres actives pour le shell v7."""

from __future__ import annotations

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


def _summarize_collection(values: Any, max_items: int = 3) -> str | None:
    """Résume une collection (set, list, tuple) en texte compact."""
    if not values:
        return None
    items = sorted(str(v).strip() for v in values if str(v).strip())
    if not items:
        return None
    if len(items) <= max_items:
        return ", ".join(items)
    remaining = len(items) - max_items
    return f"{', '.join(items[:max_items])} +{remaining}"


def _is_dimension_filter_active(
    mode_key: str,
    excl_key: str,
    ss_key: str,
) -> tuple[bool, str | None]:
    """Détermine si un filtre dimension est actif et construit son résumé.

    Un filtre est actif quand :
    - mode "exclude" ET des exclusions explicites existent
    - mode "include" ET le set sélectionné n'est pas vide (sélection manuelle)

    Returns:
        (actif, résumé) — résumé est None si inactif.
    """
    mode = str(st.session_state.get(mode_key) or "exclude")
    if mode == "exclude":
        exclusions: set = st.session_state.get(excl_key) or set()
        if not exclusions:
            return False, None
        summary = _summarize_collection(exclusions)
        return True, summary
    else:
        selected = st.session_state.get(ss_key)
        if not selected:
            return False, None
        summary = _summarize_collection(
            selected if isinstance(selected, (set, list, tuple)) else [selected]
        )
        return bool(summary), summary


def get_active_filter_chips() -> list[tuple[str, str]]:
    """Retourne les filtres actifs sous forme de couples (label, valeur).

    N'affiche pas de chip quand un filtre dimension est à "tout sélectionné"
    (exclusions vides = aucune contrainte), ni la chip Scope (redondante avec
    la caption du bandeau).
    """
    chips: list[tuple[str, str]] = []

    dimension_configs = [
        (
            SK.FILTER_PLAYLISTS,
            "_playlists_filter_mode",
            "_playlists_exclusions",
            t("v7_chip_playlists"),
        ),
        (SK.FILTER_MODES, "_modes_filter_mode", "_modes_exclusions", t("v7_chip_modes")),
        (SK.FILTER_MAPS, "_maps_filter_mode", "_maps_exclusions", t("v7_chip_maps")),
    ]
    for ss_key, mode_key, excl_key, label in dimension_configs:
        active, summary = _is_dimension_filter_active(mode_key, excl_key, ss_key)
        if active and summary:
            chips.append((label, summary))

    picked_sessions = st.session_state.get(SK.PICKED_SESSIONS)
    if picked_sessions:
        summary = (
            _summarize_collection(picked_sessions)
            if isinstance(picked_sessions, (set, list, tuple))
            else _summarize_values(picked_sessions)
        )
        if summary:
            chips.append((t("v7_chip_sessions"), summary))

    # Période et Scope (Période/Sessions) ne sont pas tracées en chips L2 :
    # - la période est toujours initialisée aux bornes du dataset (sidebar)
    # - le mode est déjà visible via le segmented_control du bandeau

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
