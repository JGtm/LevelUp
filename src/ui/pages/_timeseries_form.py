"""Section forme récente — onglet Résumé de la page Séries temporelles."""

from __future__ import annotations

import logging

import polars as pl
import streamlit as st

from src.analysis._performance_form import compute_form_score_history
from src.ui.chart_utils import safe_chart_render
from src.ui.components.browser_storage import hints_visible
from src.ui.i18n import get_lang, t
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, fragment_if_available
from src.visualization._form_score import plot_form_score_history

logger = logging.getLogger(__name__)


def _current_form_value(history_df: pl.DataFrame, highlight_ids: set[str]) -> float | None:
    """Retourne la valeur moyenne de form_score sur les matchs de la sélection."""
    if history_df.is_empty() or "form_score" not in history_df.columns or not highlight_ids:
        return None
    sub = history_df.filter(pl.col("match_id").cast(pl.Utf8).is_in(highlight_ids))
    if sub.is_empty():
        return None
    valid = sub["form_score"].drop_nulls()
    return float(valid.mean()) if len(valid) > 0 else None


def _baseline_form_value(history_df: pl.DataFrame, highlight_ids: set[str]) -> float | None:
    """Retourne la valeur moyenne de form_score juste avant la sélection (référence delta)."""
    if history_df.is_empty() or "form_score" not in history_df.columns or not highlight_ids:
        return None
    before = history_df.filter(~pl.col("match_id").cast(pl.Utf8).is_in(highlight_ids))
    if before.is_empty():
        return None
    valid = before["form_score"].drop_nulls()
    return float(valid.tail(14).mean()) if len(valid) > 0 else None


@fragment_if_available
def render_form_score_section(
    df_full: pl.DataFrame | None,
    dff: pl.DataFrame,
) -> None:
    """Affiche le score de forme individuel (rolling calculé sur l'historique, affiché sur la sélection).

    Le rolling est calculé sur df_full pour être significatif (14 vs 90 matchs),
    mais seuls les matchs de dff sont affichés dans le graphe. La métrique résume
    le form_score moyen sur la sélection courante.

    Args:
        df_full: Historique complet (toutes sessions, non filtré). Fallback sur dff si None.
        dff: Matchs de la sélection courante (filtrés par mode, date, etc.).
    """
    history_base = df_full if df_full is not None and not df_full.is_empty() else dff
    if history_base.is_empty() or "performance_score" not in history_base.columns:
        logger.debug("render_form_score_section: pas de données performance_score disponibles.")
        return

    # Calculer le rolling sur l'historique complet pour des valeurs significatives
    history_with_form = compute_form_score_history(history_base)
    if "form_score" not in history_with_form.columns:
        logger.debug("render_form_score_section: form_score absent après compute.")
        return

    # Extraire les match_ids de la sélection courante pour filtrer l'affichage
    dff_ids: set[str] = set()
    if "match_id" in dff.columns and not dff.is_empty():
        dff_ids = set(dff["match_id"].cast(pl.Utf8).to_list())

    # Filtrer l'affichage sur la sélection uniquement (pas toute l'histoire)
    if dff_ids and "match_id" in history_with_form.columns:
        display_df = history_with_form.filter(pl.col("match_id").cast(pl.Utf8).is_in(dff_ids))
    else:
        display_df = history_with_form

    if display_df.is_empty():
        logger.debug("render_form_score_section: aucun match à afficher après filtre dff.")
        return

    logger.debug(
        "render_form_score_section: historique=%d matchs, affiché=%d matchs.",
        len(history_with_form),
        len(display_df),
    )

    current_val = _current_form_value(history_with_form, dff_ids) if dff_ids else None
    baseline_val = _baseline_form_value(history_with_form, dff_ids) if dff_ids else None
    logger.debug(
        "render_form_score_section: current_val=%s baseline_val=%s.", current_val, baseline_val
    )

    st.subheader(t("ts_form_score_title"))
    if hints_visible():
        st.caption(t("ts_form_score_caption"))

    col_chart, col_kpi = st.columns([4, 1])

    with col_kpi:
        if current_val is not None:
            delta = round(current_val - baseline_val, 1) if baseline_val is not None else None
            st.metric(
                label=t("ts_form_score_current"),
                value=f"{current_val:+.1f}",
                delta=f"{delta:+.1f}" if delta is not None else None,
            )

    with col_chart, safe_chart_render():
        fig = plot_form_score_history(
            {"": display_df},
            highlight_match_ids=set(),
            lang=get_lang(),
            height=320,
        )
        if fig is not None:
            st.plotly_chart(fig, width="stretch", config=PLOTLY_CLEAN_CONFIG)
