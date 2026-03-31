"""Graphiques de séries temporelles."""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl
from plotly.subplots import make_subplots

from src.config import HALO_COLORS, PLOT_CONFIG
from src.ui.components.chart_annotations import add_extreme_annotations
from src.ui.i18n.viz import viz_t
from src.visualization._compat import (
    DataFrameLike,
    ensure_polars,
    ensure_polars_series,
    smart_scatter,
)
from src.visualization._permin_helpers import build_symmetric_abs_ticks
from src.visualization.theme import apply_halo_plot_style, get_legend_horizontal_bottom


def _normalize_df(df: DataFrameLike) -> pl.DataFrame:
    """Normalise un DataFrame en Polars (compat arrière)."""
    return ensure_polars(df)


def _rolling_mean(series: pl.Series, window: int = 10) -> pl.Series:
    """Calcule la moyenne mobile.

    Args:
        series: Série Polars (accepte aussi Pandas pour compat arrière).
        window: Taille de la fenêtre.

    Returns:
        Série Polars avec moyenne mobile.
    """
    w = int(window) if window and window > 0 else 1
    if not isinstance(series, pl.Series):
        series = ensure_polars_series(series)
    return series.rolling_mean(window_size=w, min_samples=1)


def _build_kda_customdata(d: pl.DataFrame, lang: str = "fr") -> tuple[list, str]:
    """Construit le customdata et le template hover commun pour graphiques KDA.

    Args:
        d: DataFrame trié avec colonnes kills, deaths, assists, accuracy, ratio.
        lang: Langue ("fr" ou "en").

    Returns:
        Tuple (customdata, common_hover).
    """
    common_hover = viz_t("hover_kda_combined", lang)
    accuracy = d["accuracy"].cast(pl.Float64, strict=False).fill_null(0).round(2)
    customdata = list(
        zip(
            d["kills"].to_list(),
            d["deaths"].to_list(),
            d["assists"].to_list(),
            accuracy.to_list(),
            d["ratio"].to_list(),
            strict=False,
        )
    )
    return customdata, common_hover


def _add_kda_traces(  # noqa: PLR0913
    fig: go.Figure,
    x_idx: list[int],
    d: pl.DataFrame,
    customdata: list,
    common_hover: str,
    colors: dict,
    lang: str = "fr",
) -> None:
    """Ajoute les traces Kills/Deaths/Ratio au subplot KDA."""
    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=d["kills"].to_list(),
            name=viz_t("trace_kills", lang),
            marker_color=colors["cyan"],
            opacity=PLOT_CONFIG.bar_opacity,
            alignmentgroup="kda_main",
            offsetgroup="kills",
            width=0.42,
            customdata=customdata,
            hovertemplate=common_hover,
        ),
        secondary_y=False,
    )
    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=d["deaths"].to_list(),
            name=viz_t("trace_deaths", lang),
            marker_color=colors["red"],
            opacity=PLOT_CONFIG.bar_opacity_secondary,
            alignmentgroup="kda_main",
            offsetgroup="deaths",
            width=0.42,
            customdata=customdata,
            hovertemplate=common_hover,
        ),
        secondary_y=False,
    )
    fig.add_trace(
        smart_scatter(
            x=x_idx,
            y=d["ratio"].to_list(),
            mode="lines",
            name=viz_t("trace_ratio", lang),
            line={"width": PLOT_CONFIG.line_width, "color": colors["green"]},
            customdata=customdata,
            hovertemplate=common_hover,
        ),
        secondary_y=True,
    )


