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
from src.ui.i18n import get_lang, t
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, PLOTLY_STATIC_CONFIG, fragment_if_available
from src.visualization import (
    plot_average_life,
    plot_per_minute_timeseries,
    plot_performance_timeseries,
    plot_timeseries,
    plot_trio_metric,
)
from src.visualization._compat import DataFrameLike, ensure_polars


@fragment_if_available
def render_comparison_charts(  # noqa: PLR0913
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
            plot_timeseries(sub, title=f"{me_name} — matchs avec {friend_name}", lang=get_lang()),
            width="stretch",
            key=f"friend_ts_me_{friend_xuid}",
            config=PLOTLY_CLEAN_CONFIG,
        )
    with c2:
        if friend_sub.is_empty():
            st.warning(t("error_chart", error="charger les stats du coéquipier"))
        else:
            st.plotly_chart(
                plot_timeseries(
                    friend_sub, title=f"{friend_name} — matchs avec {me_name}", lang=get_lang()
                ),
                width="stretch",
                key=f"friend_ts_fr_{friend_xuid}",
                config=PLOTLY_CLEAN_CONFIG,
            )

    c3, c4 = st.columns(2)
    with c3:
        st.plotly_chart(
            plot_per_minute_timeseries(
                sub, title=f"{me_name} — stats/min (avec {friend_name})", lang=get_lang()
            ),
            width="stretch",
            key=f"friend_pm_me_{friend_xuid}",
            config=PLOTLY_CLEAN_CONFIG,
        )
    with c4:
        if not friend_sub.is_empty():
            st.plotly_chart(
                plot_per_minute_timeseries(
                    friend_sub,
                    title=f"{friend_name} — stats/min (avec {me_name})",
                    lang=get_lang(),
                ),
                width="stretch",
                key=f"friend_pm_fr_{friend_xuid}",
                config=PLOTLY_CLEAN_CONFIG,
            )

    c5, c6 = st.columns(2)
    with c5:
        if not sub.drop_nulls(subset=["average_life_seconds"]).is_empty():
            st.plotly_chart(
                plot_average_life(
                    sub,
                    title=t("tm_lifespan_with", player=me_name, partner=friend_name),
                    lang=get_lang(),
                ),
                width="stretch",
                key=f"friend_life_me_{friend_xuid}",
                config=PLOTLY_CLEAN_CONFIG,
            )
    with c6:
        if (
            not friend_sub.is_empty()
            and not friend_sub.drop_nulls(subset=["average_life_seconds"]).is_empty()
        ):
            st.plotly_chart(
                plot_average_life(
                    friend_sub,
                    title=t("tm_lifespan_with", player=friend_name, partner=me_name),
                    lang=get_lang(),
                ),
                width="stretch",
                key=f"friend_life_fr_{friend_xuid}",
                config=PLOTLY_CLEAN_CONFIG,
            )

    # Graphes de performance
    c7, c8 = st.columns(2)
    with c7:
        st.plotly_chart(
            plot_performance_timeseries(
                sub,
                title=f"{me_name} — Performance (avec {friend_name})",
                show_smooth=show_smooth,
                lang=get_lang(),
            ),
            width="stretch",
            key=f"friend_perf_me_{friend_xuid}",
            config=PLOTLY_CLEAN_CONFIG,
        )
    with c8:
        if not friend_sub.is_empty():
            st.plotly_chart(
                plot_performance_timeseries(
                    friend_sub,
                    title=f"{friend_name} — Performance (avec {me_name})",
                    show_smooth=show_smooth,
                    lang=get_lang(),
                ),
                width="stretch",
                key=f"friend_perf_fr_{friend_xuid}",
                config=PLOTLY_CLEAN_CONFIG,
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
    _lang = get_lang()
    fig_spree = plot_fn(
        series,
        metric_col="max_killing_spree",
        title=t("tm_killing_spree"),
        y_axis_title=t("tm_killing_spree"),
        hover_label=t("tm_killing_spree"),
        colors=colors_by_name,
        smooth_window=10,
        show_smooth_lines=show_smooth,
        lang=_lang,
    )
    if fig_spree is None:
        st.info(t("insufficient_data_chart"))
    else:
        st.plotly_chart(
            fig_spree,
            width="stretch",
            key=f"friend_spree_multi_{key_suffix}",
            config=PLOTLY_CLEAN_CONFIG,
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
        lang=_lang,
    )
    if fig_hs is None:
        st.info(t("insufficient_data_chart"))
    else:
        st.plotly_chart(
            fig_hs,
            width="stretch",
            key=f"friend_hs_multi_{key_suffix}",
            config=PLOTLY_CLEAN_CONFIG,
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
        lang=_lang,
    )
    if fig_pk is None:
        st.info(t("insufficient_data_chart"))
    else:
        st.plotly_chart(
            fig_pk,
            width="stretch",
            key=f"friend_pk_multi_{key_suffix}",
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


@fragment_if_available
def render_trio_charts(  # noqa: PLR0913
    d_self: DataFrameLike,
    d_f1: DataFrameLike,
    d_f2: DataFrameLike,
    me_name: str,
    f1_name: str,
    f2_name: str,
    f1_xuid: str,
    f2_xuid: str,
    *,
    d_f3: DataFrameLike | None = None,
    f3_name: str | None = None,
    f3_xuid: str | None = None,
    colors_by_name: dict[str, str] | None = None,
) -> None:
    """Affiche les graphes d'escouade (3 ou 4 joueurs).

    Args:
        d_self: DataFrame du joueur principal.
        d_f1: DataFrame du premier coéquipier.
        d_f2: DataFrame du deuxième coéquipier.
        me_name: Nom du joueur principal.
        f1_name: Nom du premier coéquipier.
        f2_name: Nom du deuxième coéquipier.
        f1_xuid: XUID du premier coéquipier.
        f2_xuid: XUID du deuxième coéquipier.
        d_f3: DataFrame optionnel du 3ème coéquipier.
        f3_name: Nom optionnel du 3ème coéquipier.
        f3_xuid: XUID optionnel du 3ème coéquipier.
        colors_by_name: Mapping nom → couleur hex (Okabe-Ito recommandé).
    """
    names: tuple[str, ...] = (me_name, f1_name, f2_name)
    if f3_name:
        names = names + (f3_name,)

    key_suffix = f"{f1_xuid}_{f2_xuid}"
    if f3_xuid:
        key_suffix += f"_{f3_xuid}"

    _lang = get_lang()
    st.plotly_chart(
        plot_trio_metric(
            d_self,
            d_f1,
            d_f2,
            metric="kills",
            names=names,
            title=t("tm_kills"),
            y_title=t("tm_kills"),
            lang=_lang,
            d_f3=d_f3,
            colors_by_name=colors_by_name,
        ),
        width="stretch",
        key=f"trio_kills_{key_suffix}",
        config=PLOTLY_CLEAN_CONFIG,
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
            lang=_lang,
            d_f3=d_f3,
            colors_by_name=colors_by_name,
        ),
        width="stretch",
        key=f"trio_deaths_{key_suffix}",
        config=PLOTLY_CLEAN_CONFIG,
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
            lang=_lang,
            d_f3=d_f3,
            colors_by_name=colors_by_name,
        ),
        width="stretch",
        key=f"trio_assists_{key_suffix}",
        config=PLOTLY_CLEAN_CONFIG,
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
            lang=_lang,
            d_f3=d_f3,
            colors_by_name=colors_by_name,
        ),
        width="stretch",
        key=f"trio_ratio_{key_suffix}",
        config=PLOTLY_CLEAN_CONFIG,
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
            lang=_lang,
            d_f3=d_f3,
            colors_by_name=colors_by_name,
        ),
        width="stretch",
        key=f"trio_accuracy_{key_suffix}",
        config=PLOTLY_CLEAN_CONFIG,
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
            lang=_lang,
            d_f3=d_f3,
            colors_by_name=colors_by_name,
        ),
        width="stretch",
        key=f"trio_life_{key_suffix}",
        config=PLOTLY_CLEAN_CONFIG,
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
            lang=_lang,
            d_f3=d_f3,
            colors_by_name=colors_by_name,
        ),
        width="stretch",
        key=f"trio_performance_{key_suffix}",
        config=PLOTLY_CLEAN_CONFIG,
    )
