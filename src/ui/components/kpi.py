"""Composants KPI (indicateurs clés de performance).

Ce module contient les fonctions pour afficher
des cartes KPI et résumés statistiques.
"""

from __future__ import annotations

import html as html_mod

import streamlit as st

from src.ui.i18n import t


def render_kpi_cards(cards: list[tuple[str, str]], *, dense: bool = True) -> None:
    """Affiche une grille de cartes KPI.

    Args:
        cards: Liste de tuples (label, valeur) à afficher.
        dense: Mode compact (défaut: True).
    """
    if not cards:
        return
    grid_class = "os-kpi-grid os-kpi-grid--dense" if dense else "os-kpi-grid"
    items = "".join(
        f"<div class='os-kpi'><div class='os-kpi__label'>{html_mod.escape(str(label))}</div>"
        f"<div class='os-kpi__value'>{html_mod.escape(str(value))}</div></div>"
        for (label, value) in cards
    )
    st.markdown(f"<div class='{grid_class}'>{items}</div>", unsafe_allow_html=True)


def render_top_summary(total_matches: int, rates) -> None:
    """Affiche le résumé des parties sélectionnées avec les résultats.

    Args:
        total_matches: Nombre total de matchs sélectionnés.
        rates: Objet avec attributs wins, losses, ties, no_finish.
    """
    if total_matches <= 0:
        st.markdown(
            "<div class='os-top-summary'>"
            f"  <div class='os-top-summary__empty'>{t('kpi_no_matches')}</div>"
            "</div>",
            unsafe_allow_html=True,
        )
        return

    wins = int(getattr(rates, "wins", 0) or 0)
    losses = int(getattr(rates, "losses", 0) or 0)
    ties = int(getattr(rates, "ties", 0) or 0)
    no_finish = int(getattr(rates, "no_finish", 0) or 0)

    st.markdown(
        "<div class='os-top-summary'>"
        "  <div class='os-top-summary__row'>"
        "    <div class='os-top-summary__left'>"
        f"      <div class='os-top-summary__kicker'>{t('kpi_selected_matches')}</div>"
        f"      <div class='os-top-summary__count'>{total_matches}</div>"
        "    </div>"
        "    <div class='os-top-summary__chips'>"
        f"      <div class='os-top-chip os-top-chip--win'><span class='os-top-chip__label'>{t('kpi_wins')}</span><span class='os-top-chip__value'>{wins}</span></div>"
        f"      <div class='os-top-chip os-top-chip--loss'><span class='os-top-chip__label'>{t('kpi_losses')}</span><span class='os-top-chip__value'>{losses}</span></div>"
        f"      <div class='os-top-chip os-top-chip--tie'><span class='os-top-chip__label'>{t('kpi_ties')}</span><span class='os-top-chip__value'>{ties}</span></div>"
        f"      <div class='os-top-chip os-top-chip--nf'><span class='os-top-chip__label'>{t('kpi_no_finish')}</span><span class='os-top-chip__value'>{no_finish}</span></div>"
        "    </div>"
        "  </div>"
        "</div>",
        unsafe_allow_html=True,
    )
