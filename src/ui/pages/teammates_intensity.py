"""Heatmap d'intensité interactive par joueur pour la page Teammates.

Affiche la heatmap match × phase (profil de kills normalisé) pour le joueur
sélectionné via un segmented_control. Les données de tous les joueurs sont
chargées en une seule requête ; le toggle est côté Streamlit.

Architecture : couche UI pure — pas de logique métier ici.
  - Analyse  : src/analysis/match_intensity.compute_match_intensity_profiles
  - Viz      : src/visualization/match_intensity_heatmap.plot_match_intensity_heatmap
  - Cache    : src/ui/_cache_queries.cached_load_kill_timing_for_matches
"""

from __future__ import annotations

import logging

import polars as pl
import streamlit as st

from src.ui.chart_utils import safe_chart_render
from src.ui.components.browser_storage import hints_visible
from src.ui.i18n import t
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, fragment_if_available
from src.visualization._plot_options import PlotOptions

logger = logging.getLogger(__name__)


def _sort_profile_by_match_order(
    profile: object,
    match_ids_ordered: list[str],
) -> object:
    """Réordonne le profil selon l'ordre chronologique fourni par le caller."""
    from src.analysis.match_intensity import IntensityProfile

    order_map = {mid: i for i, mid in enumerate(match_ids_ordered)}
    sorted_df = (
        profile.df.with_columns(
            pl.col("match_id")
            .replace_strict(order_map, default=9999, return_dtype=pl.Int32)
            .alias("_order")
        )
        .sort("_order")
        .drop("_order")
    )
    return IntensityProfile(df=sorted_df, n_buckets=profile.n_buckets)


def _build_y_labels_from_me_df(
    profile_match_ids: list[str],
    me_df: pl.DataFrame | None,
) -> list[str]:
    """Étiquettes axe Y via prepare_time_axis (carte si dispo, sinon date).

    Filtre me_df aux seuls matchs présents dans profile_match_ids et les
    ordonne selon cet ordre avant d'appeler prepare_time_axis.
    """
    from src.visualization._timeseries_helpers import prepare_time_axis

    if me_df is None or me_df.is_empty() or not profile_match_ids:
        return [f"#{i + 1}" for i in range(len(profile_match_ids))]

    order_map = {mid: i for i, mid in enumerate(profile_match_ids)}
    filtered = (
        me_df.filter(pl.col("match_id").cast(pl.Utf8).is_in(profile_match_ids))
        .with_columns(
            pl.col("match_id")
            .cast(pl.Utf8)
            .replace_strict(order_map, default=9999, return_dtype=pl.Int32)
            .alias("_prof_order")
        )
        .sort("_prof_order")
        .drop("_prof_order")
    )
    if filtered.is_empty():
        return [f"#{i + 1}" for i in range(len(profile_match_ids))]

    _, labels, _ = prepare_time_axis(filtered)
    return labels


def _compute_player_profile(
    events_df: pl.DataFrame,
    xuid: str | None,
    match_ids_ordered: list[str],
) -> object | None:
    """Calcule et ordonne le profil d'intensité pour un joueur (ou toute l'escouade si xuid=None)."""
    from src.analysis.match_intensity import compute_match_intensity_profiles

    player_events = events_df if xuid is None else events_df.filter(pl.col("xuid") == xuid)
    if player_events.is_empty():
        return None

    profile = compute_match_intensity_profiles(player_events, n_buckets=10)
    if profile.df.is_empty() or len(profile.df) < 2:
        return None

    return _sort_profile_by_match_order(profile, match_ids_ordered)


@fragment_if_available
def render_squad_intensity_heatmap(
    db_path: str,
    xuid_name_map: dict[str, str],
    match_ids_ordered: list[str],
    lang: str = "fr",
    me_df: pl.DataFrame | None = None,
) -> None:
    """Heatmap d'intensité de kills par joueur — toggle via segmented_control.

    Args:
        db_path: Chemin vers la DB du joueur principal (pour cached_load).
        xuid_name_map: {xuid: display_name} des joueurs de l'escouade.
        match_ids_ordered: match_ids dans l'ordre chronologique.
        lang: Langue d'affichage.
        me_df: DataFrame du joueur principal avec ``match_id``, ``start_time``
            et optionnellement ``map_ui``/``map_name`` — utilisé pour les
            étiquettes de l'axe Y (carte si disponible, sinon date).
    """
    if len(match_ids_ordered) < 3 or not xuid_name_map:
        return

    from src.ui._cache_queries import cached_load_kill_timing_for_matches

    all_xuids = list(xuid_name_map.keys())
    all_names = list(xuid_name_map.values())

    raw = cached_load_kill_timing_for_matches(
        db_path,
        all_xuids[0],
        tuple(match_ids_ordered),
        xuids=tuple(all_xuids),
    )
    if not raw:
        return

    events_df = pl.DataFrame(
        raw,
        schema={"match_id": pl.Utf8, "time_ms": pl.Int64, "xuid": pl.Utf8},
    )

    st.subheader(t("tm_intensity_title"))
    if hints_visible():
        st.caption(t("tm_intensity_caption"))

    _all_label = t("tm_intensity_all")
    options = [_all_label] + all_names
    selected_name = st.segmented_control(
        t("tm_intensity_player_select"),
        options=options,
        default=_all_label,
        key="squad_intensity_player_select",
    )
    if selected_name is None:
        selected_name = _all_label

    if selected_name == _all_label:
        profile = _compute_player_profile(events_df, xuid=None, match_ids_ordered=match_ids_ordered)
    else:
        selected_xuid = all_xuids[all_names.index(selected_name)]
        profile = _compute_player_profile(events_df, selected_xuid, match_ids_ordered)

    if profile is None:
        st.info(t("tm_intensity_no_data"))
        return

    st.caption(t("tm_intensity_match_count").format(n=len(profile.df)))

    from src.visualization.match_intensity_heatmap import plot_match_intensity_heatmap

    y_labels = _build_y_labels_from_me_df(profile.df["match_id"].to_list(), me_df)

    with safe_chart_render():
        fig = plot_match_intensity_heatmap(
            profile, opts=PlotOptions(lang=lang), match_labels=y_labels
        )
        if fig is not None:
            st.plotly_chart(fig, width="stretch", config=PLOTLY_CLEAN_CONFIG)
        else:
            st.info(t("tm_intensity_no_data"))
