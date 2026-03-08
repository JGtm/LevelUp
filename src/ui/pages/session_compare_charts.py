"""Graphiques et tableaux de comparaison de sessions.

Ce module contient les fonctions de visualisation extraites de
session_compare.py pour respecter la limite de 800 lignes par fichier :
- Tableau historique des parties
- Radar chart comparatif
- Graphique en barres comparatif
- Tendance de participation
"""

from __future__ import annotations

from typing import TYPE_CHECKING

import plotly.graph_objects as go
import polars as pl
import streamlit as st

from src.ui.chart_utils import safe_chart_render
from src.ui.i18n import t
from src.ui.pages._session_compare_history import (  # noqa: F401
    render_session_history_table,
)
from src.ui.streamlit_modern import PLOTLY_STATIC_CONFIG, fragment_if_available
from src.visualization._compat import (
    DataFrameLike,
    ensure_polars,
)

if TYPE_CHECKING:
    pass


# Couleurs distinctes pour les sessions (contraste élevé, accessible daltoniens)
SESSION_COLORS = {
    "session_a": "#E74C3C",  # Rouge corail
    "session_a_fill": "rgba(231, 76, 60, 0.3)",
    "session_b": "#3498DB",  # Bleu vif
    "session_b_fill": "rgba(52, 152, 219, 0.3)",
    "historical": "#9B59B6",  # Violet
    "historical_fill": "rgba(155, 89, 182, 0.2)",
}


# ════════════════════════════════════════════════════════════════════════════
# Radar chart comparatif
# ════════════════════════════════════════════════════════════════════════════


@fragment_if_available
def render_comparison_radar_chart(
    perf_a: dict,
    perf_b: dict,
    hist_avg: dict | None = None,
) -> None:
    """Affiche le radar chart comparatif avec moyenne historique optionnelle.

    Args:
        perf_a: Métriques de la session A.
        perf_b: Métriques de la session B.
        hist_avg: Moyenne historique des sessions similaires (optionnel).
    """
    categories = [t("sc_radar_kd"), t("sc_radar_win"), t("col_accuracy")]

    def _normalize_for_radar(kd, wr, acc):
        kd_norm = min(100, (kd or 0) * 50)  # F/M 2.0 = 100
        wr_norm = wr or 0  # Déjà en %
        acc_norm = acc if acc is not None else 50  # Déjà en %
        return [kd_norm, wr_norm, acc_norm]

    values_a = _normalize_for_radar(perf_a["kd_ratio"], perf_a["win_rate"], perf_a["accuracy"])
    values_b = _normalize_for_radar(perf_b["kd_ratio"], perf_b["win_rate"], perf_b["accuracy"])

    fig_radar = go.Figure()

    # Moyenne historique en fond (si disponible)
    hist_n = int((hist_avg or {}).get("session_count", 0) or 0)
    if hist_avg and hist_n >= 1:
        values_hist = _normalize_for_radar(
            hist_avg.get("kd_ratio"), hist_avg.get("win_rate"), hist_avg.get("accuracy")
        )
        suffix = " ⚠️" if hist_n < 3 else ""
        fig_radar.add_trace(
            go.Scatterpolar(
                r=values_hist + [values_hist[0]],
                theta=categories + [categories[0]],
                fill="toself",
                name=t("sc_hist_avg_trace", n=hist_n, suffix=suffix),
                line_color=SESSION_COLORS["historical"],
                fillcolor=SESSION_COLORS["historical_fill"],
                line={"dash": "dot"},
            )
        )

    fig_radar.add_trace(
        go.Scatterpolar(
            r=values_a + [values_a[0]],  # Fermer le polygone
            theta=categories + [categories[0]],
            fill="toself",
            name="Session A",
            line_color=SESSION_COLORS["session_a"],
            fillcolor=SESSION_COLORS["session_a_fill"],
        )
    )

    fig_radar.add_trace(
        go.Scatterpolar(
            r=values_b + [values_b[0]],
            theta=categories + [categories[0]],
            fill="toself",
            name="Session B",
            line_color=SESSION_COLORS["session_b"],
            fillcolor=SESSION_COLORS["session_b_fill"],
        )
    )

    fig_radar.update_layout(
        polar={
            "radialaxis": {"visible": True, "range": [0, 100]},
            "bgcolor": "rgba(0,0,0,0)",
        },
        showlegend=True,
        paper_bgcolor="rgba(0,0,0,0)",
        plot_bgcolor="rgba(0,0,0,0)",
        font={"color": "#E0E0E0"},
        height=400,
    )

    with safe_chart_render():
        if fig_radar is not None:
            st.plotly_chart(fig_radar, width="stretch", config=PLOTLY_STATIC_CONFIG)
        else:
            st.info(t("insufficient_data_chart"))


