"""Page Séries temporelles.

Graphes d'évolution des statistiques dans le temps.

8bis.A6 : Downsampling pour graphiques avec >200 points (gain ~35% rendu).
"""

from __future__ import annotations

import polars as pl
import streamlit as st

from src.config import HALO_COLORS
from src.data.services.timeseries_service import TimeseriesService
from src.ui.i18n import t
from src.ui.streamlit_modern import fragment_if_available
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization.distributions import (
    plot_correlation_scatter,
    plot_first_event_distribution,
    plot_histogram,
    plot_kda_distribution,
)
from src.visualization.performance import (
    plot_cumulative_kd,
    plot_cumulative_net_score,
    plot_rolling_kd,
    plot_session_trend,
)
from src.visualization.timeseries import (
    plot_assists_timeseries,
    plot_average_life,
    plot_damage_dealt_taken,
    plot_per_minute_timeseries,
    plot_performance_timeseries,
    plot_rank_score,
    plot_shots_accuracy,
    plot_spree_headshots_accuracy,
    plot_timeseries,
)

# =============================================================================
# Downsampling pour performance (8bis.A6)
# =============================================================================

MAX_PLOT_POINTS = 200


def _downsample_for_plot(df: pl.DataFrame, max_points: int = MAX_PLOT_POINTS) -> pl.DataFrame:
    """Réduit le DataFrame pour le rendu graphique (conserve tendance).

    Garde le premier, le dernier, et un échantillonnage régulier entre les deux.
    Idéal pour les timeseries où on veut voir la tendance générale.

    Args:
        df: DataFrame d'entrée trié par start_time.
        max_points: Nombre maximum de points à conserver.

    Returns:
        DataFrame réduit si nécessaire.
    """
    if len(df) <= max_points:
        return df

    # Calculer le pas d'échantillonnage
    step = len(df) // max_points

    # Indices à conserver : 0, step, 2*step, ..., dernier
    indices = list(range(0, len(df), step))
    if indices[-1] != len(df) - 1:
        indices.append(len(df) - 1)

    return df[indices]


# =============================================================================
# Sous-fonctions de rendu extraites du monolithe (Sprint 16)
# =============================================================================


@fragment_if_available
def _render_kda_section(dff: pl.DataFrame) -> None:
    """Affiche le graphe KDA et sa distribution."""
    try:
        # 8bis.A6 : Downsampling pour performance
        df_plot = _downsample_for_plot(dff)
        fig = plot_timeseries(df_plot)
        if fig is not None:
            st.plotly_chart(fig, width="stretch", config={"displayModeBar": False})
        else:
            st.info(t("insufficient_data_chart"))
    except Exception as e:
        st.warning(t("error_chart", error=e))

    st.subheader(t("ts_fda"))
    valid = dff.drop_nulls(subset=["kda"]) if "kda" in dff.columns else pl.DataFrame()
    if valid.is_empty():
        st.info(t("ts_fda_unavailable"))
    else:
        m = st.columns(1)
        m[0].metric(t("ts_kda_mean_label"), f"{valid['kda'].mean():.2f}", label_visibility="collapsed")
        try:
            fig_dist = plot_kda_distribution(dff)
            if fig_dist is not None:
                st.plotly_chart(fig_dist, width="stretch", config={"staticPlot": True})
            else:
                st.info(t("insufficient_data_chart"))
        except Exception as e:
            st.warning(t("error_chart", error=e))