def plot_timeseries(df: DataFrameLike, title: str | None = None, lang: str = "fr") -> go.Figure:
    """Graphique principal: Kills/Deaths/Ratio dans le temps."""
    df_pl = pl.DataFrame() if df is None else ensure_polars(df)

    title = title or viz_t("title_kda", lang)
    if df_pl.is_empty():
        fig = go.Figure()
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16},
        )
        return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.tall_height)

    fig = make_subplots(rows=1, cols=1, specs=[[{"secondary_y": True}]])
    colors = HALO_COLORS.as_dict()
    d = df_pl.sort("start_time")
    x_idx = list(range(len(d)))

    customdata, common_hover = _build_kda_customdata(d, lang=lang)
    _add_kda_traces(fig, x_idx, d, customdata, common_hover, colors, lang=lang)

    fig.update_layout(
        title=title,
        legend=get_legend_horizontal_bottom(),
        margin={"l": 40, "r": 20, "t": 80, "b": 90},
        hovermode="x unified",
        barmode="group",
        bargap=0.15,
        bargroupgap=0.06,
    )
    fig.update_xaxes(type="category")
    fig.update_yaxes(
        title_text=viz_t("axis_kills_deaths", lang), rangemode="tozero", secondary_y=False
    )
    fig.update_yaxes(title_text=viz_t("axis_ratio", lang), secondary_y=True)

    _map_tick_col = "map_ui" if "map_ui" in d.columns else "map_name"
    labels = (
        [
            f"#{i + 1}<br>{mn}" if mn else f"#{i + 1}"
            for i, mn in enumerate(d[_map_tick_col].fill_null("").to_list())
        ]
        if _map_tick_col in d.columns
        else d["start_time"].dt.strftime(FMT_TICK_DATETIME).to_list()
    )
    step = max(1, len(labels) // 10) if len(labels) > 1 else 1
    fig.update_xaxes(
        title_text=viz_t("axis_match_number", lang),
        tickmode="array",
        tickvals=x_idx[::step],
        ticktext=labels[::step],
        tickangle=-45,
    )

    add_extreme_annotations(
        fig,
        x_idx,
        d["ratio"].to_list(),
        metric_name="ratio",
        show_max=True,
        show_min=False,
        max_color="#FFD700",
        secondary_y=True,
    )

    return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.tall_height)


def plot_assists_timeseries(
    df: DataFrameLike, title: str | None = None, lang: str = "fr"
) -> go.Figure:
    """Graphique des assistances dans le temps.

    Args:
        df: DataFrame avec colonnes assists, start_time, etc.
        title: Titre du graphique.

    Returns:
        Figure Plotly.
    """
    df_pl = ensure_polars(df)

    title = title or viz_t("title_assists", lang)
    colors = HALO_COLORS.as_dict()
    d = df_pl.sort("start_time")
    x_idx = list(range(len(d)))
    labels = d["start_time"].dt.strftime(FMT_TICK_DATETIME).to_list()
    step = max(1, len(labels) // 10) if len(labels) > 1 else 1

    accuracy = d["accuracy"].cast(pl.Float64, strict=False).fill_null(0).round(2)
    customdata = list(
        zip(
            d["kills"].to_list(),
            d["deaths"].to_list(),
            d["assists"].to_list(),
            accuracy.to_list(),
            d["ratio"].to_list(),
            (d["map_ui"] if "map_ui" in d.columns else d["map_name"]).fill_null("").to_list(),
            d["playlist_name"].fill_null("").to_list(),
            d["match_id"].to_list(),
            strict=False,
        )
    )
    hover = viz_t("hover_assists_combined", lang)

    fig = go.Figure()
    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=d["assists"].to_list(),
            name=viz_t("trace_assists", lang),
            marker_color=colors["violet"],
            opacity=PLOT_CONFIG.bar_opacity,
            customdata=customdata,
            hovertemplate=hover,
        )
    )

    assists_series = d["assists"].cast(pl.Float64, strict=False)
    smooth = _rolling_mean(assists_series, window=10)
    fig.add_trace(
        smart_scatter(
            x=x_idx,
            y=smooth.to_list(),
            mode="lines",
            name=viz_t("trace_avg_smoothed", lang),
            line={"width": PLOT_CONFIG.line_width, "color": colors["green"]},
            hovertemplate=viz_t("hover_avg_smoothed", lang),
        )
    )

    fig.update_layout(
        title=title,
        margin={"l": 40, "r": 20, "t": 60, "b": 90},
        hovermode="x unified",
        legend=get_legend_horizontal_bottom(),
    )
    fig.update_yaxes(title_text=viz_t("axis_assists", lang), rangemode="tozero")
    fig.update_xaxes(
        title_text=viz_t("axis_match_number", lang),
        tickmode="array",
        tickvals=x_idx[::step],
        ticktext=labels[::step],
        type="category",
    )

    return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.default_height)


