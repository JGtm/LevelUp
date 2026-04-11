"""Section forme récente — onglet Résumé de la page Séries temporelles."""

from __future__ import annotations

import logging

import polars as pl
import streamlit as st

from src.analysis._performance_form import DETAIL_THRESHOLD, compute_form_score_history
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


def _load_bucket_series(
    history_with_form: pl.DataFrame,
    dff_ids: set[str],
    db_path: str,
    xuid: str,
) -> dict[str, pl.DataFrame] | None:
    """Charge et calcule les buckets intra-match si données disponibles."""
    from src.analysis._performance_form import compute_bucket_form_score
    from src.data.services._form_bucket_queries import load_bucket_data

    events_by_match, match_meta = load_bucket_data(db_path, xuid, list(dff_ids))
    if not events_by_match:
        return None
    bucket_df = compute_bucket_form_score(history_with_form, events_by_match, match_meta)
    if bucket_df.is_empty():
        return None
    logger.debug("_load_bucket_series: %d buckets générés.", len(bucket_df))
    return {"": bucket_df}


def _prepare_form_display(
    df_full: pl.DataFrame | None,
    dff: pl.DataFrame,
) -> tuple[pl.DataFrame, pl.DataFrame, set[str]] | None:
    """Prépare history_with_form, display_df et dff_ids. Retourne None si données absentes."""
    history_base = df_full if df_full is not None and not df_full.is_empty() else dff
    if history_base.is_empty() or "performance_score" not in history_base.columns:
        logger.debug("_prepare_form_display: pas de données performance_score.")
        return None
    history_with_form = compute_form_score_history(history_base)
    if "form_score" not in history_with_form.columns:
        logger.debug("_prepare_form_display: form_score absent après compute.")
        return None
    dff_ids: set[str] = set()
    if "match_id" in dff.columns and not dff.is_empty():
        dff_ids = set(dff["match_id"].cast(pl.Utf8).to_list())
    if dff_ids and "match_id" in history_with_form.columns:
        display_df = history_with_form.filter(pl.col("match_id").cast(pl.Utf8).is_in(dff_ids))
    else:
        display_df = history_with_form
    if display_df.is_empty():
        logger.debug("_prepare_form_display: aucun match après filtre dff.")
        return None
    logger.debug(
        "_prepare_form_display: historique=%d matchs, affiché=%d matchs.",
        len(history_with_form),
        len(display_df),
    )
    return history_with_form, display_df, dff_ids


@fragment_if_available
def render_form_score_section(
    df_full: pl.DataFrame | None,
    dff: pl.DataFrame,
    *,
    db_path: str | None = None,
    xuid: str | None = None,
) -> None:
    """Affiche le score de forme individuel (rolling calculé sur l'historique, affiché sur la sélection).

    Le rolling est calculé sur df_full pour être significatif (14 vs 90 matchs),
    mais seuls les matchs de dff sont affichés dans le graphe. La métrique résume
    le form_score moyen sur la sélection courante.

    Si la sélection contient ≤ DETAIL_THRESHOLD matchs ET que db_path/xuid sont fournis,
    active le mode détail : points bucket intra-match ancrés sur le form_score du match.

    Args:
        df_full: Historique complet (toutes sessions, non filtré). Fallback sur dff si None.
        dff: Matchs de la sélection courante (filtrés par mode, date, etc.).
        db_path: Chemin vers stats.duckdb du joueur (optionnel, pour mode détail).
        xuid: XUID du joueur (optionnel, pour mode détail).
    """
    prepared = _prepare_form_display(df_full, dff)
    if prepared is None:
        return
    history_with_form, display_df, dff_ids = prepared

    current_val = _current_form_value(history_with_form, dff_ids) if dff_ids else None
    baseline_val = _baseline_form_value(history_with_form, dff_ids) if dff_ids else None

    bucket_series: dict[str, pl.DataFrame] | None = None
    if len(display_df) <= DETAIL_THRESHOLD and bool(db_path) and bool(xuid):
        bucket_series = _load_bucket_series(history_with_form, dff_ids, db_path, xuid)  # type: ignore[arg-type]

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
            bucket_series_by_name=bucket_series,
        )
        if fig is not None:
            st.plotly_chart(fig, width="stretch", config=PLOTLY_CLEAN_CONFIG)