# ════════════════════════════════════════════════════════════════════════════
# Graphique en barres comparatif
# ════════════════════════════════════════════════════════════════════════════


def _prepare_bar_metrics(
    perf_a: dict,
    perf_b: dict,
    hist_avg: dict | None = None,
) -> dict:
    """Prépare les données métriques pour le graphique en barres.

    Args:
        perf_a: Métriques de la session A.
        perf_b: Métriques de la session B.
        hist_avg: Moyenne historique (optionnel).

    Returns:
        Dict contenant les valeurs préparées pour A, B et l'historique.
    """

    def _per_match(total: float | int | None, matches: int | None) -> float:
        m = int(matches or 0)
        if m <= 0:
            return 0.0
        try:
            return float(total or 0.0) / float(m)
        except Exception:
            return 0.0

    left_metrics = [t("sc_kills_per_match"), t("sc_deaths_per_match"), t("sc_kd_ratio")]
    right_metric = t("sc_radar_win")

    a_left = [
        _per_match(perf_a.get("kills"), perf_a.get("matches")),
        _per_match(perf_a.get("deaths"), perf_a.get("matches")),
        float(perf_a.get("kd_ratio") or 0.0),
    ]
    b_left = [
        _per_match(perf_b.get("kills"), perf_b.get("matches")),
        _per_match(perf_b.get("deaths"), perf_b.get("matches")),
        float(perf_b.get("kd_ratio") or 0.0),
    ]
    a_wr = float(perf_a.get("win_rate") or 0.0)
    b_wr = float(perf_b.get("win_rate") or 0.0)

    result: dict = {
        "left_metrics": left_metrics,
        "right_metric": right_metric,
        "a_left": a_left,
        "b_left": b_left,
        "a_wr": a_wr,
        "b_wr": b_wr,
    }

    hist_n = int((hist_avg or {}).get("session_count", 0) or 0)
    if hist_avg and hist_n >= 1:
        result["hist"] = {
            "h_left": [
                float(hist_avg.get("kills_per_match", 0) or 0.0),
                float(hist_avg.get("deaths_per_match", 0) or 0.0),
                float(hist_avg.get("kd_ratio", 0) or 0.0),
            ],
            "h_wr": float(hist_avg.get("win_rate", 0) or 0.0),
            "name": t("sc_hist_avg_trace", n=hist_n, suffix=(" ⚠️" if hist_n < 3 else "")),
        }

    return result


def _add_historical_traces(
    fig: go.Figure,
    metrics: dict,
    left_metrics: list[str],
    right_metric: str,
) -> None:
    """Ajoute les traces de la moyenne historique au graphique en barres."""
    h = metrics["hist"]
    hist_marker = {
        "color": SESSION_COLORS["historical"],
        "pattern": {"shape": ".", "fgcolor": "rgba(255,255,255,0.75)", "solidity": 0.10},
    }
    fig.add_trace(
        go.Bar(
            name=h["name"],
            x=left_metrics,
            y=h["h_left"],
            marker=hist_marker,
            opacity=0.45,
            hovertemplate="%{x} (moy. hist): %{y:.2f}<extra></extra>",
            legendgroup="H",
            showlegend=True,
        )
    )
    fig.add_trace(
        go.Bar(
            name=h["name"],
            x=[right_metric],
            y=[h["h_wr"]],
            marker=hist_marker,
            opacity=0.45,
            hovertemplate="%{x} (moy. hist): %{y:.1f}%<extra></extra>",
            legendgroup="H",
            showlegend=False,
            yaxis="y2",
        )
    )