@fragment_if_available
def _render_cumulative_performance(dff: pl.DataFrame) -> None:
    """Affiche les graphes de performance cumulée et tendance (Sprint 6)."""
    st.divider()
    st.subheader(t("ts_cumulative"))
    st.caption(
        t("ts_cumulative_caption")
    )
    cumul = TimeseriesService.compute_cumulative_metrics(dff)
    if cumul is not None:
        try:
            st.plotly_chart(
                plot_cumulative_net_score(
                    cumul.cumul_net, time_played_seconds=cumul.time_played_seconds
                ),
                width="stretch",
                config={"displayModeBar": False},
            )
            st.plotly_chart(
                plot_cumulative_kd(cumul.cumul_kd, time_played_seconds=cumul.time_played_seconds),
                width="stretch",
                config={"displayModeBar": False},
            )
            st.plotly_chart(
                plot_rolling_kd(cumul.rolling_kd, window_size=5),
                width="stretch",
                config={"displayModeBar": False},
            )
            if cumul.has_enough_for_trend:
                st.plotly_chart(
                    plot_session_trend(cumul.pl_df),
                    width="stretch",
                    config={"staticPlot": True},
                )
            else:
                st.info(t("ts_trend_min_matches"))
        except Exception as e:
            st.warning(t("error_chart", error=e))
    else:
        st.info(t("ts_col_missing_cumul"))


@fragment_if_available
def _render_distributions(dff: pl.DataFrame) -> None:
    """Affiche les distributions statistiques (Sprint 5.4.3 + Sprint 6)."""
    st.divider()
    st.subheader(t("ts_distributions"))
    st.caption(t("ts_distributions_caption"))

    colors = HALO_COLORS.as_dict()
    _render_distribution_row1(dff, colors)
    _render_distribution_row2(dff, colors)
    _render_distribution_row3(dff, colors)


def _render_distribution_row1(dff: pl.DataFrame, colors: dict) -> None:
    """Ligne 1 : précision + kills."""
    col1, col2 = st.columns(2)
    with col1:
        _render_single_histogram(
            dff,
            "accuracy",
            t("ts_dist_accuracy_title"),
            t("ts_accuracy_label"),
            colors["cyan"],
        )
    with col2:
        _render_single_histogram(
            dff,
            "kills",
            t("ts_dist_kills_title"),
            "Frags",
            colors["green"],
        )


def _render_distribution_row2(dff: pl.DataFrame, colors: dict) -> None:
    """Ligne 2 : durée de vie + score de performance."""
    col3, col4 = st.columns(2)
    with col3:
        life_col = (
            "avg_life_seconds" if "avg_life_seconds" in dff.columns else "average_life_seconds"
        )
        _render_single_histogram(
            dff,
            life_col,
            t("ts_dist_life_title"),
            t("ts_life_label"),
            colors["amber"],
        )
    with col4:
        _render_single_histogram(
            dff,
            "performance_score",
            t("ts_dist_perf_title"),
            "Score",
            colors["violet"],
        )


def _render_distribution_row3(dff: pl.DataFrame, colors: dict) -> None:
    """Ligne 3 : score/min + win rate glissant (Sprint 6)."""
    col5, col6 = st.columns(2)
    with col5:
        spm_data = TimeseriesService.compute_score_per_minute(dff)
        if spm_data.has_data:
            fig_spm = plot_histogram(
                spm_data.values,
                title=t("ts_dist_score_per_min_title"),
                x_label=t("ts_score_per_min_label"),
                y_label="Matchs",
                show_kde=True,
                color=colors["amber"],
            )
            st.plotly_chart(fig_spm, width="stretch", config={"staticPlot": True})
        elif "personal_score" not in dff.columns or "time_played_seconds" not in dff.columns:
            st.info(t("ts_col_missing_score_per_min"))
        else:
            st.info(t("ts_insufficient_score_per_min"))

    with col6:
        wr_data = TimeseriesService.compute_rolling_win_rate(dff)
        if wr_data.has_data:
            fig_wr = plot_histogram(
                wr_data.values,
                title=t("ts_dist_win_rate_title"),
                x_label=t("ts_win_rate_label"),
                y_label=t("ts_frequency_label"),
                show_kde=True,
                color=colors["green"],
            )
            st.plotly_chart(fig_wr, width="stretch", config={"staticPlot": True})
        elif wr_data.missing_column:
            st.info(t("ts_col_missing_outcome"))
        elif wr_data.not_enough_matches:
            st.info(t("ts_insufficient_win_rate"))
        else:
            st.info(t("ts_insufficient_win_rate_dist"))


