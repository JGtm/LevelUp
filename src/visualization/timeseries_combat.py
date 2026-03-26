"""Graphiques de séries temporelles — Combat (Sprint 16).

Fonctions déplacées depuis ``timeseries.py`` pour alléger le module principal.
Les fonctions de progression (performance, rang, LUSR) sont dans
``_timeseries_progression.py``.
"""

import plotly.graph_objects as go
import polars as pl
from plotly.subplots import make_subplots

from src.config import PLOT_CONFIG
from src.data.domain.refdata import Outcome
from src.ui.i18n.viz import viz_t
from src.visualization._compat import DataFrameLike, ensure_polars, smart_scatter  # noqa: F401
from src.visualization._timeseries_helpers import COLORS, apply_chrono_xaxis, prepare_time_axis
from src.visualization._timeseries_progression import (  # noqa: F401 — re-exports
    plot_lusr_timeseries,
    plot_performance_timeseries,
    plot_rank_score,
)
from src.visualization.theme import apply_halo_plot_style, get_legend_horizontal_bottom
from src.visualization.timeseries import _normalize_df, _rolling_mean


def plot_average_life(df: DataFrameLike, title: str | None = None, lang: str = "fr") -> go.Figure:
    """Graphique de la durée de vie moyenne.

    Args:
        df: DataFrame (Pandas ou Polars) avec colonne average_life_seconds.
        title: Titre du graphique.

    Returns:
        Figure Plotly.
    """
    d = _normalize_df(df)
    if title is None:
        title = viz_t("title_avg_life", lang)

    d = d.filter(pl.col("average_life_seconds").is_not_null()).sort("start_time")
    x_idx, labels, step = prepare_time_axis(d)

    y = d["average_life_seconds"].cast(pl.Float64, strict=False)
    custom = list(
        zip(
            d["deaths"].fill_null(0).cast(pl.Int64).to_list(),
            d["time_played_seconds"].cast(pl.Float64, strict=False).to_list(),
            d["match_id"].cast(pl.Utf8).to_list(),
            strict=False,
        )
    )

    fig = go.Figure()
    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=y.to_list(),
            name=viz_t("trace_lifespan", lang),
            marker_color=COLORS["green"],
            opacity=PLOT_CONFIG.bar_opacity,
            customdata=custom,
            hovertemplate=viz_t("hover_lifespan", lang),
        )
    )

    fig.add_trace(
        smart_scatter(
            x=x_idx,
            y=_rolling_mean(y, window=10).to_list(),
            mode="lines",
            name=viz_t("trace_avg_smoothed", lang),
            line={"width": PLOT_CONFIG.line_width, "color": COLORS["cyan"]},
            hovertemplate=viz_t("hover_avg_smoothed_s", lang),
        )
    )

    fig.update_layout(
        title=title,
        margin={"l": 40, "r": 20, "t": 50, "b": 90},
        hovermode="x unified",
        legend=get_legend_horizontal_bottom(),
    )
    fig.update_yaxes(title_text=viz_t("axis_seconds", lang), rangemode="tozero")
    apply_chrono_xaxis(fig, x_idx, labels, step, lang)

    return apply_halo_plot_style(fig, height=PLOT_CONFIG.short_height)


