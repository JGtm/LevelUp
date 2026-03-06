"""Tableau historique des parties — extrait de session_compare_charts.py."""

from __future__ import annotations

import html as html_lib

import polars as pl
import streamlit as st

from src.analysis.performance_score import compute_performance_series
from src.ui import translate_pair_name
from src.ui.components.performance import get_score_class
from src.ui.date_formats import FMT_DATETIME_FR_SHORT_YEAR
from src.ui.i18n import get_lang, get_outcome_map, get_weekdays, t
from src.ui.pages.session_compare_logic import outcome_class
from src.ui.vectorize_helpers import build_mapping
from src.visualization._compat import DataFrameLike, ensure_polars


def _build_history_dataframe(  # noqa: C901, PLR0912
    df_sess: DataFrameLike,
    df_full: DataFrameLike | None = None,
) -> tuple[pl.DataFrame, pl.Series | None]:
    """Construit le DataFrame d'affichage et les scores de performance.

    Prépare les colonnes à afficher (heure, mode, carte, frags, résultat, etc.)
    et calcule les scores de performance relatifs.

    Args:
        df_sess: DataFrame de la session (copie modifiable).
        df_full: DataFrame complet pour le calcul du score relatif.

    Returns:
        Tuple (df_display Polars préparé, Series Polars des scores de performance).
    """
    df_sess = ensure_polars(df_sess)
    if df_full is not None:
        df_full = ensure_polars(df_full)

    # Trier par start_time si disponible (pour l'ordre chronologique)
    if "start_time" in df_sess.columns:
        df_sess = df_sess.sort("start_time")

    # Traduire le mode si non traduit
    if "pair_fr" not in df_sess.columns and "pair_name" in df_sess.columns:
        _pair_map = build_mapping(df_sess["pair_name"], translate_pair_name)
        df_sess = df_sess.with_columns(
            pl.col("pair_name")
            .cast(pl.Utf8)
            .replace_strict(
                _pair_map, default=pl.col("pair_name").cast(pl.Utf8), return_dtype=pl.Utf8
            )
            .alias("pair_fr")
        )

    # Préparer les colonnes à afficher
    display_cols: list[str] = []
    col_map: dict[str, str] = {}

    if "start_time" in df_sess.columns:
        _lang = get_lang()
        _weekdays = get_weekdays(_lang)
        df_sess = df_sess.with_columns(
            (
                pl.col("start_time")
                .dt.weekday()
                .replace_strict(_weekdays, default="-", return_dtype=pl.Utf8)
                + pl.lit(" ")
                + pl.col("start_time").dt.strftime(FMT_DATETIME_FR_SHORT_YEAR)
            )
            .fill_null("-")
            .alias(t("col_time"))
        )
        display_cols.append(t("col_time"))

    if "pair_fr" in df_sess.columns:
        col_map["pair_fr"] = t("col_mode")
        display_cols.append("pair_fr")
    elif "pair_name" in df_sess.columns:
        _pair_map2 = build_mapping(df_sess["pair_name"], translate_pair_name)
        df_sess = df_sess.with_columns(
            pl.col("pair_name")
            .cast(pl.Utf8)
            .replace_strict(
                _pair_map2, default=pl.col("pair_name").cast(pl.Utf8), return_dtype=pl.Utf8
            )
            .alias("mode_traduit")
        )
        col_map["mode_traduit"] = t("col_mode")
        display_cols.append("mode_traduit")

    if "map_ui" in df_sess.columns:
        col_map["map_ui"] = t("col_map")
        display_cols.append("map_ui")
    elif "map_name" in df_sess.columns:
        col_map["map_name"] = t("col_map")
        display_cols.append("map_name")

    for c in ["kills", "deaths", "assists"]:
        if c in df_sess.columns:
            col_map[c] = {
                "kills": t("col_kills"),
                "deaths": t("col_deaths"),
                "assists": t("col_assists"),
            }[c]
            display_cols.append(c)

    if "outcome" in df_sess.columns:
        _outcome_map = get_outcome_map()
        df_sess = df_sess.with_columns(
            pl.col("outcome")
            .replace_strict(
                _outcome_map,
                default="-",
                return_dtype=pl.Utf8,
            )
            .fill_null("-")
            .alias(t("col_result"))
        )
        display_cols.append(t("col_result"))

    # Colonne Performance RELATIVE (après Résultat)
    history_df = df_full if df_full is not None else df_sess
    perf_series = compute_performance_series(df_sess, history_df)
    df_sess = df_sess.with_columns(perf_series.alias("Performance"))
    df_sess = df_sess.with_columns(
        pl.when(pl.col("Performance").is_not_null())
        .then(pl.col("Performance").round(0).cast(pl.Int64).cast(pl.Utf8))
        .otherwise(pl.lit("-"))
        .alias("Perf_display")
    )
    display_cols.append("Perf_display")
    col_map["Perf_display"] = t("col_performance")

    if "team_mmr" in df_sess.columns:
        df_sess = df_sess.with_columns(
            pl.when(pl.col("team_mmr").is_not_null())
            .then(pl.col("team_mmr").round(0).cast(pl.Int64).cast(pl.Utf8))
            .otherwise(pl.lit("-"))
            .alias(t("scc_mmr_team"))
        )
        display_cols.append(t("scc_mmr_team"))

    if "enemy_mmr" in df_sess.columns:
        df_sess = df_sess.with_columns(
            pl.when(pl.col("enemy_mmr").is_not_null())
            .then(pl.col("enemy_mmr").round(0).cast(pl.Int64).cast(pl.Utf8))
            .otherwise(pl.lit("-"))
            .alias(t("scc_mmr_enemy"))
        )
        display_cols.append(t("scc_mmr_enemy"))

    # Sélectionner et renommer les colonnes
    df_display = df_sess.select(display_cols).rename(col_map)

    # Garder les scores de performance pour la coloration
    perf_scores = df_sess.get_column("Performance") if "Performance" in df_sess.columns else None

    return df_display, perf_scores


