"""Panneau de légende joueurs pour la page Coéquipiers.

Pour changer de stratégie de positionnement, modifier ``_PANEL_MODE`` :

- ``"fixed"``   → ``position:fixed`` côté droit, toujours visible en scrollant ✅ (défaut)
- ``"sidebar"`` → ``st.sidebar``, très simple à tester
- ``"hidden"``  → désactivé (debug / comparaison visuelle)
"""

from __future__ import annotations

import logging

import streamlit as st

from src.ui.i18n import get_lang

logger = logging.getLogger(__name__)

# ── Stratégie ─────────────────────────────────────────────────────────────────
# Changer cette valeur pour tester une autre approche sans toucher au reste.
_PANEL_MODE: str = "fixed"

# CSS du panneau fixe — modifier uniquement ce bloc pour ajuster l'apparence.
_FIXED_STYLE: str = (
    "position:fixed;"
    "right:1.2rem;"
    "top:50%;"
    "transform:translateY(-50%);"
    "z-index:999;"
    "background:rgba(14,17,23,0.92);"
    "border:1px solid rgba(255,255,255,0.14);"
    "border-radius:10px;"
    "padding:10px 14px;"
    "backdrop-filter:blur(6px);"
    "max-width:150px;"
    "font-size:13px;"
    "color:white;"
    "line-height:1.6;"
)

_LABEL_FR = "Joueurs"
_LABEL_EN = "Players"


def render_player_legend_panel(colors_by_name: dict[str, str]) -> None:
    """Injecte le panneau de légende joueurs selon la stratégie ``_PANEL_MODE``."""
    if not colors_by_name:
        logger.debug("render_player_legend_panel: colors_by_name vide, panneau ignoré")
        return

    label = _LABEL_FR if get_lang() == "fr" else _LABEL_EN

    if _PANEL_MODE == "hidden":
        logger.debug("render_player_legend_panel: mode 'hidden', panneau désactivé")
        return

    if _PANEL_MODE == "sidebar":
        logger.debug("render_player_legend_panel: mode 'sidebar', %d joueurs", len(colors_by_name))
        _render_sidebar(colors_by_name, label)
        return

    # "fixed" (défaut)
    logger.debug("render_player_legend_panel: mode 'fixed', %d joueurs", len(colors_by_name))
    _render_fixed(colors_by_name, label)


def _render_fixed(colors_by_name: dict[str, str], label: str) -> None:
    """Panneau ``position:fixed`` côté droit — toujours visible en scrollant."""
    dots = "".join(
        f'<div style="display:flex;align-items:center;gap:8px;margin:3px 0;">'
        f'<span style="width:11px;height:11px;border-radius:50%;background:{color};'
        f'display:inline-block;flex-shrink:0;"></span>'
        f'<span style="white-space:nowrap;overflow:hidden;text-overflow:ellipsis;'
        f'max-width:105px;" title="{name}">{name}</span>'
        f"</div>"
        for name, color in colors_by_name.items()
    )
    header = (
        f'<div style="font-weight:600;font-size:10px;text-transform:uppercase;'
        f'letter-spacing:.08em;color:rgba(255,255,255,.5);margin-bottom:6px;">'
        f"{label}</div>"
    )
    html = f'<div style="{_FIXED_STYLE}">{header}{dots}</div>'
    st.markdown(html, unsafe_allow_html=True)


def _render_sidebar(colors_by_name: dict[str, str], label: str) -> None:
    """Alternative sidebar — changer ``_PANEL_MODE = 'sidebar'`` pour tester."""
    st.sidebar.markdown("---")
    st.sidebar.markdown(f"**{label}**")
    for name, color in colors_by_name.items():
        dot = (
            f'<span style="background:{color};width:10px;height:10px;'
            f'border-radius:50%;display:inline-block;margin-right:6px;"></span>'
        )
        st.sidebar.markdown(f"{dot}{name}", unsafe_allow_html=True)
