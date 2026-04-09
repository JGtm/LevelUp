"""Graphiques par carte pour la page Coéquipiers.

Extrait de teammates_views.py pour respecter la limite de 500 lignes.
Contient le rendu des charts par carte : lollipop, timeline, bullet,
perf vs historique, heatmap escouade, vue carte 1 coéquipier.
"""

from __future__ import annotations

import logging
from pathlib import Path

import polars as pl
import streamlit as st

logger = logging.getLogger(__name__)

from src.analysis import compute_map_breakdown
from src.ui.chart_utils import safe_chart_render
from src.ui.components.browser_storage import hints_visible
from src.ui.components.info_note import render_info_note
from src.ui.i18n import t
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, PLOTLY_STATIC_CONFIG
from src.visualization import (
    plot_map_perf_vs_history,
    plot_map_winrate_bullet,
    plot_squad_map_heatmap,
    plot_squad_performance_timeline,
)
from src.visualization._compat import DataFrameLike, ensure_polars


def render_map_charts_section(
    sub_all: DataFrameLike,
    full_squad_df: DataFrameLike,
    breakdown_all: DataFrameLike,
    lang: str,
) -> None:
    """Affiche lollipop + timeline + bullet + perf vs historique pour la vue multi-coéquipiers.

    Args:
        sub_all: Matchs filtrés avec les coéquipiers (sélection courante).
        full_squad_df: Tous les matchs avec les coéquipiers (historique complet).
        breakdown_all: Breakdown par carte déjà calculé.
        lang: Langue.
    """
    sub_pl = ensure_polars(sub_all)
    full_pl = ensure_polars(full_squad_df)
    bd_all = ensure_polars(breakdown_all)

    if bd_all.is_empty():
        logger.debug("render_map_charts_section: breakdown vide, affichage info")
        st.info(t("tm_not_enough_matches"))
        return

    view = bd_all.head(20).reverse()
    logger.debug("render_map_charts_section: %d cartes, %d matchs session", len(view), len(sub_pl))

    # Ordre chronologique des cartes (oldest first) depuis la sélection courante
    map_order: list[str] | None = None
    if not sub_pl.is_empty() and "start_time" in sub_pl.columns:
        _map_col = "map_ui" if "map_ui" in sub_pl.columns else "map_name"
        map_order = (
            sub_pl.sort("start_time")
            .unique(subset=[_map_col], keep="first", maintain_order=True)
            .filter(pl.col(_map_col).is_not_null())[_map_col]
            .to_list()
        )

    # C — Bullet win rate vs historique
    if not full_pl.is_empty():
        bd_history = _compute_history_breakdown(full_pl)
        st.markdown(f"##### {t('tm_map_bullet_title')}")
        with safe_chart_render():
            fig_bullet = plot_map_winrate_bullet(view, bd_history, lang=lang, map_order=map_order)
            if fig_bullet is not None:
                st.plotly_chart(fig_bullet, width="stretch", config=PLOTLY_STATIC_CONFIG)

        # Feature 2 — Perf vs historique
        st.markdown(f"##### {t('tm_perf_vs_history_title')}")
        with safe_chart_render():
            fig_perf = plot_map_perf_vs_history(view, bd_history, lang=lang, map_order=map_order)
            if fig_perf is not None:
                st.plotly_chart(fig_perf, width="stretch", config=PLOTLY_STATIC_CONFIG)


def render_squad_heatmap(series: list[tuple[str, DataFrameLike]], lang: str) -> None:
    """Affiche la heatmap de performance par joueur × carte (Feature 1).

    Args:
        series: Liste de (nom_joueur, df_matchs).
        lang: Langue.
    """
    if len(series) < 2:
        return
    st.markdown(f"##### {t('tm_map_squad_heatmap_title')}")
    with safe_chart_render():
        fig_hm = plot_squad_map_heatmap(series, lang=lang)
        if fig_hm is not None:
            st.plotly_chart(fig_hm, width="stretch", config=PLOTLY_STATIC_CONFIG)


