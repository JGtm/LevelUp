"""Section d'evolution du rating (LUSR/CSR) pour la page timeseries."""

import polars as pl
import streamlit as st

from src.ui.chart_utils import safe_chart_render
from src.ui.i18n import t
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG

_MATCH_LABEL_THRESHOLD = 50  # En dessous : #N + carte ; au-dessus : regroupement semaine


def render_skill_rank_progression(
    dff: pl.DataFrame,
    db_path: str | None,
    xuid: str | None,
    lang: str = "fr",
) -> None:
    """Affiche l'evolution LUSR/CSR filtree sur les matchs courants."""
    if not db_path or not xuid:
        return

    from src.ui.pages.career_data import _load_lusr_history
    from src.visualization.timeseries_combat import plot_lusr_timeseries

    history_all = _load_lusr_history(db_path, xuid)
    if not history_all:
        return

    df_hist = pl.DataFrame(history_all)
    if "match_id" in dff.columns:
        df_hist = df_hist.filter(pl.col("match_id").is_in(dff["match_id"].cast(pl.Utf8).to_list()))
    if df_hist.is_empty():
        return

    types_av = (
        df_hist["rating_type"].drop_nulls().unique().to_list()
        if "rating_type" in df_hist.columns
        else []
    )
    has_lusr, has_csr = "LUSR" in types_av, "CSR" in types_av
    if not has_lusr and not has_csr:
        return

    st.subheader(t("ts_skill_rank_evolution"))
    if has_lusr and has_csr:
        sel = st.radio(
            t("ts_skill_rank_type"),
            ["LUSR", "CSR"],
            horizontal=True,
            key="ts_skill_rank_type_radio",
        )
        df_plot = df_hist.filter(pl.col("rating_type") == sel)
    else:
        sel = "LUSR" if has_lusr else "CSR"
        df_plot = df_hist

    if (
        len(df_plot) <= _MATCH_LABEL_THRESHOLD
        and "map_name" in dff.columns
        and "match_id" in dff.columns
    ):
        map_ref = dff.select(["match_id", "map_name"]).with_columns(
            pl.col("match_id").cast(pl.Utf8)
        )
        df_plot = df_plot.join(map_ref, on="match_id", how="left")
    elif len(df_plot) > _MATCH_LABEL_THRESHOLD:
        df_plot = (
            df_plot.with_columns(pl.col("start_time").dt.truncate("1w").alias("_bucket"))
            .sort("start_time", descending=True)
            .unique(subset=["_bucket"], keep="first")
            .drop("_bucket")
            .sort("start_time")
        )

    with safe_chart_render():
        fig = plot_lusr_timeseries(
            df_plot, title=sel, playlist_group=None, show_smooth=False, lang=lang
        )
        st.plotly_chart(fig, width="stretch", config=PLOTLY_CLEAN_CONFIG)
