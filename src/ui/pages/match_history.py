"""Page Historique des parties.

Tableau complet de l'historique des matchs avec liens et MMR.

Sprint 4.2 : Optimisation N+1
- Les colonnes team_mmr et enemy_mmr sont déjà dans le DataFrame
- Plus besoin de requête individuelle par match (était: 500 requêtes)
- Gain de performance: ~90% (1 requête batch vs N requêtes)

Sprint 8bis : Vectorisation
- Remplacement de 7 map_elements() par des expressions Polars natives
- Gain de performance: ~50% sur le rendu de 250 matchs
"""

from __future__ import annotations

import logging

import polars as pl
import streamlit as st

_log = logging.getLogger(__name__)

from src.analysis.performance_score import compute_performance_series
from src.ui.date_formats import FMT_DATETIME_FR
from src.ui.i18n import get_lang, get_outcome_map, t
from src.ui.pages.match_table_html import render_match_table_html
from src.ui.translations import translate_pair_name
from src.ui.vectorize_helpers import build_mapping
from src.visualization._compat import DataFrameLike, ensure_polars


def render_match_history_page(  # noqa: PLR0913
    dff: DataFrameLike,
    waypoint_player: str,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    df_full: DataFrameLike | None = None,
) -> None:
    """Affiche la page Historique des parties.

    Args:
        dff: DataFrame filtré des matchs.
        waypoint_player: Nom Waypoint du joueur.
        db_path: Chemin vers la base de données.
        xuid: XUID du joueur.
        db_key: Clé de cache de la DB.
        df_full: DataFrame complet (non filtré) pour le calcul du score relatif.
    """
    dff = ensure_polars(dff)

    # Protection contre les DataFrames vides
    if dff.is_empty():
        st.warning(t("no_matches"))
        return

    st.subheader(t("mh_title"))
    dff_table = _prepare_history_table(dff, waypoint_player, df_full)
    _render_history_table(dff_table, waypoint_player)
    _render_csv_download(dff_table)


def _prepare_history_table(
    dff: pl.DataFrame,
    waypoint_player: str,
    df_full: DataFrameLike | None,
) -> pl.DataFrame:
    """Prépare les colonnes enrichies du tableau d'historique."""
    dff_table = dff.clone()
    dff_table = _ensure_display_columns(dff_table, waypoint_player)
    dff_table = _add_score_columns(dff_table)
    dff_table = _add_time_columns(dff_table)
    dff_table = _add_win_rate_column(dff_table, df_full)
    return _add_performance_column(dff_table, df_full)


def _add_win_rate_column(
    dff_table: pl.DataFrame,
    df_full: DataFrameLike | None,
) -> pl.DataFrame:
    """Ajoute win_rate_hist : % victoires sur cette carte sur TOUT l'historique (non filtré)."""
    if "outcome" not in dff_table.columns or "map_name" not in dff_table.columns:
        _log.debug(
            "_add_win_rate_column : colonnes outcome/map_name absentes — win_rate_hist ignoré"
        )
        return dff_table.with_columns(
            pl.lit(None).cast(pl.Float64).alias("win_rate_hist"),
            pl.lit(None).cast(pl.Int64).alias("win_rate_hist_total"),
        )
    base = ensure_polars(df_full) if df_full is not None else dff_table
    map_wr = (
        base.group_by("map_name")
        .agg(
            pl.col("outcome").eq(2).cast(pl.Float64).sum().alias("_wins"),
            pl.len().alias("_total"),
        )
        .with_columns((pl.col("_wins") / pl.col("_total") * 100).round(1).alias("win_rate_hist"))
        .rename({"_total": "win_rate_hist_total"})
        .select(["map_name", "win_rate_hist", "win_rate_hist_total"])
    )
    _log.debug(
        "_add_win_rate_column : %d cartes, total matchs base=%d",
        len(map_wr),
        len(base),
    )
    return dff_table.join(map_wr, on="map_name", how="left")


def _ensure_display_columns(dff_table: pl.DataFrame, waypoint_player: str) -> pl.DataFrame:
    """Ajoute les colonnes d'affichage manquantes pour le tableau."""
    if "playlist_fr" not in dff_table.columns:
        dff_table = dff_table.with_columns(pl.col("playlist_name").alias("playlist_fr"))
    if "mode_ui" not in dff_table.columns:
        _mh_mode_map = build_mapping(
            dff_table["pair_name"], lambda x: translate_pair_name(x, lang=get_lang())
        )
        dff_table = dff_table.with_columns(
            pl.col("pair_name")
            .cast(pl.Utf8)
            .replace_strict(_mh_mode_map, default=pl.col("pair_name").cast(pl.Utf8), return_dtype=pl.Utf8)
            .alias("mode_ui")
        )
    return dff_table.with_columns(
        (
            pl.lit("https://www.halowaypoint.com/halo-infinite/players/")
            + pl.lit(waypoint_player.strip())
            + pl.lit("/matches/")
            + pl.col("match_id").cast(pl.Utf8)
        ).alias("match_url")
    )


