"""Graphiques pour la page Coéquipiers.

Fonctions de visualisation pour comparer les performances avec les coéquipiers.
"""

from __future__ import annotations

import logging

import plotly.graph_objects as go
import polars as pl
import streamlit as st

logger = logging.getLogger(__name__)

from src.config import OKABE_ITO_PALETTE
from src.ui.chart_utils import safe_chart_render
from src.ui.i18n import get_lang, t
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, PLOTLY_STATIC_CONFIG, fragment_if_available
from src.visualization import plot_trio_metric
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization.teammates_hs_pk import plot_hs_pk_stacked
from src.visualization.trio import plot_trio_kills_deaths


@fragment_if_available
def render_metric_bar_charts(
    series: list[tuple[str, DataFrameLike]],
    colors_by_name: dict[str, str],
    show_smooth: bool,
    key_suffix: str,
    plot_fn,
) -> None:
    """Affiche les graphes de barres pour les métriques.

    Args:
        series: Liste de tuples (nom, DataFrame).
        colors_by_name: Mapping nom → couleur.
        show_smooth: Afficher les courbes lissées.
        key_suffix: Suffixe pour les clés Streamlit.
        plot_fn: Fonction de tracé.
    """
    _lang = get_lang()
    for metric_col, label, key_prefix in [
        ("max_killing_spree", t("tm_killing_spree"), "friend_spree_multi"),
    ]:
        fig = plot_fn(
            series,
            metric_col=metric_col,
            title=label,
            y_axis_title=label,
            hover_label=label,
            colors=colors_by_name,
            smooth_window=10,
            show_smooth_lines=show_smooth,
            lang=_lang,
        )
        if fig is None:
            st.info(t("insufficient_data_chart"))
        else:
            st.plotly_chart(
                fig,
                width="stretch",
                key=f"{key_prefix}_{key_suffix}",
                config=PLOTLY_CLEAN_CONFIG,
            )

    # ── Graphe combiné HS + PK ──────────────────────────────────────────────
    fig_combined = plot_hs_pk_stacked(
        series,
        colors=colors_by_name,
        lang=_lang,
    )
    if fig_combined is None:
        st.info(t("insufficient_data_chart"))
    else:
        st.plotly_chart(
            fig_combined,
            width="stretch",
            key=f"friend_hs_pk_stacked_{key_suffix}",
            config=PLOTLY_CLEAN_CONFIG,
        )


def render_outcome_bar_chart(dfr: DataFrameLike) -> None:
    """Affiche le graphe de distribution des résultats.

    Args:
        dfr: DataFrame avec colonne 'my_outcome'.
    """
    dfr = ensure_polars(dfr)
    outcome_map = {
        2: t("outcome_win"),
        3: t("outcome_loss"),
        1: t("outcome_draw"),
        4: t("outcome_dnf"),
    }
    dfr = dfr.with_columns(
        pl.col("my_outcome")
        .replace_strict(outcome_map, default=t("outcome_unknown"), return_dtype=pl.Utf8)
        .alias("my_outcome_label")
    )
    ordered_labels = [
        t("outcome_win"),
        t("outcome_loss"),
        t("outcome_draw"),
        t("outcome_dnf"),
        t("outcome_unknown"),
    ]
    counts_df = dfr.group_by("my_outcome_label").len().rename({"len": "count"})
    all_labels = pl.DataFrame({"my_outcome_label": ordered_labels})
    counts_df = all_labels.join(counts_df, on="my_outcome_label", how="left").fill_null(0)
    colors = OKABE_ITO_PALETTE
    fig = go.Figure(
        data=[
            go.Bar(
                x=counts_df["my_outcome_label"].to_list(),
                y=counts_df["count"].to_list(),
                marker_color=colors[0],
            )
        ]
    )
    fig.update_layout(height=300, margin={"l": 40, "r": 20, "t": 30, "b": 40})
    st.plotly_chart(fig, width="stretch", config=PLOTLY_STATIC_CONFIG)


