"""Graphiques par carte (map) — ratio win/loss et comparaison de métriques.

Les graphiques outcome-focused (lollipop, timeline, bullet, perf vs historique)
sont dans ``maps_outcome.py``.
"""

import plotly.graph_objects as go
import polars as pl

from src.analysis.performance_config import SCORE_THRESHOLDS
from src.config import HALO_COLORS, PLOT_CONFIG
from src.ui.i18n.viz import viz_t
from src.visualization._compat import DataFrameLike, ensure_polars, to_pandas_for_plotly
from src.visualization.theme import apply_halo_plot_style, get_legend_horizontal_bottom


def plot_map_comparison(
    df_breakdown: DataFrameLike,
    metric: str,
    title: str,
    color_by_sign: bool = False,
) -> go.Figure:
    """Graphique de comparaison d'une métrique par carte.

    Args:
        df_breakdown: DataFrame issu de compute_map_breakdown.
        metric: Nom de la colonne à afficher (ratio_global, win_rate, accuracy_avg).
        title: Titre du graphique.
        color_by_sign: Si True, colore les barres en vert (>= 0) / rouge (< 0).

    Returns:
        Figure Plotly (barres horizontales).
    """
    colors = HALO_COLORS.as_dict()
    df_pl = ensure_polars(df_breakdown).drop_nulls(subset=[metric])

    if df_pl.is_empty():
        fig = go.Figure()
        fig.update_layout(
            height=PLOT_CONFIG.default_height, margin={"l": 40, "r": 20, "t": 30, "b": 40}
        )
        return apply_halo_plot_style(fig, height=PLOT_CONFIG.default_height)

    d = to_pandas_for_plotly(df_pl)

    if color_by_sign:
        _c = HALO_COLORS.as_dict()

        def _perf_color(v: float) -> str:
            if v >= SCORE_THRESHOLDS["excellent"]:
                return _c["green"]
            if v >= SCORE_THRESHOLDS["good"]:
                return _c["cyan"]
            if v >= SCORE_THRESHOLDS["average"]:
                return _c["amber"]
            if v >= SCORE_THRESHOLDS["below_average"]:
                return _c.get("orange", "#FF8C00")
            return _c["red"]

        bar_colors = [_perf_color(v) for v in d[metric]]
    else:
        bar_colors = colors["cyan"]

    fig = go.Figure(
        data=[
            go.Bar(
                x=d[metric],
                y=d["map_name"],
                orientation="h",
                marker_color=bar_colors,
                customdata=list(zip(d["matches"], d.get("accuracy_avg"), strict=False)),
                hovertemplate=("%{y}<br>value=%{x:.1f}<br>matches=%{customdata[0]}<extra></extra>"),
            )
        ]
    )
    fig.update_layout(
        height=PLOT_CONFIG.tall_height,
        title=title,
        margin={"l": 40, "r": 20, "t": 60, "b": 90},
        legend=get_legend_horizontal_bottom(),
    )

    return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.tall_height)


def _winloss_display_params(d: object, lang: str, absolute_counts: bool) -> dict:
    """Calcule x, textes et hover selon le mode absolu ou pourcentage."""
    import numpy as _np  # noqa: PLC0415

    if absolute_counts:
        wins = (d["win_rate"] * d["matches"]).round().astype(int)  # type: ignore[index]
        losses = (d["loss_rate"] * d["matches"]).round().astype(int)  # type: ignore[index]
        others = _np.maximum(d["matches"] - wins - losses, 0)  # type: ignore[index]
        return {
            "x_win": wins,
            "x_loss": losses,
            "x_other": others,
            "text_win": [f"{r:.0%}" if r > 0 else "" for r in d["win_rate"]],  # type: ignore[index]
            "text_loss": [f"{r:.0%}" if r > 0 else "" for r in d["loss_rate"]],  # type: ignore[index]
            "text_other": [""] * len(d["matches"]),  # type: ignore[arg-type]
            "hover_win": "%{y}<br>"
            + viz_t("trace_wins", lang)
            + "=%{x}  (total=%{customdata[0]})<extra></extra>",
            "hover_loss": "%{y}<br>"
            + viz_t("trace_losses", lang)
            + "=%{x}  (total=%{customdata[0]})<extra></extra>",
            "hover_other": "%{y}<br>autres=%{x}  (total=%{customdata[0]})<extra></extra>",
            "x_axis_fmt": ",d",
            "x_axis_title": viz_t("axis_matches", lang),
            "x_range": None,
        }
    return {
        "x_win": d["win_rate"],
        "x_loss": d["loss_rate"],
        "x_other": d["other_rate"],  # type: ignore[index]
        "text_win": [],
        "text_loss": [],
        "text_other": [],
        "hover_win": "%{y}<br>win=%{x:.1%}<br>matchs=%{customdata[0]}<extra></extra>",
        "hover_loss": "%{y}<br>loss=%{x:.1%}<br>matchs=%{customdata[0]}<extra></extra>",
        "hover_other": "%{y}<br>autres=%{x:.1%}<br>matchs=%{customdata[0]}<extra></extra>",
        "x_axis_fmt": ".0%",
        "x_axis_title": viz_t("axis_win_rate", lang),
        "x_range": [0, 1],
    }


