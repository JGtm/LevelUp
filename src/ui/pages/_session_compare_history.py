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
from src.ui.pages.win_loss_table_style import map_name_cell_html
from src.ui.vectorize_helpers import build_mapping
from src.visualization._compat import DataFrameLike, ensure_polars


def _build_history_dataframe(
    df_sess: DataFrameLike,
    df_full: DataFrameLike | None = None,
) -> tuple[pl.DataFrame, pl.Series | None, pl.Series | None]:
    """Construit le DataFrame d'affichage et les scores de performance.

    Prépare les colonnes à afficher (heure, mode, carte, frags, résultat, etc.)
    et calcule les scores de performance relatifs.

    Args:
        df_sess: DataFrame de la session (copie modifiable).
        df_full: DataFrame complet pour le calcul du score relatif.

    Returns:
        Tuple (df_display Polars préparé, Series scores de performance, Series map_id).
    """
    df_sess = _prepare_history_source(df_sess)
    df_full_pl = ensure_polars(df_full) if df_full is not None else None
    df_sess, display_cols, col_map = _build_display_metadata(df_sess)
    df_sess, perf_scores = _add_performance_display(df_sess, df_full_pl)
    display_cols.append("Perf_display")
    col_map["Perf_display"] = t("col_performance")
    df_sess, display_cols = _add_mmr_display_columns(df_sess, display_cols)
    map_ids = df_sess.get_column("map_id") if "map_id" in df_sess.columns else None
    df_display = df_sess.select(display_cols).rename(col_map)
    return df_display, perf_scores, map_ids


def _prepare_history_source(df_sess: DataFrameLike) -> pl.DataFrame:
    """Normalise la source d'historique et prépare la traduction de mode."""
    df_pl = ensure_polars(df_sess)
    if "start_time" in df_pl.columns:
        df_pl = df_pl.sort("start_time")
    if "pair_fr" in df_pl.columns or "mode_ui" in df_pl.columns or "pair_name" not in df_pl.columns:
        return df_pl
    if "pair_name_fr" in df_pl.columns:
        return df_pl.with_columns(
            pl.coalesce([pl.col("pair_name_fr"), pl.col("pair_name").cast(pl.Utf8)]).alias("pair_fr")
        )
    pair_map = build_mapping(df_pl["pair_name"], lambda x: translate_pair_name(x, lang=get_lang()))
    return df_pl.with_columns(
        pl.col("pair_name")
        .cast(pl.Utf8)
        .replace_strict(pair_map, default=pl.col("pair_name").cast(pl.Utf8), return_dtype=pl.Utf8)
        .alias("pair_fr")
    )


def _build_display_metadata(
    df_sess: pl.DataFrame,
) -> tuple[pl.DataFrame, list[str], dict[str, str]]:
    """Construit les colonnes affichées et leurs labels traduits."""
    display_cols: list[str] = []
    col_map: dict[str, str] = {}
    df_sess = _add_time_display_column(df_sess, display_cols)
    df_sess = _add_mode_display_column(df_sess, display_cols, col_map)
    _add_map_display_column(df_sess, display_cols, col_map)
    _add_stat_display_columns(df_sess, display_cols, col_map)
    df_sess = _add_outcome_display_column(df_sess, display_cols)
    return df_sess, display_cols, col_map


def _add_time_display_column(df_sess: pl.DataFrame, display_cols: list[str]) -> pl.DataFrame:
    """Ajoute la colonne horaire formatée si disponible."""
    if "start_time" not in df_sess.columns:
        return df_sess
    weekdays = get_weekdays(get_lang())
    time_col = t("col_time")
    display_cols.append(time_col)
    return df_sess.with_columns(
        (
            pl.col("start_time")
            .dt.weekday()
            .replace_strict(weekdays, default="-", return_dtype=pl.Utf8)
            + pl.lit(" ")
            + pl.col("start_time").dt.strftime(FMT_DATETIME_FR_SHORT_YEAR)
        )
        .fill_null("-")
        .alias(time_col)
    )


def _add_mode_display_column(
    df_sess: pl.DataFrame,
    display_cols: list[str],
    col_map: dict[str, str],
) -> pl.DataFrame:
    """Ajoute la colonne mode à afficher."""
    if "mode_ui" in df_sess.columns:
        col_map["mode_ui"] = t("col_mode")
        display_cols.append("mode_ui")
        return df_sess
    if "pair_fr" in df_sess.columns:
        col_map["pair_fr"] = t("col_mode")
        display_cols.append("pair_fr")
        return df_sess
    if "pair_name" not in df_sess.columns:
        return df_sess
    pair_map = build_mapping(df_sess["pair_name"], lambda x: translate_pair_name(x, lang=get_lang()))
    df_sess = df_sess.with_columns(
        pl.col("pair_name")
        .cast(pl.Utf8)
        .replace_strict(pair_map, default=pl.col("pair_name").cast(pl.Utf8), return_dtype=pl.Utf8)
        .alias("mode_traduit")
    )
    col_map["mode_traduit"] = t("col_mode")
    display_cols.append("mode_traduit")
    return df_sess


