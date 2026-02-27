"""Graphiques pour la page Coéquipiers.

Fonctions de visualisation pour comparer les performances avec les coéquipiers.
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl
import streamlit as st

from src.config import HALO_COLORS
from src.ui.i18n import t
from src.ui.streamlit_modern import fragment_if_available
from src.visualization import (
    plot_average_life,
    plot_per_minute_timeseries,
    plot_performance_timeseries,
    plot_timeseries,
    plot_trio_metric,
)
from src.visualization._compat import DataFrameLike, ensure_polars


@fragment_if_available
def render_comparison_charts(
    sub: DataFrameLike,
    friend_sub: DataFrameLike,
    me_name: str,
    friend_name: str,
    friend_xuid: str,
    show_smooth: bool = True,
) -> None:
    """Affiche les graphes de comparaison côte à côte.

    Args:
        sub: DataFrame des matchs du joueur principal.
        friend_sub: DataFrame des matchs du coéquipier.
        me_name: Nom du joueur principal.
        friend_name: Nom du coéquipier.
        friend_xuid: XUID du coéquipier.
        show_smooth: Afficher les courbes lissées.
    """
    sub = ensure_polars(sub)
    friend_sub = ensure_polars(friend_sub)
    c1, c2 = st.columns(2)
    with c1:
        st.plotly_chart(
            plot_timeseries(sub, title=f"{me_name} — matchs avec {friend_name}"),
            width="stretch",
            key=f"friend_ts_me_{friend_xuid}",
            config={"displayModeBar": False},
        )
    with c2:
        if friend_sub.is_empty():
            st.warning(t("error_chart", error="charger les stats du coéquipier"))
        else:
            st.plotly_chart(
                plot_timeseries(friend_sub, title=f"{friend_name} — matchs avec {me_name}"),
                width="stretch",
                key=f"friend_ts_fr_{friend_xuid}",
                config={"displayModeBar": False},
            )

    c3, c4 = st.columns(2)
    with c3:
        st.plotly_chart(
            plot_per_minute_timeseries(sub, title=f"{me_name} — stats/min (avec {friend_name})"),
            width="stretch",
            key=f"friend_pm_me_{friend_xuid}",
            config={"displayModeBar": False},
        )
    with c4:
        if not friend_sub.is_empty():
            st.plotly_chart(
                plot_per_minute_timeseries(
                    friend_sub,
                    title=f"{friend_name} — stats/min (avec {me_name})",
                ),
                width="stretch",
                key=f"friend_pm_fr_{friend_xuid}",
                config={"displayModeBar": False},
            )

    c5, c6 = st.columns(2)
    with c5:
        if not sub.drop_nulls(subset=["average_life_seconds"]).is_empty():
            st.plotly_chart(
                plot_average_life(sub, title=t("tm_lifespan_with", player=me_name, partner=friend_name)),
                width="stretch",
                key=f"friend_life_me_{friend_xuid}",
                config={"displayModeBar": False},
            )
    with c6:
        if (
            not friend_sub.is_empty()
            and not friend_sub.drop_nulls(subset=["average_life_seconds"]).is_empty()
        ):
            st.plotly_chart(
                plot_average_life(
                    friend_sub, title=t("tm_lifespan_with", player=friend_name, partner=me_name)
                ),
                width="stretch",
                key=f"friend_life_fr_{friend_xuid}",
                config={"displayModeBar": False},
            )

    # Graphes de performance
    c7, c8 = st.columns(2)
    with c7:
        st.plotly_chart(
            plot_performance_timeseries(
                sub, title=f"{me_name} — Performance (avec {friend_name})", show_smooth=show_smooth
            ),
            width="stretch",
            key=f"friend_perf_me_{friend_xuid}",
            config={"displayModeBar": False},
        )
    with c8:
        if not friend_sub.is_empty():
            st.plotly_chart(
                plot_performance_timeseries(
                    friend_sub,
                    title=f"{friend_name} — Performance (avec {me_name})",
                    show_smooth=show_smooth,
                ),
                width="stretch",
                key=f"friend_perf_fr_{friend_xuid}",
                config={"displayModeBar": False},
            )


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
    fig_spree = plot_fn(
        series,
        metric_col="max_killing_spree",
        title=t("tm_killing_spree"),
        y_axis_title=t("tm_killing_spree"),
        hover_label=t("tm_killing_spree"),
        colors=colors_by_name,
        smooth_window=10,
        show_smooth_lines=show_smooth,
    )
    if fig_spree is None:
        st.info(t("insufficient_data_chart"))
    else:
        st.plotly_chart(
            fig_spree,
            width="stretch",
            key=f"friend_spree_multi_{key_suffix}",
            config={"displayModeBar": False},
        )

    fig_hs = plot_fn(
        series,
        metric_col="headshot_kills",
        title=t("tm_headshots"),
        y_axis_title=t("tm_headshots"),
        hover_label=t("tm_headshots"),
        colors=colors_by_name,
        smooth_window=10,
        show_smooth_lines=show_smooth,
    )
    if fig_hs is None:
        st.info(t("insufficient_data_chart"))
    else:
        st.plotly_chart(
            fig_hs,
            width="stretch",
            key=f"friend_hs_multi_{key_suffix}",
            config={"displayModeBar": False},
        )

    fig_pk = plot_fn(
        series,
        metric_col="perfect_kills",
        title=t("tm_perfect_kills"),
        y_axis_title=t("tm_perfect_kills"),
        hover_label=t("tm_perfect_kills"),
        colors=colors_by_name,
        smooth_window=10,
        show_smooth_lines=show_smooth,
    )
    if fig_pk is None:
        st.info(t("insufficient_data_chart"))
    else:
        st.plotly_chart(
            fig_pk,
            width="stretch",
            key=f"friend_pk_multi_{key_suffix}",
            config={"displayModeBar": False},
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
    colors = HALO_COLORS.as_dict()
    fig = go.Figure(
        data=[
            go.Bar(
                x=counts_df["my_outcome_label"].to_list(),
                y=counts_df["count"].to_list(),
                marker_color=colors["cyan"],
            )
        ]
    )
    fig.update_layout(height=300, margin={"l": 40, "r": 20, "t": 30, "b": 40})
    st.plotly_chart(fig, width="stretch", config={"staticPlot": True})


@fragment_if_available
def render_trio_charts(
    d_self: DataFrameLike,
    d_f1: DataFrameLike,
    d_f2: DataFrameLike,
    me_name: str,
    f1_name: str,
    f2_name: str,
    f1_xuid: str,
    f2_xuid: str,
) -> None:
    """Affiche les graphes trio (moi + 2 coéquipiers).

    Args:
        d_self: DataFrame du joueur principal.
        d_f1: DataFrame du premier coéquipier.
        d_f2: DataFrame du deuxième coéquipier.
        me_name: Nom du joueur principal.
        f1_name: Nom du premier coéquipier.
        f2_name: Nom du deuxième coéquipier.
        f1_xuid: XUID du premier coéquipier.
        f2_xuid: XUID du deuxième coéquipier.
    """
    names = (me_name, f1_name, f2_name)

    st.plotly_chart(
        plot_trio_metric(
            d_self,
            d_f1,
            d_f2,
            metric="kills",
            names=names,
            title=t("tm_kills"),
            y_title=t("tm_kills"),
        ),
        width="stretch",
        key=f"trio_kills_{f1_xuid}_{f2_xuid}",
        config={"displayModeBar": False},
    )
    st.plotly_chart(
        plot_trio_metric(
            d_self,
            d_f1,
            d_f2,
            metric="deaths",
            names=names,
            title=t("tm_deaths"),
            y_title=t("tm_deaths"),
        ),
        width="stretch",
        key=f"trio_deaths_{f1_xuid}_{f2_xuid}",
        config={"displayModeBar": False},
    )
    st.plotly_chart(
        plot_trio_metric(
            d_self,
            d_f1,
            d_f2,
            metric="assists",
            names=names,
            title=t("tm_assists"),
            y_title=t("tm_assists"),
        ),
        width="stretch",
        key=f"trio_assists_{f1_xuid}_{f2_xuid}",
        config={"displayModeBar": False},
    )
    st.plotly_chart(
        plot_trio_metric(
            d_self,
            d_f1,
            d_f2,
            metric="ratio",
            names=names,
            title=t("tm_kda"),
            y_title=t("tm_kda"),
            y_format=".3f",
        ),
        width="stretch",
        key=f"trio_ratio_{f1_xuid}_{f2_xuid}",
        config={"displayModeBar": False},
    )
    st.plotly_chart(
        plot_trio_metric(
            d_self,
            d_f1,
            d_f2,
            metric="accuracy",
            names=names,
            title=t("tm_accuracy"),
            y_title="%",
            y_suffix="%",
            y_format=".2f",
        ),
        width="stretch",
        key=f"trio_accuracy_{f1_xuid}_{f2_xuid}",
        config={"displayModeBar": False},
    )
    st.plotly_chart(
        plot_trio_metric(
            d_self,
            d_f1,
            d_f2,
            metric="average_life_seconds",
            names=names,
            title=t("tm_avg_life"),
            y_title=t("tm_seconds"),
            y_format=".1f",
        ),
        width="stretch",
        key=f"trio_life_{f1_xuid}_{f2_xuid}",
        config={"displayModeBar": False},
    )
    st.plotly_chart(
        plot_trio_metric(
            d_self,
            d_f1,
            d_f2,
            metric="performance",
            names=names,
            title=t("tm_performance"),
            y_title=t("tm_score"),
            y_format=".1f",
        ),
        width="stretch",
        key=f"trio_performance_{f1_xuid}_{f2_xuid}",
        config={"displayModeBar": False},
    )