def _add_winloss_traces(fig: go.Figure, d: object, p: dict, colors: dict, lang: str) -> None:
    """Ajoute les 3 traces Win/Loss/Autre sur la figure."""
    _bar_kwargs = {"orientation": "h", "customdata": d[["matches"]].values}  # type: ignore[index]
    _tf = {"size": 11, "color": "white", "weight": "bold"}
    fig.add_trace(
        go.Bar(
            x=p["x_win"],
            y=d["map_name"],
            name=viz_t("trace_wins", lang),  # type: ignore[index]
            marker_color=colors["green"],
            opacity=0.70,
            hovertemplate=p["hover_win"],
            text=p["text_win"],
            textposition="inside",
            insidetextanchor="middle",
            textfont=_tf,
            **_bar_kwargs,
        )
    )
    fig.add_trace(
        go.Bar(
            x=p["x_loss"],
            y=d["map_name"],
            name=viz_t("trace_losses", lang),  # type: ignore[index]
            marker_color=colors["red"],
            opacity=0.55,
            hovertemplate=p["hover_loss"],
            text=p["text_loss"],
            textposition="inside",
            insidetextanchor="middle",
            textfont=_tf,
            **_bar_kwargs,
        )
    )
    fig.add_trace(
        go.Bar(
            x=p["x_other"],
            y=d["map_name"],
            name=viz_t("trace_others_tie_unfinished", lang),  # type: ignore[index]
            marker_color=colors["violet"],
            opacity=0.35,
            hovertemplate=p["hover_other"],
            text=p["text_other"],
            textposition="inside",
            **_bar_kwargs,
        )
    )


def plot_map_ratio_with_winloss(
    df_breakdown: DataFrameLike,
    title: str,
    lang: str = "fr",
    absolute_counts: bool = False,
) -> go.Figure:
    """Graphique de ratio par carte avec taux de victoire/défaite."""
    colors = HALO_COLORS.as_dict()
    df_pl = ensure_polars(df_breakdown).drop_nulls(subset=["win_rate", "loss_rate"])

    if df_pl.is_empty():
        fig = go.Figure()
        fig.update_layout(
            height=PLOT_CONFIG.default_height, margin={"l": 40, "r": 20, "t": 30, "b": 40}
        )
        return apply_halo_plot_style(fig, height=PLOT_CONFIG.default_height)

    df_pl = df_pl.with_columns(
        (pl.lit(1.0) - pl.col("win_rate").cast(pl.Float64) - pl.col("loss_rate").cast(pl.Float64))
        .clip(0.0, 1.0)
        .alias("other_rate")
    )
    d = to_pandas_for_plotly(df_pl)
    p = _winloss_display_params(d, lang, absolute_counts)

    fig = go.Figure()
    _add_winloss_traces(fig, d, p, colors, lang)
    fig.update_layout(
        height=PLOT_CONFIG.tall_height,
        title=title,
        margin={"l": 40, "r": 20, "t": 60, "b": 90},
        barmode="stack",
        bargap=0.18,
        legend=get_legend_horizontal_bottom(),
    )
    fig.update_xaxes(title_text=p["x_axis_title"], tickformat=p["x_axis_fmt"])
    if p["x_range"] is not None:
        fig.update_xaxes(range=p["x_range"])
    return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.tall_height)