def render_single_map_section(
    sub: DataFrameLike,
    dfr: DataFrameLike,
    series: list[tuple[str, DataFrameLike]],
    lang: str,
) -> None:
    """Affiche les graphiques par carte pour la vue 1 coéquipier.

    Args:
        sub: Matchs filtrés avec le coéquipier (sélection courante).
        dfr: Tous les matchs avec le coéquipier (historique complet).
        series: [(me_name, sub), (friend_name, friend_sub)] pour la heatmap.
        lang: Langue.
    """
    sub_pl = ensure_polars(sub)
    dfr_pl = ensure_polars(dfr)
    if sub_pl.is_empty():
        return

    bd_current = compute_map_breakdown(sub_pl)
    if bd_current.is_empty():
        logger.debug("render_single_map_section: breakdown vide pour %d matchs", len(sub_pl))
        return

    st.subheader(t("tm_by_map"))
    view = bd_current.sort("win_rate").head(20).reverse()

    # Ordre chronologique des cartes (oldest first) depuis la sélection courante
    map_order_single: list[str] | None = None
    if "start_time" in sub_pl.columns:
        _map_col = "map_ui" if "map_ui" in sub_pl.columns else "map_name"
        map_order_single = (
            sub_pl.sort("start_time")
            .unique(subset=[_map_col], keep="first", maintain_order=True)
            .filter(pl.col(_map_col).is_not_null())[_map_col]
            .to_list()
        )

    # C + Feature 2 — vs historique (seulement si assez de données)
    if not dfr_pl.is_empty():
        bd_history = _compute_history_breakdown(dfr_pl)
        st.markdown(f"##### {t('tm_map_bullet_title')}")
        with safe_chart_render():
            fig_bullet = plot_map_winrate_bullet(
                view, bd_history, lang=lang, map_order=map_order_single
            )
            if fig_bullet is not None:
                st.plotly_chart(fig_bullet, width="stretch", config=PLOTLY_STATIC_CONFIG)

        # Feature 2 — Perf vs historique
        st.markdown(f"##### {t('tm_perf_vs_history_title')}")
        with safe_chart_render():
            fig_perf = plot_map_perf_vs_history(
                view, bd_history, lang=lang, map_order=map_order_single
            )
            if fig_perf is not None:
                st.plotly_chart(fig_perf, width="stretch", config=PLOTLY_STATIC_CONFIG)
    render_squad_heatmap(series, lang=lang)


def _enrich_me_df_from_shared(
    me_df: pl.DataFrame,
    shared_path: str,
    me_name: str,
    all_match_ids: list[str],
) -> pl.DataFrame:
    """Enrichit me_df avec team_mmr et outcome depuis shared_matches."""
    from src.data.services._teammates_perf_queries import (
        load_outcome_by_match,
        load_team_mmr_by_match,
    )

    mmr_by_match = load_team_mmr_by_match(shared_path, me_name, all_match_ids)
    if mmr_by_match:
        mmr_df = pl.DataFrame(
            [{"match_id": k, "team_mmr": v} for k, v in mmr_by_match.items()],
            schema={"match_id": pl.Utf8, "team_mmr": pl.Float64},
        )
        me_df = me_df.join(mmr_df, on="match_id", how="left")
    outcome_by_match = load_outcome_by_match(shared_path, me_name, all_match_ids)
    if outcome_by_match:
        outcome_df = pl.DataFrame(
            [{"match_id": k, "outcome": v} for k, v in outcome_by_match.items()],
            schema={"match_id": pl.Utf8, "outcome": pl.Int32},
        )
        me_df = me_df.join(outcome_df, on="match_id", how="left")
    return me_df


