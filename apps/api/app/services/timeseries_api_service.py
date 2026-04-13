"""Service Séries temporelles — Slice 3B.

Point d'entrée : ``get_timeseries_page``.
Toutes les importations src.* sont lazy pour permettre le mocking en tests.
"""

from __future__ import annotations

import logging

from apps.api.app.deps.players import PlayerContext
from apps.api.app.schemas.career import PlotlyFigurePayload
from apps.api.app.schemas.filters import FilterContextInput
from apps.api.app.schemas.timeseries import (
    TimeseriesCumulTab,
    TimeseriesDistributionsTab,
    TimeseriesFormTab,
    TimeseriesIntensityTab,
    TimeseriesKpiCard,
    TimeseriesPageResponse,
    TimeseriesQueryRequest,
    TimeseriesSummaryTab,
)

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Utilitaire sérialisation Plotly
# ---------------------------------------------------------------------------


def _fig_to_payload(fig: object) -> PlotlyFigurePayload | None:
    """Sérialise un ``go.Figure`` en ``PlotlyFigurePayload``."""
    if fig is None:
        return None
    try:
        j = fig.to_plotly_json()  # type: ignore[attr-defined]
        return PlotlyFigurePayload(data=j.get("data", []), layout=j.get("layout", {}))
    except Exception:
        logger.debug("_fig_to_payload: conversion échouée", exc_info=True)
        return None


# ---------------------------------------------------------------------------
# Chargement / filtrage
# ---------------------------------------------------------------------------


def _load_df_full(player: PlayerContext):  # type: ignore[return]
    """Charge le DataFrame complet des matchs du joueur."""
    try:
        from apps.api.app.services.match_history_service import _load_matches_full

        return _load_matches_full(player)
    except Exception:
        logger.warning("_load_df_full: impossible de charger les matchs", exc_info=True)
        try:
            import polars as pl

            return pl.DataFrame()
        except ImportError:
            return []


def _apply_filters(df_full, player: PlayerContext, filters: FilterContextInput):  # type: ignore[return]
    """Applique les filtres sur le DataFrame complet."""
    try:
        from apps.api.app.services.match_history_service import _apply_filter

        return _apply_filter(df_full, player, filters)
    except Exception:
        logger.debug("_apply_filters: erreur filtrage", exc_info=True)
        return df_full


# ---------------------------------------------------------------------------
# Construction des KPI cards
# ---------------------------------------------------------------------------


def _build_kpi_cards(dff, df_full) -> list[TimeseriesKpiCard]:
    """Construit les KPI cards depuis le DataFrame filtré."""
    try:
        import polars  # noqa: F401 — vérification disponibilité
    except ImportError:
        return []

    if dff is None or (hasattr(dff, "is_empty") and dff.is_empty()):
        return []

    cards: list[TimeseriesKpiCard] = []

    total = len(dff)
    cards.append(TimeseriesKpiCard(key="total_matches", label="Matchs", value=str(total)))

    try:
        wins = int((dff["outcome"] == 2).sum()) if "outcome" in dff.columns else 0
        wr = round(wins / total * 100, 1) if total > 0 else 0.0
        cards.append(
            TimeseriesKpiCard(key="win_rate", label="Victoires", value=f"{wr} %", color="green")
        )
    except Exception:
        pass

    try:
        kills = float(dff["kills"].sum()) if "kills" in dff.columns else 0.0
        deaths = float(dff["deaths"].sum()) if "deaths" in dff.columns else 0.0
        assists = float(dff["assists"].sum()) if "assists" in dff.columns else 0.0

        kda = (kills + assists) / deaths if deaths > 0 else kills + assists
        kda_color = "green" if kda >= 1.0 else "red"
        cards.append(TimeseriesKpiCard(key="kda", label="KDA", value=f"{kda:.2f}", color=kda_color))

        kills_pm = kills / total if total > 0 else 0.0
        cards.append(
            TimeseriesKpiCard(key="kills_per_match", label="Kills/match", value=f"{kills_pm:.1f}")
        )
    except Exception:
        pass

    try:
        if "performance_score" in dff.columns:
            ps_mean = float(dff["performance_score"].mean() or 0.0)
            cards.append(
                TimeseriesKpiCard(
                    key="perf_score",
                    label="Score perf. moyen",
                    value=f"{ps_mean:.1f}",
                )
            )
    except Exception:
        pass

    return cards


# ---------------------------------------------------------------------------
# Onglet Résumé
# ---------------------------------------------------------------------------


def _build_summary_tab(dff, df_full) -> TimeseriesSummaryTab:
    """Construit l'onglet KPIs / Résumé."""
    kpi_cards = _build_kpi_cards(dff, df_full)
    kda_dist_chart: PlotlyFigurePayload | None = None

    try:
        from src.visualization.distributions import plot_kda_distribution

        fig = plot_kda_distribution(dff, lang="fr")
        kda_dist_chart = _fig_to_payload(fig)
    except Exception:
        logger.debug("_build_summary_tab: kda_distribution échoué", exc_info=True)

    return TimeseriesSummaryTab(
        kpi_cards=kpi_cards,
        kda_dist_chart=kda_dist_chart,
    )


