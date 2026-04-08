"""Composants KPI (indicateurs clés de performance).

Ce module contient les fonctions pour afficher
des cartes KPI et résumés statistiques.
"""

from __future__ import annotations

import html as html_mod

import streamlit as st

from src.ui.i18n import t


def render_combined_kpi_cards(cards: list[dict]) -> None:
    """Affiche une grille de cases KPI avec valeur principale, sous-valeur et tendance vs all-time.

    Chaque dict contient :
        label (str)          — intitulé de la case
        main  (str)          — valeur principale (gros texte)
        sub   (str | None)   — sous-valeur (ex: "1.2/min"), optionnel
        trend (str)          — "above" | "near" | "below" | "none"
    """
    if not cards:
        return
    items = []
    for c in cards:
        trend = c.get("trend", "none")
        trend_class = f" os-kpi--{trend}" if trend != "none" else ""
        if c.get("wide"):
            trend_class += " os-kpi--wide"
        if c.get("bar"):
            segments = "".join(
                f"<div style='flex:{pct};background:{html_mod.escape(color)};height:100%'></div>"
                for color, pct, _ in c["bar"]
                if pct > 0
            )
            legend = " · ".join(
                f"<span style='color:{html_mod.escape(color)}'>{html_mod.escape(lbl)}</span>"
                for color, pct, lbl in c["bar"]
                if pct > 0
            )
            body_html = (
                f"<div class='os-kpi__bar-wrap'>"
                f"<div class='os-kpi__bar-track'>{segments}</div>"
                f"<div class='os-kpi__bar-legend'>{legend}</div>"
                f"</div>"
            )
        else:
            sub_html = ""
            if c.get("sub"):
                sub_html = f"<span class='os-kpi__subvalue'>{html_mod.escape(str(c['sub']))}</span>"
            body_html = (
                f"<div class='os-kpi__value-row'>"
                f"<span class='os-kpi__value'>{html_mod.escape(str(c['main']))}</span>"
                f"{sub_html}"
                f"</div>"
            )
        items.append(
            f"<div class='os-kpi{trend_class}'>"
            f"<div class='os-kpi__label'>{html_mod.escape(str(c['label']))}</div>"
            f"{body_html}"
            f"</div>"
        )
    st.markdown(
        f"<div class='os-kpi-grid'>{''.join(items)}</div>",
        unsafe_allow_html=True,
    )


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