def _build_bar_chart_figure(metrics: dict) -> go.Figure:
    """Construit la figure Plotly du graphique en barres comparatif.

    Args:
        metrics: Données préparées par _prepare_bar_metrics.

    Returns:
        Figure Plotly configurée.
    """
    left_metrics = metrics["left_metrics"]
    right_metric = metrics["right_metric"]

    fig_bar = go.Figure()

    # Axe gauche : frags/morts/ratio
    fig_bar.add_trace(
        go.Bar(
            name="Session A",
            x=left_metrics,
            y=metrics["a_left"],
            marker_color=SESSION_COLORS["session_a"],
            hovertemplate="%{x} (A): %{y:.2f}<extra></extra>",
            legendgroup="A",
            showlegend=True,
        )
    )
    fig_bar.add_trace(
        go.Bar(
            name="Session B",
            x=left_metrics,
            y=metrics["b_left"],
            marker_color=SESSION_COLORS["session_b"],
            hovertemplate="%{x} (B): %{y:.2f}<extra></extra>",
            legendgroup="B",
            showlegend=True,
        )
    )

    # Axe droit : victoire (%)
    fig_bar.add_trace(
        go.Bar(
            name="Session A",
            x=[right_metric],
            y=[metrics["a_wr"]],
            marker_color=SESSION_COLORS["session_a"],
            hovertemplate="%{x} (A): %{y:.1f}%<extra></extra>",
            legendgroup="A",
            showlegend=False,
            yaxis="y2",
        )
    )
    fig_bar.add_trace(
        go.Bar(
            name="Session B",
            x=[right_metric],
            y=[metrics["b_wr"]],
            marker_color=SESSION_COLORS["session_b"],
            hovertemplate="%{x} (B): %{y:.1f}%<extra></extra>",
            legendgroup="B",
            showlegend=False,
            yaxis="y2",
        )
    )

    # Ajouter la moyenne historique si disponible
    if "hist" in metrics:
        _add_historical_traces(fig_bar, metrics, left_metrics, right_metric)

    fig_bar.update_layout(
        barmode="group",
        paper_bgcolor="rgba(0,0,0,0)",
        plot_bgcolor="rgba(0,0,0,0)",
        font={"color": "#E0E0E0"},
        xaxis={"showgrid": False},
        yaxis={
            "showgrid": True,
            "gridcolor": "rgba(255,255,255,0.1)",
            "title": t("sc_per_match_ratio"),
        },
        yaxis2={
            "title": t("sc_radar_win"),
            "overlaying": "y",
            "side": "right",
            "showgrid": False,
            "rangemode": "tozero",
        },
        height=350,
    )

    return fig_bar


@fragment_if_available
def render_comparison_bar_chart(
    perf_a: dict,
    perf_b: dict,
    hist_avg: dict | None = None,
) -> None:
    """Affiche le graphique en barres comparatif.

    Orchestre la préparation des métriques et la construction de la figure.

    Args:
        perf_a: Métriques de la session A.
        perf_b: Métriques de la session B.
        hist_avg: Moyenne historique des sessions similaires (optionnel).
    """
    metrics = _prepare_bar_metrics(perf_a, perf_b, hist_avg)
    with safe_chart_render():
        fig = _build_bar_chart_figure(metrics)
        if fig is not None:
            st.plotly_chart(fig, width="stretch", config=PLOTLY_STATIC_CONFIG)
        else:
            st.info(t("insufficient_data_chart"))


# ════════════════════════════════════════════════════════════════════════════
# Tendance de participation (PersonalScores)
# ════════════════════════════════════════════════════════════════════════════