# ---------------------------------------------------------------------------
# Onglet Cumul
# ---------------------------------------------------------------------------


def _build_cumul_tab(dff) -> TimeseriesCumulTab:
    """Construit l'onglet cumulatif (net score, K/D, rolling K/D)."""
    cumul_net_chart: PlotlyFigurePayload | None = None
    cumul_kd_chart: PlotlyFigurePayload | None = None
    rolling_kd_chart: PlotlyFigurePayload | None = None

    try:
        from src.data.services.timeseries_service import TimeseriesService
        from src.visualization.performance import (
            plot_cumulative_kd,
            plot_cumulative_net_score,
            plot_rolling_kd,
        )

        metrics = TimeseriesService.compute_cumulative_metrics(dff)
        if metrics:
            fig_net = plot_cumulative_net_score(
                metrics.cumul_net,
                time_played_seconds=metrics.time_played_seconds,
            )
            cumul_net_chart = _fig_to_payload(fig_net)

            fig_kd = plot_cumulative_kd(
                metrics.cumul_kd,
                time_played_seconds=metrics.time_played_seconds,
            )
            cumul_kd_chart = _fig_to_payload(fig_kd)

            fig_rolling = plot_rolling_kd(metrics.rolling_kd)
            rolling_kd_chart = _fig_to_payload(fig_rolling)
    except Exception:
        logger.debug("_build_cumul_tab: erreur calcul cumulatif", exc_info=True)

    return TimeseriesCumulTab(
        cumul_net_chart=cumul_net_chart,
        cumul_kd_chart=cumul_kd_chart,
        rolling_kd_chart=rolling_kd_chart,
    )


# ---------------------------------------------------------------------------
# Onglet Forme
# ---------------------------------------------------------------------------


def _build_form_tab(dff) -> TimeseriesFormTab:
    """Construit l'onglet forme récente (EWMA, régression, net score/h)."""
    ewma_kd_chart: PlotlyFigurePayload | None = None
    regression_chart: PlotlyFigurePayload | None = None
    net_score_per_hour_chart: PlotlyFigurePayload | None = None

    try:
        from src.data.services.timeseries_service import TimeseriesService
        from src.visualization.performance import (
            plot_ewma_kd,
            plot_net_score_per_hour,
            plot_regression_trend,
        )

        ewma_df = TimeseriesService.compute_ewma_kd(dff)
        if not ewma_df.is_empty():
            outcome_vals = dff["outcome"].to_list() if "outcome" in dff.columns else None
            reg_data = TimeseriesService.compute_linear_regression_kd(ewma_df)

            fig_ewma = plot_ewma_kd(
                ewma_df,
                regression_data=reg_data,
                outcome_values=outcome_vals,
            )
            ewma_kd_chart = _fig_to_payload(fig_ewma)

            if reg_data:
                fig_reg = plot_regression_trend(reg_data)
                regression_chart = _fig_to_payload(fig_reg)

        nph_df = TimeseriesService.compute_rolling_net_score_per_hour(dff)
        if nph_df is not None and not nph_df.is_empty():
            outcome_vals = dff["outcome"].to_list() if "outcome" in dff.columns else None
            fig_nph = plot_net_score_per_hour(nph_df, outcome_values=outcome_vals)
            net_score_per_hour_chart = _fig_to_payload(fig_nph)
    except Exception:
        logger.debug("_build_form_tab: erreur calcul forme", exc_info=True)

    return TimeseriesFormTab(
        ewma_kd_chart=ewma_kd_chart,
        regression_chart=regression_chart,
        net_score_per_hour_chart=net_score_per_hour_chart,
    )


# ---------------------------------------------------------------------------
# Onglet Intensité
# ---------------------------------------------------------------------------