def _add_map_display_column(
    df_sess: pl.DataFrame,
    display_cols: list[str],
    col_map: dict[str, str],
) -> None:
    """Ajoute la colonne carte à afficher."""
    if "map_ui" in df_sess.columns:
        col_map["map_ui"] = t("col_map")
        display_cols.append("map_ui")
    elif "map_name" in df_sess.columns:
        col_map["map_name"] = t("col_map")
        display_cols.append("map_name")


def _add_stat_display_columns(
    df_sess: pl.DataFrame,
    display_cols: list[str],
    col_map: dict[str, str],
) -> None:
    """Ajoute les stats K/D/A disponibles."""
    for column_name, label in {
        "kills": t("col_kills"),
        "deaths": t("col_deaths"),
        "assists": t("col_assists"),
    }.items():
        if column_name in df_sess.columns:
            col_map[column_name] = label
            display_cols.append(column_name)


def _add_outcome_display_column(df_sess: pl.DataFrame, display_cols: list[str]) -> pl.DataFrame:
    """Ajoute la colonne résultat traduite."""
    if "outcome" not in df_sess.columns:
        return df_sess
    result_col = t("col_result")
    display_cols.append(result_col)
    return df_sess.with_columns(
        pl.col("outcome")
        .replace_strict(get_outcome_map(), default="-", return_dtype=pl.Utf8)
        .fill_null("-")
        .alias(result_col)
    )


def _add_performance_display(
    df_sess: pl.DataFrame,
    df_full: pl.DataFrame | None,
) -> tuple[pl.DataFrame, pl.Series | None]:
    """Ajoute les colonnes de performance relative et retourne les scores bruts."""
    history_df = df_full if df_full is not None else df_sess
    perf_series = compute_performance_series(df_sess, history_df)
    df_sess = df_sess.with_columns(perf_series.alias("Performance"))
    df_sess = df_sess.with_columns(
        pl.when(pl.col("Performance").is_not_null())
        .then(pl.col("Performance").round(0).cast(pl.Int64).cast(pl.Utf8))
        .otherwise(pl.lit("-"))
        .alias("Perf_display")
    )
    perf_scores = df_sess.get_column("Performance") if "Performance" in df_sess.columns else None
    return df_sess, perf_scores


def _add_mmr_display_columns(
    df_sess: pl.DataFrame, display_cols: list[str]
) -> tuple[pl.DataFrame, list[str]]:
    """Ajoute les colonnes MMR formatées si présentes."""
    for source_col, label in (("team_mmr", t("scc_mmr_team")), ("enemy_mmr", t("scc_mmr_enemy"))):
        if source_col not in df_sess.columns:
            continue
        df_sess = df_sess.with_columns(
            pl.when(pl.col(source_col).is_not_null())
            .then(pl.col(source_col).round(0).cast(pl.Int64).cast(pl.Utf8))
            .otherwise(pl.lit("-"))
            .alias(label)
        )
        display_cols.append(label)
    return df_sess, display_cols


def _render_history_html(
    df_display: pl.DataFrame,
    perf_scores: pl.Series | None = None,
    map_ids: pl.Series | None = None,
) -> None:
    """Génère et affiche le tableau HTML stylisé des parties.

    Args:
        df_display: DataFrame Polars préparé pour l'affichage.
        perf_scores: Series Polars des scores de performance pour la coloration.
        map_ids: Series Polars des map_id pour les thumbnails et traductions FR.
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
            elif col == t("col_map"):
                map_id = map_ids[idx] if map_ids is not None else None
                cells.append(map_name_cell_html(val, map_id))
            else:
                cells.append(f"<td>{html_lib.escape(str(val) if val is not None else '-')}</td>")
        html_rows.append("<tr>" + "".join(cells) + "</tr>")

    header_cells = "".join(f"<th>{html_lib.escape(c)}</th>" for c in df_display.columns)
    html_table = f"""
    <div class="os-table-wrap os-table-wrap--map-hover">
    <table class="session-history-table">
    <thead><tr>{header_cells}</tr></thead>
    <tbody>{"".join(html_rows)}</tbody>
    </table>
    </div>
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
    df_display, perf_scores, map_ids = _build_history_dataframe(df_sess.clone(), df_full)
    _render_history_html(df_display, perf_scores, map_ids)
