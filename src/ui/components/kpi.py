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


def render_compact_html_cards(
    cards: list[tuple[str, str, str | None, str | None]],
    *,
    dense: bool = False,
) -> None:
    """Affiche une grille de cartes KPI compactes avec support HTML des valeurs.

    Args:
        cards: Liste de tuples (label, value_html, kpi_color, sub_html).
            - label: titre de la carte (sera échappé).
            - value_html: valeur, peut contenir du HTML (spans de couleur, badges…).
            - kpi_color: couleur CSS inline optionnelle pour la valeur (ex. '#c62828').
            - sub_html: sous-texte HTML optionnel affiché sous la valeur.
        dense: Espacement réduit entre les cartes.
    """
    if not cards:
        return
    grid_class = "os-kpi-grid os-kpi-grid--dense" if dense else "os-kpi-grid"
    items = ""
    for label, value_html, kpi_color, sub_html in cards:
        safe_label = html_mod.escape(str(label))
        color_attr = f" style='color:{kpi_color}'" if kpi_color else ""
        sub_part = f"<div class='os-kpi__sub'>{sub_html}</div>" if sub_html else ""
        items += (
            f"<div class='os-kpi'>"
            f"<div class='os-kpi__label'>{safe_label}</div>"
            f"<div class='os-kpi__value'{color_attr}>{value_html}</div>"
            f"{sub_part}"
            f"</div>"
        )
    st.markdown(f"<div class='{grid_class}'>{items}</div>", unsafe_allow_html=True)


def render_top_summary(
    total_matches: int,
    rates,
    avg_duration: str | None = None,
    total_duration: str | None = None,
) -> None:
    """Affiche le résumé des parties sélectionnées avec les résultats.

    Args:
        total_matches: Nombre total de matchs sélectionnés.
        rates: Objet avec attributs wins, losses, ties, no_finish.
        avg_duration: Durée moyenne formatée (optionnel).
        total_duration: Durée totale formatée (optionnel).
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

    dur_html = ""
    if avg_duration:
        lbl = html_mod.escape(t("kpi_avg_duration"))
        val = html_mod.escape(avg_duration)
        dur_html += f"<div class='os-top-chip'><span class='os-top-chip__label'>{lbl}</span><span class='os-top-chip__value'>{val}</span></div>"
    if total_duration:
        lbl = html_mod.escape(t("kpi_total_duration"))
        val = html_mod.escape(total_duration)
        dur_html += f"<div class='os-top-chip'><span class='os-top-chip__label'>{lbl}</span><span class='os-top-chip__value'>{val}</span></div>"

    count_chip = (
        f"<div class='os-top-chip os-top-chip--count'>"
        f"<span class='os-top-chip__label'>{html_mod.escape(t('kpi_selected_matches'))}</span>"
        f"<span class='os-top-chip__value'>{total_matches}</span>"
        f"</div>"
    )
    st.markdown(
        "<div class='os-top-summary'>"
        f"  <div class='os-top-summary__chips'>"
        f"    {count_chip}"
        f"    {dur_html}"
        f"    <div class='os-top-chip os-top-chip--win'><span class='os-top-chip__label'>{t('kpi_wins')}</span><span class='os-top-chip__value'>{wins}</span></div>"
        f"    <div class='os-top-chip os-top-chip--loss'><span class='os-top-chip__label'>{t('kpi_losses')}</span><span class='os-top-chip__value'>{losses}</span></div>"
        f"    <div class='os-top-chip os-top-chip--tie'><span class='os-top-chip__label'>{t('kpi_ties')}</span><span class='os-top-chip__value'>{ties}</span></div>"
        f"    <div class='os-top-chip os-top-chip--nf'><span class='os-top-chip__label'>{t('kpi_no_finish')}</span><span class='os-top-chip__value'>{no_finish}</span></div>"
        "  </div>"
        "</div>",
        unsafe_allow_html=True,
    )