def _add_permin_rolling_lines(  # noqa: PLR0913
    fig: go.Figure,
    x_idx: list[int],
    kpm: pl.Series,
    dpm: pl.Series,
    apm: pl.Series,
    colors: dict[str, str],
    lang: str = "fr",
) -> None:
    """Ajoute les 3 courbes de moyenne mobile par minute (frags, morts, assistances).

    Args:
        fig: Figure Plotly à enrichir.
        x_idx: Index des matchs.
        kpm: Série kills per minute.
        dpm: Série deaths per minute (valeurs négatives — sous l'axe X).
        apm: Série assists per minute.
        colors: Dict de couleurs HALO.
    """
    fig.add_trace(
        smart_scatter(
            x=x_idx,
            y=_rolling_mean(kpm, window=10).to_list(),
            mode="lines",
            name=viz_t("trace_avg_kills_per_min", lang),
            line={"width": PLOT_CONFIG.line_width, "color": colors["cyan"]},
            hovertemplate=viz_t("hover_avg", lang),
        )
    )
    dpm_rolling = _rolling_mean(dpm, window=10).to_list()
    fig.add_trace(
        smart_scatter(
            x=x_idx,
            y=dpm_rolling,
            mode="lines",
            name=viz_t("trace_avg_deaths_per_min", lang),
            line={"width": PLOT_CONFIG.line_width, "color": colors["red"], "dash": "dot"},
            customdata=[abs(v) for v in dpm_rolling],
            hovertemplate=viz_t("hover_avg_abs", lang),
        )
    )
    fig.add_trace(
        smart_scatter(
            x=x_idx,
            y=_rolling_mean(apm, window=10).to_list(),
            mode="lines",
            name=viz_t("trace_avg_assists_per_min", lang),
            line={"width": PLOT_CONFIG.line_width, "color": colors["violet"], "dash": "dot"},
            hovertemplate=viz_t("hover_avg", lang),
        )
    )