def _render_single_histogram(
    dff: pl.DataFrame,
    column: str,
    title: str,
    x_label: str,
    color: str,
    min_data: int = 6,
) -> None:
    """Affiche un histogramme simple pour une colonne donnée."""
    if column not in dff.columns:
        st.info(f"Colonne {column} non disponible dans les données.")
        return
    data = dff[column].drop_nulls()
    if len(data) > min_data - 1:
        fig = plot_histogram(
            data,
            title=title,
            x_label=x_label,
            y_label="Matchs",
            show_kde=True,
            color=color,
        )
        st.plotly_chart(fig, width="stretch", config={"staticPlot": True})
    elif len(data) == 0:
        st.info(t("no_data_filter"))
    else:
        st.info(
            f"Pas assez de données ({len(data)} matchs). "
            f"Il en faut au moins {min_data} pour afficher la distribution."
        )


@fragment_if_available
def _render_correlations(dff: pl.DataFrame) -> None:
    """Affiche les graphes de corrélation (Sprint 5.4.5 + Sprint 6)."""
    st.divider()
    st.subheader(t("ts_correlations"))
    st.caption(t("ts_correlations_caption"))

    _render_correlation_row1(dff)
    _render_correlation_row2(dff)
    _render_mmr_correlation(dff)


def _render_correlation_row1(dff: pl.DataFrame) -> None:
    """Durée de vie vs Frags + Précision vs FDA."""
    col1, col2 = st.columns(2)
    life_col = "avg_life_seconds" if "avg_life_seconds" in dff.columns else "average_life_seconds"
    with col1:
        _render_scatter(
            dff,
            life_col,
            "kills",
            "outcome",
            "Durée de vie vs frags",
            "Durée de vie (s)",
            "Frags",
        )
    with col2:
        _render_scatter(
            dff,
            "accuracy",
            "kda",
            "outcome",
            "Précision vs FDA",
            "Précision (%)",
            "FDA",
        )


def _render_correlation_row2(dff: pl.DataFrame) -> None:
    """Durée de vie vs Morts + Frags vs Morts (Sprint 6)."""
    col3, col4 = st.columns(2)
    life_col = "avg_life_seconds" if "avg_life_seconds" in dff.columns else "average_life_seconds"
    with col3:
        _render_scatter(
            dff,
            life_col,
            "deaths",
            "outcome",
            "Durée de vie vs morts",
            "Durée de vie (s)",
            "Morts",
        )
    with col4:
        _render_scatter(
            dff,
            "kills",
            "deaths",
            "outcome",
            "Frags vs morts",
            "Frags",
            "Morts",
        )


def _render_mmr_correlation(dff: pl.DataFrame) -> None:
    """Team MMR vs Enemy MMR (Sprint 6)."""
    _render_scatter(
        dff,
        "team_mmr",
        "enemy_mmr",
        "outcome",
        "MMR Équipe vs MMR Adversaire",
        "MMR Équipe",
        "MMR Adversaire",
    )


def _render_scatter(
    dff: pl.DataFrame,
    x_col: str,
    y_col: str,
    color_col: str,
    title: str,
    x_label: str,
    y_label: str,
    min_data: int = 6,
) -> None:
    """Affiche un scatter de corrélation avec validation des données."""
    if x_col not in dff.columns or y_col not in dff.columns:
        st.info(t("insufficient_data_chart"))
        return
    valid = dff.drop_nulls(subset=[x_col, y_col])
    if len(valid) <= min_data - 1:
        st.info(
            f"Pas assez de données ({len(valid)} matchs). "
            f"Il en faut au moins {min_data} pour la corrélation."
        )
        return
    try:
        fig = plot_correlation_scatter(
            dff,
            x_col,
            y_col,
            color_col=color_col,
            title=title,
            x_label=x_label,
            y_label=y_label,
            show_trendline=True,
        )
        if fig is not None:
            st.plotly_chart(fig, width="stretch", config={"displayModeBar": False})
        else:
            st.info(t("ts_corr_gen_error", title=title))
    except Exception as e:
        st.warning(t("error_chart", error=e))