def _plot_trio_metric_chart(  # noqa: PLR0913
    *,
    d_self: DataFrameLike,
    d_f1: DataFrameLike,
    d_f2: DataFrameLike | None,
    d_f3: DataFrameLike | None,
    names: tuple[str, ...],
    lang: str,
    colors_by_name: dict[str, str] | None,
    metric: str,
    title: str,
    y_title: str,
    key: str,
    y_suffix: str = "",
    y_format: str = "",
    is_inverse: bool = False,
) -> None:
    """Rend un seul graphique trio_metric via st.plotly_chart."""
    st.plotly_chart(
        plot_trio_metric(
            d_self,
            d_f1,
            d_f2,
            metric=metric,
            names=names,
            title=title,
            y_title=y_title,
            y_suffix=y_suffix,
            y_format=y_format,
            lang=lang,
            d_f3=d_f3,
            colors_by_name=colors_by_name,
            is_inverse=is_inverse,
        ),
        width="stretch",
        key=key,
        config=PLOTLY_CLEAN_CONFIG,
    )


# (metric, title_key, ytitle_key, key_prefix, extra_kwargs)
_TRIO_METRIC_SPECS: list[tuple[str, str, str, str, dict]] = [
    ("assists", "tm_assists", "tm_assists", "trio_assists", {}),
    ("ratio", "tm_kda", "tm_kda", "trio_ratio", {"y_format": ".3f"}),
    ("accuracy", "tm_accuracy", None, "trio_accuracy", {"y_suffix": "%", "y_format": ".2f"}),
    ("average_life_seconds", "tm_avg_life", "tm_seconds", "trio_life", {"y_format": ".1f"}),
    ("performance", "tm_performance", "tm_score", "trio_performance", {"y_format": ".1f"}),
]


@fragment_if_available
def render_trio_charts(  # noqa: PLR0913
    d_self: DataFrameLike,
    d_f1: DataFrameLike,
    d_f2: DataFrameLike | None,
    me_name: str,
    f1_name: str,
    f2_name: str | None,
    f1_xuid: str,
    f2_xuid: str | None,
    *,
    d_f3: DataFrameLike | None = None,
    f3_name: str | None = None,
    f3_xuid: str | None = None,
    colors_by_name: dict[str, str] | None = None,
) -> None:
    """Affiche les graphes de métriques pour une escouade de 2, 3 ou 4 joueurs."""
    names: tuple[str, ...] = (me_name, f1_name)
    if f2_name:
        names += (f2_name,)
    if f3_name:
        names += (f3_name,)
    key_suffix = f1_xuid + (f"_{f2_xuid}" if f2_xuid else "") + (f"_{f3_xuid}" if f3_xuid else "")
    _lang = get_lang()

    # Graphe combiné kills↑/morts↓ (remplace les deux graphes séparés)
    with safe_chart_render():
        st.plotly_chart(
            plot_trio_kills_deaths(
                d_self,
                d_f1,
                d_f2,
                names=names,
                title=t("tm_kills_deaths"),
                lang=_lang,
                d_f3=d_f3,
                colors_by_name=colors_by_name,
            ),
            width="stretch",
            key=f"trio_kd_{key_suffix}",
            config=PLOTLY_CLEAN_CONFIG,
        )

    _shared: dict = {
        "d_self": d_self,
        "d_f1": d_f1,
        "d_f2": d_f2,
        "d_f3": d_f3,
        "names": names,
        "lang": _lang,
        "colors_by_name": colors_by_name,
    }
    for metric, title_key, ytitle_key, key_prefix, extra in _TRIO_METRIC_SPECS:
        _plot_trio_metric_chart(
            **_shared,
            metric=metric,
            title=t(title_key),
            y_title=t(ytitle_key) if ytitle_key else "%",
            key=f"{key_prefix}_{key_suffix}",
            **extra,
        )
