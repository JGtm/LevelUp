"""Service Comparaison de sessions — Slice 3C.

Point d'entrée : ``get_session_compare``.
Toutes les importations src.* sont lazy pour permettre le mocking en tests.
"""

from __future__ import annotations

import logging

from apps.api.app._db_helpers import Outcome
from apps.api.app._pure_bridge import infer_session_dominant_category
from apps.api.app.deps.players import PlayerContext
from apps.api.app.schemas.career import PlotlyFigurePayload
from apps.api.app.schemas.timeseries import (
    SessionCompareEntry,
    SessionCompareMetricRow,
    SessionCompareRequest,
    SessionCompareResponse,
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
# Chargement des données
# ---------------------------------------------------------------------------


def _load_df_full(player: PlayerContext):  # type: ignore[return]
    """Charge le DataFrame complet des matchs du joueur."""
    try:
        from apps.api.app.services.match_history_service import _load_matches_full

        return _load_matches_full(player)
    except Exception:
        logger.warning("_load_df_full: impossible de charger les matchs", exc_info=True)
        import polars as pl

        return pl.DataFrame()


def _get_available_sessions(df_full) -> list[str]:
    """Extrait la liste des sessions disponibles (triées par date décroissante)."""
    try:
        import polars as pl

        if df_full is None or (hasattr(df_full, "is_empty") and df_full.is_empty()):
            return []
        if "session_label" not in df_full.columns:
            return []

        agg = (
            df_full.filter(pl.col("session_label").is_not_null())
            .group_by(["session_label"])
            .agg(pl.col("start_time").max().alias("last_time"))
            .sort("last_time", descending=True)
        )
        return agg["session_label"].cast(pl.Utf8).to_list()
    except Exception:
        logger.debug("_get_available_sessions: erreur", exc_info=True)
        return []


def _filter_by_session(df_full, session_label: str):  # type: ignore[return]
    """Filtre le DataFrame par session_label."""
    try:
        import polars as pl

        return df_full.filter(pl.col("session_label") == session_label)
    except Exception:
        return df_full


# ---------------------------------------------------------------------------
# Construction d'une SessionCompareEntry
# ---------------------------------------------------------------------------


def _build_session_entry(df_session, label: str) -> SessionCompareEntry:
    """Construit une SessionCompareEntry depuis le DataFrame d'une session."""
    total = len(df_session)
    wins = 0
    losses = 0
    kda: float | None = None
    perf_score: float | None = None
    with_friends = False
    dominant_category: str | None = None
    start_time: str | None = None
    end_time: str | None = None

    try:
        if "outcome" in df_session.columns:
            wins = int((df_session["outcome"] == Outcome.WIN).sum())
            losses = int((df_session["outcome"] == Outcome.LOSS).sum())

        if "kills" in df_session.columns and "deaths" in df_session.columns:
            kills = float(df_session["kills"].sum())
            deaths = float(df_session["deaths"].sum())
            assists = float(df_session["assists"].sum()) if "assists" in df_session.columns else 0.0
            kda = round((kills + assists) / deaths, 2) if deaths > 0 else round(kills + assists, 2)

        from src.analysis.performance_score import compute_session_performance_score_v2

        result = compute_session_performance_score_v2(df_session)
        perf_score = round(float(result.get("score") or 0.0), 1)

        if "is_with_friends" in df_session.columns:
            with_friends = bool(df_session["is_with_friends"].max())

        dominant_category = infer_session_dominant_category(df_session)

        if "start_time" in df_session.columns:
            times = df_session["start_time"].drop_nulls()
            if not times.is_empty():
                start_time = str(times.min())
                end_time = str(times.max())
    except Exception:
        logger.debug("_build_session_entry: erreur", exc_info=True)

    return SessionCompareEntry(
        session_label=label,
        start_time=start_time,
        end_time=end_time,
        total_matches=total,
        wins=wins,
        losses=losses,
        kda=kda,
        performance_score=perf_score,
        with_friends=with_friends,
        dominant_category=dominant_category,
    )


# ---------------------------------------------------------------------------
# Construction des métriques de comparaison
# ---------------------------------------------------------------------------


def _winner(val_a: float | None, val_b: float | None) -> str | None:
    """Détermine qui est 'gagnant' selon la valeur (plus grand = meilleur)."""
    if val_a is None or val_b is None:
        return None
    if abs(val_a - val_b) < 1e-6:
        return "tie"
    return "a" if val_a > val_b else "b"


def _build_metric_rows(perf_a: dict, perf_b: dict) -> list[SessionCompareMetricRow]:
    """Construit les lignes de comparaison métrique A vs B."""
    rows: list[SessionCompareMetricRow] = []

    metrics = [
        ("score", "Score perf.", lambda v: f"{v:.1f}" if v is not None else "N/A"),
        ("kd_ratio", "K/D", lambda v: f"{v:.2f}" if v is not None else "N/A"),
        ("win_rate", "Victoires (%)", lambda v: f"{v:.1f} %" if v is not None else "N/A"),
        ("accuracy", "Précision (%)", lambda v: f"{v:.1f} %" if v is not None else "N/A"),
        (
            "kills_per_match",
            "Kills/match",
            lambda v: f"{v:.1f}" if v is not None else "N/A",
        ),
        (
            "avg_life_seconds",
            "Vie moy. (s)",
            lambda v: f"{v:.1f}" if v is not None else "N/A",
        ),
    ]

    for key, label, fmt in metrics:
        va = perf_a.get(key)
        vb = perf_b.get(key)
        va_float = float(va) if va is not None else None
        vb_float = float(vb) if vb is not None else None

        delta: str | None = None
        if va_float is not None and vb_float is not None and vb_float != 0:
            diff = va_float - vb_float
            delta = f"{'+' if diff >= 0 else ''}{diff:.2f}"

        rows.append(
            SessionCompareMetricRow(
                key=key,
                label=label,
                value_a=fmt(va_float),
                value_b=fmt(vb_float),
                delta=delta,
                winner=_winner(va_float, vb_float),
            )
        )

    return rows


# ---------------------------------------------------------------------------
# Construction du radar chart
# ---------------------------------------------------------------------------


def _build_radar_chart(perf_a: dict, perf_b: dict) -> PlotlyFigurePayload | None:
    """Construit le radar chart comparatif Session A vs Session B."""
    try:
        import plotly.graph_objects as go

        from src.visualization.theme import apply_halo_plot_style

        categories = ["K/D", "Victoires (%)", "Précision (%)"]

        def normalize(kd, wr, acc) -> list[float]:
            kd_norm = min(100.0, (kd or 0.0) * 50.0)
            wr_norm = float(wr or 0.0)
            acc_norm = float(acc if acc is not None else 50.0)
            return [kd_norm, wr_norm, acc_norm]

        vals_a = normalize(perf_a.get("kd_ratio"), perf_a.get("win_rate"), perf_a.get("accuracy"))
        vals_b = normalize(perf_b.get("kd_ratio"), perf_b.get("win_rate"), perf_b.get("accuracy"))

        fig = go.Figure()
        for vals, name, color in [
            (vals_a, "Session A", "#4ecdc4"),
            (vals_b, "Session B", "#f7b731"),
        ]:
            fig.add_trace(
                go.Scatterpolar(
                    r=vals + [vals[0]],
                    theta=categories + [categories[0]],
                    fill="toself",
                    name=name,
                    line_color=color,
                )
            )
        fig.update_layout(polar={"radialaxis": {"range": [0, 100]}})
        fig = apply_halo_plot_style(fig, height=350)
        return _fig_to_payload(fig)
    except Exception:
        logger.debug("_build_radar_chart: erreur", exc_info=True)
        return None


# ---------------------------------------------------------------------------
# Construction du chart K/D progression
# ---------------------------------------------------------------------------


def _build_kd_progression_chart(
    df_a, df_b, label_a: str, label_b: str
) -> PlotlyFigurePayload | None:
    """Construit le graphique de progression K/D cumulé pour les deux sessions."""
    try:
        from src.visualization.performance import plot_cumulative_comparison

        fig = plot_cumulative_comparison(df_a, df_b, label_a=label_a, label_b=label_b)
        return _fig_to_payload(fig)
    except Exception:
        logger.debug("_build_kd_progression_chart: erreur", exc_info=True)
        return None


# ---------------------------------------------------------------------------
# Construction des tableaux maps/modes
# ---------------------------------------------------------------------------


def _build_maps_table(df_a, df_b) -> list[dict]:
    """Construit le tableau comparatif des maps."""
    try:
        import polars as pl

        rows: list[dict] = []
        if "map_ui" not in df_a.columns:
            return rows

        maps = set()
        if "map_ui" in df_a.columns:
            maps.update(df_a["map_ui"].drop_nulls().to_list())
        if "map_ui" in df_b.columns:
            maps.update(df_b["map_ui"].drop_nulls().to_list())

        for m in sorted(maps):
            da = df_a.filter(pl.col("map_ui") == m)
            db = df_b.filter(pl.col("map_ui") == m)
            rows.append(
                {
                    "map": m,
                    "count_a": len(da),
                    "count_b": len(db),
                    "wins_a": int((da["outcome"] == Outcome.WIN).sum())
                    if "outcome" in da.columns
                    else 0,
                    "wins_b": int((db["outcome"] == Outcome.WIN).sum())
                    if "outcome" in db.columns
                    else 0,
                }
            )
        return rows
    except Exception:
        logger.debug("_build_maps_table: erreur", exc_info=True)
        return []


def _build_modes_table(df_a, df_b) -> list[dict]:
    """Construit le tableau comparatif des modes."""
    try:
        import polars as pl

        rows: list[dict] = []
        if "mode_ui" not in df_a.columns and "mode_ui" not in df_b.columns:
            return rows

        modes = set()
        if "mode_ui" in df_a.columns:
            modes.update(df_a["mode_ui"].drop_nulls().to_list())
        if "mode_ui" in df_b.columns:
            modes.update(df_b["mode_ui"].drop_nulls().to_list())

        for mo in sorted(modes):
            da = df_a.filter(pl.col("mode_ui") == mo) if "mode_ui" in df_a.columns else df_a.head(0)
            db = df_b.filter(pl.col("mode_ui") == mo) if "mode_ui" in df_b.columns else df_b.head(0)
            rows.append(
                {
                    "mode": mo,
                    "count_a": len(da),
                    "count_b": len(db),
                }
            )
        return rows
    except Exception:
        logger.debug("_build_modes_table: erreur", exc_info=True)
        return []


# ---------------------------------------------------------------------------
# Point d'entrée public
# ---------------------------------------------------------------------------


def get_session_compare(
    player: PlayerContext, request: SessionCompareRequest
) -> SessionCompareResponse:
    """Construit la réponse complète pour la page Comparaison de sessions."""
    df_full = _load_df_full(player)

    import polars as pl

    if not isinstance(df_full, pl.DataFrame) or df_full.is_empty():
        return SessionCompareResponse(available_sessions=[], metrics=[])

    available = _get_available_sessions(df_full)
    if len(available) < 2:
        return SessionCompareResponse(available_sessions=available, metrics=[])

    # Sélection des sessions A et B
    label_a = request.session_a if request.session_a in available else available[0]
    label_b = request.session_b if request.session_b in available else available[1]
    if label_a == label_b and len(available) >= 2:
        label_b = available[1] if label_a == available[0] else available[0]

    df_a = _filter_by_session(df_full, label_a)
    df_b = _filter_by_session(df_full, label_b)

    is_a_empty = not isinstance(df_a, pl.DataFrame) or df_a.is_empty()
    is_b_empty = not isinstance(df_b, pl.DataFrame) or df_b.is_empty()

    if is_a_empty or is_b_empty:
        return SessionCompareResponse(available_sessions=available, metrics=[])

    entry_a = _build_session_entry(df_a, label_a)
    entry_b = _build_session_entry(df_b, label_b)

    # Calcul des stats agrégées pour les métriques
    try:
        from src.analysis.performance_score import compute_session_performance_score_v2

        perf_a = compute_session_performance_score_v2(df_a)
        perf_b = compute_session_performance_score_v2(df_b)
    except Exception:
        logger.debug("get_session_compare: erreur calcul perf", exc_info=True)
        perf_a = {}
        perf_b = {}

    metrics = _build_metric_rows(perf_a, perf_b)
    radar_chart = _build_radar_chart(perf_a, perf_b)
    kd_chart = _build_kd_progression_chart(df_a, df_b, label_a, label_b)
    maps_table = _build_maps_table(df_a, df_b)
    modes_table = _build_modes_table(df_a, df_b)

    return SessionCompareResponse(
        session_a=entry_a,
        session_b=entry_b,
        available_sessions=available,
        metrics=metrics,
        radar_chart=radar_chart,
        kd_progression_chart=kd_chart,
        maps_table=maps_table,
        modes_table=modes_table,
    )