@fragment_if_available
def _render_first_event_section(
    dff: pl.DataFrame,
    db_path: str | None,
    xuid: str | None,
) -> None:
    """Affiche la distribution du premier frag / première mort (Sprint 5.4.4)."""
    st.divider()
    st.subheader(t("ts_first_event"))
    st.caption(t("ts_first_event_caption"))

    _match_ids = dff["match_id"].cast(pl.Utf8).to_list() if "match_id" in dff.columns else []
    first_event = TimeseriesService.load_first_event_times(db_path, xuid, _match_ids)

    if first_event.available:
        try:
            fig_events = plot_first_event_distribution(
                first_event.first_kills,
                first_event.first_deaths,
                title=None,
            )
            if fig_events is not None:
                st.plotly_chart(fig_events, width="stretch", config={"staticPlot": True})
            else:
                st.info(t("insufficient_data_chart"))
        except Exception as e:
            st.warning(t("error_chart", error=e))
    else:
        st.info(
            "Données d'événements non disponibles (premier frag / première mort). "
            "L'**Actualiser** récupère déjà ces données pour les **nouveaux** matchs. "
            "Pour les matchs déjà en base sans événements film, active dans **Paramètres** "
            "→ **Options du bouton Actualiser** l'option **Backfill events**, puis **Actualiser**."
        )


@fragment_if_available
def _render_advanced_sections(
    dff: pl.DataFrame,
    df_full: pl.DataFrame | None,
    db_path: str | None,
    xuid: str | None,
) -> None:
    """Affiche Performance, Assists, Stats/min, Average Life, Spree, Sprint 7."""
    history = df_full if df_full is not None else dff
    st.subheader(t("ts_performance"))
    try:
        fig_perf = plot_performance_timeseries(dff, df_history=history)
        if fig_perf is not None:
            st.plotly_chart(fig_perf, width="stretch", config={"displayModeBar": False})
        else:
            st.info(t("insufficient_data_chart"))
    except Exception as e:
        st.warning(t("error_chart", error=e))

    st.subheader(t("ts_assists"))
    try:
        fig_assists = plot_assists_timeseries(dff)
        if fig_assists is not None:
            st.plotly_chart(fig_assists, width="stretch", config={"displayModeBar": False})
        else:
            st.info(t("insufficient_data_chart"))
    except Exception as e:
        st.warning(t("error_chart", error=e))

    st.subheader(t("ts_per_minute"))
    try:
        fig_spm = plot_per_minute_timeseries(dff)
        if fig_spm is not None:
            st.plotly_chart(fig_spm, width="stretch", config={"displayModeBar": False})
        else:
            st.info(t("insufficient_data_chart"))
    except Exception as e:
        st.warning(t("error_chart", error=e))

    st.subheader(t("ts_lifespan"))
    if dff.drop_nulls(subset=["average_life_seconds"]).is_empty():
        st.info(t("ts_lifespan_unavailable"))
    else:
        try:
            fig_life = plot_average_life(dff)
            if fig_life is not None:
                st.plotly_chart(fig_life, width="stretch", config={"displayModeBar": False})
            else:
                st.info(t("insufficient_data_chart"))
        except Exception as e:
            st.warning(t("error_chart", error=e))

    _render_spree_section(dff, db_path, xuid)
    _render_sprint7_sections(dff)


