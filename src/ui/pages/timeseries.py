"""Page Séries temporelles.

Graphes d'évolution des statistiques dans le temps.

8bis.A6 : Downsampling pour graphiques avec >200 points (gain ~35% rendu).
"""

from __future__ import annotations

import logging
import re

import polars as pl
import streamlit as st

from src.data.services.timeseries_service import TimeseriesService
from src.ui.chart_utils import safe_chart_render
from src.ui.components.browser_storage import hints_visible
from src.ui.i18n import get_lang, t
from src.ui.pages._timeseries_distributions import render_correlations, render_distributions
from src.ui.pages._timeseries_intensity import render_intensity_heatmap as _render_intensity_heatmap
from src.ui.pages._timeseries_weapons import render_weapon_kills_chart as _render_weapon_kills_chart
from src.ui.pages.timeseries_skill_rank import render_skill_rank_progression
from src.ui.pages.win_loss import (
    _render_heatmap_section as _render_wl_heatmap_section,
)
from src.ui.pages.win_loss import (
    _render_map_mode_breakdown,
    _render_outcomes_over_time,
    _render_personal_score_section,
    _render_streak_section,
    _render_top_by_week,
    _render_winrate_perf_vs_history,
)
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, PLOTLY_STATIC_CONFIG, fragment_if_available
from src.visualization._chart_series import downsample_for_plot
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization._plot_options import PlotOptions
from src.visualization.distributions import (
    plot_first_event_distribution,
    plot_kda_distribution,
)
from src.visualization.performance import (
    plot_cumulative_kd_with_ci,
    plot_ewma_kd,
    plot_net_score_per_hour,
    plot_regression_trend,
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
# Sous-fonctions de rendu extraites du monolithe (Sprint 16)
# =============================================================================

logger = logging.getLogger(__name__)


@fragment_if_available
def _render_kda_section(
    dff: pl.DataFrame,
    lang: str = "fr",
    *,
    db_path: str | None = None,
    xuid: str | None = None,
) -> None:
    """Affiche le graphe KDA et sa distribution."""
    with safe_chart_render():
        # 8bis.A6 : Downsampling pour performance
        df_plot = downsample_for_plot(dff)
        fig = plot_timeseries(df_plot, lang=lang)
        if fig is not None:
            st.plotly_chart(fig, width="stretch", config=PLOTLY_CLEAN_CONFIG)
        else:
            st.info(t("insufficient_data_chart"))

    st.subheader(t("ts_fda"))
    valid = dff.drop_nulls(subset=["kda"]) if "kda" in dff.columns else pl.DataFrame()
    if valid.is_empty():
        st.info(t("ts_fda_unavailable"))
    else:
        m = st.columns(1)
        m[0].metric(
            t("ts_kda_mean_label"), f"{valid['kda'].mean():.2f}", label_visibility="collapsed"
        )
        with safe_chart_render():
            fig_dist = plot_kda_distribution(dff, lang=lang)
            if fig_dist is not None:
                st.plotly_chart(fig_dist, width="stretch", config=PLOTLY_STATIC_CONFIG)
            else:
                st.info(t("insufficient_data_chart"))

    if db_path and xuid:
        _render_weapon_kills_chart(dff, db_path=db_path, xuid=xuid, lang=lang)


@fragment_if_available
def _render_cumulative_performance(dff: pl.DataFrame, lang: str = "fr") -> None:
    """Affiche les graphes de progression cumulée et de forme récente (refonte)."""
    st.subheader(t("ts_cumulative"))
    if hints_visible():
        st.caption(t("ts_cumulative_caption"))

    cumul = TimeseriesService.compute_cumulative_metrics(dff)
    if cumul is None:
        st.info(t("ts_col_missing_cumul"))
        return

    alpha = 0.20
    outcome_values: list[int | None] | None = None
    if "outcome" in dff.columns:
        outcome_values = dff.sort("start_time")["outcome"].to_list()
    st.markdown(f"#### {t('ts_section_cumulative')}")

    nph_data = TimeseriesService.compute_rolling_net_score_per_hour(dff)
    if nph_data is not None:
        with safe_chart_render():
            fig_nph = plot_net_score_per_hour(
                nph_data,
                lang=lang,
                outcome_values=outcome_values,
            )
            st.plotly_chart(fig_nph, width="stretch", config=PLOTLY_CLEAN_CONFIG)
        _render_note(t("ts_note_nph"))
    else:
        st.info(t("ts_nph_unavailable"))

    ci_data = TimeseriesService.compute_cumulative_kd_with_ci(dff)
    if ci_data is not None:
        with safe_chart_render():
            fig_ci = plot_cumulative_kd_with_ci(
                ci_data,
                lang=lang,
                outcome_values=outcome_values,
            )
            st.plotly_chart(fig_ci, width="stretch", config=PLOTLY_CLEAN_CONFIG)
        _render_note(t("ts_note_ci"))
    else:
        st.info(t("ts_ci_unavailable"))

    # ── Section : Forme récente ───────────────────────────────────────────────
    st.markdown(f"#### {t('ts_section_recent')}")

    ewma_data = TimeseriesService.compute_ewma_kd(dff, alpha=alpha)
    regression = TimeseriesService.compute_linear_regression_kd(ewma_data)

    with safe_chart_render():
        fig_ewma = plot_ewma_kd(
            ewma_data,
            regression_data=regression,
            outcome_values=outcome_values,
            opts=PlotOptions(lang=lang),
        )
        st.plotly_chart(fig_ewma, width="stretch", config=PLOTLY_CLEAN_CONFIG)
    _render_note(t("ts_note_ewma"))

    if cumul.has_enough_for_trend:
        st.markdown(f"##### {t('ts_regression_subheader')}")
        with safe_chart_render():
            fig_reg = plot_regression_trend(regression, lang=lang)
            st.plotly_chart(fig_reg, width="stretch", config=PLOTLY_STATIC_CONFIG)
        _render_note(t("ts_note_regression"))
    else:
        st.info(t("ts_trend_min_matches"))


def _render_note(text: str) -> None:
    """Encadré conclusif discret sous chaque graphe (thème Halo)."""
    if not hints_visible():
        return
    lines = text.split("\n")
    parts: list[str] = []
    in_list = False
    for line in lines:
        if line.startswith("- "):
            if not in_list:
                parts.append("<ul>")
                in_list = True
            item = re.sub(r"\*\*(.+?)\*\*", r"<strong>\1</strong>", line[2:])
            parts.append(f"<li>{item}</li>")
        else:
            if in_list:
                parts.append("</ul>")
                in_list = False
            if line.strip():
                item = re.sub(r"\*\*(.+?)\*\*", r"<strong>\1</strong>", line)
                parts.append(f"<p>{item}</p>")
    if in_list:
        parts.append("</ul>")
    st.markdown(
        f'<div class="ts-note">{"".join(parts)}</div>',
        unsafe_allow_html=True,
    )


@fragment_if_available
def _render_first_event_section(
    dff: pl.DataFrame,
    db_path: str | None,
    xuid: str | None,
    lang: str = "fr",
) -> None:
    """Affiche la distribution du premier frag / première mort (Sprint 5.4.4)."""
    st.subheader(t("ts_first_event"))
    st.caption(t("ts_first_event_caption")) if hints_visible() else None

    _match_ids = dff["match_id"].cast(pl.Utf8).to_list() if "match_id" in dff.columns else []
    first_event = TimeseriesService.load_first_event_times(db_path, xuid, _match_ids)

    if first_event.available:
        with safe_chart_render():
            fig_events = plot_first_event_distribution(
                first_event.first_kills,
                first_event.first_deaths,
                title=None,
                lang=lang,
            )
            if fig_events is not None:
                st.plotly_chart(fig_events, width="stretch", config=PLOTLY_STATIC_CONFIG)
            else:
                st.info(t("insufficient_data_chart"))
    else:
        st.info(t("ts_first_event_no_data"))


@fragment_if_available
def _render_advanced_sections(  # noqa: PLR0912
    dff: pl.DataFrame,
    df_full: pl.DataFrame | None,
    db_path: str | None,
    xuid: str | None,
    lang: str = "fr",
) -> None:
    """Affiche Performance, Assists, Stats/min, Average Life, Spree, Sprint 7."""
    history = df_full if df_full is not None else dff
    st.subheader(t("ts_performance"))
    with safe_chart_render():
        fig_perf = plot_performance_timeseries(dff, df_history=history, lang=lang)
        if fig_perf is not None:
            st.plotly_chart(fig_perf, width="stretch", config=PLOTLY_CLEAN_CONFIG)
        else:
            st.info(t("insufficient_data_chart"))

    st.subheader(t("ts_assists"))
    with safe_chart_render():
        fig_assists = plot_assists_timeseries(dff, lang=lang)
        if fig_assists is not None:
            st.plotly_chart(fig_assists, width="stretch", config=PLOTLY_CLEAN_CONFIG)
        else:
            st.info(t("insufficient_data_chart"))

    st.subheader(t("ts_per_minute"))
    with safe_chart_render():
        fig_spm = plot_per_minute_timeseries(dff, lang=lang)
        if fig_spm is not None:
            st.plotly_chart(fig_spm, width="stretch", config=PLOTLY_CLEAN_CONFIG)
        else:
            st.info(t("insufficient_data_chart"))

    st.subheader(t("ts_lifespan"))
    if dff.drop_nulls(subset=["average_life_seconds"]).is_empty():
        st.info(t("ts_lifespan_unavailable"))
    else:
        with safe_chart_render():
            fig_life = plot_average_life(dff, lang=lang)
            if fig_life is not None:
                st.plotly_chart(fig_life, width="stretch", config=PLOTLY_CLEAN_CONFIG)
            else:
                st.info(t("insufficient_data_chart"))

    _render_spree_section(dff, db_path, xuid, lang=lang)
    _render_sprint7_sections(dff, lang=lang)


def _render_spree_section(
    dff: pl.DataFrame,
    db_path: str | None,
    xuid: str | None,
    lang: str = "fr",
) -> None:
    """Affiche la section Folie meurtrière / Tirs à la tête / Frags parfaits."""
    st.subheader(t("ts_spree"))

    from src.app.state import get_page_context

    _ctx_db, _ctx_xuid, _ = get_page_context()
    _db_path = db_path or _ctx_db
    _xuid = xuid or _ctx_xuid
    _match_ids = dff["match_id"].cast(pl.Utf8).to_list() if "match_id" in dff.columns else []
    pk_data = TimeseriesService.load_perfect_kills(_db_path, _xuid, _match_ids)

    with safe_chart_render():
        fig_spree = plot_spree_headshots_accuracy(dff, perfect_counts=pk_data.counts, lang=lang)
        if fig_spree is not None:
            st.plotly_chart(fig_spree, width="stretch", config=PLOTLY_CLEAN_CONFIG)
        else:
            st.info(t("insufficient_data_chart"))


def _render_sprint7_sections(dff: pl.DataFrame, lang: str = "fr") -> None:
    """Affiche les sections Sprint 7 : tirs, dégâts, rang."""
    # 7.5 — Tirs et précision
    _has_shots = any(c in dff.columns for c in ("shots_fired", "shots_hit"))
    if _has_shots:
        st.divider()
        st.subheader(t("ts_shots"))
        if hints_visible():
            st.caption(t("ts_shots_caption"))
        with safe_chart_render():
            fig_shots = plot_shots_accuracy(dff, lang=lang)
            if fig_shots is not None:
                st.plotly_chart(fig_shots, width="stretch", config=PLOTLY_CLEAN_CONFIG)
            else:
                st.info(t("insufficient_data_chart"))

    # 7.4 — Dégâts infligés vs subis
    _has_damage = any(c in dff.columns for c in ("damage_dealt", "damage_taken"))
    if _has_damage:
        st.divider()
        st.subheader(t("ts_damage"))
        if hints_visible():
            st.caption(t("ts_damage_caption"))
        with safe_chart_render():
            fig_damage = plot_damage_dealt_taken(dff, lang=lang)
            if fig_damage is not None:
                st.plotly_chart(fig_damage, width="stretch", config=PLOTLY_CLEAN_CONFIG)
            else:
                st.info(t("insufficient_data_chart"))

    # 7.3 — Rang et score personnel
    _has_rank_score = "rank" in dff.columns or "personal_score" in dff.columns
    if _has_rank_score:
        st.divider()
        st.subheader(t("ts_rank_score"))
        if hints_visible():
            st.caption(t("ts_rank_score_caption"))
        with safe_chart_render():
            fig_rank = plot_rank_score(dff, lang=lang)
            if fig_rank is not None:
                st.plotly_chart(fig_rank, width="stretch", config=PLOTLY_CLEAN_CONFIG)
            else:
                st.info(t("insufficient_data_chart"))


# =============================================================================
# Point d'entrée (orchestrateur réduit)
# =============================================================================


@fragment_if_available
def render_timeseries_page(  # noqa: PLR0913
    dff: DataFrameLike,
    df_full: DataFrameLike | None = None,
    *,
    base: DataFrameLike | None = None,
    picked_session_labels: list[str] | None = None,
    db_path: str | None = None,
    xuid: str | None = None,
    db_key: tuple[int, int] | None = None,
) -> None:
    """Affiche la page Séries temporelles (5 onglets).

    Args:
        dff: DataFrame filtré des matchs.
        df_full: DataFrame complet pour le calcul du score relatif.
        base: DataFrame de base (toutes les parties hors Firefight) pour les comparaisons carte.
        picked_session_labels: Labels des sessions sélectionnées.
        db_path: Chemin vers la DB (optionnel, pour les features DuckDB).
        xuid: XUID du joueur (optionnel, pour les features DuckDB).
        db_key: Clé de cache de la DB.
    """
    dff = ensure_polars(dff)
    df_full = ensure_polars(df_full) if df_full is not None else None
    base_df = ensure_polars(base) if base is not None else dff
    if dff.is_empty():
        st.warning(t("no_matches"))
        return

    logger.debug(
        "render_timeseries_page: %d matchs, base=%d, session_scope=%s",
        len(dff),
        len(base_df),
        bool(picked_session_labels),
    )

    # Calculer le score de performance via service (Sprint 14)
    dff = TimeseriesService.enrich_performance_score(
        dff,
        df_full,
    )

    lang = get_lang()
    current_mode = st.session_state.get("filter_mode")
    is_session_scope = bool(current_mode == "Sessions" and picked_session_labels)

    _tab_summary, _tab_maps, _tab_dist, _tab_prog, _tab_adv = st.tabs(
        [
            t("ts_tab_kda"),
            t("ts_tab_maps"),
            t("ts_tab_distributions"),
            t("ts_tab_advanced"),
            t("ts_tab_progression"),
        ]
    )

    with _tab_summary:
        _render_kda_section(dff, lang=lang, db_path=db_path, xuid=xuid)
        _render_outcomes_over_time(dff, is_session_scope)
        _render_streak_section(dff)

    with _tab_maps:
        _render_map_mode_breakdown(dff)
        _render_winrate_perf_vs_history(dff, base_df)

    with _tab_dist:
        render_distributions(dff, lang=lang)
        render_correlations(dff, lang=lang)

    with _tab_prog:
        _render_first_event_section(dff, db_path, xuid, lang=lang)
        _render_advanced_sections(dff, df_full, db_path, xuid, lang=lang)
        _render_personal_score_section(dff)

    with _tab_adv:
        if db_path and xuid:
            _render_intensity_heatmap(dff, db_path=db_path, xuid=xuid, lang=lang)
        render_skill_rank_progression(dff, db_path, xuid, lang=lang)
        _render_cumulative_performance(dff, lang=lang)
        _render_wl_heatmap_section(dff)
        _render_top_by_week(dff)