def plot_per_minute_timeseries(
    df: DataFrameLike, title: str | None = None, lang: str = "fr"
) -> go.Figure:
    """Graphique des stats par minute.

    Args:
        df: DataFrame avec colonnes kills_per_min, deaths_per_min, assists_per_min.
        title: Titre du graphique.

    Returns:
        Figure Plotly.
    """
    df_pl = ensure_polars(df)

    title = title or viz_t("title_permin", lang)
    colors = HALO_COLORS.as_dict()
    d = df_pl.sort("start_time")
    _permin_cols = ("kills_per_min", "deaths_per_min", "assists_per_min")
    if not all(c in d.columns for c in _permin_cols):
        _tps = pl.col("time_played_seconds").cast(pl.Float64, strict=False)
        d = d.with_columns(
            [
                (pl.col("kills").cast(pl.Float64, strict=False) / (_tps / 60))
                .fill_nan(0.0)
                .fill_null(0.0)
                .alias("kills_per_min"),
                (pl.col("deaths").cast(pl.Float64, strict=False) / (_tps / 60))
                .fill_nan(0.0)
                .fill_null(0.0)
                .alias("deaths_per_min"),
                (pl.col("assists").cast(pl.Float64, strict=False) / (_tps / 60))
                .fill_nan(0.0)
                .fill_null(0.0)
                .alias("assists_per_min"),
            ]
        )
    x_idx = list(range(len(d)))
    _map_tick_col2 = "map_ui" if "map_ui" in d.columns else "map_name"
    labels = (
        [
            f"#{i + 1}<br>{mn}" if mn else f"#{i + 1}"
            for i, mn in enumerate(d[_map_tick_col2].fill_null("").to_list())
        ]
        if _map_tick_col2 in d.columns
        else d["start_time"].dt.strftime(FMT_TICK_DATETIME).to_list()
    )
    step = max(1, len(labels) // 10) if len(labels) > 1 else 1

    time_played = d["time_played_seconds"].cast(pl.Float64, strict=False)
    customdata = list(
        zip(
            time_played.to_list(),
            d["kills"].to_list(),
            d["deaths"].to_list(),
            d["assists"].to_list(),
            d["match_id"].to_list(),
            strict=False,
        )
    )

    kpm = d["kills_per_min"].cast(pl.Float64, strict=False)
    dpm = d["deaths_per_min"].cast(pl.Float64, strict=False)
    apm = d["assists_per_min"].cast(pl.Float64, strict=False)
    dpm_neg = (-dpm).to_list()  # morts sous l'axe X ; abs conservée en customdata[5]
    customdata_with_dpm = [(*row, a) for row, a in zip(customdata, dpm.to_list(), strict=False)]

    fig = go.Figure()
    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=kpm.to_list(),
            name=viz_t("trace_kills_per_min", lang),
            marker_color=colors["cyan"],
            opacity=PLOT_CONFIG.bar_opacity,
            customdata=customdata_with_dpm,
            hovertemplate=viz_t("hover_kpm", lang),
        )
    )
    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=dpm_neg,
            name=viz_t("trace_deaths_per_min", lang),
            marker_color=colors["red"],
            opacity=0.4,  # version désaturée pour indiquer la valeur négative
            customdata=customdata_with_dpm,
            hovertemplate=viz_t("hover_dpm_neg", lang),
        )
    )
    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=apm.to_list(),
            name=viz_t("trace_assists_per_min", lang),
            marker_color=colors["violet"],
            opacity=PLOT_CONFIG.bar_opacity_secondary,
            customdata=customdata_with_dpm,
            hovertemplate=viz_t("hover_apm", lang),
        )
    )

    _add_permin_rolling_lines(fig, x_idx, kpm, -dpm, apm, colors, lang=lang)

    tickvals, ticktext = build_symmetric_abs_ticks(
        max(max(kpm.to_list(), default=0.1), max(apm.to_list(), default=0.1)),
        max((abs(v) for v in dpm_neg), default=0.1),
    )
    fig.update_layout(
        title=title,
        margin={"l": 40, "r": 20, "t": 60, "b": 90},
        hovermode="x unified",
        legend=get_legend_horizontal_bottom(),
        barmode="group",
        bargap=0.15,
        bargroupgap=0.06,
    )
    fig.update_yaxes(
        title_text=viz_t("axis_per_minute", lang),
        tickvals=tickvals,
        ticktext=ticktext,
    )
    fig.update_xaxes(
        title_text=viz_t("axis_match_number", lang),
        tickmode="array",
        tickvals=x_idx[::step],
        ticktext=labels[::step],
        tickangle=-45,
        type="category",
    )

    return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.default_height)


def plot_accuracy_last_n(df: DataFrameLike, n: int, lang: str = "fr") -> go.Figure:
    """Graphique de précision sur les N derniers matchs.

    Args:
        df: DataFrame avec colonne accuracy.
        n: Nombre de matchs à afficher.

    Returns:
        Figure Plotly.
    """
    df_pl = ensure_polars(df)

    colors = HALO_COLORS.as_dict()
    d = df_pl.drop_nulls(subset=["accuracy"]).tail(n)

    fig = go.Figure(
        data=[
            smart_scatter(
                x=d["start_time"].to_list(),
                y=d["accuracy"].to_list(),
                mode="lines",
                name=viz_t("trace_accuracy", lang),
                line={"width": PLOT_CONFIG.line_width, "color": colors["violet"]},
                hovertemplate=viz_t("hover_accuracy_pct", lang),
            )
        ]
    )
    fig.update_layout(height=PLOT_CONFIG.short_height, margin={"l": 40, "r": 20, "t": 30, "b": 40})
    fig.update_yaxes(title_text="%", rangemode="tozero")

    return apply_halo_plot_style(fig, height=PLOT_CONFIG.short_height)


# Re-exports depuis timeseries_combat (compat backward — Sprint 16)
from src.ui.date_formats import FMT_TICK_DATETIME
from src.visualization.timeseries_combat import (  # noqa: E402, F401
    plot_average_life,
    plot_damage_dealt_taken,
    plot_performance_timeseries,
    plot_rank_score,
    plot_shots_accuracy,
    plot_spree_headshots_accuracy,
    plot_streak_chart,
)
