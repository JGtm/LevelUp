"""Section heatmap d'intensité pour la page Timeseries."""

from __future__ import annotations

import logging

import polars as pl
import streamlit as st

from src.ui.chart_utils import safe_chart_render
from src.ui.components.browser_storage import hints_visible
from src.ui.i18n import t
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, fragment_if_available
from src.visualization._plot_options import PlotOptions
from src.visualization.match_intensity_heatmap import plot_match_intensity_heatmap

logger = logging.getLogger(__name__)


def _reorder_profile_by_date(
    profile: object,
    dff: pl.DataFrame,
) -> object:
    """Réordonne le profil d'intensité par date de match."""
    from src.analysis.match_intensity import IntensityProfile

    if "start_time" not in dff.columns:
        return profile
    order = dff.sort("start_time")["match_id"].cast(pl.Utf8).to_list()
    order_map = {mid: i for i, mid in enumerate(order)}
    profile_with_order = (
        profile.df.with_columns(
            pl.col("match_id")
            .replace_strict(order_map, default=9999, return_dtype=pl.Int32)
            .alias("_order")
        )
        .sort("_order")
        .drop("_order")
    )
    return IntensityProfile(df=profile_with_order, n_buckets=profile.n_buckets)


def _apply_outcome_filter(
    profile: object,
    dff: pl.DataFrame,
    outcome_filter: str | None,
    filter_opts: list[str],
) -> object:
    """Filtre le profil par résultat (victoires/défaites)."""
    from src.analysis.match_intensity import IntensityProfile

    if not outcome_filter or "outcome" not in dff.columns:
        return profile
    if outcome_filter == filter_opts[1]:
        ids = dff.filter(pl.col("outcome") == 2)["match_id"].cast(pl.Utf8).to_list()
        return IntensityProfile(
            df=profile.df.filter(pl.col("match_id").is_in(ids)), n_buckets=profile.n_buckets
        )
    if outcome_filter == filter_opts[2]:
        ids = dff.filter(pl.col("outcome") == 3)["match_id"].cast(pl.Utf8).to_list()
        return IntensityProfile(
            df=profile.df.filter(pl.col("match_id").is_in(ids)), n_buckets=profile.n_buckets
        )
    return profile


@fragment_if_available
def render_intensity_heatmap(
    dff: pl.DataFrame,
    *,
    db_path: str,
    xuid: str,
    lang: str = "fr",
) -> None:
    """Heatmap d'intensité : profil de kills normalisé par phase pour chaque match."""
    match_ids = dff["match_id"].cast(pl.Utf8).to_list() if "match_id" in dff.columns else []
    if len(match_ids) < 3:
        return

    logger.debug("intensity_heatmap: %d matchs à analyser", len(match_ids))

    from src.analysis.match_intensity import compute_match_intensity_profiles
    from src.ui._cache_queries import cached_load_kill_timing_for_matches

    raw = cached_load_kill_timing_for_matches(db_path, xuid, tuple(match_ids), xuids=(xuid,))
    if not raw:
        return

    events_df = pl.DataFrame(
        raw,
        schema={"match_id": pl.Utf8, "time_ms": pl.Int64, "xuid": pl.Utf8},
    )
    profile = compute_match_intensity_profiles(events_df, n_buckets=10)
    if profile.df.is_empty() or len(profile.df) < 3:
        return

    profile = _reorder_profile_by_date(profile, dff)

    st.subheader(t("ts_match_intensity"))
    if hints_visible():
        st.caption(t("ts_match_intensity_caption"))

    filter_opts = [t("ts_intensity_all"), t("ts_intensity_wins"), t("ts_intensity_losses")]
    outcome_filter = st.segmented_control(
        t("ts_intensity_filter"),
        options=filter_opts,
        default=filter_opts[0],
        key="intensity_heatmap_filter",
    )
    filtered_profile = _apply_outcome_filter(profile, dff, outcome_filter, filter_opts)

    st.caption(t("ts_intensity_match_count").format(n=len(filtered_profile.df)))

    with safe_chart_render():
        fig = plot_match_intensity_heatmap(filtered_profile, opts=PlotOptions(lang=lang))
        if fig is not None:
            st.plotly_chart(fig, width="stretch", config=PLOTLY_CLEAN_CONFIG)
        else:
            st.info(t("ts_intensity_no_data"))