@fragment_if_available
def _load_participation_profiles(  # noqa: PLR0913
    repo: object,
    match_ids_a: list,
    match_ids_b: list,
    df_session_a: pl.DataFrame,
    df_session_b: pl.DataFrame,
    db_path: str,
) -> list:
    """Charge et construit les profils de participation pour les deux sessions."""
    from src.visualization.participation_radar import (
        compute_participation_profile,
        get_radar_thresholds,
    )

    def _match_row_from_df(dff: pl.DataFrame) -> dict | None:
        if dff.is_empty():
            return None
        return {
            "deaths": int(dff.get_column("deaths").sum()) if "deaths" in dff.columns else 0,
            "time_played_seconds": float(dff.get_column("time_played_seconds").sum())
            if "time_played_seconds" in dff.columns
            else 600.0 * len(dff),
            "pair_name": dff[0, "pair_name"]
            if "pair_name" in dff.columns and len(dff) > 0
            else None,
        }

    thresholds = get_radar_thresholds(db_path) if db_path else None
    _SCALE_KEYS = ("objectifs", "combat", "support", "score")

    def _session_thresholds(base: dict | None, n_matches: int) -> dict | None:
        if base is None or n_matches <= 1:
            return base
        return {k: v * n_matches if k in _SCALE_KEYS else v for k, v in base.items()}

    profiles = []
    df_a = repo.load_personal_score_awards_as_polars(match_ids=match_ids_a) if match_ids_a else None  # type: ignore[union-attr]
    df_b = repo.load_personal_score_awards_as_polars(match_ids=match_ids_b) if match_ids_b else None  # type: ignore[union-attr]

    if (df_a is None or df_a.is_empty()) and (df_b is None or df_b.is_empty()):
        return []

    if df_a is not None and not df_a.is_empty():
        match_row_a = _match_row_from_df(df_session_a)
        profile_a = compute_participation_profile(
            df_a,
            match_row=match_row_a,
            name="Session A",
            color=SESSION_COLORS["session_a"],
            pair_name=match_row_a.get("pair_name") if match_row_a else None,
            thresholds=_session_thresholds(thresholds, len(match_ids_a)),
        )
        profiles.append(profile_a)

    if df_b is not None and not df_b.is_empty():
        match_row_b = _match_row_from_df(df_session_b)
        profile_b = compute_participation_profile(
            df_b,
            match_row=match_row_b,
            name="Session B",
            color=SESSION_COLORS["session_b"],
            pair_name=match_row_b.get("pair_name") if match_row_b else None,
            thresholds=_session_thresholds(thresholds, len(match_ids_b)),
        )
        profiles.append(profile_b)
    return profiles


def render_participation_trend_section(  # noqa: C901
    df_session_a: DataFrameLike,
    df_session_b: DataFrameLike,
    db_path: str,
    xuid: str,
) -> None:
    """Affiche la tendance de participation entre deux sessions (Sprint 8.2)."""
    from src.data.repositories import DuckDBRepository

    df_session_a = ensure_polars(df_session_a)
    df_session_b = ensure_polars(df_session_b)

    try:
        repo = DuckDBRepository(db_path, xuid)
        if not repo.has_personal_score_awards():
            return

        match_ids_a = (
            df_session_a.get_column("match_id").to_list() if not df_session_a.is_empty() else []
        )
        match_ids_b = (
            df_session_b.get_column("match_id").to_list() if not df_session_b.is_empty() else []
        )
        if not match_ids_a and not match_ids_b:
            return

        profiles = _load_participation_profiles(
            repo, match_ids_a, match_ids_b, df_session_a, df_session_b, db_path
        )
        if not profiles:
            return

        from src.ui.pages._session_compare_viz import _build_participation_bar_chart

        st.markdown("---")
        st.markdown(t("sc_participation_profile"))
        st.caption(t("sc_participation_comparison"))

        with safe_chart_render():
            fig = _build_participation_bar_chart(profiles)
            if fig is not None:
                st.plotly_chart(fig, width="stretch", config=PLOTLY_STATIC_CONFIG)
            else:
                st.info(t("insufficient_data_chart"))

    except Exception:
        pass