def plot_spree_headshots_accuracy(
    df: DataFrameLike,
    perfect_counts: dict[str, int] | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Graphique combiné: Spree, Tirs à la tête, Précision et Perfect kills.

    Args:
        df: DataFrame avec colonnes max_killing_spree, headshot_kills, accuracy.
        perfect_counts: Dict optionnel {match_id: count} pour les médailles Perfect.

    Returns:
        Figure Plotly avec axe Y secondaire pour la précision.
    """
    d = _normalize_df(df)

    d = d.sort("start_time")
    x_idx, labels, step = prepare_time_axis(d)

    if "max_killing_spree" in d.columns:
        spree = d["max_killing_spree"].cast(pl.Float64, strict=False).to_list()
    else:
        spree = [float("nan")] * d.height

    fig = make_subplots(rows=1, cols=1, specs=[[{"secondary_y": True}]])

    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=spree,
            name=viz_t("trace_killing_spree", lang),
            marker_color=COLORS["amber"],
            opacity=PLOT_CONFIG.bar_opacity,
            alignmentgroup="spree_hs",
            offsetgroup="spree",
            width=0.42,
            hovertemplate=viz_t("hover_killing_spree", lang),
        ),
        secondary_y=False,
    )

    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=d["headshot_kills"].to_list(),
            name=viz_t("trace_headshots", lang),
            marker_color=COLORS["red"],
            opacity=0.70,
            alignmentgroup="spree_hs",
            offsetgroup="headshots",
            width=0.42,
            hovertemplate=viz_t("hover_headshots", lang),
        ),
        secondary_y=False,
    )

    # Frags parfaits (médaille Perfect)
    if "match_id" in d.columns and perfect_counts is not None:
        match_ids = d["match_id"].cast(pl.Utf8).to_list()
        perfect_series = [perfect_counts.get(mid, 0) for mid in match_ids]
    else:
        perfect_series = [0] * d.height
    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=perfect_series,
            name=viz_t("trace_perfect_kills", lang),
            marker_color=COLORS["green"],
            opacity=0.65,
            alignmentgroup="spree_hs",
            offsetgroup="perfect",
            width=0.28,
            hovertemplate=viz_t("hover_perfect_sprees", lang),
        ),
        secondary_y=False,
    )

    apply_chrono_xaxis(fig, x_idx, labels, step, lang, as_category=False)

    fig.update_layout(
        height=420,
        margin={"l": 40, "r": 50, "t": 30, "b": 90},
        legend=get_legend_horizontal_bottom(),
        hovermode="x unified",
        barmode="group",
        bargap=0.15,
        bargroupgap=0.06,
    )

    fig.update_yaxes(
        title_text=viz_t("axis_spree_headshots", lang), rangemode="tozero", secondary_y=False
    )

    return apply_halo_plot_style(fig, height=420)


def plot_streak_chart(
    df: DataFrameLike,
    title: str | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Graphique des séries de victoires et défaites dans le temps.

    Args:
        df: DataFrame avec colonnes outcome, start_time.
        title: Titre du graphique.

    Returns:
        Figure Plotly.
    """
    d = _normalize_df(df)
    d = d.sort("start_time")

    # Filtrer : ne garder que V/D
    d = d.filter(pl.col("outcome").is_in([2, 3]))
    if d.height == 0:
        fig = go.Figure()
        fig.add_annotation(
            text=viz_t("empty_no_streak_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16},
        )
        return apply_halo_plot_style(fig, title=title or None, height=PLOT_CONFIG.short_height)

    x_idx, labels, step = prepare_time_axis(d)

    # Calculer la série : cumul dans chaque streak
    outcome_col = d["outcome"]
    is_win = (outcome_col == Outcome.WIN).cast(pl.Int64)
    new_streak = (outcome_col != outcome_col.shift(1)).fill_null(True)
    streak_group = new_streak.cast(pl.Int64).cum_sum()

    streak_counter: list[int] = []
    prev_group = -1
    count = 0
    for g in streak_group.to_list():
        if g != prev_group:
            count = 1
            prev_group = g
        else:
            count += 1
        streak_counter.append(count)

    is_win_list = is_win.to_list()
    streak_values = [c if w == 1 else -c for c, w in zip(streak_counter, is_win_list, strict=False)]

    bar_colors = [COLORS["green"] if v > 0 else COLORS["red"] for v in streak_values]

    fig = go.Figure()
    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=streak_values,
            marker_color=bar_colors,
            opacity=0.85,
            hovertemplate=viz_t("hover_streak", lang),
            customdata=labels,
            showlegend=False,
        )
    )

    layout_kwargs: dict = {
        "margin": {"l": 40, "r": 20, "t": 40 if title else 10, "b": 90},
        "hovermode": "x unified",
    }
    if title is not None:
        layout_kwargs["title"] = title
    fig.update_layout(**layout_kwargs)
    fig.update_yaxes(
        title_text=viz_t("axis_streak", lang),
        zeroline=True,
        zerolinecolor="rgba(255,255,255,0.75)",
        zerolinewidth=2,
    )
    apply_chrono_xaxis(fig, x_idx, labels, step, lang)

    return apply_halo_plot_style(fig, height=PLOT_CONFIG.short_height)


def _add_damage_traces(  # noqa: PLR0913
    fig: go.Figure,
    d: pl.DataFrame,
    x_idx: list,
    col: str,
    color: str,
    opacity: float,
    bar_key: str,
    hover_key: str,
    avg_key: str,
    lang: str,
    *,
    line_dash: str | None = None,
) -> None:
    """Ajoute une paire Bar + rolling mean pour une colonne de dégâts."""
    if col not in d.columns:
        return
    series = d[col].cast(pl.Float64, strict=False).fill_null(0)
    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=series.to_list(),
            name=viz_t(bar_key, lang),
            marker_color=color,
            opacity=opacity,
            hovertemplate=viz_t(hover_key, lang),
        )
    )
    line_opts: dict = {"width": PLOT_CONFIG.line_width, "color": color}
    if line_dash:
        line_opts["dash"] = line_dash
    fig.add_trace(
        smart_scatter(
            x=x_idx,
            y=_rolling_mean(series, window=10).to_list(),
            mode="lines",
            name=viz_t(avg_key, lang),
            line=line_opts,
            hovertemplate=viz_t("hover_avg0", lang),
        )
    )


