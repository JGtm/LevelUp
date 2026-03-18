"""Graphiques par carte pour la page Coéquipiers.

Extrait de teammates_views.py pour respecter la limite de 500 lignes.
Contient le rendu des charts par carte : lollipop, timeline, bullet,
perf vs historique, heatmap escouade, vue carte 1 coéquipier.
"""

from __future__ import annotations

import logging

import polars as pl
import streamlit as st

logger = logging.getLogger(__name__)

from src.analysis import compute_map_breakdown
from src.ui.chart_utils import safe_chart_render
from src.ui.i18n import t
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, PLOTLY_STATIC_CONFIG
from src.visualization import (
    plot_map_outcome_timeline,
    plot_map_perf_vs_history,
    plot_map_winrate_bullet,
    plot_squad_map_heatmap,
)
from src.visualization._compat import DataFrameLike, ensure_polars


def render_map_charts_section(
    sub_all: DataFrameLike,
    full_squad_df: DataFrameLike,
    breakdown_all: DataFrameLike,
    lang: str,
) -> None:
    """Affiche lollipop + timeline + bullet + perf vs historique pour la vue multi-coéquipiers.

    Args:
        sub_all: Matchs filtrés avec les coéquipiers (sélection courante).
        full_squad_df: Tous les matchs avec les coéquipiers (historique complet).
        breakdown_all: Breakdown par carte déjà calculé.
        lang: Langue.
    """
    sub_pl = ensure_polars(sub_all)
    full_pl = ensure_polars(full_squad_df)
    bd_all = ensure_polars(breakdown_all)

    if bd_all.is_empty():
        logger.debug("render_map_charts_section: breakdown vide, affichage info")
        st.info(t("tm_not_enough_matches"))
        return

    view = bd_all.head(20).reverse()
    logger.debug("render_map_charts_section: %d cartes, %d matchs session", len(view), len(sub_pl))
    session_ids = sub_pl["match_id"].cast(pl.Utf8).to_list() if not sub_pl.is_empty() else []

    # Ordre chronologique des cartes (oldest first) depuis la sélection courante
    map_order: list[str] | None = None
    if not sub_pl.is_empty() and "start_time" in sub_pl.columns:
        map_order = (
            sub_pl.sort("start_time")
            .unique(subset=["map_name"], keep="first", maintain_order=True)
            .filter(pl.col("map_name").is_not_null())["map_name"]
            .to_list()
        )

    # B — Timeline (DISABLED: désactivé temporairement, conserver le code)
    if False:  # timeline disabled — conserver le code pour usage futur  # noqa: SIM210
        st.markdown(f"##### {t('tm_map_timeline_title')}")
        st.caption(t("tm_map_timeline_caption"))
        with safe_chart_render():
            fig_tl = plot_map_outcome_timeline(full_pl, session_match_ids=session_ids, lang=lang)
            if fig_tl is not None:
                st.plotly_chart(fig_tl, width="stretch", config=PLOTLY_CLEAN_CONFIG)

    # C — Bullet win rate vs historique
    if not full_pl.is_empty():
        bd_history = _compute_history_breakdown(full_pl)
        st.markdown(f"##### {t('tm_map_bullet_title')}")
        with safe_chart_render():
            fig_bullet = plot_map_winrate_bullet(view, bd_history, lang=lang, map_order=map_order)
            if fig_bullet is not None:
                st.plotly_chart(fig_bullet, width="stretch", config=PLOTLY_STATIC_CONFIG)

        # Feature 2 — Perf vs historique
        st.markdown(f"##### {t('tm_perf_vs_history_title')}")
        with safe_chart_render():
            fig_perf = plot_map_perf_vs_history(view, bd_history, lang=lang, map_order=map_order)
            if fig_perf is not None:
                st.plotly_chart(fig_perf, width="stretch", config=PLOTLY_STATIC_CONFIG)


def render_squad_heatmap(series: list[tuple[str, DataFrameLike]], lang: str) -> None:
    """Affiche la heatmap de performance par joueur × carte (Feature 1).

    Args:
        series: Liste de (nom_joueur, df_matchs).
        lang: Langue.
    """
    if len(series) < 2:
        return
    st.markdown(f"##### {t('tm_map_squad_heatmap_title')}")
    with safe_chart_render():
        fig_hm = plot_squad_map_heatmap(series, lang=lang)
        if fig_hm is not None:
            st.plotly_chart(fig_hm, width="stretch", config=PLOTLY_STATIC_CONFIG)


def render_single_map_section(
    sub: DataFrameLike,
    dfr: DataFrameLike,
    series: list[tuple[str, DataFrameLike]],
    lang: str,
) -> None:
    """Affiche les graphiques par carte pour la vue 1 coéquipier.

    Args:
        sub: Matchs filtrés avec le coéquipier (sélection courante).
        dfr: Tous les matchs avec le coéquipier (historique complet).
        series: [(me_name, sub), (friend_name, friend_sub)] pour la heatmap.
        lang: Langue.
    """
    sub_pl = ensure_polars(sub)
    dfr_pl = ensure_polars(dfr)
    if sub_pl.is_empty():
        return

    bd_current = compute_map_breakdown(sub_pl)
    if bd_current.is_empty():
        logger.debug("render_single_map_section: breakdown vide pour %d matchs", len(sub_pl))
        return

    st.subheader(t("tm_by_map"))
    view = bd_current.sort("win_rate").head(20).reverse()
    session_ids = sub_pl["match_id"].cast(pl.Utf8).to_list()

    # Ordre chronologique des cartes (oldest first) depuis la sélection courante
    map_order_single: list[str] | None = None
    if "start_time" in sub_pl.columns:
        map_order_single = (
            sub_pl.sort("start_time")
            .unique(subset=["map_name"], keep="first", maintain_order=True)
            .filter(pl.col("map_name").is_not_null())["map_name"]
            .to_list()
        )

    # B — Timeline (DISABLED: désactivé temporairement, conserver le code)
    if False:  # timeline disabled — conserver le code pour usage futur  # noqa: SIM210
        st.markdown(f"##### {t('tm_map_timeline_title')}")
        st.caption(t("tm_map_timeline_caption"))
        with safe_chart_render():
            fig_tl = plot_map_outcome_timeline(dfr_pl, session_match_ids=session_ids, lang=lang)
            if fig_tl is not None:
                st.plotly_chart(fig_tl, width="stretch", config=PLOTLY_CLEAN_CONFIG)

    # C + Feature 2 — vs historique (seulement si assez de données)
    if not dfr_pl.is_empty():
        bd_history = _compute_history_breakdown(dfr_pl)
        st.markdown(f"##### {t('tm_map_bullet_title')}")
        with safe_chart_render():
            fig_bullet = plot_map_winrate_bullet(
                view, bd_history, lang=lang, map_order=map_order_single
            )
            if fig_bullet is not None:
                st.plotly_chart(fig_bullet, width="stretch", config=PLOTLY_STATIC_CONFIG)

        # Feature 2 — Perf vs historique
        st.markdown(f"##### {t('tm_perf_vs_history_title')}")
        with safe_chart_render():
            fig_perf = plot_map_perf_vs_history(
                view, bd_history, lang=lang, map_order=map_order_single
            )
            if fig_perf is not None:
                st.plotly_chart(fig_perf, width="stretch", config=PLOTLY_STATIC_CONFIG)
    render_squad_heatmap(series, lang=lang)


def _compute_history_breakdown(full_df: pl.DataFrame) -> pl.DataFrame:
    """Calcule le breakdown historique complet (min_matches=1) depuis un DataFrame de matchs."""
    return compute_map_breakdown(full_df)