def _build_intensity_tab(dff, player: PlayerContext) -> TimeseriesIntensityTab:
    """Construit l'onglet intensité (heatmap kill profil + score/min)."""
    intensity_heatmap: PlotlyFigurePayload | None = None
    score_per_minute_chart: PlotlyFigurePayload | None = None

    match_ids: list[str] = []
    try:
        import polars as pl

        if "match_id" in dff.columns:
            match_ids = dff["match_id"].cast(pl.Utf8).to_list()
    except Exception:
        pass

    # Heatmap d'intensité
    if len(match_ids) >= 3:
        try:
            import polars as pl

            from src.analysis.match_intensity import compute_match_intensity_profiles
            from src.data.repositories.duckdb_repo import DuckDBRepository
            from src.visualization.match_intensity_heatmap import plot_match_intensity_heatmap

            with DuckDBRepository(
                player.db_path,
                xuid=player.xuid,
                shared_db_path=player.shared_db_path,
                metadata_db_path=player.metadata_db_path,
                read_only=True,
            ) as repo:
                events = repo.load_kill_timing_for_matches(
                    match_ids=match_ids,
                    xuids=[player.xuid] if player.xuid else None,
                )

            if events:
                events_df = pl.DataFrame(events)
                if (
                    not events_df.is_empty()
                    and "match_id" in events_df.columns
                    and "time_ms" in events_df.columns
                ):
                    profile = compute_match_intensity_profiles(
                        events_df.select(["match_id", "time_ms"])
                    )
                    fig = plot_match_intensity_heatmap(profile)
                    intensity_heatmap = _fig_to_payload(fig)
        except Exception:
            logger.debug("_build_intensity_tab: heatmap échoué", exc_info=True)

    # Score par minute
    try:
        from src.data.services.timeseries_service import TimeseriesService
        from src.visualization.distributions import plot_histogram

        spm = TimeseriesService.compute_score_per_minute(dff)
        if spm.has_data:
            fig_spm = plot_histogram(
                spm.values,
                title="Score / Minute",
                x_label="Score/min",
                y_label="Matchs",
            )
            score_per_minute_chart = _fig_to_payload(fig_spm)
    except Exception:
        logger.debug("_build_intensity_tab: score/min échoué", exc_info=True)

    return TimeseriesIntensityTab(
        intensity_heatmap=intensity_heatmap,
        score_per_minute_chart=score_per_minute_chart,
    )


# ---------------------------------------------------------------------------
# Onglet Distributions
# ---------------------------------------------------------------------------


def _build_distributions_tab(dff, player: PlayerContext) -> TimeseriesDistributionsTab:
    """Construit l'onglet distributions (KDA, first events, corrélations)."""
    kda_distribution: PlotlyFigurePayload | None = None
    first_kill_dist: PlotlyFigurePayload | None = None
    correlations: list[PlotlyFigurePayload] = []

    try:
        from src.visualization.distributions import plot_kda_distribution

        fig = plot_kda_distribution(dff, lang="fr")
        kda_distribution = _fig_to_payload(fig)
    except Exception:
        logger.debug("_build_distributions_tab: kda_dist échoué", exc_info=True)

    # Distribution first kill/death
    match_ids: list[str] = []
    try:
        import polars as pl

        if "match_id" in dff.columns:
            match_ids = dff["match_id"].cast(pl.Utf8).to_list()
    except Exception:
        pass

    if match_ids:
        try:
            from src.data.services.timeseries_service import TimeseriesService
            from src.visualization._distributions_advanced import plot_first_event_distribution

            events_data = TimeseriesService.load_first_event_times(
                player.db_path, player.xuid, match_ids
            )
            if events_data.available:
                fig_fev = plot_first_event_distribution(
                    events_data.first_kills, events_data.first_deaths
                )
                first_kill_dist = _fig_to_payload(fig_fev)
        except Exception:
            logger.debug("_build_distributions_tab: first_events échoué", exc_info=True)

    return TimeseriesDistributionsTab(
        kda_distribution=kda_distribution,
        first_kill_dist=first_kill_dist,
        correlations=correlations,
    )


# ---------------------------------------------------------------------------
# Point d'entrée public
# ---------------------------------------------------------------------------


def get_timeseries_page(
    player: PlayerContext, request: TimeseriesQueryRequest
) -> TimeseriesPageResponse:
    """Construit la réponse complète pour la page Séries temporelles."""
    df_full = _load_df_full(player)

    try:
        import polars as pl

        is_polars = isinstance(df_full, pl.DataFrame)
    except ImportError:
        is_polars = False

    if not is_polars or (hasattr(df_full, "is_empty") and df_full.is_empty()):
        return _empty_response()

    dff = _apply_filters(df_full, player, request.filters)
    if hasattr(dff, "is_empty") and dff.is_empty():
        return _empty_response()

    # Enrichir avec performance_score
    try:
        from src.data.services.timeseries_service import TimeseriesService

        dff = TimeseriesService.enrich_performance_score(dff, df_full)
    except Exception:
        logger.debug("get_timeseries_page: enrichissement PS échoué", exc_info=True)

    total_matches = len(dff)

    return TimeseriesPageResponse(
        total_matches=total_matches,
        summary_tab=_build_summary_tab(dff, df_full),
        cumul_tab=_build_cumul_tab(dff),
        form_tab=_build_form_tab(dff),
        intensity_tab=_build_intensity_tab(dff, player),
        distributions_tab=_build_distributions_tab(dff, player),
    )


def _empty_response() -> TimeseriesPageResponse:
    """Retourne une réponse vide lorsqu'il n'y a pas de données."""
    return TimeseriesPageResponse(
        total_matches=0,
        summary_tab=TimeseriesSummaryTab(kpi_cards=[]),
        cumul_tab=TimeseriesCumulTab(),
        form_tab=TimeseriesFormTab(),
        intensity_tab=TimeseriesIntensityTab(),
        distributions_tab=TimeseriesDistributionsTab(correlations=[]),
    )