def _render_spree_section(
    dff: pl.DataFrame,
    db_path: str | None,
    xuid: str | None,
) -> None:
    """Affiche la section Folie meurtrière / Tirs à la tête / Frags parfaits."""
    st.subheader(t("ts_spree"))

    _db_path = db_path or st.session_state.get("db_path")
    _xuid = xuid or st.session_state.get("player_xuid") or st.session_state.get("xuid")
    _match_ids = dff["match_id"].cast(pl.Utf8).to_list() if "match_id" in dff.columns else []
    pk_data = TimeseriesService.load_perfect_kills(_db_path, _xuid, _match_ids)

    try:
        fig_spree = plot_spree_headshots_accuracy(dff, perfect_counts=pk_data.counts)
        if fig_spree is not None:
            st.plotly_chart(fig_spree, width="stretch", config={"displayModeBar": False})
        else:
            st.info(t("insufficient_data_chart"))
    except Exception as e:
        st.warning(t("error_chart", error=e))


def _render_sprint7_sections(dff: pl.DataFrame) -> None:
    """Affiche les sections Sprint 7 : tirs, dégâts, rang."""
    # 7.5 — Tirs et précision
    _has_shots = any(c in dff.columns for c in ("shots_fired", "shots_hit"))
    if _has_shots:
        st.divider()
        st.subheader(t("ts_shots"))
        st.caption(t("ts_shots_caption"))
        try:
            fig_shots = plot_shots_accuracy(dff, title=None)
            if fig_shots is not None:
                st.plotly_chart(fig_shots, width="stretch", config={"displayModeBar": False})
            else:
                st.info(t("insufficient_data_chart"))
        except Exception as e:
            st.warning(t("error_chart", error=e))

    # 7.4 — Dégâts infligés vs subis
    _has_damage = any(c in dff.columns for c in ("damage_dealt", "damage_taken"))
    if _has_damage:
        st.divider()
        st.subheader(t("ts_damage"))
        st.caption(t("ts_damage_caption"))
        try:
            fig_damage = plot_damage_dealt_taken(dff, title=None)
            if fig_damage is not None:
                st.plotly_chart(fig_damage, width="stretch", config={"displayModeBar": False})
            else:
                st.info(t("insufficient_data_chart"))
        except Exception as e:
            st.warning(t("error_chart", error=e))

    # 7.3 — Rang et score personnel
    _has_rank_score = "rank" in dff.columns or "personal_score" in dff.columns
    if _has_rank_score:
        st.divider()
        st.subheader(t("ts_rank_score"))
        st.caption(t("ts_rank_score_caption"))
        try:
            fig_rank = plot_rank_score(dff, title="Rang et score personnel")
            if fig_rank is not None:
                st.plotly_chart(fig_rank, width="stretch", config={"displayModeBar": False})
            else:
                st.info(t("insufficient_data_chart"))
        except Exception as e:
            st.warning(t("error_chart", error=e))


# =============================================================================
# Point d'entrée (orchestrateur réduit)
# =============================================================================


@fragment_if_available
def render_timeseries_page(
    dff: DataFrameLike,
    df_full: DataFrameLike | None = None,
    *,
    db_path: str | None = None,
    xuid: str | None = None,
) -> None:
    """Affiche la page Séries temporelles.

    Args:
        dff: DataFrame filtré des matchs.
        df_full: DataFrame complet pour le calcul du score relatif.
        db_path: Chemin vers la DB (optionnel, pour les features DuckDB).
        xuid: XUID du joueur (optionnel, pour les features DuckDB).
    """
    dff = ensure_polars(dff)
    df_full = ensure_polars(df_full) if df_full is not None else None
    if dff.is_empty():
        st.warning(t("no_matches"))
        return

    # Calculer le score de performance via service (Sprint 14)
    dff = TimeseriesService.enrich_performance_score(
        dff,
        df_full,
    )

    with st.spinner(t("ts_computing")):
        _render_kda_section(dff)
        _render_cumulative_performance(dff)
        _render_distributions(dff)
        _render_correlations(dff)
        _render_first_event_section(dff, db_path, xuid)
        _render_advanced_sections(dff, df_full, db_path, xuid)