def _render_history_html(
    df_display: pl.DataFrame,
    perf_scores: pl.Series | None = None,
) -> None:
    """Génère et affiche le tableau HTML stylisé des parties.

    Args:
        df_display: DataFrame Polars préparé pour l'affichage.
        perf_scores: Series Polars des scores de performance pour la coloration.
    """
    html_rows: list[str] = []
    for idx, row in enumerate(df_display.iter_rows(named=True)):
        cells: list[str] = []
        for col in df_display.columns:
            val = row[col]
            if col == t("col_result"):
                css_class = outcome_class(str(val))
                cells.append(f"<td class='{css_class}'>{html_lib.escape(str(val))}</td>")
            elif col == t("col_performance"):
                # Coloration selon le score
                perf_val = perf_scores[idx] if perf_scores is not None else None
                css_class = get_score_class(perf_val)
                cells.append(
                    f"<td class='{css_class}'>{html_lib.escape(str(val) if val is not None else '-')}</td>"
                )
            else:
                cells.append(f"<td>{html_lib.escape(str(val) if val is not None else '-')}</td>")
        html_rows.append("<tr>" + "".join(cells) + "</tr>")

    header_cells = "".join(f"<th>{html_lib.escape(c)}</th>" for c in df_display.columns)
    html_table = f"""
    <table class="session-history-table">
    <thead><tr>{header_cells}</tr></thead>
    <tbody>{"".join(html_rows)}</tbody>
    </table>
    """
    st.markdown(html_table, unsafe_allow_html=True)


def render_session_history_table(
    df_sess: DataFrameLike,
    session_name: str,
    df_full: DataFrameLike | None = None,
) -> None:
    """Affiche le tableau historique d'une session.

    Orchestre la construction du DataFrame d'affichage et le rendu HTML.

    Args:
        df_sess: DataFrame de la session.
        session_name: Nom de la session pour les messages.
        df_full: DataFrame complet pour le calcul du score relatif.
    """
    df_sess = ensure_polars(df_sess)
    if df_sess.is_empty():
        st.info(t("sc_no_matches_in_session", session_name=session_name))
        return

    if df_full is not None:
        df_full = ensure_polars(df_full)
    df_display, perf_scores = _build_history_dataframe(df_sess.clone(), df_full)
    _render_history_html(df_display, perf_scores)