def render_squad_timeline(
    db_path: str,
    me_name: str,
    friend_names: list[str],
    all_match_ids: list[str],
    lang: str = "fr",
) -> None:
    """Affiche la timeline d'évolution de performance d'escouade (historique complet)."""
    if not all_match_ids:
        return

    from src.analysis._performance_squad import compute_squad_timeseries
    from src.data.services._teammates_perf_queries import (
        load_perf_enrichment_with_session,
    )
    from src.utils.paths import get_shared_matches_path_from_player

    base_dir = Path(db_path).parent.parent

    def _build_df(player_db: str) -> pl.DataFrame:
        enrichment = load_perf_enrichment_with_session(player_db, all_match_ids)
        if not enrichment:
            return pl.DataFrame()
        rows = [
            {
                "match_id": str(mid),
                "performance_score": float(e["perf"]),
                "session_id": str(e["session_id"]) if e["session_id"] is not None else None,
                "session_label": str(e["session_label"])
                if e["session_label"] is not None
                else None,
            }
            for mid, e in enrichment.items()
        ]
        return pl.DataFrame(
            rows,
            schema={
                "match_id": pl.Utf8,
                "performance_score": pl.Float64,
                "session_id": pl.Utf8,
                "session_label": pl.Utf8,
            },
        )

    me_df = _build_df(db_path)
    if me_df.is_empty():
        return

    # Enrichir me_df avec team_mmr et outcome depuis shared_matches
    shared_path = get_shared_matches_path_from_player(db_path)
    if shared_path:
        me_df = _enrich_me_df_from_shared(me_df, str(shared_path), me_name, all_match_ids)

    series: list[tuple[str, pl.DataFrame]] = [(me_name, me_df)]
    for fname in friend_names:
        friend_df = _build_df(str(base_dir / fname / "stats.duckdb"))
        if not friend_df.is_empty():
            series.append((fname, friend_df))

    if len(series) < 2:
        return

    ts = compute_squad_timeseries(series)
    if ts.is_empty():
        return
    key = "squad_timeline_" + "_".join(n for n, _ in series)
    with safe_chart_render():
        fig = plot_squad_performance_timeline(ts, lang=lang)
        if fig is not None:
            st.plotly_chart(fig, width="stretch", config=PLOTLY_CLEAN_CONFIG, key=key)


def render_squad_cadence_section(
    db_path: str,
    xuid_name_map: dict[str, str],
    all_match_ids: list[str],
    lang: str = "fr",
    color_map: dict[str, str] | None = None,
) -> None:
    """Affiche le profil de tempo synchronisé (moyenne de kills/phase par joueur)."""
    if not all_match_ids or len(xuid_name_map) < 2:
        return

    logger.debug(
        "squad_cadence: %d joueurs, %d matchs",
        len(xuid_name_map),
        len(all_match_ids),
    )

    from src.analysis.match_intensity import compute_squad_cadence_profiles
    from src.ui._cache_queries import cached_load_kill_timing_for_matches
    from src.visualization._plot_options import PlotOptions
    from src.visualization.squad_cadence_chart import plot_squad_cadence_profiles

    all_xuids = list(xuid_name_map.keys())
    all_names = list(xuid_name_map.values())
    raw = cached_load_kill_timing_for_matches(
        db_path,
        all_xuids[0],
        tuple(all_match_ids),
        xuids=tuple(all_xuids),
    )
    if not raw:
        return

    events_df = pl.DataFrame(
        raw,
        schema={"match_id": pl.Utf8, "time_ms": pl.Int64, "xuid": pl.Utf8},
    )

    profiles = compute_squad_cadence_profiles(events_df, xuid_name_map, n_buckets=10)
    if profiles.is_empty():
        return

    st.subheader(t("tm_squad_cadence"))
    if hints_visible():
        st.caption(t("tm_squad_cadence_caption"))

    with safe_chart_render():
        fig = plot_squad_cadence_profiles(
            profiles,
            all_names,
            opts=PlotOptions(lang=lang, height_px=340),
            color_map=color_map,
        )
        if fig is not None:
            st.plotly_chart(fig, width="stretch", config=PLOTLY_CLEAN_CONFIG)
            render_info_note(t("tm_note_cadence"))
        else:
            st.info(t("tm_squad_cadence_no_data"))


def _compute_history_breakdown(full_df: pl.DataFrame) -> pl.DataFrame:
    """Calcule le breakdown historique complet (min_matches=1) depuis un DataFrame de matchs."""
    from src.ui.i18n import get_lang

    # Même correction que win_loss.py : full_df vient du df brut (sans map_ui).
    # Sans map_ui, compute_map_breakdown utiliserait map_name (EN) → inner join
    # avec view (FR, calculé depuis sub_all/bd_all qui a map_ui) raterait les 4 cartes FR.
    if "map_ui" not in full_df.columns and "map_name_fr" in full_df.columns and get_lang() == "fr":
        full_df = full_df.with_columns(
            pl.coalesce(
                [pl.col("map_name_fr").cast(pl.Utf8), pl.col("map_name").cast(pl.Utf8)]
            ).alias("map_ui")
        )
    return compute_map_breakdown(full_df)