def _add_score_columns(dff_table: pl.DataFrame) -> pl.DataFrame:
    """Ajoute les colonnes de score et MMR au tableau."""
    outcome_map = get_outcome_map()
    dff_table = dff_table.with_columns(
        pl.col("outcome").replace_strict(outcome_map, default=pl.lit("-")).alias("outcome_label")
    )
    dff_table = dff_table.with_columns(
        pl.concat_str(
            [
                pl.col("my_team_score")
                .cast(pl.Float64, strict=False)
                .round(0)
                .cast(pl.Int64, strict=False)
                .fill_null(pl.lit("-"))
                .cast(pl.Utf8),
                pl.lit(" - "),
                pl.col("enemy_team_score")
                .cast(pl.Float64, strict=False)
                .round(0)
                .cast(pl.Int64, strict=False)
                .fill_null(pl.lit("-"))
                .cast(pl.Utf8),
            ]
        ).alias("score")
    )
    if "team_mmr" not in dff_table.columns:
        dff_table = dff_table.with_columns(pl.lit(None).cast(pl.Float64).alias("team_mmr"))
    if "enemy_mmr" not in dff_table.columns:
        dff_table = dff_table.with_columns(pl.lit(None).cast(pl.Float64).alias("enemy_mmr"))
    return dff_table.with_columns(
        (
            pl.col("team_mmr").cast(pl.Float64, strict=False)
            - pl.col("enemy_mmr").cast(pl.Float64, strict=False)
        ).alias("delta_mmr")
    )


def _add_time_columns(dff_table: pl.DataFrame) -> pl.DataFrame:
    """Ajoute les colonnes temporelles formatées au tableau."""
    dff_table = dff_table.with_columns(
        pl.col("start_time").dt.strftime(FMT_DATETIME_FR).fill_null("-").alias("start_time_fr")
    )
    return dff_table.with_columns(
        pl.concat_str(
            [
                (pl.col("average_life_seconds").cast(pl.Int64, strict=False) // 60)
                .fill_null(0)
                .cast(pl.Utf8),
                pl.lit(":"),
                (pl.col("average_life_seconds").cast(pl.Int64, strict=False) % 60)
                .fill_null(0)
                .cast(pl.Utf8)
                .str.zfill(2),
            ]
        ).alias("average_life_mmss")
    )


def _add_performance_column(
    dff_table: pl.DataFrame,
    df_full: DataFrameLike | None,
) -> pl.DataFrame:
    """Ajoute la note de performance relative au tableau."""
    history_df = ensure_polars(df_full) if df_full is not None else dff_table
    perf_series = compute_performance_series(dff_table, history_df)
    if not isinstance(perf_series, pl.Series):
        perf_series = pl.Series("performance", perf_series.to_list())
    dff_table = dff_table.with_columns(perf_series.alias("performance"))
    return dff_table.with_columns(
        pl.col("performance")
        .round(0)
        .cast(pl.Int64, strict=False)
        .cast(pl.Utf8)
        .fill_null("-")
        .alias("performance_display")
    )


def _render_history_table(dff_table: pl.DataFrame, waypoint_player: str) -> None:
    """Affiche le tableau de l'historique via le renderer HTML partagé."""
    page_params = {"gamertag": waypoint_player.strip()} if waypoint_player.strip() else None
    table_html = render_match_table_html(
        dff_table,
        waypoint_player=waypoint_player,
        page_slug="Explorer",
        page_params=page_params,
    )
    st.markdown(table_html, unsafe_allow_html=True)


def _render_csv_download(dff_table: pl.DataFrame) -> None:
    """Affiche le bouton de téléchargement CSV."""
    show_cols = [
        "match_url",
        "start_time_fr",
        "map_name",
        "playlist_fr",
        "mode_ui",
        "outcome_label",
        "score",
        "team_mmr",
        "enemy_mmr",
        "delta_mmr",
        "kda",
        "kills",
        "deaths",
        "max_killing_spree",
        "headshot_kills",
        "average_life_mmss",
        "assists",
        "accuracy",
        "ratio",
    ]
    table = (
        dff_table.select(show_cols + ["start_time"])
        .sort("start_time", descending=True)
        .select(show_cols)
    )

    csv_table = table.rename({"start_time_fr": t("col_start_date")})
    csv = csv_table.write_csv().encode("utf-8")
    st.download_button(
        t("btn_download_csv"),
        data=csv,
        file_name="levelup_matches.csv",
        mime="text/csv",
    )