def plot_damage_dealt_taken(
    df: DataFrameLike,
    title: str | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Graphique des dégâts infligés et subis par match.

    Args:
        df: DataFrame avec colonnes damage_dealt, damage_taken, start_time.
        title: Titre du graphique.

    Returns:
        Figure Plotly.
    """
    d = _normalize_df(df)
    if title is None:
        title = viz_t("title_damage", lang)

    d = d.sort("start_time")
    x_idx, labels, step = prepare_time_axis(d)

    fig = go.Figure()

    _add_damage_traces(
        fig,
        d,
        x_idx,
        "damage_dealt",
        COLORS["cyan"],
        0.80,
        "trace_dmg_dealt",
        "hover_dmg_dealt",
        "trace_dmg_dealt_avg",
        lang,
    )
    _add_damage_traces(
        fig,
        d,
        x_idx,
        "damage_taken",
        COLORS["red"],
        0.65,
        "trace_dmg_taken",
        "hover_dmg_taken",
        "trace_dmg_taken_avg",
        lang,
        line_dash="dot",
    )

    fig.update_layout(
        title=title,
        margin={"l": 40, "r": 20, "t": 40, "b": 90},
        hovermode="x unified",
        legend=get_legend_horizontal_bottom(),
        barmode="group",
        bargap=0.15,
        bargroupgap=0.06,
    )
    fig.update_yaxes(title_text=viz_t("axis_damage", lang), rangemode="tozero")
    apply_chrono_xaxis(fig, x_idx, labels, step, lang)

    return apply_halo_plot_style(fig, height=PLOT_CONFIG.default_height)


def _add_shots_traces(
    fig: go.Figure,
    d: pl.DataFrame,
    x_idx: list[int],
    lang: str,
) -> None:
    """Ajoute les traces tirs tirés/touchés et précision au graphique."""
    if "shots_fired" in d.columns:
        fired = d["shots_fired"].cast(pl.Float64, strict=False).fill_null(0)
        fig.add_trace(
            go.Bar(
                x=x_idx,
                y=fired.to_list(),
                name=viz_t("trace_shots_fired", lang),
                marker_color=COLORS["amber"],
                opacity=0.70,
                alignmentgroup="shots",
                offsetgroup="fired",
                width=0.42,
                hovertemplate=viz_t("hover_shots_fired", lang),
            ),
            secondary_y=False,
        )

    if "shots_hit" in d.columns:
        hit = d["shots_hit"].cast(pl.Float64, strict=False).fill_null(0)
        fig.add_trace(
            go.Bar(
                x=x_idx,
                y=hit.to_list(),
                name=viz_t("trace_shots_hit", lang),
                marker_color=COLORS["green"],
                opacity=0.70,
                alignmentgroup="shots",
                offsetgroup="hit",
                width=0.42,
                hovertemplate=viz_t("hover_shots_hit", lang),
            ),
            secondary_y=False,
        )

    if "accuracy" in d.columns:
        accuracy = d["accuracy"].cast(pl.Float64, strict=False)
        fig.add_trace(
            smart_scatter(
                x=x_idx,
                y=accuracy.to_list(),
                mode="lines",
                name=viz_t("trace_accuracy", lang),
                line={"width": PLOT_CONFIG.line_width, "color": COLORS["violet"]},
                hovertemplate=viz_t("hover_accuracy_pct", lang),
            ),
            secondary_y=True,
        )


def plot_shots_accuracy(
    df: DataFrameLike,
    title: str | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Graphique des tirs (tirés/touchés) en barres groupées avec courbe de précision.

    Args:
        df: DataFrame avec colonnes shots_fired, shots_hit, accuracy, start_time.
        title: Titre du graphique.

    Returns:
        Figure Plotly avec axe Y secondaire pour la précision.
    """
    d = _normalize_df(df)
    if title is None:
        title = viz_t("title_shots", lang)

    d = d.sort("start_time")
    x_idx, labels, step = prepare_time_axis(d)

    fig = make_subplots(rows=1, cols=1, specs=[[{"secondary_y": True}]])

    _add_shots_traces(fig, d, x_idx, lang)

    apply_chrono_xaxis(fig, x_idx, labels, step, lang, as_category=False)

    fig.update_layout(
        title=title,
        height=420,
        margin={"l": 40, "r": 50, "t": 40, "b": 90},
        legend=get_legend_horizontal_bottom(),
        hovermode="x unified",
        barmode="group",
        bargap=0.15,
        bargroupgap=0.06,
    )

    fig.update_yaxes(title_text=viz_t("axis_shots", lang), rangemode="tozero", secondary_y=False)
    fig.update_yaxes(
        title_text=viz_t("trace_accuracy", lang),
        ticksuffix="%",
        rangemode="tozero",
        secondary_y=True,
    )

    return apply_halo_plot_style(fig, height=420)
